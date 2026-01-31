package analyzer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const statusFileName = "status.json"

// Status represents the current analysis status
type Status struct {
	Status string     `json:"status"`          // "idle" | "analyzing"
	Since  *time.Time `json:"since,omitempty"` // when analysis started
}

// SetStatus writes status to .memo/status.json
func SetStatus(memoDir string, status string) error {
	path := filepath.Join(memoDir, statusFileName)

	s := Status{Status: status}
	if status == "analyzing" {
		now := time.Now()
		s.Since = &now
	}

	data, err := json.Marshal(s)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// GetStatus reads status from .memo/status.json
// Returns "idle" if file doesn't exist or is invalid
func GetStatus(memoDir string) Status {
	path := filepath.Join(memoDir, statusFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		return Status{Status: "idle"}
	}

	var s Status
	if err := json.Unmarshal(data, &s); err != nil {
		return Status{Status: "idle"}
	}

	return s
}
