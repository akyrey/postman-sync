package postman

import (
	"fmt"
	"sort"
	"strings"

	"github.com/akyrey/postman-sync/config"
)

// FlattenSingleFolders unwraps any folder that contains exactly one request
// whose name matches the folder name, replacing the folder with the request.
func FlattenSingleFolders(items []CollectionItem) []CollectionItem {
	result := make([]CollectionItem, 0, len(items))
	for _, item := range items {
		item := item // capture
		if item.IsFolder() {
			children := FlattenSingleFolders(*item.Items)
			if len(children) == 1 && !children[0].IsFolder() && children[0].Name == item.Name {
				// Replace folder with its single child.
				result = append(result, children[0])
				continue
			}
			item.Items = &children
		}
		result = append(result, item)
	}
	return result
}

// SortItemsAlpha sorts items alphabetically by name at every level.
func SortItemsAlpha(items []CollectionItem) []CollectionItem {
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	for i, item := range items {
		if item.IsFolder() {
			sorted := SortItemsAlpha(*item.Items)
			items[i].Items = &sorted
		}
	}
	return items
}

// ApplyCommonHeaders injects the configured headers into every leaf request.
// If a header with the same key already exists, its value and disabled flag
// are updated. Otherwise the header is appended.
func ApplyCommonHeaders(items []CollectionItem, headers []config.Header) {
	for i := range items {
		if items[i].IsFolder() {
			ApplyCommonHeaders(*items[i].Items, headers)
			continue
		}
		if items[i].Request == nil {
			continue
		}
		for _, cfgHdr := range headers {
			applyHeader(&items[i].Request.Header, cfgHdr)
		}
	}
}

func applyHeader(headers *[]Header, h config.Header) {
	for i, existing := range *headers {
		if strings.EqualFold(existing.Key, h.Key) {
			(*headers)[i].Value = h.Value
			(*headers)[i].Disabled = h.Disabled
			return
		}
	}
	*headers = append(*headers, Header{
		Key:      h.Key,
		Value:    h.Value,
		Disabled: h.Disabled,
	})
}

// ApplyAuth sets the collection-level auth from config.
func ApplyAuth(col *Collection, authCfg *config.Auth) error {
	if authCfg == nil {
		return nil
	}
	attrs := make([]AuthAttribute, 0, len(authCfg.Attributes))
	for _, a := range authCfg.Attributes {
		attrs = append(attrs, AuthAttribute{Key: a.Key, Value: a.Value, Type: a.Type})
	}
	auth, err := BuildAuth(authCfg.Type, attrs)
	if err != nil {
		return fmt.Errorf("building collection auth: %w", err)
	}
	col.Auth = auth
	return nil
}

// ApplyScripts sets collection-level pre-request and test events from config.
func ApplyScripts(col *Collection, scriptsCfg *config.Scripts) {
	if scriptsCfg == nil {
		return
	}
	col.Events = buildEvents(scriptsCfg, col.Events)
}

// ApplyFolderOverrides applies per-folder auth/script overrides.
// Matching is case-sensitive on the folder name.
func ApplyFolderOverrides(items []CollectionItem, overrides map[string]config.FolderOverride) error {
	for i := range items {
		if !items[i].IsFolder() {
			continue
		}
		if override, ok := overrides[items[i].Name]; ok {
			if override.Auth != nil {
				attrs := make([]AuthAttribute, 0, len(override.Auth.Attributes))
				for _, a := range override.Auth.Attributes {
					attrs = append(attrs, AuthAttribute{Key: a.Key, Value: a.Value, Type: a.Type})
				}
				auth, err := BuildAuth(override.Auth.Type, attrs)
				if err != nil {
					return fmt.Errorf("building auth for folder %q: %w", items[i].Name, err)
				}
				items[i].Auth = auth
			}
			if override.Scripts != nil {
				items[i].Events = buildEvents(override.Scripts, items[i].Events)
			}
		}
		// Recurse into sub-folders.
		if err := ApplyFolderOverrides(*items[i].Items, overrides); err != nil {
			return err
		}
	}
	return nil
}

