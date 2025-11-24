package db

import (
	"database/sql"
	"fmt"
	"strings"

	"migration_tool/internal/models"
)

type Repository struct {
	DB *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{DB: db}
}

// CreateShardTasks batches insertion of shard tasks
func (r *Repository) CreateShardTasks(shards []string) error {
	if len(shards) == 0 {
		return nil
	}
	
	// Construct query
	query := "INSERT INTO shard_tasks (shard_path, status) VALUES "
	vals := []interface{}{}
	placeholders := []string{}
	
	for _, s := range shards {
		placeholders = append(placeholders, "(?, 'pending')")
		vals = append(vals, s)
	}
	query += strings.Join(placeholders, ",")
	
	_, err := r.DB.Exec(query, vals...)
	return err
}

// ClaimShardTask attempts to claim a pending shard task for this node
// Uses FOR UPDATE SKIP LOCKED for MySQL 8.0+ / PostgreSQL
func (r *Repository) ClaimShardTask(nodeID string) (*models.ShardTask, error) {
	tx, err := r.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Find a pending task
    // Note: SQLite does not support SKIP LOCKED. This query is optimized for MySQL.
	row := tx.QueryRow("SELECT id, shard_path FROM shard_tasks WHERE status = 'pending' LIMIT 1 FOR UPDATE SKIP LOCKED")
	
	var task models.ShardTask
	err = row.Scan(&task.ID, &task.ShardPath)
	if err == sql.ErrNoRows {
		return nil, nil // No tasks available
	} else if err != nil {
        // Fallback for databases that don't support SKIP LOCKED (like SQLite or older MySQL)
		return nil, err
	}

	// Update status to processing
	_, err = tx.Exec("UPDATE shard_tasks SET status = 'processing', assigned_node_id = ?, updated_at = NOW() WHERE id = ?", nodeID, task.ID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

    task.Status = "processing"
    assigned := nodeID
    task.AssignedNodeID = &assigned
	return &task, nil
}

// UpdateShardStatus updates the status of a shard task
func (r *Repository) UpdateShardStatus(id int64, status string) error {
	_, err := r.DB.Exec("UPDATE shard_tasks SET status = ?, updated_at = NOW() WHERE id = ?", status, id)
	return err
}

// CreateFileTasks batches insertion of file tasks
func (r *Repository) CreateFileTasks(tasks []models.FileTask) error {
	if len(tasks) == 0 {
		return nil
	}
	
	query := "INSERT INTO file_tasks (shard_id, source_path, destination_key, file_size, status) VALUES "
	vals := []interface{}{}
	placeholders := []string{}
	
	for _, t := range tasks {
		placeholders = append(placeholders, "(?, ?, ?, ?, 'pending')")
		vals = append(vals, t.ShardID, t.SourcePath, t.DestinationKey, t.FileSize)
	}
	query += strings.Join(placeholders, ",")
	
	_, err := r.DB.Exec(query, vals...)
	return err
}

// GetPendingFileTasks fetches a batch of pending file tasks
func (r *Repository) GetPendingFileTasks(limit int) ([]models.FileTask, error) {
	tx, err := r.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.Query("SELECT id, source_path, destination_key, file_size FROM file_tasks WHERE status = 'pending' LIMIT ? FOR UPDATE SKIP LOCKED", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.FileTask
    var ids []interface{}
	for rows.Next() {
		var t models.FileTask
		if err := rows.Scan(&t.ID, &t.SourcePath, &t.DestinationKey, &t.FileSize); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
        ids = append(ids, t.ID)
	}
    
    if len(tasks) == 0 {
        return tasks, nil
    }

    // Mark as processing
    placeholders := strings.Repeat("?,", len(ids))
    placeholders = placeholders[:len(placeholders)-1]
    query := fmt.Sprintf("UPDATE file_tasks SET status = 'processing', updated_at = NOW() WHERE id IN (%s)", placeholders)
	_, err = tx.Exec(query, ids...)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (r *Repository) UpdateFileTaskStatus(id int64, status string, lastError string) error {
    var err error
    if lastError != "" {
	    _, err = r.DB.Exec("UPDATE file_tasks SET status = ?, last_error = ?, updated_at = NOW() WHERE id = ?", status, lastError, id)
    } else {
        _, err = r.DB.Exec("UPDATE file_tasks SET status = ?, updated_at = NOW() WHERE id = ?", status, id)
    }
	return err
}

func (r *Repository) UpdateFileTaskRetry(id int64, status string, lastError string) error {
    _, err := r.DB.Exec("UPDATE file_tasks SET status = ?, retry_count = retry_count + 1, last_error = ?, updated_at = NOW() WHERE id = ?", status, lastError, id)
    return err
}
