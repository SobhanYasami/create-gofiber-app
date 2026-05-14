package main

import (
	"fmt"
	"strings"
)

// bt is used to insert a backtick into concatenated string templates.
const bt = "`"

// r replaces MODULE and APPNAME placeholders in a template string.
func r(tpl string, c Config) string {
	return strings.NewReplacer(
		"MODULE", c.ModuleName,
		"APPNAME", c.AppName,
	).Replace(tpl)
}

// ════════════════════════════════════════════════════════════════════════════
// go.mod
// ════════════════════════════════════════════════════════════════════════════

func genGoMod(c Config) string {
	var deps strings.Builder
	deps.WriteString("\tgithub.com/gofiber/fiber/v2 v2.52.5\n")
	deps.WriteString("\tgithub.com/golang-jwt/jwt/v5 v5.2.1\n")
	deps.WriteString("\tgolang.org/x/crypto v0.23.0\n")

	switch {
	case c.IsGORM():
		deps.WriteString("\tgorm.io/gorm v1.25.10\n")
		switch {
		case c.IsPostgres():
			deps.WriteString("\tgorm.io/driver/postgres v1.5.7\n")
		case c.IsMySQL():
			deps.WriteString("\tgorm.io/driver/mysql v1.5.4\n")
		case c.IsSQLite():
			deps.WriteString("\tgorm.io/driver/sqlite v1.5.5\n")
		}
	case c.IsSQLx():
		deps.WriteString("\tgithub.com/jmoiron/sqlx v1.3.5\n")
		if c.IsPostgres() {
			deps.WriteString("\tgithub.com/jackc/pgx/v5 v5.5.5\n")
		} else if c.IsMySQL() {
			deps.WriteString("\tgithub.com/go-sql-driver/mysql v1.7.1\n")
		}
	case c.IsRaw():
		switch {
		case c.IsPostgres():
			deps.WriteString("\tgithub.com/jackc/pgx/v5 v5.5.5\n")
		case c.IsMySQL():
			deps.WriteString("\tgithub.com/go-sql-driver/mysql v1.7.1\n")
		case c.IsMongoDB():
			deps.WriteString("\tgo.mongodb.org/mongo-driver v1.14.0\n")
		case c.IsSQLite():
			deps.WriteString("\tmodernc.org/sqlite v1.29.6\n")
		}
	case c.IsQMGO():
		deps.WriteString("\tgithub.com/qiniu/qmgo v1.1.8\n")
		deps.WriteString("\tgo.mongodb.org/mongo-driver v1.14.0\n")
	}

	return fmt.Sprintf("module %s\n\ngo 1.24\n\nrequire (\n%s)\n",
		c.ModuleName, deps.String())
}

// ════════════════════════════════════════════════════════════════════════════
// .env.example
// ════════════════════════════════════════════════════════════════════════════

func genEnvExample(c Config) string {
	var sb strings.Builder
	sb.WriteString("PORT=3000\n")
	sb.WriteString("JWT_SECRET=change-me-in-production\n\n")

	switch c.DB {
	case "postgres":
		sb.WriteString("# PostgreSQL connection string\n")
		sb.WriteString("DATABASE_DSN=postgresql://postgres:postgres@localhost:5432/appdb?sslmode=disable\n")
	case "mysql":
		sb.WriteString("# MySQL connection string\n")
		sb.WriteString("DATABASE_DSN=root:password@tcp(localhost:3306)/appdb?parseTime=true\n")
	case "mongodb":
		sb.WriteString("# MongoDB connection\n")
		sb.WriteString("MONGO_URI=mongodb://localhost:27017\n")
		sb.WriteString("MONGO_DB=appdb\n")
	case "sqlite":
		sb.WriteString("# SQLite database file path\n")
		sb.WriteString("SQLITE_PATH=./data.db\n")
	}
	return sb.String()
}

// ════════════════════════════════════════════════════════════════════════════
// .gitignore
// ════════════════════════════════════════════════════════════════════════════

func genGitIgnore() string {
	return `# Binaries
/bin/
*.exe
*.out

# Environment
.env
.env.local

# SQLite
*.db
*.db-shm
*.db-wal

# Go
vendor/
`
}

// ════════════════════════════════════════════════════════════════════════════
// Makefile
// ════════════════════════════════════════════════════════════════════════════

func genMakefile(c Config) string {
	return r(`.PHONY: run build test lint tidy

BIN := bin/APPNAME

run:
	go run cmd/api/main.go

build:
	go build -ldflags="-s -w" -o $(BIN) ./cmd/api

test:
	go test -race ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

.DEFAULT_GOAL := run
`, c)
}

// ════════════════════════════════════════════════════════════════════════════
// README.md
// ════════════════════════════════════════════════════════════════════════════

