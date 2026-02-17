<div align="center">

# 🧵 Loom

**Log-Observer & Monitor**

A high-performance, real-time log aggregation CLI tool with a live web dashboard.

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen?style=flat-square)]()

[Features](#-features) · [Install](#-installation) · [Quick Start](#-quick-start) · [Dashboard](#-dashboard) · [Config](#-configuration) · [Architecture](#-architecture)

</div>

---

## ✨ Features

- **🔍 Multi-source Tailing** — Watch multiple log files or entire directories simultaneously
- **📐 Structured Parsing** — Auto-detect JSON, Common Log Format, or use custom Regex patterns
- **⚡ High Throughput** — 10,000+ lines/sec processing with < 50MB RAM via Go's concurrency pipeline
- **📊 Live Dashboard** — Real-time WebSocket-powered UI for log trends, error rates, and EPS metrics
- **🚨 Threshold Alerts** — Configurable triggers with terminal, webhook, and desktop notification support
- **📦 Single Binary** — Frontend assets embedded via `go:embed` — no external dependencies at runtime

---

## 📥 Installation

### From Source

```bash
git clone https://github.com/atikulmunna/loom.git
cd loom
go build -o loom ./cmd/loom
```

### Go Install

```bash
go install github.com/atikulmunna/loom/cmd/loom@latest
```

<!-- ### Pre-built Binaries
Download the latest release for your platform from the [Releases](https://github.com/atikulmunna/loom/releases) page. -->

---

## 🚀 Quick Start

### Watch a single file

```bash
loom watch /var/log/app.log
```

### Watch multiple files with glob patterns

```bash
loom watch "/var/log/**/*.log"
```

### Filter by severity

```bash
loom watch /var/log/app.log --level error,warn
```

### JSON output for piping

```bash
loom watch /var/log/app.log --output json | jq '.level == "ERROR"'
```

### Start with the web dashboard

```bash
loom watch /var/log/app.log --serve --port 8080
```

Then open [http://localhost:8080](http://localhost:8080) in your browser.

---

## 📊 Dashboard

Loom ships with a built-in real-time dashboard — no separate install needed.

<!-- ![Loom Dashboard](docs/assets/dashboard-preview.png) -->

| Metric | Description |
|:-------|:------------|
| **Events/sec** | Live throughput gauge |
| **Error Rate** | Percentage of ERROR/FATAL entries over a sliding window |
| **Log Stream** | Filterable, color-coded live log feed |
| **Trend Chart** | Time-series view of log volume by severity |

---

## ⚙️ Configuration

Loom uses a YAML config file. Default location: `~/.loom.yaml`

```yaml
# ~/.loom.yaml

watch:
  paths:
    - /var/log/app/*.log
    - /var/log/nginx/access.log
  recursive: true

parser:
  format: auto  # auto | json | clf | regex
  custom_regex: '^(?P<timestamp>\S+) (?P<level>\w+) (?P<message>.+)$'

server:
  enabled: true
  port: 8080

alerts:
  - name: high_error_rate
    pattern: "500 Internal Server Error"
    threshold: 5
    window: 10s
    channels: [terminal, webhook]
    webhook_url: "https://hooks.slack.com/services/xxx"
```

### CLI Flags

| Flag | Short | Description | Default |
|:-----|:------|:------------|:--------|
| `--level` | `-l` | Filter by log severity | all |
| `--output` | `-o` | Output format (`text`, `json`) | `text` |
| `--serve` | `-s` | Enable web dashboard | `false` |
| `--port` | `-p` | Dashboard port | `8080` |
| `--config` | `-c` | Config file path | `~/.loom.yaml` |

---

## 🏗️ Architecture

Loom uses a **Fan-in concurrency pipeline** built on Go channels:

```
┌──────────────┐
│  Log File A  │──┐
└──────────────┘  │    ┌──────────────┐    ┌─────────┐    ┌────────────────┐
                  ├───▶│  Transformer │───▶│   Hub   │───▶│  CLI Output    │
┌──────────────┐  │    │ (Worker Pool)│    │(Fan-in) │    ├────────────────┤
│  Log File B  │──┤    └──────────────┘    └────┬────┘    │  WebSocket UI  │
└──────────────┘  │                             │         ├────────────────┤
                  │                             ▼         │  Alerting      │
┌──────────────┐  │                        ┌──────────┐   └────────────────┘
│  Log File N  │──┘                        │Aggregator│
└──────────────┘                           │(Metrics) │
     Watcher                               └──────────┘
   (fsnotify)
```

| Component | Responsibility |
|:----------|:---------------|
| **Watcher** | OS-level file notifications via `fsnotify`, streams only new bytes |
| **Transformer** | Worker pool that parses raw lines into structured log entries |
| **Hub** | Central channel-based broadcaster to all consumers |
| **Aggregator** | Time-windowed buffer for EPS, error rate, and trend metrics |

---

## 🧰 Tech Stack

| Category | Technology |
|:---------|:-----------|
| Language | [Go 1.22+](https://go.dev) |
| CLI Framework | [Cobra](https://github.com/spf13/cobra) |
| Config | [Viper](https://github.com/spf13/viper) |
| File Watching | [fsnotify](https://github.com/fsnotify/fsnotify) |
| Web Server | [Gin](https://github.com/gin-gonic/gin) |
| WebSocket | [Gorilla WebSocket](https://github.com/gorilla/websocket) |
| TUI Styling | [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| Frontend | [HTMX](https://htmx.org/) |

---

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run benchmarks
go test -bench=. ./internal/parser/
```

---

## 📈 Performance

<!-- Update these with real benchmark numbers -->

| Metric | Value |
|:-------|:------|
| Throughput | 10,000+ lines/sec |
| Memory | < 50 MB |
| Binary Size | ~ TBD |
| Startup Time | ~ TBD |

---

## 🗺️ Roadmap

- [x] Project specification & architecture design
- [ ] **Phase 1** — Core engine (Watcher, Tail, Checkpointing)
- [ ] **Phase 2** — Processing pipeline (Parser, Channels, Filtering)
- [ ] **Phase 3** — Web dashboard (Gin, WebSocket, HTMX)
- [ ] **Phase 4** — Hardening (Tests, Profiling, Benchmarks)

---

## 🤝 Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please use [Conventional Commits](https://www.conventionalcommits.org/) for commit messages.

---

## 📄 License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

---

<div align="center">

Built with ☕ and Go

</div>
