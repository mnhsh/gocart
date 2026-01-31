# Scalable E-Commerce Platform

A microservices-based e-commerce platform built with Go, featuring event-driven architecture with RabbitMQ and a CLI demo tool.

## Motivation

This project was built as a portfolio piece to demonstrate backend engineering skills. It showcases:

- **Microservices Architecture** - 5 independent services with clear boundaries
- **Event-Driven Design** - Asynchronous stock updates via RabbitMQ
- **API Gateway Pattern** - Centralized routing, authentication, and RBAC
- **Database Per Service** - Each service owns its data (4 PostgreSQL instances)
- **Clean Go Code** - Idiomatic patterns, SQLC for type-safe database queries
- **Full Working Demo** - CLI tool to exercise the entire system end-to-end

## Features

- User registration and JWT authentication with refresh tokens
- Role-based access control (admin vs regular users)
- Product catalog with admin CRUD operations
- Shopping cart management
- Order creation and cancellation
- Async stock updates via RabbitMQ (order → stock decrement, cancel → stock restore)
- Interactive CLI for demoing the complete flow

## Architecture

```
                                    ┌─────────────┐
                                    │     CLI     │
                                    └──────┬──────┘
                                           │
                                           ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                            API Gateway (:8080)                                │
│                    JWT Auth · RBAC · Request Routing                          │
└──────────────────────────────────────────────────────────────────────────────┘
         │                    │                    │                    │
         ▼                    ▼                    ▼                    ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│  User Service   │  │ Product Service │  │  Cart Service   │  │  Order Service  │
│    (:8081)      │  │    (:8082)      │  │    (:8083)      │  │    (:8084)      │
├─────────────────┤  ├─────────────────┤  ├─────────────────┤  ├─────────────────┤
│ • Registration  │  │ • Product CRUD  │  │ • Add/remove    │  │ • Create order  │
│ • JWT Auth      │  │ • Stock mgmt    │  │ • Update qty    │  │ • Cancel order  │
│ • Token refresh │  │ • RabbitMQ      │  │ • Price lookup  │  │ • RabbitMQ      │
│                 │  │   consumer      │  │                 │  │   publisher     │
└────────┬────────┘  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘
         │                    │                    │                    │
         ▼                    ▼                    ▼                    ▼
    [PostgreSQL]         [PostgreSQL]        [PostgreSQL]         [PostgreSQL]
      :5433                :5434               :5435                :5436

                    ┌───────────────────────────────────────┐
                    │              RabbitMQ                 │
                    │          (:5672 / :15672)             │
                    ├───────────────────────────────────────┤
                    │  Exchange: orders                     │
                    │  • order.created  → decrement stock   │
                    │  • order.cancelled → restore stock    │
                    └───────────────────────────────────────┘
                              ▲                   │
                              │                   │
                   publish ───┘                   └─── consume
                (order-service)               (product-service)
```

## Tech Stack

