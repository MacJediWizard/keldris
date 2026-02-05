# Security Headers Configuration

Keldris implements comprehensive HTTP security headers to protect against common web vulnerabilities. These headers are automatically applied to all API responses.

## Security Headers Applied

| Header | Default Value | Purpose |
|--------|---------------|---------|
| `X-Frame-Options` | `DENY` | Prevents clickjacking by blocking iframe embedding |
| `X-Content-Type-Options` | `nosniff` | Prevents MIME-type sniffing attacks |
| `Content-Security-Policy` | Restrictive policy (see below) | Controls resource loading to prevent XSS |
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains` | Enforces HTTPS (production only) |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Controls referrer information leakage |
| `Permissions-Policy` | Restrictive (see below) | Disables dangerous browser features |
| `Cross-Origin-Opener-Policy` | `same-origin` | Isolates browsing context |
| `Cross-Origin-Resource-Policy` | `same-origin` | Restricts resource loading to same origin |
| `Cross-Origin-Embedder-Policy` | `require-corp` | Requires CORP/CORS for cross-origin resources |
| `X-XSS-Protection` | `1; mode=block` | Legacy XSS protection for older browsers |
| `X-Permitted-Cross-Domain-Policies` | `none` | Prevents Adobe Flash cross-domain requests |

## Default Content-Security-Policy

The default production CSP:

```
default-src 'self';
script-src 'self';
style-src 'self' 'unsafe-inline';
img-src 'self' data: https:;
font-src 'self';
connect-src 'self';
frame-ancestors 'none';
form-action 'self';
base-uri 'self';
object-src 'none';
upgrade-insecure-requests
```

## Default Permissions-Policy

Restrictive permissions that disable potentially dangerous browser features:

```
accelerometer=(), ambient-light-sensor=(), autoplay=(), battery=(),
camera=(), display-capture=(), document-domain=(), encrypted-media=(),
fullscreen=(self), geolocation=(), gyroscope=(), magnetometer=(),
microphone=(), midi=(), payment=(), picture-in-picture=(),
publickey-credentials-get=(), screen-wake-lock=(), usb=(), xr-spatial-tracking=()
```

## Configuration

### Production Configuration

For production deployments, use the default secure configuration:

```go
import "github.com/MacJediWizard/keldris/internal/api/middleware"

