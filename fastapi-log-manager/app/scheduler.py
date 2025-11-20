"""
定时任务调度器 - 负责定期清理日志
"""
from apscheduler.schedulers.background import BackgroundScheduler
from apscheduler.triggers.cron import CronTrigger
from apscheduler.triggers.interval import IntervalTrigger
from loguru import logger
from typing import Optional

from app.log_manager import get_log_manager


class LogCleanupScheduler:
    """日志清理定时任务调度器"""
    
    def __init__(self):
        self.scheduler = BackgroundScheduler()
        self.log_manager = get_log_manager()
        self._is_running = False
    
    def start(self):
        """启动定时任务"""
        if self._is_running:
            logger.warning("调度器已经在运行中")
            return
        
        logger.info("启动日志清理调度器...")
        
        # 任务1: 每天凌晨2点清理过期日志
        self.scheduler.add_job(
            func=self._daily_cleanup,
            trigger=CronTrigger(hour=2, minute=0),
            id="daily_cleanup",
            name="每日日志清理",
            replace_existing=True,
        )
        
        # 任务2: 每小时检查磁盘空间
        self.scheduler.add_job(
            func=self._check_disk_space,
            trigger=IntervalTrigger(hours=1),
            id="disk_check",
            name="磁盘空间检查",
            replace_existing=True,
        )
        
        # 任务3: 每6小时检查日志大小
        self.scheduler.add_job(
            func=self._check_log_size,
            trigger=IntervalTrigger(hours=6),
            id="log_size_check",
            name="日志大小检查",
            replace_existing=True,
        )
        
        # 任务4: 每周压缩旧日志
        self.scheduler.add_job(
            func=self._weekly_compression,
            trigger=CronTrigger(day_of_week='sun', hour=3, minute=0),
            id="weekly_compression",
            name="每周日志压缩",
            replace_existing=True,
        )
        
        self.scheduler.start()
        self._is_running = True
        
        logger.info("✅ 日志清理调度器启动成功")
        self._print_scheduled_jobs()
    
    def stop(self):
        """停止定时任务"""
        if not self._is_running:
            return
        
        logger.info("停止日志清理调度器...")
        self.scheduler.shutdown()
        self._is_running = False
        logger.info("✅ 日志清理调度器已停止")
    
    def _daily_cleanup(self):
        """每日清理任务"""
        logger.info("🔄 执行每日日志清理任务...")
        try:
            result = self.log_manager.clean_old_logs()
            logger.info(f"✅ 每日清理完成: {result}")
        except Exception as e:
            logger.error(f"❌ 每日清理失败: {e}")
    
    def _check_disk_space(self):
        """检查磁盘空间"""
        logger.debug("检查磁盘空间...")
        try:
            if self.log_manager.is_disk_space_critical():
                logger.warning("⚠️  磁盘空间告急，执行紧急清理")
                result = self.log_manager.emergency_cleanup()
                logger.warning(f"紧急清理结果: {result}")
            else:
                usage = self.log_manager.get_disk_usage()
                logger.debug(f"磁盘使用率: {usage['percent']}%")
        except Exception as e:
            logger.error(f"❌ 磁盘空间检查失败: {e}")
    
    def _check_log_size(self):
        """检查日志大小"""
        logger.debug("检查日志目录大小...")
        try:
            log_size = self.log_manager.get_log_size()
            logger.debug(f"日志目录大小: {log_size['total_size_mb']} MB")
            
            # 如果日志超过1GB，清理到500MB以下
            if log_size['total_size_mb'] > 1000:
                logger.info(f"日志大小超过1GB ({log_size['total_size_mb']} MB)，执行清理")
                result = self.log_manager.clean_by_size(max_size_mb=500)
                logger.info(f"清理结果: {result}")
        except Exception as e:
            logger.error(f"❌ 日志大小检查失败: {e}")
    
    def _weekly_compression(self):
        """每周压缩任务"""
        logger.info("🔄 执行每周日志压缩任务...")
        try:
            result = self.log_manager.compress_old_logs(days=3)
            logger.info(f"✅ 压缩完成: {result}")
        except Exception as e:
            logger.error(f"❌ 压缩失败: {e}")
    
    def _print_scheduled_jobs(self):
        """打印已调度的任务"""
        logger.info("已调度的任务:")
        for job in self.scheduler.get_jobs():
            logger.info(f"  - {job.name} (ID: {job.id})")
            logger.info(f"    下次运行: {job.next_run_time}")
    
    def get_jobs_status(self) -> list:
        """获取所有任务状态"""
        jobs = []
        for job in self.scheduler.get_jobs():
            jobs.append({
                "id": job.id,
                "name": job.name,
                "next_run": job.next_run_time.isoformat() if job.next_run_time else None,
                "trigger": str(job.trigger),
            })
        return jobs
    
    def trigger_job(self, job_id: str):
        """手动触发任务"""
        logger.info(f"手动触发任务: {job_id}")
        job = self.scheduler.get_job(job_id)
        if job:
            job.modify(next_run_time=None)  # 立即执行
            logger.info(f"任务 {job_id} 已触发")
        else:
            logger.error(f"任务 {job_id} 不存在")


# 全局调度器实例
scheduler: Optional[LogCleanupScheduler] = None


def get_scheduler() -> LogCleanupScheduler:
    """获取调度器实例（单例）"""
    global scheduler
    if scheduler is None:
        scheduler = LogCleanupScheduler()
    return scheduler
