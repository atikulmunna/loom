package tailer

import (
	"encoding/json"
	"os"
	"sync"
)

// checkpointData is the on-disk JSON structure for persisted offsets.
type checkpointData struct {
	Offsets map[string]int64 `json:"offsets"`
}

// Checkpoint persists file read offsets so tailing can resume after a restart.
type Checkpoint struct {
	mu   sync.RWMutex
	path string
	data checkpointData
}

// NewCheckpoint creates or loads a checkpoint file at the given path.
func NewCheckpoint(path string) (*Checkpoint, error) {
	c := &Checkpoint{
		path: path,
		data: checkpointData{Offsets: make(map[string]int64)},
	}

	// Try to load existing checkpoint.
	raw, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(raw, &c.data)
	}
	if c.data.Offsets == nil {
		c.data.Offsets = make(map[string]int64)
	}

	return c, nil
}

// Get returns the saved offset for a file path.
func (c *Checkpoint) Get(path string) (int64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.data.Offsets[path]
	return v, ok
}

// Set records the current offset for a file path.
func (c *Checkpoint) Set(path string, offset int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data.Offsets[path] = offset
}

// Save writes the checkpoint data to disk atomically.
func (c *Checkpoint) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	raw, err := json.MarshalIndent(c.data, "", "  ")
	if err != nil {
		return err
	}

	// Write to a temp file first, then rename for atomicity.
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, c.path)
}
