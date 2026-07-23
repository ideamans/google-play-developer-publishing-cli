package api

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Doc is a decoded JSON object. The Android Publisher API returns plain REST
// resources with no envelope, so commands work with these generic documents
// rather than a struct per schema.
type Doc map[string]any

// Str returns a string field. Numbers are formatted, everything else yields "".
func (d Doc) Str(key string) string {
	if d == nil {
		return ""
	}
	switch v := d[key].(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		return ""
	}
}

// Int returns an integer field. String-encoded int64s (the API returns
// versionCodes and timestamps as strings in some responses) are parsed too.
func (d Doc) Int(key string) int64 {
	if d == nil {
		return 0
	}
	switch v := d[key].(type) {
	case float64:
		return int64(v)
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}

// Float returns a float field.
func (d Doc) Float(key string) float64 {
	if d == nil {
		return 0
	}
	switch v := d[key].(type) {
	case float64:
		return v
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0
		}
		return f
	default:
		return 0
	}
}

// Bool returns a boolean field.
func (d Doc) Bool(key string) bool {
	if d == nil {
		return false
	}
	v, _ := d[key].(bool)
	return v
}

// Has reports whether the key is present.
func (d Doc) Has(key string) bool {
	if d == nil {
		return false
	}
	_, ok := d[key]
	return ok
}

// Doc returns a nested object, or nil.
func (d Doc) Doc(key string) Doc {
	if d == nil {
		return nil
	}
	m, _ := d[key].(map[string]any)
	return Doc(m)
}

// Docs returns a nested array of objects, skipping non-object elements.
func (d Doc) Docs(key string) []Doc {
	if d == nil {
		return nil
	}
	arr, _ := d[key].([]any)
	out := make([]Doc, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			out = append(out, Doc(m))
		}
	}
	return out
}

// Strings returns a nested array of strings, formatting numbers as decimals.
func (d Doc) Strings(key string) []string {
	if d == nil {
		return nil
	}
	arr, _ := d[key].([]any)
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		switch v := item.(type) {
		case string:
			out = append(out, v)
		case float64:
			out = append(out, strconv.FormatFloat(v, 'f', -1, 64))
		}
	}
	return out
}

// Decode re-decodes the document into v.
func (d Doc) Decode(v any) error {
	b, err := json.Marshal(d)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// DecodeInto re-decodes a nested field into v.
func (d Doc) DecodeInto(key string, v any) error {
	raw, ok := d[key]
	if !ok {
		return fmt.Errorf("no field %q in response", key)
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
