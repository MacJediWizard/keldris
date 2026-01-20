# Keldris

<p align="center">
  <strong>Keeper of your data</strong><br>
  Self-hosted backup solution with OIDC authentication, Restic engine, and enterprise features
</p>

<p align="center">
  <a href="#features">Features</a> •
  <a href="#tech-stack">Tech Stack</a> •
  <a href="#installation">Installation</a> •
  <a href="#documentation">Documentation</a> •
  <a href="#license">License</a>
</p>

---

> ⚠️ **UNDER ACTIVE DEVELOPMENT** - Keldris is currently being built. Features listed below are planned or in progress. Star this repo to follow along!

---

## What is Keldris?

Keldris is a self-hosted, agent-based backup solution designed for homelabs, small businesses, and enterprises. It provides centralized backup management with modern authentication, encrypted storage, and comprehensive monitoring.

### Why Keldris?

- **OIDC/SSO First** - Authentik, Keycloak, or any OIDC provider
- **Agent-Based** - Lightweight Go agents on each machine
- **Restic-Powered** - Battle-tested encryption and deduplication
- **Multi-Tenant** - Organizations, roles, and permissions built-in
- **Cloud & Local** - S3, B2, Dropbox, local storage, and more

---

## Features

### Core Backup Features
| Feature | Status |
|---------|--------|
| Restic-powered encrypted backups | ✅ Done |
| Scheduled backups (cron) | ✅ Done |
| Multiple storage backends (Local, S3, B2, Dropbox) | 🚧 In Progress |
| Retention policy automation | 🚧 In Progress |
| Backup verification & integrity checks | 🚧 In Progress |
| Bandwidth scheduling & limits | 🚧 In Progress |
| Compression settings | 🚧 In Progress |
| Pre/post backup scripts | 🚧 In Progress |
| Exclude patterns library | 🚧 In Progress |
| Multi-repository backup (primary + secondaries) | 🚧 In Progress |
| Backup policies & templates | 🚧 In Progress |

### Restore Features
| Feature | Status |
|---------|--------|
| Full restore UI | 🚧 In Progress |
| File/folder browser in snapshots | 🚧 In Progress |
| Partial restore (single files) | 🚧 In Progress |
| Cross-agent restore | 🚧 In Progress |
| Snapshot comparison (diff) | 🚧 In Progress |
| File version history | 🚧 In Progress |
| Restore dry-run | 📋 Planned |
| Snapshot mount (FUSE) | 📋 Planned |

### Agent Features
| Feature | Status |
|---------|--------|
| Go agent (Linux, macOS, Windows) | ✅ Done |
| Agent health monitoring | 🚧 In Progress |
| Agent self-update | 🚧 In Progress |
| Agent groups | 🚧 In Progress |
| Agent details page | 🚧 In Progress |
| Platform installers (systemd, launchd, Windows Service) | 🚧 In Progress |
| Remote commands from UI | 📋 Planned |
| Network drive support (NFS/SMB/CIFS) | 🚧 In Progress |

### Docker & Container Support
| Feature | Status |
|---------|--------|
| Docker volume backup | 🚧 In Progress |
| Docker Compose stack backup | 🚧 In Progress |
| Docker image backup | 📋 Planned |
| Docker network backup | 📋 Planned |
| Docker secrets backup | 📋 Planned |
| Docker Swarm support | 📋 Planned |
| Docker exec hooks (pre/post backup) | 🚧 In Progress |
| Docker labels config | 📋 Planned |
| Docker logs backup | 📋 Planned |
| Docker health monitoring | 📋 Planned |
| Komodo integration | 📋 Planned |
| Test restore automation | 📋 Planned |
| Test backup validation | 📋 Planned |

### Authentication & Security
| Feature | Status |
|---------|--------|
| OIDC authentication (Authentik-first) | ✅ Done |
| Multi-organization support | 🚧 In Progress |
| Role-based access control (RBAC) | 🚧 In Progress |
| Agent API key authentication | 🚧 In Progress |
| Backup encryption key management | 🚧 In Progress |
| SSO group sync | 🚧 In Progress |
| Two-factor agent registration | 🚧 In Progress |
| Session management | 📋 Planned |
| IP allowlist | 📋 Planned |
| Audit logging | 🚧 In Progress |
| Immutable backups | 📋 Planned |
| Legal hold | 📋 Planned |

### Monitoring & Alerts
| Feature | Status |
|---------|--------|
| Metrics dashboard | 🚧 In Progress |
| Monitoring & alerts | 🚧 In Progress |
| Gatus-compatible health endpoints | 🚧 In Progress |
| Email notifications | 🚧 In Progress |
| Slack/Teams/Discord notifications | 📋 Planned |
| Webhook notifications | 📋 Planned |
| PagerDuty integration | 📋 Planned |
| Scheduled reports (weekly/monthly) | 🚧 In Progress |
| Deduplication stats | 🚧 In Progress |
| Cost estimation | 🚧 In Progress |
| SLA tracking | 📋 Planned |

