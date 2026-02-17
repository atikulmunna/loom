package parser

import (
	"fmt"
	"testing"
)

// BenchmarkJSONParser measures JSON log parsing throughput.
func BenchmarkJSONParser(b *testing.B) {
	p := NewJSONParser()
	line := `{"level":"error","message":"disk full","timestamp":"2026-02-17T12:00:00Z","service":"api","request_id":"abc-123"}`

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		p.Parse(line, "bench.log")
	}
}

// BenchmarkCLFParser measures CLF log parsing throughput.
func BenchmarkCLFParser(b *testing.B) {
	p := NewCLFParser()
	line := `127.0.0.1 - frank [17/Feb/2026:12:00:00 +0000] "GET /api/health HTTP/1.1" 500 1234`

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		p.Parse(line, "bench.log")
	}
}

// BenchmarkRegexParser measures custom regex parsing throughput.
func BenchmarkRegexParser(b *testing.B) {
	p, _ := NewRegexParser(`^(?P<timestamp>\S+) (?P<level>\w+) (?P<message>.+)$`)
	line := "2026-02-17T12:00:00Z ERROR something failed badly"

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		p.Parse(line, "bench.log")
	}
}

// BenchmarkAutoParser measures auto-detection parsing throughput.
func BenchmarkAutoParser(b *testing.B) {
	p := NewAutoParser()
	lines := []string{
		`{"level":"error","message":"oom killed"}`,
		`127.0.0.1 - - [17/Feb/2026:12:00:00 +0000] "GET / HTTP/1.1" 200 5678`,
		`2026-02-17 WARN disk usage at 90%`,
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		p.Parse(lines[i%3], "bench.log")
	}
}

// BenchmarkParserThroughput measures sustained lines/sec over a large batch.
func BenchmarkParserThroughput(b *testing.B) {
	p := NewAutoParser()

	// Generate diverse log lines.
	lines := make([]string, 1000)
	for i := range lines {
		switch i % 4 {
		case 0:
			lines[i] = fmt.Sprintf(`{"level":"info","message":"request %d completed","latency_ms":42}`, i)
		case 1:
			lines[i] = fmt.Sprintf(`127.0.0.1 - - [17/Feb/2026:12:00:00 +0000] "GET /page/%d HTTP/1.1" 200 5678`, i)
		case 2:
			lines[i] = fmt.Sprintf("2026-02-17T12:00:00Z ERROR failed to process item %d", i)
		case 3:
			lines[i] = fmt.Sprintf("2026-02-17T12:00:00Z WARN slow query detected: %dms", i*10)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		p.Parse(lines[i%1000], "bench.log")
	}
}
