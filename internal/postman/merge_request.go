package postman

import "strings"

// mergeRequest combines a request regenerated from the spec (new) with the
// existing request (old), keeping data the user edited in Postman:
//   - headers, query params, form-data and urlencoded fields matched by key
//     keep their old value and enabled/disabled state, while the description
//     (which carries "(Required)" and type info) is refreshed from the spec;
//   - path variables keep their old value;
//   - raw JSON bodies are merged field by field (see mergeRawJSONBody).
//
// Entries that only exist in the old request are dropped when they came from
// the spec (they have a description) and kept when they were added by hand.
// Everything else (method, URL structure, description) comes from new.
func mergeRequest(old, new *Request) *Request {
	if old == nil || new == nil {
		return new
	}
	out := *new
	out.Header = mergeByKey(old.Header, new.Header, headerMerge)
	out.URL = mergeURL(old.URL, new.URL)
	out.Body = mergeBody(old.Body, new.Body)
	return &out
}

func mergeURL(old, new *URL) *URL {
	if old == nil || new == nil {
		return new
	}
	out := *new
	out.Query = mergeByKey(old.Query, new.Query, urlParamMerge)
	out.Variable = mergeByKey(old.Variable, new.Variable, pathVariableMerge)
	return &out
}

func mergeBody(old, new *Body) *Body {
	if old == nil || new == nil || old.Mode != new.Mode {
		return new
	}
	out := *new
	out.Disabled = old.Disabled
	switch new.Mode {
	case "raw":
		out.Raw = mergeRawJSONBody(old.Raw, new.Raw)
	case "formdata":
		out.FormData = mergeByKey(old.FormData, new.FormData, formParamMerge)
	case "urlencoded":
		out.URLEncoded = mergeByKey(old.URLEncoded, new.URLEncoded, formParamMerge)
	default:
		// graphql, file, ...: nothing to reconcile, keep the user's body.
		return old
	}
	return &out
}

// keyedMerge describes how to merge a list of key/value entries.
type keyedMerge[T any] struct {
	// key identifies an entry; entries with the same key are merged.
	key func(T) string
	// carry returns the spec entry with user-owned fields copied from old.
	carry func(old, new T) T
	// keepManual reports whether an old entry missing from the spec was added
	// by hand and must be kept. nil drops every old-only entry.
	keepManual func(T) bool
}

// mergeByKey returns the entries of new, in spec order, merged with the old
// entry of the same key, followed by the manually-added old entries.
func mergeByKey[T any](old, new []T, m keyedMerge[T]) []T {
	if len(old) == 0 {
		return new
	}
	oldByKey := make(map[string]T, len(old))
	for _, o := range old {
		k := m.key(o)
		if _, dup := oldByKey[k]; !dup {
			oldByKey[k] = o
		}
	}

	out := make([]T, 0, len(new))
	seen := make(map[string]bool, len(new))
	for _, n := range new {
		k := m.key(n)
		if o, ok := oldByKey[k]; ok && !seen[k] {
			n = m.carry(o, n)
		}
		seen[k] = true
		out = append(out, n)
	}

	if m.keepManual == nil {
		return out
	}
	for _, o := range old {
		k := m.key(o)
		if !seen[k] && m.keepManual(o) {
			seen[k] = true
			out = append(out, o)
		}
	}
	return out
}

// generatedBySpec reports whether an entry was produced by Postman's OpenAPI
// importer, which always attaches the parameter description. Entries without a
// description are treated as added by hand.
func generatedBySpec(e Extra) bool {
	_, ok := e["description"]
	return ok
}

var headerMerge = keyedMerge[Header]{
	key: func(h Header) string { return strings.ToLower(h.Key) },
	carry: func(old, new Header) Header {
		new.Value = old.Value
		new.Disabled = old.Disabled
		return new
	},
	keepManual: func(h Header) bool { return !generatedBySpec(h.Extra) },
}

var urlParamMerge = keyedMerge[URLParam]{
	key: func(p URLParam) string {
		if p.Key == nil {
			return ""
		}
		return *p.Key
	},
	carry: func(old, new URLParam) URLParam {
		new.Value = old.Value
		new.Disabled = old.Disabled
		return new
	},
	keepManual: func(p URLParam) bool { return !generatedBySpec(p.Extra) },
}

// Path variables must match the URL path, so stale ones are always dropped.
var pathVariableMerge = keyedMerge[Variable]{
	key: func(v Variable) string { return v.Key },
	carry: func(old, new Variable) Variable {
		new.Value = old.Value
		return new
	},
}

// formUserFields are the form field properties owned by the user.
var formUserFields = []string{"value", "src"}

var formParamMerge = keyedMerge[FormParam]{
	key: func(p FormParam) string { return p.Key },
	carry: func(old, new FormParam) FormParam {
		if old.Type != new.Type {
			return new // text ↔ file: the old value no longer applies
		}
		extra := make(Extra, len(new.Extra)+len(formUserFields))
		for k, v := range new.Extra {
			extra[k] = v
		}
		for _, k := range formUserFields {
			if v, ok := old.Extra[k]; ok {
				extra[k] = v
			}
		}
		new.Extra = extra
		new.Disabled = old.Disabled
		return new
	},
	keepManual: func(p FormParam) bool { return !generatedBySpec(p.Extra) },
}
