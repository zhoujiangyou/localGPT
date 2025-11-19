"""
FastAPI主应用 - 集成日志管理和清理功能
"""
from fastapi import FastAPI, HTTPException
from fastapi.responses import JSONResponse
from contextlib import asynccontextmanager
from loguru import logger
from typing import Optional

from app.log_manager import get_log_manager
from app.scheduler import get_scheduler


# 应用生命周期管理
@asynccontextmanager
async def lifespan(app: FastAPI):
    """应用启动和关闭时的操作"""
    # 启动时
    logger.info("=" * 60)
    logger.info("🚀 FastAPI应用启动中...")
    logger.info("=" * 60)
    
    # 初始化日志管理器
    log_manager = get_log_manager()
    logger.info("✅ 日志管理器初始化完成")
    
    # 启动定时清理任务
    scheduler = get_scheduler()
    scheduler.start()
    logger.info("✅ 定时清理任务启动完成")
    
    # 打印初始状态
    status = log_manager.get_status()
    logger.info(f"📊 日志目录: {status['log_directory']}")
    logger.info(f"📊 磁盘使用率: {status['disk_usage']['percent']}%")
    logger.info(f"📊 日志文件数: {status['log_files']}")
    logger.info(f"📊 日志总大小: {status['log_size']['total_size_mb']} MB")
    
    logger.info("=" * 60)
    logger.info("✅ FastAPI应用启动完成")
    logger.info("=" * 60)
    
    yield
    
    # 关闭时
    logger.info("=" * 60)
    logger.info("🛑 FastAPI应用关闭中...")
    logger.info("=" * 60)
    
    scheduler.stop()
    logger.info("✅ 定时任务已停止")
    
    logger.info("=" * 60)
    logger.info("✅ FastAPI应用已关闭")
    logger.info("=" * 60)


# 创建FastAPI应用
app = FastAPI(
    title="FastAPI日志管理示例",
    description="集成自动日志清理和管理功能的FastAPI应用",
    version="1.0.0",
    lifespan=lifespan,
)


@app.get("/")
async def root():
    """根路径"""
    logger.info("访问根路径")
    return {
        "message": "FastAPI日志管理示例应用",
        "endpoints": {
            "status": "/log/status",
            "cleanup": "/log/cleanup",
            "compress": "/log/compress",
            "disk": "/log/disk",
            "jobs": "/log/jobs",
        }
    }


@app.get("/log/status")
async def get_log_status():
    """获取日志状态"""
    logger.info("查询日志状态")
    try:
        log_manager = get_log_manager()
        status = log_manager.get_status()
        return JSONResponse(content=status)
    except Exception as e:
        logger.error(f"获取日志状态失败: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/log/cleanup")
async def cleanup_logs(days: Optional[int] = None):
    """
    手动清理日志
    
    Args:
        days: 保留天数，不指定则使用默认配置
    """
    logger.info(f"手动触发日志清理，保留天数: {days}")
    try:
        log_manager = get_log_manager()
        
        if days is not None:
            result = log_manager.clean_old_logs(days=days)
        else:
            result = log_manager.clean_old_logs()
        
        return JSONResponse(content={
            "success": True,
            "message": "日志清理完成",
            "result": result
        })
    except Exception as e:
        logger.error(f"清理日志失败: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/log/cleanup/size")
async def cleanup_by_size(max_size_mb: int = 500):
    """
    按大小清理日志
    
    Args:
        max_size_mb: 日志目录最大大小（MB）
    """
    logger.info(f"按大小清理日志，目标: {max_size_mb} MB")
    try:
        log_manager = get_log_manager()
        result = log_manager.clean_by_size(max_size_mb=max_size_mb)
        
        return JSONResponse(content={
            "success": True,
            "message": "日志清理完成",
            "result": result
        })
    except Exception as e:
        logger.error(f"清理日志失败: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/log/cleanup/emergency")
async def emergency_cleanup():
    """紧急清理日志（删除最旧的50%）"""
    logger.warning("触发紧急清理")
    try:
        log_manager = get_log_manager()
        result = log_manager.emergency_cleanup()
        
        return JSONResponse(content={
            "success": True,
            "message": "紧急清理完成",
            "result": result
        })
    except Exception as e:
        logger.error(f"紧急清理失败: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/log/compress")
async def compress_logs(days: int = 3):
    """
    压缩旧日志
    
    Args:
        days: 压缩多少天前的日志
    """
    logger.info(f"压缩 {days} 天前的日志")
    try:
        log_manager = get_log_manager()
        result = log_manager.compress_old_logs(days=days)
        
        return JSONResponse(content={
            "success": True,
            "message": "日志压缩完成",
            "result": result
        })
    except Exception as e:
        logger.error(f"压缩日志失败: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/log/disk")
async def get_disk_usage():
    """获取磁盘使用情况"""
    logger.debug("查询磁盘使用情况")
    try:
        log_manager = get_log_manager()
        usage = log_manager.get_disk_usage()
        
        return JSONResponse(content={
            "disk_usage": usage,
            "critical": log_manager.is_disk_space_critical(),
            "threshold": log_manager.disk_threshold_percent,
        })
    except Exception as e:
        logger.error(f"获取磁盘使用情况失败: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/log/jobs")
async def get_scheduled_jobs():
    """获取定时任务状态"""
    logger.debug("查询定时任务状态")
    try:
        scheduler = get_scheduler()
        jobs = scheduler.get_jobs_status()
        
        return JSONResponse(content={
            "jobs": jobs,
            "count": len(jobs),
        })
    except Exception as e:
        logger.error(f"获取定时任务状态失败: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/log/jobs/{job_id}/trigger")
async def trigger_job(job_id: str):
    """手动触发定时任务"""
    logger.info(f"手动触发任务: {job_id}")
    try:
        scheduler = get_scheduler()
        scheduler.trigger_job(job_id)
        
        return JSONResponse(content={
            "success": True,
            "message": f"任务 {job_id} 已触发"
        })
    except Exception as e:
        logger.error(f"触发任务失败: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/test/generate-logs")
async def generate_test_logs(count: int = 100):
    """生成测试日志（用于测试）"""
    logger.info(f"生成 {count} 条测试日志")
    
    for i in range(count):
        if i % 10 == 0:
            logger.error(f"测试错误日志 {i}")
        elif i % 5 == 0:
            logger.warning(f"测试警告日志 {i}")
        else:
            logger.info(f"测试信息日志 {i}")
    
    return JSONResponse(content={
        "success": True,
        "message": f"已生成 {count} 条测试日志"
    })


# 健康检查端点
@app.get("/health")
async def health_check():
    """健康检查"""
    try:
        log_manager = get_log_manager()
        disk_usage = log_manager.get_disk_usage()
        
        # 如果磁盘使用率超过90%，返回不健康
        if disk_usage["percent"] >= 90:
            return JSONResponse(
                status_code=503,
                content={
                    "status": "unhealthy",
                    "reason": f"磁盘使用率过高: {disk_usage['percent']}%"
                }
            )
        
        return JSONResponse(content={
            "status": "healthy",
            "disk_usage_percent": disk_usage["percent"]
        })
    except Exception as e:
        logger.error(f"健康检查失败: {e}")
        return JSONResponse(
            status_code=503,
            content={"status": "unhealthy", "reason": str(e)}
        )


if __name__ == "__main__":
    import uvicorn
    
    uvicorn.run(
        "app.main:app",
        host="0.0.0.0",
        port=8000,
        reload=True,
        log_level="info",
    )
