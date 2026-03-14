package resend

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitRef(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		expected []string
	}{
		{
			name:     "standard schema ref",
			ref:      "#/components/schemas/Foo",
			expected: []string{"#", "components", "schemas", "Foo"},
		},
		{
			name:     "empty string",
			ref:      "",
			expected: nil,
		},
		{
			name:     "just hash",
			ref:      "#",
			expected: []string{"#"},
		},
		{
			name:     "trailing slash",
			ref:      "#/components/schemas/Foo/",
			expected: []string{"#", "components", "schemas", "Foo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, splitRef(tt.ref))
		})
	}
}

func TestResolveRef(t *testing.T) {
	schemas := map[string]any{
		"Email": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{"type": "string"},
			},
		},
	}

	t.Run("resolves valid local ref", func(t *testing.T) {
		result := resolveRef("#/components/schemas/Email", schemas)
		require.NotNil(t, result)
		assert.Equal(t, "object", result["type"])
	})

	t.Run("returns nil for missing schema", func(t *testing.T) {
		assert.Nil(t, resolveRef("#/components/schemas/Missing", schemas))
	})

	t.Run("returns nil for external ref", func(t *testing.T) {
		assert.Nil(t, resolveRef("https://example.com/spec.yaml#/Foo", schemas))
	})

	t.Run("returns nil for wrong path depth", func(t *testing.T) {
		assert.Nil(t, resolveRef("#/definitions/Foo", schemas))
	})

	t.Run("returns nil for empty ref", func(t *testing.T) {
		assert.Nil(t, resolveRef("", schemas))
	})
}

func TestResolveSchemas(t *testing.T) {
	t.Run("extracts schemas from spec", func(t *testing.T) {
		spec := map[string]any{
			"components": map[string]any{
				"schemas": map[string]any{
					"Foo": map[string]any{"type": "object"},
				},
			},
		}
		schemas := resolveSchemas(spec)
		require.NotNil(t, schemas)
		assert.Contains(t, schemas, "Foo")
	})

	t.Run("returns nil when no components", func(t *testing.T) {
		assert.Nil(t, resolveSchemas(map[string]any{}))
	})

	t.Run("returns nil when no schemas", func(t *testing.T) {
		spec := map[string]any{
			"components": map[string]any{},
		}
		assert.Nil(t, resolveSchemas(spec))
	})
}

func TestParamExampleValue(t *testing.T) {
	t.Run("uses existing example on param", func(t *testing.T) {
		p := map[string]any{"example": "my-value"}
		assert.Equal(t, "my-value", paramExampleValue("whatever", p))
	})

	t.Run("uses schema-level example", func(t *testing.T) {
		p := map[string]any{
			"schema": map[string]any{"example": "schema-val"},
		}
		assert.Equal(t, "schema-val", paramExampleValue("whatever", p))
	})

	t.Run("generates UUID for id params", func(t *testing.T) {
		p := map[string]any{}
		assert.Equal(t, "479e3145-dd38-476b-932c-529ceb705947", paramExampleValue("email_id", p))
		assert.Equal(t, "479e3145-dd38-476b-932c-529ceb705947", paramExampleValue("id", p))
		assert.Equal(t, "479e3145-dd38-476b-932c-529ceb705947", paramExampleValue("domain_id", p))
	})

	t.Run("generates 10 for limit", func(t *testing.T) {
		assert.Equal(t, 10, paramExampleValue("limit", map[string]any{}))
	})

	t.Run("generates empty string for pagination cursors", func(t *testing.T) {
		assert.Equal(t, "", paramExampleValue("after", map[string]any{}))
		assert.Equal(t, "", paramExampleValue("before", map[string]any{}))
	})

	t.Run("generates example for unknown param", func(t *testing.T) {
		assert.Equal(t, "example", paramExampleValue("foo", map[string]any{}))
	})
}

