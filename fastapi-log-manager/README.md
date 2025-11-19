# FastAPI 日志管理和自动清理方案

## 📖 项目简介

一个完整的FastAPI日志管理解决方案，解决日志文件写满磁盘导致实例重启的问题。

### 核心功能

- ✅ **自动日志轮转**: 基于时间和大小的日志轮转
- ✅ **定时清理**: 自动清理过期日志文件
- ✅ **磁盘监控**: 实时监控磁盘空间使用情况
- ✅ **紧急清理**: 磁盘空间告急时自动清理
- ✅ **日志压缩**: 自动压缩旧日志节省空间
- ✅ **RESTful API**: 提供完整的管理接口
- ✅ **健康检查**: 监控磁盘使用率

## 🚀 快速开始

### 安装依赖

```bash
cd /workspace/fastapi-log-manager
pip install -r requirements.txt
```

### 启动应用

```bash
python -m app.main
```

或使用uvicorn：

```bash
uvicorn app.main:app --host 0.0.0.0 --port 8000 --reload
```

访问：http://localhost:8000

## 📊 功能特性

### 1. 自动日志轮转

使用loguru库实现智能日志轮转：

- **按时间轮转**: 每天凌晨00:00自动轮转
- **按大小轮转**: 单个文件超过100MB自动轮转
- **自动压缩**: 旧日志自动压缩为zip格式
- **分级存储**: 错误日志单独存储，保留更久

### 2. 定时清理任务

内置4个定时任务：

| 任务 | 频率 | 说明 |
|------|------|------|
| 每日清理 | 每天02:00 | 清理超过保留期的日志 |
| 磁盘检查 | 每小时 | 检查磁盘空间，超过阈值触发紧急清理 |
| 大小检查 | 每6小时 | 检查日志目录大小，超过限制自动清理 |
| 每周压缩 | 每周日03:00 | 压缩3天前的未压缩日志 |

### 3. 多种清理策略

#### 按时间清理
```python
# 清理7天前的日志
log_manager.clean_old_logs(days=7)
```

#### 按大小清理
```python
# 清理到500MB以下
log_manager.clean_by_size(max_size_mb=500)
```

#### 紧急清理
```python
# 删除最旧的50%日志
log_manager.emergency_cleanup()
```

### 4. 磁盘监控

实时监控磁盘使用情况：

```python
usage = log_manager.get_disk_usage()
# {
#     "total_gb": 100.0,
#     "used_gb": 75.5,
#     "free_gb": 24.5,
#     "percent": 75.5
# }
```

当磁盘使用率超过阈值（默认80%）时，自动触发紧急清理。

## 📡 API接口

### 查看日志状态

```bash
GET /log/status
```

响应：
```json
{
  "log_directory": "/path/to/logs",
  "retention_days": 7,
  "disk_usage": {
    "total_gb": 100.0,
    "used_gb": 75.5,
    "free_gb": 24.5,
    "percent": 75.5
  },
  "log_size": {
    "total_size_mb": 250.5,
    "file_count": 45
  },
  "log_files": 45,
  "oldest_log": {
    "name": "app_2024-01-01.log",
    "size_mb": 12.5
  }
}
```

### 手动清理日志

```bash
# 清理7天前的日志
POST /log/cleanup?days=7

# 按大小清理（保留最新500MB）
POST /log/cleanup/size?max_size_mb=500

# 紧急清理
POST /log/cleanup/emergency
```

### 压缩旧日志

```bash
# 压缩3天前的日志
POST /log/compress?days=3
```

### 查看磁盘使用情况

```bash
GET /log/disk
```

### 查看定时任务

```bash
GET /log/jobs
```

### 手动触发任务

```bash
POST /log/jobs/daily_cleanup/trigger
```

### 健康检查

```bash
GET /health
```

## 🔧 配置说明

### 环境变量配置

创建 `.env` 文件：

```bash
# 日志配置
LOG_DIR=logs
LOG_RETENTION_DAYS=7
LOG_MAX_FILE_SIZE=100 MB
LOG_ROTATION_TIME=00:00
LOG_COMPRESSION=zip

# 磁盘监控
DISK_THRESHOLD_PERCENT=80

# 定时任务
DAILY_CLEANUP_HOUR=2
DISK_CHECK_INTERVAL_HOURS=1
```

### 代码配置

在应用中自定义配置：

```python
from app.log_manager import LogManager

log_manager = LogManager(
    log_dir="logs",              # 日志目录
    retention_days=7,            # 保留天数
    max_file_size="100 MB",      # 单文件最大大小
    rotation_time="00:00",       # 轮转时间
    compression="zip",           # 压缩格式
    disk_threshold_percent=80,   # 磁盘阈值
)
```

## 🛠️ 独立脚本使用

### 使用Python脚本

```bash
# 清理7天前的日志
python scripts/cleanup_logs.py --days 7

# 按大小清理
python scripts/cleanup_logs.py --max-size 500

# 压缩旧日志
python scripts/cleanup_logs.py --compress --compress-days 3

# 紧急清理
python scripts/cleanup_logs.py --emergency
```

### 使用Cron定时任务

添加到crontab：

