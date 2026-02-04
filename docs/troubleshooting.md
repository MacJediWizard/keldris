# Troubleshooting Guide

This guide helps you diagnose and resolve common issues with Keldris.

## Server Issues

### Server won't start

**Symptoms:**
- Server exits immediately after starting
- Error messages in logs about configuration

**Solutions:**

1. **Check database connectivity:**
   ```bash
   # Test PostgreSQL connection
   psql $DATABASE_URL -c "SELECT 1"
   ```

2. **Verify environment variables:**
   ```bash
   # Check required variables are set
   echo $DATABASE_URL
   echo $OIDC_ISSUER_URL
   echo $SESSION_SECRET
   ```

3. **Check port availability:**
   ```bash
   # Ensure port 8080 is not in use
   lsof -i :8080
   ```

4. **Review logs:**
   ```bash
   # Docker
   docker-compose logs keldris

   # Systemd
   journalctl -u keldris -f
   ```

### Database connection errors

**Symptoms:**
- "connection refused" errors
- "authentication failed" errors

**Solutions:**

1. **Verify PostgreSQL is running:**
   ```bash
   systemctl status postgresql
   ```

2. **Check pg_hba.conf allows connections:**
   ```
   # /etc/postgresql/15/main/pg_hba.conf
   host    keldris    keldris    127.0.0.1/32    scram-sha-256
   ```

3. **Test connection manually:**
   ```bash
   psql -h localhost -U keldris -d keldris
   ```

### OIDC authentication fails

**Symptoms:**
- "invalid_client" errors
- Redirect loops
- "OIDC provider unreachable" in health check

**Solutions:**

1. **Verify OIDC configuration:**
   - Check `OIDC_ISSUER_URL` is correct
   - Verify client ID and secret match provider settings
   - Ensure redirect URI is registered with provider

2. **Test OIDC endpoint:**
   ```bash
   curl $OIDC_ISSUER_URL/.well-known/openid-configuration
   ```

3. **Check callback URL:**
   - Must match exactly: `{SERVER_URL}/auth/callback`
   - Ensure HTTPS if using secure cookies

## Agent Issues

### Agent won't connect

**Symptoms:**
- "connection refused" or "timeout" errors
- Agent shows as offline in UI

**Solutions:**

1. **Verify server URL:**
   ```bash
   keldris-agent status
   # Check server URL in output
   ```

2. **Test network connectivity:**
   ```bash
   curl -v https://your-keldris-server.com/health
   ```

3. **Check firewall rules:**
   - Agent needs outbound HTTPS (port 443)
   - If custom port, ensure it's allowed

4. **Verify API key:**
   - Regenerate if expired or compromised
   - Check key has correct permissions

### Agent registration fails

**Symptoms:**
- "Invalid registration code" error
- "API key invalid" error

**Solutions:**

1. **Check registration code:**
   - Codes expire after 10 minutes
   - Generate a new code from the UI

2. **Verify server time:**
   - Ensure server and agent clocks are synchronized
   - Use NTP for time synchronization

3. **Check agent version:**
   - Ensure agent version is compatible with server
   - Update to latest version if needed

### Agent service won't start

**Symptoms:**
- Service fails to start
- Permission denied errors

**Solutions:**

**Linux:**
```bash
# Check service status
systemctl status keldris-agent

# View detailed logs
journalctl -u keldris-agent -n 100

# Fix permissions
sudo chmod 755 /usr/local/bin/keldris-agent
```

**macOS:**
```bash
# Check if loaded
launchctl list | grep keldris

# View logs
tail -100 ~/.config/keldris/agent.log

# Reload
launchctl unload ~/Library/LaunchAgents/io.keldris.agent.plist
launchctl load ~/Library/LaunchAgents/io.keldris.agent.plist
```

**Windows (PowerShell as Admin):**
```powershell
# Check service
Get-Service KeldrisAgent

# View event log
Get-EventLog -LogName Application -Source KeldrisAgent -Newest 50
```

## Backup Issues

### Backup fails with "repository not found"

**Solutions:**

1. **Verify repository configuration:**
   - Check storage credentials are correct
   - Ensure bucket/path exists

2. **Test repository access:**
   ```bash
   # The agent can test connectivity
   keldris-agent test-repo --repository-id <id>
   ```

