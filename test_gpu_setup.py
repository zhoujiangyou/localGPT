#!/usr/bin/env python3
"""
GPU环境测试脚本
用于验证GPU和PyTorch是否正确配置
"""

import sys

def test_pytorch():
    """测试PyTorch安装"""
    try:
        import torch
        print(f"✅ PyTorch已安装: {torch.__version__}")
        return True
    except ImportError:
        print("❌ PyTorch未安装")
        return False

def test_cuda():
    """测试CUDA可用性"""
    try:
        import torch
        if torch.cuda.is_available():
            print(f"✅ CUDA可用")
            print(f"   CUDA版本: {torch.version.cuda}")
            print(f"   GPU数量: {torch.cuda.device_count()}")
            
            for i in range(torch.cuda.device_count()):
                gpu_name = torch.cuda.get_device_name(i)
                gpu_memory = torch.cuda.get_device_properties(i).total_memory / 1024**3
                print(f"   GPU {i}: {gpu_name} ({gpu_memory:.1f} GB)")
            
            return True
        else:
            print("⚠️  CUDA不可用 - 可能原因：")
            print("   1. 系统中没有NVIDIA GPU")
            print("   2. NVIDIA驱动未安装")
            print("   3. PyTorch安装时未包含CUDA支持")
            return False
    except Exception as e:
        print(f"❌ CUDA测试失败: {e}")
        return False

def test_nvidia_smi():
    """测试nvidia-smi命令"""
    import subprocess
    try:
        result = subprocess.run(['nvidia-smi'], capture_output=True, timeout=5)
        if result.returncode == 0:
            print("✅ nvidia-smi命令可用")
            return True
        else:
            print("⚠️  nvidia-smi命令执行失败")
            return False
    except FileNotFoundError:
        print("⚠️  nvidia-smi命令未找到")
        print("   提示: 请确保已安装NVIDIA驱动")
        return False
    except Exception as e:
        print(f"❌ nvidia-smi测试失败: {e}")
        return False

def test_simple_computation():
    """测试简单的GPU计算"""
    try:
        import torch
        if not torch.cuda.is_available():
            print("⏭️  跳过GPU计算测试（CUDA不可用）")
            return False
        
        print("\n🧪 执行GPU计算测试...")
        device = torch.device('cuda:0')
        
        # 创建测试张量
        a = torch.randn(1000, 1000, device=device)
        b = torch.randn(1000, 1000, device=device)
        
        # 执行计算
        c = torch.matmul(a, b)
        torch.cuda.synchronize()
        
        result = c.sum().item()
        print(f"✅ GPU计算测试通过 (结果: {result:.2f})")
        return True
    except Exception as e:
        print(f"❌ GPU计算测试失败: {e}")
        return False

def main():
    print("=" * 60)
    print("GPU环境测试")
    print("=" * 60)
    print()
    
    results = []
    
    print("1️⃣  检查PyTorch安装...")
    results.append(("PyTorch", test_pytorch()))
    print()
    
    print("2️⃣  检查CUDA可用性...")
    results.append(("CUDA", test_cuda()))
    print()
    
    print("3️⃣  检查nvidia-smi命令...")
    results.append(("nvidia-smi", test_nvidia_smi()))
    print()
    
    print("4️⃣  测试GPU计算...")
    results.append(("GPU计算", test_simple_computation()))
    print()
    
    # 总结
    print("=" * 60)
    print("测试总结")
    print("=" * 60)
    
    for name, passed in results:
        status = "✅ 通过" if passed else "❌ 失败"
        print(f"{name:20s} {status}")
    
    print()
    
    all_passed = all(result for _, result in results)
    if all_passed:
        print("🎉 所有测试通过！可以运行gpu_load_controller.py")
        print("\n使用示例:")
        print("  python3 gpu_load_controller.py --target 30")
    else:
        print("⚠️  部分测试未通过")
        print("\n如果CUDA不可用，可能的解决方案：")
        print("1. 确保系统有NVIDIA GPU")
        print("2. 安装最新的NVIDIA驱动")
        print("3. 重新安装支持CUDA的PyTorch:")
        print("   pip3 install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cu118")

if __name__ == '__main__':
    main()
