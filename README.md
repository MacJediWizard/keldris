# Keldris

Self-hosted backup management with OIDC auth. Built on Restic.

Built for teams that outgrew basic backup tools but don't want to pay enterprise pricing.

> ⚠️ **Active development** - Not ready for production yet. Following along? Star the repo.

## Why I'm building this

I wanted a backup solution that could:
- Use existing OIDC/SSO infrastructure (not another set of credentials)
- Manage Restic repos across multiple machines from one UI
- Actually tell me when backups fail
- Handle multi-tenant environments properly

...I couldn't find one. So I'm building it.

## What it does

- **OIDC-first auth** - Authentik, Keycloak, whatever you use
- **Agent-based** - Small Go binary on each machine, talks to central server
- **Restic under the hood** - Encryption, deduplication, the good stuff
- **Multi-tenant** - Orgs and RBAC if you need it

---

## Status

Tracking what's done and what's next.

### Done
- Restic-powered encrypted backups
- Scheduled backups (cron)
- OIDC authentication (Authentik-first)
- Go agent (Linux, macOS, Windows)
- Docker deployment
- CI/CD with GitHub Actions

### Working on now
- Storage backends (S3, B2, Dropbox, local)
- Retention policies
- Restore UI with file browser
- Agent health monitoring
- Docker volume/compose backup
- Email notifications
- RBAC and multi-org

### On the roadmap
- Snapshot mounting (FUSE)
- Prometheus metrics
- Slack/Discord/webhook notifications
- Admin panel
- DR runbooks
- Import existing Restic repos

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

## Getting started

Not ready for general use yet. If you want to poke around:

```bash
git clone https://github.com/MacJediWizard/keldris.git
cd keldris
cp .env.example .env
# Edit .env with your OIDC settings
docker-compose up -d
```

You'll need Docker, PostgreSQL 15+, and an OIDC provider.

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

PRs welcome once this is more stable. For now, feel free to open issues.

---

## License

AGPLv3 - See [LICENSE](LICENSE)
