package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/atikulmunna/loom/internal/model"
)

// Parser converts a raw log line into a structured LogEntry.
type Parser interface {
	Parse(raw string, source string) model.LogEntry
}

// ---------------------------------------------------------------------------
// JSON Parser
// ---------------------------------------------------------------------------

// JSONParser handles JSON-formatted log lines.
// Recognizes common field names: level, msg/message, timestamp/time/ts.
type JSONParser struct{}

func NewJSONParser() *JSONParser { return &JSONParser{} }

func (p *JSONParser) Parse(raw string, source string) model.LogEntry {
	entry := base(raw, source)

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return entry // not valid JSON, return as-is
	}

	// Extract level.
	if v, ok := strField(data, "level", "severity"); ok {
		entry.Level = normalizeLevel(v)
	}

	// Extract message.
	if v, ok := strField(data, "message", "msg"); ok {
		entry.Message = v
	}

	// Extract timestamp.
	if v, ok := strField(data, "timestamp", "time", "ts"); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			entry.Timestamp = t
		}
	}

	// Store remaining fields.
	entry.Fields = make(map[string]string)
	skip := map[string]bool{"level": true, "severity": true, "message": true, "msg": true, "timestamp": true, "time": true, "ts": true}
	for k, v := range data {
		if !skip[k] {
			entry.Fields[k] = fmt.Sprintf("%v", v)
		}
	}

	return entry
}

// ---------------------------------------------------------------------------
// CLF Parser (Common Log Format)
// ---------------------------------------------------------------------------

// CLFParser handles Apache/Nginx Common Log Format lines.
// Format: host ident authuser [date] "request" status bytes
type CLFParser struct {
	re *regexp.Regexp
}

func NewCLFParser() *CLFParser {
	return &CLFParser{
		re: regexp.MustCompile(`^(\S+) (\S+) (\S+) \[([^\]]+)\] "([^"]*)" (\d{3}) (\S+)`),
	}
}

func (p *CLFParser) Parse(raw string, source string) model.LogEntry {
	entry := base(raw, source)

	matches := p.re.FindStringSubmatch(raw)
	if matches == nil {
		return entry
	}

	// Parse timestamp: 17/Feb/2026:12:00:00 +0000
	if t, err := time.Parse("02/Jan/2006:15:04:05 -0700", matches[4]); err == nil {
		entry.Timestamp = t
	}

	// Determine level from HTTP status code.
	status := matches[6]
	entry.Level = statusToLevel(status)
	entry.Message = matches[5] // the request line

	entry.Fields = map[string]string{
		"host":   matches[1],
		"ident":  matches[2],
		"user":   matches[3],
		"status": status,
		"bytes":  matches[7],
	}

	return entry
}

// statusToLevel maps HTTP status codes to log severity levels.
func statusToLevel(status string) string {
	if len(status) == 0 {
		return "INFO"
	}
	switch status[0] {
	case '5':
		return "ERROR"
	case '4':
		return "WARN"
	default:
		return "INFO"
	}
}

// ---------------------------------------------------------------------------
// Regex Parser (user-defined patterns)
// ---------------------------------------------------------------------------

// RegexParser uses a user-supplied regex with named capture groups.
// Recognized groups: timestamp, level, message (all optional).
type RegexParser struct {
	re *regexp.Regexp
}

func NewRegexParser(pattern string) (*RegexParser, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}
	return &RegexParser{re: re}, nil
}

func (p *RegexParser) Parse(raw string, source string) model.LogEntry {
	entry := base(raw, source)

	matches := p.re.FindStringSubmatch(raw)
	if matches == nil {
		return entry
	}

	names := p.re.SubexpNames()
	entry.Fields = make(map[string]string)

	for i, name := range names {
		if i == 0 || name == "" {
			continue
		}
		val := matches[i]
		entry.Fields[name] = val

		switch name {
		case "level":
			entry.Level = normalizeLevel(val)
		case "message":
			entry.Message = val
		case "timestamp":
			if t, err := time.Parse(time.RFC3339, val); err == nil {
				entry.Timestamp = t
			}
		}
	}

	return entry
}

// ---------------------------------------------------------------------------
// Auto Parser (format auto-detection)
// ---------------------------------------------------------------------------

// AutoParser tries parsers in order: JSON → CLF → keyword fallback.
type AutoParser struct {
	jsonParser *JSONParser
	clfParser  *CLFParser
}

func NewAutoParser() *AutoParser {
	return &AutoParser{
		jsonParser: NewJSONParser(),
		clfParser:  NewCLFParser(),
	}
}

func (p *AutoParser) Parse(raw string, source string) model.LogEntry {
	trimmed := strings.TrimSpace(raw)

	// Try JSON first.
	if len(trimmed) > 0 && trimmed[0] == '{' {
		entry := p.jsonParser.Parse(raw, source)
		if entry.Message != raw { // parsing extracted something
			return entry
		}
	}

	// Try CLF.
	entry := p.clfParser.Parse(raw, source)
	if entry.Message != raw { // parsing extracted something
		return entry
	}

	// Fallback: keyword-based level detection.
	return keywordParse(raw, source)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// base returns a LogEntry with defaults populated.
func base(raw, source string) model.LogEntry {
	return model.LogEntry{
		Timestamp: time.Now(),
		Source:    source,
		Raw:       raw,
		Level:    "INFO",
		Message:  raw,
	}
}

// keywordParse detects severity from keywords in the line.
func keywordParse(line, source string) model.LogEntry {
	entry := base(line, source)
	upper := strings.ToUpper(line)

	switch {
	case strings.Contains(upper, "FATAL"):
		entry.Level = "FATAL"
	case strings.Contains(upper, "ERROR"):
		entry.Level = "ERROR"
	case strings.Contains(upper, "WARN"):
		entry.Level = "WARN"
	case strings.Contains(upper, "DEBUG"):
		entry.Level = "DEBUG"
	}

	return entry
}

// normalizeLevel normalizes common level strings to a standard set.
func normalizeLevel(s string) string {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "FATAL", "CRITICAL", "CRIT":
		return "FATAL"
	case "ERROR", "ERR":
		return "ERROR"
	case "WARN", "WARNING":
		return "WARN"
	case "DEBUG", "TRACE":
		return "DEBUG"
	default:
		return "INFO"
	}
}

// strField returns the first matching string value from a map.
func strField(data map[string]interface{}, keys ...string) (string, bool) {
	for _, k := range keys {
		if v, ok := data[k]; ok {
			s := fmt.Sprintf("%v", v)
			if s != "" {
				return s, true
			}
		}
	}
	return "", false
}
