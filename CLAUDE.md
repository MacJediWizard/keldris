# Keldris - AI Agent Instructions

> **Keldris** - Keeper of your data  
> Self-hosted backup solution with OIDC auth, Restic engine, cloud storage support

## Tech Stack

| Layer | Technology |
|-------|------------|
| Server | Go 1.24+ / Gin / PostgreSQL |
| Agent | Go 1.24+ / Cobra |
| Frontend | React 18 / TypeScript / Vite / Tailwind |
| Auth | OIDC (Authentik-first) + API keys for agents |
| Backup | Restic |
| License | AGPLv3 |

## Repository Structure

```
keldris/
├── cmd/
│   ├── keldris-server/main.go
│   └── keldris-agent/main.go
├── internal/
│   ├── api/handlers/
│   ├── api/middleware/
│   ├── auth/
│   ├── backup/
│   ├── config/
│   ├── crypto/
│   ├── db/migrations/
│   └── models/
├── pkg/models/
├── web/src/
├── docker/
├── Makefile
├── go.mod
└── conductor.json
```

---

## Parallel Task Breakdown

### Phase 1 (Start Together)

**Track 1: Database & Models**
```
Set up PostgreSQL database layer:
1. Create internal/db/db.go with pgx connection pool
2. Create migrations in internal/db/migrations/
3. Create internal/models/ with Go structs
4. Add migration runner in cmd/migrate/main.go
```

**Track 6: Frontend Shell**
```
Build React frontend shell (no API calls yet):
1. Initialize Vite + React + TypeScript in web/
2. Create layout components (Sidebar, Header)
3. Create placeholder pages (Dashboard, Agents, etc.)
4. Style with Tailwind
```

### Phase 2 (After Track 1)

**Track 2: OIDC Authentication**
```
Implement OIDC authentication:
1. Create internal/auth/oidc.go with coreos/go-oidc
2. Create internal/auth/session.go
3. Create handlers: /auth/login, /auth/callback, /auth/logout
```

**Track 4: Agent CLI**
```
Build the keldris-agent CLI:
1. Create cmd/keldris-agent/main.go with Cobra
2. Implement: register, config, backup, restore, status
3. Create internal/config/agent.go for YAML config
```

### Phase 3 (After Track 2)

**Track 3: API Handlers**
```
Build REST API handlers:
1. Create internal/api/routes.go with Gin
2. Create middleware (auth, cors, ratelimit)
3. Create handlers for agents, repositories, schedules, backups
```

**Track 5: Restic Integration**
```
Implement Restic backup wrapper:
1. Create internal/backup/restic.go
2. Create internal/backup/scheduler.go with robfig/cron
3. Create internal/backup/storage.go for backends
```

### Phase 4 (After APIs)

**Track 6: Frontend Complete**
```
Connect frontend to real APIs:
1. Create lib/api.ts with fetch wrapper
2. Wire up all pages to API endpoints
3. Add error handling and loading states
```

**Track 7: Docker & CI/CD**
```
Create deployment:
1. Create docker/Dockerfile.server and Dockerfile.agent
2. Create docker/docker-compose.yml
3. Create .github/workflows/ for CI/CD
```

---

## Code Standards (MANDATORY)

### Go
- All code must pass `go vet ./...` and `staticcheck ./...`
- All exported functions must have doc comments
- Error handling must be explicit - NEVER use `_ = err`
- No dead code or unused imports

### TypeScript
- Biome must pass with zero errors
- No `any` types without justification
- All components must have typed props

### Security (NON-NEGOTIABLE)
- **NEVER** log credentials, tokens, API keys
- **ALWAYS** use parameterized queries
- **ALWAYS** validate user input
- Session cookies: `HttpOnly=true, Secure=true, SameSite=Lax`
- Encrypt credentials at rest with AES-256-GCM

---

## Commands

```bash
make deps    # Install dependencies
make dev     # Run server + frontend
make test    # Run all tests
make lint    # Run linters
make build   # Build everything
```

---

## Journal Updates

Update JOURNAL.md after completing features:
```markdown
## YYYY-MM-DD - Title
### What / Why / How / Result
```
