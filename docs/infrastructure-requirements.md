# Infrastructure Requirements

## Go Runtime

**Minimum version: Go 1.25.7+**

Go 1.25.7 or later is required for production deployments. Earlier versions contain stdlib vulnerabilities that affect Keldris server and agent binaries.

### Patched Vulnerabilities

| ID | Summary |
|----|---------|
| GO-2026-4341 | stdlib vulnerability patched in Go 1.25.7 |
| GO-2026-4340 | stdlib vulnerability patched in Go 1.25.7 |
| GO-2026-4337 | stdlib vulnerability patched in Go 1.25.7 |

All three are standard library vulnerabilities that affect any Go binary compiled with earlier toolchains. Upgrading the Go compiler and rebuilding is the only remediation.

### Upgrading in Docker

Update the builder stage base image in your Dockerfiles:

```dockerfile
# Before
FROM golang:1.24-alpine AS builder

# After
FROM golang:1.25.7-alpine AS builder
```

Both `docker/Dockerfile.server` and `docker/Dockerfile.agent` must be updated. After changing the base image, rebuild:

```bash
docker build -f docker/Dockerfile.server -t keldris-server .
docker build -f docker/Dockerfile.agent -t keldris-agent .
```

### Upgrading in CI/CD

Update the Go version in your CI/CD pipeline configuration:

1. **GitHub Actions** - Update `go-version` in workflow files:
   ```yaml
   - uses: actions/setup-go@v5
     with:
       go-version: '1.25.7'
   ```

2. **GitLab CI** - Update the image tag:
   ```yaml
   image: golang:1.25.7-alpine
   ```

3. **Local development** - Update via your package manager or download from https://go.dev/dl/:
   ```bash
   # macOS (Homebrew)
   brew upgrade go

   # Verify
   go version
   ```

Ensure all environments (development, CI, staging, production) use Go 1.25.7+ to avoid shipping binaries with known vulnerabilities.
