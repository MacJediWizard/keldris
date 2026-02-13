# Production Security Recommendations

This guide covers security hardening for production Keldris deployments.

## Session Configuration

Keldris supports two session timeout mechanisms:

- **SESSION_MAX_AGE** - Absolute session lifetime. The session cookie expires after this many seconds regardless of activity.
- **SESSION_IDLE_TIMEOUT** - Inactivity timeout. The session expires if no requests are made within this window. Set to `0` to disable.

### Development Defaults

| Variable | Default | Description |
|----------|---------|-------------|
| `SESSION_MAX_AGE` | `86400` (24h) | Generous for local development |
| `SESSION_IDLE_TIMEOUT` | `1800` (30m) | Reasonable idle cutoff |

### Production Recommendations

```bash
# Shorter absolute lifetime limits exposure from stolen sessions
SESSION_MAX_AGE=3600       # 1 hour

# Aggressive idle timeout for sensitive environments
SESSION_IDLE_TIMEOUT=900   # 15 minutes
```

For environments handling sensitive backup data or regulated workloads, consider:

```bash
SESSION_MAX_AGE=1800       # 30 minutes
SESSION_IDLE_TIMEOUT=600   # 10 minutes
```

### How It Works

1. When a user logs in via OIDC, the session cookie is created with `MaxAge` set to `SESSION_MAX_AGE`.
2. Each authenticated request updates the `last_activity` timestamp in the session.
3. On subsequent requests, if `last_activity` is older than `SESSION_IDLE_TIMEOUT`, the session is treated as expired and the user must re-authenticate.
4. The cookie's `MaxAge` provides a hard upper bound - even with continuous activity, the session expires after `SESSION_MAX_AGE` seconds.

## Session Cookie Security

These are enforced by default and cannot be weakened:

- **HttpOnly** - Session cookies are never accessible to JavaScript
- **SameSite=Lax** - Prevents CSRF in most scenarios
- **Secure** - Set to `true` in production (requires HTTPS)

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
# Always set to production
ENV=production

# Use a strong random secret (minimum 32 bytes)
# Generate with: openssl rand -base64 48
SESSION_SECRET=<random-value>

# Restrict CORS to your actual frontend domain
CORS_ORIGINS=https://backups.example.com

# Generate a strong encryption key for data at rest
# Generate with: openssl rand -hex 32
ENCRYPTION_KEY=<random-value>
```
