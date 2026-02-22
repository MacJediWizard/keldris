<p align="center">
  <img src="https://cdn.macjediwizard.com/cdn/Keldris%20Branding%20Images/keldris-webicon-727336cd.png" alt="Keldris" width="200">
</p>

<h1 align="center">Keldris</h1>
<p align="center"><strong>Secure Keeper of Your Data</strong></p>

<p align="center">
  Self-hosted backup management with OIDC auth. Built on Restic.<br>
  <em>For teams that outgrew basic backup tools but don't want to pay enterprise pricing.</em>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/status-v1.0.0--beta.4-blue" alt="Status">
  <img src="https://img.shields.io/badge/status-v1.0.0--beta.1-blue" alt="Status">
  <img src="https://img.shields.io/github/license/MacJediWizard/keldris" alt="License">
  <img src="https://img.shields.io/badge/Go-1.25.7+-00ADD8?logo=go" alt="Go">
  <img src="https://img.shields.io/badge/React-18-61DAFB?logo=react" alt="React">
</p>

---

## Screenshots

<p align="center">
  <img src="docs/images/screenshots/dashboard.svg" alt="Dashboard" width="800"><br>
  <em>Dashboard - backup status at a glance</em>
</p>

<p align="center">
  <img src="docs/images/screenshots/agents.svg" alt="Agent Management" width="800"><br>
  <em>Agent management across your infrastructure</em>
</p>

<p align="center">
  <img src="docs/images/screenshots/file-browser.svg" alt="File Browser" width="800"><br>
  <em>Browse and restore files from any snapshot</em>
</p>

<p align="center">
  <img src="docs/images/screenshots/schedules.svg" alt="Backup Schedules" width="800"><br>
  <em>Flexible cron-based scheduling</em>
</p>

---

## Why I'm building this

I wanted a backup solution that could:
- Use existing OIDC/SSO infrastructure (not another set of credentials)
- Manage Restic repos across multiple machines from one UI
- Actually tell me when backups fail
- Handle multi-tenant environments properly

...I couldn't find one. So I'm building it.

## Features

### Backup & Recovery
- **Restic-powered** - Encrypted, deduplicated backups you can trust
- **Flexible scheduling** - Cron-based with configurable retention policies
- **Full restore UI** - Browse snapshots, compare diffs, restore individual files
- **File version history** - Track changes across snapshots
- **Pre/post scripts** - Run custom commands before and after backups
- **Docker backup support** - Back up Docker volumes and containers

### Storage Backends
- **Cloud** - Amazon S3, Backblaze B2, Dropbox
- **Self-hosted** - Local disk, SFTP, REST server
- **Network mounts** - NFS, SMB with automatic detection and handling

### Authentication & Security
- **OIDC/SSO** - Authentik, Keycloak, Okta, Auth0, Azure AD, Google Workspace
- **SSO group sync** - Map identity provider groups to Keldris roles
- **Role-based access control** - Fine-grained permissions per organization
- **AES-256-GCM encryption** - Backup credentials and sensitive config encrypted at rest
- **Session hardening** - HttpOnly/Secure/SameSite cookies with idle and absolute timeouts
- **Security headers** - CSP, HSTS, X-Frame-Options enabled by default
- **Rate limiting** - Per-IP with optional Redis backend for distributed deployments

### Monitoring & Alerting
- **Agent health monitoring** - Real-time status with health history
- **Notifications** - Email, Slack, Discord, Microsoft Teams, PagerDuty
- **SLA tracking** - Monitor backup compliance against defined targets
- **Prometheus metrics** - `/metrics` endpoint for Grafana dashboards

### Management
- **Multi-organization** - Full multi-tenant isolation with RBAC
- **Audit logging** - Track every administrative action for compliance
- **Cost estimation** - Forecast storage costs across backends
- **Backup tagging** - Organize and filter backups with custom tags
- **Agent groups** - Logical grouping for bulk operations

### Operations
- **DR runbooks** - Auto-generated disaster recovery procedures
- **DR test automation** - Validate your recovery plan actually works
- **White labeling** - Custom branding for MSPs and internal deployments
- **Air gap deployment** - Run fully offline with no internet dependency
- **Onboarding wizard** - Guided first-run setup

### Platform
- **Cross-platform agent** - Single Go binary for Linux, macOS, and Windows
- **Docker deployment** - Production-ready compose files with health checks
- **Dark mode** - Because we're not animals

---

## Status

### Roadmap
- Snapshot mounting (FUSE)
- Import existing Restic repositories
- Mobile-responsive layout
### Implemented
- Restic-powered encrypted backups with deduplication
- Flexible cron-based scheduling with retention policies
- OIDC authentication (tested with Authentik, Keycloak)
- Go agent for Linux, macOS, and Windows
- Storage backends: S3, B2, Dropbox, local, SFTP, REST
- Full restore UI with file browser
- Snapshot comparison (diff between backups)
- File version history browser
- Agent health monitoring with history
- Backup tagging and organization
- Pre/post backup scripts
- Network mount detection and handling
- Email notifications
- Slack notifications
- Discord notifications
- Teams notifications
- Multi-org with RBAC
- Audit logging
- Dark mode
- First-run onboarding wizard
- DR runbook generation
- Prometheus metrics endpoint
- Docker deployment

### In Progress
- Agent CLI backup/restore commands
- Email report delivery
- DR test automation
- UI polish and bug fixes

### Roadmap
- Snapshot mounting (FUSE)
- Import existing Restic repos
- Mobile-friendly improvements

---

## Tech Stack

