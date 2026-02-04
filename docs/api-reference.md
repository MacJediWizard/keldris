# API Reference

Keldris provides a REST API for programmatic access. All API endpoints require authentication.

## Authentication

### Session Authentication

The web interface uses session-based authentication via OIDC. Sessions are stored in HTTP-only cookies.

### API Key Authentication

For programmatic access, use API keys. Include the key in the `Authorization` header:

```
Authorization: Bearer your-api-key
```

Generate API keys from the web interface under **Settings > API Keys**.

## Base URL

All API endpoints are prefixed with `/api/v1`.

```
https://keldris.example.com/api/v1
```

## Common Response Formats

### Success Response

```json
{
  "data": { ... },
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 100
  }
}
```

### Error Response

```json
{
  "error": "error message",
  "code": "ERROR_CODE",
  "details": { ... }
}
```

## Endpoints

### Health

#### GET /health

Check server health (no authentication required).

**Response:**
```json
{
  "status": "healthy",
  "checks": {
    "database": {
      "status": "healthy",
      "duration": "1.234ms"
    },
    "oidc": {
      "status": "healthy",
      "duration": "45.678ms"
    }
  }
}
```

### Agents

#### GET /api/v1/agents

List all agents in the current organization.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `page` | int | Page number (default: 1) |
| `per_page` | int | Items per page (default: 20, max: 100) |
| `status` | string | Filter by status: online, offline, error |
| `search` | string | Search by name or hostname |

**Response:**
```json
{
  "data": [
    {
      "id": "uuid",
      "name": "web-server-01",
      "hostname": "web-01.example.com",
      "status": "online",
      "last_seen": "2024-01-15T10:30:00Z",
      "version": "1.0.0",
      "os": "linux",
      "arch": "amd64"
    }
  ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 5
  }
}
```

#### GET /api/v1/agents/:id

Get a specific agent.

#### DELETE /api/v1/agents/:id

Delete an agent.

### Repositories

#### GET /api/v1/repositories

List all repositories.

**Response:**
```json
{
  "data": [
    {
      "id": "uuid",
      "name": "production-backups",
      "type": "s3",
      "status": "connected",
      "stats": {
        "total_size": 1073741824,
        "snapshot_count": 42
      }
    }
  ]
}
```

#### POST /api/v1/repositories

Create a new repository.

**Request Body:**
```json
{
  "name": "my-repository",
  "type": "s3",
  "config": {
    "endpoint": "s3.amazonaws.com",
    "bucket": "my-bucket",
    "region": "us-east-1",
    "access_key_id": "...",
    "secret_access_key": "..."
  },
  "password": "repository-encryption-password"
}
```

#### GET /api/v1/repositories/:id

Get repository details.

#### PUT /api/v1/repositories/:id

Update a repository.

#### DELETE /api/v1/repositories/:id

Delete a repository.

#### POST /api/v1/repositories/:id/check

Run a repository integrity check.

### Schedules

#### GET /api/v1/schedules

List all backup schedules.

**Response:**
```json
{
  "data": [
    {
      "id": "uuid",
      "name": "Daily Production Backup",
      "cron": "0 0 2 * * *",
      "agent_id": "uuid",
      "repository_id": "uuid",
      "paths": ["/data", "/etc"],
      "enabled": true,
      "next_run": "2024-01-16T02:00:00Z"
    }
  ]
}
```

#### POST /api/v1/schedules

Create a new schedule.

**Request Body:**
```json
{
  "name": "Daily Backup",
  "cron": "0 0 2 * * *",
  "agent_id": "uuid",
  "repository_id": "uuid",
  "paths": ["/data"],
  "exclude_patterns": ["*.log", "*.tmp"],
  "retention": {
    "keep_daily": 7,
    "keep_weekly": 4,
    "keep_monthly": 6
  },
  "enabled": true
}
```

#### POST /api/v1/schedules/:id/run

Trigger an immediate backup.

### Backups

#### GET /api/v1/backups

List all backups.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `agent_id` | uuid | Filter by agent |
| `repository_id` | uuid | Filter by repository |
| `schedule_id` | uuid | Filter by schedule |
| `status` | string | Filter by status |
| `from` | datetime | Start date |
| `to` | datetime | End date |

**Response:**
```json
{
  "data": [
    {
      "id": "uuid",
      "snapshot_id": "abc123",
      "status": "success",
      "started_at": "2024-01-15T02:00:00Z",
      "finished_at": "2024-01-15T02:15:00Z",
      "size_added": 104857600,
      "files_new": 150,
      "files_changed": 42,
      "files_unmodified": 10000
    }
  ]
}
```

#### GET /api/v1/backups/:id

Get backup details.

#### GET /api/v1/backups/:id/files

List files in a backup snapshot.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `path` | string | Directory path to list |

### Snapshots

#### GET /api/v1/snapshots

List all snapshots.

#### GET /api/v1/snapshots/:id

Get snapshot details.

#### GET /api/v1/snapshots/:id/files

Browse files in a snapshot.

#### POST /api/v1/snapshots/:id/restore

Initiate a restore operation.

**Request Body:**
```json
{
  "target_agent_id": "uuid",
  "target_path": "/restore",
  "paths": ["/data/important"],
  "overwrite": false
}
```

#### GET /api/v1/snapshots/compare

Compare two snapshots.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `snapshot_a` | string | First snapshot ID |
| `snapshot_b` | string | Second snapshot ID |

### Alerts

#### GET /api/v1/alerts

List alerts.

#### PUT /api/v1/alerts/:id/acknowledge

Acknowledge an alert.

#### PUT /api/v1/alerts/:id/resolve

Resolve an alert.

### Statistics

#### GET /api/v1/stats

Get storage statistics.

**Response:**
```json
{
  "total_size": 10737418240,
  "total_snapshots": 150,
  "total_agents": 10,
  "total_repositories": 3,
  "backup_success_rate": 0.98,
  "storage_by_repository": [
    {
      "repository_id": "uuid",
      "name": "production",
      "size": 5368709120
    }
  ]
}
```

#### GET /api/v1/stats/timeline

Get backup timeline data.

### Audit Logs

#### GET /api/v1/audit-logs

List audit log entries.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `user_id` | uuid | Filter by user |
| `action` | string | Filter by action type |
| `from` | datetime | Start date |
| `to` | datetime | End date |

## Webhooks

Keldris can send webhooks for various events.

### Event Types

| Event | Description |
|-------|-------------|
| `backup.started` | Backup job started |
| `backup.success` | Backup completed successfully |
| `backup.failure` | Backup failed |
| `agent.online` | Agent came online |
| `agent.offline` | Agent went offline |
| `alert.created` | New alert created |

### Webhook Payload

```json
{
  "event": "backup.success",
  "timestamp": "2024-01-15T02:15:00Z",
  "data": {
    "backup_id": "uuid",
    "agent_name": "web-server-01",
    "repository_name": "production",
    "duration_seconds": 900,
    "size_added": 104857600
  }
}
```

## Rate Limiting

API requests are rate limited to prevent abuse:

- Default: 100 requests per minute
- Headers indicate limit status:
  - `X-RateLimit-Limit`: Maximum requests per period
  - `X-RateLimit-Remaining`: Requests remaining
  - `X-RateLimit-Reset`: Seconds until limit resets

## Pagination

List endpoints support pagination:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `page` | Page number | 1 |
| `per_page` | Items per page | 20 |

Response includes metadata:

```json
{
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

## Interactive Documentation

Swagger/OpenAPI documentation is available at:

```
https://keldris.example.com/api/docs
```
