#!/usr/bin/env python3
"""
独立的日志清理脚本
可以通过cron定时执行
"""
import sys
import argparse
from pathlib import Path

# 添加项目根目录到路径
sys.path.insert(0, str(Path(__file__).parent.parent))

from app.log_manager import LogManager
from loguru import logger


def main():
    parser = argparse.ArgumentParser(description="日志清理脚本")
    
    parser.add_argument(
        "--log-dir",
        type=str,
        default="logs",
        help="日志目录路径"
    )
    
    parser.add_argument(
        "--days",
        type=int,
        default=7,
        help="保留天数"
    )
    
    parser.add_argument(
        "--max-size",
        type=int,
        help="按大小清理，指定最大大小（MB）"
    )
    
    parser.add_argument(
        "--compress",
        action="store_true",
        help="压缩旧日志"
    )
    
    parser.add_argument(
        "--compress-days",
        type=int,
        default=3,
        help="压缩多少天前的日志"
    )
    
    parser.add_argument(
        "--emergency",
        action="store_true",
        help="紧急清理（删除50%最旧日志）"
    )
    
    args = parser.parse_args()
    
    # 初始化日志管理器
    log_manager = LogManager(
        log_dir=args.log_dir,
        retention_days=args.days,
    )
    
    logger.info("=" * 60)
    logger.info("日志清理脚本启动")
    logger.info("=" * 60)
    
    # 打印当前状态
    status = log_manager.get_status()
    logger.info(f"日志目录: {status['log_directory']}")
    logger.info(f"日志文件数: {status['log_files']}")
    logger.info(f"日志总大小: {status['log_size']['total_size_mb']} MB")
    logger.info(f"磁盘使用率: {status['disk_usage']['percent']}%")
    logger.info("=" * 60)
    
    # 执行清理
    if args.emergency:
        logger.warning("执行紧急清理...")
        result = log_manager.emergency_cleanup()
        logger.info(f"清理结果: {result}")
    
    elif args.max_size:
        logger.info(f"按大小清理，目标: {args.max_size} MB")
        result = log_manager.clean_by_size(max_size_mb=args.max_size)
        logger.info(f"清理结果: {result}")
    
    else:
        logger.info(f"按时间清理，保留 {args.days} 天")
        result = log_manager.clean_old_logs(days=args.days)
        logger.info(f"清理结果: {result}")
    
    # 压缩旧日志
    if args.compress:
        logger.info(f"压缩 {args.compress_days} 天前的日志")
        result = log_manager.compress_old_logs(days=args.compress_days)
        logger.info(f"压缩结果: {result}")
    
    # 打印最终状态
    logger.info("=" * 60)
    status = log_manager.get_status()
    logger.info(f"清理后日志文件数: {status['log_files']}")
    logger.info(f"清理后日志总大小: {status['log_size']['total_size_mb']} MB")
    logger.info(f"清理后磁盘使用率: {status['disk_usage']['percent']}%")
    logger.info("=" * 60)
    logger.info("✅ 日志清理完成")
    logger.info("=" * 60)


if __name__ == "__main__":
    main()
