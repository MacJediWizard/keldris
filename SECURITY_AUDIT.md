# Security Audit Report — Keldris

**Date:** 2026-02-12
**Branch:** MacJediWizard/security-audit
**Go Version:** 1.24.0 (go.mod) / runtime go1.25.5
**Scope:** Full codebase (server, agent, frontend)

---

## 1. Dependency Vulnerability Scan (govulncheck)

Ran `govulncheck ./...` against the full module. **5 vulnerabilities found:**

| ID | Package | Severity | Current | Fixed In | Description |
|----|---------|----------|---------|----------|-------------|
| GO-2026-4341 | net/url | High | go1.25.5 | go1.25.6 | Memory exhaustion via large URL parsing |
| GO-2026-4340 | crypto/tls | Medium | go1.25.5 | go1.25.6 | TLS handshake issue |
| GO-2026-4337 | crypto/tls | Medium | go1.25.5 | go1.25.7 | TLS session resumption vulnerability |
| GO-2025-4233 | github.com/quic-go/quic-go | High | v0.54.0 | v0.57.0 | HTTP/3 QPACK denial-of-service |
| GO-2025-3540 | github.com/redis/go-redis/v9 | Medium | v9.0.4 | v9.6.3 | Out-of-order response handling |

**Remediation:**
- Upgrade Go runtime to >= 1.25.7 (fixes all 3 stdlib vulns)
- Upgrade `quic-go` to >= v0.57.0
- Upgrade `go-redis/v9` to >= v9.6.3

**Status:** go-redis upgraded to v9.17.3. quic-go upgraded to v0.59.0 (indirect dependency via testcontainers). Go runtime upgrade to >= 1.25.7 requires infrastructure change (tracked separately).

---

## 2. Sensitive Data in Logs

### Findings

| Severity | Location | Issue |
|----------|----------|-------|
| **HIGH** | `internal/notifications/service.go` (multiple lines) | Webhook URLs logged as `recipient` field at INFO level. Webhook URLs for Slack, Teams, and Discord contain embedded authentication tokens. |
| Low | `internal/api/middleware/logger.go:17` | Query strings logged — could contain sensitive params if misused. Currently no sensitive data passes via query strings. |

**What's done well:**
- Restic passwords passed via `RESTIC_PASSWORD` env var, never CLI args
- `restic.go` has `redactArgs()` for log safety
- API keys are SHA-256 hashed before storage, never logged in plaintext
- OIDC tokens/secrets not logged
- Session secrets not logged
- Encryption keys not logged
- Database credentials not logged

**Fix applied:** Webhook URLs are now redacted in `finalizeLog` — only the channel ID is logged, not the URL.

---

## 3. Input Validation Audit

### Findings

| Severity | Location | Issue |
|----------|----------|-------|
| **MEDIUM** | `handlers/schedules.go:283` | Cron expressions accepted without validation (TODO in code). Malformed expressions could cause scheduler errors at runtime. |
| **MEDIUM** | `handlers/search.go:64` | Search query `q` has no max length — could be used for oversized queries to the database. |
| **MEDIUM** | `handlers/agents.go:697-701` | HealthHistory `limit` param has no upper bound cap — user can request `limit=999999`. |
| **MEDIUM** | `handlers/repositories.go:581,609` | `ShouldBindJSON` error and `ParseBackend` error messages returned to client — could expose internal Go type info. |

**What's done well:**
- Agent hostname: `binding:"required,min=1,max=255"`
- Search limit: bounded 1-100
- UUID params: all parsed and validated
- Date params: RFC3339 format enforced
- API key format: `kld_` prefix required with length checks
- Session secret: minimum 32 bytes enforced
- Repository type: validated against whitelist
- Exclude pattern categories: validated against enum
- Notification event types: validated against known types
- All request bodies use `ShouldBindJSON` with `binding:` tags

**Fixes applied:** Search query capped at 1000 chars. HealthHistory limit capped at 1000. Cron expression validated with `robfig/cron` parser.

