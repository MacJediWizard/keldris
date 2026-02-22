# Production Security Guide

This guide covers security hardening for production Keldris deployments. Complete the [Security Checklist](security-checklist.md) before exposing your instance to production traffic.

## TLS / HTTPS

Keldris does not terminate TLS itself. Use a reverse proxy (nginx, Caddy, Traefik) for TLS termination.

### Reverse Proxy Setup

**Nginx with Let's Encrypt:**

```nginx
server {
    listen 80;
    server_name backups.example.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name backups.example.com;

    ssl_certificate     /etc/letsencrypt/live/backups.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/backups.example.com/privkey.pem;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # HSTS - browsers will refuse HTTP for 1 year
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options DENY always;
    add_header X-Content-Type-Options nosniff always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    client_max_body_size 10m;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

**Caddy (automatic TLS and HSTS):**

```caddyfile
backups.example.com {
    reverse_proxy localhost:8080

    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
        X-Frame-Options DENY
        X-Content-Type-Options nosniff
        Referrer-Policy "strict-origin-when-cross-origin"
    }

    request_body {
        max_size 10MB
    }
}
```

Caddy provisions TLS certificates via Let's Encrypt automatically and redirects HTTP to HTTPS by default.

### HSTS Behavior

When `ENV=production`, Keldris sets security headers on API responses. The reverse proxy should also set `Strict-Transport-Security` to cover static assets and frontend routes. Once HSTS is active, browsers refuse plaintext HTTP connections for the configured `max-age` duration.

### Why a Reverse Proxy

- **TLS termination** - Handles certificate management and renewal
- **Request filtering** - Body size limits, header sanitization, rate limiting
- **Slowloris protection** - Keldris sets `ReadHeaderTimeout: 10s`, but the proxy adds another layer
- **Trusted proxies** - Keldris reads `X-Forwarded-For` and `X-Real-IP` for client IP detection

## Environment Variables

### Secrets Management

Never commit `.env` files to version control. Use a secrets manager for production deployments.

```bash
# Generate secrets
openssl rand -base64 48   # SESSION_SECRET (minimum 32 bytes)
openssl rand -hex 32      # ENCRYPTION_KEY (AES-256, exactly 32 bytes)
```

**Required production variables:**

```bash
ENV=production
SESSION_SECRET=<generated-value>
ENCRYPTION_KEY=<generated-value>
DATABASE_URL=postgresql://keldris:PASSWORD@db:5432/keldris?sslmode=require
OIDC_ISSUER=https://auth.example.com
OIDC_CLIENT_ID=keldris
OIDC_CLIENT_SECRET=<from-oidc-provider>
OIDC_REDIRECT_URL=https://backups.example.com/auth/callback
CORS_ORIGINS=https://backups.example.com
```

### Secrets Managers

Use platform-specific secrets management instead of `.env` files:

```bash
# Docker Swarm
docker secret create keldris_session_secret session_secret.txt
docker secret create keldris_encryption_key encryption_key.txt

# Kubernetes
kubectl create secret generic keldris-secrets \
  --from-literal=SESSION_SECRET="$(openssl rand -base64 48)" \
  --from-literal=ENCRYPTION_KEY="$(openssl rand -hex 32)" \
  --from-literal=OIDC_CLIENT_SECRET="your-client-secret"

# HashiCorp Vault
vault kv put secret/keldris \
  session_secret="$(openssl rand -base64 48)" \
  encryption_key="$(openssl rand -hex 32)"
```

### Key Rotation

Rotate `SESSION_SECRET` and `ENCRYPTION_KEY` periodically:

1. **SESSION_SECRET** - Rotating this invalidates all active sessions. Users must re-authenticate. Schedule rotation during maintenance windows.
2. **ENCRYPTION_KEY** - Rotating this requires re-encrypting all stored credentials. Plan a migration before changing this value.
3. **OIDC_CLIENT_SECRET** - Rotate through your OIDC provider. Update both the provider and Keldris simultaneously.

## Database

### Connection Security

Always use TLS for PostgreSQL connections in production:

```bash
# Require TLS (verify server certificate exists)
DATABASE_URL=postgresql://keldris:PASSWORD@db.internal:5432/keldris?sslmode=require

