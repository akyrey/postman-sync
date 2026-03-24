package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// sanitizeFilename replaces filesystem-unsafe characters with underscores.
func sanitizeFilename(name string) string {
	const unsafe = `/\:*?"<>|`
	for _, c := range unsafe {
		name = strings.ReplaceAll(name, string(c), "_")
	}
	return name
}

// writeJSON marshals data to path, creating parent directories as needed.
func writeJSON(path string, data any, pretty bool) error {
	var b []byte
	var err error
	if pretty {
		b, err = json.MarshalIndent(data, "", "  ")
	} else {
		b, err = json.Marshal(data)
	}
	if err != nil {
		return fmt.Errorf("marshalling JSON: %w", err)
	}
	return os.WriteFile(path, b, 0o644)
}

// readJSON reads a JSON file into target.
func readJSON(path string, target any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}
	return json.Unmarshal(b, target)
}

// listEntityFiles returns paths of all .json files directly inside dir.
func listEntityFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("directory %q does not exist", dir)
		}
		return nil, fmt.Errorf("reading directory %q: %w", dir, err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	return files, nil
}

// shouldImport reports whether name should be imported given the selection.
// It returns true if selection contains "all" or the exact name.
func shouldImport(name string, selection []string) bool {
	for _, s := range selection {
		if s == "all" || s == name {
			return true
		}
	}
	return false
}
