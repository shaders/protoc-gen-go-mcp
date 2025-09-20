package runtime

import (
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

// Tool represents an MCP tool definition with essential fields
type Tool struct {
	Name        string
	Description string
	JSONSchema  string
}

// Option defines functional options for MCP functions
type Option func(*config)

// ExtraProperty defines an additional property to add to tool schemas
type ExtraProperty struct {
	Name        string
	Description string
	Required    bool
	ContextKey  interface{}
}

type config struct {
	ExtraProperties []ExtraProperty
}

// WithExtraProperties adds extra properties to tool schemas and extracts them from request arguments
func WithExtraProperties(properties ...ExtraProperty) Option {
	return func(c *config) {
		c.ExtraProperties = append(c.ExtraProperties, properties...)
	}
}

// NewConfig creates a new config instance
func NewConfig() *config {
	return &config{}
}

// AddExtraPropertiesToTool modifies a tool's schema to include additional properties
func AddExtraPropertiesToTool(tool mcp.Tool, properties []ExtraProperty) mcp.Tool {
	if len(properties) == 0 {
		return tool
	}

	// Parse the existing schema
	var schema map[string]interface{}
	if err := json.Unmarshal(tool.RawInputSchema, &schema); err != nil {
		// If we can't parse the schema, return the original tool
		return tool
	}

	// Add extra properties to schema
	var schemaProperties map[string]interface{}
	if props, ok := schema["properties"].(map[string]interface{}); ok {
		schemaProperties = props
	} else {
		schemaProperties = make(map[string]interface{})
		schema["properties"] = schemaProperties
	}

	// Get existing required fields
	var requiredFields []interface{}
	if req, ok := schema["required"].([]interface{}); ok {
		requiredFields = req
	}

	// Add each extra property
	for _, prop := range properties {
		// All extra properties are treated as strings by default
		propertyDef := map[string]interface{}{
			"type":        "string",
			"description": prop.Description,
		}

		schemaProperties[prop.Name] = propertyDef

		// Add to required fields if needed
		if prop.Required {
			requiredFields = append(requiredFields, prop.Name)
		}
	}

	// Update required array
	if len(requiredFields) > 0 {
		schema["required"] = requiredFields
	}

	// Marshal the modified schema back
	modifiedSchema, err := json.Marshal(schema)
	if err != nil {
		// If marshaling fails, return the original tool
		return tool
	}

	// Create a new tool with the modified schema
	modifiedTool := tool
	modifiedTool.RawInputSchema = json.RawMessage(modifiedSchema)
	return modifiedTool
}
