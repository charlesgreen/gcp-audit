package redact

import (
	"bytes"
	"encoding/json"
)

const marker = "[REDACTED]"

var secretKeys = map[string]bool{
	"privateKeyData":   true,
	"privateKey":       true,
	"sharedSecret":     true,
	"sharedSecretHash": true,
	"clientSecret":     true,
	"password":         true,
	"psk":              true,
}

// JSON replaces secret-bearing fields.
func JSON(in []byte) ([]byte, error) {
	trim := bytes.TrimSpace(in)
	if len(trim) == 0 {
		return in, nil
	}
	var v any
	if err := json.Unmarshal(trim, &v); err != nil {
		return in, nil
	}
	walk(v)
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

func walk(v any) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			if secretKeys[k] {
				if s, ok := child.(string); ok && s != "" {
					t[k] = marker
					continue
				}
			}
			walk(child)
		}
	case []any:
		for _, child := range t {
			walk(child)
		}
	}
}
