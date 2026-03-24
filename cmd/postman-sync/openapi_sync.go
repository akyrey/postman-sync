package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/akyrey/postman-sync/internal/config"
	"github.com/akyrey/postman-sync/internal/openapi"
	"github.com/akyrey/postman-sync/internal/postman"
)

func newOpenAPISyncCmd() *cobra.Command {
	var openapiPath string

	cmd := &cobra.Command{
		Use:   "openapi-sync",
		Short: "Import an OpenAPI spec into Postman and sync the collection",
		Long: `Loads an OpenAPI spec, imports it into Postman as a temporary collection,
applies configured transforms (auth, headers, scripts, folder overrides, base URL,
doc links), merges the result into the existing collection (or creates a new one),
then deletes the temporary collection.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			if err := cfg.ValidateGlobal(); err != nil {
				return err
			}
			if cmd.Flags().Changed("openapi-path") {
				cfg.OpenAPI.Path = openapiPath
			}
			if err := cfg.ValidateOpenAPISync(); err != nil {
				return err
			}

			start := time.Now()
			if err := runOpenAPISync(cfg); err != nil {
				return err
			}
			fmt.Printf("Completed in %.1fs\n", time.Since(start).Seconds())
			return nil
		},
	}

	cmd.Flags().StringVar(&openapiPath, "openapi-path", "", "Path to the OpenAPI spec (overrides config openapi.path)")
	return cmd
}

func runOpenAPISync(cfg *config.Config) error {
	o := cfg.OpenAPI

	// ── 1. Load & optionally sanitise the OpenAPI spec ─────────────────────
	fmt.Printf("Loading OpenAPI spec from %q...\n", o.Path)
	spec, err := openapi.Load(o.Path)
	if err != nil {
		return err
	}
	if o.SanitizeEnums {
		openapi.SanitizeEnums(spec)
	}

	// ── 2. Import spec into Postman (creates a temporary collection) ───────
	client := postman.NewClient(cfg.PostmanAPIKey, cfg.WorkspaceID)

	fmt.Println("Importing spec into Postman...")
	tmpID, err := client.ImportOpenAPI(spec)
	if err != nil {
		return err
	}

	// ── 3. Download the generated collection ──────────────────────────────
	fmt.Printf("Fetching generated collection %q...\n", tmpID)
	generated, err := client.GetCollection(tmpID)
	if err != nil {
		return err
	}

	// ── 4. Delete the temporary collection ────────────────────────────────
	fmt.Printf("Deleting temporary collection %q...\n", tmpID)
	if err := client.DeleteCollection(tmpID); err != nil {
		// Non-fatal: warn but continue.
		fmt.Fprintf(os.Stderr, "warning: could not delete temporary collection %q: %v\n", tmpID, err)
	}

	// ── 5. Transform the generated collection ─────────────────────────────
	col := &generated.Collection

	col.Items = postman.FlattenSingleFolders(col.Items)
	col.Items = postman.SortItemsAlpha(col.Items)

	if len(o.CommonHeaders) > 0 {
		postman.ApplyCommonHeaders(col.Items, o.CommonHeaders)
	}

	if err := postman.ApplyAuth(col, o.Auth); err != nil {
		return err
	}

	postman.ApplyScripts(col, o.Scripts)

	if len(o.FolderOverrides) > 0 {
		if err := postman.ApplyFolderOverrides(col.Items, o.FolderOverrides); err != nil {
			return err
		}
	}

	if o.Auth != nil && o.Auth.Propagation == "inherit" {
		postman.PropagateAuthInherit(col.Items, o.FolderOverrides)
	}

	if o.BaseURL != "" {
		postman.SetBaseURL(col.Items, o.BaseURL)
	}

	if o.DocLinks != nil && o.DocLinks.BaseURL != "" {
		postman.AddDocLinks(col.Items, o.DocLinks.BaseURL)
	}

	// ── 6. Merge into existing collection (or create new) ─────────────────
	colName := col.Info.Name
	fmt.Printf("Looking up existing collections in workspace...\n")
	existing, err := client.GetWorkspaceCollections()
	if err != nil {
		return err
	}

	if existingID, found := existing[colName]; found {
		fmt.Printf("Merging into existing collection %q (%s)...\n", colName, existingID)
		oldWrapper, err := client.GetCollection(existingID)
		if err != nil {
			return err
		}

		// Merge items: new spec drives structure, old drives auth/scripts.
		mergedItems := postman.MergeItems(oldWrapper.Collection.Items, col.Items)
		oldWrapper.Collection.Items = mergedItems

		// Apply collection-level auth from config (if set), otherwise keep old.
		if col.Auth != nil {
			oldWrapper.Collection.Auth = col.Auth
		}

		// Apply collection-level events from config (if set), otherwise keep old.
		if len(col.Events) > 0 {
			oldWrapper.Collection.Events = col.Events
		}

		if err := client.UpdateCollection(existingID, oldWrapper); err != nil {
			return err
		}
		fmt.Printf("Updated collection %q.\n", colName)
	} else {
		fmt.Printf("Creating new collection %q...\n", colName)
		// Strip Postman-generated IDs for portability.
		stripIDs(col.Items)

		if err := client.CreateCollection(generated); err != nil {
			return err
		}
		fmt.Printf("Created collection %q.\n", colName)
	}

	return nil
}

// stripIDs removes Postman-generated id fields from all items.
func stripIDs(items []postman.CollectionItem) {
	for i := range items {
		items[i].ID = ""
		if items[i].IsFolder() {
			stripIDs(*items[i].Items)
		}
	}
}