| Component | Technology |
|-----------|------------|
| **Server** | Go 1.25.7+ / Gin / PostgreSQL |
| **Agent** | Go 1.25.7+ / Cobra |
| **Frontend** | React 18 / TypeScript / Vite / Tailwind CSS |
| **Authentication** | OIDC (coreos/go-oidc) |
| **Backup Engine** | Restic |
| **State Management** | TanStack Query |
| **Linting** | Biome (frontend) / staticcheck (Go) |

---

## Getting Started

### Prerequisites

- Docker and Docker Compose v2+
- An OIDC provider (Authentik, Keycloak, Okta, Auth0, Azure AD, or Google Workspace)

### Quick Start (Docker)

1. Clone the repository:
   ```bash
   git clone https://github.com/MacJediWizard/keldris.git
   cd keldris
   ```

2. Configure environment:
   ```bash
   cp .env.example .env
   ```

3. Generate security keys and add to `.env`:
   ```bash
   openssl rand -base64 48  # Use for SESSION_SECRET (min 32 bytes)
   openssl rand -hex 32     # Use for ENCRYPTION_KEY (AES-256, exactly 32 bytes)
   ```

4. Change the default database password in `.env`:
   ```bash
   # Replace 'changeme' in DATABASE_URL with a strong password
   # Also set POSTGRES_PASSWORD to the same value for docker-compose
   ```

5. Edit `.env` and configure your OIDC provider:
   ```
   OIDC_ISSUER=https://your-auth-server/application/o/keldris/
   OIDC_CLIENT_ID=your-client-id
   OIDC_CLIENT_SECRET=your-client-secret
   OIDC_REDIRECT_URL=http://localhost:8080/auth/callback
   ```

   See [docs/oidc-setup.md](docs/oidc-setup.md) for provider-specific instructions.

6. Start the services:
   ```bash
   cd docker
   docker compose up -d
   ```

7. Access the UI at [http://localhost:8080](http://localhost:8080)

### Using Pre-built Images

Pull from GitHub Container Registry instead of building from source:

```bash
docker pull ghcr.io/macjediwizard/keldris-server:1.0.0-beta.4
docker pull ghcr.io/macjediwizard/keldris-agent:1.0.0-beta.4
```

See [docker/docker-compose.images.yml](docker/docker-compose.images.yml) for production deployment with pre-built images.

### Installing the Agent

See [docs/agent-installation.md](docs/agent-installation.md) for Linux, macOS, and Windows installation.

For setup guides, see:

- [OIDC Setup Guide](docs/oidc-setup.md) - Configure your identity provider
- [Agent Installation](docs/agent-installation.md) - Install and configure backup agents
- [Production Security Guide](docs/production-security.md) - Hardening for production deployments
- [Bare Metal Restore](docs/bare-metal-restore.md) - Full system recovery procedures
- [Network Mounts](docs/network-mounts.md) - NFS, SMB, and network storage configuration
- [Infrastructure Requirements](docs/infrastructure-requirements.md) - Hardware and software prerequisites

For setup guides, see:

- [OIDC Setup Guide](docs/oidc-setup.md) - Configure your identity provider
- [Agent Installation](docs/agent-installation.md) - Install and configure backup agents
- [Production Security Guide](docs/production-security.md) - Hardening for production deployments
You'll need Docker, Go 1.25.7+ (for security patches), PostgreSQL 15+, and an OIDC provider.

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

## Editions

Keldris is available in three editions:

| Feature | Free | Pro | Enterprise |
|---------|:----:|:---:|:----------:|
| Encrypted backups (Restic) | ✓ | ✓ | ✓ |
| Cross-platform agent | ✓ | ✓ | ✓ |
| Web UI | ✓ | ✓ | ✓ |
| Cron scheduling | ✓ | ✓ | ✓ |
| Local & S3 storage | ✓ | ✓ | ✓ |
| Basic OIDC auth | ✓ | ✓ | ✓ |
| All storage backends | | ✓ | ✓ |
| Advanced retention policies | | ✓ | ✓ |
| Multi-organization | | ✓ | ✓ |
| Webhook notifications | | ✓ | ✓ |
| Advanced reporting | | ✓ | ✓ |
| Priority email support | | ✓ | ✓ |
| SAML/SSO providers | | | ✓ |
| Audit compliance reports | | | ✓ |
| Custom integrations | | | ✓ |
| SLA & dedicated support | | | ✓ |
| **Price** | **Free** | **$9/agent/mo** | **Contact us** |

The Free edition is open-source under AGPLv3. Pro and Enterprise are commercial licenses.

---

## Security

Keldris is built with security as a priority:

- **OIDC-first authentication** - No local passwords; delegates to your identity provider
- **Encryption at rest** - Backup credentials and sensitive configuration encrypted with AES-256-GCM
- **Session hardening** - HttpOnly, Secure, SameSite=Lax cookies with configurable idle and absolute timeouts
- **Security headers** - CSP, HSTS, X-Frame-Options, and more applied by default
- **Rate limiting** - Per-IP rate limiting with optional Redis backend for distributed deployments
- **Multi-tenant isolation** - All queries scoped to organization ID
- **Audit logging** - Track administrative actions for compliance

For production deployment, see:

- [Production Security Guide](docs/production-security.md) - Hardening recommendations
- [Security Checklist](docs/security-checklist.md) - Pre-deployment checklist and proxy configuration

---

## Contributing

PRs welcome once this is more stable. For now, feel free to open issues.

---

## License

AGPLv3 (Free edition) - See [LICENSE](LICENSE)

Pro and Enterprise editions are available under commercial license.
