package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// PathSegment represents a single segment in a query path
type PathSegment struct {
	Key     string
	Index   int
	IsIndex bool
}

// ListKeysResult is the result of list_keys operation
type ListKeysResult struct {
	Type   string   `json:"type"`            // "dict" or "list"
	Keys   []string `json:"keys,omitempty"`  // for dict
	Length int      `json:"length,omitempty"` // for list
}

// GetValueResult is the result of get_value operation
type GetValueResult struct {
	Value string `json:"value"`
}

// Allowed index files
var allowedFiles = map[string]bool{
	"arch":      true,
	"interface": true,
	"stories":   true,
	"issues":    true,
}

// ParsePath parses a path like [arch][modules][0][name] into file and segments
// Uses a state machine to handle escaping
func ParsePath(path string) (file string, segments []PathSegment, err error) {
	if len(path) == 0 {
		return "", nil, fmt.Errorf("empty path")
	}

	var result []PathSegment
	var current strings.Builder
	inBracket := false
	escaped := false

	for i := 0; i < len(path); i++ {
		c := path[i]

		if escaped {
			// Handle escape sequences
			switch c {
			case '[', ']', '\\':
				current.WriteByte(c)
			default:
				return "", nil, fmt.Errorf("invalid escape sequence at position %d", i)
			}
			escaped = false
			continue
		}

		switch c {
		case '\\':
			escaped = true
		case '[':
			if inBracket {
				return "", nil, fmt.Errorf("unexpected '[' at position %d", i)
			}
			inBracket = true
		case ']':
			if !inBracket {
				return "", nil, fmt.Errorf("unexpected ']' at position %d", i)
			}
			inBracket = false
			key := current.String()
			current.Reset()

			if key == "" {
				return "", nil, fmt.Errorf("empty segment at position %d", i)
			}

			// Check if it's a numeric index
			if idx, err := strconv.Atoi(key); err == nil && idx >= 0 {
				result = append(result, PathSegment{Index: idx, IsIndex: true})
			} else {
				// Validate key: no control characters
				if err := validateKey(key); err != nil {
					return "", nil, err
				}
				result = append(result, PathSegment{Key: key, IsIndex: false})
			}
		default:
			if !inBracket {
				return "", nil, fmt.Errorf("unexpected character '%c' at position %d", c, i)
			}
			current.WriteByte(c)
		}
	}

	if inBracket {
		return "", nil, fmt.Errorf("unclosed bracket")
	}
	if escaped {
		return "", nil, fmt.Errorf("trailing escape character")
	}
	if len(result) == 0 {
		return "", nil, fmt.Errorf("no segments in path")
	}

	// First segment must be the file name
	if result[0].IsIndex {
		return "", nil, fmt.Errorf("first segment must be file name, not index")
	}
	file = result[0].Key
	if !allowedFiles[file] {
		return "", nil, fmt.Errorf("invalid file: %s (allowed: arch, interface, stories, issues)", file)
	}

	return file, result[1:], nil
}

// validateKey checks for forbidden characters in keys
func validateKey(key string) error {
	if len(key) > 100 {
		return fmt.Errorf("key too long: %d chars (max 100)", len(key))
	}
	for i, c := range key {
		if c < 32 || c == 127 {
			return fmt.Errorf("control character in key at position %d", i)
		}
	}
	return nil
}

// traverse navigates through the data using segments
func traverse(data any, segments []PathSegment) (any, error) {
	current := data

	for i, seg := range segments {
		if seg.IsIndex {
			arr, ok := current.([]any)
			if !ok {
				return nil, fmt.Errorf("segment %d: expected array, got %T", i, current)
			}
			if seg.Index < 0 || seg.Index >= len(arr) {
				return nil, fmt.Errorf("segment %d: index %d out of bounds (length %d)", i, seg.Index, len(arr))
			}
			current = arr[seg.Index]
		} else {
			obj, ok := current.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("segment %d: expected object, got %T", i, current)
			}
			val, exists := obj[seg.Key]
			if !exists {
				return nil, fmt.Errorf("segment %d: key '%s' not found", i, seg.Key)
			}
			current = val
		}
	}

	return current, nil
}

// loadFile loads and parses a JSON file from the index directory
func loadFile(indexDir, file string) (any, error) {
	path := filepath.Join(indexDir, file+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	return result, nil
}

// ListKeys returns keys/length for the value at the given path
func ListKeys(indexDir, path string) (*ListKeysResult, error) {
	file, segments, err := ParsePath(path)
	if err != nil {
		return nil, err
	}

	data, err := loadFile(indexDir, file)
	if err != nil {
		return nil, err
	}

	value, err := traverse(data, segments)
	if err != nil {
		return nil, err
	}

	switch v := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		return &ListKeysResult{Type: "dict", Keys: keys}, nil
	case []any:
		return &ListKeysResult{Type: "list", Length: len(v)}, nil
	default:
		return nil, fmt.Errorf("value is not a dict or list, it's %T", value)
	}
}

// GetValue returns the JSON string of the value at the given path
func GetValue(indexDir, path string) (*GetValueResult, error) {
	file, segments, err := ParsePath(path)
	if err != nil {
		return nil, err
	}

	data, err := loadFile(indexDir, file)
	if err != nil {
		return nil, err
	}

	value, err := traverse(data, segments)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal value: %w", err)
	}

	return &GetValueResult{Value: string(jsonBytes)}, nil
}
