package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// ── styles ────────────────────────────────────────────────────────────────────

var (
	sBrand   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	sSuccess = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("82"))
	sDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	sCode    = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Background(lipgloss.Color("235")).Padding(0, 1)
	sErr     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	sAdd     = lipgloss.NewStyle().Foreground(lipgloss.Color("71"))
	sBadge   = lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Background(lipgloss.Color("235")).Padding(0, 1)
)

func main() {
	fmt.Println()
	fmt.Println(sBrand.Render("  create-gofiber-app") + "  " + sDim.Render("GoFiber v2 · go 1.24 · production scaffold"))
	fmt.Println(sDim.Render("  ─────────────────────────────────────────────────"))
	fmt.Println()

	cfg := Config{}

	// Optional: project name as positional arg
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		cfg.AppName = strings.TrimSpace(os.Args[1])
	}

	// ── Phase 1: project name ─────────────────────────────────────────────
	if cfg.AppName == "" {
		must(huh.NewForm(huh.NewGroup(
			huh.NewInput().
				Title("Project name").
				Placeholder("my-gofiber-app").
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("project name is required")
					}
					if strings.ContainsAny(s, " /\\") {
						return fmt.Errorf("no spaces or slashes")
					}
					return nil
				}).
				Value(&cfg.AppName),
		)).Run())
	}

	// ── Phase 2: module path + database ──────────────────────────────────
	must(huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Go module path").
				Placeholder("github.com/you/"+cfg.AppName).
				Value(&cfg.ModuleName).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("module path is required")
					}
					return nil
				}),

			huh.NewSelect[string]().
				Title("Database").
				Options(
					huh.NewOption("PostgreSQL", "postgres"),
					huh.NewOption("MySQL", "mysql"),
					huh.NewOption("MongoDB", "mongodb"),
					huh.NewOption("SQLite", "sqlite"),
				).
				Value(&cfg.DB),
		),
	).Run())

	if strings.TrimSpace(cfg.ModuleName) == "" {
		cfg.ModuleName = "github.com/you/" + cfg.AppName
	}

	// ── Phase 3: data layer (options depend on DB chosen above) ──────────
	must(huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("Data layer").
			Description("Query strategy / ORM").
			Options(LayerOptions(cfg.DB)...).
			Value(&cfg.Layer),
	)).Run())

	// ── Summary ───────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println(sDim.Render("  ─────────────────────────────────────────────────"))
	fmt.Printf("  %s  %s\n", sDim.Render("project"), cfg.AppName)
	fmt.Printf("  %s  %s\n", sDim.Render("module "), cfg.ModuleName)
	fmt.Printf("  %s  %s\n", sDim.Render("db     "), sBadge.Render(cfg.DBLabel()))
	fmt.Printf("  %s  %s\n", sDim.Render("layer  "), sBadge.Render(cfg.LayerLabel()))
	fmt.Println(sDim.Render("  ─────────────────────────────────────────────────"))
	fmt.Println()

	// ── Generate ──────────────────────────────────────────────────────────
	outDir := filepath.Join(".", cfg.AppName)
	files := Generate(cfg)

	for _, f := range files {
		dest := filepath.Join(outDir, f.Path)
		if err := writeFile(dest, f.Content); err != nil {
			fmt.Fprintf(os.Stderr, "%s %s: %v\n", sErr.Render("  ✗"), f.Path, err)
			os.Exit(1)
		}
		fmt.Printf("  %s  %s\n", sAdd.Render("CREATE"), sDim.Render(f.Path))
	}

	// ── Done ──────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Printf("%s  %d files in ./%s\n\n",
		sSuccess.Render("  ✓ Done!"), len(files), cfg.AppName)

	fmt.Println(sDim.Render("  Next steps:"))
	printStep("cd " + cfg.AppName)
	if cfg.DB != "sqlite" {
		printStep("docker compose up -d")
	}
	printStep("go mod tidy")
	printStep("cp .env.example .env")
	printStep("go run cmd/api/main.go")
	fmt.Println()
}

func printStep(cmd string) {
	fmt.Printf("    %s\n", sCode.Render(cmd))
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// must exits cleanly on Ctrl-C and fatally on any real error.
func must(err error) {
	if err == nil {
		return
	}
	if err.Error() == "user aborted" {
		fmt.Println(sDim.Render("\n  Aborted."))
		os.Exit(0)
	}
	fmt.Fprintf(os.Stderr, "%s %v\n", sErr.Render("  error:"), err)
	os.Exit(1)
}
