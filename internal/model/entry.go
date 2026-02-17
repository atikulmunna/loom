package model

import "time"

// LogEntry represents a single parsed log line.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`  // originating file path
	Raw      string    `json:"raw"`     // original line text
	Level    string    `json:"level"`   // INFO, WARN, ERROR, FATAL
	Message  string    `json:"message"` // parsed message content
}
