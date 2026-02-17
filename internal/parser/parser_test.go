package parser

import (
	"testing"
)

func TestJSONParser(t *testing.T) {
	p := NewJSONParser()

	entry := p.Parse(`{"level":"error","message":"disk full","timestamp":"2026-02-17T12:00:00Z"}`, "/var/log/app.log")

	if entry.Level != "ERROR" {
		t.Errorf("expected level ERROR, got %s", entry.Level)
	}
	if entry.Message != "disk full" {
		t.Errorf("expected message 'disk full', got %q", entry.Message)
	}
	if entry.Source != "/var/log/app.log" {
		t.Errorf("expected source '/var/log/app.log', got %q", entry.Source)
	}
	if entry.Timestamp.Year() != 2026 {
		t.Errorf("expected year 2026, got %d", entry.Timestamp.Year())
	}
}

func TestJSONParserAltFields(t *testing.T) {
	p := NewJSONParser()

	// Test with "severity" + "msg" instead of "level" + "message".
	entry := p.Parse(`{"severity":"warning","msg":"high latency","ts":"2026-02-17T12:00:00Z"}`, "app.log")

	if entry.Level != "WARN" {
		t.Errorf("expected level WARN, got %s", entry.Level)
	}
	if entry.Message != "high latency" {
		t.Errorf("expected message 'high latency', got %q", entry.Message)
	}
}

func TestJSONParserInvalidJSON(t *testing.T) {
	p := NewJSONParser()

	entry := p.Parse("not json at all", "test.log")

	// Should return raw line as-is.
	if entry.Message != "not json at all" {
		t.Errorf("expected raw line as message, got %q", entry.Message)
	}
	if entry.Level != "INFO" {
		t.Errorf("expected default level INFO, got %s", entry.Level)
	}
}

func TestCLFParser(t *testing.T) {
	p := NewCLFParser()

	line := `127.0.0.1 - frank [17/Feb/2026:12:00:00 +0000] "GET /api/health HTTP/1.1" 500 1234`
	entry := p.Parse(line, "access.log")

	if entry.Level != "ERROR" {
		t.Errorf("expected level ERROR for status 500, got %s", entry.Level)
	}
	if entry.Message != "GET /api/health HTTP/1.1" {
		t.Errorf("expected request as message, got %q", entry.Message)
	}
	if entry.Fields["status"] != "500" {
		t.Errorf("expected status 500, got %q", entry.Fields["status"])
	}
	if entry.Fields["host"] != "127.0.0.1" {
		t.Errorf("expected host 127.0.0.1, got %q", entry.Fields["host"])
	}
	if entry.Fields["user"] != "frank" {
		t.Errorf("expected user frank, got %q", entry.Fields["user"])
	}
}

func TestCLFParser200(t *testing.T) {
	p := NewCLFParser()

	line := `192.168.1.1 - - [17/Feb/2026:12:00:00 +0000] "GET / HTTP/1.1" 200 5678`
	entry := p.Parse(line, "access.log")

	if entry.Level != "INFO" {
		t.Errorf("expected INFO for status 200, got %s", entry.Level)
	}
}

func TestCLFParser404(t *testing.T) {
	p := NewCLFParser()

	line := `10.0.0.1 - - [17/Feb/2026:12:00:00 +0000] "GET /missing HTTP/1.1" 404 0`
	entry := p.Parse(line, "access.log")

	if entry.Level != "WARN" {
		t.Errorf("expected WARN for status 404, got %s", entry.Level)
	}
}

func TestRegexParser(t *testing.T) {
	p, err := NewRegexParser(`^(?P<timestamp>\S+) (?P<level>\w+) (?P<message>.+)$`)
	if err != nil {
		t.Fatal(err)
	}

	entry := p.Parse("2026-02-17T12:00:00Z ERROR something failed badly", "app.log")

	if entry.Level != "ERROR" {
		t.Errorf("expected level ERROR, got %s", entry.Level)
	}
	if entry.Message != "something failed badly" {
		t.Errorf("expected message 'something failed badly', got %q", entry.Message)
	}
}

func TestRegexParserInvalidPattern(t *testing.T) {
	_, err := NewRegexParser(`[invalid`)
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestAutoParserJSON(t *testing.T) {
	p := NewAutoParser()

	entry := p.Parse(`{"level":"error","message":"oom killed"}`, "app.log")

	if entry.Level != "ERROR" {
		t.Errorf("expected ERROR, got %s", entry.Level)
	}
	if entry.Message != "oom killed" {
		t.Errorf("expected 'oom killed', got %q", entry.Message)
	}
}

func TestAutoParserCLF(t *testing.T) {
	p := NewAutoParser()

	entry := p.Parse(`10.0.0.1 - - [17/Feb/2026:12:00:00 +0000] "POST /data HTTP/1.1" 503 0`, "access.log")

	if entry.Level != "ERROR" {
		t.Errorf("expected ERROR for 503, got %s", entry.Level)
	}
	if entry.Message != "POST /data HTTP/1.1" {
		t.Errorf("expected request as message, got %q", entry.Message)
	}
}

func TestAutoParserPlainText(t *testing.T) {
	p := NewAutoParser()

	entry := p.Parse("2026-02-17 WARN disk usage at 90%", "sys.log")

	if entry.Level != "WARN" {
		t.Errorf("expected WARN via keyword detection, got %s", entry.Level)
	}
}
