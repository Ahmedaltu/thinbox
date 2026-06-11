package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const stateDir = "/var/lib/thinbox/containers"

// ContainerState holds the runtime metadata for a single container.
// It is persisted as JSON at <stateDir>/<id>/state.json for the lifetime
// of the container process.
type ContainerState struct {
	ID        string    `json:"id"`
	PID       int       `json:"pid"`
	Image     string    `json:"image"`
	Command   []string  `json:"command"`
	StartedAt time.Time `json:"started_at"`
	Status    string    `json:"status"`
}

// Save writes s to disk. Called by Run() after the container process starts.
func Save(s *ContainerState) error {
	dir := filepath.Join(stateDir, s.ID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("state: mkdir: %w", err)
	}
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("state: marshal: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "state.json"), data, 0600)
}

// Delete removes the state directory for the given container ID.
// Called by Run() after the container process exits.
func Delete(id string) error {
	return os.RemoveAll(filepath.Join(stateDir, id))
}

// List reads all persisted container states from disk.
// Returns an empty slice (not an error) if no containers are running.
func List() ([]*ContainerState, error) {
	entries, err := os.ReadDir(stateDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("state: readdir: %w", err)
	}

	var states []*ContainerState
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(stateDir, e.Name(), "state.json"))
		if err != nil {
			continue // skip corrupt or incomplete entries
		}
		var s ContainerState
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}
		states = append(states, &s)
	}
	return states, nil
}
