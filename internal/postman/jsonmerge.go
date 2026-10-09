package postman

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// rawBodyIndent matches the indentation Postman uses for generated raw bodies.
const rawBodyIndent = "  "

type jsonKind int

const (
	jsonScalar jsonKind = iota
	jsonObject
	jsonArray
)

// jsonNode is a minimal JSON tree that keeps object key order, so merged
// bodies keep the field order of the spec instead of being sorted.
type jsonNode struct {
	kind   jsonKind
	fields []jsonField     // jsonObject
	items  []*jsonNode     // jsonArray
	scalar json.RawMessage // jsonScalar
}

type jsonField struct {
	key   string
	value *jsonNode
}

func (n *jsonNode) field(key string) (*jsonNode, bool) {
	for _, f := range n.fields {
		if f.key == key {
			return f.value, true
		}
	}
	return nil, false
}

// mergeRawJSONBody merges a user-edited raw body (old) with the body generated
// from the spec (new):
//   - object fields present in both keep the old value (recursively),
//   - fields only in the spec are added, fields no longer in the spec are dropped,
//   - when a value changes kind (e.g. string → object) the spec wins,
//   - arrays keep the old elements; object elements are merged against the
//     spec's first element so they gain/lose fields like any other object.
//
// If either side is not valid JSON (e.g. it contains unquoted {{variables}}),
// the old body is considered hand-written and kept unchanged.
func mergeRawJSONBody(old, new string) string {
	if strings.TrimSpace(old) == "" {
		return new
	}
	if old == new {
		return old
	}
	oldNode, err := parseJSON(old)
	if err != nil {
		return old
	}
	newNode, err := parseJSON(new)
	if err != nil {
		return old
	}
	out, err := formatJSON(mergeJSON(oldNode, newNode))
	if err != nil {
		return old
	}
	return out
}

func mergeJSON(old, new *jsonNode) *jsonNode {
	if old.kind != new.kind {
		return new
	}
	switch new.kind {
	case jsonObject:
		merged := &jsonNode{kind: jsonObject, fields: make([]jsonField, 0, len(new.fields))}
		for _, f := range new.fields {
			value := f.value
			if ov, ok := old.field(f.key); ok {
				value = mergeJSON(ov, f.value)
			}
			merged.fields = append(merged.fields, jsonField{key: f.key, value: value})
		}
		return merged
	case jsonArray:
		if len(new.items) == 0 || new.items[0].kind != jsonObject {
			return old
		}
		template := new.items[0]
		merged := &jsonNode{kind: jsonArray, items: make([]*jsonNode, 0, len(old.items))}
		for _, item := range old.items {
			merged.items = append(merged.items, mergeJSON(item, template))
		}
		return merged
	default:
		return old
	}
}

func parseJSON(s string) (*jsonNode, error) {
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	n, err := decodeJSONNode(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("unexpected data after JSON value")
	}
	return n, nil
}

func decodeJSONNode(dec *json.Decoder) (*jsonNode, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	delim, isDelim := tok.(json.Delim)
	if !isDelim {
		raw, err := encodeJSONScalar(tok)
		return &jsonNode{kind: jsonScalar, scalar: raw}, err
	}

	switch delim {
	case '{':
		n := &jsonNode{kind: jsonObject}
		for dec.More() {
			keyTok, err := dec.Token()
			if err != nil {
				return nil, err
			}
			key, ok := keyTok.(string)
			if !ok {
				return nil, fmt.Errorf("unexpected object key %v", keyTok)
			}
			value, err := decodeJSONNode(dec)
			if err != nil {
				return nil, err
			}
			n.fields = append(n.fields, jsonField{key: key, value: value})
		}
		_, err := dec.Token() // closing '}'
		return n, err
	case '[':
		n := &jsonNode{kind: jsonArray}
		for dec.More() {
			item, err := decodeJSONNode(dec)
			if err != nil {
				return nil, err
			}
			n.items = append(n.items, item)
		}
		_, err := dec.Token() // closing ']'
		return n, err
	default:
		return nil, fmt.Errorf("unexpected delimiter %q", delim)
	}
}

// encodeJSONScalar encodes a decoded token without HTML escaping, so
// placeholders such as "<string>" stay readable.
func encodeJSONScalar(v any) (json.RawMessage, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

func formatJSON(n *jsonNode) (string, error) {
	var compact bytes.Buffer
	if err := writeJSONNode(&compact, n); err != nil {
		return "", err
	}
	var out bytes.Buffer
	if err := json.Indent(&out, compact.Bytes(), "", rawBodyIndent); err != nil {
		return "", err
	}
	return out.String(), nil
}

func writeJSONNode(buf *bytes.Buffer, n *jsonNode) error {
	switch n.kind {
	case jsonObject:
		buf.WriteByte('{')
		for i, f := range n.fields {
			if i > 0 {
				buf.WriteByte(',')
			}
			key, err := encodeJSONScalar(f.key)
			if err != nil {
				return err
			}
			buf.Write(key)
			buf.WriteByte(':')
			if err := writeJSONNode(buf, f.value); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	case jsonArray:
		buf.WriteByte('[')
		for i, item := range n.items {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeJSONNode(buf, item); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	default:
		buf.Write(n.scalar)
	}
	return nil
}
