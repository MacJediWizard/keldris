# Changelog

All notable changes to Keldris will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/MacJediWizard/keldris/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/MacJediWizard/keldris/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/MacJediWizard/keldris/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/MacJediWizard/keldris/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/MacJediWizard/keldris/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/MacJediWizard/keldris/releases/tag/v0.1.0
