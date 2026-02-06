# Installation Guide

This guide covers all installation methods for the Keldris server and agents.

## Server Installation

### Docker (Recommended)

The easiest way to run Keldris is with Docker Compose.

#### Prerequisites

- Docker 20.10+
- Docker Compose 2.0+
- 2GB RAM minimum
- 10GB disk space for the database

#### Quick Start

```bash
# Clone the repository
git clone https://github.com/MacJediWizard/keldris.git
cd keldris

# Copy environment template
cp .env.example .env

# Edit configuration
nano .env

# Start all services
docker-compose up -d
```

#### Docker Compose File

The included `docker-compose.yml` provides:

- Keldris server
- PostgreSQL 15 database
- Automatic database migrations

```yaml
services:
  keldris:
    image: ghcr.io/macjediwizard/keldris:latest
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://keldris:password@db:5432/keldris?sslmode=disable
    depends_on:
      - db

  db:
    image: postgres:15
    volumes:
      - postgres_data:/var/lib/postgresql/data
    environment:
      - POSTGRES_USER=keldris
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=keldris

volumes:
  postgres_data:
```

### Binary Installation

For non-Docker deployments, download the pre-built binary.

#### Download

```bash
# Linux (amd64)
curl -Lo keldris https://releases.keldris.io/latest/keldris-linux-amd64
chmod +x keldris

# Linux (arm64)
curl -Lo keldris https://releases.keldris.io/latest/keldris-linux-arm64
chmod +x keldris

# macOS (Intel)
curl -Lo keldris https://releases.keldris.io/latest/keldris-darwin-amd64
chmod +x keldris

# macOS (Apple Silicon)
curl -Lo keldris https://releases.keldris.io/latest/keldris-darwin-arm64
chmod +x keldris
```

#### Database Setup

Keldris requires PostgreSQL 15+.

```bash
# Create database and user
sudo -u postgres psql
CREATE USER keldris WITH PASSWORD 'your-secure-password';
CREATE DATABASE keldris OWNER keldris;
\q
```

#### Run the Server

```bash
export DATABASE_URL="postgres://keldris:password@localhost:5432/keldris?sslmode=disable"
export OIDC_ISSUER_URL="https://your-auth-provider.com"
export OIDC_CLIENT_ID="keldris"
export OIDC_CLIENT_SECRET="your-secret"
export SESSION_SECRET="generate-a-random-secret"

./keldris server
```

#### Systemd Service

Create `/etc/systemd/system/keldris.service`:

```ini
[Unit]
Description=Keldris Backup Server
After=network-online.target postgresql.service
Wants=network-online.target

[Service]
Type=simple
User=keldris
WorkingDirectory=/opt/keldris
ExecStart=/opt/keldris/keldris server
Restart=always
RestartSec=10
EnvironmentFile=/etc/keldris/keldris.env

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable keldris
sudo systemctl start keldris
```

### Building from Source

#### Prerequisites

- Go 1.24+
- Node.js 20+
- pnpm

#### Build

```bash
git clone https://github.com/MacJediWizard/keldris.git
cd keldris

# Install dependencies
make deps

# Build server and agent
make build

# Binaries are in ./bin/
./bin/keldris server
```

## Agent Installation

See [Agent Deployment](agent-deployment.md) for detailed agent installation instructions.

### Quick Install

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

## Reverse Proxy Setup

### Nginx

```nginx
server {
    listen 443 ssl http2;
    server_name keldris.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

### Traefik

```yaml
http:
  routers:
    keldris:
      rule: "Host(`keldris.example.com`)"
      service: keldris
      tls:
        certResolver: letsencrypt

  services:
    keldris:
      loadBalancer:
        servers:
          - url: "http://keldris:8080"
```

### Caddy

```
keldris.example.com {
    reverse_proxy localhost:8080
}
```

## Upgrading

### Docker

```bash
# Pull latest image
docker-compose pull

# Restart with new image
docker-compose up -d

# Database migrations run automatically
```

### Binary

```bash
# Stop the service
sudo systemctl stop keldris

# Download new binary
curl -Lo /opt/keldris/keldris https://releases.keldris.io/latest/keldris-linux-amd64
chmod +x /opt/keldris/keldris

# Start the service (migrations run automatically)
sudo systemctl start keldris
```

## High Availability

For production deployments, consider:

1. **Database**: Use PostgreSQL with streaming replication
2. **Load Balancing**: Run multiple Keldris instances behind a load balancer
3. **Session Storage**: Configure Redis for shared session storage
4. **Storage**: Use highly available storage backends (S3, etc.)

## Security Considerations

1. Always use HTTPS in production
2. Use strong, unique passwords for the database
3. Keep the session secret secure and never commit it to version control
4. Regularly update to the latest version
5. Configure firewall rules to restrict access
6. Enable audit logging
