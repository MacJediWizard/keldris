# Keldris - AI Agent Instructions

> **Keldris** - Keeper of your data  
> Self-hosted backup solution with OIDC auth, Restic engine, cloud storage support

## Tech Stack (DO NOT CHANGE)

| Layer | Technology | Version |
|-------|------------|---------|
| Server | Go | 1.25.7+ |
| Server Framework | Gin | latest |
| Database | PostgreSQL | 15+ |
| ORM | pgx/v5 raw queries | latest |
| Agent | Go + Cobra | 1.25.7+ |
| Frontend | React | 18+ |
| Frontend Build | Vite | 6+ |
| Frontend Language | TypeScript (strict) | 5.6+ |
| Styling | Tailwind CSS | 3.4+ |
| State/Fetching | @tanstack/react-query | 5+ |
| Routing | react-router-dom | 7+ |
| OIDC | coreos/go-oidc + golang.org/x/oauth2 | latest |
| Backup Engine | Restic (exec wrapper) | latest |
| i18next | i18next | ^25.7.4 |
| i18n React Bindings | react-i18next | ^16.x |
| i18n React Bindings | react-i18next | ^15.x |
| Linting | Biome (frontend), staticcheck (Go) | latest |

---

## Project Structure (FOLLOW EXACTLY)

```
keldris/
├── cmd/
│   ├── keldris-server/main.go
│   └── keldris-agent/main.go
├── internal/
│   ├── api/
│   │   ├── handlers/          # One file per resource
│   │   ├── middleware/        # Auth, CORS, rate limit, etc.
│   │   └── routes.go          # All route registration
│   ├── auth/
│   │   ├── oidc.go
│   │   ├── session.go
│   │   └── apikey.go
│   ├── backup/
│   │   ├── restic.go
│   │   ├── scheduler.go
│   │   └── backends/           # One file per backend
│   ├── config/
│   │   ├── server.go
│   │   └── agent.go
│   ├── crypto/
│   │   └── aes.go
│   ├── db/
│   │   ├── migrations/        # Sequential numbered SQL files
│   │   ├── db.go
│   │   └── store.go           # Query methods
│   ├── models/                # One file per model
│   ├── notifications/
│   │   ├── email.go
│   │   ├── slack.go
│   │   ├── webhook.go
│   │   └── templates/
│   └── ...                    # Business logic in domain packages (health/, metrics/, etc.)
├── pkg/models/                # Shared models (agent<->server)
├── web/
│   ├── src/
│   │   ├── components/
│   │   │   ├── ui/            # Reusable primitives (Button, Input, Modal, etc.)
│   │   │   ├── Layout.tsx      # App layout component
│   │   │   └── features/      # Feature-specific components
│   │   ├── pages/             # One file per route
│   │   ├── hooks/             # One file per resource (useAgents, useBackups, etc.)
│   │   ├── lib/
│   │   │   ├── api.ts         # API client
│   │   │   ├── types.ts       # TypeScript types
│   │   │   ├── utils.ts       # Utility functions
│   │   │   └── constants.ts   # App constants
│   │   ├── App.tsx
│   │   └── main.tsx
│   └── ...
├── docker/
├── scripts/
├── docs/
└── ...
```

---

## Code Patterns (MANDATORY)

### Go Handler Pattern
```go
// internal/api/handlers/agents.go
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/MacJediWizard/keldris/internal/db"
	"github.com/MacJediWizard/keldris/internal/models"
)

type AgentHandler struct {
	db *db.DB
}

func NewAgentHandler(db *db.DB) *AgentHandler {
	return &AgentHandler{db: db}
}

// List returns all agents for the current organization.
func (h *AgentHandler) List(c *gin.Context) {
	orgID := c.GetString("org_id") // Set by auth middleware
	
	agents, err := h.db.ListAgentsByOrg(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch agents"})
		return
	}
	
	c.JSON(http.StatusOK, agents)
}

// Create registers a new agent.
func (h *AgentHandler) Create(c *gin.Context) {
	var req models.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// ... implementation
}
```

### Go Model Pattern
```go
// internal/models/agent.go
package models

import (
	"time"

	"github.com/google/uuid"
)

type Agent struct {
	ID        uuid.UUID  `json:"id"`
	OrgID     uuid.UUID  `json:"org_id"`
	Hostname  string     `json:"hostname"`
	Status    string     `json:"status"`
	LastSeen  *time.Time `json:"last_seen"`
	OSInfo    *OSInfo    `json:"os_info,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type OSInfo struct {
	OS      string `json:"os"`
	Arch    string `json:"arch"`
	Version string `json:"version"`
}

type CreateAgentRequest struct {
	Hostname string `json:"hostname" binding:"required"`
}

type UpdateAgentRequest struct {
	Status *string `json:"status,omitempty"`
}
```

### Go Database Query Pattern
```go
// internal/db/store.go
package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/MacJediWizard/keldris/internal/models"
)

