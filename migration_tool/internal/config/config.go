package config

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	DBConnectionString string `mapstructure:"db_connection_string"`
	NodeID             string `mapstructure:"node_id"`
	NASRoot            string `mapstructure:"nas_root"`
	KS3Region          string `mapstructure:"ks3_region"`
	KS3Endpoint        string `mapstructure:"ks3_endpoint"`
	KS3Bucket          string `mapstructure:"ks3_bucket"`
	WorkerPoolSize     int    `mapstructure:"worker_pool_size"`
	DBBatchSize        int    `mapstructure:"db_batch_size"`
	RetryLimit         int    `mapstructure:"retry_limit"`
}

func LoadConfig() *Config {
	viper.SetDefault("worker_pool_size", 50)
	viper.SetDefault("db_batch_size", 100)
	viper.SetDefault("retry_limit", 3)
	viper.SetDefault("node_id", "worker-1") // Default, should be unique per pod
    
    // Environment variables
    viper.AutomaticEnv()

	// Optional: Read from config file
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
    viper.AddConfigPath("/etc/migration_tool/")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if using env vars
            log.Println("No config file found, using defaults and env vars")
		} else {
			log.Fatalf("Error reading config file: %v", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("Unable to decode into struct: %v", err)
	}
    
    // Basic validation
    if cfg.NodeID == "" {
        hostname, _ := os.Hostname()
        cfg.NodeID = hostname
    }

	return &cfg
}
