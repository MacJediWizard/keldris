# Changelog

All notable changes to Keldris will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.6.0] - 2026-03-02

### Added
- Auto-register agent during install across Linux, macOS, and Windows platforms
- Onboarding agent step generates install commands with registration baked in
- License management UI with activation, trial start, and tier display
- Ed25519 license validation with phone-home heartbeat system
- 3-layer feature gating (DB tier, entitlement nonce, refresh token)
- OIDC configuration in onboarding wizard with test connection button
- SMTP configuration in onboarding wizard with test connection
- Inline repository creation form in onboarding wizard
- First-time server setup wizard with superuser creation
- Login page with local auth and OIDC support
- Multi-channel notifications (Slack, Teams, Discord, PagerDuty, Webhooks)
- Dark mode support across all pages
- Docker container and volume backup support
- Proxmox VM backup integration
- Direct PostgreSQL and MySQL/MariaDB backup support
- Agent registration with 2FA verification codes
- Air-gap mode for fully offline Enterprise operation
- White-label branding for Enterprise organizations
- SLA tracking and compliance system
- Ransomware detection system
- Snapshot immutability for compliance
- Legal hold feature for snapshots
- Resumable backup checkpoints for interrupted backups
- FUSE snapshot mount for browsing backups
- Agent command queue for push commands from UI
- Geo-replication to secondary regions
- Storage tiering with hot/warm/cold/archive lifecycle
- Backup concurrency limits with queueing system
- Backup calendar UI with month view
- File diff viewer and search across snapshots
- System health overview for admins
- Server and agent logs viewer
- Diagnostic support bundle generation
- Agent debug mode for verbose logging
- Configuration export/import for migration
- Bulk selection and actions for tables
- Keyboard shortcuts for power users
- Global search with autocomplete and filters
- Real-time activity feed
- User favorites (star) feature
- Recently viewed items tracker
- Contextual help tooltips throughout UI
- Comprehensive UI component library
- Comprehensive test coverage across all packages

### Changed
- Onboarding agent step now generates platform-specific install commands with auto-registration
- Install scripts download from GitHub Releases instead of releases.keldris.io
- Installer log functions redirect to stderr to avoid polluting captures
- Organization auto-renamed from license company name during onboarding
- Immediate heartbeat sent on license activation
- Nonce-based CSP replaces unsafe-inline
- Code-split frontend for reduced bundle size

### Fixed
- macOS installer duplicate DOWNLOAD_BASE_URL, VERSION variable, and download_url
- Windows installer duplicate $DownloadUrl param and $Version parameter
- Linux installer log functions polluting stdout captures
- OIDC onboarding step 402 error and retry storm
- SetLicenseKey context cancellation during license server calls
- Duplicate Set-Cookie in OIDC callback causing 502 behind reverse proxy
- 401 redirect loop on setup and login pages
- Rate limit exceeded during setup wizard
- DNS rebinding TOCTOU in SSRF validation
- Gin router panics from conflicting route parameters
- Multiple migration version conflicts resolved

## [0.5.0] - 2026-01-21

### Added
- Repository import feature for existing Restic backups
- Agent registration with 2FA verification codes
- OIDC group synchronization to organization roles
- Cloud storage cost estimation feature
- Changelog UI with version history

### Changed
- Updated logo to use CDN image

## [0.4.0] - 2026-01-15

### Added
- SSO group mappings for automatic role assignment
- Maintenance window scheduling and notifications
- DR runbook management and testing
- Storage cost forecasting

### Fixed
- Agent health metrics collection timing issues
- Backup retention policy edge cases

## [0.3.0] - 2026-01-08

### Added
- File history browser for point-in-time recovery
- Snapshot comparison and diff viewer
- Policy-based backup configuration
- Agent groups for fleet management

### Changed
- Improved dashboard metrics performance
- Enhanced backup script execution logging

## [0.2.0] - 2025-12-20

### Added
- Multi-repository backup targets with priority ordering
- Replication status tracking between repositories
- Exclude patterns library with built-in templates
- Tag-based organization for backups

### Fixed
- Schedule timezone handling for international users
- Repository connection testing reliability

## [0.1.0] - 2025-12-01

### Added
- Initial release of Keldris backup management platform
- Agent-based backup orchestration with Restic
- Multiple repository type support (Local, S3, B2, SFTP, REST, Dropbox)
- Schedule-based backup automation with cron expressions
- Retention policy management
- OIDC authentication integration
- Multi-organization support with role-based access
- Real-time backup monitoring and alerts
- Audit logging for compliance

[Unreleased]: https://github.com/MacJediWizard/keldris/compare/v0.6.0...HEAD
[0.6.0]: https://github.com/MacJediWizard/keldris/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/MacJediWizard/keldris/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/MacJediWizard/keldris/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/MacJediWizard/keldris/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/MacJediWizard/keldris/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/MacJediWizard/keldris/releases/tag/v0.1.0
