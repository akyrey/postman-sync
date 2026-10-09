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
	var openapiRootFile string
	var noPreserve bool
	var dryRun bool
	var noBackup bool

	cmd := &cobra.Command{
		Use:   "openapi-sync",
		Short: "Import an OpenAPI spec into Postman and sync the collection",
		Long: `Loads an OpenAPI spec, imports it into Postman as a temporary collection,
applies configured transforms (auth, headers, scripts, folder overrides, base URL,
doc links), merges the result into the existing collection (or creates a new one),
then deletes the temporary collection.

The spec path may point to a single file (JSON or YAML) or a directory containing
a multi-file spec. When a directory is given, external $ref references are resolved
and bundled into a single document before import.`,
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
			if cmd.Flags().Changed("openapi-root-file") {
				cfg.OpenAPI.RootFile = openapiRootFile
			}
			if noPreserve {
				cfg.OpenAPI.PreserveRequestData = false
			}
			if noBackup {
				cfg.OpenAPI.Backup.Enabled = false
			}
			if err := cfg.ValidateOpenAPISync(); err != nil {
				return err
			}

			start := time.Now()
			if err := runOpenAPISync(cfg, dryRun); err != nil {
				return err
			}
			fmt.Printf("Completed in %.1fs\n", time.Since(start).Seconds())
			return nil
		},
	}

	cmd.Flags().StringVar(&openapiPath, "openapi-path", "", "Path to the OpenAPI spec file or directory (overrides config openapi.path)")
	cmd.Flags().StringVar(&openapiRootFile, "openapi-root-file", "", "Root spec filename when --openapi-path is a directory (overrides config openapi.root_file)")
	cmd.Flags().BoolVar(&noPreserve, "no-preserve", false, "Regenerate headers, params and bodies of existing requests from the spec, discarding edits made in Postman (overrides config openapi.preserve_request_data)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would change in the Postman collection without updating it (the spec is still converted through a temporary collection, which is deleted)")
	cmd.Flags().BoolVar(&noBackup, "no-backup", false, "Do not back up the existing collection before updating it (overrides config openapi.backup.enabled)")
	return cmd
}

func runOpenAPISync(cfg *config.Config, dryRun bool) error {
	o := cfg.OpenAPI
	client := postman.NewClient(cfg.PostmanAPIKey, cfg.WorkspaceID)

	generated, err := generateCollection(client, o)
	if err != nil {
		return err
	}
	col := &generated.Collection

	// Snapshot what the spec generated before merging: it becomes the next
	// sync's baseline once Postman has been updated.
	snapshot, err := snapshotBaseline(col)
	if err != nil {
		return err
	}

	colName := col.Info.Name
	fmt.Printf("Looking up existing collections in workspace...\n")
	existing, err := client.GetWorkspaceCollections()
	if err != nil {
		return err
	}

	if existingID, found := existing[colName]; found {
		err = updateCollection(client, o, existingID, col, dryRun)
	} else {
		err = createCollection(client, generated, dryRun)
	}
	if err != nil || dryRun {
		return err
	}

	if err := saveBaseline(o.StateDir, colName, snapshot); err != nil {
		// Non-fatal: the next sync falls back to a two-way merge.
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}
	return nil
}

// generateCollection converts the OpenAPI spec into a Postman collection
// (through a temporary collection, deleted afterwards) and applies the
// configured transforms.
func generateCollection(client *postman.Client, o *config.OpenAPIConfig) (*postman.CollectionWrapper, error) {
	// ── 1. Load & optionally sanitise the OpenAPI spec ─────────────────────
	fmt.Printf("Loading OpenAPI spec from %q...\n", o.Path)
	spec, err := openapi.LoadAndBundle(o.Path, o.RootFile)
	if err != nil {
		return nil, err
	}
	if o.SanitizeEnums {
		openapi.SanitizeEnums(spec)
	}

	// ── 2. Import spec into Postman (creates a temporary collection) ───────
	fmt.Println("Importing spec into Postman...")
	tmpID, err := client.ImportOpenAPI(spec, o.EffectiveImportOptions())
	if err != nil {
		return nil, err
	}

	// ── 3. Download the generated collection ──────────────────────────────
	fmt.Printf("Fetching generated collection %q...\n", tmpID)
	generated, err := client.GetCollection(tmpID)

	// ── 4. Delete the temporary collection ────────────────────────────────
	fmt.Printf("Deleting temporary collection %q...\n", tmpID)
	if delErr := client.DeleteCollection(tmpID); delErr != nil {
		// Non-fatal: warn but continue.
		fmt.Fprintf(os.Stderr, "warning: could not delete temporary collection %q: %v\n", tmpID, delErr)
	}
	if err != nil {
		return nil, err
	}

	// ── 5. Transform the generated collection ─────────────────────────────
	if err := transformCollection(&generated.Collection, o); err != nil {
		return nil, err
	}
	return generated, nil
}

