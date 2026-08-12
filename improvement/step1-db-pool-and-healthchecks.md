# Step 1: Database Pool and Health Checks

## Goal

Refactor the backend database access from per-request Supabase connections to a process-level `pgxpool.Pool`. The service should reuse database connections while the Go process is alive, return English JSON errors for request-time failures, and expose clear health endpoints for Render and frontend status checks.

This step only plans the backend connection and error-handling refactor. It does not change frontend UI, crawler behavior, or API route names yet.

## Current Problem

The current backend opens a new Supabase connection inside each data fetch function and closes it at the end of the request. It also uses `log.Fatal` in request-time code paths. This is risky for deployment because a temporary database query failure can terminate the whole backend process.

For a stable deployed service:

- startup errors may stop the process when the service cannot be configured correctly;
- request-time errors must be returned to the client as HTTP JSON responses;
- the database pool should manage connection creation, reuse, idle cleanup, and broken connection replacement.

## Implementation Plan

### 1. Create the Database Pool During Startup

Create the pool in `main()` through a reusable helper:

```go
func newDBPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error)
```

The helper should:

- parse `SUPABASE_DB_URL` with `pgxpool.ParseConfig`;
- configure conservative pool settings for a Render/Supabase deployment;
- return `(*pgxpool.Pool, error)` instead of calling `log.Fatal`.

Recommended initial pool settings:

```go
poolConfig.MinConns = 0
poolConfig.MaxConns = 5
poolConfig.MaxConnIdleTime = 5 * time.Minute
poolConfig.MaxConnLifetime = time.Hour
poolConfig.HealthCheckPeriod = time.Minute
```

`MinConns` should remain low because Render and Supabase may sleep in free or low-cost environments. Keeping idle connections alive is less important than handling cold starts and temporary unavailability cleanly.

### 2. Keep Fatal Errors Only in Startup Paths

`main()` may still stop the process for unrecoverable startup errors:

- `SUPABASE_DB_URL` is missing;
- `SUPABASE_DB_URL` has an invalid format;
- the HTTP server cannot listen on the configured port.

Do not stop the process only because a startup-time database ping fails. The service can still expose `/healthz`, and database readiness should be reported through `/readyz` and business API responses.

### 3. Inject Dependencies With a Server Struct

Avoid a global `dbPool`. Store dependencies on a server struct:

```go
type Server struct {
    db *pgxpool.Pool
}
```

Register Gin handlers as method values:

```go
srv := &Server{db: dbPool}

router.POST("/result", srv.getData)
router.POST("/metadata", srv.getMeta)
router.GET("/healthz", srv.healthz)
router.GET("/readyz", srv.readyz)
```

This keeps handler dependencies explicit and makes future testing easier.

### 4. Add Health and Readiness Endpoints

`GET /healthz` should only verify that the Go backend process can respond. It should not ping Supabase.

Example response:

```json
{
  "status": "ok",
  "service": "mono-back"
}
```

`GET /readyz` should check whether Supabase is currently reachable by running `SELECT 1` with a short timeout. This verifies that the pool can acquire a connection and execute a minimal query.

Ready response:

```json
{
  "status": "ready",
  "database": "ok"
}
```

Unavailable response:

```json
{
  "status": "unavailable",
  "database": "unavailable",
  "message": "Database is temporarily unavailable."
}
```

Render health checks should use `/healthz`, not `/readyz`, so a temporary Supabase issue does not make Render treat the backend instance as broken.

### 5. Do Not Ping Before Every Business Request

Business handlers should not ping the database before each query. `pgxpool` already creates, reuses, validates, and replaces connections as needed. A database that was temporarily unavailable does not make the pool object permanently invalid.

The request path should be:

```text
request -> validate parameters -> query through pgxpool -> return data or JSON error
```

If a query fails because Supabase is unavailable, return a `503` JSON error. The next request can naturally try again through the same pool.

### 6. Remove Request-Time log.Fatal

Remove `log.Fatal` from request-time database code. Repository/data functions should return errors to handlers instead of terminating the process.

Examples of request-time failures:

- invalid `source`;
- invalid `page`;
- database query failure;
- row scan failure;
- Supabase temporary unavailability.

Handlers should translate those errors into English JSON responses.

Recommended generic database error:

```json
{
  "code": "DATABASE_UNAVAILABLE",
  "message": "Restaurant data is temporarily unavailable. Please try again later."
}
```

Recommended validation error:

```json
{
  "code": "INVALID_REQUEST",
  "message": "The request parameters are invalid."
}
```

## Acceptance Criteria

- The backend creates one `pgxpool.Pool` at startup and reuses it for `/result`, `/metadata`, and `/readyz`.
- `dbPool` is not a global variable.
- `/healthz` responds without touching the database.
- `/readyz` returns `200` when Supabase can execute `SELECT 1` and `503` when it cannot.
- A failed `/result` or `/metadata` database query returns an English JSON error and does not stop the Go process.
- Request-time code no longer calls `log.Fatal`.
- The backend still supports environment-based deployment configuration.

## Notes for Later Steps

- Frontend cold-start handling should call or infer `/healthz` and `/readyz` states in a later step.
- API route renaming, response contract cleanup, and frontend error UI are separate follow-up tasks.
- External monitors may call `/readyz` to observe database availability, but the application should not rely on keep-alive traffic for correctness.
