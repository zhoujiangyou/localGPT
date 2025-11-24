package models

import "time"

type ShardTask struct {
	ID             int64     `db:"id"`
	ShardPath      string    `db:"shard_path"`
	Status         string    `db:"status"`
	AssignedNodeID *string   `db:"assigned_node_id"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

type FileTask struct {
	ID             int64     `db:"id"`
	ShardID        int64     `db:"shard_id"`
	SourcePath     string    `db:"source_path"`
	DestinationKey string    `db:"destination_key"`
	FileSize       int64     `db:"file_size"`
	MD5Checksum    string    `db:"md5_checksum"`
	Status         string    `db:"status"`
	RetryCount     int       `db:"retry_count"`
	LastError      *string   `db:"last_error"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}
