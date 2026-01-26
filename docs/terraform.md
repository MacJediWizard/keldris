# Terraform Provider for Keldris

The Keldris Terraform provider allows you to manage your backup infrastructure as code. Define agents, repositories, schedules, and policies using Terraform's declarative configuration language.

## Installation

### From Terraform Registry (Future)

Once published to the Terraform Registry, you can use:

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

### Local Development

Build and install the provider locally:

```bash
cd terraform-provider-keldris
go build -o terraform-provider-keldris
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/MacJediWizard/keldris/1.0.0/darwin_amd64
mv terraform-provider-keldris ~/.terraform.d/plugins/registry.terraform.io/MacJediWizard/keldris/1.0.0/darwin_amd64/
```

## Configuration

Configure the provider with your Keldris server URL and API key:

```hcl
provider "keldris" {
  url     = "https://keldris.example.com"
  api_key = var.keldris_api_key
}
```

### Environment Variables

You can also use environment variables:

- `KELDRIS_URL` - The Keldris server URL
- `KELDRIS_API_KEY` - Your API key for authentication

## Resources

### keldris_agent

Manages a backup agent.

```hcl
resource "keldris_agent" "web_server" {
  hostname = "web-server-01"
}

# The API key is only available at creation time
output "agent_api_key" {
  value     = keldris_agent.web_server.api_key
  sensitive = true
}
```

#### Attributes

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `hostname` | string | yes | The hostname of the agent |

#### Read-Only Attributes

| Name | Type | Description |
|------|------|-------------|
| `id` | string | The unique identifier |
| `status` | string | The current status (pending, active, offline, disabled) |
| `api_key` | string | The API key (only available at creation) |

### keldris_repository

Manages a backup repository.

```hcl
# S3 Repository
resource "keldris_repository" "s3_backup" {
  name           = "production-backups"
  type           = "s3"
  escrow_enabled = true

  s3_bucket            = "my-backup-bucket"
  s3_region            = "us-west-2"
  s3_access_key_id     = var.aws_access_key
  s3_secret_access_key = var.aws_secret_key
  s3_path              = "backups/"
}

# Local Repository
resource "keldris_repository" "local_backup" {
  name = "local-backups"
  type = "local"

  local_path = "/mnt/backups"
}

# SFTP Repository
resource "keldris_repository" "sftp_backup" {
  name = "remote-backups"
  type = "sftp"

  sftp_host     = "backup.example.com"
  sftp_port     = 22
  sftp_user     = "backup"
  sftp_password = var.sftp_password
  sftp_path     = "/backups"
}

# B2 Repository
resource "keldris_repository" "b2_backup" {
  name = "cloud-backups"
  type = "b2"

  b2_account_id      = var.b2_account_id
  b2_application_key = var.b2_app_key
  b2_bucket          = "my-backup-bucket"
  b2_path            = "keldris/"
}
```

#### Attributes

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | yes | The repository name |
| `type` | string | yes | The type (s3, b2, sftp, local, rest, dropbox) |
| `escrow_enabled` | bool | no | Enable key escrow for password recovery |

#### S3 Configuration

| Name | Type | Description |
|------|------|-------------|
| `s3_bucket` | string | S3 bucket name |
| `s3_region` | string | S3 region |
| `s3_endpoint` | string | S3 endpoint URL (for S3-compatible storage) |
| `s3_access_key_id` | string | Access key ID |
| `s3_secret_access_key` | string | Secret access key |
| `s3_path` | string | Path prefix within the bucket |

#### SFTP Configuration

| Name | Type | Description |
|------|------|-------------|
| `sftp_host` | string | SFTP server hostname |
| `sftp_port` | int | SFTP server port (default: 22) |
| `sftp_user` | string | SFTP username |
| `sftp_password` | string | SFTP password |
| `sftp_private_key` | string | SFTP private key (PEM format) |
| `sftp_path` | string | Path on the SFTP server |

### keldris_schedule

Manages a backup schedule.

```hcl
resource "keldris_schedule" "daily_backup" {
  name            = "Daily Home Backup"
  agent_id        = keldris_agent.web_server.id
  cron_expression = "0 2 * * *"  # Daily at 2 AM

  paths = [
    "/home",
    "/var/www"
  ]

  excludes = [
    "*.tmp",
    "*.log",
    "node_modules"
  ]

  retention_policy {
    keep_last    = 5
    keep_daily   = 7
    keep_weekly  = 4
    keep_monthly = 6
  }

  backup_window {
    start = "00:00"
    end   = "06:00"
  }

  repositories {
    repository_id = keldris_repository.s3_backup.id
    priority      = 0
    enabled       = true
  }

  bandwidth_limit_kb = 10240  # 10 MB/s
  enabled            = true
}
```

#### Attributes

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | yes | The schedule name |
| `agent_id` | string | yes | The agent ID this schedule belongs to |
| `cron_expression` | string | yes | Cron expression for scheduling |
| `paths` | list(string) | yes | Paths to back up |
| `excludes` | list(string) | no | Patterns to exclude |
| `enabled` | bool | no | Whether the schedule is enabled (default: true) |
| `bandwidth_limit_kb` | int | no | Upload bandwidth limit in KB/s |
| `compression_level` | string | no | Compression level (off, auto, max) |
| `excluded_hours` | list(int) | no | Hours (0-23) when backups should not run |

