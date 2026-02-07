# Proxy Configuration

Keldris Agent supports HTTP, HTTPS, and SOCKS5 proxies for all network connections including:

- Server API communication
- Update checks and downloads
- Notification services (webhooks, Slack, Discord, Teams, PagerDuty)
- Integration connections (Komodo)

## Configuration

### Using CLI Commands

```bash
# Set HTTP proxy
keldris-agent config set-proxy --http http://proxy:8080

# Set HTTPS proxy (often the same as HTTP proxy)
keldris-agent config set-proxy --https http://proxy:8080

# Set SOCKS5 proxy with authentication
keldris-agent config set-proxy --socks5 socks5://username:password@proxy:1080

# Set hosts to bypass proxy
keldris-agent config set-proxy --no-proxy "localhost,127.0.0.1,.internal.com"

# Set multiple options at once
keldris-agent config set-proxy \
  --http http://proxy:8080 \
  --https http://proxy:8080 \
  --no-proxy "localhost,127.0.0.1,.internal.com"

# View current proxy configuration
keldris-agent config show

# Clear all proxy settings
keldris-agent config clear-proxy
```

### Direct Configuration File

Edit `~/.keldris/config.yml`:

```yaml
server_url: https://keldris.example.com
api_key: your-api-key
hostname: my-server
auto_check_update: true

proxy:
  http_proxy: http://proxy.company.com:8080
  https_proxy: http://proxy.company.com:8080
  no_proxy: localhost,127.0.0.1,.company.internal
  socks5_proxy: ""
```

## Proxy Types

### HTTP/HTTPS Proxy

Standard HTTP proxies that forward HTTP and HTTPS traffic. Most corporate proxies use this type.

```yaml
proxy:
  http_proxy: http://proxy:8080
  https_proxy: http://proxy:8080
```

With authentication:

```yaml
proxy:
  http_proxy: http://user:password@proxy:8080
  https_proxy: http://user:password@proxy:8080
```

### SOCKS5 Proxy

SOCKS5 proxies provide protocol-agnostic proxying. When a SOCKS5 proxy is configured, it takes precedence over HTTP/HTTPS proxies.

```yaml
proxy:
  socks5_proxy: socks5://proxy:1080
```

With authentication:

```yaml
proxy:
  socks5_proxy: socks5://user:password@proxy:1080
```

### No-Proxy List

Hosts that should bypass the proxy. Supports:

- Exact hostname matching: `example.com`
- Domain suffix matching: `.example.com` (matches `api.example.com`, `www.example.com`)
- Subdomain matching: `example.com` (also matches `sub.example.com`)
- Wildcard: `*` (bypasses all hosts)

```yaml
proxy:
  no_proxy: localhost,127.0.0.1,.internal.company.com,10.0.0.0/8
```

## Testing Proxy Configuration

After configuring proxy settings, test the connection:

```bash
# Test with default URL (https://www.google.com)
keldris-agent config test-proxy

# Test with your Keldris server
keldris-agent config test-proxy --url https://keldris.example.com/health

# Test with a specific endpoint
keldris-agent config test-proxy --url https://api.github.com
```

## Environment Variables for Restic

For backup operations using Restic, proxy settings are passed via environment variables. You can add these to your ResticConfig Env map:

```yaml
# Example backend configuration with proxy env vars
env:
  HTTP_PROXY: http://proxy:8080
  HTTPS_PROXY: http://proxy:8080
  NO_PROXY: localhost,127.0.0.1
```

Restic natively supports these standard proxy environment variables:

- `HTTP_PROXY` or `http_proxy`
- `HTTPS_PROXY` or `https_proxy`
- `NO_PROXY` or `no_proxy`

## Troubleshooting

### Connection Timeout

If connections timeout through the proxy:

1. Verify proxy URL is correct
2. Check if proxy requires authentication
3. Ensure the agent can reach the proxy server
4. Try increasing timeout values

### Authentication Failures

If you receive authentication errors:

1. Verify username and password are URL-encoded if they contain special characters
2. Check if the proxy accepts the authentication method used
3. Try testing with a browser or curl first

### SOCKS5 vs HTTP Proxy

If SOCKS5 is configured, it takes precedence over HTTP/HTTPS proxy settings. To use HTTP proxy instead:

```bash
keldris-agent config set-proxy --socks5 ""
```

### Viewing Debug Information

Check the status command output for proxy information:

```bash
keldris-agent status
```

Output includes:
- Server URL
- Hostname
- Proxy configuration (if set)
- Connection status

## Security Considerations

- Proxy credentials are stored in the configuration file with restricted permissions (0600)
- When displaying proxy settings, passwords are masked
- Consider using SOCKS5 with authentication for enhanced security
- Review your no-proxy list to ensure sensitive internal traffic stays internal