---

## 4. Path Traversal Audit

### Findings

| Severity | Location | Issue |
|----------|----------|-------|
| Low | `handlers/snapshots.go` — `CreateRestoreRequest.TargetPath` | User-provided target path for restores. Execution happens agent-side via Restic, not on the server. Agent runs as a system service with its own filesystem scope. |
| Low | `handlers/backup_scripts.go` | Accepts script content. Execution is agent-side only. |

**What's done well:**
- `backends/local.go`: `filepath.IsAbs()` enforced for local backend paths
- Restic handles path validation for backup/restore operations
- No server-side file operations use unsanitized user input
- Backup paths configured per-schedule, not per-request
- Exclude patterns are passed to Restic (not used for server-side file operations)

**Assessment:** No server-side path traversal vulnerabilities. Agent-side paths are inherently trusted (agent runs on the target machine with its own access controls).

---

## 5. Error Message Audit

### Findings

| Severity | Location | Issue |
|----------|----------|-------|
| **HIGH** | `handlers/repositories.go:549,626` | Backend `TestConnection` and `TestConnectionRequest` return raw `err.Error()` to client. Could expose internal network topology, hostnames, filesystem paths, or cloud API error details. |
| Low | `handlers/repositories.go:581` | `ShouldBindJSON` error returned to client — may expose expected type names. |

**What's done well:**
- All handlers use generic error messages for 500 errors (e.g., "failed to list agents")
- Database errors never exposed to clients
- Auth failures use generic messages ("unauthorized", "invalid credentials")
- 404s use resource-neutral messages ("agent not found")
- No stack traces returned to clients
- OIDC errors wrapped with context, not forwarded raw
- Crypto/encryption errors generalized before returning

**Fix applied:** Connection test errors now return a generic "Connection test failed" message instead of raw error text. Detailed errors logged server-side only.

---

## 6. OWASP Top 10 (2021) Checklist

### A01 — Broken Access Control
| Check | Status | Notes |
|-------|--------|-------|
| Multi-tenant isolation (org_id) | PASS | All queries filter by org_id. Verified across handlers. |
| RBAC enforcement | PASS | `RequireRole` middleware checks role hierarchy (Owner > Admin > Member > Readonly). |
| Session management | PASS | HttpOnly, Secure, SameSite=Lax. Idle timeout (30min). Max age (24hr). |
| API key scoping | PASS | API keys scoped to agents, SHA-256 hashed. |
| CORS | PASS | Production requires explicit origin whitelist. Panics if empty. |

### A02 — Cryptographic Failures
| Check | Status | Notes |
|-------|--------|-------|
| Secrets at rest | PASS | AES-256-GCM with random nonces. 32-byte master key enforced. |
| Passwords in transit | PASS | OIDC flow — no password handling. Restic passwords via env vars. |
| API key storage | PASS | SHA-256 hashed. Raw key shown once at creation. |
| Session secrets | PASS | 32-byte minimum. Secure cookie settings. |
| Key escrow | PASS | Encrypted with AES-256-GCM, admin-only recovery. |

### A03 — Injection
| Check | Status | Notes |
|-------|--------|-------|
| SQL injection | PASS | All queries use parameterized statements ($1, $2, etc.). Verified across `store.go`. |
| Command injection | PASS | Restic uses `exec.CommandContext` with argument arrays — no shell invocation. |
| XSS | PASS | React escapes by default. CSP with nonce-based script policy. |
| LDAP/XML injection | N/A | Not used. |

### A04 — Insecure Design
| Check | Status | Notes |
|-------|--------|-------|
| Rate limiting | PASS | `ulule/limiter` with memory or Redis store. Applied to API routes. |
| Account enumeration | PASS | OIDC flow — no user/password endpoint. |
| Multi-tenancy | PASS | Org isolation at query level. |

