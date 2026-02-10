# Keldris Full Integration Verification Report

**Date:** 2026-02-09
**Branch:** MacJediWizard/full-integration-verify

---

## 1. Build Verification

| Command | Result | Notes |
|---------|--------|-------|
| `make deps` | :white_check_mark: PASSED | Go modules downloaded, npm ci installed 207 packages, 0 vulnerabilities |
| `make lint` | :white_check_mark: PASSED | `go vet`, `staticcheck`, and `biome check` all pass. 79 frontend files checked. |
| `make test` | :white_check_mark: PASSED | All tests pass with `-race -cover`. See coverage details below. |
| `make build` | :white_check_mark: PASSED | Server, agent, and frontend (Vite) all build successfully. 188 modules transformed. |

### Test Coverage Summary

| Package | Coverage | Notes |
|---------|----------|-------|
| internal/license | 98.3% | Excellent |
| internal/crypto | 81.5% | Good |
| internal/config | 50.0% | Moderate |
| internal/api/middleware | 32.2% | Low |
| internal/auth | 31.1% | Low |
| internal/updater | 26.0% | Low |
| internal/models | 25.3% | Low |
| internal/notifications | 20.0% | Low |
| internal/api/handlers | 5.2% | Very low |
| internal/backup | 5.5% | Very low |
| internal/db | 1.1% | Very low |
| cmd/*, docs/api, internal/api, backup/backends, backup/docker, cost, dr, excludes, health, maintenance, metrics, monitoring, reports | 0.0% | No test files or untestable mains |
| pkg/models | N/A | No test files |

:warning: **Warning:** Many packages have low or zero test coverage. Not broken, but should be improved over time.

### Build Warnings (Non-blocking)

- `go-m1cpu` external dependency emits two C compiler warnings (`-Wgnu-folding-constant`) during test builds on Apple Silicon. These are harmless and from a third-party dependency.

---

## 2. Database Migration Verification

:white_check_mark: **PASSED** - 29 migration files, sequential numbering, valid SQL

| Check | Result |
|-------|--------|
| Migration files present | 29 files (001-029) |
| Sequential numbering | No gaps, no duplicates |
| SQL syntax | All valid PostgreSQL |

### Migration File List

| # | File | Description |
|---|------|-------------|
| 001 | `001_initial_schema.sql` | Organizations, users, agents, repositories, schedules, backups |
| 002 | `002_org_memberships.sql` | Multi-org with RBAC and invitations |
| 003 | `003_alerts.sql` | Alert rules and alerts |
| 004 | `004_add_notifications.sql` | Notification channels, preferences, logging |
| 005 | `005_repository_keys.sql` | Repository encryption keys with escrow |
| 006 | `006_add_audit_logs.sql` | Audit log tracking |
| 007 | `007_add_storage_stats.sql` | Storage statistics |
| 008 | `008_add_retention_logging.sql` | Retention action logging |
| 009 | `009_verification.sql` | Backup verification schedules |
| 010 | `010_add_bandwidth_controls.sql` | Bandwidth limits and time windows |
| 011 | `011_add_restores.sql` | Restore operation tracking |
| 012 | `012_add_dr_tables.sql` | DR runbooks and test tracking |
| 013 | `013_multi_repo_schedules.sql` | Multi-repository and replication |
| 014 | `014_add_tags.sql` | Backup/snapshot tagging |
| 015 | `015_metrics_history.sql` | Time-series metrics |
| 016 | `016_agent_health.sql` | Agent health monitoring |
| 017 | `017_add_reports.sql` | Report schedules and history |
| 018 | `018_add_backup_policies.sql` | Reusable backup templates |
| 019 | `019_agent_groups.sql` | Agent grouping |
| 020 | `020_add_compression_level.sql` | Compression settings |
| 021 | `021_add_exclude_patterns.sql` | Exclude patterns library |
| 022 | `022_add_maintenance_windows.sql` | Maintenance pause windows |
| 023 | `023_backup_scripts.sql` | Pre/post backup hooks |
| 024 | `024_add_network_mounts.sql` | Network mount tracking |
| 025 | `025_add_onboarding.sql` | Onboarding progress |
| 026 | `026_add_snapshot_comments.sql` | Snapshot comments/notes |
| 027 | `027_add_cost_estimation.sql` | Cost estimation and alerts |
| 028 | `028_add_sso_group_mappings.sql` | SSO group-to-role mappings |
| 029 | `029_add_soft_delete.sql` | Soft delete for historical records |

---

## 3. API Route Verification

:white_check_mark: **PASSED** - All registered endpoints have corresponding handler files and functions

### Public Routes (No Auth)

| Method | Path | Handler | File |
|--------|------|---------|------|
| GET | `/health` | HealthHandler.Check | `health.go` |
| GET | `/api/v1/version` | VersionHandler.Get | `version.go` |
| GET | `/api/v1/metrics` | MetricsHandler.Get | `metrics.go` |
| GET | `/api/docs/*` | Swagger UI | `docs/api/` |

### Auth Routes

| Method | Path | Handler | File |
|--------|------|---------|------|
| GET | `/auth/login` | AuthHandler.Login | `auth.go` |
| GET | `/auth/callback` | AuthHandler.Callback | `auth.go` |
| POST | `/auth/logout` | AuthHandler.Logout | `auth.go` |
| GET | `/auth/me` | AuthHandler.Me | `auth.go` |

### Protected API Routes (`/api/v1/*` with auth + audit middleware)

| Resource | Handler File | Exists |
|----------|-------------|--------|
| Agents | `agents.go` | :white_check_mark: |
| Agent API (key-based) | `agent_api.go` | :white_check_mark: |
| Agent Groups | `agent_groups.go` | :white_check_mark: |
| Alerts | `alerts.go` | :white_check_mark: |
| Audit Logs | `audit_logs.go` | :white_check_mark: |
| Backups | `backups.go` | :white_check_mark: |
| Backup Scripts | `backup_scripts.go` | :white_check_mark: |
| Cost Estimation | `cost_estimation.go` | :white_check_mark: |
| DR Runbooks | `dr_runbooks.go` | :white_check_mark: |
| DR Tests | `dr_tests.go` | :white_check_mark: |
| Exclude Patterns | `exclude_patterns.go` | :white_check_mark: |
| File History | `file_history.go` | :white_check_mark: |
| Maintenance | `maintenance.go` | :white_check_mark: |
| Notifications | `notifications.go` | :white_check_mark: |
| Onboarding | `onboarding.go` | :white_check_mark: |
| Organizations | `organizations.go` | :white_check_mark: |
| Policies | `policies.go` | :white_check_mark: |
| Reports | `reports.go` | :white_check_mark: |
| Repositories | `repositories.go` | :white_check_mark: |
| Restore | `restore.go` | :white_check_mark: |
| Schedules | `schedules.go` | :white_check_mark: |
| Search | `search.go` | :white_check_mark: |
| Snapshots | `snapshots.go` | :white_check_mark: |
| SSO Group Mappings | `sso_group_mappings.go` | :white_check_mark: |
| Stats (Storage) | `stats.go` | :white_check_mark: |
| Tags | `tags.go` | :white_check_mark: |
| Version | `version.go` | :white_check_mark: |

Each handler uses the `RegisterRoutes(*gin.RouterGroup)` pattern and defines its own store interface.

---

## 4. Frontend Route Verification

:white_check_mark: **PASSED** - 26 routes, all components exist, zero orphaned pages

| Route Path | Component | File Exists |
|-----------|-----------|-------------|
| `/` | Dashboard | :white_check_mark: |
| `/agents` | Agents | :white_check_mark: |
| `/agents/:id` | AgentDetails | :white_check_mark: |
| `/agent-groups` | AgentGroups | :white_check_mark: |
| `/repositories` | Repositories | :white_check_mark: |
| `/schedules` | Schedules | :white_check_mark: |
| `/policies` | Policies | :white_check_mark: |
| `/backups` | Backups | :white_check_mark: |
| `/dr-runbooks` | DRRunbooks | :white_check_mark: |
| `/restore` | Restore | :white_check_mark: |
| `/file-history` | FileHistory | :white_check_mark: |
| `/snapshots/compare` | SnapshotCompare | :white_check_mark: |
| `/alerts` | Alerts | :white_check_mark: |
| `/notifications` | Notifications | :white_check_mark: |
| `/reports` | Reports | :white_check_mark: |
| `/audit-logs` | AuditLogs | :white_check_mark: |
| `/stats` | StorageStats | :white_check_mark: |
| `/stats/:id` | RepositoryStatsDetail | :white_check_mark: |
| `/tags` | Tags | :white_check_mark: |
| `/costs` | CostEstimation | :white_check_mark: |
| `/organization/members` | OrganizationMembers | :white_check_mark: |
| `/organization/settings` | OrganizationSettings | :white_check_mark: |
| `/organization/sso` | OrganizationSSOSettings | :white_check_mark: |
| `/organization/maintenance` | Maintenance | :white_check_mark: |
| `/organization/new` | NewOrganization | :white_check_mark: |
| `/onboarding` | Onboarding | :white_check_mark: |

All 26 pages use `React.lazy()` code-splitting. All imports resolve. 31 custom hooks verified present.

---

## 5. Model/Store Verification

:white_check_mark: **PASSED** - 34 models in `internal/models/`, 3 shared models in `pkg/models/`, 330+ store methods

### Models with Full CRUD (31/34)

Agent, AgentGroup, AlertRule, Backup, BackupScript, CostAlert, DRRunbook, DRTest, DRTestSchedule, ExcludePattern, MaintenanceWindow, Organization, OrgMembership, OrgInvitation, Policy, Repository, RepositoryKey, ReplicationStatus, ReportSchedule, Restore, Schedule, ScheduleRepository, SnapshotComment, SSOGroupMapping, StoragePricing, Tag, User, Verification, VerificationSchedule, NotificationChannel, NotificationPreference

### Models with Partial Operations (by design)

| Model | Operations | Reason |
|-------|-----------|--------|
| AuditLog | Create + List | Immutable append-only log |
| CostEstimate | Create + List | Append-only historical data |
| StorageStats | Create + List | Append-only timeseries |
| ReportHistory | Create + List + Update | Historical record |
| NotificationLog | Create + List + Update | Append-only with status updates |
| OnboardingProgress | Create + Get + Update | State machine (no delete) |
| AgentHealthHistory | Create + Get | Append-only health tracking |
| Alert | Create + Get + Update | Status-based lifecycle (no hard delete) |

### :warning: Warning: MetricsHistory Gap

- `CreateMetricsHistory()` exists in the store but there are **no corresponding Get/List methods**
- File: `internal/db/store.go`
- Impact: Metrics are written but cannot be retrieved via store. The `GetDashboardStats()` and other aggregate methods may handle this via direct SQL, but dedicated retrieval methods are missing.

### Shared Models (`pkg/models/`)

| File | Structs |
|------|---------|
| `agent.go` | AgentInfo, OSInfo, HeartbeatRequest, HeartbeatMetrics, HeartbeatResponse |
| `backup.go` | BackupStatus, BackupRequest, BackupResponse |
| `common.go` | APIError, APIResponse |

---

## 6. Feature Completeness Check

### License Features

:white_check_mark: **EXISTS** - `internal/license/features.go`

| Feature | Tier |
|---------|------|
| OIDC Authentication | Pro+ |
| Audit Logging | Pro+ |
| Multiple Organizations | Enterprise |
| SLA Tracking | Enterprise |
| White-Label Branding | Enterprise |
| Air-Gapped Deployment | Enterprise |

### Notification Channels

:white_check_mark: **ALL 6 PRESENT**

| Channel | File | Config Struct |
|---------|------|---------------|
| Email | `email.go` | EmailChannelConfig |
| Slack | `slack.go` | SlackChannelConfig |
| Teams | `teams.go` | TeamsChannelConfig |
| Discord | `discord.go` | DiscordChannelConfig |
| PagerDuty | `pagerduty.go` | PagerDutyChannelConfig |
| Webhook | `webhook.go` | WebhookChannelConfig |

### Storage Backends

:white_check_mark: **ALL 6 PRESENT** (exceeds the 4 required)

| Backend | File | Type Constant |
|---------|------|---------------|
| Local | `backends/local.go` | `RepositoryTypeLocal` |
| S3 | `backends/s3.go` | `RepositoryTypeS3` |
| B2 | `backends/b2.go` | `RepositoryTypeB2` |
| Dropbox | `backends/dropbox.go` | `RepositoryTypeDropbox` |
| SFTP | `backends/sftp.go` | `RepositoryTypeSFTP` |
| REST | `backends/rest.go` | `RepositoryTypeRest` |

### Docker Backup Support

:white_check_mark: **FULLY IMPLEMENTED** - `internal/backup/docker/`

| File | Purpose |
|------|---------|
| `docker.go` | Container and volume operations |
| `detector.go` | Auto-discovery of containers/volumes |
| `volumes.go` | Volume management |
| `types.go` | Type definitions |

### Backup Core Files

:white_check_mark: Both `restic.go` and `scheduler.go` exist with additional supporting files:
`storage.go`, `stats.go`, `stats_collector.go`, `retention.go`, `verification.go`, `compare.go`, `filehistory.go`, `network_drives.go`

---

## 7. Dependency Check

### Go Dependencies (`go.mod`)

:white_check_mark: **All required dependencies present**

| Dependency | Version | Purpose |
|------------|---------|---------|
| github.com/gin-gonic/gin | v1.11.0 | Web framework |
| github.com/jackc/pgx/v5 | v5.7.2 | PostgreSQL driver |
| github.com/coreos/go-oidc/v3 | v3.17.0 | OIDC auth |
| golang.org/x/oauth2 | v0.34.0 | OAuth2 |
| golang.org/x/crypto | v0.40.0 | Cryptography |
| github.com/spf13/cobra | v1.8.1 | Agent CLI |
| github.com/rs/zerolog | v1.33.0 | Logging |
| github.com/ulule/limiter/v3 | v3.11.2 | Rate limiting |
| github.com/gorilla/sessions | v1.4.0 | Session management |
| github.com/aws/aws-sdk-go-v2 | v1.41.1 | AWS S3 |
| github.com/google/uuid | v1.6.0 | UUIDs |
| github.com/robfig/cron/v3 | v3.0.1 | Cron scheduling |

### Frontend Dependencies (`web/package.json`)

:white_check_mark: **All required dependencies present**

| Dependency | Version | Purpose |
|------------|---------|---------|
| react | ^18.3.1 | UI framework |
| react-dom | ^18.3.1 | React DOM |
| react-router-dom | ^7.1.0 | Client routing |
| @tanstack/react-query | ^5.62.0 | Data fetching |
| tailwindcss | ^3.4.17 | Styling |
| typescript | ~5.6.2 | Type safety |
| vite | ^6.0.0 | Build tool |
| @biomejs/biome | ^1.9.4 | Linter |
| i18next | ^25.7.4 | Internationalization |

### `go mod tidy`

:white_check_mark: No changes - `go.mod` and `go.sum` are clean.

### `npm ci`

:white_check_mark: 207 packages installed, 0 vulnerabilities.

---

## 8. Security Check

### Hardcoded Secrets

:white_check_mark: **No production secrets found**

- `.env.example` contains only placeholder values
- `internal/auth/session_test.go` uses a test-only secret (acceptable)
- All credentials loaded via environment variables

### Auth Middleware Coverage

:white_check_mark: **All protected routes have auth middleware**

| Route Group | Auth Method |
|-------------|-------------|
| `/health`, `/api/v1/version`, `/api/v1/metrics` | Public (no auth) |
| `/auth/*` | Auth routes (login/callback/logout) |
| `/api/v1/*` | `AuthMiddleware` (session-based) + `AuditMiddleware` |
| `/api/v1/agent/*` | `APIKeyMiddleware` (agent key-based) |

### CORS Configuration

:white_check_mark: **EXISTS** - `internal/api/middleware/cors.go`

- Configurable origins, credentials allowed, proper methods/headers
- :warning: AllowedOrigins defaults to empty (allows all origins in dev mode) - should be explicitly set for production

### Rate Limiting

:white_check_mark: **EXISTS** - `internal/api/middleware/ratelimit.go`

- Memory-based store via `ulule/limiter/v3`
- Default: 100 requests per minute (configurable)
- Applied globally

### Query Safety

:white_check_mark: All database queries use parameterized `$1, $2, ...` placeholders via `pgx/v5`. No string concatenation in SQL.

### Session Security

:white_check_mark: Cookies: `HttpOnly=true`, `Secure=true`, `SameSite=Lax`, 24h expiry, min 32-byte secret enforced.

### Encryption at Rest

:white_check_mark: AES-256-GCM via `internal/crypto/aes.go` for repository configs and notification channel configs.

---

## 9. Docker Verification

| Check | Result | Notes |
|-------|--------|-------|
| `docker/Dockerfile.server` | :white_check_mark: EXISTS | Multi-stage build (Node + Go), non-root user, alpine 3.19 |
| `docker/Dockerfile.agent` | :white_check_mark: EXISTS | Go builder with restic, cross-compilation support, non-root |
| `docker/docker-compose.yml` | :white_check_mark: EXISTS | 3 services (server, postgres, agent), health checks, volumes |

### docker-compose.yml Details

- Server: Depends on healthy postgres, exposes 8080, OIDC env vars
- PostgreSQL: 15-alpine, `pg_isready` health check, persistent volume
- Agent: Optional via profiles, depends on healthy server
- All services: `restart: unless-stopped`

:warning: **Note:** `docker-compose config` not run (requires Docker daemon). Files are syntactically valid upon manual inspection.

---

## 10. Documentation Check

| Check | Result | Notes |
|-------|--------|-------|
| `README.md` | :white_check_mark: EXISTS | Comprehensive: features, architecture diagram, tech stack, getting started, editions table |
| `.claude/CLAUDE.md` | :white_check_mark: EXISTS | Full development guidelines, patterns, conventions |
| `docs/` directory | :white_check_mark: HAS CONTENT | agent-installation.md, bare-metal-restore.md, network-mounts.md, gatus-config-example.yaml |
| API docs | :white_check_mark: EXISTS | Swagger/OpenAPI: swagger.yaml (52K), swagger.json (112K), docs.go (112K) |
| API docs endpoint | :white_check_mark: EXISTS | `/api/docs/*` registered in routes.go |

### :warning: README Outdated Roadmap

The README lists "Slack/Discord notifications" as **Roadmap** items, but both are already fully implemented in `internal/notifications/`. The Roadmap section should be updated to move these to "Implemented".

---

## Summary

### :white_check_mark: Passed Checks (38)

- `make deps` builds successfully
- `make lint` passes (go vet, staticcheck, biome)
- `make test` all tests pass
- `make build` server, agent, and frontend all compile
- 29 migrations with correct sequential numbering
- All migration SQL is syntactically valid
- All API handler files exist
- All handler functions referenced in routes.go exist
- All 26 frontend routes map to existing page components
- All lazy imports resolve
- No orphaned page files
- 31 custom React hooks verified
- 34 model files with comprehensive struct definitions
- 330+ store methods covering all models
- License features.go exists with 6 gated features
- All 6 notification channels implemented (email, slack, teams, discord, pagerduty, webhook)
- All 6 storage backends implemented (local, s3, b2, dropbox, sftp, rest)
- Docker backup support fully implemented
- restic.go and scheduler.go exist
- All required Go dependencies present
- All required frontend dependencies present
- `go mod tidy` produces no changes
- `npm ci` installs with 0 vulnerabilities
- No hardcoded production secrets
- Auth middleware covers all protected routes
- CORS middleware configured
- Rate limiting middleware configured
- All SQL queries use parameterized placeholders
- Session cookies properly secured
- AES-256-GCM encryption for sensitive data at rest
- Both Dockerfiles exist and follow best practices
- docker-compose.yml valid with health checks
- README.md is comprehensive
- CLAUDE.md development guidelines present
- docs/ directory has substantive content
- Swagger/OpenAPI docs generated
- API docs endpoint registered
- Audit logging middleware on all protected routes

### :x: Failed Checks (0)

None.

### :warning: Warnings (5)

1. **Low test coverage** - Many packages at 0-5% coverage. Priority files: `internal/api/handlers/`, `internal/backup/`, `internal/db/store.go`
2. **MetricsHistory store gap** - `CreateMetricsHistory()` exists but no `GetMetricsHistory()` or `ListMetricsHistory()`. File: `internal/db/store.go`
3. **README roadmap outdated** - Slack/Discord notifications listed as "Roadmap" but are already implemented. File: `README.md`
4. **CORS dev default** - Empty AllowedOrigins allows all origins; ensure this is set for production. File: `internal/api/middleware/cors.go`
5. **Rate limiting is memory-based** - Works for single server but not for distributed deployments. File: `internal/api/middleware/ratelimit.go`

### Files That Need Fixes (Do Not Fix)

| File | Issue | Priority |
|------|-------|----------|
| `internal/db/store.go` | Add `GetMetricsHistory()` and `ListMetricsHistory()` methods | Medium |
| `README.md` | Update Roadmap section - move Slack/Discord to Implemented | Low |
| `internal/api/middleware/cors.go` | Document production CORS configuration requirement | Low |
| `internal/api/handlers/*_test.go` | Add tests for handlers (5.2% coverage) | Medium |
| `internal/backup/*_test.go` | Add tests for backup package (5.5% coverage) | Medium |
| `internal/db/store_test.go` | Add tests for store methods (1.1% coverage) | Medium |
