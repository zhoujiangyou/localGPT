#!/usr/bin/env python3
"""
GPU负载控制器 V2 - 改进版
使用多线程和并发计算来更精确地控制GPU利用率
"""

import torch
import time
import subprocess
import sys
import threading
from typing import Optional
from queue import Queue

class GPULoadControllerV2:
    def __init__(self, target_utilization: float = 30.0, gpu_id: int = 0):
        """
        初始化GPU负载控制器 V2
        
        Args:
            target_utilization: 目标GPU利用率百分比 (0-100)
            gpu_id: GPU设备ID
        """
        self.target_utilization = target_utilization
        self.gpu_id = gpu_id
        self.device = None
        self.running = False
        
        # 改进的控制参数
        self.num_workers = 4  # 并发工作线程数
        self.matrix_size = 3000  # 初始矩阵大小
        self.batch_size = 1  # 批处理大小
        self.compute_intensity = 1  # 计算强度（每次迭代的计算次数）
        
        # 自适应调整参数（更激进）
        self.adjustment_factor = 0.15  # 从0.1提高到0.15
        self.min_matrix_size = 1000
        self.max_matrix_size = 16000  # 从8000提高到16000
        self.min_intensity = 1
        self.max_intensity = 10
        
        # 统计信息
        self.stats_lock = threading.Lock()
        self.current_util = 0.0
        self.utilization_history = []
        self.max_history = 20
        
    def check_gpu_available(self) -> bool:
        """检查GPU是否可用"""
        if not torch.cuda.is_available():
            print("❌ CUDA不可用，请确保已安装NVIDIA驱动和CUDA")
            return False
        
        gpu_count = torch.cuda.device_count()
        print(f"✅ 检测到 {gpu_count} 个GPU设备")
        
        if self.gpu_id >= gpu_count:
            print(f"❌ GPU ID {self.gpu_id} 不存在")
            return False
            
        self.device = torch.device(f'cuda:{self.gpu_id}')
        gpu_name = torch.cuda.get_device_name(self.gpu_id)
        gpu_props = torch.cuda.get_device_properties(self.gpu_id)
        gpu_memory_gb = gpu_props.total_memory / 1024**3
        
        print(f"✅ 使用GPU: {gpu_name}")
        print(f"📊 显存容量: {gpu_memory_gb:.1f} GB")
        print(f"🔢 计算能力: {gpu_props.major}.{gpu_props.minor}")
        print(f"🧵 并发工作线程: {self.num_workers}")
        
        return True
    
    def get_gpu_utilization(self) -> Optional[float]:
        """获取当前GPU利用率"""
        try:
            result = subprocess.run(
                ['nvidia-smi', '--query-gpu=utilization.gpu', '--format=csv,noheader,nounits', 
                 f'--id={self.gpu_id}'],
                capture_output=True,
                text=True,
                timeout=2
            )
            
            if result.returncode == 0:
                utilization = float(result.stdout.strip())
                return utilization
            else:
                return None
        except Exception as e:
            return None
    
    def get_gpu_memory_info(self) -> tuple:
        """获取GPU内存信息"""
        try:
            result = subprocess.run(
                ['nvidia-smi', '--query-gpu=memory.used,memory.total', '--format=csv,noheader,nounits',
                 f'--id={self.gpu_id}'],
                capture_output=True,
                text=True,
                timeout=2
            )
            
            if result.returncode == 0:
                used, total = result.stdout.strip().split(',')
                return float(used), float(total)
            else:
                return 0, 0
        except Exception as e:
            return 0, 0
    
    def get_gpu_temperature(self) -> Optional[float]:
        """获取GPU温度"""
        try:
            result = subprocess.run(
                ['nvidia-smi', '--query-gpu=temperature.gpu', '--format=csv,noheader,nounits',
                 f'--id={self.gpu_id}'],
                capture_output=True,
                text=True,
                timeout=2
            )
            
            if result.returncode == 0:
                temp = float(result.stdout.strip())
                return temp
            else:
                return None
        except Exception as e:
            return None
    
    def adjust_workload(self, current_util: float):
        """根据当前利用率调整工作负载 - 改进版"""
        diff = self.target_utilization - current_util
        
        # 记录历史利用率
        with self.stats_lock:
            self.current_util = current_util
            self.utilization_history.append(current_util)
            if len(self.utilization_history) > self.max_history:
                self.utilization_history.pop(0)
        
        # 计算平均利用率来避免过度反应
        avg_util = sum(self.utilization_history) / len(self.utilization_history) if self.utilization_history else current_util
        avg_diff = self.target_utilization - avg_util
        
        # 根据差异大小使用不同的调整策略
        if abs(diff) > 20:  # 差异很大，需要快速调整
            adjustment = 0.25
        elif abs(diff) > 10:  # 差异较大
            adjustment = 0.15
        elif abs(diff) > 5:  # 差异中等
            adjustment = 0.08
        else:  # 差异较小，微调
            adjustment = 0.03
        
        if avg_diff > 0:  # 需要增加负载
            # 优先增加矩阵大小
            new_size = int(self.matrix_size * (1 + adjustment))
            if new_size <= self.max_matrix_size:
                self.matrix_size = new_size
            elif self.compute_intensity < self.max_intensity:
                # 如果矩阵已经很大，增加计算强度
                self.compute_intensity = min(self.compute_intensity + 1, self.max_intensity)
                
        else:  # 需要减少负载
            # 优先减少计算强度
            if self.compute_intensity > self.min_intensity:
                self.compute_intensity = max(self.compute_intensity - 1, self.min_intensity)
            else:
                # 然后减少矩阵大小
                new_size = int(self.matrix_size * (1 - adjustment))
                self.matrix_size = max(new_size, self.min_matrix_size)
    
    def compute_workload(self, worker_id: int):
        """执行GPU计算任务 - 改进版"""
        try:
            # 使用独立的CUDA流来实现并发
            stream = torch.cuda.Stream()
            
            with torch.cuda.stream(stream):
                # 执行多次计算以增加强度
                for _ in range(self.compute_intensity):
                    # 矩阵乘法
                    matrix_a = torch.randn(self.matrix_size, self.matrix_size, 
                                          device=self.device, dtype=torch.float32)
                    matrix_b = torch.randn(self.matrix_size, self.matrix_size, 
                                          device=self.device, dtype=torch.float32)
                    
                    result = torch.matmul(matrix_a, matrix_b)
                    
                    # 额外的计算来增加GPU使用
                    result = torch.nn.functional.relu(result)
                    result = torch.pow(result, 2)
                    result = torch.sum(result)
                    
                    # 清理中间结果
                    del matrix_a, matrix_b
            
            # 确保计算完成
            stream.synchronize()
            
        except RuntimeError as e:
            if "out of memory" in str(e):
                # 如果显存不足，减小矩阵
                self.matrix_size = max(int(self.matrix_size * 0.8), self.min_matrix_size)
                torch.cuda.empty_cache()
    
    def worker_thread(self, worker_id: int):
        """工作线程"""
        while self.running:
            self.compute_workload(worker_id)
            # 非常短的休眠以避免完全占用CPU
            time.sleep(0.001)
    
    def monitor_thread(self):
        """监控线程 - 调整负载和显示状态"""
        iteration = 0
        last_adjustment = time.time()
        
        while self.running:
            time.sleep(0.5)  # 每0.5秒监控一次
            
            # 获取GPU状态
            current_util = self.get_gpu_utilization()
            mem_used, mem_total = self.get_gpu_memory_info()
            temp = self.get_gpu_temperature()
            
            if current_util is not None:
                # 每2秒调整一次负载
                if time.time() - last_adjustment > 2.0:
                    self.adjust_workload(current_util)
                    last_adjustment = time.time()
                
                # 每5次迭代打印一次状态
                iteration += 1
                if iteration % 10 == 0:
                    mem_percent = (mem_used / mem_total * 100) if mem_total > 0 else 0
                    
                    # 计算平均利用率
                    with self.stats_lock:
                        avg_util = sum(self.utilization_history) / len(self.utilization_history) if self.utilization_history else current_util
                    
                    temp_str = f"{temp:.0f}°C" if temp is not None else "N/A"
                    print(f"[{time.strftime('%H:%M:%S')}] "
                          f"实时: {current_util:5.1f}% | "
                          f"平均: {avg_util:5.1f}% | "
                          f"目标: {self.target_utilization:5.1f}% | "
                          f"温度: {temp_str} | "
                          f"内存: {mem_used:.0f}MB ({mem_percent:.1f}%) | "
                          f"矩阵: {self.matrix_size}x{self.matrix_size} | "
                          f"强度: {self.compute_intensity}")
    
    def run(self, duration: Optional[int] = None):
        """
        运行GPU负载控制器
        
        Args:
            duration: 运行时长（秒），None表示持续运行直到手动停止
        """
        if not self.check_gpu_available():
            return
        
        print(f"\n🚀 开始运行GPU负载控制器 V2")
        print(f"📊 目标利用率: {self.target_utilization}%")
        print(f"⏱️  运行时长: {'持续运行 (Ctrl+C停止)' if duration is None else f'{duration}秒'}")
        print("-" * 90)
        
        self.running = True
        start_time = time.time()
        
        # 启动工作线程
        workers = []
        for i in range(self.num_workers):
            t = threading.Thread(target=self.worker_thread, args=(i,), daemon=True)
            t.start()
            workers.append(t)
        
        # 启动监控线程
        monitor = threading.Thread(target=self.monitor_thread, daemon=True)
        monitor.start()
        
        try:
            # 主线程等待
            while self.running:
                if duration is not None and (time.time() - start_time) >= duration:
                    break
                time.sleep(1)
                
        except KeyboardInterrupt:
            print("\n\n⏹️  收到停止信号")
        finally:
            self.running = False
            
            # 等待线程结束
            time.sleep(0.5)
            
            elapsed = time.time() - start_time
            print("-" * 90)
            print(f"✅ GPU负载控制器已停止")
            print(f"📊 总运行时间: {elapsed:.1f}秒")
            
            # 统计信息
            if self.utilization_history:
                avg_util = sum(self.utilization_history) / len(self.utilization_history)
                min_util = min(self.utilization_history)
                max_util = max(self.utilization_history)
                error = abs(avg_util - self.target_utilization)
                
                print(f"📈 利用率统计:")
                print(f"   平均: {avg_util:.1f}% (误差: {error:.1f}%)")
                print(f"   范围: {min_util:.1f}% - {max_util:.1f}%")
            
            # 清理GPU内存
            if self.device is not None:
                torch.cuda.empty_cache()
                print("🧹 已清理GPU内存")


