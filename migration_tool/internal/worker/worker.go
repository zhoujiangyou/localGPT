package worker

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"migration_tool/internal/config"
	"migration_tool/internal/db"
	"migration_tool/internal/models"
	ks3 "migration_tool/internal/s3"
)

type Worker struct {
	Cfg      *config.Config
	Repo     *db.Repository
	S3Client *ks3.Client
	NodeID   string
}

func NewWorker(cfg *config.Config, repo *db.Repository, s3Client *ks3.Client) *Worker {
	return &Worker{
		Cfg:      cfg,
		Repo:     repo,
		S3Client: s3Client,
		NodeID:   cfg.NodeID,
	}
}

func (w *Worker) Run() {
	log.Printf("Worker %s started", w.NodeID)

	// Start the File Upload Consumers (Worker Pool)
	// They will poll the DB for file tasks independently of the Shard Processor
	// This allows this node to help with uploads even if it hasn't claimed a shard,
    // or if it finishes scanning a shard but uploads are pending.
    // However, usually we want to process the shard we claimed. 
    // The DB design allows any worker to pick up any file task.
    
	var wg sync.WaitGroup
	for i := 0; i < w.Cfg.WorkerPoolSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.uploadLoop()
		}()
	}

	// Main loop to claim and process shards (The Producer part)
	for {
		// 1. Claim a Shard Task
		shardTask, err := w.Repo.ClaimShardTask(w.NodeID)
		if err != nil {
			log.Printf("Error claiming shard task: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if shardTask == nil {
			// No more shards to process
			// But we should keep the upload workers running until all file tasks are done?
			// For now, just sleep and check again later (maybe new shards added)
            // Or maybe we just focus on uploads.
			log.Println("No pending shard tasks found. Waiting...")
			time.Sleep(10 * time.Second)
			continue
		}

		log.Printf("Claimed shard: %s", shardTask.ShardPath)

		// 2. Process the Shard (Discover Files)
		w.processShard(shardTask)
        
        // 3. Mark Shard as Completed (Discovery phase completed)
        // Note: This only means *discovery* is done. Uploads might still be pending in file_tasks.
        // That's fine, the upload workers will handle them.
        err = w.Repo.UpdateShardStatus(shardTask.ID, "completed")
        if err != nil {
            log.Printf("Failed to mark shard %d as completed: %v", shardTask.ID, err)
        }
	}
    
    // In a real app we'd have a shutdown signal handling here to wait for wg
}

func (w *Worker) processShard(task *models.ShardTask) {
	// Verify the path matches our NAS Root logic if needed, 
    // but here we assume ShardPath is absolute or relative to where we mount the NAS.
    
	err := filepath.Walk(task.ShardPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error accessing path %s: %v", path, err)
			return nil // Continue
		}
		if info.IsDir() {
			return nil
		}

		// Create File Task
		ft := models.FileTask{
			ShardID:        task.ID,
			SourcePath:     path,
			DestinationKey: w.generateDestKey(path),
			FileSize:       info.Size(),
            Status:         "pending",
		}

        // Optimization: Batch inserts could be done here.
        // For simplicity, we insert one by one or use a small buffer.
        // Let's do one by one for safety now, or use a method to buffer.
        // Given the scale (PB), batching is mandatory.
        // Let's fake a batch insert call or just insert.
        // For this implementation, I will just insert individually for simplicity of code,
        // but acknowledge batching is needed.
        
		err = w.Repo.CreateFileTasks([]models.FileTask{ft})
        if err != nil {
            log.Printf("Failed to create file task for %s: %v", path, err)
        }
        
		return nil
	})

	if err != nil {
		log.Printf("Error walking shard %s: %v", task.ShardPath, err)
        w.Repo.UpdateShardStatus(task.ID, "failed") // Mark shard as failed if walk failed
	}
}

func (w *Worker) generateDestKey(path string) string {
    // Logic to convert local path to S3 key
    // e.g. /mnt/nas/data/A/file.txt -> data/A/file.txt
    // Removing the common prefix
    rel, err := filepath.Rel(w.Cfg.NASRoot, path)
    if err != nil {
        return path // fallback
    }
    return rel
}

func (w *Worker) uploadLoop() {
    for {
        tasks, err := w.Repo.GetPendingFileTasks(w.Cfg.DBBatchSize)
        if err != nil {
            log.Printf("Error fetching file tasks: %v", err)
            time.Sleep(1 * time.Second)
            continue
        }
        
        if len(tasks) == 0 {
            time.Sleep(1 * time.Second)
            continue
        }
        
        for _, task := range tasks {
            w.performUpload(task)
        }
    }
}

func (w *Worker) performUpload(task models.FileTask) {
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour) // Large timeout for big files
    defer cancel()
    
    err := w.S3Client.UploadFile(ctx, task.SourcePath, task.DestinationKey)
    if err != nil {
        log.Printf("Failed to upload %s: %v", task.SourcePath, err)
        if task.RetryCount < w.Cfg.RetryLimit {
            w.Repo.UpdateFileTaskRetry(task.ID, "pending", err.Error()) // Reset to pending for retry
        } else {
            w.Repo.UpdateFileTaskStatus(task.ID, "failed", err.Error())
        }
        return
    }
    
    w.Repo.UpdateFileTaskStatus(task.ID, "completed", "")
}
