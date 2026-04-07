# Job Portal Backend

A simple job portal backend built with Go, Gorilla Mux, and PostgreSQL.

Current implementation includes:
- User registration and login with bcrypt password hashing
- JWT authentication and role-based authorization
- User listing and profile endpoint
- Jobs CRUD APIs
- Job applications flow
- Employer-only application status updates

## Tech Stack

- Go
- Gorilla Mux
- PostgreSQL
- JWT (github.com/golang-jwt/jwt/v5)
- Docker and Docker Compose
- golang-migrate (via Docker image)

## Project Structure

- cmd/main.go: API entrypoint
- db/db.go: database connection
- db/migrations: SQL migrations
- internal/models: data models
- internal/repository: database access layer
- internal/handlers: HTTP handlers
- internal/middleware: auth and role middleware
- internal/auth: JWT helpers

## Prerequisites

- Go (version from go.mod)
- Docker and Docker Compose
- Make

## Environment Variables

Copy the example values into your .env:

- DB_URL: PostgreSQL connection string
- JWT_SECRET: secret used for signing JWT tokens

Example values are available in .env.example.

## Run Locally (Go)

1. Start PostgreSQL:

```bash
docker compose up -d db
```

2. Run migrations (from host to local DB):

```bash
make migrate-up DB_URL='postgres://postgres:postgres@localhost:5432/job_portal?sslmode=disable'
```

3. Start API:

```bash
go run cmd/main.go
```

API runs on http://localhost:8080.

## Run with Docker Compose

1. Start database:

```bash
docker compose up -d db
```

2. Run migrations against db container network:

```bash
make migrate-up DB_URL='postgres://postgres:postgres@db:5432/job_portal?sslmode=disable' MIGRATE_NETWORK=talentdock_default
```

3. Start API container:

```bash
JWT_SECRET='dev-secret-please-change' docker compose up -d --build api
```

## Migration Commands

Apply migrations:

```bash
make migrate-up DB_URL='<your_db_url>'
```

Rollback one migration:

```bash
make migrate-down DB_URL='<your_db_url>'
```

## Authentication and Roles

Login returns a JWT token:

- Use header: Authorization: Bearer <token>

Supported registration roles:
- employer
- jobseeker

Role rules:
- employer: can create jobs, delete jobs, update application status
- jobseeker: can apply to jobs

## API Endpoints

### Public

- GET /
- POST /register
- POST /login
- GET /users
- GET /users/{id}
- GET /jobs
- GET /jobs/{id}
- PUT /jobs/{id}
- GET /jobs/{id}/applications

### Authenticated

- GET /me

### Employer only

- POST /jobs
- DELETE /jobs/{id}
- PUT /applications/{id}/status

### Jobseeker only

- POST /jobs/{id}/apply

## Quick Test Flow

1. Register employer and jobseeker users.
2. Login both users and save tokens.
3. Create a job with employer token.
4. Apply to that job with jobseeker token.
5. Update application status with employer token.
6. Call /me with each token.

## Notes

- PostgreSQL is the source of truth.
- The project is being built incrementally by phases.
- More features (tests, docs, redis, queue, search, grpc, CI/CD) are planned next.
