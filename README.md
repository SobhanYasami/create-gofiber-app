# create-gofiber-app

> `cargo new` / `create-next-app` for Go — scaffolds a production-ready
> [GoFiber v2](https://gofiber.io) REST API in under a minute.

---

## Table of contents

1. [Install](#1-install)
2. [Usage](#2-usage)
3. [Repo structure (important for contributors)](#3-repo-structure-important-for-contributors)
4. [Generated project layout](#4-generated-project-layout)
5. [Directory reference](#5-directory-reference)
6. [Stack matrix](#6-stack-matrix)
7. [Architectural guarantees](#7-architectural-guarantees)
8. [Contributing](#8-contributing)

---

## 1. Install

### Prerequisites

| Tool | Min version |
|------|-------------|
| Go   | 1.21        |

### Step 1 — install the binary

```bash
go install github.com/SobhanYasami/create-gofiber-app@latest
```

### Step 2 — add Go bin directory to PATH

The binary lands in `$(go env GOPATH)/bin` (usually `~/go/bin`).
If you see "command not found", run:

```bash
# bash / zsh — add to ~/.bashrc or ~/.zshrc then reload
export PATH="$PATH:$(go env GOPATH)/bin"

# fish
fish_add_path (go env GOPATH)/bin

# verify it works
which create-gofiber-app
```

---

## 2. Usage

```bash
# Pass the project name as an argument (skips the name prompt)
create-gofiber-app myapp

# Fully interactive — prompts for everything
create-gofiber-app
```

The CLI walks through three prompts:

```
  create-gofiber-app  GoFiber v2 · go 1.24 · production scaffold

  ┌─ Project name ──────────────────┐
  │  myapp                          │
  └─────────────────────────────────┘

  ┌─ Go module path ────────────────┐
  │  github.com/you/myapp           │
  └─────────────────────────────────┘

  ┌─ Database ──────────────────────┐
  │  ● PostgreSQL                   │
  │    MySQL                        │
  │    MongoDB                      │
  │    SQLite                       │
  └─────────────────────────────────┘

  ┌─ Data layer ────────────────────┐
  │  ● GORM  (ORM, auto-migrate)    │
  │    sqlx  (query builder)        │
  │    pgx   (raw driver, no ORM)   │
  └─────────────────────────────────┘
```

Files written:

```
  CREATE  go.mod
  CREATE  .env.example
  CREATE  .gitignore
  CREATE  Makefile
  CREATE  README.md
  CREATE  cmd/api/main.go
  CREATE  internal/config/config.go
  CREATE  internal/platform/db/db.go
  CREATE  internal/domain/user/user.go
  CREATE  internal/domain/user/repository.go
  CREATE  internal/repository/user_repo.go
  CREATE  internal/handler/user_handler.go
  CREATE  internal/server/server.go
  CREATE  internal/middleware/auth.go
  CREATE  docker-compose.yml           (skipped for SQLite)
  CREATE  migrations/000001_init.sql   (SQL databases only)

  ✓ Done!  16 files in ./myapp

  Next steps:
    cd myapp
    docker compose up -d
    go mod tidy
    cp .env.example .env
    go run cmd/api/main.go
```

---

## 3. Repo structure (important for contributors)

```
create-gofiber-app/          ← repo root  (package main)
├── main.go
├── go.mod
├── go.sum                   ← MUST be committed (run go mod tidy to generate)
├── README.md
└── generator/               ← MUST be a subdirectory  (package generator)
    ├── doc.go
    ├── config.go
    ├── generator.go
    └── files.go
```

### Why does structure matter?

Go requires **one package per directory**. `main.go` is `package main`.
`generator/*.go` is `package generator`. If you flatten them into the same
directory you get a compile error:

```
found packages generator (config.go) and main (main.go) in ...
```

### Setting up the repo from scratch

```bash
git clone https://github.com/SobhanYasami/create-gofiber-app
cd create-gofiber-app

# Move generator files into the subdirectory if they ended up at root
mkdir -p generator
git mv config.go generator/
git mv generator.go generator/
git mv files.go generator/
git mv doc.go generator/     # if present

# Generate go.sum — required for go install to work on fresh machines
go mod tidy

git add .
git commit -m "fix: correct package layout and add go.sum"
git push
```

---

## 4. Generated project layout

```
myapp/
├── cmd/
│   └── api/
│       └── main.go              # Entrypoint: wire deps, start server, graceful shutdown
├── internal/
│   ├── config/
│   │   └── config.go            # Typed config from env vars
│   ├── domain/
│   │   └── user/
│   │       ├── user.go          # User entity + CreateUserRequest, UpdateUserRequest
│   │       └── repository.go    # Repository interface — zero DB coupling
│   ├── handler/
│   │   └── user_handler.go      # Fiber CRUD handlers for /api/v1/users
│   ├── middleware/
│   │   └── auth.go              # JWT Bearer auth (HS256, alg-substitution safe)
│   ├── platform/
│   │   └── db/
│   │       └── db.go            # DB connection factory → (conn, closeFunc, error)
│   ├── repository/
│   │   └── user_repo.go         # Concrete DB implementation
│   └── server/
│       └── server.go            # Fiber app wiring + route registration
├── migrations/
│   └── 000001_init.sql          # SQL schema (SQL databases only)
├── .env.example
├── .gitignore
├── docker-compose.yml           # Not generated for SQLite
├── go.mod
├── Makefile
└── README.md
```

---

## 5. Directory reference

### `cmd/api/` — Application entrypoint

Calls `config.Load()`, opens the DB via `platform/db.New()`, wires the repo
and handler, starts Fiber, and blocks until `SIGINT`/`SIGTERM`.

`main.go` is **identical for all 10 DB+layer combos** — Go's `:=` type
inference selects the correct concrete type from `db.New()` automatically:

```go
dbConn, closeDB, err := db.New(cfg)   // *gorm.DB, *pgxpool.Pool, *mongo.Database, etc.
defer closeDB()
userRepo    := repository.NewUserRepository(dbConn)
userHandler := handler.NewUserHandler(userRepo)
app         := server.New(cfg, userHandler)
```

---

### `internal/config/` — Environment configuration

Reads env vars at startup; crashes early on missing required values.

```bash
# .env.example (PostgreSQL)
PORT=3000
JWT_SECRET=change-me-in-production
DATABASE_DSN=postgresql://postgres:postgres@localhost:5432/appdb?sslmode=disable
```

```go
cfg := config.Load()
fmt.Println(cfg.Port)        // "3000"
fmt.Println(cfg.DatabaseDSN) // "postgresql://..."
```

---

### `internal/domain/user/` — Entities and contracts

The `Repository` interface is the dependency-inversion boundary.
All ID parameters are `string` so handlers stay DB-agnostic:

```go
type Repository interface {
    Create(ctx context.Context, u *User) error
    GetByID(ctx context.Context, id string) (*User, error)
    GetByEmail(ctx context.Context, email string) (*User, error)
    Update(ctx context.Context, id string, req UpdateUserRequest) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, limit, offset int) ([]User, error)
}
```

SQL repos parse `id string` → `int64`; MongoDB repos parse to `primitive.ObjectID`.

---

### `internal/platform/db/` — Connection factory

Returns a typed connection handle + a teardown closure:

```
postgres + gorm  → (*gorm.DB,        func(), error)
postgres + pgx   → (*pgxpool.Pool,   func(), error)
postgres + sqlx  → (*sqlx.DB,        func(), error)
mysql    + gorm  → (*gorm.DB,        func(), error)
mysql    + sqlx  → (*sqlx.DB,        func(), error)
mysql    + raw   → (*sql.DB,         func(), error)
mongodb  + raw   → (*mongo.Database, func(), error)
mongodb  + qmgo  → (*qmgo.Database,  func(), error)
sqlite   + gorm  → (*gorm.DB,        func(), error)
sqlite   + raw   → (*sql.DB,         func(), error)
```

---

### `internal/repository/` — Storage implementations

Implements `domain/user.Repository` for each DB+layer combo.
ID conversion examples:

```go
// SQL (postgres + pgx raw)
uid, _ := strconv.ParseInt(id, 10, 64)
r.db.Exec(ctx, "SELECT ... WHERE id = $1", uid)

// MongoDB (mongo-driver)
oid, _ := primitive.ObjectIDFromHex(id)
r.coll.FindOne(ctx, bson.M{"_id": oid}).Decode(&u)
```

---

### `internal/handler/` — HTTP layer

Thin Fiber handlers: parse body → call repository → return JSON.
All routes require `Authorization: Bearer <jwt>`.

```
POST   /api/v1/users         create a user
GET    /api/v1/users         list users (?limit=20&offset=0)
GET    /api/v1/users/:id     get by id
PUT    /api/v1/users/:id     update
DELETE /api/v1/users/:id     delete
```

---

### `internal/middleware/` — JWT auth

HS256 Bearer token validation. Explicitly asserts `*jwt.SigningMethodHMAC`
to block `alg: none` substitution attacks (CVE class: JWT algorithm confusion):

```go
if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
    return nil, fiber.NewError(401, "unexpected signing algorithm")
}
```

---

### `internal/server/` — Fiber wiring

```go
app := server.New(cfg, userHandler)
app.Listen(":3000")
```

Registers global `recover` + `logger` middleware, mounts all user routes.

---

### `migrations/` — SQL schema

Compatible with [golang-migrate](https://github.com/golang-migrate/migrate)
and [goose](https://github.com/pressly/goose).

```sql
-- PostgreSQL example
CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

For GORM stacks `AutoMigrate` runs on startup, so this file is for version-
controlled schema history.

---

## 6. Stack matrix

| DB         | Layer        | Connection type       | Notes                           |
|------------|--------------|-----------------------|---------------------------------|
| PostgreSQL | GORM         | `*gorm.DB`            | AutoMigrate; `$1` placeholders  |
| PostgreSQL | sqlx         | `*sqlx.DB`            | `RETURNING` in Create           |
| PostgreSQL | pgx raw      | `*pgxpool.Pool`       | pgxpool; `$1` placeholders      |
| MySQL      | GORM         | `*gorm.DB`            | AutoMigrate; `?` placeholders   |
| MySQL      | sqlx         | `*sqlx.DB`            | `LastInsertId` in Create        |
| MySQL      | database/sql | `*sql.DB`             | Raw; `?` placeholders           |
| MongoDB    | mongo-driver | `*mongo.Database`     | ObjectID hex as public ID       |
| MongoDB    | qmgo         | `*qmgo.Database`      | qmgo fluent API                 |
| SQLite     | GORM         | `*gorm.DB`            | `MaxOpenConns(1)` enforced      |
| SQLite     | database/sql | `*sql.DB`             | modernc pure-Go driver          |

---

## 7. Architectural guarantees

These hold for **every generated project**, regardless of which stack you pick:

| Guarantee | How |
|-----------|-----|
| `cmd/api/main.go` identical across all stacks | `:=` type inference on `db.New()` |
| Domain has zero DB coupling | `repository.go` uses only stdlib types |
| Handlers are stack-agnostic | `string` IDs, repos handle conversion |
| `db.New` always deferrable | `(T, func(), error)` — close func is always valid |
| JWT is alg-safe | Explicit `*jwt.SigningMethodHMAC` assertion |
| SQLite is concurrency-safe | `SetMaxOpenConns(1)` |

---

## 8. Contributing

```bash
git clone https://github.com/SobhanYasami/create-gofiber-app
cd create-gofiber-app
go mod tidy
go run . myapp              # generate a test project
ls myapp/                   # inspect output
```

To add a new database or query layer, see the step-by-step guide in
[`generator/doc.go`](generator/doc.go).