func genReadme(c Config) string {
	dbLabel := c.DBLabel()
	layerLabel := c.LayerLabel()

	dockerStep := ""
	if c.DB != "sqlite" {
		dockerStep = "\n    docker compose up -d     # start " + dbLabel + "\n"
	}

	return fmt.Sprintf(`# %s

GoFiber v2 backend API — scaffolded by create-gofiber-app.

## Stack

| Component   | Choice        |
|-------------|---------------|
| Framework   | GoFiber v2    |
| Database    | %s            |
| Data layer  | %s            |
| Auth        | JWT (HS256)   |

## Quick start
%s
    go mod tidy
    cp .env.example .env
    # edit .env with your credentials
    go run cmd/api/main.go

## Endpoints

All routes require "Authorization: Bearer <token>" except where noted.

    POST   /api/v1/users         create user
    GET    /api/v1/users         list users  ?limit=20&offset=0
    GET    /api/v1/users/:id     get user
    PUT    /api/v1/users/:id     update user
    DELETE /api/v1/users/:id     delete user

## Project layout

    cmd/api/         application entrypoint
    internal/
      config/        typed env-var config
      domain/user/   entity, request types, repository interface
      handler/       fiber request handlers
      middleware/     JWT auth
      platform/db/   connection setup
      repository/    concrete DB implementation
      server/        fiber app wiring + routes
    migrations/      SQL schema (SQL databases only)
`, c.AppName, dbLabel, layerLabel, dockerStep)
}

// ════════════════════════════════════════════════════════════════════════════
// cmd/api/main.go  (UNIFORM across all combos via type inference)
// ════════════════════════════════════════════════════════════════════════════

func genCmdMain(c Config) string {
	return r(`package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"MODULE/internal/config"
	"MODULE/internal/handler"
	"MODULE/internal/platform/db"
	"MODULE/internal/repository"
	"MODULE/internal/server"
)

func main() {
	cfg := config.Load()

	dbConn, closeDB, err := db.New(cfg)
	if err != nil {
		slog.Error("database init failed", "error", err)
		os.Exit(1)
	}
	defer closeDB()

	userRepo := repository.NewUserRepository(dbConn)
	userHandler := handler.NewUserHandler(userRepo)
	app := server.New(cfg, userHandler)

	go func() {
		slog.Info("server starting", "port", cfg.Port)
		if err := app.Listen(":" + cfg.Port); err != nil {
			slog.Error("listen error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(ctx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	slog.Info("server stopped")
}
`, c)
}

// ════════════════════════════════════════════════════════════════════════════
// internal/config/config.go
// ════════════════════════════════════════════════════════════════════════════

func genConfig(c Config) string {
	var fields, loads strings.Builder

	fields.WriteString("\tPort      string\n")
	fields.WriteString("\tJWTSecret string\n")

	loads.WriteString("\t\tPort:      getEnv(\"PORT\", \"3000\"),\n")
	loads.WriteString("\t\tJWTSecret: getEnv(\"JWT_SECRET\", \"change-me-in-production\"),\n")

	switch c.DB {
	case "postgres", "mysql":
		fields.WriteString("\tDatabaseDSN string\n")
		loads.WriteString("\t\tDatabaseDSN: getEnv(\"DATABASE_DSN\", \"\"),\n")
	case "mongodb":
		fields.WriteString("\tMongoURI string\n")
		fields.WriteString("\tMongoDB  string\n")
		loads.WriteString("\t\tMongoURI: getEnv(\"MONGO_URI\", \"mongodb://localhost:27017\"),\n")
		loads.WriteString("\t\tMongoDB:  getEnv(\"MONGO_DB\", \"appdb\"),\n")
	case "sqlite":
		fields.WriteString("\tSQLitePath string\n")
		loads.WriteString("\t\tSQLitePath: getEnv(\"SQLITE_PATH\", \"./data.db\"),\n")
	}

	return fmt.Sprintf(`package config

import "os"

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
%s}

// Load reads environment variables and returns a populated Config.
// Fails loudly on missing required vars (DATABASE_DSN, MONGO_URI) so that
// misconfigured deploys crash at startup rather than at first request.
func Load() *Config {
	return &Config{
%s	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
`, fields.String(), loads.String())
}

// ════════════════════════════════════════════════════════════════════════════
// internal/platform/db/db.go
// ════════════════════════════════════════════════════════════════════════════

func genPlatformDB(c Config) string {
	switch {
	case c.IsGORM() && c.IsPostgres():
		return genDBGORMPostgres(c)
	case c.IsGORM() && c.IsMySQL():
		return genDBGORMMySQL(c)
	case c.IsGORM() && c.IsSQLite():
		return genDBGORMSQLite(c)
	case c.IsSQLx() && c.IsPostgres():
		return genDBSQLxPostgres(c)
	case c.IsSQLx() && c.IsMySQL():
		return genDBSQLxMySQL(c)
	case c.IsRaw() && c.IsPostgres():
		return genDBRawPgx(c)
	case c.IsRaw() && c.IsMySQL():
		return genDBRawMySQL(c)
	case c.IsRaw() && c.IsSQLite():
		return genDBRawSQLite(c)
	case c.IsRaw() && c.IsMongoDB():
		return genDBRawMongo(c)
	case c.IsQMGO():
		return genDBQMGO(c)
	}
	return ""
}

