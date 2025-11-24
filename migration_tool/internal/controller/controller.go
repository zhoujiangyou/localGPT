package controller

import (
	"log"
	"os"
	"path/filepath"

	"migration_tool/internal/config"
	"migration_tool/internal/db"
)

type Controller struct {
	Cfg  *config.Config
	Repo *db.Repository
}

func NewController(cfg *config.Config, repo *db.Repository) *Controller {
	return &Controller{
		Cfg:  cfg,
		Repo: repo,
	}
}

func (c *Controller) Run() {
	log.Println("Controller started. Scanning for shards...")

	root := c.Cfg.NASRoot
	entries, err := os.ReadDir(root)
	if err != nil {
		log.Fatalf("Failed to read NAS root directory %s: %v", root, err)
	}

	var shards []string
	for _, entry := range entries {
		if entry.IsDir() {
			fullPath := filepath.Join(root, entry.Name())
			shards = append(shards, fullPath)
		}
	}

	log.Printf("Found %d shards (top-level directories).", len(shards))

	// Batch insert shards
    // In a real scenario with thousands of top level dirs, we'd batch this loop.
	err = c.Repo.CreateShardTasks(shards)
	if err != nil {
		log.Fatalf("Failed to create shard tasks: %v", err)
	}

	log.Println("Successfully initialized shard tasks.")
}
