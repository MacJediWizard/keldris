# Network Mount Backup Support

This guide covers how to back up network-mounted filesystems (NFS, SMB/CIFS) with Keldris.

## Overview

Keldris supports backing up data from network-mounted filesystems including:

- **NFS** (Network File System) - Common on Linux/Unix systems
- **SMB/CIFS** (Server Message Block) - Common on Windows and mixed environments

The agent automatically detects mounted network shares and reports their status to the server.

## Quick Start

### 1. Mount Your Network Share

Mount your network share as you normally would:

**NFS (Linux):**
```bash
sudo mount -t nfs server:/share /mnt/nfs-share
```

**SMB (Linux):**
```bash
sudo mount -t cifs //server/share /mnt/smb-share -o username=user,password=pass
```

**macOS:**
```bash
# NFS
sudo mount -t nfs server:/share /Volumes/nfs-share

# SMB (via Finder or command line)
mount_smbfs //user@server/share /Volumes/smb-share
```

### 2. Verify Detection

Check that the agent detects your mounts:

```bash
keldris-agent mounts
```

Example output:
```
Detected Network Mounts:
  Path: /mnt/nfs-share
    Type:   nfs
    Remote: 192.168.1.100:/data
    Status: connected

  Path: /mnt/smb-share
    Type:   cifs
    Remote: //fileserver/documents
    Status: connected
```

### 3. Create a Backup Schedule

Include the mount path in your backup schedule paths. The web UI will show available network mounts when creating schedules.

## Mount Status States

| Status | Description |
|--------|-------------|
| **connected** | Mount is accessible and ready for backup |
| **stale** | Mount exists but is unresponsive (common with NFS timeout) |
| **disconnected** | Mount point exists but the remote is unavailable |

## Handling Unavailable Mounts

When creating a schedule, you can configure what happens if a network mount is unavailable when a backup runs:

| Behavior | Description |
|----------|-------------|
| **Fail backup** (default) | The backup fails with an error. Use this when the network data is critical. |
| **Skip backup** | The backup is skipped silently. Use this for optional/non-critical network shares. |

Configure this in the web UI under **Advanced Settings > On Network Mount Unavailable**.

## Platform Support

### Linux

Network mounts are detected by parsing `/proc/mounts` for the following filesystem types:
- `nfs`, `nfs4` - NFS mounts
- `cifs`, `smbfs` - SMB/CIFS mounts

### macOS

Network mounts are detected by parsing the output of the `mount` command for:
- `nfs` - NFS mounts
- `smbfs` - SMB mounts

### Windows

Network mounts are detected using the `net use` command, which lists mapped network drives.

## Best Practices

### 1. Use Persistent Mounts

Add network mounts to `/etc/fstab` (Linux) or use automount to ensure they're available after reboot:

```bash
# /etc/fstab example for NFS
192.168.1.100:/share /mnt/nfs-share nfs defaults,_netdev 0 0

# /etc/fstab example for SMB (with credentials file)
//server/share /mnt/smb-share cifs credentials=/etc/smbcredentials,_netdev 0 0
```

The `_netdev` option ensures the mount waits for network availability.

### 2. Handle Stale Mounts

NFS mounts can become "stale" if the server is unreachable. Configure soft mounts with timeouts to prevent backups from hanging:

```bash
# Soft mount with timeout
sudo mount -t nfs -o soft,timeo=30,retrans=3 server:/share /mnt/nfs-share
```

### 3. Set Appropriate Backup Windows

Network backups can be slower and use significant bandwidth. Consider:

- Setting a **backup window** during off-peak hours
- Using **bandwidth limits** to avoid saturating your network
- Excluding busy hours using **excluded hours**

### 4. Use Skip for Non-Critical Data

If a network share contains non-critical data (e.g., media files, caches), set **On Network Mount Unavailable** to **Skip** to prevent backup failures when the share is temporarily offline.

### 5. Monitor Mount Status

The agent reports mount status with each heartbeat. Check the Agents page in the web UI to see:
- Number of detected mounts
- How many are connected vs unavailable

## Troubleshooting

### Mount not detected

1. Verify the mount is active:
   ```bash
   mount | grep -E 'nfs|cifs|smbfs'
   ```

2. Check the agent can read the mount point:
   ```bash
   ls -la /mnt/your-mount
   ```

3. Restart the agent to refresh mount detection:
   ```bash
   # Linux
   sudo systemctl restart keldris-agent

   # macOS
   launchctl unload ~/Library/LaunchAgents/io.keldris.agent.plist
   launchctl load ~/Library/LaunchAgents/io.keldris.agent.plist
   ```

### Mount shows as stale

1. Check if the NFS server is reachable:
   ```bash
   ping nfs-server
   showmount -e nfs-server
   ```

2. Try unmounting and remounting:
   ```bash
   sudo umount -f /mnt/stale-mount
   sudo mount /mnt/stale-mount
   ```

3. For persistent stale mounts, a reboot may be required.

### Backup fails with mount unavailable

1. Check mount status:
   ```bash
   keldris-agent mounts
   ```

2. If the mount is intentionally offline, update the schedule to use **Skip backup** behavior.

3. For critical mounts, investigate and fix the mount issue before the next backup.

### Permission denied on network share

Ensure the agent process has permission to read the mounted filesystem:

1. Check mount options include appropriate user/group settings
2. For SMB, verify credentials are correct
3. For NFS, check export permissions on the server

## Agent CLI Reference

### List Detected Mounts

```bash
keldris-agent mounts
```

Shows all detected network mounts with their type, remote location, and current status.

## API Reference

### Agent Heartbeat

The agent includes network mount information in its heartbeat request:

```json
{
  "network_mounts": [
    {
      "path": "/mnt/nfs-share",
      "type": "nfs",
      "remote": "192.168.1.100:/data",
      "status": "connected",
      "last_checked": "2024-01-20T10:30:00Z"
    }
  ]
}
```

### Schedule Configuration

Schedules include mount behavior configuration:

```json
{
  "name": "Network Backup",
  "paths": ["/mnt/nfs-share", "/home"],
  "on_mount_unavailable": "skip"
}
```

Valid values for `on_mount_unavailable`:
- `fail` - Fail the backup if any network path is unavailable (default)
- `skip` - Skip the backup if any network path is unavailable
