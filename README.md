# Keldris

![Status](https://img.shields.io/badge/status-in%20development-yellow)
![License](https://img.shields.io/badge/license-AGPL%20v3-blue)

**Keeper of your data** — Enterprise-grade, self-hosted backup solution with OIDC authentication, Restic-powered encryption, and multi-cloud storage support.

> ⚠️ **Coming Soon** — Keldris is currently in active development. Features listed below are being implemented and may not be available yet.

---

## Overview

Keldris is a comprehensive backup solution designed for teams and organizations that need:

- 🔐 **Enterprise Authentication** — OIDC/SSO integration (Authentik-first)
- 🖥️ **Agent-Based Architecture** — Lightweight Go agents for Linux, macOS, and Windows
- 📦 **Restic-Powered** — Encrypted, deduplicated, verifiable backups
- ☁️ **Multi-Cloud Storage** — S3, B2, Dropbox, local, and more
- 🏢 **Multi-Tenant** — Organizations, roles, and permissions
- 🐳 **Docker-Native** — Full Docker and Docker Compose backup support

---

## Features

### Core Backup
- [x] Restic-powered encrypted backups
- [x] Scheduled backups with cron expressions
- [x] Multiple storage backends (Local, S3, B2, Dropbox, SFTP)
- [x] Retention policy automation
- [x] Backup verification and integrity checks
- [x] Bandwidth scheduling and throttling
- [x] Pre/post backup scripts
- [x] Compression settings
- [x] Exclude patterns library

### Multi-Repository Support
- [x] Backup to multiple repositories with priority ordering
- [x] Automatic failover (retry 3x, then try secondary)
- [x] True replication via `restic copy`
- [x] Sync status tracking between repos

### Restore
- [x] Full restore workflow UI
- [x] Browse snapshots by date
- [x] File/folder browser within snapshots
- [x] Partial restore (single files/folders)
- [x] Cross-agent restore
- [x] Snapshot comparison (diff between snapshots)
- [x] File version history browser
- [x] Restore dry-run preview

### Docker Support
- [x] Docker container and volume backup
- [x] Docker Compose full stack backup
- [x] Docker image backup
- [x] Docker network configuration backup
- [x] Docker secrets and configs backup
- [x] Docker Swarm support
- [x] Docker health monitoring
- [x] Docker logs backup
- [x] Pre/post backup hooks inside containers
- [x] Label-based backup configuration
- [x] Private registry credential management
- [x] Komodo integration

### Agent Management
- [x] Lightweight Go agents (Linux, macOS, Windows)
- [x] Agent self-update mechanism
- [x] Platform installers (systemd, launchd, Windows Service)
- [x] Agent groups (by environment/purpose)
- [x] Agent health monitoring
- [x] Detailed agent stats page
- [x] Remote command execution
- [x] Network drive support (NFS/SMB/CIFS)

### Authentication & Security
- [x] OIDC authentication (Authentik-first)
- [x] Multi-organization support
- [x] Role-based access control (Owner, Admin, Member, Readonly)
- [x] SSO group sync
- [x] Two-factor agent registration
- [x] API key management with rotation
- [x] Session management
- [x] IP allowlist
- [x] Audit logging
- [x] Backup encryption key management
- [x] Immutable backups (object lock)
- [x] Legal hold for compliance

### Monitoring & Alerts
- [x] Gatus-compatible health endpoints
- [x] Prometheus metrics endpoint
- [x] Agent heartbeat monitoring
- [x] Backup SLA tracking
- [x] Storage usage tracking
- [x] Alert rules engine
- [x] Multi-channel notifications (Email, Slack, Teams, Discord, PagerDuty, Webhooks)

### Reporting & Analytics
- [x] Metrics dashboard
- [x] Backup success/failure rates
- [x] Storage growth charts
- [x] Deduplication stats
- [x] Cost estimation for cloud storage
- [x] Scheduled email reports (daily/weekly/monthly)
- [x] Export reports as CSV/JSON

### Administration
- [x] Superuser/global admin
- [x] Organization management
- [x] Usage quotas per org
- [x] System settings page
- [x] Maintenance windows
- [x] Backup policies (templates)
- [x] Bulk operations
- [x] Import existing Restic repos
- [x] License management

### User Experience
- [x] Dark mode
- [x] Localization (multi-language)
- [x] Onboarding wizard
- [x] Keyboard shortcuts
- [x] Global search
- [x] Tags and filtering
- [x] Saved filters
- [x] Activity feed
- [x] Favorites and recent items
- [x] Contextual help tooltips
- [x] Backup comments/notes

### Developer & DevOps
- [x] REST API with OpenAPI docs
- [x] Terraform provider
- [x] Ansible role
- [x] Docker deployment
- [x] CI/CD with GitHub Actions
- [x] API rate limiting
- [x] Debug mode and support bundles

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      keldris-server                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │  React UI   │  │   Go API    │  │    PostgreSQL       │  │
│  │  (Vite/TS)  │  │   (Gin)     │  │    (Multi-tenant)   │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
│                           │                                 │
│              ┌────────────┴────────────┐                    │
│              │    Restic + Storage     │                    │
│              │  (S3/B2/Local/SFTP)     │                    │
│              └─────────────────────────┘                    │
└─────────────────────────────────────────────────────────────┘
                            │ HTTPS
             ┌──────────────┼──────────────┐
             │              │              │
      ┌──────┴─────┐ ┌──────┴─────┐ ┌──────┴─────┐
      │  keldris   │ │  keldris   │ │  keldris   │
      │   agent    │ │   agent    │ │   agent    │
      │  (Linux)   │ │  (macOS)   │ │ (Windows)  │
      └────────────┘ └────────────┘ └────────────┘
```

---

## Tech Stack

| Component | Technology |
|-----------|------------|
| Server | Go 1.24+, Gin |
| Database | PostgreSQL 15+ |
| Agent | Go 1.24+, Cobra |
| Frontend | React 18, TypeScript, Vite, Tailwind CSS |
| Backup Engine | Restic |
| Authentication | OIDC (coreos/go-oidc) |
| Notifications | SMTP, Slack, Webhooks |

---

## Quick Start

> 🚧 **Coming Soon** — Installation instructions will be available when Keldris reaches beta.

### Prerequisites

- Docker and Docker Compose
- PostgreSQL 15+
- OIDC Provider (Authentik, Keycloak, etc.)

### Docker Deployment

```bash
# Clone the repository
git clone https://github.com/MacJediWizard/keldris.git
cd keldris

# Configure environment
cp docker/.env.example docker/.env
# Edit docker/.env with your settings

# Start services
docker compose -f docker/docker-compose.yml up -d
```

### Agent Installation

```bash
# Linux (systemd)
curl -fsSL https://keldris.com/install.sh | bash

# macOS (launchd)
curl -fsSL https://keldris.com/install-mac.sh | bash

# Windows (PowerShell as Admin)
irm https://keldris.com/install.ps1 | iex
```

### Register Agent

```bash
keldris register --server https://your-keldris-server.com
```

---

## Development

```bash
# Install dependencies
make deps

# Run development servers
make dev

# Run tests
make test

# Run linters
make lint

# Build binaries
make build
```

---

## Documentation

> 📚 **Coming Soon** — Full documentation will be available at [docs.keldris.com](https://docs.keldris.com)

- Installation Guide
- OIDC Setup (Authentik, Keycloak)
- Agent Deployment
- Storage Backend Configuration
- API Reference
- Backup Strategies
- Disaster Recovery

---

## Roadmap

### Phase 1 — Core (In Progress)
- [x] Database schema and migrations
- [x] OIDC authentication
- [x] Basic agent registration
- [x] Restic integration
- [x] Schedule management
- [x] Basic web UI

### Phase 2 — Enterprise Features (In Progress)
- [x] Multi-org and RBAC
- [x] Email notifications
- [x] Storage backends
- [x] Retention automation
- [x] Agent installers

### Phase 3 — Advanced Features (In Progress)
- [ ] Full Docker support
- [ ] Monitoring and alerts
- [ ] Scheduled reports
- [ ] Metrics dashboard
- [ ] Dark mode and localization

### Phase 4 — Polish (Planned)
- [ ] Onboarding wizard
- [ ] API documentation
- [ ] Terraform provider
- [ ] Public beta release

---

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

```bash
# Fork the repository
# Create a feature branch
git checkout -b feature/your-feature

# Make changes and test
make lint && make test

# Submit a pull request
```

---

## License

Keldris is licensed under the [GNU Affero General Public License v3.0](LICENSE).

---

## About

**Keldris** — *Keeper of your data*

Built by [MacJediWizard Consulting, Inc.](https://macjediwizard.com)

Powered by **NeuroHolocron**

---

## Links

- 🌐 Website: [keldris.com](https://keldris.com) *(coming soon)*
- 📚 Documentation: [docs.keldris.com](https://docs.keldris.com) *(coming soon)*
- 🐛 Issues: [GitHub Issues](https://github.com/MacJediWizard/keldris/issues)
- 💬 Discussions: [GitHub Discussions](https://github.com/MacJediWizard/keldris/discussions)