3. **For S3 repositories:**
   - Check bucket region is correct
   - Verify IAM permissions

### Backup is slow

**Solutions:**

1. **Check network bandwidth:**
   - Run speed test to storage endpoint
   - Consider local caching

2. **Optimize exclude patterns:**
   - Exclude temporary files, logs, cache directories
   - Add `node_modules`, `.git`, etc.

3. **Adjust concurrency:**
   ```yaml
   # agent.yaml
   backup:
     read_concurrency: 4  # Increase for SSD
   ```

4. **Use compression:**
   ```yaml
   backup:
     compression: auto  # or max for slow networks
   ```

### Backup hangs

**Solutions:**

1. **Check for locked files:**
   - Windows: VSS may be required for open files
   - Close applications locking files if possible

2. **Increase timeout:**
   - Large files may need more time
   - Check storage backend responsiveness

3. **Check disk space:**
   - Temporary space needed for compression
   - Ensure adequate space on agent

## Restore Issues

### Restore fails with permission errors

**Solutions:**

1. **Run with appropriate privileges:**
   - Linux/macOS: Use `sudo` for system files
   - Windows: Run as Administrator

2. **Check target directory permissions:**
   ```bash
   ls -la /target/directory
   chown -R user:group /target/directory
   ```

3. **Disable SELinux temporarily (if applicable):**
   ```bash
   setenforce 0  # Temporary
   # Investigate proper SELinux policy after
   ```

### Restored files have wrong ownership

**Solutions:**

1. **Restore with correct user:**
   - Run restore as the intended file owner
   - Or fix ownership after restore

2. **Use restore options:**
   - Some restores allow specifying owner/group

## Storage Issues

### S3: Access Denied

**Solutions:**

1. **Check IAM policy:**
   ```json
   {
     "Version": "2012-10-17",
     "Statement": [
       {
         "Effect": "Allow",
         "Action": [
           "s3:GetObject",
           "s3:PutObject",
           "s3:DeleteObject",
           "s3:ListBucket"
         ],
         "Resource": [
           "arn:aws:s3:::your-bucket",
           "arn:aws:s3:::your-bucket/*"
         ]
       }
     ]
   }
   ```

2. **Verify bucket policy allows access**

3. **Check region configuration matches bucket region**

### B2: Connection errors

**Solutions:**

1. **Verify application key permissions:**
   - Needs read and write access to bucket

2. **Check key hasn't expired**

3. **Ensure bucket name is correct (case-sensitive)**

### Local storage: No space left

**Solutions:**

1. **Check disk usage:**
   ```bash
   df -h /path/to/repo
   ```

2. **Run repository maintenance:**
   - Prune old snapshots
   - Run garbage collection

3. **Move to larger storage**

## Performance Issues

### Web UI is slow

**Solutions:**

1. **Check database performance:**
   ```sql
   -- Check for slow queries
   SELECT * FROM pg_stat_activity WHERE state = 'active';
   ```

2. **Add database indexes if missing**

3. **Increase database connection pool:**
   ```env
   DB_MAX_OPEN_CONNS=50
   ```

### High memory usage

**Solutions:**

1. **Adjust Go garbage collection:**
   ```env
   GOGC=50  # More aggressive GC
   ```

2. **Reduce concurrent operations**

3. **Check for memory leaks in logs**

## Getting Help

If you can't resolve your issue:

1. **Check logs thoroughly:**
   - Server logs
   - Agent logs
   - Database logs

2. **Enable debug logging:**
   ```env
   LOG_LEVEL=debug
   ```

3. **Collect support bundle:**
   - Go to **Settings > Support**
   - Download diagnostic bundle
   - Excludes sensitive data

4. **Open an issue:**
   - https://github.com/MacJediWizard/keldris/issues
   - Include logs and reproduction steps

## Common Error Messages

| Error | Cause | Solution |
|-------|-------|----------|
| `connection refused` | Server not running or wrong port | Check server status and port |
| `certificate verify failed` | TLS certificate issue | Check certificates or use `--insecure` for testing |
| `repository does not exist` | Wrong path or not initialized | Verify repository path and initialize if new |
| `wrong password` | Incorrect repository password | Use correct password |
| `parent snapshot not found` | Repository corruption | Run repository check and repair |
| `pack file damaged` | Storage corruption | Restore from backup or run repair |
