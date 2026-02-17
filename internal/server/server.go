package server

import (
	"embed"
	"io/fs"
	"net/http"

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

	s := &Server{
		engine:     engine,
		hub:        h,
		aggregator: agg,
		port:       port,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Serve embedded static files.
	webContent, _ := fs.Sub(webFS, "web")
	s.engine.StaticFS("/static", http.FS(webContent))

	// Dashboard.
	s.engine.GET("/", func(c *gin.Context) {
		c.FileFromFS("index.html", http.FS(webContent))
	})

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
}

// Start runs the server. Blocks until the server is stopped.
func (s *Server) Start() error {
	return s.engine.Run(":" + s.port)
}
