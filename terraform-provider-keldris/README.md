# Terraform Provider for Keldris

This is the official Terraform provider for [Keldris](https://github.com/MacJediWizard/keldris), an enterprise backup management system.

## Features

- **Resources:**
  - `keldris_agent` - Manage backup agents
  - `keldris_repository` - Manage backup repositories (S3, B2, SFTP, local, REST)
  - `keldris_schedule` - Manage backup schedules
  - `keldris_policy` - Manage backup policy templates

- **Data Sources:**
  - `keldris_agents` - List all agents
  - `keldris_repositories` - List all repositories

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24 (for building)

## Building

```bash
go build -o terraform-provider-keldris
```

## Installation

### Local Development

```bash
# Build the provider
go build -o terraform-provider-keldris

# Create the plugins directory
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/MacJediWizard/keldris/1.0.0/darwin_amd64

# Copy the provider
cp terraform-provider-keldris ~/.terraform.d/plugins/registry.terraform.io/MacJediWizard/keldris/1.0.0/darwin_amd64/
```

### Terraform Registry (Future)

Once published to the Terraform Registry:

```hcl
terraform {
  required_providers {
    keldris = {
      source  = "MacJediWizard/keldris"
      version = "~> 1.0"
    }
  }
}
```

## Quick Start

```hcl
provider "keldris" {
  url     = "https://keldris.example.com"
  api_key = var.keldris_api_key
}

# Create a backup repository
resource "keldris_repository" "s3" {
  name = "production-backups"
  type = "s3"

  s3_bucket            = "my-backup-bucket"
  s3_region            = "us-west-2"
  s3_access_key_id     = var.aws_access_key
  s3_secret_access_key = var.aws_secret_key
}

# Create an agent
resource "keldris_agent" "server" {
  hostname = "web-server-01"
}

# Create a backup schedule
resource "keldris_schedule" "daily" {
  name            = "Daily Backup"
  agent_id        = keldris_agent.server.id
  cron_expression = "0 2 * * *"

  paths = ["/home", "/var/www"]

  retention_policy {
    keep_daily = 7
  }

  repositories {
    repository_id = keldris_repository.s3.id
    priority      = 0
    enabled       = true
  }
}
```

## Configuration

The provider can be configured using:

1. **Provider block:**
   ```hcl
   provider "keldris" {
     url     = "https://keldris.example.com"
     api_key = "kld_your_api_key"
   }
   ```

2. **Environment variables:**
   - `KELDRIS_URL` - The Keldris server URL
   - `KELDRIS_API_KEY` - Your API key

## Documentation

See the full documentation at [docs/terraform.md](../docs/terraform.md) or the example configurations in [examples/terraform/](../examples/terraform/).

## Development

### Running Tests

```bash
go test ./...
```

### Generating Documentation

```bash
go generate ./...
```

## License

This provider is distributed under the same license as the main Keldris project.
