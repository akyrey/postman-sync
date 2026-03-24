package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/akyrey/postman-sync/internal/config"
	"github.com/akyrey/postman-sync/internal/postman"
)

func newExportCmd() *cobra.Command {
	var outputDir string
	var collections string
	var environments string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export collections and environments from Postman to local files",
		Long: `Fetches selected collections and/or environments from the configured
Postman workspace and writes them as JSON files under the output directory:

  <output_dir>/collections/<name>.json
  <output_dir>/environments/<name>.json

Use "all" to export every entity of that type, or provide a comma-separated
list of names to export specific ones.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			if err := cfg.ValidateGlobal(); err != nil {
				return err
			}
			if cmd.Flags().Changed("output-dir") {
				cfg.Export.OutputDir = outputDir
			}
			if cmd.Flags().Changed("collections") {
				cfg.Export.Collections = splitCSV(collections)
			}
			if cmd.Flags().Changed("environments") {
				cfg.Export.Environments = splitCSV(environments)
			}
			if err := cfg.ValidateExport(); err != nil {
				return err
			}

			start := time.Now()
			if err := runExport(cfg); err != nil {
				return err
			}
			fmt.Printf("Completed in %.1fs\n", time.Since(start).Seconds())
			return nil
		},
	}

	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Output directory (overrides config export.output_dir)")
	cmd.Flags().StringVar(&collections, "collections", "", `Collections to export: comma-separated names or "all" (overrides config)`)
	cmd.Flags().StringVar(&environments, "environments", "", `Environments to export: comma-separated names or "all" (overrides config)`)
	return cmd
}

func runExport(cfg *config.Config) error {
	client := postman.NewClient(cfg.PostmanAPIKey, cfg.WorkspaceID)
	e := cfg.Export
	exported := 0

	if len(e.Collections) > 0 {
		dir := filepath.Join(e.OutputDir, "collections")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating collections directory: %w", err)
		}

		available, err := client.GetWorkspaceCollections()
		if err != nil {
			return err
		}

		toExport := resolveEntities(e.Collections, available)
		for name, id := range toExport {
			fmt.Printf("Exporting collection %q...\n", name)
			wrapper, err := client.GetCollection(id)
			if err != nil {
				return err
			}
			path := filepath.Join(dir, sanitizeFilename(name)+".json")
			if err := writeJSON(path, wrapper, e.Pretty); err != nil {
				return fmt.Errorf("writing collection %q: %w", name, err)
			}
			exported++
		}
	}

	if len(e.Environments) > 0 {
		dir := filepath.Join(e.OutputDir, "environments")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating environments directory: %w", err)
		}

		available, err := client.GetWorkspaceEnvironments()
		if err != nil {
			return err
		}

		toExport := resolveEntities(e.Environments, available)
		for name, id := range toExport {
			fmt.Printf("Exporting environment %q...\n", name)
			wrapper, err := client.GetEnvironment(id)
			if err != nil {
				return err
			}
			path := filepath.Join(e.OutputDir, "environments", sanitizeFilename(name)+".json")
			if err := writeJSON(path, wrapper, e.Pretty); err != nil {
				return fmt.Errorf("writing environment %q: %w", name, err)
			}
			exported++
		}
	}

	fmt.Printf("Exported %d entities to %q.\n", exported, e.OutputDir)
	return nil
}
