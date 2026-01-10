# Scalable E-Commerce Platform

A microservices-based e-commerce platform built with Go. This project demonstrates a clean migration from a monolithic architecture to microservices.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Clients                                  │
│                    (Web, Mobile, API)                           │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      API Gateway                                 │
│                    (Port 8080)                                   │
│  - Request routing                                               │
│  - JWT authentication                                            │
│  - Admin authorization                                           │
└─────────────────────────────────────────────────────────────────┘
                    │                    │
         ┌──────────┘                    └──────────┐
         ▼                                          ▼
┌─────────────────────────────┐   ┌─────────────────────────────┐
│      User Service           │   │     Product Service         │
│       (Port 8081)           │   │       (Port 8082)           │
│  - User registration        │   │  - Product catalog          │
│  - Authentication (login)   │   │  - Product CRUD (admin)     │
│  - JWT token generation     │   │  - Stock management         │
│  - Refresh token mgmt       │   │                             │
└─────────────────────────────┘   └─────────────────────────────┘
              │                                 │
              ▼                                 ▼
┌─────────────────────────────┐   ┌─────────────────────────────┐
│     PostgreSQL (users)      │   │  PostgreSQL (products)      │
│   - users table             │   │   - products table          │
│   - refresh_tokens table    │   │                             │
└─────────────────────────────┘   └─────────────────────────────┘
```

## Services

### API Gateway (`services/api-gateway`)
The entry point for all client requests. Handles:
- Request routing to appropriate microservices
- JWT token validation
- Role-based access control (admin routes)
- Request/response proxying

### User Service (`services/user-service`)
Manages user authentication and identity:
- User registration (`POST /api/users`)
- Login with JWT generation (`POST /api/login`)
- Token refresh (`POST /api/refresh`)
- Token revocation / logout (`POST /api/revoke`)
- Internal user lookup (for gateway)

### Product Service (`services/product-service`)
Handles the product catalog:
- List products (`GET /api/products`)
- Get product by ID (`GET /api/products/{id}`)
- Create product - admin only (`POST /admin/products`)
- Update product - admin only (`PATCH /admin/products/{id}`)

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Start all services
docker compose up -d

# View logs
docker compose logs -f

# Stop all services
docker compose down
```

The API will be available at `http://localhost:8080`

### Running Locally

1. **Prerequisites:**
   - Go 1.22+
   - PostgreSQL 16+
   - sqlc (for database code generation)

2. **Set up databases:**
   ```bash
   # Create user database
   createdb users
   
   # Create product database
   createdb products
   ```

3. **Run migrations:**
   ```bash
   # For user service
   cd services/user-service
   goose -dir sql/schema postgres "postgres://postgres:postgres@localhost:5432/users?sslmode=disable" up
   
   # For product service
   cd services/product-service
   goose -dir sql/schema postgres "postgres://postgres:postgres@localhost:5432/products?sslmode=disable" up
   ```

4. **Configure environment:**
   ```bash
   # Copy example env files
   cp services/user-service/.env.example services/user-service/.env
   cp services/product-service/.env.example services/product-service/.env
   cp services/api-gateway/.env.example services/api-gateway/.env
   ```

5. **Start services:**
   ```bash
   # Terminal 1 - User Service
   cd services/user-service && go run .
   
   # Terminal 2 - Product Service
   cd services/product-service && go run .
   
   # Terminal 3 - API Gateway
   cd services/api-gateway && go run .
   ```

## API Endpoints

### Public Endpoints (No Auth Required)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/users` | Register a new user |
| POST | `/api/login` | Login and get JWT token |
| POST | `/api/refresh` | Refresh access token (uses cookie) |
| POST | `/api/revoke` | Revoke refresh token (logout) |
| GET | `/api/products` | List all active products |
| GET | `/api/products/{id}` | Get product by ID |

### Protected Endpoints (Auth Required)

| Method | Endpoint | Role | Description |
|--------|----------|------|-------------|
| GET | `/api/me` | user | Get current user info |

### Admin Endpoints (Admin Role Required)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/admin/products` | Create a new product |
| PATCH | `/admin/products/{id}` | Update a product |

## Example Usage

### Register a User
```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password123"}'
```

### Login
```bash
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password123"}'
```

### Get Products
```bash
curl http://localhost:8080/api/products
```

### Create Product (Admin)
```bash
curl -X POST http://localhost:8080/admin/products \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin-jwt-token>" \
  -d '{"name": "Widget", "price_cents": 999, "stock": 100}'
```

## Development

### Regenerate Database Code
```bash
# User Service
cd services/user-service && sqlc generate

# Product Service
cd services/product-service && sqlc generate
```

### Run Tests
```bash
# Run all tests
go test ./...

# Run tests for specific service
cd services/user-service && go test ./...
```

### Build Docker Images
```bash
docker compose build
```

## Environment Variables

### API Gateway
| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `SECRET_KEY` | JWT signing key | - |
| `USER_SERVICE_URL` | User service URL | `http://localhost:8081` |
| `PRODUCT_SERVICE_URL` | Product service URL | `http://localhost:8082` |

### User Service
| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8081` |
| `PLATFORM` | Environment (dev/prod) | `dev` |
| `SECRET_KEY` | JWT signing key | - |
| `DB_URL` | PostgreSQL connection string | - |

### Product Service
| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8082` |
| `PLATFORM` | Environment (dev/prod) | `dev` |
| `DB_URL` | PostgreSQL connection string | - |

## Legacy Monolith

The original monolithic application code is preserved in the root directory (`main.go`, `internal/`, `sql/`). It can still be run independently if needed:

```bash
# Run the monolith
go run main.go
```

## Tech Stack

- **Language:** Go 1.22
- **Database:** PostgreSQL 16
- **SQL Generation:** SQLC
- **Migrations:** Goose
- **Authentication:** JWT (HS256)
- **Password Hashing:** Argon2id
- **Containerization:** Docker