func genDBGORMPostgres(c Config) string {
	return r(`package db

import (
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"MODULE/internal/config"
	"MODULE/internal/domain/user"
)

// New opens a GORM connection to PostgreSQL, configures the pool,
// runs AutoMigrate, and returns a teardown func.
func New(cfg *config.Config) (*gorm.DB, func(), error) {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("gorm open: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("get sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	if err := db.AutoMigrate(&user.User{}); err != nil {
		return nil, nil, fmt.Errorf("auto-migrate: %w", err)
	}
	slog.Info("database ready", "driver", "postgres+gorm")
	return db, func() { sqlDB.Close() }, nil
}
`, c)
}

func genDBGORMMySQL(c Config) string {
	return r(`package db

import (
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"MODULE/internal/config"
	"MODULE/internal/domain/user"
)

func New(cfg *config.Config) (*gorm.DB, func(), error) {
	db, err := gorm.Open(mysql.Open(cfg.DatabaseDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("gorm open: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("get sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	if err := db.AutoMigrate(&user.User{}); err != nil {
		return nil, nil, fmt.Errorf("auto-migrate: %w", err)
	}
	slog.Info("database ready", "driver", "mysql+gorm")
	return db, func() { sqlDB.Close() }, nil
}
`, c)
}

func genDBGORMSQLite(c Config) string {
	return r(`package db

import (
	"fmt"
	"log/slog"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"MODULE/internal/config"
	"MODULE/internal/domain/user"
)

func New(cfg *config.Config) (*gorm.DB, func(), error) {
	db, err := gorm.Open(sqlite.Open(cfg.SQLitePath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("gorm open: %w", err)
	}
	// SQLite supports only one writer at a time.
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(&user.User{}); err != nil {
		return nil, nil, fmt.Errorf("auto-migrate: %w", err)
	}
	slog.Info("database ready", "driver", "sqlite+gorm", "path", cfg.SQLitePath)
	return db, func() { sqlDB.Close() }, nil
}
`, c)
}

func genDBSQLxPostgres(c Config) string {
	return r(`package db

import (
	"fmt"
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver for database/sql
	"github.com/jmoiron/sqlx"
	"MODULE/internal/config"
)

func New(cfg *config.Config) (*sqlx.DB, func(), error) {
	db, err := sqlx.Connect("pgx", cfg.DatabaseDSN)
	if err != nil {
		return nil, nil, fmt.Errorf("sqlx connect: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	slog.Info("database ready", "driver", "postgres+sqlx")
	return db, func() { db.Close() }, nil
}
`, c)
}

func genDBSQLxMySQL(c Config) string {
	return r(`package db

import (
	"fmt"
	"log/slog"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"MODULE/internal/config"
)

func New(cfg *config.Config) (*sqlx.DB, func(), error) {
	db, err := sqlx.Connect("mysql", cfg.DatabaseDSN)
	if err != nil {
		return nil, nil, fmt.Errorf("sqlx connect: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	slog.Info("database ready", "driver", "mysql+sqlx")
	return db, func() { db.Close() }, nil
}
`, c)
}

func genDBRawPgx(c Config) string {
	return r(`package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"MODULE/internal/config"
)

func New(cfg *config.Config) (*pgxpool.Pool, func(), error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseDSN)
	if err != nil {
		return nil, nil, fmt.Errorf("parse dsn: %w", err)
	}
	poolCfg.MaxConns = 25

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("new pool: %w", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("ping: %w", err)
	}
	slog.Info("database ready", "driver", "postgres+pgx")
	return pool, func() { pool.Close() }, nil
}
`, c)
}

func genDBRawMySQL(c Config) string {
	return r(`package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"MODULE/internal/config"
)

func New(cfg *config.Config) (*sql.DB, func(), error) {
	db, err := sql.Open("mysql", cfg.DatabaseDSN)
	if err != nil {
		return nil, nil, fmt.Errorf("sql open: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("ping: %w", err)
	}
	slog.Info("database ready", "driver", "mysql+raw")
	return db, func() { db.Close() }, nil
}
`, c)
}

func genDBRawSQLite(c Config) string {
	return r(`package db

import (
	"database/sql"
	"fmt"
	"log/slog"

	_ "modernc.org/sqlite"
	"MODULE/internal/config"
)

func New(cfg *config.Config) (*sql.DB, func(), error) {
	db, err := sql.Open("sqlite", cfg.SQLitePath)
	if err != nil {
		return nil, nil, fmt.Errorf("sql open: %w", err)
	}
	// Single writer to avoid SQLITE_BUSY.
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("ping: %w", err)
	}
	slog.Info("database ready", "driver", "sqlite+raw", "path", cfg.SQLitePath)
	return db, func() { db.Close() }, nil
}
`, c)
}

