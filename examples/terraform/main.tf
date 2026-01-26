# Keldris Backup Infrastructure Example
#
# This example demonstrates how to use the Keldris Terraform provider
# to manage backup infrastructure as code.

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

# =============================================================================
# REPOSITORIES
# =============================================================================

# Primary S3 backup repository
resource "keldris_repository" "primary" {
  name           = "primary-backups"
  type           = "s3"
  escrow_enabled = true

  s3_bucket            = var.s3_bucket
  s3_region            = var.s3_region
  s3_access_key_id     = var.aws_access_key
  s3_secret_access_key = var.aws_secret_key
  s3_path              = "backups/primary/"
}

# Secondary local repository for quick restores
resource "keldris_repository" "local" {
  name = "local-fast-restore"
  type = "local"

  local_path = "/mnt/backup-cache"
}

# =============================================================================
# POLICIES
# =============================================================================

# Standard production backup policy
resource "keldris_policy" "production" {
  name        = "Production Standard"
  description = "Standard backup policy for production servers"

  cron_expression = "0 2 * * *"  # Daily at 2 AM

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
    "__pycache__",
    "*.pyc"
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

  bandwidth_limit_kb = 10240  # 10 MB/s
}

# Database backup policy - more frequent, longer retention
resource "keldris_policy" "database" {
  name        = "Database Critical"
  description = "Critical database backup policy with hourly backups"

  cron_expression = "0 * * * *"  # Every hour

  paths = [
    "/var/lib/postgresql",
    "/var/lib/mysql",
    "/var/backups/database"
  ]

  retention_policy {
    keep_last    = 24
    keep_hourly  = 24
    keep_daily   = 30
    keep_weekly  = 12
    keep_monthly = 24
    keep_yearly  = 5
  }
}

# =============================================================================
# AGENTS
# =============================================================================

# Web servers
resource "keldris_agent" "web" {
  count    = var.web_server_count
  hostname = "web-${format("%02d", count.index + 1)}"
}

# Database servers
resource "keldris_agent" "database" {
  count    = var.db_server_count
  hostname = "db-${format("%02d", count.index + 1)}"
}

# =============================================================================
# SCHEDULES
# =============================================================================

# Web server schedules using production policy
resource "keldris_schedule" "web" {
  count = var.web_server_count

  name            = "Web Server ${count.index + 1} Daily Backup"
  agent_id        = keldris_agent.web[count.index].id
  cron_expression = keldris_policy.production.cron_expression

  paths    = keldris_policy.production.paths
  excludes = keldris_policy.production.excludes

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

  # Primary repository
  repositories {
    repository_id = keldris_repository.primary.id
    priority      = 0
    enabled       = true
  }

  # Secondary local repository for quick restores
  repositories {
    repository_id = keldris_repository.local.id
    priority      = 1
    enabled       = true
  }

  bandwidth_limit_kb = 10240
  enabled            = true
}

# Database server schedules using database policy
resource "keldris_schedule" "database" {
  count = var.db_server_count

  name            = "Database Server ${count.index + 1} Hourly Backup"
  agent_id        = keldris_agent.database[count.index].id
  cron_expression = keldris_policy.database.cron_expression

  paths = keldris_policy.database.paths

  retention_policy {
    keep_last    = 24
    keep_hourly  = 24
    keep_daily   = 30
    keep_weekly  = 12
    keep_monthly = 24
    keep_yearly  = 5
  }

  repositories {
    repository_id = keldris_repository.primary.id
    priority      = 0
    enabled       = true
  }

  enabled = true
}

# =============================================================================
# OUTPUTS
# =============================================================================

output "web_agent_api_keys" {
  description = "API keys for web server agents (save these securely!)"
  value = {
    for agent in keldris_agent.web : agent.hostname => agent.api_key
  }
  sensitive = true
}

output "db_agent_api_keys" {
  description = "API keys for database server agents (save these securely!)"
  value = {
    for agent in keldris_agent.database : agent.hostname => agent.api_key
  }
  sensitive = true
}

output "repository_password" {
  description = "Repository encryption password (save this securely!)"
  value       = keldris_repository.primary.password
  sensitive   = true
}

output "repository_ids" {
  description = "Repository IDs"
  value = {
    primary = keldris_repository.primary.id
    local   = keldris_repository.local.id
  }
}
