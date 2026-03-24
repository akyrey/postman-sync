package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/akyrey/postman-sync/internal/config"
	"github.com/akyrey/postman-sync/internal/postman"
)

func newImportCmd() *cobra.Command {
	var inputDir string
	var collections string
	var environments string
	var strategy string

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import collections and environments from local files into Postman",
		Long: `Reads JSON files from the input directory and pushes them to Postman.

  <input_dir>/collections/<name>.json   → collections
  <input_dir>/environments/<name>.json  → environments

Use "all" to import every file found, or provide a comma-separated list of
names (without the .json extension) to import specific ones.

Import strategies (collections only):
  overwrite  Replace the entire collection (default)
  merge      Preserve auth, scripts, and saved responses from the existing
             collection; update request definitions from the file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			if err := cfg.ValidateGlobal(); err != nil {
				return err
			}
			if cmd.Flags().Changed("input-dir") {
				cfg.Import.InputDir = inputDir
			}
			if cmd.Flags().Changed("collections") {
				if cfg.Import.Collections == nil {
					cfg.Import.Collections = &config.ImportEntityConfig{}
				}
				cfg.Import.Collections.Names = splitCSV(collections)
			}
			if cmd.Flags().Changed("environments") {
				if cfg.Import.Environments == nil {
					cfg.Import.Environments = &config.ImportEntityConfig{}
				}
				cfg.Import.Environments.Names = splitCSV(environments)
			}
			if cmd.Flags().Changed("strategy") {
				if cfg.Import.Collections == nil {
					cfg.Import.Collections = &config.ImportEntityConfig{}
				}
				cfg.Import.Collections.Strategy = strategy
			}
			if err := cfg.ValidateImport(); err != nil {
				return err
			}

			start := time.Now()
			if err := runImport(cfg); err != nil {
				return err
			}
			fmt.Printf("Completed in %.1fs\n", time.Since(start).Seconds())
			return nil
		},
	}

	cmd.Flags().StringVar(&inputDir, "input-dir", "", "Input directory (overrides config import.input_dir)")
	cmd.Flags().StringVar(&collections, "collections", "", `Collections to import: comma-separated names or "all" (overrides config)`)
	cmd.Flags().StringVar(&environments, "environments", "", `Environments to import: comma-separated names or "all" (overrides config)`)
	cmd.Flags().StringVar(&strategy, "strategy", "", `Import strategy for collections: "overwrite" or "merge" (overrides config)`)
	return cmd
}

func runImport(cfg *config.Config) error {
	client := postman.NewClient(cfg.PostmanAPIKey, cfg.WorkspaceID)
	i := cfg.Import
	imported := 0

	if i.Collections != nil && len(i.Collections.Names) > 0 {
		dir := filepath.Join(i.InputDir, "collections")
		files, err := listEntityFiles(dir)
		if err != nil {
			return fmt.Errorf("listing collection files: %w", err)
		}

		existing, err := client.GetWorkspaceCollections()
		if err != nil {
			return err
		}

		strategy := i.Collections.Strategy
		if strategy == "" {
			strategy = "overwrite"
		}

		for _, filePath := range files {
			baseName := strings.TrimSuffix(filepath.Base(filePath), ".json")
			if !shouldImport(baseName, i.Collections.Names) {
				continue
			}

			var wrapper postman.CollectionWrapper
			if err := readJSON(filePath, &wrapper); err != nil {
				return fmt.Errorf("reading collection file %q: %w", filePath, err)
			}

			colName := wrapper.Collection.Info.Name
			if existingID, found := existing[colName]; found {
				if strategy == "merge" {
					old, err := client.GetCollection(existingID)
					if err != nil {
						return err
					}
					merged := postman.MergeItems(old.Collection.Items, wrapper.Collection.Items)
					old.Collection.Items = merged
					fmt.Printf("Merging collection %q (%s)...\n", colName, existingID)
					if err := client.UpdateCollection(existingID, old); err != nil {
						return err
					}
				} else {
					fmt.Printf("Overwriting collection %q (%s)...\n", colName, existingID)
					if err := client.UpdateCollection(existingID, &wrapper); err != nil {
						return err
					}
				}
			} else {
				fmt.Printf("Creating collection %q...\n", colName)
				stripIDs(wrapper.Collection.Items)
				if err := client.CreateCollection(&wrapper); err != nil {
					return err
				}
			}
			imported++
		}
	}

	if i.Environments != nil && len(i.Environments.Names) > 0 {
		dir := filepath.Join(i.InputDir, "environments")
		files, err := listEntityFiles(dir)
		if err != nil {
			return fmt.Errorf("listing environment files: %w", err)
		}

		existing, err := client.GetWorkspaceEnvironments()
		if err != nil {
			return err
		}

		for _, filePath := range files {
			baseName := strings.TrimSuffix(filepath.Base(filePath), ".json")
			if !shouldImport(baseName, i.Environments.Names) {
				continue
			}

			var wrapper postman.EnvironmentWrapper
			if err := readJSON(filePath, &wrapper); err != nil {
				return fmt.Errorf("reading environment file %q: %w", filePath, err)
			}

			envName := wrapper.Environment.Name
			if existingID, found := existing[envName]; found {
				fmt.Printf("Updating environment %q (%s)...\n", envName, existingID)
				if err := client.UpdateEnvironment(existingID, &wrapper); err != nil {
					return err
				}
			} else {
				fmt.Printf("Creating environment %q...\n", envName)
				wrapper.Environment.ID = ""
				if err := client.CreateEnvironment(&wrapper); err != nil {
					return err
				}
			}
			imported++
		}
	}

	fmt.Printf("Imported %d entities from %q.\n", imported, i.InputDir)
	return nil
}
