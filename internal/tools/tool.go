package tools

import "context"

type Tool struct {
	Name        string
	Description string
	Parameters  map[string]any
	Execute     func(context.Context, string, string) (string, error)
}
