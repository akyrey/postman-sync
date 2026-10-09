package postman

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// Extra holds JSON fields that are not explicitly modelled by a struct.
//
// Postman's collection format is much richer than what this tool needs to
// manipulate (descriptions, body options, form data, protocol profile
// behaviour, ...). Types embedding an Extra field keep those unknown fields on
// decode and write them back on encode, so a fetch → transform → update cycle
// never silently drops data.
type Extra map[string]json.RawMessage

var knownKeysCache sync.Map // reflect.Type → map[string]struct{}

// knownKeys returns the JSON keys declared by the struct fields of t.
func knownKeys(t reflect.Type) map[string]struct{} {
	if cached, ok := knownKeysCache.Load(t); ok {
		return cached.(map[string]struct{})
	}
	keys := make(map[string]struct{}, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("json")
		if tag == "-" || !f.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(tag, ",")
		if name == "" {
			name = f.Name
		}
		keys[name] = struct{}{}
	}
	knownKeysCache.Store(t, keys)
	return keys
}

// unmarshalWithExtra decodes data into known (a pointer to a method-less alias
// struct) and stores every key not declared by its fields into extra.
func unmarshalWithExtra(data []byte, known any, extra *Extra) error {
	if err := json.Unmarshal(data, known); err != nil {
		return err
	}
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}
	keys := knownKeys(reflect.TypeOf(known).Elem())
	unknown := Extra{}
	for k, v := range all {
		if _, ok := keys[k]; !ok {
			unknown[k] = v
		}
	}
	*extra = nil
	if len(unknown) > 0 {
		*extra = unknown
	}
	return nil
}

// marshalWithExtra encodes known (a method-less alias struct value) and merges
// in the extra fields. Explicitly modelled fields always take precedence.
func marshalWithExtra(known any, extra Extra) ([]byte, error) {
	b, err := json.Marshal(known)
	if err != nil || len(extra) == 0 {
		return b, err
	}
	var out map[string]json.RawMessage
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("re-reading encoded fields: %w", err)
	}
	keys := knownKeys(reflect.TypeOf(known))
	for k, v := range extra {
		if _, modelled := keys[k]; modelled {
			continue // a modelled field that was cleared must stay cleared
		}
		out[k] = v
	}
	return json.Marshal(out)
}

// sameRawJSON reports whether two raw JSON values are semantically equal.
// Byte comparison is not enough: encoding/json may escape characters such as
// "<" (as \u003c) when re-encoding, while Postman returns them literally.
// Missing (nil) values are only equal to each other.
func sameRawJSON(a, b json.RawMessage) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	var va, vb any
	if json.Unmarshal(a, &va) != nil || json.Unmarshal(b, &vb) != nil {
		return bytes.Equal(a, b)
	}
	return reflect.DeepEqual(va, vb)
}

// rawJSONText returns the text of a raw JSON value: the decoded string for a
// JSON string, the compact JSON otherwise, and "" when missing.
func rawJSONText(v json.RawMessage) string {
	if v == nil {
		return ""
	}
	var s string
	if json.Unmarshal(v, &s) == nil {
		return s
	}
	var compact bytes.Buffer
	if json.Compact(&compact, v) != nil {
		return string(v)
	}
	return compact.String()
}