func genDBRawMongo(c Config) string {
	return r(`package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"MODULE/internal/config"
)

// New connects to MongoDB and returns *mongo.Database (not the client) so
// repositories can call db.Collection() directly without knowing the URI.
func New(cfg *config.Config) (*mongo.Database, func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		return nil, nil, fmt.Errorf("mongo connect: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, nil, fmt.Errorf("mongo ping: %w", err)
	}
	slog.Info("database ready", "driver", "mongodb", "db", cfg.MongoDB)
	db := client.Database(cfg.MongoDB)
	return db, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		client.Disconnect(ctx) //nolint:errcheck
	}, nil
}
`, c)
}

func genDBQMGO(c Config) string {
	return r(`package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/qiniu/qmgo"
	"MODULE/internal/config"
)

func New(cfg *config.Config) (*qmgo.Database, func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := qmgo.NewClient(ctx, &qmgo.Config{Uri: cfg.MongoURI})
	if err != nil {
		return nil, nil, fmt.Errorf("qmgo connect: %w", err)
	}
	slog.Info("database ready", "driver", "mongodb+qmgo", "db", cfg.MongoDB)
	db := client.Database(cfg.MongoDB)
	return db, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		client.Close(ctx) //nolint:errcheck
	}, nil
}
`, c)
}

// ════════════════════════════════════════════════════════════════════════════
// internal/domain/user/user.go
// ════════════════════════════════════════════════════════════════════════════

func genDomainUser(c Config) string {
	if c.IsMongoDB() {
		return genDomainUserMongo(c)
	}
	return genDomainUserSQL(c)
}

func genDomainUserSQL(_ Config) string {
	// Struct tags: json (API), db (sqlx scanner), gorm (ORM metadata).
	// The gorm tags are just string metadata — no gorm import required here.
	return `package user

import (
	"errors"
	"time"
)

// ErrNotFound is returned by repository methods when a record does not exist.
var ErrNotFound = errors.New("user: record not found")

// User is the core domain entity. Struct tags support json, sqlx (db:),
// and GORM (gorm:) without importing any framework in this package.
type User struct {
	ID           int64     ` + bt + `json:"id"         db:"id"            gorm:"primaryKey;autoIncrement"` + bt + `
	Name         string    ` + bt + `json:"name"       db:"name"          gorm:"not null"` + bt + `
	Email        string    ` + bt + `json:"email"      db:"email"         gorm:"uniqueIndex;not null"` + bt + `
	PasswordHash string    ` + bt + `json:"-"          db:"password_hash" gorm:"column:password_hash"` + bt + `
	CreatedAt    time.Time ` + bt + `json:"created_at" db:"created_at"    gorm:"autoCreateTime"` + bt + `
	UpdatedAt    time.Time ` + bt + `json:"updated_at" db:"updated_at"    gorm:"autoUpdateTime"` + bt + `
}

type CreateUserRequest struct {
	Name     string ` + bt + `json:"name"` + bt + `
	Email    string ` + bt + `json:"email"` + bt + `
	Password string ` + bt + `json:"password"` + bt + `
}

type UpdateUserRequest struct {
	Name  string ` + bt + `json:"name"` + bt + `
	Email string ` + bt + `json:"email"` + bt + `
}
`
}

func genDomainUserMongo(_ Config) string {
	return `package user

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var ErrNotFound = errors.New("user: record not found")

// User stores ObjectID as the domain ID. The hex string representation is
// used as the public API identifier (URL params, JSON responses).
type User struct {
	ID           primitive.ObjectID ` + bt + `json:"id"         bson:"_id,omitempty"` + bt + `
	Name         string             ` + bt + `json:"name"       bson:"name"` + bt + `
	Email        string             ` + bt + `json:"email"      bson:"email"` + bt + `
	PasswordHash string             ` + bt + `json:"-"          bson:"password_hash"` + bt + `
	CreatedAt    time.Time          ` + bt + `json:"created_at" bson:"created_at"` + bt + `
	UpdatedAt    time.Time          ` + bt + `json:"updated_at" bson:"updated_at"` + bt + `
}

type CreateUserRequest struct {
	Name     string ` + bt + `json:"name"` + bt + `
	Email    string ` + bt + `json:"email"` + bt + `
	Password string ` + bt + `json:"password"` + bt + `
}

type UpdateUserRequest struct {
	Name  string ` + bt + `json:"name"` + bt + `
	Email string ` + bt + `json:"email"` + bt + `
}
`
}

// ════════════════════════════════════════════════════════════════════════════
// internal/domain/user/repository.go  (UNIFORM — string IDs)
// ════════════════════════════════════════════════════════════════════════════

