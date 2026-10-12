# Trail Tools - AI Coding Agent Instructions

## Architecture Overview

This is a full-stack Go/React/TypeScript/PostgreSQL application for trail running tools. The key architectural pattern is **Connect RPC over HTTP/2** for type-safe client-server communication using Protocol Buffers.

### Core Components

- **Backend**: Go server with Connect RPC handlers in `internal/services/`
- **Frontend**: React/TypeScript SPA using Connect Query for API calls
- **Database**: PostgreSQL with SQLC-generated queries and migrations
- **Authentication**: Passkeys (WebAuthn) with session-based auth via cookies, and API tokens for API and MCP clients
- **Build System**: Make-driven with Docker containers for Node.js tools

## Development Workflows

### Essential Commands

```bash
make run      # Start full stack (Postgres + Go server)
make watch    # Development mode with frontend/CSS auto-rebuild
make gen      # Regenerate all code (protobuf, SQLC, frontend assets)
make format   # Format Go, TypeScript, and protobuf files
make lint     # Run all linters
```

**Critical**: Changes to Go server require manual restart of `make watch`. Frontend changes auto-rebuild.

### Code Generation Pipeline

1. **Protobuf**: `api/*/v1/*.proto` → `internal/gen/` (Go) + `web/gen/` (TypeScript) + `internal/apidesc/descriptor.binpb` (descriptor set with comments, embedded for MCP)
2. **SQLC**: `internal/*/queries/*.sql` + migrations → `internal/*/internal/*.sql.go`
3. **Frontend**: `web/index.tsx` → `web/dist/` via esbuild + Tailwind CSS
4. **Asset Embedding**: `web/dist/` → embedded in Go binary via `//go:embed dist` in `web/web.go`

## Service Layer Patterns

### Connect RPC Services

Services implement the generated `*ServiceHandler` interface:

```go
// internal/services/athlete/athlete_service.go
var _ athletesv1connect.AthleteServiceHandler = (*Service)(nil)

func (s *Service) CreateAthlete(ctx context.Context, req *connect.Request[athletesv1.CreateAthleteRequest]) (*connect.Response[athletesv1.CreateAthleteResponse], error) {
    // Authentication via internal/authn context extraction
    // Repository pattern via interface injection
}
```

### MCP Server

`internal/mcp` exposes every Connect RPC as an MCP tool at `/mcp`, built at startup from the descriptors in `internal/apidesc`. Tool descriptions are the proto comments, so write them for an AI agent. Exclude a method with `option (mcp.v1.method).exclude = true` and hide server-set fields from tool inputs with `[(mcp.v1.field).output_only = true]`. MCP clients authenticate with API tokens (`Authorization: Bearer`), handled by `internal/authn`.

### Repository Pattern

Services depend on repository interfaces (see `AthleteRepository` in `athlete_service.go`), implemented by SQLC-generated code in `internal/*/internal/`.

### Authentication Flow

1. Passkey sign in creates a session cookie
2. `internal/authn` middleware extracts the user from the session cookie or an API token
3. Services use `authn.GetUser(ctx)` or `authn.GetUserID(ctx)` for access control

## Frontend Patterns

### Connect Query Integration

```tsx
// Type-safe API calls with generated hooks
import { useQuery } from "@connectrpc/connect-query";
import { listAthletes } from "gen/athletes/v1/athletes-AthleteService_connectquery";

const { data, error } = useQuery(listAthletes);
```

### Component Structure

- Pages in `web/pages/` (Athletes.tsx, Settings.tsx)
- Reusable components in `web/components/` with domain-specific folders
- Base styles via Tailwind CSS with custom `base.css`

## Database Patterns

### Migration-Driven Schema

All schema changes via numbered migrations in `internal/store/migrations/`. SQLC generates type-safe queries from `.sql` files.

### SQLC Configuration

- Separate configs per domain (`athletes`, `users`, `activities`, `routes`, `races`, `plans`) in `sqlc.yaml`
- Custom type overrides (e.g., `pg_catalog.numeric` → `decimal.Decimal`)
- Pointer structs for nullable fields

## Key Integration Points

### Docker Dependencies

- PostgreSQL on `:5432` with password `password`
- Node.js tools run in Alpine containers with proper user mapping

### Build Tool Integration

Go tools are prefixed with `go tool` (e.g., `go tool buf`, `go tool esbuild`, `go tool sqlc`) for consistent toolchain management.

## Project-Specific Conventions

- **Error Handling**: Use `store.ErrNotFound` for repository not-found cases
- **Logging**: Structured logging with `slogor` wrapper around `slog`
- **UUIDs**: PostgreSQL `gen_random_uuid()` for all primary keys
- **Decimal Precision**: `shopspring/decimal` for financial/scientific precision (blood lactate)
- **Generated Code**: Never edit files with `// Code generated .* DO NOT EDIT` headers

## Testing

`make test` starts Postgres and runs `go test -race ./...` with `TEST_DATABASE_URL` set. `internal/store/storetest` gives each test its own migrated database; tests needing a database are skipped without it. `internal/services` has integration tests driving the Connect handlers over HTTP with a fake map data backend.

## Common Debugging

- Frontend API errors often indicate Connect transport misconfiguration
- Database connection issues: ensure `make db` container is running
- Missing types after schema changes: run `make gen`