func transformCollection(col *postman.Collection, o *config.OpenAPIConfig) error {
	col.Items = postman.FlattenSingleFolders(col.Items)
	col.Items = postman.SortItemsAlpha(col.Items)

	applyCommonParams(col.Items, o)

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

	postman.SyncRawQuery(col.Items)
	return nil
}

// updateCollection merges the generated collection into the existing one and
// uploads the result, or only prints the changes when dryRun is set.
func updateCollection(client *postman.Client, o *config.OpenAPIConfig, id string, col *postman.Collection, dryRun bool) error {
	fmt.Printf("Merging into existing collection %q (%s)...\n", col.Info.Name, id)
	current, err := client.GetCollection(id)
	if err != nil {
		return err
	}

	baseline, err := loadBaseline(o.StateDir, col.Info.Name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v; merging without baseline\n", err)
	}
	if baseline == nil && o.PreserveRequestData {
		fmt.Println("No baseline from a previous sync: using two-way merge.")
	}

	updated := mergeCollection(&current.Collection, col, o, baseline)
	if dryRun {
		fmt.Printf("Dry run: changes that would be applied to %q (nothing was written):\n", col.Info.Name)
		printChanges(os.Stdout, postman.DiffCollections(&current.Collection, &updated.Collection))
		return nil
	}

	if o.Backup.Enabled {
		path, err := backupCollection(o.StateDir, current, time.Now(), o.Backup.Keep)
		if err != nil {
			return fmt.Errorf("backing up collection before update (use --no-backup to skip): %w", err)
		}
		fmt.Printf("Backed up current collection to %q.\n", path)
	}

	if err := client.UpdateCollection(id, updated); err != nil {
		return err
	}
	fmt.Printf("Updated collection %q.\n", col.Info.Name)
	return nil
}

// mergeCollection returns the collection to upload: current with the
// generated items merged in and config-driven collection settings applied.
// current is not modified.
func mergeCollection(current, generated *postman.Collection, o *config.OpenAPIConfig, baseline []postman.CollectionItem) *postman.CollectionWrapper {
	out := *current

	// Merge items: new spec drives structure, old drives auth/scripts
	// (and header/param/body values when preserving request data).
	out.Items = postman.MergeItemsWith(current.Items, generated.Items,
		postman.MergeOptions{PreserveRequestData: o.PreserveRequestData, Baseline: baseline})
	// Configured headers/params always win over values kept from Postman.
	applyCommonParams(out.Items, o)
	// Merging may have changed which query params are enabled.
	postman.SyncRawQuery(out.Items)

	// Apply collection-level auth from config (if set), otherwise keep old.
	if generated.Auth != nil {
		out.Auth = generated.Auth
	}

	// Apply collection-level events from config (if set), otherwise keep old.
	if len(generated.Events) > 0 {
		out.Events = generated.Events
	}
	return &postman.CollectionWrapper{Collection: out}
}

// createCollection uploads the generated collection as a new collection, or
// only reports it when dryRun is set.
func createCollection(client *postman.Client, generated *postman.CollectionWrapper, dryRun bool) error {
	name := generated.Collection.Info.Name
	if dryRun {
		fmt.Printf("Dry run: would create collection %q with %d request(s) (nothing was written).\n",
			name, postman.CountRequests(generated.Collection.Items))
		return nil
	}

	fmt.Printf("Creating new collection %q...\n", name)
	// Strip Postman-generated IDs for portability.
	stripIDs(generated.Collection.Items)

	if err := client.CreateCollection(generated); err != nil {
		return err
	}
	fmt.Printf("Created collection %q.\n", name)
	return nil
}

// applyCommonParams injects the configured common headers and query params.
func applyCommonParams(items []postman.CollectionItem, o *config.OpenAPIConfig) {
	if len(o.CommonHeaders) > 0 {
		postman.ApplyCommonHeaders(items, o.CommonHeaders)
	}
	if len(o.CommonQueryParams) > 0 {
		postman.ApplyCommonQueryParams(items, o.CommonQueryParams)
	}
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