func (db *DB) ListAgentsByOrg(ctx context.Context, orgID string) ([]models.Agent, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT id, org_id, hostname, status, last_seen, os_info, created_at, updated_at
		FROM agents
		WHERE org_id = $1
		ORDER BY hostname ASC
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []models.Agent
	for rows.Next() {
		var a models.Agent
		if err := rows.Scan(&a.ID, &a.OrgID, &a.Hostname, &a.Status, &a.LastSeen, &a.OSInfo, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, nil
}
```

### Go Migration Pattern
```sql
-- internal/db/migrations/002_add_notifications.sql
-- Migration: Add notification preferences

ALTER TABLE users ADD COLUMN notification_email BOOLEAN DEFAULT true;
ALTER TABLE users ADD COLUMN notification_slack BOOLEAN DEFAULT false;

CREATE TABLE notification_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL, -- email, slack, webhook, pagerduty
    config_encrypted BYTEA NOT NULL,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_notification_channels_org ON notification_channels(org_id);
```

### React Component Pattern
```tsx
// web/src/components/features/AgentCard.tsx
import { Agent } from '@/lib/types';
import { Badge } from '@/components/ui/Badge';
import { formatDistanceToNow } from '@/lib/utils';

interface AgentCardProps {
	agent: Agent;
	onDelete: (id: string) => void;
}

export function AgentCard({ agent, onDelete }: AgentCardProps) {
	return (
		<div className="rounded-lg border bg-white p-4 shadow-sm">
			<div className="flex items-center justify-between">
				<h3 className="font-medium text-gray-900">{agent.hostname}</h3>
				<Badge variant={agent.status === 'active' ? 'success' : 'warning'}>
					{agent.status}
				</Badge>
			</div>
			<p className="mt-1 text-sm text-gray-500">
				Last seen: {agent.lastSeen ? formatDistanceToNow(agent.lastSeen) : 'Never'}
			</p>
			<button
				onClick={() => onDelete(agent.id)}
				className="mt-2 text-sm text-red-600 hover:text-red-800"
			>
				Remove
			</button>
		</div>
	);
}
```

### React Hook Pattern
```tsx
// web/src/hooks/useAgents.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { Agent, CreateAgentRequest } from '@/lib/types';

export function useAgents() {
	return useQuery({
		queryKey: ['agents'],
		queryFn: () => api.get<Agent[]>('/api/v1/agents'),
	});
}

export function useAgent(id: string) {
	return useQuery({
		queryKey: ['agents', id],
		queryFn: () => api.get<Agent>(`/api/v1/agents/${id}`),
		enabled: !!id,
	});
}

export function useCreateAgent() {
	const queryClient = useQueryClient();
	
	return useMutation({
		mutationFn: (data: CreateAgentRequest) => api.post<Agent>('/api/v1/agents', data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['agents'] });
		},
	});
}

export function useDeleteAgent() {
	const queryClient = useQueryClient();
	
	return useMutation({
		mutationFn: (id: string) => api.delete(`/api/v1/agents/${id}`),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['agents'] });
		},
	});
}
```

### React Page Pattern
```tsx
// web/src/pages/Agents.tsx
import { useState } from 'react';
import { useAgents, useDeleteAgent } from '@/hooks/useAgents';
import { AgentCard } from '@/components/features/AgentCard';
import { Button } from '@/components/ui/Button';
import { Modal } from '@/components/ui/Modal';
import { CreateAgentForm } from '@/components/features/CreateAgentForm';
import { LoadingSpinner } from '@/components/ui/LoadingSpinner';
import { ErrorMessage } from '@/components/ui/ErrorMessage';

export default function AgentsPage() {
	const [showCreateModal, setShowCreateModal] = useState(false);
	const { data: agents, isLoading, error } = useAgents();
	const deleteAgent = useDeleteAgent();

	if (isLoading) return <LoadingSpinner />;
	if (error) return <ErrorMessage message="Failed to load agents" />;

	return (
		<div className="p-6">
			<div className="mb-6 flex items-center justify-between">
				<h1 className="text-2xl font-bold text-gray-900">Agents</h1>
				<Button onClick={() => setShowCreateModal(true)}>Add Agent</Button>
			</div>

			<div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
				{agents?.map((agent) => (
					<AgentCard
						key={agent.id}
						agent={agent}
						onDelete={(id) => deleteAgent.mutate(id)}
					/>
				))}
			</div>

			<Modal open={showCreateModal} onClose={() => setShowCreateModal(false)}>
				<CreateAgentForm onSuccess={() => setShowCreateModal(false)} />
			</Modal>
		</div>
	);
}
```

### API Client Pattern
```tsx
// web/src/lib/api.ts
const BASE_URL = import.meta.env.VITE_API_URL || '';

class ApiError extends Error {
	constructor(public status: number, message: string) {
		super(message);
		this.name = 'ApiError';
	}
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
	const response = await fetch(`${BASE_URL}${path}`, {
		...options,
		credentials: 'include',
		headers: {
			'Content-Type': 'application/json',
			...options.headers,
		},
	});

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: 'Unknown error' }));
		throw new ApiError(response.status, error.error || 'Unknown error');
	}

	if (response.status === 204) return undefined as T;
	return response.json();
}

