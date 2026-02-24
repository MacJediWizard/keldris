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
