# Security Checklist

Pre-deployment security checklist for Keldris. Complete all items before exposing your instance to production traffic.

## Required Environment Variables

These must be set before starting Keldris in production. The server will refuse to start or operate insecurely without them.

| Variable | How to Generate | Notes |
|----------|----------------|-------|
| `ENV` | Set to `production` | Enables Secure cookies, strict CORS, HSTS |
| `SESSION_SECRET` | `openssl rand -base64 48` | Minimum 32 bytes, used for session cookie encryption |
| `ENCRYPTION_KEY` | `openssl rand -hex 32` | AES-256 key for encrypting credentials at rest |
| `DATABASE_URL` | — | PostgreSQL 15+ connection string with TLS (`sslmode=require`) |
| `OIDC_ISSUER` | — | Your OIDC provider's issuer URL |
| `OIDC_CLIENT_ID` | — | OAuth2 client ID |
| `OIDC_CLIENT_SECRET` | — | OAuth2 client secret |
| `OIDC_REDIRECT_URL` | — | Must use HTTPS in production |
| `CORS_ORIGINS` | — | Exact frontend origin (e.g., `https://backups.example.com`) |

### Optional but Recommended

| Variable | Default | Notes |
|----------|---------|-------|
| `SESSION_MAX_AGE` | `86400` (24h) | Reduce to `3600` (1h) for sensitive environments |
| `SESSION_IDLE_TIMEOUT` | `1800` (30m) | Reduce to `900` (15m) for sensitive environments |
| `REDIS_URL` | — | Required for distributed rate limiting across multiple instances |

## Pre-Deployment Checklist

### Secrets and Credentials

- [ ] `SESSION_SECRET` is randomly generated and unique to this deployment
- [ ] `ENCRYPTION_KEY` is randomly generated and unique to this deployment
- [ ] `OIDC_CLIENT_SECRET` is stored securely (not in version control)
- [ ] No secrets are logged or exposed in error messages
- [ ] `.env` file is not committed to version control

### Network and TLS

- [ ] Keldris is behind a reverse proxy (nginx, Caddy, or Traefik)
- [ ] TLS is terminated at the reverse proxy with a valid certificate
- [ ] The application port (default 8080) is not directly exposed to the internet
- [ ] HSTS is enabled at the proxy level
- [ ] HTTP-to-HTTPS redirect is configured

### Authentication

- [ ] OIDC provider is configured and tested
- [ ] `OIDC_REDIRECT_URL` uses HTTPS
- [ ] Session timeouts are configured for your security requirements
- [ ] Cookie `Secure` flag is active (`ENV=production`)

### CORS and CSRF

- [ ] `CORS_ORIGINS` is set to your exact frontend domain
- [ ] `CORS_ORIGINS` does not include wildcards or localhost in production
- [ ] SameSite=Lax cookies are active (automatic in production)

### Database

- [ ] PostgreSQL connection uses TLS (`sslmode=require` or `sslmode=verify-full`)
- [ ] Database credentials are not shared with other applications
- [ ] Database user has minimum required privileges
- [ ] Backups of the PostgreSQL database are configured

### Request Limits

- [ ] Reverse proxy enforces request body size limits (recommended: 10 MB)
- [ ] Rate limiting is active (default: 100 requests/minute per IP)
- [ ] `ReadHeaderTimeout` is set (default: 10 seconds)

### Webhook Security

- [ ] Outbound network access from Keldris is restricted to necessary destinations
- [ ] Cloud metadata endpoints (169.254.169.254) are blocked at the network level
- [ ] Webhook recipients verify `X-Keldris-Signature` HMAC signatures

## Recommended Nginx Configuration

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

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options DENY always;
    add_header X-Content-Type-Options nosniff always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    # Request limits
    client_max_body_size 10m;
    client_body_timeout 30s;
    client_header_timeout 10s;

    # Rate limiting zone (define in http block)
    # limit_req_zone $binary_remote_addr zone=keldris:10m rate=10r/s;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Timeouts
        proxy_connect_timeout 10s;
        proxy_read_timeout 60s;
        proxy_send_timeout 60s;

        # Rate limiting (uncomment after defining zone)
        # limit_req zone=keldris burst=20 nodelay;
    }
}
```

## Recommended Caddy Configuration

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

Caddy automatically provisions TLS certificates via Let's Encrypt and redirects HTTP to HTTPS.

## Monitoring Recommendations

### Application Health

- Monitor the `/api/v1/health` endpoint for uptime checks
- Set up alerts for HTTP 5xx error rate spikes
- Track response latency at the P95 and P99 levels

### Prometheus Metrics

Keldris exposes a Prometheus metrics endpoint. Scrape it to monitor:

- Request rate and latency by endpoint
- Error rates by status code
- Active sessions count

### Security Monitoring

- **Failed authentication attempts** - Monitor OIDC callback errors and 401 responses
- **Rate limit hits** - High 429 response rates may indicate brute-force attempts
- **Audit log review** - Regularly review the audit log for unexpected admin actions
- **Session anomalies** - Watch for sessions from unusual IPs or user agents

### Log Aggregation

Keldris outputs structured JSON logs. Ship them to a centralized logging system (e.g., Loki, ELK, Datadog) for:

- Security incident investigation
- Compliance audit trails
- Performance troubleshooting

### Recommended Alerts

| Metric | Threshold | Action |
|--------|-----------|--------|
| HTTP 5xx rate | > 1% of requests | Investigate server errors |
| 401/403 rate | > 10/minute from single IP | Possible brute force |
| 429 rate | Sustained elevation | Review rate limit configuration |
| Health check failure | 2+ consecutive failures | Restart service, check dependencies |
| Certificate expiry | < 14 days | Renew TLS certificate |