### Administration
| Feature | Status |
|---------|--------|
| Superuser/admin panel | 📋 Planned |
| System settings page | 📋 Planned |
| User management | 📋 Planned |
| Organization management | 📋 Planned |
| Usage quotas | 📋 Planned |
| License management | 📋 Planned |
| Maintenance windows | 🚧 In Progress |

### User Experience
| Feature | Status |
|---------|--------|
| Dark mode | 🚧 In Progress |
| Localization (multi-language) | 🚧 In Progress |
| Onboarding wizard | 🚧 In Progress |
| Tags & search | 🚧 In Progress |
| Backup comments | 🚧 In Progress |
| Keyboard shortcuts | 📋 Planned |
| Global search | 🚧 In Progress |
| Activity feed | 📋 Planned |
| Favorites | 📋 Planned |

### Disaster Recovery
| Feature | Status |
|---------|--------|
| DR runbook generator | 🚧 In Progress |
| Geo-redundancy | 📋 Planned |
| Import existing Restic repos | 🚧 In Progress |

### DevOps & Integration
| Feature | Status |
|---------|--------|
| Docker deployment | ✅ Done |
| CI/CD (GitHub Actions) | ✅ Done |
| API documentation (OpenAPI) | 🚧 In Progress |
| Prometheus metrics | 📋 Planned |
| Terraform provider | 📋 Planned |
| Ansible role | 📋 Planned |

### Application-Specific Backup
| Feature | Status |
|---------|--------|
| Pi-hole backup | 📋 Planned |
| App hook templates (PostgreSQL, MySQL, MongoDB, Redis, etc.) | 🚧 In Progress |

---

## Tech Stack

| Component | Technology |
|-----------|------------|
| **Server** | Go 1.24+ / Gin / PostgreSQL |
| **Agent** | Go 1.24+ / Cobra |
| **Frontend** | React 18 / TypeScript / Vite / Tailwind CSS |
| **Authentication** | OIDC (coreos/go-oidc) |
| **Backup Engine** | Restic |
| **State Management** | TanStack Query |
| **Linting** | Biome (frontend) / staticcheck (Go) |

---

## Installation

> 🚧 **Coming Soon** - Installation instructions will be available when Keldris reaches beta.

### Prerequisites (Planned)
- Docker & Docker Compose
- PostgreSQL 15+
- OIDC Provider (Authentik, Keycloak, etc.)

### Quick Start (Coming Soon)
```bash
# Clone the repository
git clone https://github.com/MacJediWizard/keldris.git
cd keldris

# Configure environment
cp .env.example .env
# Edit .env with your OIDC settings

# Start with Docker
docker-compose up -d
```

### Agent Installation (Coming Soon)
```bash
# Linux
curl -fsSL https://keldris.com/install.sh | sudo bash

# macOS
brew install keldris-agent

# Windows
winget install keldris-agent
```

---

## Documentation

> 🚧 **Coming Soon** - Full documentation will be available at [docs.keldris.com](https://docs.keldris.com)

- Installation Guide
- OIDC Setup (Authentik, Keycloak)
- Agent Deployment
- Storage Backend Configuration
- API Reference
- Backup Strategies
- Disaster Recovery

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        keldris-server                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │   React UI   │  │   Go API     │  │    PostgreSQL        │   │
│  │  (Vite/TS)   │  │  (Gin)       │  │    (Multi-tenant)    │   │
│  └──────────────┘  └──────────────┘  └──────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                           │ HTTPS
            ┌──────────────┼──────────────┐
            │              │              │
     ┌──────┴─────┐ ┌──────┴─────┐ ┌──────┴─────┐
     │  keldris   │ │  keldris   │ │  keldris   │
     │   agent    │ │   agent    │ │   agent    │
     │  (Linux)   │ │  (macOS)   │ │ (Windows)  │
     └────────────┘ └────────────┘ └────────────┘
            │              │              │
            ▼              ▼              ▼
     ┌─────────────────────────────────────────┐
     │         Storage Backends                │
     │   S3 / B2 / Dropbox / Local / NFS       │
     └─────────────────────────────────────────┘
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

# Build everything
make build
```

---

## Contributing

Contributions are welcome! Please read our contributing guidelines (coming soon) before submitting PRs.

---

## Roadmap

- [x] Phase 1-4: Core functionality (Server, Agent, API, Frontend)
- [ ] Phase 5: Extended features (Notifications, Storage backends, Retention)
- [ ] Phase 6: Monitoring & Security (Alerts, Encryption, Audit logs)
- [ ] Phase 7: Enterprise features (Multi-org, RBAC, DR)
- [ ] Phase 8: Docker support (Volumes, Compose, Swarm)
- [ ] Phase 9: Polish (Dark mode, Localization, Onboarding)
- [ ] Beta Release
- [ ] v1.0 Release

---

## License

AGPLv3 - See [LICENSE](LICENSE)

---

## Acknowledgments

- [Restic](https://restic.net/) - The backup engine powering Keldris
- [Authentik](https://goauthentik.io/) - Primary OIDC provider for development

---

<p align="center">
  <strong>Powered by NeuroHolocron</strong><br>
  © MacJediWizard Consulting, Inc.
</p>
