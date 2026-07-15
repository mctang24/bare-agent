package tools

import "testing"

func TestReadOnlyTools(t *testing.T) {
	registered := ReadOnlyTools()
	if len(registered) != 3 {
		t.Fatalf("ReadOnlyTools() length = %d, want 3", len(registered))
	}

	wantNames := map[string]bool{
		"list_files":  false,
		"read_file":   false,
		"search_text": false,
	}
	for _, tool := range registered {
		seen, exists := wantNames[tool.Name]
		if !exists {
			t.Errorf("ReadOnlyTools() contains unexpected tool %q", tool.Name)
			continue
		}
		if seen {
			t.Errorf("ReadOnlyTools() contains duplicate tool %q", tool.Name)
		}
		wantNames[tool.Name] = true
		if tool.Description == "" {
			t.Errorf("tool %q has empty description", tool.Name)
		}
		if tool.Parameters == nil {
			t.Errorf("tool %q has nil parameters", tool.Name)
		}
		if tool.Execute == nil {
			t.Errorf("tool %q has nil execute function", tool.Name)
		}
	}

	for name, seen := range wantNames {
		if !seen {
			t.Errorf("ReadOnlyTools() is missing tool %q", name)
		}
	}
}
