# Auth Service

The `auth_service` is a core microservice in the Omnichannel platform responsible for authentication, authorization (RBAC), and multi-tenant workspace management. It is built in Go and utilizes the [Authboss](https://github.com/volatiletech/authboss) framework for robust, secure identity management.

## Key Capabilities & Features

### 1. Authentication (Powered by Authboss)
- **JSON API**: Fully customized Authboss to act as a headless JSON API (rather than a traditional HTML-rendering monolith) via custom `JSONRenderer` and `JSONResponder` implementations.
- **Login / Registration**: Secure user authentication and account creation workflows.
- **JWT Sessions & Redis**: Uses JWTs for session management. When a user logs out, their JWT is placed on a Redis-backed blacklist to immediately invalidate the token across all microservices.
- **Password Recovery & Email Confirmation**: Built-in endpoints to handle password resets and account verification workflows.

### 2. Multi-Tenant Workspace Management
- Every user belongs to a `Workspace` (a logical grouping representing an organization or tenant).
- **Workspace Isolation**: Database queries and RBAC are scoped to ensure users can only access data and roles within their designated workspace.
- Users can fetch their specific workspace details and list all users belonging to their workspace.

### 3. Role-Based Access Control (RBAC)
- **Granular Permissions**: System-wide permissions are pre-defined in the database.
- **Custom Roles**: Users can create custom roles within their workspace and assign specific permissions to them.
- **System Roles**: Pre-configured protected roles (e.g., `Admin`, `Agent`) that cannot be deleted or modified.
- **Role Assignment**: Admins can assign and update the roles of other users in their workspace.

### 4. API Gateway Introspection
- Exposes a dedicated `/api/auth/introspect` endpoint.
- The API Gateway uses this endpoint to silently validate incoming JWTs against the Redis blacklist and extract the user's claims (User ID, Workspace ID, Role ID, Permissions).
- This allows the Gateway to securely inject user identity headers (`X-User-Id`, `X-Workspace-Id`, etc.) before routing requests to other downstream microservices.

---

## Directory Structure

* `cmd/api/main.go` - Application entry point. Bootstraps the database, Redis, Authboss configuration, and registers all HTTP routes (using `go-chi/chi`).
* `internal/auth/`
  * `handlers.go` - Custom HTTP handlers for RBAC management, workspace operations, profile fetching, and token introspection.
  * `storer.go` - Implementation of the Authboss `ServerStorer` and `User` interfaces, bridging Authboss logic with our PostgreSQL database.
  * `responder.go` & `renderer.go` - Custom overrides to force Authboss into operating strictly as a JSON API, returning structured standard API responses instead of HTML templates.
* `internal/db/` - Auto-generated database access layer (powered by `sqlc`). Contains the raw SQL queries and their type-safe Go method equivalents.
* `migrations/` - PostgreSQL schema migration files defining the `users`, `workspaces`, `roles`, and `role_permissions` tables.

---

## Getting Started

### Prerequisites
- Docker & Docker Compose
- PostgreSQL (handled by Docker Compose)
- Redis (handled by Docker Compose)

### Running Locally
The `auth_service` is designed to be run as part of the broader microservice ecosystem. From the root `backend-go-microservices` directory, run:

```bash
docker-compose up --build auth_service
```

This will automatically:
1. Spin up the Postgres and Redis dependencies.
2. Apply the latest database migrations.
3. Start the `auth_service` on port `8080`. (Note: Access it through the API Gateway at port `8000` in production/full-stack scenarios).

### Modifying Database Queries
We use `sqlc` to generate type-safe Go code from raw SQL. If you add or modify a query in `internal/db/query.sql`:
1. Ensure `sqlc` is installed locally.
2. Run `sqlc generate` in the `auth_service` root directory.
*(Note: If `sqlc` is unavailable locally, manually update the `query.sql.go` and `querier.go` interface instead).*
