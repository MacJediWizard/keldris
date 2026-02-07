# Prometheus Metrics

Keldris exposes metrics in Prometheus exposition format at the `/metrics` endpoint. This allows you to monitor backup operations, agent health, and storage utilization.

## Endpoint

```
GET /metrics
```

No authentication is required for this endpoint.

## Available Metrics

### Backup Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `keldris_backup_total` | counter | Total number of backups |
| `keldris_backup_status_total{status="..."}` | counter | Total number of backups by status (completed, failed, running, canceled) |
| `keldris_backup_duration_seconds` | histogram | Histogram of backup duration in seconds |
| `keldris_backup_size_bytes` | gauge | Total size of completed backups in bytes |

The histogram uses the following bucket boundaries (in seconds):
- 60 (1 minute)
- 300 (5 minutes)
- 600 (10 minutes)
- 1800 (30 minutes)
- 3600 (1 hour)
- 7200 (2 hours)
- 14400 (4 hours)
- 28800 (8 hours)

### Agent Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `keldris_agents_total` | gauge | Total number of registered agents |
| `keldris_agents_online` | gauge | Number of online (active) agents |

### Storage Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `keldris_storage_used_bytes` | gauge | Total storage used in bytes |

### Server Health Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `keldris_info{component="server"}` | gauge | Server information (always 1) |
| `keldris_up{component="database"}` | gauge | Database health status (1 = healthy, 0 = unhealthy) |

### Database Pool Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `keldris_db_connections_total` | gauge | Total number of connections in the pool |
| `keldris_db_connections_acquired` | gauge | Number of currently acquired connections |
| `keldris_db_connections_idle` | gauge | Number of idle connections |
| `keldris_db_connections_max` | gauge | Maximum number of connections in the pool |
| `keldris_db_connections_constructing` | gauge | Number of connections being constructed |
| `keldris_db_acquire_empty_total` | counter | Total number of acquire attempts that had to wait |
| `keldris_db_acquire_canceled_total` | counter | Total number of acquire attempts that were canceled |
| `keldris_db_lifetime_destroy_total` | counter | Total connections destroyed due to max lifetime |
| `keldris_db_idle_destroy_total` | counter | Total connections destroyed due to max idle time |

## Prometheus Scrape Configuration

Add the following to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'keldris'
    scrape_interval: 30s
    scrape_timeout: 10s
    static_configs:
      - targets: ['keldris-server:8080']
    metrics_path: /metrics
    scheme: http
```

For HTTPS with self-signed certificates:

```yaml
scrape_configs:
  - job_name: 'keldris'
    scrape_interval: 30s
    scrape_timeout: 10s
    static_configs:
      - targets: ['keldris-server:8443']
    metrics_path: /metrics
    scheme: https
    tls_config:
      insecure_skip_verify: true
```

With service discovery (Kubernetes example):

```yaml
scrape_configs:
  - job_name: 'keldris'
    scrape_interval: 30s
    kubernetes_sd_configs:
      - role: service
        namespaces:
          names:
            - keldris
    relabel_configs:
      - source_labels: [__meta_kubernetes_service_name]
        action: keep
        regex: keldris-server
      - source_labels: [__meta_kubernetes_service_port_name]
        action: keep
        regex: http
