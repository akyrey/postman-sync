package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags "-X main.version=x.y.z".
var version = "dev"

// cfgFile is the path to the config file, shared across all subcommands.
var cfgFile string

func main() {
	rootCmd := &cobra.Command{
		Use:   "postman-sync",
		Short: "Sync Postman collections, environments, and OpenAPI specs",
		Long: `postman-sync manages Postman workspaces:

  openapi-sync  Import an OpenAPI spec and sync the resulting collection
  export        Export collections and environments to local files
  import        Import collections and environments from local files`,
		SilenceUsage: true,
	}

	rootCmd.Version = version
	rootCmd.SetVersionTemplate("postman-sync {{.Version}}\n")

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "postman-sync.yaml", "Path to the postman-sync YAML config file")

	rootCmd.AddCommand(newOpenAPISyncCmd())
	rootCmd.AddCommand(newExportCmd())
	rootCmd.AddCommand(newImportCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// splitCSV splits a comma-separated string into trimmed, non-empty parts.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// resolveEntities filters the available name→id map by the selection slice.
// If selection contains "all", every available entity is returned.
// Otherwise only entities whose names appear in selection are returned; unknown
// names produce a warning on stderr.
func resolveEntities(selection []string, available map[string]string) map[string]string {
	for _, s := range selection {
		if s == "all" {
			return available
		}
	}
	result := make(map[string]string, len(selection))
	for _, name := range selection {
		if id, ok := available[name]; ok {
			result[name] = id
		} else {
			fmt.Fprintf(os.Stderr, "warning: %q not found in workspace, skipping\n", name)
		}
	}
	return result
}
