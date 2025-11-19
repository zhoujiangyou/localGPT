"""
测试脚本 - 测试日志管理功能
"""
import time
from pathlib import Path
from app.log_manager import LogManager
from loguru import logger


def test_log_rotation():
    """测试日志轮转"""
    print("\n" + "=" * 60)
    print("测试1: 日志轮转功能")
    print("=" * 60)
    
    log_manager = LogManager(log_dir="test_logs", retention_days=1)
    
    # 生成大量日志
    for i in range(1000):
        logger.info(f"测试日志 {i}")
        if i % 100 == 0:
            logger.error(f"测试错误日志 {i}")
    
    # 检查日志文件
    log_files = log_manager.get_log_files()
    print(f"✅ 生成了 {len(log_files)} 个日志文件")
    
    log_size = log_manager.get_log_size()
    print(f"✅ 日志总大小: {log_size['total_size_mb']} MB")


def test_cleanup():
    """测试日志清理"""
    print("\n" + "=" * 60)
    print("测试2: 日志清理功能")
    print("=" * 60)
    
    log_manager = LogManager(log_dir="test_logs")
    
    # 创建旧日志文件（模拟）
    test_file = Path("test_logs/old_test.log")
    test_file.write_text("old log content")
    
    # 修改文件时间为2天前
    old_time = time.time() - (2 * 86400)
    import os
    os.utime(test_file, (old_time, old_time))
    
    print("📁 创建了一个旧日志文件")
    
    # 清理1天前的日志
    result = log_manager.clean_old_logs(days=1)
    print(f"✅ 清理结果: {result}")


def test_disk_usage():
    """测试磁盘监控"""
    print("\n" + "=" * 60)
    print("测试3: 磁盘使用监控")
    print("=" * 60)
    
    log_manager = LogManager(log_dir="test_logs")
    
    usage = log_manager.get_disk_usage()
    print(f"📊 磁盘总空间: {usage['total_gb']} GB")
    print(f"📊 已使用: {usage['used_gb']} GB ({usage['percent']}%)")
    print(f"📊 可用: {usage['free_gb']} GB")
    
    if log_manager.is_disk_space_critical():
        print("⚠️  磁盘空间告急！")
    else:
        print("✅ 磁盘空间充足")


def test_compress():
    """测试日志压缩"""
    print("\n" + "=" * 60)
    print("测试4: 日志压缩功能")
    print("=" * 60)
    
    log_manager = LogManager(log_dir="test_logs")
    
    # 创建测试日志
    test_file = Path("test_logs/compress_test.log")
    test_file.write_text("test content\n" * 1000)
    
    # 修改文件时间为4天前
    old_time = time.time() - (4 * 86400)
    import os
    os.utime(test_file, (old_time, old_time))
    
    print("📁 创建了测试日志文件")
    
    # 压缩
    result = log_manager.compress_old_logs(days=3)
    print(f"✅ 压缩结果: {result}")


def test_status():
    """测试状态查询"""
    print("\n" + "=" * 60)
    print("测试5: 状态查询")
    print("=" * 60)
    
    log_manager = LogManager(log_dir="test_logs")
    
    status = log_manager.get_status()
    
    print(f"📁 日志目录: {status['log_directory']}")
    print(f"📊 日志文件数: {status['log_files']}")
    print(f"💾 日志大小: {status['log_size']['total_size_mb']} MB")
    print(f"⏰ 保留天数: {status['retention_days']}")
    
    if status['oldest_log']:
        print(f"🕒 最旧日志: {status['oldest_log']['name']}")
    if status['newest_log']:
        print(f"🕐 最新日志: {status['newest_log']['name']}")


def cleanup_test_files():
    """清理测试文件"""
    import shutil
    test_dir = Path("test_logs")
    if test_dir.exists():
        shutil.rmtree(test_dir)
        print("\n🧹 已清理测试文件")


if __name__ == "__main__":
    print("\n" + "=" * 60)
    print("FastAPI 日志管理器 - 功能测试")
    print("=" * 60)
    
    try:
        test_log_rotation()
        test_cleanup()
        test_disk_usage()
        test_compress()
        test_status()
        
        print("\n" + "=" * 60)
        print("✅ 所有测试通过！")
        print("=" * 60)
    
    except Exception as e:
        print(f"\n❌ 测试失败: {e}")
        import traceback
        traceback.print_exc()
    
    finally:
        cleanup_test_files()
