# 🎯 GPU负载控制器 - 快速入门

## ✅ 已创建的文件

1. **`gpu_load_controller.py`** - 主程序（智能GPU负载控制器）
2. **`test_gpu_setup.py`** - 环境测试脚本
3. **`GPU_LOAD_CONTROLLER_README.md`** - 完整使用文档

## 🚀 立即使用

### 方法1：默认配置（推荐）

```bash
python3 gpu_load_controller.py
```

这将保持GPU利用率在**30%**左右，持续运行直到按 `Ctrl+C` 停止。

### 方法2：自定义目标利用率

```bash
# 保持GPU利用率在50%
python3 gpu_load_controller.py --target 50

# 保持GPU利用率在20%
python3 gpu_load_controller.py --target 20

# 保持GPU利用率在70%
python3 gpu_load_controller.py --target 70
```

### 方法3：限定运行时间

```bash
# 运行5分钟（300秒）
python3 gpu_load_controller.py --target 30 --duration 300

# 运行1小时（3600秒）
python3 gpu_load_controller.py --target 30 --duration 3600
```

### 方法4：指定GPU设备

```bash
# 使用GPU 0
python3 gpu_load_controller.py --target 30 --gpu 0

# 使用GPU 1
python3 gpu_load_controller.py --target 30 --gpu 1
```

## 📊 运行效果

运行后会看到类似输出：

```
✅ 检测到 1 个GPU设备
✅ 使用GPU: NVIDIA GeForce RTX 3090

🚀 开始运行GPU负载控制器
📊 目标利用率: 30.0%
⏱️  运行时长: 持续运行 (Ctrl+C停止)
------------------------------------------------------------
[14:23:10] GPU利用率:  28.5% | 目标:  30.0% | 内存: 2048/24576MB (8.3%) | 矩阵: 2200x2200 | 计算: 45.2ms
[14:23:11] GPU利用率:  30.2% | 目标:  30.0% | 内存: 2048/24576MB (8.3%) | 矩阵: 2200x2200 | 计算: 46.1ms
...
```

## ⚙️ 环境状态

当前环境测试结果：

- ✅ **PyTorch**: 已安装 (版本 2.7.1+cu118)
- ⚠️ **CUDA**: 当前环境不可用（需要GPU硬件和驱动）
- ⚠️ **nvidia-smi**: 未安装

## 🔧 如果在有GPU的机器上使用

1. **确保已安装NVIDIA驱动**
   ```bash
   nvidia-smi
   ```

2. **测试环境**
   ```bash
   python3 test_gpu_setup.py
   ```

3. **运行GPU负载控制器**
   ```bash
   python3 gpu_load_controller.py --target 30
   ```

## 🎨 特性

✨ **智能调节** - 自动调整计算负载以达到目标利用率  
📈 **实时监控** - 显示GPU利用率、内存使用、计算时间  
🎯 **精确控制** - 利用率误差通常在±5%以内  
🔒 **安全退出** - Ctrl+C安全停止，自动清理GPU内存  
🔄 **自适应算法** - 根据实际情况动态调整策略  

## 💡 典型使用场景

| 场景 | 命令 | 说明 |
|------|------|------|
| 日常保持GPU活跃 | `python3 gpu_load_controller.py --target 30` | 低负载，不影响其他任务 |
| GPU压力测试 | `python3 gpu_load_controller.py --target 80 --duration 3600` | 高负载运行1小时 |
| 测试散热系统 | `python3 gpu_load_controller.py --target 70` | 持续中高负载 |
| 防止GPU闲置 | `python3 gpu_load_controller.py --target 20` | 最小负载保持活跃 |

## 📖 更多信息

详细使用说明请查看：**`GPU_LOAD_CONTROLLER_README.md`**

## ⚡ 常用命令速查

```bash
# 查看帮助
python3 gpu_load_controller.py --help

# 测试GPU环境
python3 test_gpu_setup.py

# 后台运行
nohup python3 gpu_load_controller.py --target 30 > gpu_load.log 2>&1 &

# 查看后台日志
tail -f gpu_load.log

# 停止后台进程
pkill -f gpu_load_controller
```

## 🛑 停止运行

在前台运行时：
- 按 **`Ctrl+C`** 即可安全停止

在后台运行时：
```bash
# 查找进程ID
ps aux | grep gpu_load_controller

# 终止进程
kill <PID>

# 或者直接
pkill -f gpu_load_controller
```

---

**提示**: 如果你在没有GPU的环境中，脚本会提示"CUDA不可用"。请将脚本转移到有NVIDIA GPU的机器上运行。
