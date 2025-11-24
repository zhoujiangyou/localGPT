package main

import (
	"flag"
	"log"

	"migration_tool/internal/config"
	"migration_tool/internal/controller"
	"migration_tool/internal/db"
	"migration_tool/internal/s3"
	"migration_tool/internal/worker"
    
    _ "github.com/go-sql-driver/mysql" // Import MySQL driver
)

func main() {
	mode := flag.String("mode", "worker", "Mode to run in: 'controller' or 'worker'")
	flag.Parse()

	// Load Config
	cfg := config.LoadConfig()

	// Connect to DB
    // Note: cfg.DBConnectionString should look like "user:password@tcp(host:port)/dbname"
	database, err := db.Connect(cfg.DBConnectionString)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	repo := db.NewRepository(database)

	if *mode == "controller" {
		ctrl := controller.NewController(cfg, repo)
		ctrl.Run()
	} else if *mode == "worker" {
        // Initialize S3 Client only for workers
        s3Client, err := s3.NewClient(cfg)
        if err != nil {
            log.Fatalf("Failed to initialize S3 client: %v", err)
        }
        
		w := worker.NewWorker(cfg, repo, s3Client)
		w.Run()
	} else {
		log.Fatalf("Unknown mode: %s", *mode)
	}
}
