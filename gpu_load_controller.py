#!/usr/bin/env python3
"""
GPU负载控制器 - 保持GPU利用率在目标水平
可以动态调整计算负载来维持指定的GPU使用率
"""

import torch
import time
import subprocess
import re
import sys
from typing import Optional

class GPULoadController:
    def __init__(self, target_utilization: float = 30.0, gpu_id: int = 0):
        """
        初始化GPU负载控制器
        
        Args:
            target_utilization: 目标GPU利用率百分比 (0-100)
            gpu_id: GPU设备ID
        """
        self.target_utilization = target_utilization
        self.gpu_id = gpu_id
        self.device = None
        self.running = False
        
        # 控制参数
        self.matrix_size = 2000  # 矩阵大小
        self.compute_time = 0.1  # 计算时间（秒）
        self.sleep_time = 0.05   # 休眠时间（秒）
        
        # 自适应调整参数
        self.adjustment_factor = 0.1
        self.min_matrix_size = 500
        self.max_matrix_size = 8000
        
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
        print(f"✅ 使用GPU: {gpu_name}")
        
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
            print(f"⚠️ 无法获取GPU利用率: {e}")
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
    
    def adjust_workload(self, current_util: float):
        """根据当前利用率调整工作负载"""
        diff = self.target_utilization - current_util
        
        if abs(diff) > 5:  # 如果差异大于5%，进行调整
            if diff > 0:  # 需要增加负载
                self.matrix_size = int(self.matrix_size * (1 + self.adjustment_factor))
                self.matrix_size = min(self.matrix_size, self.max_matrix_size)
                if self.sleep_time > 0.01:
                    self.sleep_time *= 0.9
            else:  # 需要减少负载
                self.matrix_size = int(self.matrix_size * (1 - self.adjustment_factor))
                self.matrix_size = max(self.matrix_size, self.min_matrix_size)
                self.sleep_time *= 1.1
                self.sleep_time = min(self.sleep_time, 0.5)
    
    def compute_workload(self):
        """执行GPU计算任务"""
        # 创建随机矩阵并进行运算
        matrix_a = torch.randn(self.matrix_size, self.matrix_size, device=self.device)
        matrix_b = torch.randn(self.matrix_size, self.matrix_size, device=self.device)
        
        # 矩阵乘法
        result = torch.matmul(matrix_a, matrix_b)
        
        # 额外的运算以增加负载
        result = torch.relu(result)
        result = torch.sum(result)
        
        # 同步以确保计算完成
        torch.cuda.synchronize()
        
        return result.item()
    
    def run(self, duration: Optional[int] = None):
        """
        运行GPU负载控制器
        
        Args:
            duration: 运行时长（秒），None表示持续运行直到手动停止
        """
        if not self.check_gpu_available():
            return
        
        print(f"\n🚀 开始运行GPU负载控制器")
        print(f"📊 目标利用率: {self.target_utilization}%")
        print(f"⏱️  运行时长: {'持续运行 (Ctrl+C停止)' if duration is None else f'{duration}秒'}")
        print("-" * 60)
        
        self.running = True
        start_time = time.time()
        iteration = 0
        
        try:
            while self.running:
                # 检查运行时长
                if duration is not None and (time.time() - start_time) >= duration:
                    break
                
                # 执行计算任务
                compute_start = time.time()
                _ = self.compute_workload()
                compute_elapsed = time.time() - compute_start
                
                # 获取GPU状态
                current_util = self.get_gpu_utilization()
                mem_used, mem_total = self.get_gpu_memory_info()
                
                # 调整工作负载
                if current_util is not None:
                    self.adjust_workload(current_util)
                
                # 每10次迭代打印一次状态
                iteration += 1
                if iteration % 10 == 0:
                    if current_util is not None:
                        mem_percent = (mem_used / mem_total * 100) if mem_total > 0 else 0
                        print(f"[{time.strftime('%H:%M:%S')}] "
                              f"GPU利用率: {current_util:5.1f}% | "
                              f"目标: {self.target_utilization:5.1f}% | "
                              f"内存: {mem_used:.0f}/{mem_total:.0f}MB ({mem_percent:.1f}%) | "
                              f"矩阵: {self.matrix_size}x{self.matrix_size} | "
                              f"计算: {compute_elapsed*1000:.1f}ms")
                
                # 休眠以控制负载
                time.sleep(self.sleep_time)
                
        except KeyboardInterrupt:
            print("\n\n⏹️  收到停止信号")
        finally:
            self.running = False
            elapsed = time.time() - start_time
            print("-" * 60)
            print(f"✅ GPU负载控制器已停止")
            print(f"📊 总运行时间: {elapsed:.1f}秒")
            print(f"🔄 总迭代次数: {iteration}")
            
            # 清理GPU内存
            if self.device is not None:
                torch.cuda.empty_cache()
                print("🧹 已清理GPU内存")


def main():
    import argparse
    
    parser = argparse.ArgumentParser(
        description='GPU负载控制器 - 保持GPU利用率在指定水平',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
示例:
  # 保持GPU利用率在30%
  python gpu_load_controller.py --target 30
  
  # 保持GPU利用率在50%，运行300秒
  python gpu_load_controller.py --target 50 --duration 300
  
  # 使用GPU 1，保持利用率在40%
  python gpu_load_controller.py --target 40 --gpu 1
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
    
    args = parser.parse_args()
    
    # 验证参数
    if args.target < 0 or args.target > 100:
        print("❌ 错误: 目标利用率必须在0-100之间")
        sys.exit(1)
    
    if args.duration is not None and args.duration <= 0:
        print("❌ 错误: 运行时长必须大于0")
        sys.exit(1)
    
    # 创建并运行控制器
    controller = GPULoadController(
        target_utilization=args.target,
        gpu_id=args.gpu
    )
    
    controller.run(duration=args.duration)


if __name__ == '__main__':
    main()
