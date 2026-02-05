package configfile

import (
	"encoding/json"
	"os"
	"testing"
)

func TestSchemaContainsExcludeCCToolchain(t *testing.T) {
	// Read the schema file
	schemaPath := "../../../.schema/devbox.schema.json"
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("Failed to read schema file: %v", err)
	}

	// Parse the schema
	var schema map[string]any
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("Failed to parse schema JSON: %v", err)
	}

	// Navigate to properties.shell
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Schema missing properties")
	}

	shellConfig, ok := properties["shell"].(map[string]any)
	if !ok {
		t.Fatal("Schema missing properties.shell")
	}

	shellProperties, ok := shellConfig["properties"].(map[string]any)
	if !ok {
		t.Fatal("Schema missing properties.shell.properties")
	}

	// Check that exclude_cc_toolchain exists
	excludeCCToolchain, ok := shellProperties["exclude_cc_toolchain"].(map[string]any)
	if !ok {
		t.Fatal("Schema missing exclude_cc_toolchain field in shell properties")
	}

	// Verify the field has correct type
	fieldType, ok := excludeCCToolchain["type"].(string)
	if !ok || fieldType != "boolean" {
		t.Errorf("exclude_cc_toolchain field has incorrect type: got %v, want 'boolean'", fieldType)
	}

	// Verify the field has a description
	description, ok := excludeCCToolchain["description"].(string)
	if !ok || description == "" {
		t.Error("exclude_cc_toolchain field missing description")
	}

	// Verify the field has a default value
	defaultValue, ok := excludeCCToolchain["default"].(bool)
	if !ok {
		t.Error("exclude_cc_toolchain field missing default value")
	}
	if defaultValue {
		t.Errorf("exclude_cc_toolchain default value should be false, got %v", defaultValue)
	}
}

func TestSchemaIsValidJSON(t *testing.T) {
	// Read the schema file
	schemaPath := "../../../.schema/devbox.schema.json"
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("Failed to read schema file: %v", err)
	}

	// Verify it's valid JSON
	var schema map[string]any
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("Schema is not valid JSON: %v", err)
	}

	// Verify it has required top-level fields
	requiredFields := []string{"$schema", "$id", "type", "properties"}
	for _, field := range requiredFields {
		if _, ok := schema[field]; !ok {
			t.Errorf("Schema missing required top-level field: %s", field)
		}
	}
}
