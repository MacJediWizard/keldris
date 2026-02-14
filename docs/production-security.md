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

The server also sets `ReadHeaderTimeout: 10s` on the HTTP server to protect against slow-header (Slowloris) attacks.

## Webhook URL Validation

Webhook notifications send HTTP POST requests to user-configured URLs. To prevent Server-Side Request Forgery (SSRF), deploy Keldris behind a reverse proxy or firewall that restricts outbound requests to internal networks.

### Risks

- A malicious or misconfigured webhook URL pointing to internal services (e.g., `http://169.254.169.254/` for cloud metadata, or `http://localhost:5432/`) could allow internal network probing.

### Mitigations

1. **Network-level controls** - Run Keldris in a network segment with outbound firewall rules that block access to RFC 1918 ranges (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`) and link-local addresses (`169.254.0.0/16`).
2. **DNS rebinding protection** - Use a DNS resolver that refuses to resolve private IPs for public domains.
3. **Webhook allow-listing** - If feasible, maintain an allow-list of permitted webhook destination domains.

### Webhook Security Features

Keldris signs all webhook payloads with HMAC-SHA256. The signature is sent in the `X-Keldris-Signature` header. Verify this signature on the receiving end to ensure authenticity.

```
X-Keldris-Signature: sha256=<hex-encoded-hmac>
```

## Go Runtime Requirements

Keldris requires **Go 1.25.7 or later** for both building and running.

### Why Go 1.25.7+

- TLS 1.3 support with modern cipher suites
- Improved cryptographic library hardening
- Security fixes in `net/http` and `crypto` packages
- Required by dependency tree (see `go.mod`)

### Docker Base Image

The official Docker image uses:

```dockerfile
FROM golang:1.25.7-alpine AS go-builder
FROM alpine:3.21
```

Alpine 3.21 is the current stable release. Keep both the Go toolchain and Alpine base image updated to receive security patches.

### Verifying Your Go Version

```bash
go version
# Should output: go version go1.25.7 ...
```

## Reverse Proxy Recommendations

Keldris should always run behind a reverse proxy (nginx, Caddy, Traefik, etc.) in production. The application server binds to a local port and should not be exposed directly to the internet.

### Why a Reverse Proxy

- **TLS termination** - Let the proxy handle HTTPS certificates (e.g., via Let's Encrypt).
- **Request filtering** - Enforce body size limits, rate limiting, and header sanitization.
- **HSTS** - The proxy can add `Strict-Transport-Security` headers consistently.
- **Trusted proxies** - Keldris uses Gin's `c.ClientIP()` which reads `X-Forwarded-For` and `X-Real-IP`. Configure your proxy to set these headers correctly.

### Nginx Example

```nginx
server {
    listen 443 ssl http2;
    server_name backups.example.com;

    ssl_certificate     /etc/letsencrypt/live/backups.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/backups.example.com/privkey.pem;

    # Security headers (Keldris also sets these, but belt-and-suspenders)
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options DENY always;
    add_header X-Content-Type-Options nosniff always;

    # Body size limit
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

### Caddy Example

```caddyfile
backups.example.com {
    reverse_proxy localhost:8080

    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
        X-Frame-Options DENY
        X-Content-Type-Options nosniff
    }

    request_body {
        max_size 10MB
    }
}
```

## CSRF Protection

Keldris uses multiple layers to prevent Cross-Site Request Forgery:

### SameSite Cookies

Session cookies are set with `SameSite=Lax`, which prevents the browser from sending them on cross-origin POST, PUT, PATCH, and DELETE requests. This is the primary CSRF defense.

### CORS Policy

The CORS middleware restricts which origins can make credentialed requests:

- **Production** - `CORS_ORIGINS` must be explicitly set or the server refuses to start.
- **Development** - All origins are allowed (with a warning).

Only origins listed in `CORS_ORIGINS` can include cookies (`credentials: include`) in cross-origin requests.

### OIDC State Parameter

The OIDC login flow uses a cryptographically random 256-bit state parameter to prevent login CSRF attacks. The state is validated on the callback to ensure the login was initiated by the same browser session.

### Security Headers

Keldris sets `X-Frame-Options: DENY` to prevent clickjacking and applies a strict Content-Security-Policy on API routes (`default-src 'none'; frame-ancestors 'none'`).

### Recommendations

For maximum protection in sensitive environments:

```bash
# Restrict CORS to your exact frontend domain
CORS_ORIGINS=https://backups.example.com

# Use HTTPS so Secure cookie flag is active
ENV=production
```