func TestBuildValueFromSchema(t *testing.T) {
	schemas := map[string]any{
		"Email": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id":      map[string]any{"type": "string", "format": "uuid"},
				"subject": map[string]any{"type": "string", "example": "Hello World"},
			},
		},
	}

	t.Run("string without format", func(t *testing.T) {
		schema := map[string]any{"type": "string"}
		assert.Equal(t, "example", buildValueFromSchema(schema, nil, 0))
	})

	t.Run("string with uuid format", func(t *testing.T) {
		schema := map[string]any{"type": "string", "format": "uuid"}
		assert.Equal(t, "479e3145-dd38-476b-932c-529ceb705947", buildValueFromSchema(schema, nil, 0))
	})

	t.Run("string with date-time format", func(t *testing.T) {
		schema := map[string]any{"type": "string", "format": "date-time"}
		assert.Equal(t, "2023-10-06T23:47:56.678Z", buildValueFromSchema(schema, nil, 0))
	})

	t.Run("string with example", func(t *testing.T) {
		schema := map[string]any{"type": "string", "example": "custom"}
		assert.Equal(t, "custom", buildValueFromSchema(schema, nil, 0))
	})

	t.Run("integer default", func(t *testing.T) {
		schema := map[string]any{"type": "integer"}
		assert.Equal(t, 1, buildValueFromSchema(schema, nil, 0))
	})

	t.Run("integer with example", func(t *testing.T) {
		schema := map[string]any{"type": "integer", "example": 42}
		assert.Equal(t, 42, buildValueFromSchema(schema, nil, 0))
	})

	t.Run("number default", func(t *testing.T) {
		schema := map[string]any{"type": "number"}
		assert.Equal(t, 1.0, buildValueFromSchema(schema, nil, 0))
	})

	t.Run("boolean default", func(t *testing.T) {
		schema := map[string]any{"type": "boolean"}
		assert.Equal(t, false, buildValueFromSchema(schema, nil, 0))
	})

	t.Run("boolean with example", func(t *testing.T) {
		schema := map[string]any{"type": "boolean", "example": true}
		assert.Equal(t, true, buildValueFromSchema(schema, nil, 0))
	})

	t.Run("object with properties", func(t *testing.T) {
		schema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":  map[string]any{"type": "string", "example": "Alice"},
				"count": map[string]any{"type": "integer"},
			},
		}
		result := buildValueFromSchema(schema, nil, 0)
		obj, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Alice", obj["name"])
		assert.Equal(t, 1, obj["count"])
	})

	t.Run("object without properties returns empty map", func(t *testing.T) {
		schema := map[string]any{"type": "object"}
		result := buildValueFromSchema(schema, nil, 0)
		assert.Equal(t, map[string]any{}, result)
	})

	t.Run("array with items", func(t *testing.T) {
		schema := map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "string", "example": "item1"},
		}
		result := buildValueFromSchema(schema, nil, 0)
		arr, ok := result.([]any)
		require.True(t, ok)
		require.Len(t, arr, 1)
		assert.Equal(t, "item1", arr[0])
	})

	t.Run("array without items returns empty slice", func(t *testing.T) {
		schema := map[string]any{"type": "array"}
		assert.Equal(t, []any{}, buildValueFromSchema(schema, nil, 0))
	})

	t.Run("resolves $ref", func(t *testing.T) {
		schema := map[string]any{"$ref": "#/components/schemas/Email"}
		result := buildValueFromSchema(schema, schemas, 0)
		obj, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "479e3145-dd38-476b-932c-529ceb705947", obj["id"])
		assert.Equal(t, "Hello World", obj["subject"])
	})

	t.Run("returns nil for unresolvable $ref", func(t *testing.T) {
		schema := map[string]any{"$ref": "#/components/schemas/Missing"}
		assert.Nil(t, buildValueFromSchema(schema, schemas, 0))
	})

	t.Run("stops at max depth", func(t *testing.T) {
		schema := map[string]any{"type": "string"}
		assert.Nil(t, buildValueFromSchema(schema, nil, 6))
	})

	t.Run("schema-level example takes precedence", func(t *testing.T) {
		schema := map[string]any{
			"type":    "object",
			"example": map[string]any{"custom": true},
			"properties": map[string]any{
				"name": map[string]any{"type": "string"},
			},
		}
		result := buildValueFromSchema(schema, nil, 0)
		obj, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, true, obj["custom"])
	})

	t.Run("unknown type returns nil", func(t *testing.T) {
		schema := map[string]any{"type": "binary"}
		assert.Nil(t, buildValueFromSchema(schema, nil, 0))
	})
}

func TestBuildExampleFromSchema(t *testing.T) {
	t.Run("builds from media schema", func(t *testing.T) {
		media := map[string]any{
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string", "format": "uuid"},
				},
			},
		}
		result := buildExampleFromSchema(media, nil)
		obj, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "479e3145-dd38-476b-932c-529ceb705947", obj["id"])
	})

	t.Run("returns nil when no schema", func(t *testing.T) {
		assert.Nil(t, buildExampleFromSchema(map[string]any{}, nil))
	})
}

