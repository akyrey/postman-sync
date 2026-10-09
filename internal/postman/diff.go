package postman

import (
	"encoding/json"
	"fmt"
)

// ChangeKind classifies an ItemChange.
type ChangeKind string

const (
	ChangeAdded    ChangeKind = "added"
	ChangeRemoved  ChangeKind = "removed"
	ChangeModified ChangeKind = "modified"
)

// maxDiffValueLen caps how much of a value is shown in a diff line.
const maxDiffValueLen = 60

// ItemChange describes how one folder or request differs between two
// collections. Path joins folder and item names with "/".
type ItemChange struct {
	Path    string
	Kind    ChangeKind
	Details []string
}

// DiffItems compares two item trees matched by name, like MergeItems, and
// returns the changes needed to turn old into new: changes in new's order,
// followed by removals in old's order. Added or removed folders are reported
// as a whole rather than request by request.
func DiffItems(old, new []CollectionItem) []ItemChange {
	return diffItems("", old, new)
}

func diffItems(prefix string, old, new []CollectionItem) []ItemChange {
	oldMap := itemsByName(old)
	newMap := itemsByName(new)
	var changes []ItemChange

	for _, n := range new {
		path := prefix + n.Name
		o, exists := oldMap[n.Name]
		switch {
		case !exists:
			changes = append(changes, ItemChange{Path: path, Kind: ChangeAdded, Details: folderSummary(n)})
		case n.IsFolder() != o.IsFolder():
			changes = append(changes, ItemChange{Path: path, Kind: ChangeModified, Details: []string{"replaced folder/request"}})
		default:
			if details := diffItem(*o, n); len(details) > 0 {
				changes = append(changes, ItemChange{Path: path, Kind: ChangeModified, Details: details})
			}
			if n.IsFolder() {
				changes = append(changes, diffItems(path+"/", *o.Items, *n.Items)...)
			}
		}
	}

	for _, o := range old {
		if _, kept := newMap[o.Name]; !kept {
			changes = append(changes, ItemChange{Path: prefix + o.Name, Kind: ChangeRemoved, Details: folderSummary(o)})
		}
	}
	return changes
}

func folderSummary(item CollectionItem) []string {
	if !item.IsFolder() {
		return nil
	}
	return []string{fmt.Sprintf("folder with %d request(s)", CountRequests(*item.Items))}
}

// CountRequests returns the number of leaf requests in an item tree.
func CountRequests(items []CollectionItem) int {
	n := 0
	for _, item := range items {
		if item.IsFolder() {
			n += CountRequests(*item.Items)
		} else {
			n++
		}
	}
	return n
}

// diffItem lists the changes of a single folder or request (not its children).
func diffItem(old, new CollectionItem) []string {
	var details []string
	if !old.IsFolder() {
		details = diffRequest(old.Request, new.Request)
	}
	if a, b := authLabel(old.Auth), authLabel(new.Auth); a != b || !sameJSON(old.Auth, new.Auth) {
		if a == b {
			details = append(details, "auth: "+b+" settings changed")
		} else {
			details = append(details, "auth: "+a+" → "+b)
		}
	}
	if !sameJSON(old.Events, new.Events) {
		details = append(details, "scripts changed")
	}
	if len(old.Responses) != len(new.Responses) {
		details = append(details, fmt.Sprintf("saved responses: %d → %d", len(old.Responses), len(new.Responses)))
	}
	return details
}

func authLabel(a *Auth) string {
	if a == nil {
		return "inherit"
	}
	return a.Type
}

func diffRequest(old, new *Request) []string {
	if old == nil {
		old = &Request{}
	}
	if new == nil {
		new = &Request{}
	}
	var details []string
	if old.Method != new.Method {
		details = append(details, fmt.Sprintf("method: %s → %s", old.Method, new.Method))
	}
	oldURL, newURL := orEmptyURL(old.URL), orEmptyURL(new.URL)
	if oldURL.Raw != newURL.Raw {
		details = append(details, fmt.Sprintf("url: %s → %s", quoteShort(oldURL.Raw), quoteShort(newURL.Raw)))
	}
	details = append(details, diffEntries("header", headerEntries(old.Header), headerEntries(new.Header))...)
	details = append(details, diffEntries("query", queryEntries(oldURL.Query), queryEntries(newURL.Query))...)
	details = append(details, diffEntries("path variable", variableEntries(oldURL.Variable), variableEntries(newURL.Variable))...)
	details = append(details, diffBody(old.Body, new.Body)...)
	if old.Description != new.Description {
		details = append(details, "description changed")
	}
	return details
}