#### Retention Policy Block

| Name | Type | Description |
|------|------|-------------|
| `keep_last` | int | Keep the last N snapshots |
| `keep_hourly` | int | Keep N hourly snapshots |
| `keep_daily` | int | Keep N daily snapshots |
| `keep_weekly` | int | Keep N weekly snapshots |
| `keep_monthly` | int | Keep N monthly snapshots |
| `keep_yearly` | int | Keep N yearly snapshots |

#### Backup Window Block

| Name | Type | Description |
|------|------|-------------|
| `start` | string | Start time in HH:MM format |
| `end` | string | End time in HH:MM format |

#### Repositories Block

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `repository_id` | string | yes | The repository ID |
| `priority` | int | no | Priority (0 = primary) |
| `enabled` | bool | no | Whether this association is enabled |

### keldris_policy

Manages a backup policy template.

```hcl
resource "keldris_policy" "standard" {
  name        = "Standard Daily Backup"
  description = "Standard backup policy for production servers"

  cron_expression = "0 2 * * *"

  paths = [
    "/home",
    "/var/www",
    "/etc"
  ]

  excludes = [
    "*.tmp",
    "*.log",
    "*.cache",
    "node_modules",
    ".git"
  ]

  retention_policy {
    keep_last    = 5
    keep_daily   = 7
    keep_weekly  = 4
    keep_monthly = 12
    keep_yearly  = 2
  }

  backup_window {
    start = "00:00"
    end   = "06:00"
  }

  bandwidth_limit_kb = 10240
}
```

## Data Sources

### keldris_agents

Fetches all agents in your organization.

```hcl
data "keldris_agents" "all" {}

output "agent_count" {
  value = length(data.keldris_agents.all.agents)
}

output "active_agents" {
  value = [for a in data.keldris_agents.all.agents : a.hostname if a.status == "active"]
}
```

### keldris_repositories

Fetches all repositories in your organization.

```hcl
data "keldris_repositories" "all" {}

output "repository_count" {
  value = length(data.keldris_repositories.all.repositories)
}

output "s3_repositories" {
  value = [for r in data.keldris_repositories.all.repositories : r.name if r.type == "s3"]
}
```

## Complete Example

Here's a complete example setting up a backup infrastructure:

```hcl
terraform {
  required_providers {
    keldris = {
      source  = "MacJediWizard/keldris"
      version = "~> 1.0"
    }
  }
}

provider "keldris" {
  url     = var.keldris_url
  api_key = var.keldris_api_key
}

# Variables
variable "keldris_url" {
  description = "Keldris server URL"
  type        = string
}

variable "keldris_api_key" {
  description = "Keldris API key"
  type        = string
  sensitive   = true
}

variable "aws_access_key" {
  description = "AWS access key"
  type        = string
  sensitive   = true
}

variable "aws_secret_key" {
  description = "AWS secret key"
  type        = string
  sensitive   = true
}

# Create a backup repository
resource "keldris_repository" "production" {
  name           = "production-backups"
  type           = "s3"
  escrow_enabled = true

  s3_bucket            = "company-backups"
  s3_region            = "us-west-2"
  s3_access_key_id     = var.aws_access_key
  s3_secret_access_key = var.aws_secret_key
  s3_path              = "production/"
}

# Create a backup policy
resource "keldris_policy" "standard" {
  name        = "Standard Production Backup"
  description = "Standard backup policy for production servers"

  cron_expression = "0 2 * * *"

  paths = [
    "/home",
    "/var/www",
    "/etc",
    "/opt"
  ]

  excludes = [
    "*.tmp",
    "*.log",
    "*.cache",
    "node_modules",
    ".git",
    "__pycache__"
  ]

  retention_policy {
    keep_last    = 5
    keep_daily   = 7
    keep_weekly  = 4
    keep_monthly = 12
  }
}

# Create agents for each server
resource "keldris_agent" "web_servers" {
  count    = 3
  hostname = "web-server-${count.index + 1}"
}

# Create schedules for each agent
resource "keldris_schedule" "web_backups" {
  count           = 3
  name            = "Daily backup for web-server-${count.index + 1}"
  agent_id        = keldris_agent.web_servers[count.index].id
  cron_expression = "0 2 * * *"

  paths    = keldris_policy.standard.paths
  excludes = keldris_policy.standard.excludes

  retention_policy {
    keep_last    = 5
    keep_daily   = 7
    keep_weekly  = 4
    keep_monthly = 12
  }

  repositories {
    repository_id = keldris_repository.production.id
    priority      = 0
    enabled       = true
  }

  enabled = true
}

# Outputs
output "agent_api_keys" {
  value = {
    for idx, agent in keldris_agent.web_servers : agent.hostname => agent.api_key
  }
  sensitive = true
}

output "repository_password" {
  value     = keldris_repository.production.password
  sensitive = true
}
```

## Importing Existing Resources

You can import existing resources into Terraform state:

```bash
# Import an agent
terraform import keldris_agent.example <agent-id>

# Import a repository
terraform import keldris_repository.example <repository-id>

# Import a schedule
terraform import keldris_schedule.example <schedule-id>

# Import a policy
terraform import keldris_policy.example <policy-id>
```

Note: When importing, sensitive values like API keys and passwords will not be available since they are only returned at creation time.
