# Getting Started with Keldris

Welcome to Keldris, a self-hosted backup management system built on Restic. This guide will help you get your first backup running.

## Prerequisites

Before you begin, ensure you have:

- Docker and Docker Compose installed
- PostgreSQL 15+ (or use the included Docker container)
- An OIDC provider (Authentik, Keycloak, etc.)
- Access to a storage backend (local, S3, B2, etc.)

## Quick Start

### 1. Clone and Configure

```bash
git clone https://github.com/MacJediWizard/keldris.git
cd keldris
cp .env.example .env
```

Edit `.env` with your settings:

```env
# Database
DATABASE_URL=postgres://keldris:password@localhost:5432/keldris?sslmode=disable

# OIDC Provider
OIDC_ISSUER_URL=https://your-auth-provider.com
OIDC_CLIENT_ID=keldris
OIDC_CLIENT_SECRET=your-client-secret

# Server
SERVER_URL=https://keldris.example.com
SESSION_SECRET=generate-a-random-secret
```

### 2. Start the Server

```bash
docker-compose up -d
```

The web interface will be available at `http://localhost:8080`.

### 3. Complete Onboarding

1. Navigate to the web interface
2. Log in with your OIDC provider
3. Follow the onboarding wizard to create your first organization

### 4. Install an Agent

Install the Keldris agent on machines you want to back up:

**Linux:**
```bash
curl -sSL https://releases.keldris.io/install-linux.sh | sudo bash
```

**macOS:**
```bash
curl -sSL https://releases.keldris.io/install-macos.sh | bash
```

**Windows (PowerShell as Administrator):**
```powershell
irm https://releases.keldris.io/install-windows.ps1 | iex
```

### 5. Register the Agent

After installation, register the agent with your server:

```bash
keldris-agent register --server https://your-keldris-server.com
```

You'll be prompted for an API key. Generate one from the web interface under **Agents > Register New Agent**.

### 6. Create a Repository

1. In the web interface, go to **Repositories**
2. Click **Add Repository**
3. Choose your storage backend (local, S3, B2, etc.)
4. Enter connection details and encryption password

### 7. Set Up a Backup Schedule

1. Go to **Schedules**
2. Click **Create Schedule**
3. Select the agent and repository
4. Configure the cron expression and paths to back up
5. Set retention policies

## Next Steps

- [Installation Guide](installation.md) - Detailed installation options
- [Configuration Reference](configuration.md) - All configuration options
- [Agent Deployment](agent-deployment.md) - Deploy agents at scale
- [Troubleshooting](troubleshooting.md) - Common issues and solutions

## Architecture Overview

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

## Key Concepts

### Organizations

Organizations provide multi-tenant isolation. Each organization has its own agents, repositories, and backups.

### Agents

The Keldris agent is a lightweight Go binary that runs on each machine you want to back up. It communicates with the central server over HTTPS.

### Repositories

A repository is a Restic-compatible storage location. Keldris supports multiple backends including local storage, S3, B2, Dropbox, and SFTP.

### Schedules

Schedules define when backups run using cron expressions. Each schedule can have retention policies to automatically prune old snapshots.

### Snapshots

Each backup creates a snapshot. You can browse, compare, and restore files from any snapshot through the web interface.
