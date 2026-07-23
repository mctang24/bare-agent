package tools

import (
	"strings"
	"testing"
)

func TestFileToolDefinitions(t *testing.T) {
	registered := NewFileTools().Definitions()
	if len(registered) != 5 {
		t.Fatalf("Tools() length = %d, want 5", len(registered))
	}
	wantNames := map[string]bool{
		"list_files": false, "read_file": false, "search_text": false,
		"edit_file": false, "write_file": false,
	}
	for _, tool := range registered {
		if _, ok := wantNames[tool.Name]; !ok {
			t.Fatalf("Tools() contains unexpected tool %q", tool.Name)
		}
		if wantNames[tool.Name] {
			t.Fatalf("Tools() contains duplicate tool %q", tool.Name)
		}
		wantNames[tool.Name] = true
		if tool.Description == "" || tool.Parameters == nil || tool.Execute == nil {
			t.Fatalf("tool %q has incomplete definition", tool.Name)
		}
		properties := tool.Parameters["properties"].(map[string]any)
		path := properties["path"].(map[string]any)
		description := path["description"].(string)
		if !strings.Contains(description, `Use "." for the root`) || !strings.Contains(description, "Do not use absolute paths") {
			t.Fatalf("tool %q path description does not require relative paths", tool.Name)
		}
	}
}