func TestEnrichOperation(t *testing.T) {
	t.Run("adds examples to path params", func(t *testing.T) {
		op := map[string]any{
			"parameters": []any{
				map[string]any{"name": "id", "in": "path", "schema": map[string]any{"type": "string"}},
			},
			"responses": map[string]any{
				"200": map[string]any{
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": map[string]any{"type": "object"},
						},
					},
				},
			},
		}

		enrichOperation(op, nil)

		params := op["parameters"].([]any)
		p := params[0].(map[string]any)
		examples, ok := p["examples"].(map[string]any)
		require.True(t, ok)
		assert.Contains(t, examples, "default")
	})

	t.Run("skips query params", func(t *testing.T) {
		op := map[string]any{
			"parameters": []any{
				map[string]any{"name": "limit", "in": "query", "schema": map[string]any{"type": "integer"}},
			},
			"responses": map[string]any{},
		}

		enrichOperation(op, nil)

		params := op["parameters"].([]any)
		p := params[0].(map[string]any)
		_, hasExamples := p["examples"]
		assert.False(t, hasExamples)
	})

	t.Run("skips $ref params", func(t *testing.T) {
		op := map[string]any{
			"parameters": []any{
				map[string]any{"$ref": "#/components/parameters/PaginationLimit"},
			},
			"responses": map[string]any{},
		}

		enrichOperation(op, nil)

		params := op["parameters"].([]any)
		p := params[0].(map[string]any)
		_, hasExamples := p["examples"]
		assert.False(t, hasExamples)
	})

	t.Run("does not overwrite existing examples", func(t *testing.T) {
		existing := map[string]any{"custom": map[string]any{"value": "keep-me"}}
		op := map[string]any{
			"parameters": []any{
				map[string]any{"name": "id", "in": "path", "examples": existing},
			},
			"responses": map[string]any{},
		}

		enrichOperation(op, nil)

		params := op["parameters"].([]any)
		p := params[0].(map[string]any)
		assert.Equal(t, existing, p["examples"])
	})

	t.Run("adds examples to response body", func(t *testing.T) {
		op := map[string]any{
			"responses": map[string]any{
				"200": map[string]any{
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"id": map[string]any{"type": "string"},
								},
							},
						},
					},
				},
			},
		}

		enrichOperation(op, nil)

		media := op["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)
		examples, ok := media["examples"].(map[string]any)
		require.True(t, ok)
		assert.Contains(t, examples, "default")
	})

	t.Run("adds examples to request body", func(t *testing.T) {
		op := map[string]any{
			"requestBody": map[string]any{
				"content": map[string]any{
					"application/json": map[string]any{
						"schema": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"name": map[string]any{"type": "string"},
							},
						},
					},
				},
			},
			"responses": map[string]any{},
		}

		enrichOperation(op, nil)

		media := op["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)
		examples, ok := media["examples"].(map[string]any)
		require.True(t, ok)
		assert.Contains(t, examples, "default")
	})
}

func TestEnrichSpec(t *testing.T) {
	t.Run("enriches a minimal spec", func(t *testing.T) {
		spec := map[string]any{
			"paths": map[string]any{
				"/items/{id}": map[string]any{
					"get": map[string]any{
						"parameters": []any{
							map[string]any{"name": "id", "in": "path", "schema": map[string]any{"type": "string"}},
						},
						"responses": map[string]any{
							"200": map[string]any{
								"content": map[string]any{
									"application/json": map[string]any{
										"schema": map[string]any{"type": "object"},
									},
								},
							},
						},
					},
				},
			},
		}

		enrichSpec(spec)

		op := spec["paths"].(map[string]any)["/items/{id}"].(map[string]any)["get"].(map[string]any)
		params := op["parameters"].([]any)
		p := params[0].(map[string]any)
		_, hasExamples := p["examples"]
		assert.True(t, hasExamples)
	})

	t.Run("skips non-operation keys", func(t *testing.T) {
		spec := map[string]any{
			"paths": map[string]any{
				"/items": map[string]any{
					"summary":     "Items endpoint",
					"description": "CRUD for items",
					"parameters":  []any{},
					"get": map[string]any{
						"responses": map[string]any{},
					},
				},
			},
		}

		// Should not panic.
		enrichSpec(spec)
	})

	t.Run("no-op on empty paths", func(t *testing.T) {
		spec := map[string]any{"paths": map[string]any{}}
		enrichSpec(spec)
	})

	t.Run("no-op on missing paths", func(t *testing.T) {
		spec := map[string]any{"info": map[string]any{"title": "Test"}}
		enrichSpec(spec)
	})
}

func TestContains(t *testing.T) {
	assert.True(t, contains("email_id", "id"))
	assert.True(t, contains("id", "id"))
	assert.True(t, contains("domain_id_extra", "id"))
	assert.False(t, contains("name", "id"))
	assert.False(t, contains("", "id"))
	assert.True(t, contains("abc", ""))
}
