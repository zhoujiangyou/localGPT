# GPU负载控制器使用指南

## 📌 简介

这是一个智能的GPU负载控制器，可以自动调整计算负载以维持指定的GPU利用率水平。非常适合用于：
- GPU性能测试
- 保持GPU温度稳定
- 防止GPU资源闲置
- 测试散热系统

## 🚀 快速开始

### 基本用法

```bash
# 默认保持GPU利用率在30%
python3 gpu_load_controller.py

# 保持GPU利用率在50%
python3 gpu_load_controller.py --target 50

# 运行5分钟（300秒）
python3 gpu_load_controller.py --target 30 --duration 300
```

### 命令行参数

| 参数 | 简写 | 说明 | 默认值 |
|------|------|------|--------|
| `--target` | `-t` | 目标GPU利用率百分比 (0-100) | 30 |
| `--duration` | `-d` | 运行时长（秒），不指定则持续运行 | None |
| `--gpu` | `-g` | GPU设备ID | 0 |

## 📊 输出示例

```
✅ 检测到 1 个GPU设备
✅ 使用GPU: NVIDIA GeForce RTX 3090

🚀 开始运行GPU负载控制器
📊 目标利用率: 30.0%
⏱️  运行时长: 持续运行 (Ctrl+C停止)
------------------------------------------------------------
[14:23:10] GPU利用率:  28.5% | 目标:  30.0% | 内存: 2048/24576MB (8.3%) | 矩阵: 2200x2200 | 计算: 45.2ms
[14:23:11] GPU利用率:  30.2% | 目标:  30.0% | 内存: 2048/24576MB (8.3%) | 矩阵: 2200x2200 | 计算: 46.1ms
[14:23:12] GPU利用率:  29.8% | 目标:  30.0% | 内存: 2048/24576MB (8.3%) | 矩阵: 2200x2200 | 计算: 45.8ms
```

## 🔧 技术细节

### 工作原理

1. **计算任务**: 在GPU上执行大型矩阵乘法运算
2. **监控**: 使用`nvidia-smi`实时获取GPU利用率
3. **自适应调整**: 
   - 利用率低于目标 → 增加矩阵大小，减少休眠时间
   - 利用率高于目标 → 减少矩阵大小，增加休眠时间
4. **平滑控制**: 使用调整因子逐步调整，避免震荡

### 参数范围

- **矩阵大小**: 500x500 到 8000x8000
- **计算时间**: 取决于矩阵大小和GPU性能
- **休眠时间**: 0.01秒到0.5秒
- **调整因子**: 10% (每次调整的幅度)

## ⚙️ 系统要求

### 必需

- Python 3.7+
- PyTorch (已安装)
- NVIDIA GPU
- NVIDIA驱动
- CUDA Toolkit

### 验证环境

```bash
# 检查CUDA是否可用
python3 -c "import torch; print(f'CUDA available: {torch.cuda.is_available()}')"

# 检查GPU信息
nvidia-smi

# 查看PyTorch版本
python3 -c "import torch; print(f'PyTorch: {torch.__version__}')"
```

## 💡 使用场景

### 1. 长时间保持GPU活跃

```bash
# 保持GPU利用率30%，持续运行
python3 gpu_load_controller.py --target 30
```

### 2. 测试GPU散热

```bash
# 保持GPU利用率70%，运行1小时
python3 gpu_load_controller.py --target 70 --duration 3600
```

### 3. 多GPU环境

```bash
# 在GPU 0上保持30%利用率
python3 gpu_load_controller.py --target 30 --gpu 0 &

# 在GPU 1上保持40%利用率
python3 gpu_load_controller.py --target 40 --gpu 1 &
```

## 🛑 停止脚本

- 按 `Ctrl+C` 安全停止
- 脚本会自动清理GPU内存
- 显示运行统计信息

## ⚠️ 注意事项

1. **温度监控**: 长时间运行时注意监控GPU温度
2. **电力消耗**: GPU使用会增加功耗
3. **其他任务**: 运行时可能影响其他GPU任务的性能
4. **利用率波动**: 实际利用率会在目标值附近波动±5%
5. **内存使用**: 脚本会占用部分GPU显存（通常<4GB）

## 🔍 故障排除

### 问题：提示"CUDA不可用"

**解决方案**:
```bash
# 检查NVIDIA驱动
nvidia-smi

# 检查PyTorch CUDA支持
python3 -c "import torch; print(torch.cuda.is_available())"

# 如果需要，重新安装PyTorch with CUDA
pip3 install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cu118
```

### 问题：无法获取GPU利用率

**原因**: `nvidia-smi`命令不可用

**解决方案**:
- 确保已安装NVIDIA驱动
- 检查环境变量PATH是否包含nvidia-smi路径

### 问题：利用率无法达到目标值

**可能原因**:
- 目标值过高（超过GPU最大性能）
- 目标值过低（最小计算负载已达到）
- GPU正在被其他程序使用

**解决方案**:
- 调整目标值到合理范围（20%-80%）
- 检查是否有其他程序占用GPU
- 查看GPU最大性能参数

## 📈 高级用法

### 在后台运行

```bash
# 后台运行，输出到日志文件
nohup python3 gpu_load_controller.py --target 30 > gpu_load.log 2>&1 &

# 查看日志
tail -f gpu_load.log

# 停止后台任务
ps aux | grep gpu_load_controller
kill <PID>
```

### 开机自启动

创建systemd服务文件 `/etc/systemd/system/gpu-load.service`:

```ini
[Unit]
Description=GPU Load Controller
After=network.target

[Service]
Type=simple
User=your_username
WorkingDirectory=/workspace
ExecStart=/usr/bin/python3 /workspace/gpu_load_controller.py --target 30
Restart=always

[Install]
WantedBy=multi-user.target
```

启动服务:
```bash
sudo systemctl daemon-reload
sudo systemctl enable gpu-load.service
sudo systemctl start gpu-load.service
```

## 🤝 贡献

如有问题或建议，欢迎提出issue或pull request。

## 📄 许可

MIT License
