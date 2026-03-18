package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/akyrey/postman-sync/internal/config"
	"github.com/akyrey/postman-sync/internal/openapi"
	"github.com/akyrey/postman-sync/internal/postman"
)

// version is set at build time via -ldflags "-X main.version=x.y.z".
var version = "dev"

func main() {
	configPath := flag.String("config", "postman-sync.yaml", "Path to the postman-sync YAML configuration file")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("postman-sync %s\n", version)
		os.Exit(0)
	}

	start := time.Now()

	if err := run(*configPath); err != nil {
		log.Fatalf("error: %v", err)
	}

	fmt.Printf("Completed in %.1fs\n", time.Since(start).Seconds())
}

func run(configPath string) error {
	// ── 1. Load configuration ──────────────────────────────────────────────
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// ── 2. Load & optionally sanitise the OpenAPI spec ─────────────────────
	fmt.Printf("Loading OpenAPI spec from %q...\n", cfg.OpenAPIPath)
	spec, err := openapi.Load(cfg.OpenAPIPath)
	if err != nil {
		return err
	}
	if cfg.SanitizeEnums {
		openapi.SanitizeEnums(spec)
	}

	// ── 3. Import spec into Postman (creates a temporary collection) ───────
	client := postman.NewClient(cfg.PostmanAPIKey, cfg.WorkspaceID)

	fmt.Println("Importing spec into Postman...")
	tmpID, err := client.ImportOpenAPI(spec)
	if err != nil {
		return err
	}

	// ── 4. Download the generated collection ──────────────────────────────
	fmt.Printf("Fetching generated collection %q...\n", tmpID)
	generated, err := client.GetCollection(tmpID)
	if err != nil {
		return err
	}

	// ── 5. Delete the temporary collection ────────────────────────────────
	fmt.Printf("Deleting temporary collection %q...\n", tmpID)
	if err := client.DeleteCollection(tmpID); err != nil {
		// Non-fatal: warn but continue.
		fmt.Fprintf(os.Stderr, "warning: could not delete temporary collection %q: %v\n", tmpID, err)
	}

	// ── 6. Transform the generated collection ─────────────────────────────
	col := &generated.Collection

	col.Items = postman.FlattenSingleFolders(col.Items)
	col.Items = postman.SortItemsAlpha(col.Items)

	if len(cfg.CommonHeaders) > 0 {
		postman.ApplyCommonHeaders(col.Items, cfg.CommonHeaders)
	}

	if err := postman.ApplyAuth(col, cfg.Auth); err != nil {
		return err
	}

	postman.ApplyScripts(col, cfg.Scripts)

	if len(cfg.FolderOverrides) > 0 {
		if err := postman.ApplyFolderOverrides(col.Items, cfg.FolderOverrides); err != nil {
			return err
		}
	}

	if cfg.Auth != nil && cfg.Auth.Propagation == "inherit" {
		postman.PropagateAuthInherit(col.Items, cfg.FolderOverrides)
	}

	if cfg.BaseURL != "" {
		postman.SetBaseURL(col.Items, cfg.BaseURL)
	}

	if cfg.DocLinks != nil && cfg.DocLinks.BaseURL != "" {
		postman.AddDocLinks(col.Items, cfg.DocLinks.BaseURL)
	}

	// ── 7. Merge into existing collection (or create new) ─────────────────
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