cfg := api.ProductionConfig()
// SecurityHeaders is automatically set to DefaultSecurityHeadersConfig()
```

### Development Configuration

Development uses a more permissive configuration to support hot reload and dev tools:

```go
cfg := api.DefaultConfig()
// SecurityHeaders is automatically set to DevelopmentSecurityHeadersConfig()
```

Key differences in development mode:
- HSTS is disabled (no HTTPS required)
- CSP allows `'unsafe-inline'` and `'unsafe-eval'` for development tools
- WebSocket connections allowed for hot reload
- Cross-Origin-Embedder-Policy is `credentialless` instead of `require-corp`

### Custom Configuration

For white-label deployments or specific requirements, customize the security headers:

```go
cfg := middleware.SecurityHeadersConfig{
    ContentSecurityPolicy:     "default-src 'self'; script-src 'self' https://cdn.example.com",
    FrameOptions:              "SAMEORIGIN", // Allow same-origin iframes
    ContentTypeOptions:        "nosniff",
    StrictTransportSecurity:   "max-age=63072000; includeSubDomains; preload",
    EnableHSTS:                true,
    ReferrerPolicy:            "strict-origin-when-cross-origin",
    PermissionsPolicy:         "camera=(), microphone=(), geolocation=()",
    CrossOriginOpenerPolicy:   "same-origin",
    CrossOriginResourcePolicy: "same-origin",
    CrossOriginEmbedderPolicy: "require-corp",
}
```

### White-Label CSP Configuration

For white-label deployments that need to load resources from additional origins, use the `AdditionalCSPDirectives` field to extend the default CSP:

```go
cfg := middleware.DefaultSecurityHeadersConfig()
cfg.AdditionalCSPDirectives = map[string]string{
    "script-src": "https://cdn.whitelabel.com https://analytics.whitelabel.com",
    "img-src":    "https://assets.whitelabel.com",
    "font-src":   "https://fonts.whitelabel.com",
    "frame-src":  "'self' https://embed.whitelabel.com",
}
```

This merges the additional sources with the default CSP, so the resulting `script-src` would be:
```
script-src 'self' https://cdn.whitelabel.com https://analytics.whitelabel.com
```

## Testing Security Headers

### Test Endpoint

Keldris provides a test endpoint to verify security headers are configured correctly:

```bash
curl -s http://localhost:8080/security/headers/test | jq
```

Response:
```json
{
  "status": "ok",
  "headers": {
    "content_security_policy": "default-src 'self'; ...",
    "x_frame_options": "DENY",
    "x_content_type_options": "nosniff",
    "strict_transport_security": "max-age=31536000; includeSubDomains",
    "referrer_policy": "strict-origin-when-cross-origin",
    "permissions_policy": "camera=(), microphone=(), ...",
    "cross_origin_opener_policy": "same-origin",
    "cross_origin_resource_policy": "same-origin",
    "cross_origin_embedder_policy": "require-corp",
    "x_xss_protection": "1; mode=block",
    "x_permitted_cross_domain_policies": "none"
  },
  "message": "Security headers are configured correctly"
}
```

### Authenticated Test Endpoint

For testing authenticated responses:

```bash
curl -s -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/v1/security/headers/test | jq
```

### External Verification

Use external tools to verify security headers:

1. **securityheaders.com**: Scan your deployed URL
2. **Mozilla Observatory**: https://observatory.mozilla.org/
3. **Chrome DevTools**: Network tab > Response Headers

## Best Practices

### Production Checklist

1. **Enable HSTS**: Set `EnableHSTS: true` for all production deployments using HTTPS
2. **Review CSP**: Audit the Content-Security-Policy for your specific frontend requirements
3. **Test Thoroughly**: Use the `/security/headers/test` endpoint before deployment
4. **Monitor Reports**: If using CSP reporting, monitor for violations

### Common Issues

#### Frontend Assets Not Loading

If your frontend assets fail to load, check the CSP:

```bash
# View CSP in response
curl -sI http://localhost:8080 | grep -i content-security-policy
```

Common fixes:
- Add specific CDN domains to `script-src` or `img-src`
- Use `AdditionalCSPDirectives` for white-label deployments
- Ensure `'self'` is included for same-origin resources

#### Iframe Embedding Blocked

If you need to embed the UI in an iframe:

```go
cfg.FrameOptions = "SAMEORIGIN"  // Allow same-origin iframes
// OR
cfg.FrameOptions = "ALLOW-FROM https://parent.example.com"  // Specific origin (deprecated)
```

Note: `ALLOW-FROM` is deprecated. For fine-grained control, use CSP `frame-ancestors` instead.

#### WebSocket Connections Failing

Ensure WebSocket origins are allowed in CSP:

```go
cfg.AdditionalCSPDirectives = map[string]string{
    "connect-src": "wss://your-domain.com",
}
```

## Security Considerations

### Header Precedence

Security headers are set early in the request lifecycle. Some considerations:

1. Headers are set before any application logic runs
2. Error responses also include security headers
3. Headers cannot be modified by downstream handlers

### Browser Support

All modern browsers support these security headers. For legacy browser support:

- `X-XSS-Protection` is included for older IE/Edge versions
- `X-Content-Type-Options` works in IE8+
- CSP Level 2+ features require modern browsers

### HSTS Considerations

Before enabling HSTS:

1. Ensure HTTPS is properly configured
2. Test with a short `max-age` first (e.g., 300 seconds)
3. Consider `includeSubDomains` impact on all subdomains
4. Use `preload` only after thorough testing

### CSP Reporting

To enable CSP violation reporting (future feature):

```go
cfg.ContentSecurityPolicy = cfg.ContentSecurityPolicy +
    "; report-uri /api/v1/csp-reports"
```

Note: CSP report collection is not yet implemented in the API.
