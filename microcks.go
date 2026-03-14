package resend

// enrichSpec walks all paths/operations in the OpenAPI spec and injects
// Microcks-compatible named examples into parameters and responses.
func enrichSpec(spec map[string]any) {
	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		return
	}

	// Resolve component schemas for generating example values.
	schemas := resolveSchemas(spec)

	for _, pathItem := range paths {
		methods, ok := pathItem.(map[string]any)
		if !ok {
			continue
		}

		for method, opAny := range methods {
			if method == "parameters" || method == "summary" || method == "description" {
				continue
			}
			op, ok := opAny.(map[string]any)
			if !ok {
				continue
			}
			enrichOperation(op, schemas)
		}
	}
}

// enrichOperation adds a named example ("default") to path/query parameters
// and response bodies so Microcks can create mock pairs.
func enrichOperation(op map[string]any, schemas map[string]any) {
	const exampleName = "default"

	// Add examples to path parameters only.
	// Query parameters are intentionally left without examples to avoid triggering
	// Microcks' URI_PARAMS/URI_ELEMENTS dispatchers, which require exact query
	// parameter matching in requests.
	if params, ok := op["parameters"].([]any); ok {
		for _, pAny := range params {
			p, ok := pAny.(map[string]any)
			if !ok {
				continue
			}

			// Skip $ref parameters and header parameters.
			if _, hasRef := p["$ref"]; hasRef {
				continue
			}
			in, _ := p["in"].(string)

			if in == "path" {
				if _, hasExamples := p["examples"]; !hasExamples {
					pName, _ := p["name"].(string)
					value := paramExampleValue(pName, p)
					p["examples"] = map[string]any{
						exampleName: map[string]any{
							"value": value,
						},
					}
				}
			}
		}
	}

	// Add examples to request body.
	if reqBody, ok := op["requestBody"].(map[string]any); ok {
		if content, ok := reqBody["content"].(map[string]any); ok {
			for _, mediaAny := range content {
				media, ok := mediaAny.(map[string]any)
				if !ok {
					continue
				}
				if _, hasExamples := media["examples"]; !hasExamples {
					exValue := buildExampleFromSchema(media, schemas)
					if exValue != nil {
						media["examples"] = map[string]any{
							exampleName: map[string]any{
								"value": exValue,
							},
						}
					}
				}
			}
		}
	}

	// Add examples to responses.
	responses, ok := op["responses"].(map[string]any)
	if !ok {
		return
	}
	for _, respAny := range responses {
		resp, ok := respAny.(map[string]any)
		if !ok {
			continue
		}
		contentMap, ok := resp["content"].(map[string]any)
		if !ok {
			continue
		}
		for _, mediaAny := range contentMap {
			media, ok := mediaAny.(map[string]any)
			if !ok {
				continue
			}
			if _, hasExamples := media["examples"]; !hasExamples {
				exValue := buildExampleFromSchema(media, schemas)
				if exValue != nil {
					media["examples"] = map[string]any{
						exampleName: map[string]any{
							"value": exValue,
						},
					}
				}
			}
		}
	}
}

// paramExampleValue returns a sensible example value for a path or query parameter.
func paramExampleValue(name string, p map[string]any) any {
	// Use the existing example if present.
	if ex, ok := p["example"]; ok {
		return ex
	}

	schema, _ := p["schema"].(map[string]any)
	if schema != nil {
		if ex, ok := schema["example"]; ok {
			return ex
		}
	}

	// Generate a sensible default based on the parameter name.
	switch {
	case contains(name, "id"):
		return "479e3145-dd38-476b-932c-529ceb705947"
	case name == "limit":
		return 10
	case name == "after" || name == "before":
		return ""
	default:
		return "example"
	}
}

// buildExampleFromSchema builds an example value from a media type's schema,
// resolving $ref pointers and using property-level examples.
func buildExampleFromSchema(media map[string]any, schemas map[string]any) any {
	schema, ok := media["schema"].(map[string]any)
	if !ok {
		return nil
	}
	return buildValueFromSchema(schema, schemas, 0)
}

// buildValueFromSchema recursively builds an example value from a schema definition.
func buildValueFromSchema(schema map[string]any, schemas map[string]any, depth int) any {
	if depth > 5 {
		return nil
	}

	// Resolve $ref.
	if ref, ok := schema["$ref"].(string); ok {
		resolved := resolveRef(ref, schemas)
		if resolved == nil {
			return nil
		}
		return buildValueFromSchema(resolved, schemas, depth+1)
	}

	// Use schema-level example if present.
	if ex, ok := schema["example"]; ok {
		return ex
	}

	typ, _ := schema["type"].(string)

	switch typ {
	case "object", "":
		props, ok := schema["properties"].(map[string]any)
		if !ok {
			return map[string]any{}
		}
		obj := make(map[string]any)
		for propName, propAny := range props {
			prop, ok := propAny.(map[string]any)
			if !ok {
				continue
			}
			obj[propName] = buildValueFromSchema(prop, schemas, depth+1)
		}
		return obj

	case "array":
		items, ok := schema["items"].(map[string]any)
		if !ok {
			return []any{}
		}
		item := buildValueFromSchema(items, schemas, depth+1)
		if item != nil {
			return []any{item}
		}
		return []any{}

	case "string":
		if ex, ok := schema["example"]; ok {
			return ex
		}
		format, _ := schema["format"].(string)
		switch format {
		case "uuid":
			return "479e3145-dd38-476b-932c-529ceb705947"
		case "date-time":
			return "2023-10-06T23:47:56.678Z"
		default:
			return "example"
		}

	case "integer":
		if ex, ok := schema["example"]; ok {
			return ex
		}
		return 1

	case "number":
		if ex, ok := schema["example"]; ok {
			return ex
		}
		return 1.0

	case "boolean":
		if ex, ok := schema["example"]; ok {
			return ex
		}
		return false

	default:
		return nil
	}
}

// resolveSchemas extracts the components/schemas map from the spec.
func resolveSchemas(spec map[string]any) map[string]any {
	components, ok := spec["components"].(map[string]any)
	if !ok {
		return nil
	}
	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		return nil
	}
	return schemas
}

// resolveRef resolves a $ref string like "#/components/schemas/Foo" to its schema map.
func resolveRef(ref string, schemas map[string]any) map[string]any {
	// Only handle local refs: #/components/schemas/Name
	if len(ref) < 2 || ref[0] != '#' {
		return nil
	}

	parts := splitRef(ref)
	if len(parts) != 4 || parts[1] != "components" || parts[2] != "schemas" {
		return nil
	}

	schema, ok := schemas[parts[3]].(map[string]any)
	if !ok {
		return nil
	}
	return schema
}

// splitRef splits "#/components/schemas/Foo" into ["#", "components", "schemas", "Foo"].
func splitRef(ref string) []string {
	var parts []string
	current := ""
	for _, c := range ref {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
