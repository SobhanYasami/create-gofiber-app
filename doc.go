// Package main is the create-gofiber-app CLI.
//
// All code-generation logic lives in this package (flat layout — no subpackages).
// The entry point is main(); the scaffolding engine is in generator_core.go,
// config/helpers in config.go, and all 35 file templates in files.go.
//
// # Adding a new database
//
//  1. Add an entry to LayerOptions (config.go).
//  2. Add a genDB* function (files.go) for platform/db/db.go.
//  3. Add a genRepo* function (files.go) for the repository implementation.
//  4. Wire both into genPlatformDB and genRepository switch statements.
//  5. Add a docker-compose case in genDockerCompose.
//  6. Add a SQL migration case in genMigrationSQL if applicable.
//
// # Adding a new ORM / query layer
//
// Follow steps 2–4 above; also add the new option to LayerOptions for the
// relevant databases.
//
// # Example — calling Generate programmatically
//
//	cfg := Config{
//		AppName:    "myapp",
//		ModuleName: "github.com/you/myapp",
//		DB:         "postgres",
//		Layer:      "gorm",
//	}
//	for _, f := range Generate(cfg) {
//		fmt.Printf("%-50s  %d bytes\n", f.Path, len(f.Content))
//	}
package main
