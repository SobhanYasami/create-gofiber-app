package main

import "github.com/charmbracelet/huh"

// Config holds user selections. Passed to every template function.
type Config struct {
	AppName    string
	ModuleName string
	DB         string // postgres | mysql | mongodb | sqlite
	Layer      string // gorm | sqlx | raw | qmgo
}

// ── Boolean helpers ───────────────────────────────────────────────────────────

func (c Config) IsGORM() bool     { return c.Layer == "gorm" }
func (c Config) IsSQLx() bool     { return c.Layer == "sqlx" }
func (c Config) IsRaw() bool      { return c.Layer == "raw" }
func (c Config) IsQMGO() bool     { return c.Layer == "qmgo" }
func (c Config) IsPostgres() bool { return c.DB == "postgres" }
func (c Config) IsMySQL() bool    { return c.DB == "mysql" }
func (c Config) IsMongoDB() bool  { return c.DB == "mongodb" }
func (c Config) IsSQLite() bool   { return c.DB == "sqlite" }
func (c Config) IsSQL() bool      { return c.DB != "mongodb" }
func (c Config) IsNoSQL() bool    { return c.DB == "mongodb" }

// ── Labels ────────────────────────────────────────────────────────────────────

func (c Config) DBLabel() string {
	return map[string]string{
		"postgres": "PostgreSQL",
		"mysql":    "MySQL",
		"mongodb":  "MongoDB",
		"sqlite":   "SQLite",
	}[c.DB]
}

func (c Config) LayerLabel() string {
	return map[string]string{
		"gorm": "GORM ORM",
		"sqlx": "sqlx",
		"raw":  "raw driver",
		"qmgo": "qmgo ODM",
	}[c.Layer]
}

// ── DB connection type (for generated code) ───────────────────────────────────

// DBConnType returns the Go type of the DB handle for this combination.
// Used to generate correct function signatures across files.
func (c Config) DBConnType() string {
	switch {
	case c.IsGORM():
		return "*gorm.DB"
	case c.IsSQLx():
		return "*sqlx.DB"
	case c.IsPostgres() && c.IsRaw():
		return "*pgxpool.Pool"
	case c.IsMongoDB() && c.IsRaw():
		return "*mongo.Client"
	case c.IsMongoDB() && c.IsQMGO():
		return "*qmgo.QmgoClient"
	default: // mysql raw, sqlite raw
		return "*sql.DB"
	}
}

// DBConnImport returns the import path for the connection type package.
func (c Config) DBConnImport() string {
	switch {
	case c.IsGORM():
		return "gorm.io/gorm"
	case c.IsSQLx():
		return "github.com/jmoiron/sqlx"
	case c.IsPostgres() && c.IsRaw():
		return "github.com/jackc/pgx/v5/pgxpool"
	case c.IsMongoDB() && c.IsRaw():
		return "go.mongodb.org/mongo-driver/mongo"
	case c.IsMongoDB() && c.IsQMGO():
		return "github.com/qiniu/qmgo"
	default:
		return "database/sql"
	}
}

// ── Prompt helpers ────────────────────────────────────────────────────────────

// LayerOptions returns huh select options for the given DB.
func LayerOptions(db string) []huh.Option[string] {
	switch db {
	case "postgres":
		return []huh.Option[string]{
			huh.NewOption("GORM  (ORM, auto-migrate)", "gorm"),
			huh.NewOption("sqlx  (query builder, named params)", "sqlx"),
			huh.NewOption("pgx   (raw driver, no ORM)", "raw"),
		}
	case "mysql":
		return []huh.Option[string]{
			huh.NewOption("GORM  (ORM, auto-migrate)", "gorm"),
			huh.NewOption("sqlx  (query builder, named params)", "sqlx"),
			huh.NewOption("database/sql  (raw)", "raw"),
		}
	case "mongodb":
		return []huh.Option[string]{
			huh.NewOption("mongo-driver  (official, raw)", "raw"),
			huh.NewOption("qmgo          (ODM wrapper)", "qmgo"),
		}
	case "sqlite":
		return []huh.Option[string]{
			huh.NewOption("GORM  (ORM, auto-migrate)", "gorm"),
			huh.NewOption("database/sql  (raw, modernc)", "raw"),
		}
	default:
		return []huh.Option[string]{huh.NewOption("GORM", "gorm")}
	}
}