```bash
# 每天凌晨2点清理7天前的日志
0 2 * * * cd /path/to/fastapi-log-manager && python scripts/cleanup_logs.py --days 7

# 每6小时检查并清理到500MB以下
0 */6 * * * cd /path/to/fastapi-log-manager && python scripts/cleanup_logs.py --max-size 500

# 每周日凌晨3点压缩旧日志
0 3 * * 0 cd /path/to/fastapi-log-manager && python scripts/cleanup_logs.py --compress
```

## 📝 集成到现有项目

### 方法1: 直接集成（推荐）

1. 复制相关文件到你的项目：
   - `app/log_manager.py`
   - `app/scheduler.py`

2. 在你的FastAPI应用中初始化：

```python
from fastapi import FastAPI
from contextlib import asynccontextmanager
from app.log_manager import get_log_manager
from app.scheduler import get_scheduler

@asynccontextmanager
async def lifespan(app: FastAPI):
    # 启动时
    log_manager = get_log_manager()
    scheduler = get_scheduler()
    scheduler.start()
    
    yield
    
    # 关闭时
    scheduler.stop()

app = FastAPI(lifespan=lifespan)
```

### 方法2: 仅使用日志轮转

如果只需要日志轮转功能，无需定时任务：

```python
from loguru import logger

# 移除默认handler
logger.remove()

# 配置日志
logger.add(
    "logs/app_{time:YYYY-MM-DD}.log",
    rotation="00:00",           # 每天轮转
    retention="7 days",         # 保留7天
    compression="zip",          # 压缩
    level="INFO",
)

logger.add(
    "logs/app_{time}.log",
    rotation="100 MB",          # 按大小轮转
    retention="7 days",
    compression="zip",
    level="DEBUG",
)
```

### 方法3: 仅使用独立脚本

将 `scripts/cleanup_logs.py` 复制到你的项目，配置cron定时执行。

## 🎯 使用场景

### 场景1: 高日志量应用

```python
log_manager = LogManager(
    retention_days=3,           # 只保留3天
    max_file_size="50 MB",      # 更小的文件
    disk_threshold_percent=70,  # 更低的阈值
)
```

### 场景2: 日志需要长期保存

```python
log_manager = LogManager(
    retention_days=30,          # 保留30天
    compression="gz",           # 使用gzip压缩（更高压缩率）
)
```

### 场景3: 磁盘空间有限

```python
# 启用更激进的清理策略
scheduler.scheduler.add_job(
    func=lambda: log_manager.clean_by_size(max_size_mb=200),
    trigger=IntervalTrigger(hours=2),  # 每2小时检查一次
    id="aggressive_cleanup",
)
```

## 📊 日志文件结构

```
logs/
├── app_2024-01-15.log           # 按日期的普通日志
├── app_2024-01-14.log.zip       # 压缩的旧日志
├── app_size_2024-01-15_10-30-00.log  # 按大小轮转的日志
├── error_2024-01-15.log         # 错误日志
└── error_2024-01-14.log.zip     # 压缩的错误日志
```

## 🔍 监控和告警

### 集成到监控系统

```python
import requests

# 定期检查健康状态
response = requests.get("http://your-app/health")
if response.status_code == 503:
    # 发送告警
    send_alert("磁盘空间不足")
```

### Prometheus监控

```python
from prometheus_client import Gauge

disk_usage_gauge = Gauge('disk_usage_percent', '磁盘使用率')

@app.on_event("startup")
async def update_metrics():
    log_manager = get_log_manager()
    usage = log_manager.get_disk_usage()
    disk_usage_gauge.set(usage['percent'])
```

## ⚠️ 注意事项

### 1. 权限问题

确保应用有权限写入日志目录和删除文件：

```bash
chmod 755 logs
```

### 2. 磁盘空间

定期检查磁盘空间，避免完全写满：

```bash
df -h
```

### 3. 日志重要性

不要设置过短的保留期，确保重要日志不会被误删。

### 4. 压缩性能

压缩会消耗CPU资源，建议在低峰期执行。

### 5. 生产环境建议

- 使用独立的日志分区
- 配置日志轮转到外部存储（如S3）
- 设置监控告警
- 定期备份重要日志

## 🐛 故障排除

### 问题1: 日志文件未删除

**原因**: 文件被进程占用

**解决方案**: 确保日志轮转正常，使用 `retention` 参数自动删除

### 问题2: 磁盘仍然满

**原因**: 可能有其他程序占用磁盘

**解决方案**:
```bash
# 查看磁盘使用情况
du -sh /var/log/*
df -h

# 清理系统日志
sudo journalctl --vacuum-time=3d
```

### 问题3: 定时任务未执行

**原因**: 调度器未启动

**解决方案**: 检查应用日志，确保调度器启动成功

### 问题4: 性能影响

**原因**: 清理任务占用资源

**解决方案**: 调整定时任务频率，避开业务高峰期

## 📚 相关资源

- [Loguru文档](https://loguru.readthedocs.io/)
- [APScheduler文档](https://apscheduler.readthedocs.io/)
- [FastAPI文档](https://fastapi.tiangolo.com/)

## 📄 License

MIT License