func orEmptyURL(u *URL) *URL {
	if u == nil {
		return &URL{}
	}
	return u
}

func diffBody(old, new *Body) []string {
	switch {
	case old == nil && new == nil:
		return nil
	case old == nil:
		return []string{fmt.Sprintf("body: added (%s)", new.Mode)}
	case new == nil:
		return []string{fmt.Sprintf("body: removed (%s)", old.Mode)}
	case old.Mode != new.Mode:
		return []string{fmt.Sprintf("body: mode %q → %q", old.Mode, new.Mode)}
	}
	var details []string
	if old.Raw != new.Raw {
		details = append(details, "body: raw content changed")
	}
	details = append(details, diffEntries("form field", formEntries(old.FormData), formEntries(new.FormData))...)
	details = append(details, diffEntries("form field", formEntries(old.URLEncoded), formEntries(new.URLEncoded))...)
	if len(details) == 0 && !sameJSON(old, new) {
		details = append(details, "body: settings changed")
	}
	return details
}

// diffEntry is the comparable view of a header, param, variable or form field.
type diffEntry struct {
	key         string
	value       string
	disabled    bool
	description json.RawMessage
}

// diffEntries reports added, removed and modified entries matched by key.
func diffEntries(label string, old, new []diffEntry) []string {
	oldByKey := indexByKey(old, func(e diffEntry) string { return e.key })
	newByKey := indexByKey(new, func(e diffEntry) string { return e.key })
	var details []string
	for _, n := range new {
		o, ok := oldByKey[n.key]
		if !ok {
			details = append(details, fmt.Sprintf("%s %s: added", label, n.key))
			continue
		}
		if o.value != n.value {
			details = append(details, fmt.Sprintf("%s %s: value %s → %s", label, n.key, quoteShort(o.value), quoteShort(n.value)))
		}
		if o.disabled != n.disabled {
			details = append(details, fmt.Sprintf("%s %s: disabled %t → %t", label, n.key, o.disabled, n.disabled))
		}
		if !sameRawJSON(o.description, n.description) {
			details = append(details, fmt.Sprintf("%s %s: description changed", label, n.key))
		}
	}
	for _, o := range old {
		if _, ok := newByKey[o.key]; !ok {
			details = append(details, fmt.Sprintf("%s %s: removed", label, o.key))
		}
	}
	return details
}

func headerEntries(hs []Header) []diffEntry {
	out := make([]diffEntry, 0, len(hs))
	for _, h := range hs {
		out = append(out, diffEntry{key: h.Key, value: h.Value, disabled: h.Disabled, description: h.Extra["description"]})
	}
	return out
}

func queryEntries(ps []URLParam) []diffEntry {
	out := make([]diffEntry, 0, len(ps))
	for _, p := range ps {
		out = append(out, diffEntry{key: derefString(p.Key), value: derefString(p.Value), disabled: p.Disabled, description: p.Extra["description"]})
	}
	return out
}

func variableEntries(vs []Variable) []diffEntry {
	out := make([]diffEntry, 0, len(vs))
	for _, v := range vs {
		out = append(out, diffEntry{key: v.Key, value: v.Value, description: v.Extra["description"]})
	}
	return out
}

func formEntries(fs []FormParam) []diffEntry {
	out := make([]diffEntry, 0, len(fs))
	for _, f := range fs {
		value := rawJSONText(f.Extra["value"])
		if src, ok := f.Extra["src"]; ok {
			value = rawJSONText(src)
		}
		out = append(out, diffEntry{key: f.Key, value: value, disabled: f.Disabled, description: f.Extra["description"]})
	}
	return out
}

// quoteShort quotes s, truncating it to maxDiffValueLen runes.
func quoteShort(s string) string {
	if r := []rune(s); len(r) > maxDiffValueLen {
		s = string(r[:maxDiffValueLen]) + "…"
	}
	return fmt.Sprintf("%q", s)
}

// collectionPath labels collection-level changes in DiffCollections.
const collectionPath = "(collection)"

// DiffCollections is DiffItems plus collection-level auth and scripts.
func DiffCollections(old, new *Collection) []ItemChange {
	var details []string
	if !sameJSON(old.Auth, new.Auth) {
		details = append(details, "auth: "+authLabel(old.Auth)+" → "+authLabel(new.Auth))
	}
	if !sameJSON(old.Events, new.Events) {
		details = append(details, "scripts changed")
	}
	var changes []ItemChange
	if len(details) > 0 {
		changes = append(changes, ItemChange{Path: collectionPath, Kind: ChangeModified, Details: details})
	}
	return append(changes, DiffItems(old.Items, new.Items)...)
}