func genDomainRepository(_ Config) string {
	return `package user

import "context"

// Repository defines persistence operations for User.
// All ID parameters are strings so handlers remain DB-agnostic:
//   - SQL databases  → parse to int64 inside the repo
//   - MongoDB        → parse to primitive.ObjectID inside the repo
type Repository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, id string, req UpdateUserRequest) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]User, error)
}
`
}

// ════════════════════════════════════════════════════════════════════════════
// internal/repository/user_repo.go
// ════════════════════════════════════════════════════════════════════════════

func genRepository(c Config) string {
	switch {
	case c.IsGORM():
		return genRepoGORM(c)
	case c.IsSQLx() && c.IsPostgres():
		return genRepoSQLxPostgres(c)
	case c.IsSQLx() && c.IsMySQL():
		return genRepoSQLxMySQL(c)
	case c.IsRaw() && c.IsPostgres():
		return genRepoRawPgx(c)
	case c.IsRaw() && (c.IsMySQL() || c.IsSQLite()):
		return genRepoRawSQL(c)
	case c.IsRaw() && c.IsMongoDB():
		return genRepoRawMongo(c)
	case c.IsQMGO():
		return genRepoQMGO(c)
	}
	return ""
}

func genRepoGORM(c Config) string {
	return r(`package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"gorm.io/gorm"
	"MODULE/internal/domain/user"
)

type userRepository struct{ db *gorm.DB }

// NewUserRepository returns a user.Repository backed by GORM.
func NewUserRepository(db *gorm.DB) user.Repository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, u *user.User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*user.User, error) {
	uid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, user.ErrNotFound
	}
	var u user.User
	if res := r.db.WithContext(ctx).First(&u, uid); res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return nil, user.ErrNotFound
		}
		return nil, res.Error
	}
	return &u, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var u user.User
	if res := r.db.WithContext(ctx).Where("email = ?", email).First(&u); res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return nil, user.ErrNotFound
		}
		return nil, res.Error
	}
	return &u, nil
}

func (r *userRepository) Update(ctx context.Context, id string, req user.UpdateUserRequest) error {
	uid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return user.ErrNotFound
	}
	res := r.db.WithContext(ctx).
		Model(&user.User{}).
		Where("id = ?", uid).
		Updates(map[string]any{"name": req.Name, "email": req.Email})
	if res.Error != nil {
		return fmt.Errorf("update user: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return user.ErrNotFound
	}
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	uid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return user.ErrNotFound
	}
	res := r.db.WithContext(ctx).Delete(&user.User{}, uid)
	if res.RowsAffected == 0 {
		return user.ErrNotFound
	}
	return res.Error
}

func (r *userRepository) List(ctx context.Context, limit, offset int) ([]user.User, error) {
	var users []user.User
	err := r.db.WithContext(ctx).
		Limit(limit).Offset(offset).Order("id ASC").
		Find(&users).Error
	return users, err
}
`, c)
}

func genRepoSQLxPostgres(c Config) string {
	return r(`package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/jmoiron/sqlx"
	"MODULE/internal/domain/user"
)

type userRepository struct{ db *sqlx.DB }

func NewUserRepository(db *sqlx.DB) user.Repository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, u *user.User) error {
	q := `+"`"+`INSERT INTO users (name, email, password_hash)
		  VALUES ($1, $2, $3)
		  RETURNING id, created_at, updated_at`+"`"+`
	return r.db.QueryRowContext(ctx, q, u.Name, u.Email, u.PasswordHash).
		Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*user.User, error) {
	uid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, user.ErrNotFound
	}
	var u user.User
	err = r.db.GetContext(ctx, &u, "SELECT * FROM users WHERE id = $1", uid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	return &u, err
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var u user.User
	err := r.db.GetContext(ctx, &u, "SELECT * FROM users WHERE email = $1", email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	return &u, err
}

func (r *userRepository) Update(ctx context.Context, id string, req user.UpdateUserRequest) error {
	uid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return user.ErrNotFound
	}
	res, err := r.db.ExecContext(ctx,
		"UPDATE users SET name=$1, email=$2, updated_at=NOW() WHERE id=$3",
		req.Name, req.Email, uid)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return user.ErrNotFound
	}
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	uid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return user.ErrNotFound
	}
	res, err := r.db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", uid)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return user.ErrNotFound
	}
	return nil
}

func (r *userRepository) List(ctx context.Context, limit, offset int) ([]user.User, error) {
	var users []user.User
	err := r.db.SelectContext(ctx, &users,
		"SELECT * FROM users ORDER BY id ASC LIMIT $1 OFFSET $2", limit, offset)
	return users, err
}
`, c)
}

