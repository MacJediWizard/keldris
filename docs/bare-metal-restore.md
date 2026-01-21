# Bare-Metal Restore Guide

This guide covers the complete process for restoring a system from scratch using Keldris backups.

## Prerequisites

Before starting a bare-metal restore:

1. **Install a fresh operating system** on the target machine
2. **Install Restic** on the target machine
3. **Have access to your backup repository credentials** (stored in a secure location)
4. **Network connectivity** to your backup storage (S3, B2, SFTP, etc.)

## Step 1: Install Dependencies

### Linux (Debian/Ubuntu)
```bash
sudo apt update
sudo apt install -y restic curl jq
```

### Linux (RHEL/Fedora)
```bash
sudo dnf install -y restic curl jq
```

### macOS
```bash
brew install restic curl jq
```

## Step 2: Configure Repository Access

Set the environment variables for your backup repository:

### S3 Storage
```bash
export RESTIC_REPOSITORY="s3:s3.amazonaws.com/your-bucket-name/path"
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export RESTIC_PASSWORD="your-repository-password"
```

### Backblaze B2
```bash
export RESTIC_REPOSITORY="b2:your-bucket-name:/path"
export B2_ACCOUNT_ID="your-account-id"
export B2_ACCOUNT_KEY="your-account-key"
export RESTIC_PASSWORD="your-repository-password"
```

### SFTP
```bash
export RESTIC_REPOSITORY="sftp:user@host:/path/to/repo"
export RESTIC_PASSWORD="your-repository-password"
```

### Local Storage
```bash
export RESTIC_REPOSITORY="/path/to/local/repo"
export RESTIC_PASSWORD="your-repository-password"
```

## Step 3: Verify Repository Access

Test that you can access the repository:

```bash
restic snapshots
```

This should list all available snapshots. If you get an error, verify your credentials and network connectivity.

## Step 4: Identify the Snapshot to Restore

List snapshots with details:

```bash
# List all snapshots
restic snapshots

# List snapshots for a specific hostname
restic snapshots --host original-hostname

# List snapshots with more details
restic snapshots --json | jq '.'
```

Note the snapshot ID you want to restore from. You can use:
- A full snapshot ID (e.g., `abc123def456`)
- A short snapshot ID (e.g., `abc123de`)
- `latest` for the most recent snapshot

## Step 5: Preview Restore Contents

Before restoring, preview what will be restored:

```bash
# List files in a snapshot
restic ls <snapshot-id>

# List files matching a pattern
restic ls <snapshot-id> --path "/etc/"
```

## Step 6: Perform the Restore

### Full System Restore

Restore the entire snapshot to the root filesystem:

```bash
# Restore to root (requires root privileges)
sudo restic restore <snapshot-id> --target /

# Or restore to a staging directory first
sudo restic restore <snapshot-id> --target /restore-staging
```

### Selective Restore

Restore specific paths:

```bash
# Restore only /etc
sudo restic restore <snapshot-id> --target / --include "/etc"

# Restore multiple paths
sudo restic restore <snapshot-id> --target / \
  --include "/etc" \
  --include "/home" \
  --include "/var/www"

# Exclude certain paths
sudo restic restore <snapshot-id> --target / \
  --exclude "/var/cache" \
  --exclude "/tmp"
```

### Restore with Verification

Verify restored files match the backup:

```bash
# Restore and verify
sudo restic restore <snapshot-id> --target / --verify
```

## Step 7: Post-Restore Tasks

After restoring files, complete these steps:

### 1. Restore System Configuration

```bash
# Regenerate machine ID (if needed)
sudo rm /etc/machine-id
sudo systemd-machine-id-setup

# Update hostname if changed
sudo hostnamectl set-hostname new-hostname
```

### 2. Reinstall Boot Loader (if needed)

For GRUB on BIOS systems:
```bash
sudo grub-install /dev/sda
sudo update-grub
```

For GRUB on UEFI systems:
```bash
sudo grub-install --target=x86_64-efi --efi-directory=/boot/efi
sudo update-grub
```

### 3. Restore Database (if applicable)

If you backed up database dumps:

```bash
# PostgreSQL
sudo -u postgres psql < /path/to/backup/dump.sql

# MySQL/MariaDB
mysql -u root -p < /path/to/backup/dump.sql
```

### 4. Restart Services

```bash
# Restart critical services
sudo systemctl restart networking
sudo systemctl restart sshd
sudo systemctl restart nginx  # or apache2, etc.
```

### 5. Verify Permissions

```bash
# Fix common permission issues
sudo chown -R www-data:www-data /var/www
sudo chmod 700 /root/.ssh
sudo chmod 600 /root/.ssh/authorized_keys
```

## Step 8: Reinstall Keldris Agent

After system restore, reinstall the Keldris agent:

```bash
# Download the latest agent
curl -LO https://your-keldris-server/downloads/keldris-agent

# Install
sudo mv keldris-agent /usr/local/bin/
sudo chmod +x /usr/local/bin/keldris-agent

# Configure (use your original agent credentials or register new)
keldris-agent configure \
  --server https://your-keldris-server \
  --api-key your-agent-api-key

# Start the agent
sudo systemctl enable keldris-agent
sudo systemctl start keldris-agent
```

## Troubleshooting

### Cannot Connect to Repository

1. Verify network connectivity to storage provider
2. Check credentials are correct
3. Ensure repository path is correct
4. For S3, verify bucket permissions

### Restore Fails with Permission Errors

Run the restore as root:
```bash
sudo restic restore <snapshot-id> --target /
```

### Files Missing After Restore

1. Check if files were excluded during backup
2. Verify the correct snapshot was used
3. Check if files existed at backup time

### Corrupted Repository

```bash
# Check repository integrity
restic check

# Repair if needed (use with caution)
restic repair index
restic repair snapshots
```

### Out of Disk Space

```bash
# Check available space
df -h

# Restore to external drive first
sudo restic restore <snapshot-id> --target /mnt/external/restore
```

## Recovery Time Objectives

For planning purposes:

| Data Size | Expected Restore Time (100 Mbps) |
|-----------|----------------------------------|
| 10 GB     | ~15 minutes                      |
| 100 GB    | ~2.5 hours                       |
| 1 TB      | ~24 hours                        |

Actual times depend on:
- Network bandwidth
- Storage backend performance
- Number and size of files
- CPU speed for decompression

## Quick Reference Commands

```bash
# List snapshots
restic snapshots

# Find latest snapshot for host
restic snapshots --host hostname --latest 1

# Restore latest snapshot
restic restore latest --target /

# Restore specific path from latest
restic restore latest --target / --include "/etc"

# Mount backup for browsing
restic mount /mnt/restic-mount
```

## Emergency Contacts

Keep this information updated in your DR runbook:

- **Keldris Administrator**: [contact info]
- **Storage Provider Support**: [contact info]
- **System Administrator**: [contact info]

## Testing Your Recovery Plan

Regularly test your recovery process:

1. Create a test VM or container
2. Perform a full restore
3. Verify all services start correctly
4. Document any issues found
5. Update this guide as needed

Keldris includes automated DR testing that can help validate your backups are restorable.
