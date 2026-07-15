package main

import (
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	parsed, err := parseArgs([]string{"-root", "project", "inspect", "the", "code"})
	if err != nil {
		t.Fatalf("parseArgs() error = %v", err)
	}
	if parsed.root != "project" || parsed.task != "inspect the code" {
		t.Fatalf("parseArgs() = %#v", parsed)
	}
}

func TestParseArgsWithoutTask(t *testing.T) {
	parsed, err := parseArgs(nil)
	if err != nil {
		t.Fatalf("parseArgs() error = %v", err)
	}
	if parsed.root != "." || parsed.task != "" {
		t.Fatalf("parseArgs() = %#v", parsed)
	}
}

func TestParseArgsErrors(t *testing.T) {
	for _, args := range [][]string{{"-unknown"}} {
		if _, err := parseArgs(args); err == nil || !strings.Contains(err.Error(), "parse arguments") {
			t.Fatalf("parseArgs(%q) error = %v", args, err)
		}
	}
}