| Technology | Purpose |
|------------|---------|
| Go 1.22 | Primary language |
| PostgreSQL 16 | Database (one per service) |
| RabbitMQ | Message broker for async events |
| SQLC | Type-safe SQL code generation |
| Goose | Database migrations |
| JWT (HS256) | Authentication tokens |
| Argon2id | Password hashing |
| Docker Compose | Container orchestration |

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Go 1.22+ (for CLI and local development)
- [Goose](https://github.com/pressly/goose) (for migrations)

### 1. Start Services

```bash
# Clone and enter the project
cd scalable-ecommerce

# Start all services
docker compose up -d

# Verify all containers are running
docker compose ps
```

### 2. Run Migrations

```bash
# User service
goose -dir services/user-service/sql/schema postgres \
  "postgres://postgres:postgres@localhost:5433/users?sslmode=disable" up

# Product service
goose -dir services/product-service/sql/schema postgres \
  "postgres://postgres:postgres@localhost:5434/products?sslmode=disable" up

# Cart service
goose -dir services/cart-service/sql/schema postgres \
  "postgres://postgres:postgres@localhost:5435/carts?sslmode=disable" up

# Order service
goose -dir services/order-service/sql/schema postgres \
  "postgres://postgres:postgres@localhost:5436/orders?sslmode=disable" up
```

### 3. Build and Run CLI

```bash
cd services/cli
go build -o cli .
./cli
```

### 4. Demo Credentials

| Role | Email | Password |
|------|-------|----------|
| Admin | `admin@example.com` | `admin123` |

Admin users can access the "Manage Products" menu to add/delete products.

## Usage

### CLI Demo

The CLI provides an interactive way to demo the entire system:

```
========================================
       ECOM CLI - E-Commerce Store     
========================================

--- Welcome ---
1. Login
2. Register
3. Exit
```

**Customer Flow:**
1. Register a new account or login
2. Browse products
3. Add items to cart
4. Checkout (creates order, decrements stock via RabbitMQ)
5. View orders
6. Cancel order (restores stock via RabbitMQ)

**Admin Flow:**
1. Login as `admin@example.com`
2. Select "Admin: Manage Products"
3. Add new products (name, description, price, stock)
4. Delete existing products

### API Examples

**Register a User:**
```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password123"}'
```

**Login:**
```bash
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password123"}'
```

**Get Products:**
```bash
curl http://localhost:8080/api/products
```

**Add to Cart:**
```bash
curl -X POST http://localhost:8080/api/cart/items \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"product_id": "<product-uuid>", "quantity": 2}'
```

**Create Order:**
```bash
curl -X POST http://localhost:8080/api/orders \
  -H "Authorization: Bearer <token>"
```

**Cancel Order:**
```bash
curl -X DELETE http://localhost:8080/api/orders/<order-id> \
  -H "Authorization: Bearer <token>"
```

**Create Product (Admin):**
```bash
curl -X POST http://localhost:8080/admin/products \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin-token>" \
  -d '{"name": "Gaming Laptop", "description": "High performance", "price_cents": 149999, "stock": 10}'
```

## API Endpoints

### Public Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/users` | Register a new user |
| POST | `/api/login` | Login and get JWT token |
| POST | `/api/refresh` | Refresh access token |
| POST | `/api/revoke` | Revoke refresh token (logout) |
| GET | `/api/products` | List all active products |
| GET | `/api/products/{id}` | Get product by ID |

### Protected Endpoints (Auth Required)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/me` | Get current user info |
| GET | `/api/cart` | Get user's cart |
| POST | `/api/cart/items` | Add item to cart |
| PATCH | `/api/cart/items/{id}` | Update item quantity |
| DELETE | `/api/cart/items/{id}` | Remove item from cart |
| DELETE | `/api/cart` | Clear entire cart |
| POST | `/api/orders` | Create order from cart |
| GET | `/api/orders` | List user's orders |
| GET | `/api/orders/{id}` | Get order by ID |
| DELETE | `/api/orders/{id}` | Cancel order |

### Admin Endpoints (Admin Role Required)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/admin/products` | Create a new product |
| PATCH | `/admin/products/{id}` | Update a product |
| DELETE | `/admin/products/{id}` | Delete a product |

## Services Overview

### API Gateway (`services/api-gateway`)
Entry point for all requests. Handles JWT validation, role-based authorization, and request routing to downstream services.

### User Service (`services/user-service`)
Manages user registration, authentication, and JWT token lifecycle. Passwords are hashed using Argon2id.

### Product Service (`services/product-service`)
Handles product catalog CRUD operations. Also consumes RabbitMQ messages to update stock when orders are created or cancelled.

### Cart Service (`services/cart-service`)
Manages shopping carts. Fetches current prices from product-service when items are added.

### Order Service (`services/order-service`)
Creates orders from cart contents and publishes events to RabbitMQ. Supports order cancellation.

## Event-Driven Architecture

Stock updates are handled asynchronously via RabbitMQ to decouple the order and product services:

```
┌──────────────┐     order.created      ┌─────────────────┐
│Order Service │ ──────────────────────▶│ Product Service │
│  (publisher) │                        │   (consumer)    │
│              │     order.cancelled    │                 │
│              │ ──────────────────────▶│  Updates stock  │
└──────────────┘                        └─────────────────┘
```

**Flow:**
1. User creates order → order-service publishes `order.created` event
2. product-service consumes event → decrements stock for each item
3. User cancels order → order-service publishes `order.cancelled` event
4. product-service consumes event → restores stock for each item

**Benefits:**
- Services are decoupled (order-service doesn't call product-service directly)
- Resilient to temporary failures (messages are queued)
- Scalable (can add more consumers if needed)

## Environment Variables

### API Gateway
| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `SECRET_KEY` | JWT signing key | - |
| `USER_SERVICE_URL` | User service URL | `http://localhost:8081` |
| `PRODUCT_SERVICE_URL` | Product service URL | `http://localhost:8082` |
| `CART_SERVICE_URL` | Cart service URL | `http://localhost:8083` |
| `ORDER_SERVICE_URL` | Order service URL | `http://localhost:8084` |

### User Service
| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8081` |
| `SECRET_KEY` | JWT signing key | - |
| `DB_URL` | PostgreSQL connection string | - |

### Product Service
| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8082` |
| `DB_URL` | PostgreSQL connection string | - |
| `RABBITMQ_URL` | RabbitMQ connection string | - |

### Cart Service
| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8083` |
| `DB_URL` | PostgreSQL connection string | - |
| `PRODUCT_SERVICE_URL` | Product service URL | - |

### Order Service
| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8084` |
| `DB_URL` | PostgreSQL connection string | - |
| `CART_SERVICE_URL` | Cart service URL | - |
| `PRODUCT_SERVICE_URL` | Product service URL | - |
| `RABBITMQ_URL` | RabbitMQ connection string | - |

## Development

### Regenerate Database Code

```bash
cd services/user-service && sqlc generate
cd services/product-service && sqlc generate
cd services/cart-service && sqlc generate
cd services/order-service && sqlc generate
```

### Rebuild Services

```bash
docker compose up -d --build
```

### View Logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f order-service
```

### RabbitMQ Management UI

Access the RabbitMQ management dashboard at `http://localhost:15672`

- Username: `guest`
- Password: `guest`

### Stop Services

```bash
docker compose down

# Remove volumes (reset databases)
docker compose down -v
```

## Contributing

### Clone the Repository

```bash
git clone https://github.com/mnhsh/scalable-ecommerce.git
cd scalable-ecommerce
```

### Start the Services

```bash
docker compose up -d
```

### Run Migrations

```bash
goose -dir services/user-service/sql/schema postgres \
  "postgres://postgres:postgres@localhost:5433/users?sslmode=disable" up

goose -dir services/product-service/sql/schema postgres \
  "postgres://postgres:postgres@localhost:5434/products?sslmode=disable" up

goose -dir services/cart-service/sql/schema postgres \
  "postgres://postgres:postgres@localhost:5435/carts?sslmode=disable" up

goose -dir services/order-service/sql/schema postgres \
  "postgres://postgres:postgres@localhost:5436/orders?sslmode=disable" up
```

### Build the CLI

```bash
cd services/cli
go build -o cli .
```

### Run Tests

```bash
go test ./services/...
```

### Regenerate Database Code

After modifying SQL queries:

```bash
cd services/<service-name> && sqlc generate
```

### Submit a Pull Request

If you'd like to contribute, please fork the repository and open a pull request to the `main` branch.
