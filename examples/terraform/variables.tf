# Keldris Provider Configuration
variable "keldris_url" {
  description = "The URL of the Keldris server"
  type        = string
}

variable "keldris_api_key" {
  description = "API key for authenticating with the Keldris server"
  type        = string
  sensitive   = true
}

# AWS/S3 Configuration
variable "s3_bucket" {
  description = "S3 bucket name for backups"
  type        = string
  default     = "my-keldris-backups"
}

variable "s3_region" {
  description = "AWS region for the S3 bucket"
  type        = string
  default     = "us-west-2"
}

variable "aws_access_key" {
  description = "AWS access key ID"
  type        = string
  sensitive   = true
}

variable "aws_secret_key" {
  description = "AWS secret access key"
  type        = string
  sensitive   = true
}

# Infrastructure Configuration
variable "web_server_count" {
  description = "Number of web servers to configure"
  type        = number
  default     = 3
}

variable "db_server_count" {
  description = "Number of database servers to configure"
  type        = number
  default     = 2
}
