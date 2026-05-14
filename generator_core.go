package main

// File is a generated file with its repo-relative path and content.
type File struct {
	Path    string
	Content string
}

// Generate returns every file that should be written to disk for the given config.
// Callers iterate the slice and write Path/Content pairs to the output directory.
func Generate(cfg Config) []File {
	files := []File{
		{Path: "go.mod",                            Content: genGoMod(cfg)},
		{Path: ".env.example",                      Content: genEnvExample(cfg)},
		{Path: ".gitignore",                        Content: genGitIgnore()},
		{Path: "Makefile",                          Content: genMakefile(cfg)},
		{Path: "README.md",                         Content: genReadme(cfg)},
		{Path: "cmd/api/main.go",                   Content: genCmdMain(cfg)},
		{Path: "internal/config/config.go",         Content: genConfig(cfg)},
		{Path: "internal/platform/db/db.go",        Content: genPlatformDB(cfg)},
		{Path: "internal/domain/user/user.go",      Content: genDomainUser(cfg)},
		{Path: "internal/domain/user/repository.go",Content: genDomainRepository(cfg)},
		{Path: "internal/repository/user_repo.go",  Content: genRepository(cfg)},
		{Path: "internal/handler/user_handler.go",  Content: genHandler(cfg)},
		{Path: "internal/server/server.go",         Content: genServer(cfg)},
		{Path: "internal/middleware/auth.go",        Content: genMiddlewareAuth(cfg)},
	}

	if !cfg.IsSQLite() {
		files = append(files, File{
			Path:    "docker-compose.yml",
			Content: genDockerCompose(cfg),
		})
	}

	if cfg.IsSQL() {
		files = append(files, File{
			Path:    "migrations/000001_init.sql",
			Content: genMigrationSQL(cfg),
		})
	}

	return files
}
