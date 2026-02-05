# Configuration Reference

This document covers all configuration options for the Keldris server and agent.

## Server Configuration

The Keldris server is configured through environment variables.

### Required Settings

| Variable | Description | Example |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://user:pass@host:5432/keldris?sslmode=disable` |
| `OIDC_ISSUER_URL` | OIDC provider URL | `https://auth.example.com` |
| `OIDC_CLIENT_ID` | OIDC client ID | `keldris` |
| `OIDC_CLIENT_SECRET` | OIDC client secret | `your-secret` |
| `SESSION_SECRET` | Secret for session encryption (min 32 chars) | `random-secret-here` |

### Optional Settings

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_URL` | Public URL of the server | `http://localhost:8080` |
| `PORT` | HTTP port to listen on | `8080` |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | `info` |
| `LOG_FORMAT` | Log format (json, text) | `json` |

### Database Settings

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_MAX_OPEN_CONNS` | Maximum open database connections | `25` |
| `DB_MAX_IDLE_CONNS` | Maximum idle database connections | `5` |
| `DB_CONN_MAX_LIFETIME` | Connection maximum lifetime | `5m` |

### OIDC Settings

| Variable | Description | Default |
|----------|-------------|---------|
| `OIDC_REDIRECT_URL` | OAuth callback URL | `{SERVER_URL}/auth/callback` |
| `OIDC_SCOPES` | OAuth scopes to request | `openid,profile,email` |
| `OIDC_GROUPS_CLAIM` | JWT claim containing groups | `groups` |

### Session Settings

| Variable | Description | Default |
|----------|-------------|---------|
| `SESSION_COOKIE_NAME` | Session cookie name | `keldris_session` |
| `SESSION_COOKIE_SECURE` | Use secure cookies (HTTPS only) | `true` |
| `SESSION_MAX_AGE` | Session duration | `24h` |

### Rate Limiting

| Variable | Description | Default |
|----------|-------------|---------|
| `RATE_LIMIT_REQUESTS` | Requests per period | `100` |
| `RATE_LIMIT_PERIOD` | Rate limit period | `1m` |

### Email Settings

| Variable | Description | Default |
|----------|-------------|---------|
| `SMTP_HOST` | SMTP server hostname | - |
| `SMTP_PORT` | SMTP server port | `587` |
| `SMTP_USERNAME` | SMTP username | - |
| `SMTP_PASSWORD` | SMTP password | - |
| `SMTP_FROM` | From email address | - |
| `SMTP_TLS` | Use TLS | `true` |

### Encryption Settings

| Variable | Description | Default |
|----------|-------------|---------|
| `ENCRYPTION_KEY` | Master encryption key (32 bytes, base64) | Auto-generated |

## Agent Configuration

The agent is configured via a YAML file and environment variables.

### Configuration File

Default locations:
- Linux: `/etc/keldris/agent.yaml`
- macOS: `~/.config/keldris/agent.yaml`
- Windows: `C:\ProgramData\Keldris\agent.yaml`

```yaml
# Keldris Agent Configuration

# Server connection
server:
  url: https://keldris.example.com
  api_key: your-api-key
  tls_skip_verify: false  # Only for testing

# Agent identification
agent:
  name: my-server
  tags:
    - production
    - database

# Logging
logging:
  level: info  # debug, info, warn, error
  file: /var/log/keldris/agent.log

# Health reporting
health:
  interval: 60s  # How often to report health
  timeout: 30s   # Health check timeout

# Backup defaults
backup:
  # Default exclude patterns
  exclude:
    - "*.tmp"
    - "*.log"
    - ".cache"
    - "node_modules"

  # Compression level (auto, off, max)
  compression: auto

  # Read concurrency
  read_concurrency: 2
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `KELDRIS_CONFIG_DIR` | Configuration directory | Platform-specific |
| `KELDRIS_SERVER_URL` | Server URL (overrides config) | - |
| `KELDRIS_LOG_LEVEL` | Log level | `info` |

### Command Line Flags

```bash
# Run as daemon
keldris-agent daemon

# Register with server
keldris-agent register --server https://keldris.example.com

# Check connection
keldris-agent status

# Run backup manually
keldris-agent backup --schedule schedule-id

# Show version
keldris-agent version
```

## Storage Backend Configuration

### Local Storage

```yaml
type: local
path: /backups/restic-repo
```

### S3 / S3-Compatible

```yaml
type: s3
endpoint: s3.amazonaws.com  # Or custom endpoint
bucket: my-backup-bucket
region: us-east-1
access_key_id: AKIAIOSFODNN7EXAMPLE
secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

### Backblaze B2

```yaml
type: b2
account_id: your-account-id
application_key: your-application-key
bucket: my-backup-bucket
```

### SFTP

```yaml
type: sftp
host: backup.example.com
port: 22
user: backup
path: /backups/restic-repo
# Use key authentication
key_file: /path/to/private/key
# Or password
password: your-password
```

### REST Server

```yaml
type: rest
url: https://rest-server.example.com:8000
username: user
password: password
```

## Retention Policies

Configure how long to keep backups:

```yaml
retention:
  keep_last: 7        # Keep last N snapshots
  keep_hourly: 24     # Keep N hourly snapshots
  keep_daily: 7       # Keep N daily snapshots
  keep_weekly: 4      # Keep N weekly snapshots
  keep_monthly: 12    # Keep N monthly snapshots
  keep_yearly: 2      # Keep N yearly snapshots
  keep_within: 7d     # Keep all within duration
```

## Schedule Configuration

### Cron Expressions

Keldris uses standard cron syntax with seconds:

```
┌───────────── second (0-59)
│ ┌───────────── minute (0-59)
│ │ ┌───────────── hour (0-23)
│ │ │ ┌───────────── day of month (1-31)
│ │ │ │ ┌───────────── month (1-12)
│ │ │ │ │ ┌───────────── day of week (0-6, Sun=0)
│ │ │ │ │ │
* * * * * *
```

Examples:
- `0 0 2 * * *` - Daily at 2:00 AM
- `0 0 */6 * * *` - Every 6 hours
- `0 30 1 * * 0` - Sunday at 1:30 AM
- `0 0 3 1 * *` - First of each month at 3:00 AM

## Notification Configuration

### Email Notifications

Configure per organization in the web UI under **Settings > Notifications**.

### Webhook Notifications

```yaml
webhooks:
  - url: https://hooks.example.com/backup-status
    events:
      - backup.success
      - backup.failure
      - agent.offline
    headers:
      Authorization: Bearer your-token
```

## OIDC Provider Examples

### Authentik

```env
OIDC_ISSUER_URL=https://auth.example.com/application/o/keldris/
OIDC_CLIENT_ID=keldris-client-id
OIDC_CLIENT_SECRET=keldris-client-secret
OIDC_SCOPES=openid,profile,email,groups
OIDC_GROUPS_CLAIM=groups
```

### Keycloak

```env
OIDC_ISSUER_URL=https://keycloak.example.com/realms/your-realm
OIDC_CLIENT_ID=keldris
OIDC_CLIENT_SECRET=your-secret
OIDC_SCOPES=openid,profile,email
OIDC_GROUPS_CLAIM=groups
```

### Google Workspace

```env
OIDC_ISSUER_URL=https://accounts.google.com
OIDC_CLIENT_ID=your-client-id.apps.googleusercontent.com
OIDC_CLIENT_SECRET=your-secret
OIDC_SCOPES=openid,profile,email
```
