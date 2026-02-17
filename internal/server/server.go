package server

import (
	"embed"
	"io/fs"
	"net/http"
	"net/http/pprof"

	"github.com/atikulmunna/loom/internal/aggregator"
	"github.com/atikulmunna/loom/internal/hub"
	"github.com/gin-gonic/gin"
)

//go:embed all:web
var webFS embed.FS

// Server holds the Gin engine and dependencies for the web dashboard.
type Server struct {
	engine     *gin.Engine
	hub        *hub.Hub
	aggregator *aggregator.Aggregator
	port       string
}

// New creates a web server for the Loom dashboard.
func New(h *hub.Hub, agg *aggregator.Aggregator, port string) *Server {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())

	// Disable automatic redirects that cause 301 issues.
	engine.RedirectTrailingSlash = false
	engine.RedirectFixedPath = false

	s := &Server{
		engine:     engine,
		hub:        h,
		aggregator: agg,
		port:       port,
	}

	s.setupRoutes()
	return s
}

// serveEmbedded reads a file from the embedded FS and writes it with the given content type.
func serveEmbedded(webContent fs.FS, name string, contentType string) gin.HandlerFunc {
	// Pre-read the file at startup so we don't read on every request.
	data, err := fs.ReadFile(webContent, name)
	return func(c *gin.Context) {
		if err != nil {
			c.String(http.StatusNotFound, "file not found: %s", name)
			return
		}
		c.Data(http.StatusOK, contentType, data)
	}
}

func (s *Server) setupRoutes() {
	// Extract the embedded web/ content.
	webContent, _ := fs.Sub(webFS, "web")

	// Dashboard — serve embedded files directly with correct content types.
	s.engine.GET("/", serveEmbedded(webContent, "index.html", "text/html; charset=utf-8"))
	s.engine.GET("/style.css", serveEmbedded(webContent, "style.css", "text/css; charset=utf-8"))
	s.engine.GET("/app.js", serveEmbedded(webContent, "app.js", "application/javascript; charset=utf-8"))

	// Health check.
	s.engine.GET("/healthz", func(c *gin.Context) {
		stats := s.aggregator.Snapshot()
		c.JSON(http.StatusOK, gin.H{
			"status":        "ok",
			"uptime":        stats.Uptime,
			"files_watched": stats.FilesWatched,
			"eps":           stats.EPS,
			"dropped_logs":  stats.DroppedLogs,
		})
	})

	// Metrics API.
	s.engine.GET("/api/stats", func(c *gin.Context) {
		c.JSON(http.StatusOK, s.aggregator.Snapshot())
	})

	// WebSocket.
	s.engine.GET("/ws", s.handleWebSocket)

	// pprof profiling endpoints.
	s.engine.GET("/debug/pprof/", gin.WrapF(pprof.Index))
	s.engine.GET("/debug/pprof/cmdline", gin.WrapF(pprof.Cmdline))
	s.engine.GET("/debug/pprof/profile", gin.WrapF(pprof.Profile))
	s.engine.GET("/debug/pprof/symbol", gin.WrapF(pprof.Symbol))
	s.engine.GET("/debug/pprof/trace", gin.WrapF(pprof.Trace))
	s.engine.GET("/debug/pprof/allocs", gin.WrapH(pprof.Handler("allocs")))
	s.engine.GET("/debug/pprof/heap", gin.WrapH(pprof.Handler("heap")))
	s.engine.GET("/debug/pprof/goroutine", gin.WrapH(pprof.Handler("goroutine")))
}

// Start runs the server. Blocks until the server is stopped.
func (s *Server) Start() error {
	return s.engine.Run(":" + s.port)
}