```

## Alerting Rules

Create a file `keldris-alerts.yml` in your Prometheus rules directory:

```yaml
groups:
  - name: keldris
    rules:
      # Alert when no agents are online
      - alert: KeldrisNoAgentsOnline
        expr: keldris_agents_online == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "No Keldris agents online"
          description: "All backup agents are offline. Backups cannot run."

      # Alert when agent count drops significantly
      - alert: KeldrisAgentsDegraded
        expr: keldris_agents_online < (keldris_agents_total * 0.5)
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Less than 50% of agents online"
          description: "{{ $value }} agents online out of {{ printf \"keldris_agents_total\" | query | first | value }} total."

      # Alert on backup failures
      - alert: KeldrisBackupFailures
        expr: increase(keldris_backup_status_total{status="failed"}[1h]) > 5
        for: 0m
        labels:
          severity: warning
        annotations:
          summary: "High number of backup failures"
          description: "{{ $value }} backup failures in the last hour."

      # Alert when backups are taking too long
      - alert: KeldrisSlowBackups
        expr: histogram_quantile(0.95, rate(keldris_backup_duration_seconds_bucket[1h])) > 7200
        for: 30m
        labels:
          severity: warning
        annotations:
          summary: "Backups taking longer than expected"
          description: "95th percentile backup duration is {{ $value | humanizeDuration }}."

      # Alert on database connection pool exhaustion
      - alert: KeldrisDBPoolExhausted
        expr: keldris_db_connections_acquired / keldris_db_connections_max > 0.9
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Database connection pool near capacity"
          description: "{{ $value | humanizePercentage }} of database connections in use."

      # Alert when database is unhealthy
      - alert: KeldrisDBUnhealthy
        expr: keldris_up{component="database"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Keldris database unhealthy"
          description: "Database connection has failed."

      # Alert on rapid storage growth
      - alert: KeldrisStorageGrowth
        expr: deriv(keldris_storage_used_bytes[1d]) > 10737418240
        for: 1h
        labels:
          severity: warning
        annotations:
          summary: "Rapid storage growth detected"
          description: "Storage is growing at {{ $value | humanize1024 }}/day."

      # Alert when no backups have completed recently
      - alert: KeldrisNoRecentBackups
        expr: increase(keldris_backup_status_total{status="completed"}[24h]) == 0
        for: 0m
        labels:
          severity: warning
        annotations:
          summary: "No successful backups in 24 hours"
          description: "No backups have completed successfully in the last 24 hours."
```

## Grafana Dashboard

Import the following dashboard JSON for a pre-built visualization:

```json
{
  "title": "Keldris Backup Monitoring",
  "panels": [
    {
      "title": "Agents Online",
      "type": "stat",
      "targets": [
        {
          "expr": "keldris_agents_online",
          "legendFormat": "Online"
        }
      ]
    },
    {
      "title": "Agent Status",
      "type": "gauge",
      "targets": [
        {
          "expr": "keldris_agents_online / keldris_agents_total * 100",
          "legendFormat": "% Online"
        }
      ],
      "options": {
        "minValue": 0,
        "maxValue": 100
      }
    },
    {
      "title": "Backup Success Rate",
      "type": "stat",
      "targets": [
        {
          "expr": "keldris_backup_status_total{status=\"completed\"} / keldris_backup_total * 100",
          "legendFormat": "Success %"
        }
      ]
    },
    {
      "title": "Backups by Status",
      "type": "piechart",
      "targets": [
        {
          "expr": "keldris_backup_status_total",
          "legendFormat": "{{ status }}"
        }
      ]
    },
    {
      "title": "Backup Duration (95th percentile)",
      "type": "timeseries",
      "targets": [
        {
          "expr": "histogram_quantile(0.95, rate(keldris_backup_duration_seconds_bucket[5m]))",
          "legendFormat": "p95 duration"
        }
      ]
    },
    {
      "title": "Storage Used",
      "type": "stat",
      "targets": [
        {
          "expr": "keldris_storage_used_bytes",
          "legendFormat": "Used"
        }
      ],
      "fieldConfig": {
        "defaults": {
          "unit": "bytes"
        }
      }
    },
    {
      "title": "Database Connections",
      "type": "timeseries",
      "targets": [
        {
          "expr": "keldris_db_connections_acquired",
          "legendFormat": "Acquired"
        },
        {
          "expr": "keldris_db_connections_idle",
          "legendFormat": "Idle"
        },
        {
          "expr": "keldris_db_connections_max",
          "legendFormat": "Max"
        }
      ]
    }
  ]
}
```

## Troubleshooting

### Metrics not appearing

1. Check that the server is running and accessible:
   ```bash
   curl http://keldris-server:8080/metrics
   ```

2. Verify Prometheus can reach the target:
   ```bash
   # Check Prometheus targets page
   curl http://prometheus:9090/api/v1/targets
   ```

3. Check for firewall rules blocking port 8080

### Stale metrics

Metrics are cached for 15 seconds to reduce database load. If you need more frequent updates, consider adjusting your scrape interval.

### High cardinality issues

The current implementation does not include high-cardinality labels (like agent IDs or backup IDs). If you need per-agent metrics, consider using the dashboard API endpoints instead.