### A05 — Security Misconfiguration
| Check | Status | Notes |
|-------|--------|-------|
| Security headers | PASS | CSP, X-Frame-Options: DENY, HSTS, X-Content-Type-Options, Referrer-Policy, Permissions-Policy. |
| Swagger in production | PASS | Disabled in production via environment check. |
| Debug mode | PASS | Gin set to ReleaseMode in production. |
| Default credentials | N/A | OIDC — no local credentials. |

### A06 — Vulnerable and Outdated Components
| Check | Status | Notes |
|-------|--------|-------|
| Known CVEs | **NEEDS ATTENTION** | 5 vulnerabilities found (see Section 1). go-redis and quic-go upgraded. Go runtime upgrade pending. |

### A07 — Identification and Authentication Failures
| Check | Status | Notes |
|-------|--------|-------|
| Session fixation | PASS | New session created on login. Old session cleared on logout (MaxAge=-1). |
| Brute force | PASS | Rate limiting on auth endpoints. OIDC offloads auth to identity provider. |
| Token storage | PASS | Cookies only (HttpOnly, Secure). No localStorage tokens. |

### A08 — Software and Data Integrity Failures
| Check | Status | Notes |
|-------|--------|-------|
| CI/CD security | N/A | Out of scope (infrastructure). |
| Dependency integrity | PASS | go.sum present for dependency verification. |
| OIDC state param | PASS | 32-byte crypto/rand state. Verified on callback. Cleared after use. |

### A09 — Security Logging and Monitoring
| Check | Status | Notes |
|-------|--------|-------|
| Audit logging | PASS | `AuditLogMiddleware` logs all authenticated actions with user, org, resource, action. |
| Request logging | PASS | All requests logged with method, path, status, latency, client_ip. |
| Failed auth logging | PASS | Auth failures logged at Warn level. |
| Sensitive data in logs | **FIXED** | Webhook URLs with tokens were being logged. Now redacted. |

### A10 — Server-Side Request Forgery (SSRF)
| Check | Status | Notes |
|-------|--------|-------|
| SSRF via backends | Low Risk | Backend URLs (S3, B2, SFTP) are admin-configured, not user-supplied per-request. Validated via type-specific parsers. |
| Webhook SSRF | Low Risk | Notification webhook URLs are admin-configured. Not arbitrary user input. |

---

## 7. Summary of Fixes Applied

| # | Severity | File | Fix |
|---|----------|------|-----|
| 1 | HIGH | `internal/api/handlers/repositories.go` | Connection test errors no longer return raw `err.Error()` to client |
| 2 | HIGH | `internal/notifications/service.go` | Webhook URLs redacted from log output in `finalizeLog` |
| 3 | MEDIUM | `internal/api/handlers/search.go` | Search query `q` capped at 1000 characters |
| 4 | MEDIUM | `internal/api/handlers/agents.go` | HealthHistory limit capped at 1000 |
| 5 | MEDIUM | `internal/api/handlers/schedules.go` | Cron expressions validated with `robfig/cron/v3` parser |
| 6 | MEDIUM | `go.mod` | Upgraded `go-redis/v9` v9.0.4 → v9.17.3 |
| 7 | MEDIUM | `go.mod` | Upgraded `quic-go` (indirect) v0.54.0 → v0.59.0 |

## 8. Recommendations (Not Fixed — Requires Discussion)

1. **Go Runtime Upgrade:** Upgrade to Go 1.25.7+ to fix stdlib vulnerabilities (GO-2026-4341, GO-2026-4340, GO-2026-4337). This is an infrastructure/CI change.
2. **Request Body Size Limit:** Add `gin.Engine.MaxMultipartMemory` or a middleware to cap request body size. Currently relies on reverse proxy (nginx/caddy) limits.
3. **CSRF Token:** Currently relies on SameSite=Lax cookies. Consider adding explicit CSRF tokens for state-changing operations if supporting older browsers.
4. **Query String Logging:** Consider redacting or omitting query strings from request logs to prevent accidental sensitive parameter exposure.
