# Heartbeat System Testing Guide

## Overview

The Keldris heartbeat system provides telemetry collection, license validation, and remote management capabilities. Every Keldris instance (free and paid) participates in the heartbeat cycle.

## Architecture

```
Keldris Instance                    License Server
┌─────────────┐                    ┌──────────────┐
│  Validator   │───── register ───▶│  Instances    │
│             │───── heartbeat ──▶│  Handler      │
│             │───── activate ───▶│  Licenses     │
│             │───── validate ───▶│  Handler      │
│             │◀── entitlement ──│              │
│             │◀── kill/downgrade│              │
└─────────────┘                    └──────────────┘
```

### Heartbeat Cycle
- **Interval**: Every 6 hours (all tiers)
- **Validation**: Every 24 hours (paid tiers only)
- **Grace Period**: 30 days for paid tiers when server is unreachable

### Data Sent in Heartbeat
- Instance ID and product identifier
- Agent count, user count, organization count
- Feature usage (which premium features were accessed)
- Server version and uptime
- Entitlement token hash (for integrity verification)

### Responses
- `action: "none"` — Normal operation
- `action: "downgrade"` — Admin requested tier reduction
- `action: "kill"` — Admin requested immediate free-tier fallback
- `config.feature_refresh_token` — Rotating token for feature access

## Constants

| Constant | Value | Description |
|----------|-------|-------------|
| `HeartbeatInterval` | 6 hours | How often heartbeats are sent |
| `ValidationInterval` | 24 hours | How often paid licenses are re-validated |
| `RetryInterval` | 1 hour | Retry interval when server is unreachable |
| `GracePeriod` | 30 days | How long paid tiers run offline |
| `EntitlementTokenExpiry` | 24 hours | Lifetime of signed entitlement tokens |

## Prerequisites

1. **License server running** on a known URL (default: `http://localhost:8081`)
2. **PostgreSQL database** configured for the license server
3. **jq** installed for JSON parsing in test scripts
4. **curl** available for HTTP requests

## Automated Tests

Run the test script:

```bash
# Using default server URL (localhost:8081)
./scripts/test-heartbeat.sh

# Using custom server URL
./scripts/test-heartbeat.sh https://license.example.com

# With a test license key for activation/validation tests
TEST_LICENSE_KEY="your-key-here" ./scripts/test-heartbeat.sh
```

### Test Cases

| # | Test | What It Verifies |
|---|------|-----------------|
| 1 | Registration | Instance registers with license server |
| 2 | Activation | License key activates successfully |
| 3 | Validation | Active license validates correctly |
| 4 | Normal Heartbeat | Heartbeat returns `status: ok`, `action: none`, refresh token |
| 5 | Anomaly: Premium w/o Entitlement | Free tier reporting premium feature usage creates anomaly |
| 6 | Anomaly: Tier Mismatch | Free instance claiming enterprise tier creates anomaly |
| 7 | Kill Switch | Admin kill action propagates via heartbeat |
| 8 | Downgrade | Admin downgrade action propagates via heartbeat |
| 9 | Token Parse | Heartbeat accepts entitlement token hash |
| 10 | Deactivation | License deactivates on shutdown |

## Manual Testing with curl

### Register an Instance

```bash
curl -X POST http://localhost:8081/api/v1/instances/register \
  -H "Content-Type: application/json" \
  -d '{
    "instance_id": "test-001",
    "product": "keldris",
    "hostname": "dev-machine",
    "server_version": "1.0.0",
    "tier": "free",
    "os": "linux",
    "arch": "amd64"
  }'
```

### Send a Heartbeat

```bash
curl -X POST http://localhost:8081/api/v1/instances/heartbeat \
  -H "Content-Type: application/json" \
  -d '{
    "instance_id": "test-001",
    "product": "keldris",
    "metrics": {
      "agent_count": 5,
      "user_count": 2,
      "org_count": 1,
      "feature_usage": ["audit_logs"],
      "server_version": "1.0.0",
      "uptime_hours": 48.5
    },
    "reported_tier": "pro",
    "has_valid_entitlement": true
  }'
```

