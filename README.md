# Microservices Order & Product Management System (Docker-Free)

A production-grade, distributed microservices architecture in Go containing:
1. **Order Service** (Port `8080`): Handles orders, JWT authentication/authorization, database operations, circuit breaking, retries, metrics, and Swagger UI.
2. **Product Service** (Port `8081`): An autonomous microservice providing product catalog and pricing APIs with simulated network health endpoints.
3. **Swagger API Documentation & Interactive Testing**: Embedded Swagger UI at `http://localhost:8080/swagger/index.html`.
4. **Resilience**: Sony gobreaker circuit breaker, exponential backoff retries, context cancellation, and fail-safe fallback pricing.
5. **Observability**: Structured JSON logging, Prometheus metrics (`/metrics`), health checks (`/health/live`, `/health/ready`).
6. **No Docker Requirement**: Standalone Go binaries executed natively or managed via Linux systemd.

---

## Architecture

```
                       Client / SoapUI / Swagger UI
                                    │
                       (JWT Auth: Bearer <token>)
                                    │
                                    ▼
               ┌─────────────────────────────────────────┐
               │         Order Microservice              │
               │            (Port 8080)                  │
               │  - Controllers & JWT Middleware         │
               │  - Swagger UI: /swagger/index.html      │
               │  - Resilience (Circuit Breaker / Retry) │
               │  - PostgreSQL Pool                      │
               └────────────┬──────────────────┬─────────┘
                            │                  │
         (HTTP Rest Calls)  │                  │ (SQL Queries)
                            ▼                  ▼
              ┌─────────────────────┐    ┌────────────┐
              │ Product Microservice│    │ PostgreSQL │
              │    (Port 8081)      │    │  (Port     │
              │  - Catalog & Price  │    │   5432)    │
              └─────────────────────┘    └────────────┘
```

---

## 1. Quick Start (Running Both Microservices Locally)

### Prerequisites:
- Go 1.22+
- PostgreSQL running locally on port 5432 with `orderdb` database created.

### Step 1: Start PostgreSQL Schema
```bash
psql -U postgres -d orderdb -f database/migrations/001_create_orders_table.sql
```

### Step 2: Start Product Microservice (Terminal 1)
```bash
cd product-service
go run main.go
# Runs on http://localhost:8081
```

### Step 3: Start Order Microservice (Terminal 2)
```bash
# In the project root:
cp .env.example .env
go run main.go
# Runs on http://localhost:8080
```

---

## 2. Interactive Swagger Testing

Once the Order Service is running, open your browser:
👉 **`http://localhost:8080/swagger/index.html`**

You can:
1. Use `POST /api/v1/auth/login` with:
   - Admin: `{"username": "admin", "password": "admin123"}`
   - User: `{"username": "user", "password": "user123"}`
2. Click the **Authorize 🔓** button at the top of Swagger UI and enter:
   `Bearer <YOUR_GENERATED_JWT_TOKEN>`
3. Execute all 4 business functions and health APIs directly inside Swagger.

---

## 3. Running Concurrency & Race Detector Tests
```bash
go test -v -race ./...
```