func genRepoSQLxMySQL(c Config) string {
	return r(`package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/jmoiron/sqlx"
	"MODULE/internal/domain/user"
)

type userRepository struct{ db *sqlx.DB }

func NewUserRepository(db *sqlx.DB) user.Repository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, u *user.User) error {
	res, err := r.db.ExecContext(ctx,
		"INSERT INTO users (name, email, password_hash) VALUES (?, ?, ?)",
		u.Name, u.Email, u.PasswordHash)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	id, _ := res.LastInsertId()
	u.ID = id
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*user.User, error) {
	uid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, user.ErrNotFound
	}
	var u user.User
	err = r.db.GetContext(ctx, &u,
		r.db.Rebind("SELECT * FROM users WHERE id = ?"), uid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	return &u, err
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var u user.User
	err := r.db.GetContext(ctx, &u,
		r.db.Rebind("SELECT * FROM users WHERE email = ?"), email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	return &u, err
}

func (r *userRepository) Update(ctx context.Context, id string, req user.UpdateUserRequest) error {
	uid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return user.ErrNotFound
	}
	res, err := r.db.ExecContext(ctx,
		"UPDATE users SET name=?, email=?, updated_at=NOW() WHERE id=?",
		req.Name, req.Email, uid)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return user.ErrNotFound
	}
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	uid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return user.ErrNotFound
	}
	res, err := r.db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", uid)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return user.ErrNotFound
	}
	return nil
}

func (r *userRepository) List(ctx context.Context, limit, offset int) ([]user.User, error) {
	var users []user.User
	err := r.db.SelectContext(ctx, &users,
		"SELECT * FROM users ORDER BY id ASC LIMIT ? OFFSET ?", limit, offset)
	return users, err
}
`, c)
}

func genRepoRawPgx(c Config) string {
	return r(`package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"MODULE/internal/domain/user"
)

type userRepository struct{ db *pgxpool.Pool }

func NewUserRepository(db *pgxpool.Pool) user.Repository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, u *user.User) error {
	q := `+"`"+`INSERT INTO users (name, email, password_hash)
		  VALUES ($1, $2, $3)
		  RETURNING id, created_at, updated_at`+"`"+`
	return r.db.QueryRow(ctx, q, u.Name, u.Email, u.PasswordHash).
		Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*user.User, error) {
	uid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, user.ErrNotFound
	}
	var u user.User
	err = r.db.QueryRow(ctx,
		"SELECT id, name, email, password_hash, created_at, updated_at FROM users WHERE id = $1", uid).
		Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	return &u, err
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var u user.User
	err := r.db.QueryRow(ctx,
		"SELECT id, name, email, password_hash, created_at, updated_at FROM users WHERE email = $1", email).
		Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	return &u, err
}

func (r *userRepository) Update(ctx context.Context, id string, req user.UpdateUserRequest) error {
	uid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return user.ErrNotFound
	}
	tag, err := r.db.Exec(ctx,
		"UPDATE users SET name=$1, email=$2, updated_at=NOW() WHERE id=$3",
		req.Name, req.Email, uid)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return user.ErrNotFound
	}
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	uid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return user.ErrNotFound
	}
	tag, err := r.db.Exec(ctx, "DELETE FROM users WHERE id = $1", uid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return user.ErrNotFound
	}
	return nil
}

func (r *userRepository) List(ctx context.Context, limit, offset int) ([]user.User, error) {
	rows, err := r.db.Query(ctx,
		"SELECT id, name, email, password_hash, created_at, updated_at FROM users ORDER BY id ASC LIMIT $1 OFFSET $2",
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []user.User
	for rows.Next() {
		var u user.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
`, c)
}

func genRepoRawSQL(c Config) string {
	// Covers mysql+raw and sqlite+raw.
	// sqlite uses CURRENT_TIMESTAMP; mysql uses NOW() — both map to time.Time on scan.
	updatedAt := "NOW()"
	if c.IsSQLite() {
		updatedAt = "CURRENT_TIMESTAMP"
	}
	mod := c.ModuleName
	return fmt.Sprintf(`package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"%s/internal/domain/user"
)

type userRepository struct{ db *sql.DB }

func NewUserRepository(db *sql.DB) user.Repository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, u *user.User) error {
	res, err := r.db.ExecContext(ctx,
		"INSERT INTO users (name, email, password_hash) VALUES (?, ?, ?)",
		u.Name, u.Email, u.PasswordHash)
	if err != nil {
		return fmt.Errorf("create user: %%w", err)
	}
	id, _ := res.LastInsertId()
	u.ID = id
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*user.User, error) {
	uid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, user.ErrNotFound
	}
	var u user.User
	err = r.db.QueryRowContext(ctx,
		"SELECT id, name, email, password_hash, created_at, updated_at FROM users WHERE id = ?", uid).
		Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	return &u, err
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var u user.User
	err := r.db.QueryRowContext(ctx,
		"SELECT id, name, email, password_hash, created_at, updated_at FROM users WHERE email = ?", email).
		Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	return &u, err
}

func (r *userRepository) Update(ctx context.Context, id string, req user.UpdateUserRequest) error {
	uid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return user.ErrNotFound
	}
	res, err := r.db.ExecContext(ctx,
		"UPDATE users SET name=?, email=?, updated_at=%s WHERE id=?",
		req.Name, req.Email, uid)
	if err != nil {
		return fmt.Errorf("update user: %%w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return user.ErrNotFound
	}
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	uid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return user.ErrNotFound
	}
	res, err := r.db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", uid)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return user.ErrNotFound
	}
	return nil
}

func (r *userRepository) List(ctx context.Context, limit, offset int) ([]user.User, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, name, email, password_hash, created_at, updated_at FROM users ORDER BY id ASC LIMIT ? OFFSET ?",
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []user.User
	for rows.Next() {
		var u user.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
`, mod, updatedAt)
}

