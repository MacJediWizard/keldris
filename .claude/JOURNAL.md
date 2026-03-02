# Keldris Development Journal

## 2026-01-19 - Project Initialized

### What
Created initial project scaffold.

### How
- Go server + agent stubs
- React + Vite + Tailwind frontend
- PostgreSQL schema
- Conductor configuration

### Next
Start Phase 1: Database & Models + Frontend Shell

---

## 2026-01-19 - Phase 1: Core Infrastructure (PRs #1-#12)

### What
Built the entire core platform in a single day: database layer, frontend shell, authentication, agent CLI, REST API, backup engine, frontend integration, Docker/CI, storage backends, multi-org RBAC, agent API keys, and email notifications.

### How
- PostgreSQL database layer with pgx/v5 (#1)
- React frontend shell with layout and pages (#2)
- OIDC authentication flow (#3)
- Keldris-agent CLI with Cobra (#4)
- REST API handlers for agents, repos, schedules, backups (#5)
- Restic backup wrapper with scheduler (#6)
- Frontend connected to real APIs (#7)
- Docker deployment and CI/CD pipelines (#8)
- Multiple storage backends: S3, B2, Dropbox, SFTP, REST (#9)
- Multi-organization and RBAC (#10)
- Agent API key authentication (#11)
- Email notifications for backup events (#12)

### Next
Phase 2: Monitoring, compliance, and advanced features

---

## 2026-01-19 to 2026-01-20 - Phase 2: Monitoring & Compliance (PRs #13-#24)

### What
Added encryption key management, monitoring/alerting, audit logs, storage stats, retention policies, backup verification, bandwidth controls, agent installers, agent self-update, restore workflow, DR runbooks, and multi-repo backup.

### How
- Repository encryption key management (#13)
- Monitoring and alerting system (#14)
- Audit log system for compliance (#15)
- Storage efficiency reporting and stats dashboard (#16)
- Retention policy enforcement (#17)
- Backup integrity verification (#18)
- Bandwidth and resource controls (#19)
- Agent self-update via GitHub releases (#20)
- Platform installers for Linux/macOS/Windows (#21)
- Multi-repository backup with retry/failover (#22)
- DR runbooks and testing system (#23)
- Full restore workflow with snapshot browser (#24)

### Next
Phase 3: Polish, search, and developer tools

---

## 2026-01-20 to 2026-01-21 - Phase 3: Search, Metrics & UX Polish (PRs #25-#46)

### What
Added health endpoints, API docs, tag-based search, metrics dashboard, agent details, agent health monitoring, email reports, backup policies, agent groups, compression levels, exclude patterns, maintenance windows, backup scripts, network drive support, dark mode, i18n, onboarding wizard, snapshot comparison, file history, snapshot comments, cost estimation, and OIDC group sync.

### How
- Gatus-compatible health check endpoints (#25)
- Backup tags and global search (#26)
- OpenAPI/Swagger documentation (#27)
- Comprehensive metrics and stats dashboard (#28)
- Agent details page with stats and actions (#29)
- Agent health monitoring system (#30)
- Weekly/monthly email reports (#31)
- Backup policy templates (#32)
- Agent grouping with full CRUD (#33)
- Configurable compression levels (#34)
- Pre-built exclude patterns library (#35)
- Maintenance window scheduling (#36)
- Backup scripts with pre/post hooks (#37)
- Network drive backup support (NFS/SMB/CIFS) (#38)
- UI dark mode toggle (#39)
- Multi-language support with react-i18next (#40)
- First-time onboarding wizard (#41)
- Snapshot comparison feature (#42)
- File version history browser (#43)
- Snapshot comments (#44)
- Cloud storage cost estimation (#45)
- OIDC group synchronization (#46)

### Notes
- Moved .claude/ to gitignore on this date (later reversed in 2026-02-08)
- Added AGPLv3 license
- Added README badges, logo, and editions comparison

---

## 2026-01-22 to 2026-01-24 - Phase 4: Advanced Data Protection (Branches)

### What
Built advanced data protection features across many parallel branches. These features are complete but awaiting merge to main.

### Features Built
- Data classification and labeling
- Legal hold for snapshots
- Ransomware detection
- Snapshot immutability
- Snapshot FUSE mount
- Backup dry run preview
- Resume interrupted backups (checkpoints)
- Max file size exclusion
- Restore dry run preview
- Cross-agent restore
- Cloud restore
- Partial file restore
- File search within snapshots
- Geo-replication
- Agent command queue
- Agent debug mode
- Agent registration with 2FA
- Repository import (restic)
- Repository clone config
- Schedule cloning
- CSV agent import
- Rate limiting headers and admin UI
- User sessions management
- IP allowlist
- Password policies

---

## 2026-01-25 to 2026-01-27 - Phase 5: Operations & UX (Branches)

### What
Built operational and UX features across parallel branches.

### Features Built
- Backup concurrency limits
- Backup priority queue
- Storage tiering
- Snapshot lifecycle policies
- Saved filters
- Notification rules engine
- Multi-channel notifications (Slack, Teams, Discord, PagerDuty, Webhooks)
- User-defined metadata
- Downtime tracking
- SLA tracking
- Activity feed
- User favorites
- Recent items tracker
- Breadcrumbs navigation
- Global search
- Help tooltips
- Backup calendar UI
- Snapshot file diff viewer

---

## 2026-01-27 to 2026-01-29 - Phase 6: Docker Ecosystem (Branches)

### What
Comprehensive Docker backup and management support.

### Features Built
- Docker container backup
- Docker volume backup
- Docker image backup and restore
- Docker network backup
- Docker secrets and configs backup
- Docker stack/compose backup
- Docker Swarm backup
- Docker logs backup
- Docker health monitoring
- Docker registries management
- Docker backup labels (auto-discovery)
- Container backup hooks
- Docker restore
- Komodo integration

---

## 2026-01-29 to 2026-01-31 - Phase 7: Enterprise Features (Branches)

### What
Enterprise-grade features for licensing, administration, and compliance.

### Features Built
- Pi-hole backup support
- Backup hook templates
- Restore testing/validation
- Backup validation
- Superuser admin panel
- System settings management
- User administration
- Organization admin pages
- License key generation and validation
- License feature flags
- Upgrade prompts for gated features
- Ansible agent deployment role
- Terraform provider

---

## 2026-02-01 to 2026-02-04 - Phase 8: Licensing & Portal (Branches)

### What
Built the complete licensing and portal system.

### Features Built
- Usage metering
- Pro trial (30-day)
- License portal for customers
- Enterprise branding (white-label)
- Built-in documentation viewer
- Job queue system
- Email verification
- Password reset flow
- Email invite system
- Setup wizard (first-time server setup)
- Admin health overview
- Security headers middleware

---

## 2026-02-05 to 2026-02-08 - Phase 9: Final Features & Cleanup (Branches)

### What
Final features and integration work.

### Features Built
- Anonymous telemetry
- Graceful backup shutdown
- Server update checker
- Agent diagnostics command
- Agent proxy configuration
- Config export/import
- Offline/air-gap mode
- Offline backup queue
- Skeleton loading states
- Empty states UI
- Toast notification system
- Keyboard shortcuts
- PostgreSQL direct backup
- MySQL/MariaDB direct backup
- Prometheus metrics endpoint
- Proxmox VM backup integration
- Outbound webhooks for all events

### Fixes Applied
- Stripped Co-Authored-By lines from all 2,095 commits (was added due to .claude/ being gitignored)
- Restored .claude/ directory to version control
- Only .claude/settings.local.json remains gitignored

---

## 2026-02-22 - Branch Merge Consolidation

### What
Merged all ~196 remaining feature branches into main via `remaining-integration` branch. Resolved all union merge conflicts and corruption artifacts.

### How
- Created `remaining-integration` branch from main
- Merged all feature branches using `git merge-file --union` strategy
- Fixed Go build errors: duplicate functions, leaked SQL, truncated function bodies in store.go, sla.go, notifications/service.go, middleware/auth.go
- Fixed 2,373 TypeScript errors across 39+ files caused by union merge artifacts:
  - Duplicate imports, duplicate function/component bodies, interleaved JSX
  - Duplicate mutationFn properties in hooks
  - Missing type definitions, wrong export names
  - Corrupted api.ts (28+ corruption points), types.ts (22+ duplicate types)
- Used parallel background agents for large-scale deduplication
- Fast-forward merged clean result to main

### Result
- Go build: passes clean
- TypeScript: 0 errors (`npx tsc --noEmit`)
- All feature branches merged: `git branch --no-merged main | grep MacJediWizard | wc -l` = 0

---

## Status as of 2026-02-22

### Merged to Main
All features merged — 46 original PRs + ~196 feature branches consolidated

### Architecture
- 92 database migrations
- 87 API handler files
- 60+ frontend routes
- 7 storage backends
- 6 notification channels
- 20 Docker backup files
- 15+ documentation files
- Swagger API docs

### Key Metrics
- 1,145 total commits on main
- 293 local branches (all merged)
- Go build: passes
- TypeScript: 0 errors

---

## 2026-02-22 to 2026-02-23 - CI Pipeline Fixes & Dead Code Removal

### What
Fixed all CI pipeline failures introduced by the union merge consolidation, and removed all disconnected/dead code from both Keldris and license-server repos.

### CI Fixes

**SQL Column Mismatches (store files)**
- `store_search.go`: Removed non-existent `description` column from repositories search; replaced non-existent `status` column on schedules with `CASE WHEN s.enabled THEN 'active' ELSE 'disabled' END`
- `store_recovered_full.go`: Fixed schedule queries selecting 19 columns when `scanSchedule` expects 27 (added backup_type, priority, classification fields, docker/pihole/proxmox options); fixed 3 backup queries selecting 25 columns when `scanBackups` expects 17
- `store_recovered.go`: Fixed 3 more backup queries (25→17 columns); added RowsAffected check to `UpdateSnapshotComment`
- `store_classification.go`: Fixed `bandwidth_limit_kb` → `bandwidth_limit_kbps` (2 places)
- `store_usage_metrics.go`: Fixed `schedules.org_id` reference — schedules has no org_id, must JOIN through agents

**Test Failures (5 distinct issues)**
- `Complete()` argument order: Fixed ~38 calls across store_test.go and store_integration_test.go — signature is `(snapshotID, filesNew, filesChanged, sizeBytes)`, tests had sizeBytes and filesNew swapped
- Script fields: Expanded `CreateBackup`, `UpdateBackup`, and `GetBackupByID` in store.go to include pre/post script output/error columns
- Onboarding step: Fixed test expectation from `OnboardingStepOrganization` to `OnboardingStepLicense` (step after "welcome" is "license")
- `UpdateSnapshotComment`: Added RowsAffected check to return error for non-existent comments

**Docker Build**
- Removed duplicate `FROM` lines in both Dockerfiles (union merge artifacts)
- Added `@rollup/rollup-linux-x64-musl` and `@esbuild/linux-x64` to Dockerfile.server for Alpine musl libc compatibility
- Consolidated both Dockerfiles to Alpine 3.21

### Dead Code Removal (Plan: Parts A-E)

**Keldris Backend (Part A)**
- Removed `PruneOnly()` from backup/restic.go
- Removed `GetActiveWindowFromDB()`, `GetUpcomingWindowFromDB()`, `LastRefresh()` from maintenance/maintenance.go + ~12 associated tests
- Removed `GetNextAllowedRun()`, `GetActiveDRSchedules()` from backup/scheduler.go
- Removed `DiffCompact()` from backup/compare.go + test
- Removed `DeleteDailySummariesBefore()` from db/store.go + test
- Removed `SessionOrAPIKeyMiddleware`, `OptionalAuthMiddleware` from middleware/auth.go + 4 tests

**Keldris Frontend (Part B)**
- Wired `useDashboardStats` hook to Dashboard.tsx for aggregated stat cards

**License Server (Parts C-E)**
- Added migration 007 to drop unused `admin_api_keys` table
- Added configurable function fields to mock store for 3 methods
- Removed 5 unused API methods from web/src/lib/api.ts

### Result
- CI: All 3 jobs green (Go tests, Frontend build, Docker build)
- `go vet ./...`: clean
- `staticcheck ./...`: clean
- `npx tsc --noEmit`: clean (both repos)
- License server: builds, tests pass, types check

---

## 2026-02-25 - Post-Deploy v0.0.30 Bug Fixes (Issues #256-#264)

### What
Fixed 20+ UI/functionality issues discovered after v0.0.30 deployment.

### Crash & Load Fixes
- Docker Logs page crash: null guard on `data.backups` filter (#256)
- Legal Holds page: added loading state guard before feature gate check (#257)
- Lifecycle Policies page: same loading state guard pattern (#258)
- Server Logs page: friendly message when LogBuffer not configured (404 handling) (#259)

### Dark Mode Support (12 pages)
- Agents, FileSearch, FileHistory, AuditLogs, CostEstimation, Announcements, PasswordPolicies, DockerRegistries, DowntimeHistory, OrganizationSSOSettings, Branding (#260, #261)
- Added `dark:` variant Tailwind classes to all light-mode color classes

### Multi-Channel Notifications (#262)
- Replaced email-only modal with 6-type channel selector (email, slack, teams, discord, pagerduty, webhook)
- Type-specific config forms for each channel
- Feature gating for Pro channels (Slack, Teams, Discord, PagerDuty)
- Type-specific icons in channel list
- Dark mode fixes on event badges and log status colors

### License Page Resources (#263)
- Added Servers and Storage cards to Resource Limits grid
- Updated grid to 5-column layout
- Added `max_servers` and `max_storage_bytes` to `LicenseLimits` TypeScript interface

### Onboarding Org Name (#264)
- OrganizationStep pre-fills org name from license `customer_name`
- Allows renaming default orgs to license-provided name

### Result
- `npx tsc --noEmit`: clean
- 19 files changed, 1847 insertions, 711 deletions
- 5 commits, 9 GitHub issues created (#256-#264)

---

## 2026-02-26 - Heartbeat System, Feature Gating & Telemetry Hardening

### What
Comprehensive implementation of heartbeat telemetry, feature gating for all premium features, entitlement token improvements, and CI integrity guards across both Keldris and license-server repos.

### Bug Fixes
- Fixed DR Runbooks modal triggering on free tier first login — `useDRStatus()` now catches 402 errors silently with a complete `DRStatus` fallback object
- Fixed `.gitignore` using `.claude/` (directory ignore prevented negation) — changed to `.claude/*` so `!.claude/conductor.json` works correctly
- Untracked `.claude/CLAUDE.md` and `.claude/JOURNAL.md` from git

### Heartbeat Changes
- Changed heartbeat interval from 24h to 6h for faster kill switch response
- Heartbeat now sends `integrity_hash` in metrics payload
- Heartbeat response now includes `config.feature_refresh_token` (rotating 32-byte hex)
- Validator stores refresh token with 12h expiry window (allows one missed cycle)

### Feature Gating — Route Level (Part 2)
Added `FeatureMiddleware` route wrapping for 4 previously ungated feature groups:
- Legal Holds (`FeatureLegalHolds`)
- Geo-Replication (`FeatureGeoReplication`)
- Ransomware Protection (`FeatureRansomwareProtect`)
- Custom Retention (`FeatureCustomRetention`)

### Feature Gating — Handler Level (Part 6)
Added `checker *license.FeatureChecker` field and `RequireFeature()` calls to 14 handler files:
- audit_logs, docker_backup, reports, sso_group_mappings, dr_runbooks, dr_tests, sla, notifications, repositories, lifecycle_policies, geo_replication, ransomware, legal_holds, organizations
- Storage backends gated by type in repositories.go (S3, B2, SFTP, Dropbox, REST)
- Multi-repo gated by count (second+ repo requires FeatureMultiRepo)
- Backup scheduler checks geo-replication license before cross-repo replication

### Entitlement Token (Part 7)
- Added `Nonce` field to entitlement structs in both Keldris and license-server
- License server generates 16-byte hex nonce per token
- `RequireFeature()` verifies nonce presence when validator is active

### Refresh Token (Part 8)
- `RequireFeature()` verifies refresh token validity when validator is active
- Layers 2+3 are conditional on validator presence (skipped in air-gap/test mode)
- `DynamicLicenseMiddleware` sets validator reference in gin context

### CI & Build Integrity (Part 9)
- Makefile computes SHA256 of 4 license enforcement files, embeds via LDFLAGS
- CI guard verifies `sendHeartbeat`, `heartbeatLoop`, `RequireFeature` in 14 handler files
- License server: added `DetectStaleInstances` endpoint (flags instances silent >48h)
- Fixed `FlagStaleInstances` SQL: `last_heartbeat_at` → `last_seen_at`, `resolved_at IS NULL` → `resolved = false`

### Testing & Documentation
- Created `scripts/test-heartbeat.sh` — 10 automated endpoint tests
- Created `docs/heartbeat-testing.md` — testing guide with curl examples
- Updated 11 test files to pass nil checker to handler constructors

### Files Changed — Keldris
36 files modified, 2 files created

### Files Changed — License Server
6 files modified

### Result
- `go build ./...`: clean (both repos)
- `go test ./...`: all pass (both repos)
- `go vet ./...`: clean (both repos)
- `npx @biomejs/biome check .`: 368 files, no issues
- `npm run build`: clean

---

## 2026-02-27 - Onboarding Flow Redesign (Parts 1-8)

### What
Implemented the 11-part Onboarding Flow Redesign plan (Parts 1-8). Removed OIDC from `.env`, wired the setup wizard, added a login page, made OIDC configurable through onboarding (gated by license tier), and created the dynamic OIDC provider.

### Key Changes
- **Setup wizard wired**: `SetupRequiredMiddleware` blocks API until superuser created; simplified to DB + superuser only
- **Login page**: New `/login` route with password form + optional SSO button; 401 redirects to `/login` instead of OIDC
- **OIDC removed from .env**: No longer initialized at boot from env vars; loaded from DB or configured during onboarding
- **Dynamic OIDC provider**: `OIDCProvider` wrapper with `sync.RWMutex` for hot-reload; `Get()`, `Update()`, `IsConfigured()` methods
- **OIDC onboarding step**: Added between organization and SMTP steps, gated by `FeatureOIDC` (Pro+)
- **License step first**: Reordered onboarding so license activation happens before OIDC visibility check
- **Auth status endpoint**: `GET /auth/status` reports `oidc_enabled` and `password_enabled` for login page
- **3-layer security on OIDC config**: `RequireFeature` enforces org tier, entitlement nonce, and refresh token

### Production Bugs Fixed (v0.0.43)
- Fixed OIDC 402 errors from retry storms — heartbeat now fires immediately after `SetLicenseKey` so entitlements are available before onboarding continues
- Fixed confetti animation on superuser creation, Keldris icon on login page
- Fixed upgrade prompt showing on public pages (setup, login)
- Suppressed upgrade prompt during entire onboarding flow
- Auto-rename default org from license `customer_name` during onboarding

---

## 2026-03-01 - Fix OIDC Onboarding Auto-Skip & DB Tier Persistence

### What
Fixed two critical bugs preventing the OIDC onboarding step from working:
1. **OIDC step auto-skipping** for enterprise users because `GetStatus` returned empty `license_tier`
2. **Database org tier never updated** after license activation — `SetOrgTier()` existed but was never called

### Root Cause Analysis

**Bug 1: Empty `license_tier` in status response**
- `GetStatus` used `CheckFeatureWithInfo(c.Request.Context(), ...)` which reads tier from DB
- DB had stale "free" tier because it was never updated after activation
- `omitempty` JSON tag omitted empty string → frontend saw `undefined` → treated as free → auto-skipped OIDC

**Bug 2: `SetOrgTier()` never called**
- `db/store_license.go` had `SetOrgTier()` method (upserts to `organization_licenses` table)
- `LicenseManageHandler.Activate`, `.Deactivate`, `.StartTrial` all changed in-memory tier but never persisted to DB
- This meant `RequireFeature` (used by `system_settings.go` for OIDC config) also had stale tier

### Fixes Applied

**`internal/api/handlers/license_manage.go`**:
- Added `featureChecker *license.FeatureChecker` field
- `Activate()`: calls `h.featureChecker.SetOrgTier(ctx, user.CurrentOrgID, lic.Tier)` after activation
- `Deactivate()`: calls `h.featureChecker.SetOrgTier(ctx, user.CurrentOrgID, license.TierFree)` after deactivation
- `StartTrial()`: calls `h.featureChecker.SetOrgTier(ctx, user.CurrentOrgID, lic.Tier)` after trial start

**`internal/license/features.go`**:
- Added `SetOrgTier()` method to `FeatureChecker` — writes to DB + invalidates cache

**`internal/api/handlers/onboarding.go`**:
- `GetStatus`: reads tier from gin context (`DynamicLicenseMiddleware`) for display — always current
- `completeOIDCStep`: uses `middleware.RequireFeature(c, h.checker, license.FeatureOIDC)` — canonical 3-layer check reading from DB (now correct)
- Re-added `checker *license.FeatureChecker` field

**`internal/api/middleware/features.go`**:
- Exported `GetValidator` (was `getValidator`) for use by onboarding handler

**`internal/api/routes.go`**:
- Passes `featureChecker` to both `NewLicenseManageHandler` and `NewOnboardingHandler`

### Security Layers
All 3 layers enforced consistently via `RequireFeature`:
- **Layer 1**: Org tier from DB via `FeatureChecker.CheckFeatureWithInfo()` — now correct after `SetOrgTier` fix
- **Layer 2**: Entitlement nonce from `DynamicLicenseMiddleware`
- **Layer 3**: Refresh token via `validator.HasValidRefreshToken()`

### Result
- `go build ./...`: clean
- `go test -race ./internal/api/handlers/`: all pass
- `go test -race ./internal/license/`: all pass
- 6 files changed, 105 insertions, 65 deletions

---

## 2026-03-01 - Fix OIDC Onboarding 402 (Layer 2+3 not ready) & Add Test Button

### What
Fixed the 402 "valid entitlement required" error on the OIDC onboarding step and added an OIDC test button.

### Root Cause Analysis
The previous fix (`67a0d5bd`) replaced the onboarding OIDC step's Layer 1 only check with `RequireFeature` (all 3 layers). This re-introduced the bug that `ec4f87e4` had already fixed — Layers 2+3 (entitlement nonce, refresh token) aren't reliably available immediately after license activation during onboarding.

Timeline:
1. `bb9fb05b` — OIDC step used `RequireFeature` (3 layers) → 402 error
2. `ec4f87e4` — Fixed to Layer 1 only → works
3. `67a0d5bd` — Reverted to `RequireFeature` → 402 re-introduced

### Fixes Applied

**`internal/api/handlers/onboarding.go`**:
- `completeOIDCStep`: Reverted to Layer 1 only check (`checker.CheckFeatureWithInfo`) — Layers 2+3 protect the system settings page post-onboarding
- Added `TestOIDC` endpoint (`POST /api/v1/onboarding/test-oidc`) — performs OIDC discovery against issuer URL, validates config, Layer 1 only gating

**`web/src/pages/Onboarding.tsx`**:
- Added "Test Connection" button to OIDC step — calls `POST /api/v1/onboarding/test-oidc`
- Shows green success banner with provider name on successful discovery
- Shows red error banner on failure

**`web/src/hooks/useOnboarding.ts`**:
- Added `useTestOnboardingOIDC()` mutation hook

**`web/src/lib/api.ts`**:
- Added `onboardingApi.testOIDC()` method

**`internal/api/handlers/onboarding_test.go`**:
- Added `TestOnboarding_TestOIDC` tests (success, bad issuer, missing fields)

### Design Decision: Layer 1 Only for Onboarding (REVERTED)
This approach was reverted — see next entry.

### Result
- `go build ./...`: clean
- `go vet ./...`: clean
- `go test ./internal/...`: all pass
- `npx tsc --noEmit`: clean
- `npx @biomejs/biome check`: clean

---

## 2026-03-01 - Fix All 3 Security Layers on OIDC Onboarding

### What
Restored full `RequireFeature` (all 3 security layers) on both `completeOIDCStep` and `TestOIDC` endpoints. Fixed the root cause: `SetLicenseKey` was using the HTTP request context for license server calls, which could be cancelled if the client disconnected.

### Why
User requirement: all 3 security layers (org tier, entitlement nonce, refresh token) are non-negotiable on OIDC configuration endpoints. The previous "Layer 1 only" workaround was explicitly rejected.

### Root Cause
`SetLicenseKey` passed the HTTP request context (`c.Request.Context()`) to both `activateLicense` and `sendHeartbeat`. If the client disconnected or timed out during the sequential license server calls (up to 60s total), the context would be cancelled, silently preventing Layer 2 (entitlement) and Layer 3 (refresh token) from being stored. Additionally, heartbeat failures were logged at `Debug` level, making them invisible in production.

### Changes

**`internal/license/validator.go`**:
- `SetLicenseKey` now uses `context.Background()` with 30s timeout for both `activateLicense` and `sendHeartbeat` (matches `heartbeatLoop` pattern)
- Added verification after activation: warns if entitlement (Layer 2) not received
- Added verification after heartbeat: warns if refresh token (Layer 3) not received, retries once
- Added final info log showing all 3 layer statuses after activation
- Upgraded `sendHeartbeat` failure logging from `Debug` to `Warn` level

**`internal/api/handlers/onboarding.go`**:
- `TestOIDC` endpoint now uses `RequireFeature` (all 3 layers) instead of Layer 1 only
- `completeOIDCStep` already had `RequireFeature` (restored in previous session)

### Result
- `go build ./...`: clean
- `go test -race ./...`: all pass
- `npx tsc --noEmit`: clean

---

## 2026-03-01 - Make Signing Key Required & Auto-Fetch Public Key

### What
Removed the hardcoded `DefaultLicensePublicKey` constant from Keldris. The Ed25519 public key is now fetched from the license server at startup via a new `GET /api/v1/signing-key` endpoint. The license server's signing key is now required at startup (was optional).

### Why
- Hardcoded public key was shared across all deployments — key rotation required a code change and rebuild
- License server silently started without a signing key, producing no entitlement tokens (Layer 2 broken)
- Public key should be authoritative from the license server, not baked into the binary

### Changes

**License Server** (`69f4cc7`):

`cmd/server/main.go`:
- `LS_SIGNING_PRIVATE_KEY` now required at startup (Fatal instead of Warn)

`internal/api/handlers/activations.go`:
- Added `GetSigningKey()` handler — derives public key via `.Public()`, returns `{"public_key": "<hex>"}`

`internal/api/routes.go`:
- Added `GET /api/v1/signing-key` in the public (unauthenticated) group

`cmd/keygen/main.go`:
- Renamed `AIRGAP_PUBLIC_KEY` → `LICENSE_PUBLIC_KEY`
- Updated output to note Keldris auto-fetches the key

**Keldris** (`5009f1b6`):

`internal/config/server.go`:
- Removed `DefaultLicensePublicKey` constant
- Removed `LicensePublicKey` field from `ServerConfig`
- Removed `LICENSE_PUBLIC_KEY` env var loading
- Removed unused `getEnvDefault()` helper

`cmd/keldris-server/main.go`:
- Replaced env var key loading with `fetchSigningKey()` — HTTP GET to license server
- Skipped in air-gap mode (no license server to reach)
- Logs key fingerprint (first 8 hex chars) on success

`internal/license/validator.go`:
- Fixed stale "hardcoded" comment on `verifyKeyLocally()`

`docs/heartbeat-testing.md`:
- Updated troubleshooting to reference `LICENSE_SERVER_URL` connectivity instead of env var

### Security Review
- `GetSigningKey` endpoint: verified `.Public()` returns only 32-byte public portion
- `fetchSigningKey`: HTTPS enforced, redirects disabled, response capped at 4KB, error messages sanitized
- Air-gap mode: `licPubKey` stays nil, downstream checks `len(v.publicKey) == ed25519.PublicKeySize`
- No hardcoded key remains: `grep -r "a1d5554e" .` returns nothing

### Result
- `go build ./...`: clean (both repos)
- `go test ./...`: all pass (both repos)

---

## 2026-03-01 - Fix Silent 4xx Errors & Instances Page Error Display

### What
Fixed `postJSONWithResponse` silently treating HTTP 4xx as success, and added error display to the license server's Instances page.

### Root Cause Analysis
User reported the license server's `/instances` page showed no data despite Keldris logs showing successful registration and heartbeat. Investigation revealed two bugs:

1. **`postJSONWithResponse` swallowed 4xx errors** — Only `>= 500` was treated as an error. If the license server returned 400 (Bad Request), 401, or 404, Keldris treated it as success and logged "registered with license server" even though registration failed.

2. **Instances page hid errors** — When the API call failed (auth error, network issue), `instances` was `undefined`, and `!filteredInstances?.length` evaluated to `true`, showing "No instances registered" instead of an error message.

### Changes

**Keldris** (`10a21d9c`):

`internal/license/validator.go`:
- Changed `postJSONWithResponse` error threshold from `>= 500` to `>= 400`
- Error message now includes HTTP status code and server error text
- All callers already handle errors properly (fallback to local verification or warning log)

**License Server** (`548b27e`):

`web/src/pages/Instances.tsx`:
- Destructured `error` from `useQuery` alongside `data` and `isLoading`
- Added error state branch showing red error banner with actual error message
- Now distinguishes between "no data" and "query failed"

### Impact
- Keldris will now properly log the actual error when registration/heartbeat/validation receives a 4xx
- License server web UI will show meaningful errors instead of blank tables
- Combined with request logging middleware (pushed earlier), provides full visibility for troubleshooting

### Result
- `go build ./...`: clean (both repos)
- Linting: clean

---

## 2026-03-01 - CI Fixes: Commit Outstanding Changes from Prior Sessions

### What
Fixed CI failures in both repos caused by partially committed work from prior sessions.

### Issues Found

**Keldris CI** — Two problems:
1. `staticcheck U1000`: `setupOnboardingTestRouterWithChecker` was unused (committed in prior session but never called)
2. `TestOnboarding_TestOIDC` tests referenced `POST /api/v1/onboarding/test-oidc` but the route/handler in `onboarding.go` was uncommitted

**License-server CI** — Docker build failure:
- `routes.go:96,143` referenced `instancesHandler.DetectStaleInstances` but the handler, Store interface method, and DB implementation were all uncommitted

### Root Cause
Prior sessions added routes and tests but only partially committed the changes — route registrations were pushed without their handler implementations, and tests were pushed without the endpoints they test.

### Commits

**Keldris:**
- `b09ded39` — Remove unused `setupOnboardingTestRouterWithChecker` function
- `5357a425` — Add OIDC test connection button to onboarding (handler + frontend)
- `757931cb` — Fix SetLicenseKey context cancellation, update journal

**License-server:**
- `ca48988` — Add stale instance detection, heartbeat refresh tokens, entitlement nonce

### Result
- Both repos CI green
- All Go tests pass
- Frontend builds clean
- Docker build succeeds (license-server)

---

## 2026-03-01 - Onboarding SMTP Step, Dark Mode Fixes, Docs Links, License Auto-Advance

### Overview
Four improvements to the onboarding wizard and UI polish for dark mode.

### Changes

#### 1. SMTP Onboarding Step — Full Inline Form
The SMTP step was a stub that linked to `/notifications` (blocked by onboarding redirect guard) with broken "Learn more" links. Rebuilt it as a full inline form matching the OIDCStep pattern.

**Backend (`internal/api/handlers/onboarding.go`):**
- Added `SMTPOnboardingRequest` struct (host, port, username, password, from_email, from_name, encryption)
- Added `completeSMTPStep` handler — saves SMTP settings to DB + marks step complete
- Added `TestSMTP` handler — does real TCP dial (`net.DialTimeout`) to verify SMTP server reachability
- Registered `POST /api/v1/onboarding/test-smtp` route
- Extended `OnboardingStore` interface with `GetSMTPSettings` / `UpdateSMTPSettings`
- Used `net.JoinHostPort` for IPv6-safe address formatting (caught by `go vet`)

**Frontend (`web/src/pages/Onboarding.tsx`):**
- Rebuilt `SMTPStep` with form fields: host, port, username, password, from email, from name, encryption dropdown
- Added "Test Connection" button (calls `/test-smtp`)
- Added "Skip" button (marks step complete without saving settings)
- Added "Save & Continue" button (saves settings + advances)

**Hooks & API:**
- Added `useCompleteSMTPStep` and `useTestOnboardingSMTP` hooks (`web/src/hooks/useOnboarding.ts`)
- Added `testSMTP` to `onboardingApi` (`web/src/lib/api.ts`)
- Added `SMTPOnboardingRequest` type (`web/src/lib/types.ts`)

**Tests (`internal/api/handlers/onboarding_test.go`):**
- Added `GetSMTPSettings` / `UpdateSMTPSettings` mock methods

#### 2. Dark Mode Fixes — LanguageSelector & RecentItems
Both components had white backgrounds in dark mode. Added `dark:` Tailwind variants throughout:
- `web/src/components/features/LanguageSelector.tsx` — button, dropdown, header, menu items (selected/unselected/hover states)
- `web/src/components/features/RecentItems.tsx` — clock button, dropdown panel, header, clear button, empty state, dividers, group labels, item rows, icons, text, timestamps, delete buttons

#### 3. DOCS_LINKS Fix
`DOCS_LINKS` in `Onboarding.tsx` pointed to relative paths (`/docs/getting-started`) with no matching routes. Changed to GitHub blob URLs using a `DOCS_BASE` constant:
```
https://github.com/MacJediWizard/keldris/blob/main/docs/{file}.md
```

#### 4. License Step Auto-Advance
User reported having to manually click Continue after license activation. Added `onComplete()` calls immediately after successful `activateMutation.mutateAsync()` and `startTrialMutation.mutateAsync()` so the wizard auto-advances.

### Commits
- `911ff0c` — Improve onboarding wizard and fix dark mode issues
- `d5cdf0e` — Fix Biome formatting in SMTPStep component
- `a85e88a` — Fix Biome formatting in RecentItems and import ordering in api.ts

### CI Issues & Fixes
- **Run 1**: Biome caught 3 formatting errors in Onboarding.tsx (ternary formatting, text wrapping)
- **Run 2**: Biome caught 2 more — RecentItems.tsx `<span>` formatting, api.ts import ordering (`SMTPOnboardingRequest` wasn't alphabetical)
- **Run 3**: All green (Go tests, Frontend/Biome, Docker build)

### Result
- All CI checks pass
- Go build + tests clean
- TypeScript strict mode clean
- Biome formatting clean

---

## 2026-03-02 - Inline Repository Creation Form in Onboarding Wizard

### What
The onboarding RepositoryStep linked to `/repositories` which got bounced by the onboarding redirect guard. Replaced it with a full inline repository creation form, matching the OIDC and SMTP step patterns.

### Changes

**`web/src/pages/Onboarding.tsx`:**
- Added `RepoField` helper component for labeled form inputs (keeps backend field markup DRY)
- Rebuilt `RepositoryStep` with three render paths:
  1. **Password display** — shown after creation with copy button + amber warning
  2. **Already has repository** — green success banner + Continue
  3. **Inline creation form** — name, type dropdown, backend-specific fields, key escrow, test/create buttons
- Backend-specific fields for all 6 types: local, S3, B2, SFTP (with textarea for private key), REST, Dropbox
- `buildConfig()` and `canCreate` validation adapted from `AddRepositoryModal` in `Repositories.tsx`
- Test Connection and Create Repository buttons gated by `canCreate`
- Skip button calls `onComplete` (marks step done without creating)
- Reverted the `Layout.tsx` workaround that bypassed the onboarding redirect for `/repositories`

**Imports added:**
- `useCreateRepository`, `useTestConnection` from `useRepositories` hook
- `BackendConfig`, `RepositoryType`, `TestRepositoryResponse` from types

### No backend changes
All API endpoints (`POST /api/v1/repositories`, `POST /api/v1/repositories/test-connection`) already existed.

### Result
- `npx tsc --noEmit`: clean
- `npx biome check`: clean
