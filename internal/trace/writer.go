package trace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

type Event struct {
	Timestamp time.Time `json:"timestamp"`
	RunID     string    `json:"runId"`
	Type      string    `json:"type"`
	Turn      int       `json:"turn,omitempty"`
	Data      any       `json:"data,omitempty"`
}

type Writer struct {
	Path string
}

// Append appends one event as a JSON line to the writer's path.
func (w Writer) Append(event Event) error {
	if w.Path == "" {
		return errors.New("append trace event: path is empty")
	}
	if event.RunID == "" {
		return errors.New("append trace event: run ID is empty")
	}
	if event.Type == "" {
		return errors.New("append trace event: type is empty")
	}

	encoded, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("append trace event: encode event: %w", err)
	}
	encoded = append(encoded, '\n')

	file, err := os.OpenFile(w.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("append trace event: open file: %w", err)
	}
	if _, err := file.Write(encoded); err != nil {
		_ = file.Close()
		return fmt.Errorf("append trace event: write file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("append trace event: close file: %w", err)
	}

	return nil
}
