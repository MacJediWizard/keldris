# API Rate Limiting

The Keldris API implements rate limiting to protect against abuse and ensure fair usage across all clients.

## Rate Limit Headers

All API responses include the following headers to help clients manage their request rate:

| Header | Description |
|--------|-------------|
| `X-RateLimit-Limit` | Maximum number of requests allowed in the time window |
| `X-RateLimit-Remaining` | Number of requests remaining in the current window |
| `X-RateLimit-Reset` | Unix timestamp (seconds) when the rate limit window resets |

When a rate limit is exceeded, the API responds with:
- HTTP status code `429 Too Many Requests`
- `Retry-After` header containing seconds to wait before retrying
- JSON body with rate limit details

## Response When Rate Limited

```json
{
  "error": "rate limit exceeded",
  "retry_after": 30,
  "rate_limit": {
    "limit": 100,
    "remaining": 0,
    "reset": 1706028000
  }
}
```

## Default Rate Limits

The default rate limit is configured per server deployment. Common defaults:

| Environment | Requests | Period |
|-------------|----------|--------|
| Production  | 100      | 1 minute |
| Development | 1000     | 1 minute |

## Per-Endpoint Configuration

Administrators can configure different rate limits for specific API endpoints. This allows:

- Higher limits for lightweight endpoints (e.g., health checks)
- Lower limits for resource-intensive operations (e.g., backups, restores)
- Custom limits for specific use cases

### Configuration Example

```go
endpointConfigs := []middleware.EndpointRateLimitConfig{
    {Pattern: "/api/v1/backups/*", Requests: 10, Period: "1m"},
    {Pattern: "/api/v1/health", Requests: 1000, Period: "1m"},
}
```

## Monitoring Rate Limits

Administrators can monitor rate limit statistics through:

1. **Admin Dashboard**: Navigate to Admin > Rate Limits to view:
   - Default rate limit configuration
   - Per-endpoint configurations
   - Client statistics (requests, rejections by IP)
   - Rejection rates

2. **API Endpoint**: `GET /api/v1/admin/rate-limits`
   - Requires admin authentication
   - Returns JSON with rate limit statistics

### Dashboard Statistics Response

```json
{
  "default_limit": 100,
  "default_period": "1m0s",
  "endpoint_configs": [
    {
      "pattern": "/api/v1/backups/*",
      "limit": 10,
      "period": "1m0s"
    }
  ],
  "client_stats": [
    {
      "client_ip": "192.168.1.100",
      "total_requests": 150,
      "rejected_count": 5,
      "last_request": "2024-01-23T10:30:00Z"
    }
  ],
  "total_requests": 1500,
  "total_rejected": 25
}
```

## Client Best Practices

1. **Monitor Headers**: Always check `X-RateLimit-Remaining` to proactively manage request rate.

2. **Handle 429 Responses**: Implement exponential backoff or respect the `Retry-After` header.

3. **Batch Operations**: Where possible, batch multiple operations into single requests.

4. **Cache Responses**: Cache API responses to reduce unnecessary requests.

### Example: Handling Rate Limits (JavaScript)

```javascript
async function fetchWithRateLimit(url, options = {}) {
  const response = await fetch(url, options);

  // Log rate limit info
  const limit = response.headers.get('X-RateLimit-Limit');
  const remaining = response.headers.get('X-RateLimit-Remaining');
  const reset = response.headers.get('X-RateLimit-Reset');
  console.log(`Rate limit: ${remaining}/${limit}, resets at ${new Date(reset * 1000)}`);

  // Handle rate limit exceeded
  if (response.status === 429) {
    const retryAfter = parseInt(response.headers.get('Retry-After') || '60', 10);
    console.log(`Rate limited. Retrying in ${retryAfter} seconds...`);
    await new Promise(resolve => setTimeout(resolve, retryAfter * 1000));
    return fetchWithRateLimit(url, options);
  }

  return response;
}
```

### Example: Handling Rate Limits (Go)

```go
func makeAPIRequest(url string) (*http.Response, error) {
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }

    // Check rate limit headers
    remaining := resp.Header.Get("X-RateLimit-Remaining")
    log.Printf("Rate limit remaining: %s", remaining)

    // Handle rate limit exceeded
    if resp.StatusCode == http.StatusTooManyRequests {
        retryAfter := resp.Header.Get("Retry-After")
        seconds, _ := strconv.Atoi(retryAfter)
        if seconds == 0 {
            seconds = 60
        }
        log.Printf("Rate limited. Retrying in %d seconds...", seconds)
        time.Sleep(time.Duration(seconds) * time.Second)
        return makeAPIRequest(url)
    }

    return resp, nil
}
```

## Troubleshooting

### Common Issues

1. **Frequent 429 Errors**
   - Check if your application is making unnecessary repeated requests
   - Consider implementing request caching
   - Contact your administrator if limits need adjustment

2. **Incorrect Client IP**
   - Rate limiting uses the client IP from `X-Forwarded-For` or direct connection
   - Ensure your proxy configuration correctly forwards client IPs

3. **Rate Limit Not Resetting**
   - Check the `X-RateLimit-Reset` header for the actual reset time
   - Rate limits are per-IP, ensure requests aren't coming from different IPs
