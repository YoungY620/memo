package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

var schemas = map[string]string{
	"arch.json": `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"modules": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"name": {"type": "string"},
						"description": {"type": "string"},
						"interfaces": {"type": "string"}
					},
					"required": ["name", "description", "interfaces"]
				}
			},
			"relationships": {
				"type": "object",
				"properties": {
					"diagram": {"type": "string"},
					"notes": {"type": "string"}
				},
				"required": ["diagram", "notes"]
			}
		},
		"required": ["modules", "relationships"]
	}`,
	"interface.json": `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"external": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"type": {"type": "string"},
						"name": {"type": "string"},
						"params": {"type": "string"},
						"description": {"type": "string"}
					},
					"required": ["type", "name", "params", "description"]
				}
			},
			"internal": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"type": {"type": "string"},
						"name": {"type": "string"},
						"params": {"type": "string"},
						"description": {"type": "string"}
					},
					"required": ["type", "name", "params", "description"]
				}
			}
		},
		"required": ["external", "internal"]
	}`,
	"stories.json": `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"stories": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"title": {"type": "string"},
						"tags": {"type": "array", "items": {"type": "string"}},
						"lines": {"type": "array", "items": {"type": "string"}}
					},
					"required": ["title", "tags", "lines"]
				}
			}
		},
		"required": ["stories"]
	}`,
	"issues.json": `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"issues": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"tags": {"type": "array", "items": {"type": "string"}},
						"title": {"type": "string"},
						"description": {"type": "string"},
						"locations": {
							"type": "array",
							"items": {
								"type": "object",
								"properties": {
									"file": {"type": "string"},
									"keyword": {"type": "string"},
									"line": {"type": "integer"}
								},
								"required": ["file", "keyword", "line"]
							}
						}
					},
					"required": ["tags", "title", "description", "locations"]
				}
			}
		},
		"required": ["issues"]
	}`,
}

type ValidationResult struct {
	Valid  bool
	Errors []string
}

func ValidateIndex(indexDir string) ValidationResult {
	var allErrors []string

	for filename, schemaJSON := range schemas {
		filePath := filepath.Join(indexDir, filename)
		data, err := os.ReadFile(filePath)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("%s: %v", filename, err))
			continue
		}

		schemaLoader := gojsonschema.NewStringLoader(schemaJSON)
		documentLoader := gojsonschema.NewBytesLoader(data)

		result, err := gojsonschema.Validate(schemaLoader, documentLoader)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("%s: schema validation error: %v", filename, err))
			continue
		}

		if !result.Valid() {
			for _, e := range result.Errors() {
				allErrors = append(allErrors, fmt.Sprintf("%s: %s", filename, e.String()))
			}
		}
	}

	return ValidationResult{
		Valid:  len(allErrors) == 0,
		Errors: allErrors,
	}
}

func FormatValidationErrors(result ValidationResult) string {
	if result.Valid {
		return ""
	}
	return strings.Join(result.Errors, "\n")
}