### Activate a License

```bash
curl -X POST http://localhost:8081/api/v1/licenses/activate \
  -H "Content-Type: application/json" \
  -d '{
    "license_key": "YOUR_LICENSE_KEY",
    "instance_id": "test-001",
    "product": "keldris",
    "hostname": "dev-machine",
    "server_version": "1.0.0"
  }'
```

### Validate a License

```bash
curl -X POST http://localhost:8081/api/v1/licenses/validate \
  -H "Content-Type: application/json" \
  -d '{
    "license_key": "YOUR_LICENSE_KEY",
    "instance_id": "test-001",
    "product": "keldris"
  }'
```

### Deactivate a License

```bash
curl -X POST http://localhost:8081/api/v1/licenses/deactivate \
  -H "Content-Type: application/json" \
  -d '{
    "license_key": "YOUR_LICENSE_KEY",
    "instance_id": "test-001",
    "product": "keldris"
  }'
```

## Anomaly Types

The license server detects these anomalies during heartbeat processing:

| Type | Trigger | Severity |
|------|---------|----------|
| `no_entitlement_premium_usage` | Premium features used without valid entitlement | High |
| `tier_mismatch` | Instance claims higher tier than registered | High |
| `unlicensed_premium_usage` | Premium features used with no active license | High |
| `limit_exceeded` | Resource counts exceed license limits | Medium |
| `stale_instance` | No heartbeat received for 48+ hours | Low |

### Viewing Anomalies

```bash
curl http://localhost:8081/api/v1/admin/anomalies
```

### Resolving Anomalies

```bash
curl -X POST http://localhost:8081/api/v1/admin/anomalies/{id}/resolve
```

## Kill Switch

Administrators can remotely control instances via the kill switch:

### Set Kill Action

```bash
# Downgrade to free tier
curl -X POST http://localhost:8081/api/v1/admin/instances/{id}/action \
  -H "Content-Type: application/json" \
  -d '{"action": "downgrade"}'

# Kill (force free tier + set killed flag)
curl -X POST http://localhost:8081/api/v1/admin/instances/{id}/action \
  -H "Content-Type: application/json" \
  -d '{"action": "kill"}'

# Clear action
curl -X POST http://localhost:8081/api/v1/admin/instances/{id}/action \
  -H "Content-Type: application/json" \
  -d '{"action": "none"}'
```

### Kill Switch Behavior

1. Admin sets action on instance via license server admin API
2. Next heartbeat from instance receives the action in response
3. **Downgrade**: Instance reverts to free tier, clears entitlement
4. **Kill**: Instance reverts to free tier, sets `killed` flag preventing re-activation

## Grace Period

When the license server is unreachable:

1. **Valid license key**: Verified locally using Ed25519 signature
2. **Expired key**: 30-day grace period starts
3. **Grace period running**: Instance continues on paid tier
4. **Grace period expired**: Automatic downgrade to free tier

Timeline:
```
Day 0:  Server unreachable → local key verification starts
Day 1-30: Grace period (full features continue)
Day 30: Grace period expires → downgrade to Free
```

## Troubleshooting

### Heartbeat not sending
- Check `LICENSE_SERVER_URL` environment variable
- Verify network connectivity to license server
- Check server logs for `heartbeat failed` messages

### Entitlement token issues
- Tokens expire after 24 hours; validation cycle renews them
- Check `AIRGAP_PUBLIC_KEY` matches the license server's signing key
- Look for `failed to parse entitlement token` in logs

### Grace period questions
- Grace period only applies to paid tiers (not free)
- Grace period resets when server becomes reachable again
- Air-gap mode uses local license validation (no grace period needed)

### Anomaly false positives
- Feature usage tracking may report features accessed during the grace period
- Resolve false positive anomalies via the admin API