func genRepoRawMongo(c Config) string {
	return r(`package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"MODULE/internal/domain/user"
)

type userRepository struct {
	coll *mongo.Collection
}

func NewUserRepository(db *mongo.Database) user.Repository {
	return &userRepository{coll: db.Collection("users")}
}

func (r *userRepository) Create(ctx context.Context, u *user.User) error {
	u.ID = primitive.NewObjectID()
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	_, err := r.coll.InsertOne(ctx, u)
	return err
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*user.User, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, user.ErrNotFound
	}
	var u user.User
	if err := r.coll.FindOne(ctx, bson.M{"_id": oid}).Decode(&u); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, user.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var u user.User
	if err := r.coll.FindOne(ctx, bson.M{"email": email}).Decode(&u); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, user.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) Update(ctx context.Context, id string, req user.UpdateUserRequest) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return user.ErrNotFound
	}
	res, err := r.coll.UpdateOne(ctx,
		bson.M{"_id": oid},
		bson.M{"$set": bson.M{
			"name":       req.Name,
			"email":      req.Email,
			"updated_at": time.Now(),
		}})
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if res.MatchedCount == 0 {
		return user.ErrNotFound
	}
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return user.ErrNotFound
	}
	res, err := r.coll.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return user.ErrNotFound
	}
	return nil
}

func (r *userRepository) List(ctx context.Context, limit, offset int) ([]user.User, error) {
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "_id", Value: 1}})
	cur, err := r.coll.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var users []user.User
	return users, cur.All(ctx, &users)
}
`, c)
}

func genRepoQMGO(c Config) string {
	return r(`package repository

import (
	"context"
	"time"

	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"MODULE/internal/domain/user"
)

type userRepository struct {
	coll *qmgo.Collection
}

func NewUserRepository(db *qmgo.Database) user.Repository {
	return &userRepository{coll: db.Collection("users")}
}

func (r *userRepository) Create(ctx context.Context, u *user.User) error {
	u.ID = primitive.NewObjectID()
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	_, err := r.coll.InsertOne(ctx, u)
	return err
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*user.User, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, user.ErrNotFound
	}
	var u user.User
	if err := r.coll.Find(ctx, bson.M{"_id": oid}).One(&u); err != nil {
		if qmgo.IsErrNoDocuments(err) {
			return nil, user.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var u user.User
	if err := r.coll.Find(ctx, bson.M{"email": email}).One(&u); err != nil {
		if qmgo.IsErrNoDocuments(err) {
			return nil, user.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) Update(ctx context.Context, id string, req user.UpdateUserRequest) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return user.ErrNotFound
	}
	return r.coll.UpdateOne(ctx,
		bson.M{"_id": oid},
		bson.M{"$set": bson.M{
			"name":       req.Name,
			"email":      req.Email,
			"updated_at": time.Now(),
		}})
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return user.ErrNotFound
	}
	return r.coll.Remove(ctx, bson.M{"_id": oid})
}

func (r *userRepository) List(ctx context.Context, limit, offset int) ([]user.User, error) {
	var users []user.User
	err := r.coll.Find(ctx, bson.M{}).
		Skip(int64(offset)).
		Limit(int64(limit)).
		Sort("_id").
		All(&users)
	return users, err
}
`, c)
}

// ════════════════════════════════════════════════════════════════════════════
// internal/handler/user_handler.go  (UNIFORM)
// ════════════════════════════════════════════════════════════════════════════

func genHandler(c Config) string {
	return r(`package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"MODULE/internal/domain/user"
)

// UserHandler holds CRUD fiber handlers for the User resource.
type UserHandler struct {
	repo user.Repository
}

func NewUserHandler(repo user.Repository) *UserHandler {
	return &UserHandler{repo: repo}
}

func (h *UserHandler) Create(c *fiber.Ctx) error {
	var req user.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to hash password")
	}
	u := &user.User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hash),
	}
	if err := h.repo.Create(c.Context(), u); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(u)
}

func (h *UserHandler) GetByID(c *fiber.Ctx) error {
	u, err := h.repo.GetByID(c.Context(), c.Params("id"))
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "user not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(u)
}

func (h *UserHandler) Update(c *fiber.Ctx) error {
	var req user.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.repo.Update(c.Context(), c.Params("id"), req); err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "user not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *UserHandler) Delete(c *fiber.Ctx) error {
	if err := h.repo.Delete(c.Context(), c.Params("id")); err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "user not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *UserHandler) List(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit > 100 {
		limit = 100
	}
	users, err := h.repo.List(c.Context(), limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{
		"data":   users,
		"limit":  limit,
		"offset": offset,
	})
}
`, c)
}

