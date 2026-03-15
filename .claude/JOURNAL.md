# Keldris Development Journal

## 2026-03-08 - Cross-Repo Audit: 25 Issues Fixed (Keldris PR #266, License Server PR #1)

### What
Exhaustive audit across both Keldris and license-server repos using 8 parallel analysis agents. Found 25 verified issues (2 critical, 13 high, 10 medium). All fixed, tested, and merged.

### Critical Fixes
- **Multi-org bug** (31 handler files): All handlers used `dbUser.OrgID` (user's primary/default org from DB) instead of `user.CurrentOrgID` (currently selected org from session). This completely broke the org-switcher feature — users always operated on their primary org regardless of which org they selected.
- **Postgres backup stderr deadlock** (`databases/postgres.go`): Synchronous 4KB `Read()` before `cmd.Wait()` deadlocked when stderr exceeded buffer size. Replaced with goroutine using `io.ReadAll()`.

### High Priority Fixes
- **9 missing `rows.Err()` checks** (5 store files): Added checks after every `for rows.Next()` loop in `store_backup_validation.go`, `store_email_verification.go`, `store_password_policy.go`, `store_system_settings.go`, `store_user_management.go`
- **EndImpersonation auth bypass** (`superuser.go`): Used `GetUser()` instead of `RequireSuperuser()` — any authenticated user could end impersonation sessions
- **Telemetry Stop() deadlock** (`telemetry.go`): Bare `<-doneCh` could block indefinitely during shutdown — added 5s timeout
- **Webhook goroutine context leak** (`dispatcher.go`): Bare `context.Background()` → `context.WithTimeout(context.Background(), 30s)`
- **DB backup cleanup context leak** (`db_backup.go`): Same pattern → 5-minute timeout
- **Mount auto-unmount race** (`mount.go`): Timer fire could race with `ExpiresAt` update — now re-checks under lock
- **Proxmox partial file leak** (`proxmox.go`): Failed `io.Copy` left partial file on disk — added `os.Remove(destPath)`
- **Agent client missing request contexts** (`client.go`): `http.NewRequest` → `http.NewRequestWithContext` with 30s timeout
- **Unbounded script timeout** (`scheduler.go`): User-provided timeout had no cap — added 24h maximum

### Medium Priority Fixes
- **Dashboard Math.max on empty arrays** (`Dashboard.tsx`): `Math.max(...[])` returns `-Infinity` — guarded with length check
- **Schedules state in render path** (`Schedules.tsx`): Form initialization moved from render to `useEffect`

### License Server Fixes (6 files)
- **Heartbeat entitlement token stale** (`instances.go`): Layer 2 entitlement tokens expire in 24h but heartbeat never refreshed them — now generates and returns fresh token
- **7 missing `rows.Err()` checks** (`store.go`)
- **Fragile trial string comparison** (`trials.go`): `err.Error() == "..."` → `errors.Is(err, pgx.ErrNoRows)`
- **Silent trial DB error** (`trials.go`): Non-ErrNoRows errors returned false `{"has_trial": false}` instead of 500
- **Silent metrics unmarshal** (`instances.go`): `_ = json.Unmarshal` → error check with `logger.Warn()`

### Test Changes
- Removed 46 obsolete "user_not_found" test cases from 13 test files (test path no longer exists after removing `GetUserByID` calls)
- Fixed 8 "repo access denied" tests in `verifications_test.go`: changed from different user ID / same org to different org ID (access control is now org-based)

### Files Modified
- **Keldris**: 61 files (31 handlers, 13 test files, 5 store files, 12 other)
- **License Server**: 6 files (handlers, routes, store, tests)
- Net: 613 insertions, 2574 deletions

---

## 2026-03-07 - Codebase Audit: 42-Issue Fix (PR #265)

### What
Exhaustive audit across 8 parallel analysis agents identified 42 issues (6 critical, 13 high, 13 medium, 10 low). All fixed, tested, merged to main.

### Critical Fixes
- **Plaintext config overwrite** (`notifications.go:317`): Deleted line that overwrote AES-encrypted channel config with plaintext
- **Missing TierLimits field** (`limits.go`): Added `MaxRepositories` to `TierLimits` struct and all tier definitions
- **Wrong type parse** (`superuser.go:508`): Changed `uuid.Parse` to `strconv.Atoi` for limit query param
- **Unregistered route** (`postgres.go`): Wired `EncryptPassword` handler to route group
- **Nil trial response** (`trial.go:268`): Added error check with fallback JSON

### High Priority Fixes
- **Goroutine context leaks** (5 files): Added `context.WithTimeout(context.Background(), ...)` to goroutines in superuser.go, system_health.go, audit.go, notifications/service.go (5 goroutines)
- **Session impersonation leak** (`session.go`): `SetUser` now clears impersonation keys on normal login
- **Wrong org ID in admin check** (`ratelimit.go`): Replaced DB lookup with direct `user.CurrentOrgID`
- **EOF string comparison** (`pihole.go`): Changed `err.Error() == "EOF"` to `err == io.EOF`
- **File handle leak** (`docker/logs.go`): Added `defer outFile.Close()` immediately after creation
- **Dead code** (`keymanager.go`): Removed private `masterKeyFromBase64` duplicating public method
- **Silent encryption failures** (`migration/export.go`, `import.go`): Changed `if err == nil` to `if err != nil` with warning logs
- **Stub actions returning nil** (`notifications/rules.go`): 5 stubs now return `fmt.Errorf` so callers know action failed

### Medium Priority Fixes
- **rows.Err() checks** (`store_docker.go`, `store_docker_logs.go`): Added 6 missing checks after `for rows.Next()` loops
- **JSON unmarshal errors** (`store_docker_logs.go`): 4 unchecked `json.Unmarshal` calls now log warnings
- **Ignored error** (`organizations.go:426`): Added `h.logger.Warn()` for membership lookup failure
- **Metrics scheduler nil panic** (`metrics/scheduler.go`): Guard `if s.stop == nil` in `Stop()`
- **Reports data race** (`reports/scheduler.go`): Changed pointer capture to value copy in closure
- **Shutdown drain timeout** (`shutdown/manager.go`): Replaced blocking `<-ctx.Done()` with `time.NewTimer` + `select`
- **SLA breach errors discarded** (`sla/tracker.go`): 3 instances now log via `t.logger.Error()`
- **Version parsing** (`updates/checker.go`): Replaced `fmt.Sscanf` with `strconv.Atoi` for proper error handling
- **Cache invalidation** (`useSavedFilters.ts`, `useFavorites.ts`): Fixed `['key', undefined]` → `['key']` for prefix matching

### Low Priority Fixes
- **Agent disabled schedules** (`cmd/keldris-agent/main.go`): Added `if !sched.Enabled { continue }` check
- **Agent backup timeout** (`cmd/keldris-agent/main.go`): Added 24h timeout context for restic operations
- **Agent report retry** (`agent/client.go`): 3-attempt retry with 2s delay for 5xx errors
- **HTTP connection pooling** (`agent/client.go`): Added `http.Transport` with connection limits
- **Duplicate isAdmin** (`usage.go`): Replaced inline check with shared `isAdmin()` function
- **Clarifying comments** on 4 design decisions across scheduler, runbook, storage, docker

### Test Fixes
- **License signing key mismatch** (`license_test.go`): Test key aligned to `"keldris-license-signing-key-dev"`
- **TestRedactArgs** (`restic_test.go`): Test expected no redaction but `redactArgs` redacts `--repo` values — fixed expectations

### Files Modified (35)
- `cmd/keldris-agent/main.go`
- `internal/agent/client.go`, `storage.go`
- `internal/api/handlers/` — notifications, organizations, postgres, ratelimit, superuser, system_health, trial, usage
- `internal/api/middleware/audit.go`
- `internal/auth/session.go`
- `internal/backup/` — apps/pihole, docker/logs, scheduler, restic_test
- `internal/crypto/keymanager.go`, `keymanager_test.go`
- `internal/db/store_docker.go`, `store_docker_logs.go`
- `internal/dr/runbook.go`
- `internal/license/license.go`, `limits.go`, `license_test.go`
- `internal/metrics/scheduler.go`
- `internal/migration/export.go`, `import.go`
- `internal/notifications/rules.go`, `service.go`
- `internal/reports/scheduler.go`
- `internal/shutdown/manager.go`
- `internal/sla/tracker.go`
- `internal/updates/checker.go`
- `web/src/hooks/useFavorites.ts`, `useSavedFilters.ts`

### Result
- `go test ./...`: all pass
- `staticcheck ./...`: clean
- PR #265 merged to main

---

## 2026-03-04 - Build Fix: LicenseTier Type and Docker Build

### What
Docker build (v0.0.59) failed on `tsc -b` due to `'pro'` being removed from `LicenseTier` type while `TierBadge.tsx` still used it as a `Record<LicenseTier, string>` key. Local `npx vite build` passed because it uses esbuild (no strict TS checking), but the Dockerfile runs `tsc -b && vite build`.

### Fix
Restored `'pro'` to `LicenseTier` type — the backend still sends `'pro'` for trial tier, so it must remain in the union. Docker build now passes. Agent re-registered successfully after deploy; server, agent, and license server all connected.

### Files Modified
- `web/src/lib/types.ts` — added `'pro'` back to `LicenseTier`

---

## 2026-03-04 - Exhaustive Codebase Audit: 44 Fixes Across Server, Agent, DB, Notifications, and Frontend

### What
Full codebase analysis identified 9 critical, 10 high, 15 medium, and 10 low severity issues. All 44 were fixed and verified (server, agent, frontend all build clean; staticcheck passes).

### Critical Fixes
- **Empty retention guard** (`restic.go`): `Forget()` and `Prune()` now return no-op when all `--keep-*` values are zero, preventing accidental deletion of all snapshots
- **Nil deref in heartbeat** (`cmd/keldris-agent/main.go`): Guard `req.Metrics` before accessing fields when metrics collection fails
- **Wrong restic binary** (`cmd/keldris-agent/main.go`): `runBackup`, `refreshSchedules`, `executeDryRun` now pass the managed binary path instead of relying on PATH
- **Wrong DB column names** (`store_superuser.go`, `store_telemetry.go`, `store_usage_metrics.go`): Fixed `key`→`setting_key`, `value`→`setting_value`, `last_seen_at`→`last_seen`, wrong table names, wrong status value
- **Hardcoded HMAC key** (`license.go`): Replaced with `HMAC_SIGNING_KEY` env var (dev fallback preserved)
- **Encrypted config unmarshaling** (`notifications/service.go`, `rules.go`): All 22 instances of `json.Unmarshal(channel.ConfigEncrypted)` replaced with proper `decryptConfig()` calls
- **Email sent to self** (`notifications/service.go`): Now uses `Recipients` field instead of `From` address
- **Dockerfile Go 1.25** (`Dockerfile.server`, `go.mod`): Go 1.25 doesn't exist; changed to 1.24
- **Corrupted docker-compose** (`docker-compose.images.yml`): Removed duplicate service definitions

### High Priority Fixes
- **Background services** (`cmd/keldris-server/main.go`): Wired metrics scheduler, docker monitor, downtime, DB backup, and maintenance services into lifecycle with proper Start/Stop
- **Notification service** (`main.go`): Created `alertNotificationAdapter` wrapping `notifications.Service`, replacing `NoOpNotificationSender`
- **Handler registration** (`routes.go`): Registered Postgres, Immutability, Metadata, EmailVerification handlers
- **Agent updater** (`updater.go`): Validates downloaded binary exists and has non-zero size before removing old binary
- **Queue init** (`cmd/keldris-agent/main.go`): Backup queue now initialized in daemon mode
- **Auth headers** (`agent/queue.go`): Changed `X-API-Key` to `Authorization: Bearer` to match server middleware
- **HTTP 204** (`web/src/lib/api.ts`): `handleResponse()` returns `undefined` for 204 instead of trying to parse empty JSON
- **Shutdown sequence** (`main.go`): HTTP server drains first, then license validator stops
- **License annotation** (`main.go`): MIT→AGPL-3.0
- **Unverified entitlement** (`validator.go`): Warning log when no public key available

### Medium/Low Fixes
- **rows.Err()** (23 scan loops): Added checks after every `for rows.Next()` loop across 4 store files
- **Transactions** (`store.go`, `store_server_setup.go`): Wrapped `ActivateLicense`, `CreateFirstOrganization`, `UpdateUser`, `DeleteUser` in transactions
- **JSON injection** (`store_metadata.go`): Replaced `fmt.Sprintf` with `json.Marshal` for metadata search
- **LIKE escaping** (`store_activity.go`): Escape `%` and `_` in user search queries
- **URL encoding** (`license_manage.go`): `url.QueryEscape(email)` in trial check
- **Komodo hardening** (`komodo_integration.go`): Check `Enabled` flag, don't leak internal errors
- **SMTP/OIDC tests** (`system_settings.go`): Replaced TODO stubs with actual TCP+auth and OIDC discovery
- **XSS fix** (`Documentation.tsx`): HTML-escape markdown input before rendering
- **Accessibility** (`ConfirmationModal.tsx`): Added `role="dialog"`, `aria-modal`, escape key, backdrop click
- **LicenseFeature expansion** (`types.ts`): 5→28 features to match backend
- **Dead code removal**: Deleted `useFeature.ts`, `KomodoSettings.tsx`, `MaintenancePage.tsx`
- **Duplicate route** (`App.tsx`): Removed standalone `/license` route
- **Dead links** (`upgrade.ts`): Fixed `/organization/billing` and `/contact-sales` → `/organization/license`
- **Pagination** (`store_recovered.go`): Added `LIMIT 10000` to `GetAllBackups`
- **Makefile** : Removed duplicate `.PHONY` and `staticcheck` lines
- **redactArgs** (`restic.go`): Actually redacts `--repo`, `--password`, etc. instead of being a no-op
- **Interface fixes** (`metadata.go`): Changed `interface{}` returns to typed `*models.Agent`, `*models.Repository`, `*models.Schedule`

### Files Modified (45)
- `cmd/keldris-agent/main.go`, `cmd/keldris-server/main.go`
- `docker/Dockerfile.server`, `docker/docker-compose.images.yml`
- `go.mod`, `Makefile`
- `internal/agent/queue.go`, `internal/backup/restic.go`, `internal/updater/updater.go`
- `internal/api/handlers/` — `komodo_integration.go`, `license_manage.go`, `metadata.go`, `notification_rules.go`, `system_settings.go`
- `internal/api/routes.go`
- `internal/db/` — `store.go`, `store_activity.go`, `store_concurrency.go`, `store_metadata.go`, `store_recovered.go`, `store_recovered_full.go`, `store_server_setup.go`, `store_superuser.go`, `store_telemetry.go`, `store_usage_metrics.go`
- `internal/license/license.go`, `internal/license/validator.go`
- `internal/models/notification.go`
- `internal/notifications/` — `email.go`, `pagerduty.go`, `rules.go`, `service.go`, `teams.go`, `teams_test.go`
- `web/src/` — `App.tsx`, `components/ui/ConfirmationModal.tsx`, `hooks/usePlanLimits.ts`, `lib/api.ts`, `lib/types.ts`, `lib/upgrade.ts`, `lib/utils.ts`, `pages/Documentation.tsx`
- Deleted: `web/src/hooks/useFeature.ts`, `web/src/pages/KomodoSettings.tsx`, `web/src/pages/MaintenancePage.tsx`

---

## 2026-03-03 - Dark Mode Overhaul, Dry Run Constraint Fix, Remote Uninstall

### What
Comprehensive dark mode pass across the entire frontend (49 files), fixed the `agent_commands_type_check` constraint that was blocking dry runs, and added a remote uninstall command so agents can be uninstalled from the dashboard without SSH.

### How

**Dark mode** — Added `dark:` Tailwind variants to every component and page missing them:
- **UI primitives** (11 files): Badge, Button, DataTable, DropdownMenu, ErrorMessage, Label, LoadingSpinner, Pagination, ReadOnlyBlocker, Stepper, Tabs
- **Heavy pages** (6 files): Policies, SLA, Reports, AgentGroups, Documentation, DockerBackup
- **Light pages** (9 files): Backups, DockerRegistries, RateLimits, DRRunbooks, RepositoryStatsDetail, StorageStats, Changelog, SLATracking, AgentDetails
- **Feature components** (18 files): ExportImportModal, PatternLibraryModal, ContainerHooksEditor, ImportRepositoryWizard, KeyboardShortcutsSettings, GlobalSearchBar, MaintenanceCountdown, NetworkMountSelector, BackupScriptsEditor, DockerRestoreWizard, ImportAgentsWizard, ShortcutHelpModal, AgentLogViewer, AirGapIndicator, MetadataSchemaManager, RegionMapIndicator, ReplicationStatusCard, WhatsNewModal

**Dry run constraint fix** — Migration 107 drops and recreates `agent_commands_type_check` to include `dry_run` (was missing from migration 106 which only added `update_restic`).

**Remote uninstall** — Full-stack implementation of an `uninstall` command type:
- Go model: `CommandTypeUninstall` constant + `Purge bool` in `CommandPayload`
- Agent client: `Purge` field in agent-side `CommandPayload`
- Server handler: Added `uninstall` to `CreateCommandRequest` binding validation
- Agent executor: New `"uninstall"` case reports completed then calls `runUninstall(purge, force=true)`
- DB migration: `uninstall` included in the type constraint
- Frontend types: `'uninstall'` in `CommandType` union, `purge` in `CommandPayload`
- UI: Replaced static CLI instructions in AgentDetails with two buttons — "Uninstall" (service+binary) and "Uninstall & Purge Data" (everything) — with confirmation dialogs, danger-zone styling, and dark mode support

### Files Modified
- `internal/models/agent_command.go` — `CommandTypeUninstall`, `Purge` payload field
- `internal/agent/client.go` — `Purge` in agent `CommandPayload`
- `internal/api/handlers/agent_commands.go` — `uninstall` in validation binding
- `internal/db/migrations/107_add_dry_run_command_type.sql` — new migration
- `cmd/keldris-agent/main.go` — `"uninstall"` case in command dispatch
- `web/src/lib/types.ts` — `'uninstall'` type, `purge` payload
- `web/src/pages/AgentDetails.tsx` — remote uninstall UI, dark mode
- 44 additional frontend files — dark mode variants

---

## 2026-03-03 - Agent-Side Dry Run, Config Path Fix, and UI Polish

### What
Implemented real dry run execution via the agent, fixed `sudo keldris-agent status` showing "Not configured", fixed dark mode contrast on cron badges, and improved agent page responsiveness.

### How
- **Dry run command dispatch**: Server DryRun handler now creates a `dry_run` command (like `backup_now`) instead of returning placeholder data. Agent polls for it, runs `restic backup --dry-run --json`, and reports results back.
- **Agent executor**: Added `dry_run` case in `executeCommand` switch and `executeDryRun()` function that fetches the schedule from the server, builds restic config, and runs `restic.DryRun()`.
- **Frontend async flow**: Both Schedules and Onboarding pages now dispatch the command, then poll `useCommandResult()` (2s interval) until terminal status. Results are converted from `DryRunCommandResult` into the existing `DryRunResponse` shape for display.
- **Config path fallback**: `DefaultConfigDir()` now checks `~/.keldris/config.yml` then `/etc/keldris/config.yml` as fallback, fixing the case where `sudo` doesn't carry the `KELDRIS_CONFIG_DIR` env var set in the systemd unit.
- **Cron dark mode**: Added `dark:bg-gray-700 text-gray-800 dark:text-gray-200` to cron expression badges in Schedules page.
- **Agent page responsiveness**: Lowered `useAgents()` polling from 30s to 10s and added `placeholderData: keepPreviousData` to prevent empty-state flash on navigation.

### Files Modified
- `cmd/keldris-agent/main.go` — `executeDryRun()` function and `dry_run` case
- `internal/models/agent_command.go` — `CommandTypeDryRun`, `DryRunCommandResult`
- `internal/agent/client.go` — `DryRunResultDetail` in `CommandResultDetail`
- `internal/api/handlers/schedules.go` — DryRun handler dispatches command, removed unused `formatPaths`
- `internal/config/agent.go` — `/etc/keldris` fallback in `DefaultConfigDir()`
- `web/src/lib/types.ts` — `DryRunCommandResponse`, `DryRunCommandResult`
- `web/src/lib/api.ts` — updated `dryRun` return type
- `web/src/hooks/useSchedules.ts` — `useCommandResult` hook
- `web/src/hooks/useAgents.ts` — lower polling, `keepPreviousData`
- `web/src/pages/Schedules.tsx` — async dry run flow, cron dark mode fix
- `web/src/pages/Onboarding.tsx` — async dry run flow

---

## 2026-03-03 - Add `uninstall` Subcommand to Agent Binary

### What
Users who installed the agent via a curl-pipe-to-bash one-liner don't have the install/uninstall scripts on disk afterward. Added `keldris-agent uninstall [--purge] [--force]` so users can cleanly remove the agent from any platform without needing the scripts.

### How
- **Command registration**: Added `newUninstallCmd()` to root command and to the auto-update skip list
- **Privilege checks**: Linux requires root (os.Getuid), Windows checks admin via `net session`, macOS warns if binary dir isn't writable
- **Service removal**: Linux (systemctl stop/disable/daemon-reload/reset-failed + unit file removal), macOS (launchctl unload + plist removal), Windows (sc.exe stop/delete with sleep)
- **Binary self-deletion**: Unix uses `os.Remove`; Windows renames to `.removing` since you can't delete a running exe
- **Purge mode** removes: managed restic, system restic (skips Homebrew symlinks on macOS), FUSE mounts (fusermount/umount), config dir (~/.keldris), platform config (/etc/keldris on Linux, %ProgramData%\Keldris on Windows), log files (/var/log/keldris* on Linux), temp files (keldris-*, restic-compressed-*, restic-download-*), PATH entries and KELDRIS_CONFIG_DIR env var (Windows)
- All platform-specific logic uses `runtime.GOOS` switches and `exec.Command` — no build tags or platform-specific imports needed

### Files Modified
- `cmd/keldris-agent/main.go` — newUninstallCmd, runUninstall, and platform-specific helpers

---

## 2026-03-03 - Post-Deploy UI & Agent Fixes

### What
After deploying the backup execution pipeline fix, testing revealed several UX issues: agent list didn't auto-refresh, no way to edit schedules after creation, no feedback after "Run Now", and a type mismatch on the run response.

### How
- **Agent auto-refresh**: Added `refetchInterval: 30_000` to `useAgents()` so the agent list polls every 30 seconds for newly registered agents
- **RunScheduleResponse type**: Changed `backup_id` to `command_id` to match the server's actual response after the pipeline fix
- **Schedule edit modal**: Converted `CreateScheduleModal` into a dual-purpose create/edit modal — accepts optional `editSchedule` prop, pre-fills all form fields (name, agent, repos, cron, paths, retention, bandwidth, excludes, priority), calls `useUpdateSchedule()` in edit mode. Added "Edit" button as first action in each schedule row.
- **Run Now feedback**: Added `onSuccess` callback to `runSchedule.mutate()` that shows an alert explaining the command was sent and the agent will pick it up within 60 seconds on next heartbeat

### Files Modified
- `web/src/hooks/useAgents.ts` — refetchInterval
- `web/src/lib/types.ts` — RunScheduleResponse type fix
- `web/src/pages/Schedules.tsx` — edit modal, Edit button, Run Now feedback

---

## 2026-03-03 - Fix Backup Execution Pipeline

### What
Backups triggered from the UI were broken: `Run()` created a Backup record but never dispatched a command to the agent, so backups sat in "running" forever. Also, the server's repository integrity check (`restic check`) failed because restic wasn't installed in the server Docker image.

### How
- **`schedules.go:Run()`**: Replaced premature `NewBackup`+`CreateBackup` with `NewAgentCommand(backup_now)` dispatch. The agent already handles `backup_now` commands and creates the backup record via `ReportBackup` when it actually runs — so the old approach created duplicates and orphaned records. Set command timeout to 2 hours (was 5 min default).
- **`Dockerfile.server`**: Added `restic` to Alpine packages so `VerificationScheduler` can run `restic check`.
- **`install-linux.sh`**: Added explicit `PATH` to systemd service so the agent can find `/usr/local/bin/restic` on systems with restricted default PATH.
- **`store_recovered.go`**: Added `FailStaleBackups()` — marks backups stuck in "running" for >24h as "failed" with a timeout message.
- **`main.go`**: Calls `FailStaleBackups()` on server startup after migrations.
- **DryRun**: Updated message to explain it requires agent-side execution (out of scope for now).

### Files Modified
- `internal/api/handlers/schedules.go` — core fix: command dispatch + interface update
- `internal/api/handlers/schedules_test.go` — mock update for new interface method
- `internal/db/store_recovered.go` — `FailStaleBackups()`
- `cmd/keldris-server/main.go` — stale backup cleanup on startup
- `docker/Dockerfile.server` — add restic
- `scripts/install-linux.sh` — PATH in systemd service

---

## 2026-03-03 - Quick Install with Auto-Registration

### What
When a registration code exists, the quick install command now automatically includes `KELDRIS_SERVER`, `KELDRIS_CODE`, and `KELDRIS_ORG_ID` env vars so agents self-register on install. Also replaced the `YOUR_SERVER_URL` placeholder in the registration code modal with the actual server origin.

### How
- **`constants.ts`**: Added `getInstallCommand(platform, opts?)` builder that injects env vars into install commands when registration context is provided (uses `sudo -E` on Linux, `$env:` on Windows)
- **`AgentDownloads.tsx`**: Accepts optional `registrationCode` and `orgId` props; computes server URL from `window.location.origin`; shows "(auto-registers with server)" hint when active
- **`Agents.tsx`**: Extended `GenerateCodeModal.onSuccess` to pass `org_id` from API response; tracks `orgId` in `newCode` state; computes `activeCode`/`activeOrgId` with fallback to first non-expired pending registration via `useMe().current_org_id`; replaced `YOUR_SERVER_URL` with `window.location.origin` in `RegistrationCodeModal`

### Files Modified
- `web/src/lib/constants.ts`
- `web/src/components/features/AgentDownloads.tsx`
- `web/src/pages/Agents.tsx`

---

## 2026-03-02 - Agent Security Hardening

### What
Comprehensive security audit and fixes for the agent command dispatch and binary download infrastructure.

### How
- **ReportBackup authorization**: Added schedule ownership validation (schedule.AgentID must match calling agent) and repository org validation (repo.OrgID must match agent.OrgID) before creating backup records
- **HTTPS enforcement**: `register --server` and `config set-server` now require `https://` by default; added `--insecure` flag as escape hatch for development
- **Restic download checksums**: Downloads SHA256SUMS from the release, verifies compressed binary hash before decompressing; validates all download URLs are HTTPS pointing to github.com
- **Proper HTTP client**: Replaced `http.DefaultClient` with `httpclient.NewSimple()` for TLS handshake timeout
- **Command result size limits**: Output/Error truncated to 64KB, Diagnostics map capped at 100 keys
- **Health metrics bounds**: CPU/Memory/Disk usage clamped to 0-100, byte values and uptime clamped to non-negative

### Files Modified
- `cmd/keldris-agent/main.go` — HTTPS enforcement, checksum verification, proper HTTP client
- `internal/api/handlers/agent_api.go` — ownership checks, size limits, metrics clamping

### Next
Deploy and verify agent update + restic update flows end-to-end

---

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

---

## 2026-03-02 - Inline Schedule & Verify Steps in Onboarding Wizard

### What
The onboarding ScheduleStep linked to `/schedules` and the VerifyStep told users to manually navigate to Schedules and Backups pages. Both now work entirely inline within the wizard.

### Changes

**`web/src/pages/Onboarding.tsx`:**

**ScheduleStep** — replaced link stub with inline creation form:
- Name field (default: "Daily Backup")
- Agent dropdown (auto-selects if only one agent exists)
- Repository multi-selector using `MultiRepoSelector` component
- Cron expression input with preset buttons (Daily 2AM, Every 6h, Weekly Sun)
- Paths textarea (default: `/home`)
- Retention policy with "Customize" toggle (defaults: 5 last, 7 daily, 4 weekly, 6 monthly)
- Green success banner when schedules already exist or after creation
- "Skip for now" always available

**VerifyStep** — replaced manual instructions with inline run & monitor:
- Lists all existing schedules with "Run Now" button on each
- Polls backups every 5s after triggering via `queryClient.invalidateQueries`
- Live status display per schedule: running (spinner), completed (green + size/files), failed (red + error + Try Again), canceled (yellow)
- "Complete Setup" enabled only after at least one backup completes
- "Skip for now" always available

**Parent switch/case** — updated `case 'schedule'` and `case 'verify'` to pass `onSkip={() => handleCompleteStep(...)}`.

**Imports added:**
- `useCreateSchedule`, `useRunSchedule` from `useSchedules`
- `useBackups` from `useBackups`
- `MultiRepoSelector` from `MultiRepoSelector`
- `Backup`, `ScheduleRepositoryRequest` types

### No backend changes
All API endpoints (`POST /api/v1/schedules`, `POST /api/v1/schedules/:id/run`, `GET /api/v1/backups`) already existed.

### Result
- `npx tsc --noEmit`: clean
- `npx biome check`: clean

---

## 2026-03-02 - Agent Auto-Update, Restic Management & Command Dispatch

### What
Wired up the command dispatch infrastructure so admins can push agent updates, restic updates, and other commands from the dashboard. The agent daemon now polls for commands after each heartbeat, executes them, and reports results. Restic is self-managed: auto-downloaded on startup if not found in PATH.

### Changes (14 files)

**DB Migration:**
- `106_agent_version.sql` — adds `agent_version` column to `agents` table, adds `update_restic` to command type constraint

**Server Models:**
- `internal/models/agent.go` — `AgentVersion *string` field on Agent struct
- `internal/models/agent_command.go` — `CommandTypeUpdateRestic` constant, `TargetResticVersion` on CommandPayload
- `pkg/models/agent.go` — `AgentVersion` on HeartbeatRequest

**Server DB Queries:**
- `internal/db/store.go` — `agent_version` added to SELECT/Scan in GetAgentsByOrgID, GetAgentByID, GetAgentByAPIKeyHash; added to SET in UpdateAgent
- `internal/db/store_concurrency.go` — `agent_version` in GetAgentByIDWithConcurrency
- `internal/db/store_recovered_full.go` — `agent_version` in GetAgentGroupMembers

**Server Handlers:**
- `internal/api/handlers/agent_api.go` — stores AgentVersion from heartbeat
- `internal/api/handlers/agent_commands.go` — accepts `update_restic` in validation binding

**Agent Client:**
- `internal/agent/client.go` — `AgentVersion` on HeartbeatRequest; new types/methods: `GetCommands`, `AcknowledgeCommand`, `ReportCommandResult`

**Agent Daemon:**
- `cmd/keldris-agent/main.go` — major additions:
  - Version sent in every heartbeat
  - `pollAndExecuteCommands` called as goroutine after each heartbeat
  - Command dispatcher with executors: `update` (agent binary via updater pkg), `update_restic` (download from GitHub releases), `backup_now`, `restart` (syscall.Exec), `diagnostics`
  - `resolveResticBinary` on startup: PATH → managed dir (`$CONFIG_DIR/bin/restic`) → auto-download from GitHub releases (bzip2 decompression)
  - `sync.Mutex` concurrency guard prevents duplicate command execution

**Frontend:**
- `web/src/lib/types.ts` — `agent_version` on Agent, `update_restic` on CommandType, `target_restic_version` on CommandPayload
- `web/src/pages/AgentDetails.tsx` — "Update Agent" and "Update Restic" buttons, agent version display in header, `update_restic` label in command type labels and confirmation map
- `web/src/pages/Agents.tsx` — "Version" column in agents table (header + row)

### Result
- `go build ./...`: clean
- `npx tsc --noEmit`: clean
- `npx biome check`: clean

---

## 2026-03-14 — Code Audit: Security, Accessibility, UI Polish, Tests

### What
Full audit of both Keldris and license-server repos. Fixed security vulnerabilities, accessibility gaps, UI inconsistencies, and expanded test coverage. Executed in 4 parallel batches using 11 subagents.

### Phase 1: Critical Security Fixes

**XSS in Documentation.tsx (Keldris)**
- Link regex injected URLs directly into `href` — `javascript:alert(1)` bypassed `escapeHtml()`
- Installed DOMPurify, wrapped HTML output in `DOMPurify.sanitize()` before `dangerouslySetInnerHTML`
- 15 regression tests (javascript:, data:, vbscript: URIs, img onerror, script tags, event handlers, normal rendering)

**org_id in GetAgentLogs (Keldris)**
- `store_recovered.go` filtered by `agent_id` only — `agent_logs` table has `org_id` but it wasn't in WHERE clause
- Added `orgID uuid.UUID` param, `AND org_id = $2` to query, updated interface + handler + mock

**Activation Race Condition (License Server)**
- Count-then-insert without transaction — two concurrent requests both pass limit check
- New `CreateActivationWithLimitCheck` method: `SELECT FOR UPDATE` on license row, count + insert in single tx
- Sentinel error `ErrMaxActivationsReached`, concurrent Go test (10 goroutines, verify exactly N succeed)

**Silent Audit Log Errors (License Server)**
- `_ = h.store.CreateValidationLog(...)` silently dropped audit trail errors → now logged via `h.logger.Error()`
- `json.Marshal` errors on `limitKeys`/`limits` → now return 400 instead of nil
- Added explicit comment on intentionally unchecked optional revoke reason

### Phase 2: Accessibility & TODO Completions

**Modal Focus Trap (Keldris)**
- Removed `role="button"` and `tabIndex={-1}` from overlay backdrop
- Added focus trap: save previous focus on mount, trap Tab within dialog, restore on unmount
- Added `aria-labelledby` via `useId()` + `ModalTitleIdContext` linking dialog to ModalHeader h3
- 8 new tests (focus movement, tab trapping, focus restoration, aria attributes)

**Critical TODOs Completed (Keldris)**
- `repositories.go:462` — Stale TODO removed (encrypted config already implemented above it)
- `superuser.go:332` — Created `SuperuserAuditLog` synchronously before `StartImpersonation`, passes real audit log ID
- `backup_queue.go:350` — New `GetQueuedBackupsCountByAgent` DB method replaces hardcoded `0`

### Phase 3: UI Polish

**Dark Mode Gaps**
- `Pagination.tsx` — active page button + ellipsis
- `Tags.tsx`, `DRTests.tsx`, `DRRunbooks.tsx`, `Maintenance.tsx`, `LicenseManagement.tsx`, `PasswordRequirements.tsx` — loading skeletons
- `OrganizationSettings.tsx` — role badge, cancel buttons, danger zone, delete modal

**Form Double-Submit Prevention**
- Added `disabled:cursor-not-allowed` to 14 submit buttons across SystemSettings, OrganizationSettings, Schedules, Agents

**Icon Button Accessibility**
- Added `aria-label` to 26 icon-only buttons across 20 files (theme toggle, close, search, download, favorites, etc.)

**Component Extraction**
- Created `components/ui/LoadingRow.tsx` — configurable table skeleton (width, pill, button, align, render, barClassName)
- Created `components/ui/LoadingCard.tsx` — 7 built-in variants (stat, stat-sm, alert, template, repo, sla, health)
- Removed 27 inline LoadingRow/LoadingCard definitions across pages
- Consolidated 11 status color functions into `lib/utils.ts` (restore, command, DR test/runbook, notification, health, lifecycle, docker log, user status, role badge)

### Phase 4: Test Coverage (+184 new tests)

| Page | Tests | Key areas covered |
|------|-------|-------------------|
| License.tsx | 34 | Loading, error, tier badges, limits, trial states, forms, history |
| Activity.tsx | 17 | Feed rendering, category filtering, WebSocket indicators, deduplication |
| LifecyclePolicies.tsx | 36 | CRUD, status toggle, delete confirmation, dry run, create form |
| Webhooks.tsx | 33 | CRUD, event types, delivery log modal, retry, signature verification |
| SystemHealth.tsx | 49 | All sections, warning/critical states, auto-refresh, historical data |
| Documentation.tsx | 15 | XSS vectors + normal markdown rendering (Phase 1) |

**Total vitest: 1635 (up from ~1446)**

### Verification
- `go vet ./...`: clean (both repos)
- `staticcheck ./...`: clean (both repos)
- `go test -race ./...`: pass (both repos)
- `npx vitest run`: 1635 pass (112 files)
- `npx @biomejs/biome check .`: clean (383 files)
- `npx tsc --noEmit`: clean

### Commits
- Keldris `045ede98` — Fix security vulnerabilities, improve accessibility, polish UI, expand tests (74 files, +2942/-1159)
- License Server `35a6ae3` — Fix activation race condition, audit log errors, and input validation (8 files, +257/-42)
- License Server `e89b5f0` — Add vitest unit tests for all 6 frontend pages (9 files, +2968)

### License Server Frontend Tests (added separately)
Initially missed in Phase 4 — corrected after review. Added 131 new vitest tests across 6 pages:

| Page | Tests | Key areas |
|------|-------|-----------|
| Dashboard | 6 | Stats display, loading, fallback values |
| Products | 24 | CRUD, billing units, form validation |
| Customers | 17 | CRUD, search, delete confirmation |
| Licenses | 32 | CRUD, key display, status/tier badges, trial filter |
| Instances | 24 | Status badges, anomalies, kill action, filtering |
| PricingPlans | 28 | CRUD, pricing display, active toggle, seed defaults |

**License server total vitest: 157 (up from 26)**
