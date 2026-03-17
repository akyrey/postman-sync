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
func MergeItems(old, new []CollectionItem) []CollectionItem {
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
			merged := MergeItems(oldChildren, newChildren)

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
		out := mergeLeaf(*o, n)
		result = append(result, out)
	}

	return result
}

// mergeLeaf merges a new leaf request item into the old one, preserving the
// old item's auth and event (scripts) customisations.
func mergeLeaf(old, new CollectionItem) CollectionItem {
	out := new      // start with the new item (new URL, method, body, headers, description)
	out.ID = old.ID // preserve Postman internal ID to avoid phantom diffs

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
