package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/atikulmunna/loom/internal/model"
	"github.com/atikulmunna/loom/internal/output"
	"github.com/atikulmunna/loom/internal/tailer"
	"github.com/atikulmunna/loom/internal/watcher"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch [paths...]",
	Short: "Watch log files for new entries",
	Long: `Watch one or more log files (or glob patterns) and stream new lines
to the terminal in real time. Supports colorized output and JSON mode.

Examples:
  loom watch /var/log/app.log
  loom watch "/var/log/**/*.log"
  loom watch app.log server.log --output json`,
	Args: cobra.MinimumNArgs(1),
	RunE: runWatch,
}

func init() {
	rootCmd.AddCommand(watchCmd)
}

func runWatch(cmd *cobra.Command, args []string) error {
	// --- Set up context with graceful shutdown ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\n🧵 Loom shutting down gracefully...")
		cancel()
	}()

	// --- Initialize watcher ---
	w, err := watcher.New(args)
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	watchedPaths := w.Paths()
	if len(watchedPaths) == 0 {
		return fmt.Errorf("no files matched the given patterns: %v", args)
	}

	fmt.Fprintf(os.Stderr, "🧵 Loom watching %d file(s):\n", len(watchedPaths))
	for _, p := range watchedPaths {
		fmt.Fprintf(os.Stderr, "   • %s\n", p)
	}
	fmt.Fprintln(os.Stderr)

	// --- Initialize checkpoint ---
	ckptPath := filepath.Join(".", ".loom-state.json")
	ckpt, err := tailer.NewCheckpoint(ckptPath)
	if err != nil {
		return fmt.Errorf("failed to load checkpoint: %w", err)
	}

	// --- Initialize tailer ---
	t := tailer.New(w, ckpt)

	// --- Choose renderer ---
	var renderer output.Renderer
	switch strings.ToLower(outputFmt) {
	case "json":
		renderer = output.NewJSONRenderer()
	default:
		renderer = output.NewTextRenderer()
	}

	// --- Build level filter set ---
	levelSet := make(map[string]bool)
	if levelFilter != "" {
		for _, l := range strings.Split(levelFilter, ",") {
			levelSet[strings.ToUpper(strings.TrimSpace(l))] = true
		}
	}

	// --- Start pipeline ---
	go w.Start(ctx)
	go t.Start(ctx)

	// --- Render output ---
	for entry := range t.Lines() {
		if shouldShow(entry, levelSet) {
			if err := renderer.Render(entry); err != nil {
				log.Printf("render error: %v", err)
			}
		}
	}

	return nil
}

// shouldShow returns true if the entry passes the level filter.
func shouldShow(entry model.LogEntry, levelSet map[string]bool) bool {
	if len(levelSet) == 0 {
		return true // no filter = show all
	}
	return levelSet[entry.Level]
}
