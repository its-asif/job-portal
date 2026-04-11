# Job Portal Backend

A simple job portal backend built with Go, Gorilla Mux, and PostgreSQL.

Current implementation includes:
- User registration and login with bcrypt password hashing
- JWT authentication and role-based authorization
- User listing and profile endpoint
- Jobs CRUD APIs
- Job applications flow
- Employer-only application status updates
- Redis caching for jobs, JWT blacklist logout, and auth rate limiting
- RabbitMQ event publishing on job apply
- Background worker for email delivery with SMTP (Mailtrap)
- Outbox fallback + async retry for RabbitMQ publish failures
- Dead-letter queue flow for permanently failed notifications
- Unit tests for handlers, middleware, and JWT
- Repository integration test scaffold with TEST_DB_URL
- Swagger/OpenAPI docs with Swagger UI

## Tech Stack

- Go
- Gorilla Mux
- PostgreSQL
- Redis
- RabbitMQ
- JWT (github.com/golang-jwt/jwt/v5)
- SMTP mail delivery (gopkg.in/gomail.v2)
- Docker and Docker Compose
- golang-migrate (via Docker image)

## Project Structure

- cmd/main.go: API entrypoint
- cmd/worker/main.go: notification worker entrypoint
- db/db.go: database connection
- db/migrations: SQL migrations
- internal/models: data models
- internal/repository: database access layer
- internal/handlers: HTTP handlers
- internal/middleware: auth and role middleware
- internal/auth: JWT helpers
- internal/cache: Redis client
- internal/queue: RabbitMQ client and outbox retry worker
- internal/email: SMTP email sender

## Prerequisites

- Go (version from go.mod)
- Docker and Docker Compose
- Make

## Environment Variables

Copy the example values into your .env:

- DB_URL: PostgreSQL connection string
- JWT_SECRET: secret used for signing JWT tokens
- REDIS_URL: Redis connection URL
- RABBITMQ_URL: RabbitMQ AMQP connection URL
- MAIL_HOST: SMTP host (Mailtrap sandbox host for dev)
- MAIL_PORT: SMTP port
- MAIL_USERNAME: SMTP username
- MAIL_PASSWORD: SMTP password
- MAIL_FROM: sender email address

Example values are available in .env.example.

## Run Locally (Go)

1. Start services:

```bash
docker compose up -d db redis rabbitmq
```

2. Run migrations (from host to local DB):

```bash
make migrate-up DB_URL='postgres://postgres:postgres@localhost:5432/job_portal?sslmode=disable'
```

3. Start API:

```bash
go run cmd/main.go
```

4. Start worker (for RabbitMQ email notifications):

```bash
go run cmd/worker/main.go
```

API runs on http://localhost:8080.

## Run with Docker Compose

1. Start core services:

```bash
docker compose up -d db redis rabbitmq
```

2. Run migrations against db container network:

```bash
make migrate-up DB_URL='postgres://postgres:postgres@db:5432/job_portal?sslmode=disable' MIGRATE_NETWORK=talentdock_default
```

3. Start API and worker containers:

```bash
JWT_SECRET='dev-secret-please-change' docker compose up -d --build api worker
```

4. RabbitMQ Management UI:

```text
http://localhost:15673
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

## API Documentation (Swagger)

Generate Swagger docs:

```bash
make docs
```

Start the API and open Swagger UI:

```text
http://localhost:8080/docs/index.html
```

Notes:
- Regenerate docs after endpoint/comment changes.
- Protected endpoints use Bearer token in the `Authorization` header.

## Testing

Run all tests:

```bash
go test ./...
```

Run coverage:

```bash
go test ./... -cover
```

Run the Makefile test target:

```bash
make test
```

Run verbose tests and count passing test cases:

```bash
go test ./... -v | grep -c PASS
```

### Integration Test (Repository)

The user repository integration test in `internal/repository/user_repo_test.go` requires a real PostgreSQL database.

Set TEST_DB_URL and run:

```bash
TEST_DB_URL='postgres://postgres:postgres@localhost:5432/job_portal?sslmode=disable' go test ./internal/repository -v
```

If TEST_DB_URL is not set, the test is skipped intentionally.

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

## RabbitMQ Workflow

1. Jobseeker applies to a job via POST /jobs/{id}/apply.
2. API writes application to PostgreSQL first.
3. API publishes `new_application` event to `notifications` queue.
4. If publish fails, event is stored in `outbox_events` table.
5. Outbox retry worker republishes pending events asynchronously.
6. Worker consumes `notifications` queue and sends email via SMTP.
7. On email failure, message is retried up to 3 times.
8. After max retries, message moves to `failed_notifications` queue.

## RabbitMQ and Email Test

1. Start services and run API + worker.
2. Apply to a job as a jobseeker.
3. Watch worker logs:

```bash
docker compose logs -f worker
```

4. Verify mail in Mailtrap inbox.
5. Verify queue state if needed:

```bash
docker exec job-portal-rabbitmq rabbitmqctl list_queues name messages_ready messages_unacknowledged
```

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
- Week 4 testing phase is implemented through handler tests, auth/JWT tests, and repository integration test scaffolding.
- Redis and RabbitMQ phases are implemented with outbox and worker-based notifications.
- Next planned phases: elasticsearch, grpc, and CI/CD.