# Verify server certificate against CA (strongest)
DATABASE_URL=postgresql://keldris:PASSWORD@db.internal:5432/keldris?sslmode=verify-full&sslrootcert=/path/to/ca.crt
```

### Strong Passwords

Generate a strong database password:

```bash
openssl rand -base64 32
```

### Least Privilege

Create a dedicated database user with only the permissions Keldris needs:

```sql
-- Create the keldris database and user
CREATE DATABASE keldris;
CREATE USER keldris WITH PASSWORD '<strong-password>';

-- Grant only necessary privileges
GRANT CONNECT ON DATABASE keldris TO keldris;
GRANT USAGE ON SCHEMA public TO keldris;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO keldris;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO keldris;

-- Allow creating tables for migrations
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO keldris;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO keldris;
```

### Network Restrictions

- Run PostgreSQL on a private network inaccessible from the internet
- Use firewall rules to allow connections only from the Keldris server
- Bind PostgreSQL to a private IP, not `0.0.0.0`

```bash
# postgresql.conf
listen_addresses = '10.0.1.5'

# pg_hba.conf - only allow keldris user from the app server
hostssl keldris keldris 10.0.1.10/32 scram-sha-256
```

## OIDC

### Provider Configuration

- **Always use HTTPS** for the OIDC issuer URL. Keldris validates the issuer's TLS certificate.
- Store `OIDC_CLIENT_SECRET` in a secrets manager, never in source control.
- Set `OIDC_REDIRECT_URL` to your HTTPS frontend URL.

```bash
OIDC_ISSUER=https://auth.example.com/realms/keldris
OIDC_CLIENT_ID=keldris-app
OIDC_CLIENT_SECRET=<stored-in-secrets-manager>
OIDC_REDIRECT_URL=https://backups.example.com/auth/callback
```

### Token Lifetimes

Configure short token lifetimes in your OIDC provider:

| Token | Recommended Lifetime | Notes |
|-------|---------------------|-------|
| ID Token | 5-15 minutes | Used once during login |
| Access Token | 15-30 minutes | Short-lived for API calls |
| Refresh Token | 8-24 hours | Allows session extension |

### Session Timeouts

Keldris enforces its own session timeouts independent of OIDC tokens:

```bash
# Production recommendations
SESSION_MAX_AGE=3600       # 1 hour absolute lifetime
SESSION_IDLE_TIMEOUT=900   # 15 minutes inactivity timeout

# Sensitive environments
SESSION_MAX_AGE=1800       # 30 minutes
SESSION_IDLE_TIMEOUT=600   # 10 minutes
```

### OIDC Security Checklist

- Use a dedicated OIDC client for Keldris (do not share with other apps)
- Enable PKCE if your provider supports it
- Restrict allowed redirect URIs to your exact callback URL
- Disable implicit flow in the OIDC client configuration

## Network

### Firewall Rules

Restrict inbound and outbound traffic to only what Keldris needs:

```bash
# Example: UFW on Ubuntu
# Allow HTTPS from anywhere
ufw allow 443/tcp

# Allow SSH from admin network only
ufw allow from 10.0.0.0/24 to any port 22

# Keldris listens on 8080 - only accessible from reverse proxy (localhost)
ufw deny 8080/tcp

