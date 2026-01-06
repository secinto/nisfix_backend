# NisFix Backend

B2B Supplier Security Portal API - Companies manage supplier security requirements through questionnaires and CheckFix verification reports.

## Features

- **JWT Authentication** - RS512 (RSA) token-based auth with magic link login
- **Multi-tenant** - Companies and suppliers with relationship management
- **Requirements Workflow** - State machine for requirement lifecycle (pending → approved)
- **Questionnaires** - Customizable security questionnaires with templates
- **CheckFix Integration** - Automated security verification via CheckFix API
- **Audit Logging** - Track all security-relevant actions

## Quick Start

### Prerequisites

- Go 1.24+
- MongoDB 7.0+
- OpenSSL (for key generation)

### Setup

```bash
# Clone and navigate to project
cd nisfix_backend

# Generate JWT signing keys
make generate-keys

# Copy and configure environment
cp .env.example .env
# Edit .env with your settings

# Install dependencies
make deps

# Run tests
make test

# Build and run
make run
```

### Using Docker

```bash
# Generate keys first
make generate-keys

# Start full stack (API + MongoDB)
make docker-compose-up

# View logs
make docker-compose-logs

# Stop stack
make docker-compose-down
```

## API Documentation

Once running, access Swagger UI at:
- http://localhost:8080/swagger/index.html

### Health Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Full health status with version |
| `GET /health/ping` | Simple ping/pong |
| `GET /health/live` | Kubernetes liveness probe |

### Authentication

All `/api/v1/*` endpoints require JWT authentication:

```bash
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/v1/...
```

## Project Structure

```
nisfix_backend/
├── cmd/server/          # Application entrypoint
├── internal/
│   ├── auth/            # JWT service
│   ├── config/          # Environment configuration
│   ├── database/        # MongoDB client, indexes, seeding
│   ├── handlers/        # HTTP handlers (controllers)
│   ├── middleware/      # Auth, CORS, rate limiting, etc.
│   ├── models/          # Domain models
│   ├── repository/      # Data access layer
│   └── services/        # Business logic
├── docs/                # Swagger generated docs
├── scripts/             # Utility scripts
├── keys/                # JWT keys (gitignored)
├── Makefile             # Build automation
├── Dockerfile           # Container build
└── docker-compose.yml   # Local development stack
```

## Configuration

All configuration is via environment variables (prefixed with `NISFIX_`):

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NISFIX_DATABASE_URI` | Yes | - | MongoDB connection string |
| `NISFIX_DATABASE_NAME` | No | `nisfix` | Database name |
| `NISFIX_JWT_PRIVATE_KEY_PATH` | Yes | - | Path to RSA private key |
| `NISFIX_JWT_PUBLIC_KEY_PATH` | Yes | - | Path to RSA public key |
| `NISFIX_ACCESS_TOKEN_EXPIRY` | No | `1h` | Access token TTL |
| `NISFIX_REFRESH_TOKEN_EXPIRY` | No | `720h` | Refresh token TTL (30 days) |
| `NISFIX_MAIL_SERVICE_URL` | Yes | - | Mail service endpoint |
| `NISFIX_MAIL_API_KEY` | Yes | - | Mail service API key |
| `NISFIX_SERVER_PORT` | No | `8080` | HTTP server port |
| `NISFIX_ENVIRONMENT` | No | `development` | Environment (development/production) |
| `NISFIX_MAGIC_LINK_BASE_URL` | Yes | - | Frontend URL for magic links |
| `NISFIX_ALLOWED_ORIGINS` | No | `http://localhost:3000` | CORS allowed origins |

See `.env.example` for complete list with documentation.

## Development

### Available Make Targets

```bash
make help              # Show all targets

# Building
make build             # Build for current platform
make build-linux       # Cross-compile for Linux
make build-all         # Build for all platforms

# Testing
make test              # Run all tests
make test-coverage     # Generate coverage report
make bench             # Run benchmarks

# Code Quality
make fmt               # Format code
make vet               # Run go vet
make lint              # Run golangci-lint
make check             # Run all checks

# Development
make dev               # Run with hot reload (requires air)
make run               # Build and run

# Docker
make docker-build      # Build Docker image
make docker-compose-up # Start local stack

# Utilities
make swagger           # Regenerate Swagger docs
make generate-keys     # Generate new JWT keys
make clean             # Remove build artifacts
make install-tools     # Install dev tools
```

### Hot Reload Development

```bash
# Install air (first time only)
make install-tools

# Run with hot reload
make dev
```

### Running Tests

```bash
# All tests with race detection
make test

# With coverage report
make test-coverage
open coverage.html

# Specific package
go test -v ./internal/auth/...
```

### Linting

```bash
# Install linter (first time only)
make install-tools

# Run linter
make lint
```

## Architecture

### Domain Model

```
Organization (Company/Supplier)
    │
    ├── Users (Admin/Viewer roles)
    │
    └── Relationships ──────────────────┐
            │                           │
            ▼                           ▼
        Company                     Supplier
            │                           │
            └── Requirements ───────────┘
                    │
                    ├── Questionnaire
                    │       └── Responses
                    │
                    └── CheckFix Verification
                            └── Results
```

### Request Flow

```
HTTP Request
    │
    ▼
Middleware (Auth, CORS, Rate Limit)
    │
    ▼
Handler (validation, response formatting)
    │
    ▼
Service (business logic)
    │
    ▼
Repository (data access)
    │
    ▼
MongoDB
```

## Security

- **Authentication**: RS512 JWT with short-lived access tokens
- **Authorization**: Role-based (ADMIN, VIEWER) and org-type based (COMPANY, SUPPLIER)
- **Rate Limiting**: Configurable per-endpoint limits
- **CORS**: Configurable allowed origins
- **Security Headers**: X-Content-Type-Options, X-Frame-Options, etc.
- **Input Validation**: Request validation on all endpoints

## License

Apache 2.0
