package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/atikulmunna/loom/internal/model"
)

// Renderer writes LogEntry values to an output stream.
type Renderer interface {
	Render(entry model.LogEntry) error
}

// ---------------------------------------------------------------------------
// Text Renderer (colorized terminal output)
// ---------------------------------------------------------------------------

var (
	styleInfo  = lipgloss.NewStyle().Foreground(lipgloss.Color("245")) // gray
	styleDebug = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Faint(true)
	styleWarn  = lipgloss.NewStyle().Foreground(lipgloss.Color("220")) // yellow
	styleError = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true) // red bold
	styleFatal = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("196")).
			Bold(true) // white on red
	styleSource = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Faint(true) // cyan
)

// TextRenderer prints logs to the terminal with severity-based colors.
type TextRenderer struct {
	w io.Writer
}

// NewTextRenderer returns a Renderer that writes colorized text to stdout.
func NewTextRenderer() *TextRenderer {
	return &TextRenderer{w: os.Stdout}
}

func (r *TextRenderer) Render(entry model.LogEntry) error {
	tag := styleLevelTag(entry.Level)
	src := styleSource.Render(entry.Source)
	ts := entry.Timestamp.Format("15:04:05")

	line := fmt.Sprintf("%s %s %s %s", ts, tag, src, entry.Message)
	_, err := fmt.Fprintln(r.w, line)
	return err
}

func styleLevelTag(level string) string {
	padded := fmt.Sprintf("%-5s", level)
	switch level {
	case "DEBUG":
		return styleDebug.Render(padded)
	case "WARN":
		return styleWarn.Render(padded)
	case "ERROR":
		return styleError.Render(padded)
	case "FATAL":
		return styleFatal.Render(padded)
	default:
		return styleInfo.Render(padded)
	}
}

// ---------------------------------------------------------------------------
// JSON Renderer (structured output for piping)
// ---------------------------------------------------------------------------

// JSONRenderer prints each log entry as a single JSON object per line.
type JSONRenderer struct {
	enc *json.Encoder
}

// NewJSONRenderer returns a Renderer that writes JSON lines to stdout.
func NewJSONRenderer() *JSONRenderer {
	return &JSONRenderer{enc: json.NewEncoder(os.Stdout)}
}

func (r *JSONRenderer) Render(entry model.LogEntry) error {
	return r.enc.Encode(entry)
}