# Block cloud metadata endpoints (SSRF protection)
ufw deny out to 169.254.169.254
```

### Restrict Agent API Access

Agents communicate with the server over HTTPS. Restrict agent API endpoints to known agent networks:

```nginx
# Nginx: restrict agent endpoints to internal network
location /api/v1/agents {
    allow 10.0.0.0/8;
    allow 172.16.0.0/12;
    allow 192.168.0.0/16;
    deny all;

    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

### Private Networks for Database and Storage

```
Internet
  │
  ├── [Reverse Proxy] ── Public subnet (443/tcp)
  │       │
  │       └── [Keldris Server] ── Private subnet (8080/tcp)
  │               │
  │               ├── [PostgreSQL] ── Database subnet (5432/tcp)
  │               └── [S3/B2/SFTP] ── Storage network
  │
  └── [Agents] ── Agent network (HTTPS to reverse proxy)
```

- Database and storage backends should never be directly accessible from the internet
- Use VPC peering, VPNs, or private links for cloud storage connections
- DNS rebinding protection: use a resolver that refuses to resolve private IPs for public domains

## Docker

### Non-Root Containers

Keldris Docker images run as a non-root user (`keldris`, UID 1000) by default. This is configured in both `Dockerfile.server` and `Dockerfile.agent`:

```dockerfile
RUN adduser -D -u 1000 keldris
USER keldris
```

Do not override this with `--user root` in production.

### Keep Images Updated

```bash
# Pull latest base images and rebuild
docker pull golang:1.25.7-alpine
docker pull node:20-alpine
docker pull alpine:3.21

docker build -f docker/Dockerfile.server -t keldris-server .
docker build -f docker/Dockerfile.agent -t keldris-agent .
```

Subscribe to security advisories for Alpine Linux and rebuild when patches are released.

### Read-Only Filesystem

Run containers with a read-only root filesystem. Mount writable volumes only where needed:

```bash
docker run -d \
  --read-only \
  --tmpfs /tmp:rw,noexec,nosuid,size=64m \
  -v keldris-data:/app/data:rw \
  --name keldris-server \
  keldris-server
```

```yaml
# docker-compose.yml
services:
  keldris-server:
    image: keldris-server:latest
    read_only: true
    tmpfs:
      - /tmp:rw,noexec,nosuid,size=64m
    volumes:
      - keldris-data:/app/data:rw
    security_opt:
      - no-new-privileges:true
```

### Additional Docker Hardening

```yaml
# docker-compose.yml security options
services:
  keldris-server:
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    cap_add:
      - NET_BIND_SERVICE
```

## Rate Limiting

Keldris uses [ulule/limiter](https://github.com/ulule/limiter) for rate limiting. The default is **100 requests per minute per IP**.

### Configuration

```bash
# Default: in-memory store (single instance)
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_PERIOD=1m

# Higher limits for API-heavy workloads
RATE_LIMIT_REQUESTS=500
RATE_LIMIT_PERIOD=1m
```

### Redis for Distributed Rate Limiting

When running multiple Keldris instances behind a load balancer, use Redis to share rate limit state:

```bash
REDIS_URL=redis://redis.internal:6379/0
```

Without Redis, each instance tracks rate limits independently, which means the effective limit is multiplied by the number of instances.

### Adjusting for Load

| Deployment | Recommended Limit | Notes |
|-----------|-------------------|-------|
| Small team (< 10 users) | 100 req/min | Default is sufficient |
| Medium org (10-100 users) | 300 req/min | Increase for dashboard-heavy usage |
| API-heavy / CI integration | 500-1000 req/min | Monitor 429 rates and adjust |
| Behind CDN | Consider CDN-level limiting | Supplement with application limits |
The server also sets `ReadHeaderTimeout: 10s` on the HTTP server to protect against slow-header (Slowloris) attacks.

### Proxy-Level Rate Limiting

Add rate limiting at the reverse proxy for defense in depth:

```nginx
# Define in http block
limit_req_zone $binary_remote_addr zone=keldris:10m rate=10r/s;

# Apply in location block
location / {
    limit_req zone=keldris burst=20 nodelay;
    proxy_pass http://127.0.0.1:8080;
}
```

## Audit Logging

Audit logging is a **Pro+** feature. When enabled, Keldris logs all authenticated API actions with user identity, client IP, resource type, and result (success/failure/denied).

### Enabling Audit Logs

Audit logging is automatically active when a Pro or Enterprise license is installed. No additional configuration is required.

### What Gets Logged

| Field | Description |
|-------|-------------|
| `action` | `create`, `read`, `update`, `delete` |
| `resource_type` | `agent`, `repository`, `schedule`, `backup`, `user`, `organization` |
| `result` | `success`, `failure`, `denied` |
| `user_id` | Authenticated user who performed the action |
| `ip_address` | Client IP (from `X-Forwarded-For` or direct connection) |
| `user_agent` | Browser or API client identifier |

Health check endpoints (`/health`, `/api/v1/health`) and audit log reads are excluded to avoid noise.

### Querying Audit Logs

```bash
# List recent audit events
curl -s https://backups.example.com/api/v1/audit-logs \
  -H "Cookie: session=..." | jq '.[] | {action, resource_type, result, created_at}'

# Filter by action
curl -s "https://backups.example.com/api/v1/audit-logs?action=delete" \
  -H "Cookie: session=..."
```

### Exporting to SIEM

Keldris outputs structured JSON logs. Ship them to your SIEM for centralized analysis:

```bash
# Forward container logs to a SIEM via syslog
docker run -d \
  --log-driver=syslog \
  --log-opt syslog-address=tcp://siem.internal:514 \
  --log-opt tag=keldris-server \
  keldris-server

# Or use Fluentd/Fluent Bit
docker run -d \
  --log-driver=fluentd \
  --log-opt fluentd-address=fluentd.internal:24224 \
  --log-opt tag=keldris \
  keldris-server
```

### Retention Policies

Configure log retention based on your compliance requirements:

| Regulation | Minimum Retention |
|-----------|-------------------|
| SOC 2 | 1 year |
| HIPAA | 6 years |
| GDPR | As needed (minimize) |
| PCI DSS | 1 year (3 months immediately available) |
Keldris requires **Go 1.25.7 or later** for both building and running.

### Why Go 1.25.7+

Set retention at the SIEM/log aggregator level. Keldris stores audit logs in PostgreSQL; use scheduled cleanup or archival for database-level retention.

## Backup Encryption

Keldris encrypts sensitive data at rest using **AES-256-GCM** via the `ENCRYPTION_KEY` environment variable.

### What Gets Encrypted

- Restic repository passwords
- Storage backend credentials (S3 access keys, B2 keys, SFTP passwords)
- Notification channel configurations (Slack tokens, webhook secrets)

### How It Works

The `ENCRYPTION_KEY` is a 32-byte hex-encoded key used as the AES-256-GCM master key. A random 12-byte nonce is generated for each encryption operation. Ciphertext is stored as `nonce + ciphertext + GCM tag`.

```bash
# Generate a new encryption key
openssl rand -hex 32
```dockerfile
FROM golang:1.25.7-alpine AS go-builder
FROM alpine:3.21
```

### Backing Up the Master Key

The `ENCRYPTION_KEY` is critical. If lost, all encrypted credentials become unrecoverable.

1. **Store the key in a secrets manager** (Vault, AWS Secrets Manager, etc.)
2. **Keep an offline backup** in a secure, access-controlled location (e.g., printed and stored in a safe)
3. **Do not store the key alongside database backups** - if an attacker obtains both, encryption is defeated

### Key Escrow

For organizations with compliance requirements, implement key escrow:

```bash
# Split the key using Shamir's Secret Sharing (e.g., with `ssss-split`)
echo "<ENCRYPTION_KEY>" | ssss-split -t 3 -n 5
# Requires 3 of 5 shares to reconstruct the key
# Distribute shares to different custodians
go version
# Should output: go version go1.25.7 ...
```

Store shares with separate custodians who cannot independently access backup data. Document the recovery procedure and test it periodically.

### Restic Repository Encryption

In addition to Keldris's at-rest encryption of credentials, Restic itself encrypts all backup data with a per-repository password. This provides two layers of encryption:

1. **Restic layer** - Backup data is encrypted in the storage backend
2. **Keldris layer** - Repository passwords and credentials are encrypted in PostgreSQL

## Session Security

### Cookie Configuration

These are enforced by default in production (`ENV=production`):

- **HttpOnly** - Session cookies are never accessible to JavaScript
- **SameSite=Lax** - Prevents CSRF in most scenarios
- **Secure** - Requires HTTPS (active when `ENV=production`)

### Session Timeouts
## Request Body Size Limits

Keldris enforces a default **10 MB** request body size limit on all endpoints. Requests exceeding this limit receive a `413 Request Entity Too Large` response.

This protects against denial-of-service attacks via oversized payloads.

### Reverse Proxy Configuration

If you run Keldris behind a reverse proxy (nginx, Caddy, HAProxy, etc.), configure matching or lower body size limits at the proxy level as well. This rejects oversized requests before they reach the application server.

**nginx:**

```nginx
client_max_body_size 10m;
```

**Caddy:**

```
request_body {
    max_size 10MB
}
```

## Other Production Settings

```bash
# Production defaults
SESSION_MAX_AGE=86400       # 24 hours (absolute)
SESSION_IDLE_TIMEOUT=1800   # 30 minutes (inactivity)

# Recommended for production
SESSION_MAX_AGE=3600        # 1 hour
SESSION_IDLE_TIMEOUT=900    # 15 minutes
```

How it works:
1. On login, the session cookie is created with `MaxAge` set to `SESSION_MAX_AGE`
2. Each authenticated request updates the `last_activity` timestamp
3. If `last_activity` is older than `SESSION_IDLE_TIMEOUT`, the session expires
4. The cookie `MaxAge` is a hard upper bound regardless of activity

## CSRF Protection

Keldris uses multiple layers to prevent Cross-Site Request Forgery:

- **SameSite=Lax cookies** - Primary defense; prevents cross-origin POST/PUT/PATCH/DELETE
- **CORS policy** - Only origins in `CORS_ORIGINS` can make credentialed requests
- **OIDC state parameter** - 256-bit random state prevents login CSRF
- **X-Frame-Options: DENY** - Prevents clickjacking
- **Content-Security-Policy** - Strict `default-src 'none'; frame-ancestors 'none'` on API routes

## Request Body Size Limits

Keldris enforces a **10 MB** request body limit. Configure matching limits at the proxy:

```nginx
# nginx
client_max_body_size 10m;
```

```caddyfile
# Caddy
request_body {
    max_size 10MB
}
```

## Webhook Security

### SSRF Prevention

Webhook notifications send HTTP POST requests to user-configured URLs. Mitigate SSRF:

1. **Network controls** - Block outbound access to RFC 1918 ranges and link-local addresses (`169.254.0.0/16`)
2. **DNS rebinding** - Use a resolver that refuses to resolve private IPs for public domains
3. **Allow-listing** - Maintain a list of permitted webhook destination domains when feasible

### Payload Signatures

Keldris signs all webhook payloads with HMAC-SHA256:

```
X-Keldris-Signature: sha256=<hex-encoded-hmac>
```

Verify this signature on the receiving end to ensure authenticity.

## Go Runtime Requirements

Keldris requires **Go 1.25.7+** for both building and running. This version includes critical security patches for the standard library (`net/http`, `crypto`). See [Infrastructure Requirements](infrastructure-requirements.md) for upgrade instructions.

## Pre-Deployment Checklist

Complete the full [Security Checklist](security-checklist.md) before deploying to production. It covers:

- Required environment variables and how to generate them
- Network and TLS verification
- Authentication and session configuration
- CORS and CSRF validation
- Database security
- Request limits and rate limiting
- Webhook security
- Monitoring and alerting recommendations
