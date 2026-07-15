package tools

import "context"

type Tool struct {
	Name        string
	Description string
	Parameters  map[string]any
	Execute     func(context.Context, string, string) (string, error)
}

// ReadOnlyTools returns the tools available in the read-only agent stage.
func ReadOnlyTools() []Tool {
	pathProperty := map[string]any{
		"type":        "string",
		"description": "Path relative to the agent working directory.",
	}

	return []Tool{
		{
			Name:        "list_files",
			Description: "List the direct children of a directory.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": pathProperty,
				},
				"required":             []string{"path"},
				"additionalProperties": false,
			},
			Execute: executeListFiles,
		},
		{
			Name:        "read_file",
			Description: "Read the contents of a file.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": pathProperty,
				},
				"required":             []string{"path"},
				"additionalProperties": false,
			},
			Execute: executeReadFile,
		},
		{
			Name:        "search_text",
			Description: "Search for exact text in a file or directory.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": pathProperty,
					"query": map[string]any{
						"type":        "string",
						"description": "Exact text to search for.",
					},
				},
				"required":             []string{"path", "query"},
				"additionalProperties": false,
			},
			Execute: executeSearchText,
		},
	}
}
