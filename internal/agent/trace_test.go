package agent

import (
	"encoding/hex"
	"testing"
)

func TestNewTraceID(t *testing.T) {
	first, err := newTraceID("session")
	if err != nil {
		t.Fatalf("newTraceID() error = %v", err)
	}
	second, err := newTraceID("run")
	if err != nil {
		t.Fatalf("newTraceID() error = %v", err)
	}
	if len(first) != 40 || first[:8] != "session_" || len(second) != 36 || second[:4] != "run_" {
		t.Fatalf("trace IDs = %q, %q", first, second)
	}
	if _, err := hex.DecodeString(first[8:]); err != nil {
		t.Fatalf("trace ID %q is not hexadecimal: %v", first, err)
	}
	if _, err := newTraceID(""); err == nil {
		t.Fatal("newTraceID() error = nil, want empty prefix error")
	}
}
