-- 任务分片表
CREATE TABLE IF NOT EXISTS shard_tasks (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    shard_path VARCHAR(1024) NOT NULL,
    status ENUM('pending', 'processing', 'completed', 'failed') DEFAULT 'pending',
    assigned_node_id VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_status (status),
    INDEX idx_assigned_node (assigned_node_id)
);

-- 文件任务表
CREATE TABLE IF NOT EXISTS file_tasks (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    shard_id BIGINT, -- 关联到 shard_tasks
    source_path VARCHAR(2048) NOT NULL,
    destination_key VARCHAR(2048) NOT NULL,
    file_size BIGINT,
    md5_checksum VARCHAR(32),
    status ENUM('pending', 'processing', 'completed', 'failed') DEFAULT 'pending',
    retry_count INT DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_status (status),
    INDEX idx_shard_id (shard_id),
    INDEX idx_source_path (source_path(255)) -- 限制索引长度
);
