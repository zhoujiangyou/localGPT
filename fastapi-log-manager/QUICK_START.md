# 🚀 快速开始指南

## 1️⃣ 安装（30秒）

```bash
cd /workspace/fastapi-log-manager
pip install -r requirements.txt
```

## 2️⃣ 启动应用（10秒）

```bash
python -m app.main
```

或使用uvicorn：

```bash
uvicorn app.main:app --reload
```

启动成功后访问：**http://localhost:8000**

## 3️⃣ 测试功能（2分钟）

### 生成测试日志

```bash
curl http://localhost:8000/test/generate-logs?count=1000
```

### 查看日志状态

```bash
curl http://localhost:8000/log/status
```

### 手动清理日志

```bash
curl -X POST http://localhost:8000/log/cleanup?days=7
```

### 查看磁盘使用情况

```bash
curl http://localhost:8000/log/disk
```

## 4️⃣ 查看结果

日志文件位于 `logs/` 目录：

```bash
ls -lh logs/
```

## 🎯 核心功能速览

| 功能 | API端点 | 说明 |
|------|---------|------|
| 查看状态 | GET `/log/status` | 日志和磁盘状态 |
| 手动清理 | POST `/log/cleanup` | 清理过期日志 |
| 按大小清理 | POST `/log/cleanup/size` | 清理到指定大小 |
| 紧急清理 | POST `/log/cleanup/emergency` | 删除50%旧日志 |
| 压缩日志 | POST `/log/compress` | 压缩旧日志 |
| 查看任务 | GET `/log/jobs` | 定时任务状态 |
| 健康检查 | GET `/health` | 应用健康状态 |

## 📝 集成到现有项目

### 最简单的方式

复制这两个文件到你的项目：
- `app/log_manager.py`
- `app/scheduler.py`

然后在你的FastAPI应用中：

```python
from fastapi import FastAPI
from contextlib import asynccontextmanager
from app.log_manager import get_log_manager
from app.scheduler import get_scheduler

@asynccontextmanager
async def lifespan(app: FastAPI):
    # 启动
    get_log_manager()
    get_scheduler().start()
    yield
    # 关闭
    get_scheduler().stop()

app = FastAPI(lifespan=lifespan)
```

完成！你的应用现在有了自动日志清理功能。

## 🔧 自定义配置

```python
from app.log_manager import LogManager

log_manager = LogManager(
    log_dir="logs",              # 日志目录
    retention_days=7,            # 保留7天
    max_file_size="100 MB",      # 单文件100MB
    disk_threshold_percent=80,   # 磁盘阈值80%
)
```

## 📊 定时任务说明

应用启动后自动运行以下任务：

1. **每日清理** - 每天02:00清理过期日志
2. **磁盘检查** - 每小时检查磁盘空间
3. **大小检查** - 每6小时检查日志大小
4. **每周压缩** - 每周日03:00压缩旧日志

## 🛠️ 独立脚本

如果不想集成到应用，可以用独立脚本 + cron：

```bash
# 清理日志
python scripts/cleanup_logs.py --days 7

# 添加到crontab
0 2 * * * cd /path/to/project && python scripts/cleanup_logs.py --days 7
```

## ⚠️ 常见问题

### Q: 日志在哪里？
A: 默认在 `logs/` 目录

### Q: 如何修改保留天数？
A: 修改 `LogManager` 初始化参数 `retention_days`

### Q: 磁盘还是满了怎么办？
A: 调用紧急清理接口：`POST /log/cleanup/emergency`

### Q: 如何停止定时任务？
A: 定时任务会在应用关闭时自动停止

## 🎓 下一步

- 查看 [README.md](README.md) 了解详细功能
- 查看 [API文档](http://localhost:8000/docs)
- 根据需求自定义配置

---

**就这么简单！** 🎉