// ════════════════════════════════════════════════════════════════════════════
// internal/server/server.go  (UNIFORM)
// ════════════════════════════════════════════════════════════════════════════

func genServer(c Config) string {
	return r(`package server

import (
	"github.com/gofiber/fiber/v2"
	flogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"MODULE/internal/config"
	"MODULE/internal/handler"
	"MODULE/internal/middleware"
)

// New builds and returns the fiber application with all routes registered.
func New(cfg *config.Config, userHandler *handler.UserHandler) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			msg := "internal server error"
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
				msg = e.Message
			}
			return c.Status(code).JSON(fiber.Map{"error": msg})
		},
	})

	app.Use(recover.New())
	app.Use(flogger.New(flogger.Config{
		Format: "${time} ${method} ${path} ${status} ${latency}\n",
	}))

	// Health check (no auth)
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// v1 API — all routes require a valid JWT
	v1 := app.Group("/api/v1")
	users := v1.Group("/users", middleware.JWT(cfg))
	users.Post("/", userHandler.Create)
	users.Get("/", userHandler.List)
	users.Get("/:id", userHandler.GetByID)
	users.Put("/:id", userHandler.Update)
	users.Delete("/:id", userHandler.Delete)

	return app
}
`, c)
}

// ════════════════════════════════════════════════════════════════════════════
// internal/middleware/auth.go  (UNIFORM)
// ════════════════════════════════════════════════════════════════════════════

func genMiddlewareAuth(c Config) string {
	return r(`package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"MODULE/internal/config"
)

// Claims is the JWT payload shape.
type Claims struct {
	UserID string `+bt+`json:"user_id"`+bt+`
	jwt.RegisteredClaims
}

// JWT returns a fiber middleware that validates Bearer tokens using HS256.
// The parsed UserID claim is stored in c.Locals("userID") for downstream handlers.
//
// Security notes:
//   - Rejects tokens signed with alg != HMAC (prevents alg:none attacks).
//   - Does not leak claim details in error responses.
func JWT(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		raw := c.Get("Authorization")
		if raw == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "missing authorization header")
		}
		parts := strings.SplitN(raw, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			return fiber.NewError(fiber.StatusUnauthorized, "authorization header must be 'Bearer <token>'")
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (any, error) {
			// Enforce HMAC family — reject RSA/ECDSA/none tokens.
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "unexpected signing algorithm")
			}
			return []byte(cfg.JWTSecret), nil
		}, jwt.WithExpirationRequired())

		if err != nil || !token.Valid {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
		}

		c.Locals("userID", claims.UserID)
		return c.Next()
	}
}
`, c)
}

// ════════════════════════════════════════════════════════════════════════════
// docker-compose.yml
// ════════════════════════════════════════════════════════════════════════════

func genDockerCompose(c Config) string {
	switch c.DB {
	case "postgres":
		return `services:
  postgres:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: appdb
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  pgdata:
`
	case "mysql":
		return `services:
  mysql:
    image: mysql:8-debian
    restart: unless-stopped
    environment:
      MYSQL_ROOT_PASSWORD: password
      MYSQL_DATABASE: appdb
    ports:
      - "3306:3306"
    volumes:
      - mysqldata:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 5s
      timeout: 5s
      retries: 10

volumes:
  mysqldata:
`
	case "mongodb":
		return `services:
  mongodb:
    image: mongo:7
    restart: unless-stopped
    environment:
      MONGO_INITDB_DATABASE: appdb
    ports:
      - "27017:27017"
    volumes:
      - mongodata:/data/db
    healthcheck:
      test: ["CMD", "mongosh", "--eval", "db.adminCommand('ping')"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  mongodata:
`
	}
	return ""
}

// ════════════════════════════════════════════════════════════════════════════
// migrations/000001_init.sql
// ════════════════════════════════════════════════════════════════════════════

func genMigrationSQL(c Config) string {
	switch c.DB {
	case "postgres":
		return `-- +migrate Up
CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    name          TEXT        NOT NULL,
    email         TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);

-- +migrate Down
DROP TABLE IF EXISTS users;
`
	case "mysql":
		return `-- +migrate Up
CREATE TABLE IF NOT EXISTS users (
    id            BIGINT      NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at    DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +migrate Down
DROP TABLE IF EXISTS users;
`
	case "sqlite":
		return `-- +migrate Up
CREATE TABLE IF NOT EXISTS users (
    id            INTEGER  PRIMARY KEY AUTOINCREMENT,
    name          TEXT     NOT NULL,
    email         TEXT     NOT NULL UNIQUE,
    password_hash TEXT     NOT NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);

-- +migrate Down
DROP TABLE IF EXISTS users;
`
	}
	return ""
}
