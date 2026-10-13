# 🚀 PMO AGI Backend

<p align="center">
  **Powering Project Management with a Go-based REST API**
  
  A backend service for the PMO AGI application, built with Go, Chi Router, and PostgreSQL.
</p>

---

## 📌 Overview

**PMO AGI Backend** is the server-side component of the PMO AGI application. It provides the foundation for communication between the frontend and the database through HTTP REST APIs using JSON.

The project is organized into modular components to keep configuration, database access, domain logic, middleware, HTTP routing, and response handling maintainable as the application grows.

## ✨ Key Features

- **REST API** — HTTP-based communication using JSON.
- **Authentication** — JWT Bearer token integration.
- **Authorization** — Middleware for authentication and permission checks.
- **Project Management Domain** — Dedicated domain module for project-related functionality.
- **PostgreSQL Integration** — Relational database connectivity through the PostgreSQL driver.
- **Modular Architecture** — Separation of configuration, database, domain, middleware, routing, and response layers.

> Feature availability depends on the current implementation. This documentation describes the architecture and intended responsibilities of the visible project modules.

## 🧰 Tech Stack

| Technology                                   | Purpose                      |
| -------------------------------------------- | ---------------------------- |
| [Go](https://go.dev/)                        | Backend programming language |
| [Chi](https://github.com/go-chi/chi)         | HTTP routing                 |
| [PostgreSQL 16](https://www.postgresql.org/) | Relational database          |
| JWT                                          | Bearer-token authentication  |
| `pgx`                                        | PostgreSQL driver            |
| [Next.js](https://nextjs.org/)               | Frontend application         |

## 🏗️ Architecture

```text
┌─────────────────────────┐
│     PMO AGI Frontend    │
│       Next.js :3000     │
└────────────┬────────────┘
             │
             │ HTTP/REST · JSON
             │ Authorization: Bearer <JWT>
             ▼
┌─────────────────────────┐
│     PMO AGI Backend     │
│        Go / Chi         │
│         :8080           │
├─────────────────────────┤
│ Router & Middleware     │
│ Domain / Business Logic │
│ Database Access         │
│ Standardized Responses  │
└────────────┬────────────┘
             │
             │ PostgreSQL Driver
             ▼
┌─────────────────────────┐
│       PostgreSQL 16     │
│         :5432           │
│       Database:         │
│         pmo_agi          │
└─────────────────────────┘
```

## 📂 Project Structure

```text
pmo-agi-be/
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── config/               # Application configuration
│   ├── db/                   # Database connectivity
│   ├── domain/
│   │   ├── auth/              # Authentication domain
│   │   ├── dashboard/         # Dashboard domain
│   │   ├── master/            # Master data domain
│   │   └── project/           # Project domain
│   ├── middleware/
│   │   ├── auth.go            # Authentication middleware
│   │   └── permission.go      # Permission middleware
│   ├── response/
│   │   └── response.go        # API response utilities
│   └── router/
│       └── router.go          # HTTP route registration
├── .env.example               # Environment variable template
├── go.mod                     # Go dependencies
├── go.sum                     # Dependency checksums
├── Makefile                   # Development commands
└── README.md
```

## ⚙️ Prerequisites

Before running the application, make sure you have:

- Go installed, compatible with the version specified in `go.mod`.
- PostgreSQL 16, either installed locally or running through Docker.
- Git.
- A configured environment file based on `.env.example`.

## 🚀 Getting Started

### 1. Clone the repository

```bash
git clone https://10.0.7.181/git/Nanda_Surya/pmo-agi-be.git
cd pmo-agi-be
```

### 2. Configure environment variables

Create a local `.env` file from the provided template.

**PowerShell:**

```powershell
Copy-Item .env.example .env
```

Update the variables in `.env` according to your local database configuration. Do not commit credentials or production secrets.

### 3. Start PostgreSQL

If the project includes a configured Docker Compose service for the database, start it using the appropriate Compose command. Otherwise, make sure your local PostgreSQL service is running and the database has been created.

The development architecture uses PostgreSQL on port `5432` with the database name `pmo_agi`.

### 4. Prepare the database

Use the project's database schema and migration or initialization process to prepare the required tables, functions, indexes, and seed data.

Ensure the configured database user has the necessary privileges before starting the backend.

### 5. Download Go dependencies

```bash
go mod download
```

### 6. Run the backend

Using the project's Makefile:

```bash
make run
```

Alternatively, run the server entry point directly:

```bash
go run ./cmd/server
```

The development architecture expects the backend to listen on port `8080`. Confirm the effective port in the application configuration.

## 🔐 Authentication & Authorization

The frontend communicates with protected API routes using a JWT Bearer token.

```http
Authorization: Bearer <your-jwt-token>
```

The backend contains separate middleware modules for authentication and permission checks. Route-level authorization should be verified against the current implementation before relying on it for production access control.

## 🛡️ Security Notes

- Never commit `.env`, passwords, database credentials, JWT secrets, or private keys.
- Keep `.env.example` limited to variable names and safe example values.
- Use a dedicated database account with only the privileges the application requires.
- Keep production secrets outside the source repository.
- Validate authentication and authorization on the backend, not only in the frontend.
- Do not expose internal error details or sensitive data in API responses.

## 🧪 Development

Download dependencies:

```bash
go mod download
```

Format Go source code:

```bash
gofmt -w ./cmd ./internal
```

Run Go tests:

```bash
go test ./...
```

Run static analysis:

```bash
go vet ./...
```

## 🗺️ Development Roadmap

- [ ] Complete and verify authentication flows.
- [ ] Implement and verify permission-based access control.
- [ ] Develop project and dashboard API functionality.
- [ ] Standardize API response and error handling.
- [ ] Add automated unit and integration tests.
- [ ] Document API endpoints and request/response examples.
- [ ] Prepare deployment and production configuration.

_Roadmap items should be updated to reflect actual implementation progress._

## 🤝 Contributing

1. Create a feature branch.
2. Keep changes focused and follow the existing project structure.
3. Run formatting, tests, and static analysis before submitting changes.
4. Document new environment variables and API behavior.
5. Never include secrets or local configuration files in commits.

## 📄 License

Internal project. Distribution and reuse are subject to the organization's policies and repository access rules.
