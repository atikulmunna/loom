package output

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/atikulmunna/loom/internal/model"
)

func TestJSONRenderer(t *testing.T) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	renderer := &JSONRenderer{enc: enc}

	entry := model.LogEntry{
		Timestamp: time.Date(2026, 2, 17, 12, 0, 0, 0, time.UTC),
		Source:    "/var/log/app.log",
		Raw:       "2026-02-17 ERROR something broke",
		Level:    "ERROR",
		Message:  "something broke",
	}

	if err := renderer.Render(entry); err != nil {
		t.Fatal(err)
	}

	// Parse the output JSON.
	var got model.LogEntry
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, buf.String())
	}

	if got.Level != "ERROR" {
		t.Errorf("expected level ERROR, got %s", got.Level)
	}
	if got.Message != "something broke" {
		t.Errorf("expected message 'something broke', got %q", got.Message)
	}
	if got.Source != "/var/log/app.log" {
		t.Errorf("expected source '/var/log/app.log', got %q", got.Source)
	}
}
