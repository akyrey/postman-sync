package postman

// MergeItems merges a newly-generated collection's items into the existing
// collection's items, preserving manual customisations.
//
// Rules:
//   - Items matched by name: update request fields (url, method, body, headers,
//     description) from new; keep auth and events (scripts) from old.
//   - Items only in new: added as-is (new endpoints).
//   - Items only in old: removed (spec is source of truth).
//   - Folders are recursed into.
//
// Request data (headers, params, body) always comes from new; use
// MergeItemsWith to preserve the user's edits.
func MergeItems(old, new []CollectionItem) []CollectionItem {
	return MergeItemsWith(old, new, MergeOptions{})
}

// MergeOptions tunes how MergeItemsWith reconciles matched leaf requests.
type MergeOptions struct {
	// PreserveRequestData keeps header/param values, enabled state and body
	// edits from old, see mergeRequest.
	PreserveRequestData bool
}

// MergeItemsWith is MergeItems with configurable request merging.
func MergeItemsWith(old, new []CollectionItem, opts MergeOptions) []CollectionItem {
	oldMap := make(map[string]*CollectionItem, len(old))
	for i := range old {
		item := old[i] // local copy
		oldMap[item.Name] = &item
	}

	result := make([]CollectionItem, 0, len(new))

	for _, n := range new {
		o, exists := oldMap[n.Name]
		if !exists {
			// Brand-new item: strip IDs for portability and add.
			n.ID = ""
			result = append(result, n)
			continue
		}

		if n.IsFolder() {
			// Both old and new are folders – recurse.
			oldChildren := []CollectionItem{}
			if o.Items != nil {
				oldChildren = *o.Items
			}
			newChildren := []CollectionItem{}
			if n.Items != nil {
				newChildren = *n.Items
			}
			merged := MergeItemsWith(oldChildren, newChildren, opts)

			// Carry over folder-level auth and events from old if new doesn't
			// define them (they were set by config on first run and may have
			// been tweaked manually since).
			out := n
			out.Items = &merged
			if o.Auth != nil && out.Auth == nil {
				out.Auth = o.Auth
			}
			if len(o.Events) > 0 && len(out.Events) == 0 {
				out.Events = o.Events
			}
			result = append(result, out)
			continue
		}

		// Both are leaf requests: update request data from new, preserve
		// auth and events from old.
		out := mergeLeaf(*o, n, opts)
		result = append(result, out)
	}

	return result
}

// mergeLeaf merges a new leaf request item into the old one, preserving the
// old item's auth and event (scripts) customisations.
func mergeLeaf(old, new CollectionItem, opts MergeOptions) CollectionItem {
	out := new      // start with the new item (new URL, method, body, headers, description)
	out.ID = old.ID // preserve Postman internal ID to avoid phantom diffs

	if opts.PreserveRequestData {
		out.Request = mergeRequest(old.Request, new.Request)
	}

	// Preserve manual auth customisations.
	if old.Auth != nil {
		out.Auth = old.Auth
	}

	// Preserve manually-set scripts (events).
	if len(old.Events) > 0 {
		out.Events = old.Events
	}

	// Preserve saved example responses.
	if len(old.Responses) > 0 {
		out.Responses = old.Responses
	}

	return out
}