export const api = {
	get: <T>(path: string) => request<T>(path),
	post: <T>(path: string, data: unknown) => request<T>(path, { method: 'POST', body: JSON.stringify(data) }),
	put: <T>(path: string, data: unknown) => request<T>(path, { method: 'PUT', body: JSON.stringify(data) }),
	patch: <T>(path: string, data: unknown) => request<T>(path, { method: 'PATCH', body: JSON.stringify(data) }),
	delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
};
```

### TypeScript Types Pattern
```tsx
// web/src/lib/types.ts
export interface Agent {
	id: string;
	orgId: string;
	hostname: string;
	status: 'pending' | 'active' | 'offline';
	lastSeen: string | null;
	osInfo: OSInfo | null;
	createdAt: string;
	updatedAt: string;
}

export interface OSInfo {
	os: string;
	arch: string;
	version: string;
}

export interface CreateAgentRequest {
	hostname: string;
}

// Add all types here - one interface per model
```

---

## UI Component Library (USE THESE)

Create these base components first, reuse everywhere:

```
web/src/components/ui/
├── Badge.tsx          # Status badges
├── Button.tsx         # Primary, secondary, danger variants
├── Card.tsx           # Content cards
├── DataTable.tsx      # Sortable, filterable tables
├── DropdownMenu.tsx   # Action menus
├── ErrorMessage.tsx   # Error display
├── Form.tsx           # Form wrapper
├── Input.tsx          # Text inputs
├── Label.tsx          # Form labels
├── LoadingSpinner.tsx # Loading state
├── Modal.tsx          # Dialog modals
├── Pagination.tsx     # Table pagination
├── Select.tsx         # Dropdowns
├── Tabs.tsx           # Tab navigation
├── Toast.tsx          # Notifications
└── Tooltip.tsx        # Hover tooltips
```

---

## Naming Conventions (MANDATORY)

### Files
- Go: `snake_case.go`
- TypeScript: `PascalCase.tsx` for components, `camelCase.ts` for utilities
- Migrations: `NNN_description.sql` (e.g., `001_initial_schema.sql`)

### Variables
- Go: `camelCase` for local, `PascalCase` for exported
- TypeScript: `camelCase` for variables, `PascalCase` for types/interfaces

### API Endpoints
- REST: `/api/v1/{resource}` (plural)
- Examples:
  - `GET /api/v1/agents`
  - `POST /api/v1/agents`
  - `GET /api/v1/agents/:id`
  - `PUT /api/v1/agents/:id`
  - `DELETE /api/v1/agents/:id`
  - `POST /api/v1/agents/:id/actions/restart` (for actions)

### Database
- Tables: `snake_case` plural (e.g., `agents`, `backup_schedules`)
- Columns: `snake_case` (e.g., `created_at`, `org_id`)
- Indexes: `idx_{table}_{column}` (e.g., `idx_agents_org_id`)

---

## Error Handling (MANDATORY)

### Go
```go
// Always wrap errors with context
if err != nil {
    return fmt.Errorf("failed to create agent: %w", err)
}

// API errors return consistent JSON
c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid hostname"})
c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
```

### TypeScript
```tsx
// Use error boundaries for pages
// Show user-friendly messages
// Log details to console in development
```

---

## Security Rules (NON-NEGOTIABLE)

1. **NEVER** log credentials, tokens, API keys, passwords
2. **ALWAYS** use parameterized queries - no string concatenation
3. **ALWAYS** validate user input before use
4. **ALWAYS** check org_id on every query (multi-tenant isolation)
5. Session cookies: `HttpOnly=true, Secure=true, SameSite=Lax`
6. Encrypt sensitive data at rest with AES-256-GCM

---

## Testing Pattern

### Go
```go
// internal/api/handlers/agents_test.go
func TestAgentHandler_List(t *testing.T) {
    // Setup
    db := setupTestDB(t)
    handler := NewAgentHandler(db)
    
    // Test
    // ...
    
    // Assert
    // ...
}
```

### TypeScript
```tsx
// web/src/hooks/useAgents.test.ts
import { renderHook, waitFor } from '@testing-library/react';
import { useAgents } from './useAgents';

describe('useAgents', () => {
    it('fetches agents', async () => {
        const { result } = renderHook(() => useAgents());
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(result.current.data).toHaveLength(2);
    });
});
```

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

## Git Commits (IMPORTANT)

When creating git commits:
- **DO NOT** add any `Co-Authored-By` lines to commit messages
- **DO NOT** mention Claude, AI, LLM, or any AI assistant in commit messages
- Write commit messages as if a human developer wrote them
- Keep commit messages concise and focused on what changed

Example commit message format:
```
Add agent health monitoring endpoint

- Add /api/v1/agents/:id/health endpoint
- Track health history in database
- Display health timeline in UI
```

---

## Before Each PR

1. `make lint` passes
2. `make test` passes
3. No credentials in code
4. Follows patterns above exactly
5. Updates JOURNAL.md (in .claude/ directory only)
