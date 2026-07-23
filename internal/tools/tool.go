package tools

import "context"

type Tool struct {
	Name        string
	Description string
	Parameters  Schema
	Execute     func(context.Context, string, string) (string, error)
}

type Schema struct {
	Type                 string            `json:"type"`
	Description          string            `json:"description,omitempty"`
	Properties           map[string]Schema `json:"properties,omitempty"`
	Required             []string          `json:"required,omitempty"`
	Items                *Schema           `json:"items,omitempty"`
	AdditionalProperties *bool             `json:"additionalProperties,omitempty"`
}

func ObjectSchema(properties map[string]Schema, required ...string) Schema {
	allowAdditional := false
	return Schema{
		Type:                 "object",
		Properties:           properties,
		Required:             required,
		AdditionalProperties: &allowAdditional,
	}
}

func StringSchema(description string) Schema {
	return Schema{Type: "string", Description: description}
}

func ArraySchema(items Schema, description string) Schema {
	return Schema{Type: "array", Description: description, Items: &items}
}
