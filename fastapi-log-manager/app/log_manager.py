"""
日志管理器 - 负责日志配置、轮转和清理
"""
import os
import shutil
import time
from pathlib import Path
from datetime import datetime, timedelta
from typing import Optional, List
import psutil
from loguru import logger


class LogManager:
    """日志管理器 - 管理日志文件的生命周期"""
    
    def __init__(
        self,
        log_dir: str = "logs",
        retention_days: int = 7,
        max_file_size: str = "100 MB",
        rotation_time: str = "00:00",
        compression: str = "zip",
        disk_threshold_percent: int = 80,
    ):
        """
        初始化日志管理器
        
        Args:
            log_dir: 日志目录
            retention_days: 日志保留天数
            max_file_size: 单个日志文件最大大小
            rotation_time: 日志轮转时间（24小时制）
            compression: 压缩格式（zip, gz, tar.gz）
            disk_threshold_percent: 磁盘使用率阈值（百分比）
        """
        self.log_dir = Path(log_dir)
        self.retention_days = retention_days
        self.max_file_size = max_file_size
        self.rotation_time = rotation_time
        self.compression = compression
        self.disk_threshold_percent = disk_threshold_percent
        
        # 创建日志目录
        self.log_dir.mkdir(parents=True, exist_ok=True)
        
        # 配置日志
        self._configure_logger()
    
    def _configure_logger(self):
        """配置loguru日志"""
        # 移除默认的handler
        logger.remove()
        
        # 添加控制台输出（带颜色）
        logger.add(
            sink=lambda msg: print(msg, end=""),
            format="<green>{time:YYYY-MM-DD HH:mm:ss}</green> | "
                   "<level>{level: <8}</level> | "
                   "<cyan>{name}</cyan>:<cyan>{function}</cyan>:<cyan>{line}</cyan> | "
                   "<level>{message}</level>",
            level="INFO",
            colorize=True,
        )
        
        # 添加文件输出 - 普通日志
        logger.add(
            sink=self.log_dir / "app_{time:YYYY-MM-DD}.log",
            format="{time:YYYY-MM-DD HH:mm:ss} | {level: <8} | {name}:{function}:{line} | {message}",
            level="INFO",
            rotation=self.rotation_time,  # 每天00:00轮转
            retention=f"{self.retention_days} days",  # 保留N天
            compression=self.compression,  # 压缩旧日志
            encoding="utf-8",
        )
        
        # 添加文件输出 - 按大小轮转
        logger.add(
            sink=self.log_dir / "app_size_{time:YYYY-MM-DD_HH-mm-ss}.log",
            format="{time:YYYY-MM-DD HH:mm:ss} | {level: <8} | {name}:{function}:{line} | {message}",
            level="DEBUG",
            rotation=self.max_file_size,  # 按大小轮转
            retention=f"{self.retention_days} days",
            compression=self.compression,
            encoding="utf-8",
        )
        
        # 添加文件输出 - 错误日志单独记录
        logger.add(
            sink=self.log_dir / "error_{time:YYYY-MM-DD}.log",
            format="{time:YYYY-MM-DD HH:mm:ss} | {level: <8} | {name}:{function}:{line} | {message}\n{exception}",
            level="ERROR",
            rotation=self.rotation_time,
            retention=f"{self.retention_days * 2} days",  # 错误日志保留更久
            compression=self.compression,
            encoding="utf-8",
            backtrace=True,
            diagnose=True,
        )
        
        logger.info(f"日志管理器初始化完成，日志目录: {self.log_dir.absolute()}")
    
    def get_disk_usage(self, path: Optional[str] = None) -> dict:
        """
        获取磁盘使用情况
        
        Args:
            path: 要检查的路径，默认为日志目录
            
        Returns:
            包含磁盘使用信息的字典
        """
        if path is None:
            path = self.log_dir
        
        usage = psutil.disk_usage(str(path))
        
        return {
            "total": usage.total,
            "used": usage.used,
            "free": usage.free,
            "percent": usage.percent,
            "total_gb": round(usage.total / (1024**3), 2),
            "used_gb": round(usage.used / (1024**3), 2),
            "free_gb": round(usage.free / (1024**3), 2),
        }
    
    def is_disk_space_critical(self) -> bool:
        """检查磁盘空间是否告急"""
        usage = self.get_disk_usage()
        return usage["percent"] >= self.disk_threshold_percent
    
    def get_log_files(self, pattern: str = "*") -> List[Path]:
        """
        获取日志文件列表
        
        Args:
            pattern: 文件匹配模式
            
        Returns:
            日志文件路径列表
        """
        return sorted(self.log_dir.glob(pattern), key=lambda p: p.stat().st_mtime)
    
    def get_log_size(self) -> dict:
        """获取日志目录大小信息"""
        total_size = 0
        file_count = 0
        
        for file_path in self.log_dir.rglob("*"):
            if file_path.is_file():
                total_size += file_path.stat().st_size
                file_count += 1
        
        return {
            "total_size": total_size,
            "total_size_mb": round(total_size / (1024**2), 2),
            "total_size_gb": round(total_size / (1024**3), 2),
            "file_count": file_count,
        }
    
    def clean_old_logs(self, days: Optional[int] = None) -> dict:
        """
        清理过期的日志文件
        
        Args:
            days: 保留天数，默认使用初始化时的配置
            
        Returns:
            清理统计信息
        """
        if days is None:
            days = self.retention_days
        
        cutoff_time = time.time() - (days * 86400)
        deleted_files = []
        deleted_size = 0
        
        logger.info(f"开始清理 {days} 天前的日志文件...")
        
        for file_path in self.get_log_files("*"):
            try:
                # 跳过目录
                if file_path.is_dir():
                    continue
                
                # 检查文件修改时间
                if file_path.stat().st_mtime < cutoff_time:
                    file_size = file_path.stat().st_size
                    file_path.unlink()
                    deleted_files.append(str(file_path))
                    deleted_size += file_size
                    logger.info(f"删除过期日志: {file_path.name}")
            
            except Exception as e:
                logger.error(f"删除文件失败 {file_path}: {e}")
        
        result = {
            "deleted_count": len(deleted_files),
            "deleted_size_mb": round(deleted_size / (1024**2), 2),
            "deleted_files": deleted_files,
        }
        
        logger.info(f"清理完成: 删除 {result['deleted_count']} 个文件, "
                   f"释放 {result['deleted_size_mb']} MB")
        
        return result
    
    def clean_by_size(self, max_size_mb: int = 1000) -> dict:
        """
        按大小清理日志，保留最新的日志直到总大小低于阈值
        
        Args:
            max_size_mb: 日志目录最大大小（MB）
            
        Returns:
            清理统计信息
        """
        logger.info(f"开始按大小清理日志，目标: {max_size_mb} MB")
        
        # 获取所有日志文件，按修改时间排序（旧的在前）
        files = self.get_log_files("*")
        
        deleted_files = []
        deleted_size = 0
        current_size = sum(f.stat().st_size for f in files if f.is_file())
        max_size_bytes = max_size_mb * 1024 * 1024
        
        for file_path in files:
            if file_path.is_dir():
                continue
            
            # 如果当前大小已经低于阈值，停止删除
            if current_size <= max_size_bytes:
                break
            
            try:
                file_size = file_path.stat().st_size
                file_path.unlink()
                deleted_files.append(str(file_path))
                deleted_size += file_size
                current_size -= file_size
                logger.info(f"删除日志文件: {file_path.name}")
            
            except Exception as e:
                logger.error(f"删除文件失败 {file_path}: {e}")
        
        result = {
            "deleted_count": len(deleted_files),
            "deleted_size_mb": round(deleted_size / (1024**2), 2),
            "remaining_size_mb": round(current_size / (1024**2), 2),
            "deleted_files": deleted_files,
        }
        
        logger.info(f"清理完成: 删除 {result['deleted_count']} 个文件, "
                   f"释放 {result['deleted_size_mb']} MB, "
                   f"剩余 {result['remaining_size_mb']} MB")
        
        return result
    
    def emergency_cleanup(self) -> dict:
        """
        紧急清理 - 当磁盘空间告急时
        删除最旧的50%日志文件
        """
        logger.warning("⚠️  磁盘空间告急，执行紧急清理！")
        
        files = [f for f in self.get_log_files("*") if f.is_file()]
        files_to_delete = files[:len(files)//2]  # 删除最旧的50%
        
        deleted_files = []
        deleted_size = 0
        
        for file_path in files_to_delete:
            try:
                file_size = file_path.stat().st_size
                file_path.unlink()
                deleted_files.append(str(file_path))
                deleted_size += file_size
                logger.warning(f"紧急删除: {file_path.name}")
            
            except Exception as e:
                logger.error(f"删除文件失败 {file_path}: {e}")
        
        result = {
            "deleted_count": len(deleted_files),
            "deleted_size_mb": round(deleted_size / (1024**2), 2),
            "deleted_files": deleted_files,
        }
        
        logger.warning(f"紧急清理完成: 删除 {result['deleted_count']} 个文件, "
                      f"释放 {result['deleted_size_mb']} MB")
        
        return result
    
    def compress_old_logs(self, days: int = 3) -> dict:
        """
        压缩旧日志文件
        
        Args:
            days: 压缩多少天前的日志
            
        Returns:
            压缩统计信息
        """
        import gzip
        
        logger.info(f"开始压缩 {days} 天前的未压缩日志...")
        
        cutoff_time = time.time() - (days * 86400)
        compressed_files = []
        saved_size = 0
        
        # 查找未压缩的日志文件
        for file_path in self.get_log_files("*.log"):
            if file_path.stat().st_mtime < cutoff_time:
                try:
                    original_size = file_path.stat().st_size
                    compressed_path = file_path.with_suffix(".log.gz")
                    
                    # 压缩文件
                    with open(file_path, 'rb') as f_in:
                        with gzip.open(compressed_path, 'wb') as f_out:
                            shutil.copyfileobj(f_in, f_out)
                    
                    # 删除原文件
                    file_path.unlink()
                    
                    compressed_size = compressed_path.stat().st_size
                    saved_size += (original_size - compressed_size)
                    compressed_files.append(str(file_path))
                    
                    logger.info(f"压缩完成: {file_path.name} "
                               f"({original_size/1024:.1f}KB -> {compressed_size/1024:.1f}KB)")
                
                except Exception as e:
                    logger.error(f"压缩文件失败 {file_path}: {e}")
        
        result = {
            "compressed_count": len(compressed_files),
            "saved_size_mb": round(saved_size / (1024**2), 2),
            "compressed_files": compressed_files,
        }
        
        logger.info(f"压缩完成: {result['compressed_count']} 个文件, "
                   f"节省 {result['saved_size_mb']} MB")
        
        return result
    
    def get_status(self) -> dict:
        """获取日志管理器状态"""
        disk_usage = self.get_disk_usage()
        log_size = self.get_log_size()
        
        return {
            "log_directory": str(self.log_dir.absolute()),
            "retention_days": self.retention_days,
            "disk_usage": disk_usage,
            "disk_critical": self.is_disk_space_critical(),
            "log_size": log_size,
            "log_files": len(self.get_log_files("*")),
            "oldest_log": self._get_oldest_log_info(),
            "newest_log": self._get_newest_log_info(),
        }
    
    def _get_oldest_log_info(self) -> Optional[dict]:
        """获取最旧日志文件信息"""
        files = [f for f in self.get_log_files("*") if f.is_file()]
        if not files:
            return None
        
        oldest = files[0]
        return {
            "name": oldest.name,
            "size_mb": round(oldest.stat().st_size / (1024**2), 2),
            "modified": datetime.fromtimestamp(oldest.stat().st_mtime).isoformat(),
        }
    
    def _get_newest_log_info(self) -> Optional[dict]:
        """获取最新日志文件信息"""
        files = [f for f in self.get_log_files("*") if f.is_file()]
        if not files:
            return None
        
        newest = files[-1]
        return {
            "name": newest.name,
            "size_mb": round(newest.stat().st_size / (1024**2), 2),
            "modified": datetime.fromtimestamp(newest.stat().st_mtime).isoformat(),
        }


# 全局日志管理器实例
log_manager: Optional[LogManager] = None


def get_log_manager() -> LogManager:
    """获取日志管理器实例（单例）"""
    global log_manager
    if log_manager is None:
        log_manager = LogManager()
    return log_manager
