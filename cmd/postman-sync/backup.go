package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/akyrey/postman-sync/internal/postman"
)

const (
	// backupsSubdir holds one directory per backup run, named by its UTC
	// timestamp and laid out like an export (collections/<name>.json), so a
	// backup can be restored with: import --input-dir <run dir>.
	backupsSubdir = "backups"

	// backupTimeLayout sorts lexicographically in chronological order.
	backupTimeLayout = "20060102T150405Z"
)

// backupCollection writes the collection to
// <stateDir>/backups/<timestamp>/collections/<name>.json and then removes the
// oldest backups of the same collection beyond keep (0 keeps all). It returns
// the path of the new backup.
func backupCollection(stateDir string, wrapper *postman.CollectionWrapper, now time.Time, keep int) (string, error) {
	name := wrapper.Collection.Info.Name
	runDir := filepath.Join(stateDir, backupsSubdir, now.UTC().Format(backupTimeLayout))
	path := filepath.Join(runDir, "collections", sanitizeFilename(name)+".json")

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("creating backup directory: %w", err)
	}
	if err := writeJSON(path, wrapper, true); err != nil {
		return "", fmt.Errorf("writing backup of %q: %w", name, err)
	}

	if keep > 0 {
		if err := pruneBackups(stateDir, name, keep); err != nil {
			return "", err
		}
	}
	return path, nil
}

// pruneBackups deletes the oldest backups of one collection so that at most
// keep remain. Only that collection's backup files are deleted, plus the
// directories they leave empty.
func pruneBackups(stateDir, collectionName string, keep int) error {
	pattern := filepath.Join(stateDir, backupsSubdir, "*", "collections", sanitizeFilename(collectionName)+".json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("listing backups: %w", err)
	}
	if len(files) <= keep {
		return nil
	}

	// Backup directory names are timestamps, so lexical order is chronological.
	sort.Strings(files)
	for _, f := range files[:len(files)-keep] {
		if err := os.Remove(f); err != nil {
			return fmt.Errorf("removing old backup %q: %w", f, err)
		}
		collectionsDir := filepath.Dir(f)
		if err := removeIfEmpty(collectionsDir); err != nil {
			return err
		}
		if err := removeIfEmpty(filepath.Dir(collectionsDir)); err != nil {
			return err
		}
	}
	return nil
}

// removeIfEmpty removes dir when it has no entries left.
func removeIfEmpty(dir string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading %q: %w", dir, err)
	}
	if len(entries) > 0 {
		return nil
	}
	if err := os.Remove(dir); err != nil {
		return fmt.Errorf("removing empty directory %q: %w", dir, err)
	}
	return nil
}