// PropagateAuthInherit clears the auth on every folder and leaf request so that
// they inherit authentication from their parent (collection or enclosing folder).
//
// Rules:
//   - Items whose auth type is "noauth" are left untouched – they explicitly
//     opt out of authentication and should not be changed.
//   - Items with nil auth are already inheriting; they are skipped.
//   - Folders whose name appears as a key in overrides are skipped (only the
//     folder itself; its children are still processed so they can inherit from
//     the overridden folder).
//   - For leaf requests, both CollectionItem.Auth and Request.Auth are cleared.
func PropagateAuthInherit(items []CollectionItem, overrides map[string]config.FolderOverride) {
	for i := range items {
		isOverriddenFolder := items[i].IsFolder() && len(overrides) > 0
		if isOverriddenFolder {
			if _, hasOverride := overrides[items[i].Name]; hasOverride {
				// Skip clearing this folder's auth; it has an explicit override.
				// Still recurse so its children can inherit from it.
				PropagateAuthInherit(*items[i].Items, overrides)
				continue
			}
		}

		// Clear item-level auth unless it is explicitly noauth or already nil.
		if items[i].Auth != nil && items[i].Auth.Type != "noauth" {
			items[i].Auth = nil
		}

		if items[i].IsFolder() {
			PropagateAuthInherit(*items[i].Items, overrides)
		} else if items[i].Request != nil {
			// Also clear request-level auth for leaf requests.
			if items[i].Request.Auth != nil && items[i].Request.Auth.Type != "noauth" {
				items[i].Request.Auth = nil
			}
		}
	}
}

// SetBaseURL replaces the host (and protocol/port) of every request URL with
// baseURL. If baseURL starts with "{{" it is treated as a Postman variable.
func SetBaseURL(items []CollectionItem, baseURL string) {
	for i := range items {
		if items[i].IsFolder() {
			SetBaseURL(*items[i].Items, baseURL)
			continue
		}
		if items[i].Request == nil || items[i].Request.URL == nil {
			continue
		}
		u := items[i].Request.URL
		if strings.HasPrefix(baseURL, "{{") {
			// Variable reference: host becomes a single-element array.
			u.Protocol = ""
			u.Port = ""
			u.Host = []string{baseURL}
			// Rebuild raw URL.
			u.Raw = rebuildRaw(baseURL, u.Path)
		} else {
			// Literal URL: parse protocol + host.
			proto, host := splitBaseURL(baseURL)
			u.Protocol = proto
			u.Host = strings.Split(host, ".")
			u.Port = ""
			u.Raw = rebuildRaw(baseURL, u.Path)
		}
	}
}

func rebuildRaw(base string, path []string) string {
	if len(path) == 0 {
		return base
	}
	// Join path segments, preserving :param notation.
	return base + "/" + strings.Join(path, "/")
}

func splitBaseURL(base string) (proto, host string) {
	if idx := strings.Index(base, "://"); idx >= 0 {
		return base[:idx], base[idx+3:]
	}
	return "https", base
}

// AddDocLinks appends a Markdown doc link to each leaf request's description.
func AddDocLinks(items []CollectionItem, baseDocURL string) {
	for i := range items {
		if items[i].IsFolder() {
			AddDocLinks(*items[i].Items, baseDocURL)
			continue
		}
		if items[i].Request == nil || items[i].Request.URL == nil {
			continue
		}
		path := items[i].Request.URL.Path
		if len(path) == 0 {
			continue
		}
		link := fmt.Sprintf("%s%s/operation/%s",
			baseDocURL,
			strings.Join(path[:max(0, len(path)-1)], "."),
			path[len(path)-1],
		)
		desc := items[i].Request.Description
		if desc != "" {
			desc += "\n"
		}
		items[i].Request.Description = desc + fmt.Sprintf("[Docs](%s)", link)
	}
}

// buildEvents merges config-defined scripts into an existing event list.
// Config values overwrite existing events of the same listen type.
func buildEvents(scriptsCfg *config.Scripts, existing []Event) []Event {
	eventMap := make(map[string]Event)
	for _, e := range existing {
		eventMap[e.Listen] = e
	}
	if scriptsCfg.PreRequest != "" {
		eventMap["prerequest"] = Event{
			Listen: "prerequest",
			Script: Script{
				Type: "text/javascript",
				Exec: splitLines(strings.TrimRight(scriptsCfg.PreRequest, "\n")),
			},
		}
	}
	if scriptsCfg.Test != "" {
		eventMap["test"] = Event{
			Listen: "test",
			Script: Script{
				Type: "text/javascript",
				Exec: splitLines(strings.TrimRight(scriptsCfg.Test, "\n")),
			},
		}
	}

	events := make([]Event, 0, len(eventMap))
	// Stable order: prerequest first, then test, then others.
	for _, listen := range []string{"prerequest", "test"} {
		if e, ok := eventMap[listen]; ok {
			events = append(events, e)
			delete(eventMap, listen)
		}
	}
	for _, e := range eventMap {
		events = append(events, e)
	}
	return events
}

// splitLines splits a string into lines (without trailing newline per line).
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
