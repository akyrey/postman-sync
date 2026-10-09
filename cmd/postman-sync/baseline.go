package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/akyrey/postman-sync/internal/postman"
)

// baselinesSubdir holds, per collection, the collection generated from the
// spec by the last successful openapi-sync. It is the common ancestor for the
// next sync's three-way merge.
const baselinesSubdir = "baselines"

func baselinePath(stateDir, collectionName string) string {
	return filepath.Join(stateDir, baselinesSubdir, sanitizeFilename(collectionName)+".json")
}

// loadBaseline returns the baseline items for a collection, or nil when no
// baseline has been saved yet.
func loadBaseline(stateDir, collectionName string) ([]postman.CollectionItem, error) {
	var wrapper postman.CollectionWrapper
	err := readJSON(baselinePath(stateDir, collectionName), &wrapper)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("loading baseline for %q: %w", collectionName, err)
	}
	return wrapper.Collection.Items, nil
}

// snapshotBaseline encodes the generated collection before it is merged, so
// that later in-place transforms on shared data cannot alter it.
func snapshotBaseline(col *postman.Collection) ([]byte, error) {
	b, err := json.Marshal(postman.CollectionWrapper{Collection: *col})
	if err != nil {
		return nil, fmt.Errorf("encoding baseline: %w", err)
	}
	return b, nil
}

// saveBaseline writes a snapshot taken with snapshotBaseline.
func saveBaseline(stateDir, collectionName string, snapshot []byte) error {
	path := baselinePath(stateDir, collectionName)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating baseline directory: %w", err)
	}
	if err := os.WriteFile(path, snapshot, 0o644); err != nil {
		return fmt.Errorf("writing baseline for %q: %w", collectionName, err)
	}
	return nil
}
