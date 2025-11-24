# 300PB File Migration Tool

A distributed, high-performance file migration system designed to move massive amounts of data (PB scale) from local NAS to Kingsoft Cloud KS3 (compatible with AWS S3).

## Architecture

The system consists of two main components:

1.  **Controller Node**:
    *   Responsible for scanning the top-level directories of the NAS.
    *   Creates "Shard Tasks" (one per top-level directory) in the database.
    *   Lightweight, usually runs as a single instance (Job).

2.  **Worker Node**:
    *   Horizontal scalable units.
    *   **Shard Processor**: Claims a shard task, recursively scans files, and creates "File Tasks" in the database.
    *   **Upload Workers**: A pool of goroutines that claim file tasks and perform the actual upload to KS3 using the AWS SDK v2 (with multipart support).

3.  **Database**:
    *   MySQL (8.0+) or PostgreSQL is required to support `SKIP LOCKED` for high-concurrency task queueing.

## Prerequisites

*   Go 1.24+
*   MySQL 8.0+
*   Access to NAS storage
*   KS3 Credentials (Access Key, Secret Key)

## Database Setup

Run the schema script in your MySQL database:

```sql
source scripts/schema.sql;
```

## Configuration

The application can be configured via `config.yaml` in the working directory or `/etc/migration_tool/`, or via Environment Variables.

### Example `config.yaml`

```yaml
db_connection_string: "user:password@tcp(127.0.0.1:3306)/migration_db?parseTime=true"
nas_root: "/mnt/data"
ks3_region: "BEIJING"
ks3_endpoint: "ks3-cn-beijing.ksyuncs.com"
ks3_bucket: "my-target-bucket"
worker_pool_size: 50
db_batch_size: 100
retry_limit: 3
node_id: "worker-01" # Optional, defaults to hostname
```

### Environment Variables

*   `DB_CONNECTION_STRING`
*   `NAS_ROOT`
*   `KS3_REGION`
*   `KS3_ENDPOINT`
*   `KS3_BUCKET`
*   `AWS_ACCESS_KEY_ID` (Standard AWS Env Var)
*   `AWS_SECRET_ACCESS_KEY` (Standard AWS Env Var)

## Building

```bash
cd migration_tool
go build -o migration cmd/migration/main.go
```

## Deployment

### 1. Run Controller (One-time)

Initialize the tasks by running the controller. It will scan `nas_root` and populate `shard_tasks`.

```bash
./migration -mode controller
```

### 2. Run Workers

Run as many workers as needed. They will automatically coordinate via the database.

```bash
./migration -mode worker
```

## Monitoring

The system updates task statuses in the database. You can monitor progress with SQL queries or connect Grafana to the MySQL database.

Key metrics to watch:
*   Count of `file_tasks` where `status='completed'` vs `status='pending'`.
*   Count of `shard_tasks` completed.
*   Failures in `file_tasks`.

## Resilience

*   **Crash Recovery**: If a worker crashes, its claimed shard task remains "processing". You may need a cleanup script (or manual intervention) to reset stuck tasks to "pending" if a node is permanently lost. File tasks are atomic; if an upload fails, it remains pending or is retried.
*   **Retries**: File uploads are retried up to `retry_limit`.