def main():
    import argparse
    
    parser = argparse.ArgumentParser(
        description='GPU负载控制器 V2 - 改进版，更精确的GPU利用率控制',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
示例:
  # 保持GPU利用率在30%
  python gpu_load_controller_v2.py --target 30
  
  # 保持GPU利用率在50%，运行300秒
  python gpu_load_controller_v2.py --target 50 --duration 300
  
  # 使用GPU 1，保持利用率在70%
  python gpu_load_controller_v2.py --target 70 --gpu 1
  
改进点:
  - 使用多线程并发计算
  - 更大的矩阵范围 (1000-16000)
  - 双重调节：矩阵大小 + 计算强度
  - 平滑的自适应调整算法
  - 更详细的监控信息
        """
    )
    
    parser.add_argument(
        '--target', '-t',
        type=float,
        default=30.0,
        help='目标GPU利用率百分比 (0-100)，默认: 30'
    )
    
    parser.add_argument(
        '--duration', '-d',
        type=int,
        default=None,
        help='运行时长（秒），默认: 持续运行直到Ctrl+C'
    )
    
    parser.add_argument(
        '--gpu', '-g',
        type=int,
        default=0,
        help='GPU设备ID，默认: 0'
    )
    
    parser.add_argument(
        '--workers', '-w',
        type=int,
        default=4,
        help='并发工作线程数，默认: 4'
    )
    
    args = parser.parse_args()
    
    # 验证参数
    if args.target < 0 or args.target > 100:
        print("❌ 错误: 目标利用率必须在0-100之间")
        sys.exit(1)
    
    if args.duration is not None and args.duration <= 0:
        print("❌ 错误: 运行时长必须大于0")
        sys.exit(1)
    
    if args.workers < 1 or args.workers > 32:
        print("❌ 错误: 工作线程数必须在1-32之间")
        sys.exit(1)
    
    # 创建并运行控制器
    controller = GPULoadControllerV2(
        target_utilization=args.target,
        gpu_id=args.gpu
    )
    controller.num_workers = args.workers
    
    controller.run(duration=args.duration)


if __name__ == '__main__':
    main()
