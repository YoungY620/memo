package analyzer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/YoungY620/memo/internal"
	"github.com/fsnotify/fsnotify"
)

// EntireConfig describes the detected Entire.io configuration
type EntireConfig struct {
	Enabled  bool
	Strategy string // e.g. "manual-commit"
	Branch   string // e.g. "entire/checkpoints/v1"
}

// CheckpointData holds the data extracted from a single checkpoint
type CheckpointData struct {
	CommitSHA    string
	CheckpointID string            // extracted from directory name (2-char prefix + remaining)
	Files        map[string]string // path → content on the checkpoint tree
}

// entireState is persisted to .memo/entire_state.json
type entireState struct {
	LastCommit string `json:"last_commit"`
	UpdatedAt  string `json:"updated_at"`
}

// CheckpointMonitor watches the entire/checkpoints/v1 ref for new commits
type CheckpointMonitor struct {
	workDir    string
	branch     string
	lastCommit string
	onChange   func([]CheckpointData)
	watcher    *fsnotify.Watcher
	debounceMs int
	mu         sync.Mutex
	debounce   *time.Timer
	memoDir    string
	pollTicker *time.Ticker // fallback polling when ref file is packed
}

// DetectEntire checks if Entire.io is configured in the given work directory.
// Returns nil if Entire.io is not detected.
func DetectEntire(workDir string) (*EntireConfig, error) {
	// Check for .entire/settings.json
	settingsPath := filepath.Join(workDir, ".entire", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read entire settings: %w", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse entire settings: %w", err)
	}

	branch := "entire/checkpoints/v1"

	// Verify branch exists
	cmd := exec.Command("git", "-C", workDir, "rev-parse", "--verify", branch)
	if err := cmd.Run(); err != nil {
		internal.LogDebug("Entire.io settings found but branch %s does not exist", branch)
		return nil, nil
	}

	strategy := "manual-commit"
	if s, ok := settings["strategy"].(string); ok {
		strategy = s
	}

	return &EntireConfig{
		Enabled:  true,
		Strategy: strategy,
		Branch:   branch,
	}, nil
}

// NewCheckpointMonitor creates a monitor for Entire.io checkpoint branch changes
func NewCheckpointMonitor(workDir string, debounceMs int, onChange func([]CheckpointData)) (*CheckpointMonitor, error) {
	branch := "entire/checkpoints/v1"
	memoDir := filepath.Join(workDir, ".memo")

	m := &CheckpointMonitor{
		workDir:    workDir,
		branch:     branch,
		onChange:   onChange,
		debounceMs: debounceMs,
		memoDir:    memoDir,
	}

	m.loadState()

	return m, nil
}

// Run starts the monitoring loop. It watches the git ref file if available,
// otherwise falls back to polling every 30 seconds.
func (m *CheckpointMonitor) Run() error {
	refFile := filepath.Join(m.workDir, ".git", "refs", "heads", m.branch)

	if _, err := os.Stat(refFile); err == nil {
		return m.runFsnotify(refFile)
	}

	internal.LogInfo("Checkpoint ref file not found (packed refs?), falling back to polling")
	return m.runPolling()
}

func (m *CheckpointMonitor) runFsnotify(refFile string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}
	m.watcher = watcher

	// Watch the parent directory of the ref file (fsnotify needs the directory for file events)
	refDir := filepath.Dir(refFile)
	if err := os.MkdirAll(refDir, 0755); err != nil {
		watcher.Close()
		return fmt.Errorf("failed to ensure ref directory: %w", err)
	}
	if err := watcher.Add(refDir); err != nil {
		watcher.Close()
		return fmt.Errorf("failed to watch ref directory %s: %w", refDir, err)
	}

	internal.LogInfo("Checkpoint monitor started, watching ref file: %s", refFile)
	refBase := filepath.Base(refFile)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if filepath.Base(event.Name) != refBase {
				continue
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}
			internal.LogDebug("Checkpoint ref changed: %s %s", event.Op, event.Name)
			m.scheduleCheck()

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			if err != nil {
				internal.LogError("Checkpoint watcher error: %v", err)
			}
		}
	}
}

func (m *CheckpointMonitor) runPolling() error {
	m.pollTicker = time.NewTicker(30 * time.Second)
	defer m.pollTicker.Stop()

	for range m.pollTicker.C {
		newSHA, err := m.getHeadCommit()
		if err != nil {
			internal.LogDebug("Poll: failed to get head commit: %v", err)
			continue
		}
		if newSHA != m.lastCommit {
			internal.LogDebug("Poll: detected new commit %s (was %s)", newSHA, m.lastCommit)
			m.doCheck()
		}
	}
	return nil
}

func (m *CheckpointMonitor) scheduleCheck() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.debounce != nil {
		m.debounce.Stop()
	}
	m.debounce = time.AfterFunc(time.Duration(m.debounceMs)*time.Millisecond, m.doCheck)
}

func (m *CheckpointMonitor) doCheck() {
	newSHA, err := m.getHeadCommit()
	if err != nil {
		internal.LogError("Failed to get checkpoint head: %v", err)
		return
	}

	if newSHA == m.lastCommit {
		return
	}

	internal.LogInfo("Checkpoint branch updated: %s → %s", m.lastCommit, newSHA)

	var checkpoints []CheckpointData
	if m.lastCommit == "" {
		// First run: extract all checkpoints from the tree
		checkpoints, err = m.extractAll(newSHA)
	} else {
		checkpoints, err = m.extractSince(m.lastCommit, newSHA)
	}
	if err != nil {
		internal.LogError("Failed to extract checkpoints: %v", err)
		return
	}

	// Filter by current branch
	checkpoints = m.filterByBranch(checkpoints)

	if len(checkpoints) > 0 {
		internal.LogInfo("Extracted %d checkpoint(s) for current branch", len(checkpoints))
		if m.onChange != nil {
			m.onChange(checkpoints)
		}
	} else {
		internal.LogDebug("No new checkpoints for current branch")
	}

	m.lastCommit = newSHA
	m.saveState()
}

// ScanAll performs a one-shot extraction of all checkpoints relevant to the current branch
func (m *CheckpointMonitor) ScanAll() ([]CheckpointData, error) {
	headSHA, err := m.getHeadCommit()
	if err != nil {
		return nil, fmt.Errorf("failed to get checkpoint head: %w", err)
	}

	all, err := m.extractAll(headSHA)
	if err != nil {
		return nil, err
	}

	filtered := m.filterByBranch(all)

	m.lastCommit = headSHA
	m.saveState()

	return filtered, nil
}

// Close stops the checkpoint monitor
func (m *CheckpointMonitor) Close() error {
	m.mu.Lock()
	if m.debounce != nil {
		m.debounce.Stop()
	}
	m.mu.Unlock()

	if m.pollTicker != nil {
		m.pollTicker.Stop()
	}
	if m.watcher != nil {
		return m.watcher.Close()
	}
	return nil
}

// getHeadCommit returns the current HEAD SHA of the checkpoint branch
func (m *CheckpointMonitor) getHeadCommit() (string, error) {
	cmd := exec.Command("git", "-C", m.workDir, "rev-parse", m.branch)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse %s failed: %w", m.branch, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// extractSince extracts checkpoint data from commits between oldSHA and newSHA
func (m *CheckpointMonitor) extractSince(oldSHA, newSHA string) ([]CheckpointData, error) {
	// Get list of new commits
	cmd := exec.Command("git", "-C", m.workDir, "log", "--format=%H", oldSHA+".."+newSHA)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	var commits []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			commits = append(commits, line)
		}
	}

	if len(commits) == 0 {
		return nil, nil
	}

	internal.LogDebug("Found %d new commit(s) on checkpoint branch", len(commits))

	var all []CheckpointData
	for _, commit := range commits {
		cps, err := m.extractFromCommit(commit)
		if err != nil {
			internal.LogError("Failed to extract from commit %s: %v", commit, err)
			continue
		}
		all = append(all, cps...)
	}

	return all, nil
}

// extractAll extracts all checkpoint data from the tree at the given commit
func (m *CheckpointMonitor) extractAll(commitSHA string) ([]CheckpointData, error) {
	return m.extractFromCommit(commitSHA)
}

// extractFromCommit extracts checkpoint data from a single commit's tree
func (m *CheckpointMonitor) extractFromCommit(commitSHA string) ([]CheckpointData, error) {
	// List all files in the tree
	cmd := exec.Command("git", "-C", m.workDir, "ls-tree", "-r", "--name-only", commitSHA)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-tree failed for %s: %w", commitSHA, err)
	}

	// Group files by checkpoint ID (directory structure: <2-char>/<remaining-id>/...)
	type cpFiles struct {
		id    string
		files map[string]string // path → empty (content loaded lazily)
	}
	checkpointMap := make(map[string]*cpFiles)

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Extract checkpoint ID from path: first two components form the ID
		// e.g., "ab/cdef123456/metadata.json" → checkpoint ID = "abcdef123456"
		parts := strings.SplitN(line, "/", 3)
		if len(parts) < 2 {
			continue
		}

		cpID := parts[0] + parts[1]
		if _, ok := checkpointMap[cpID]; !ok {
			checkpointMap[cpID] = &cpFiles{
				id:    cpID,
				files: make(map[string]string),
			}
		}
		checkpointMap[cpID].files[line] = ""
	}

	// Load file contents
	var result []CheckpointData
	for _, cp := range checkpointMap {
		files := make(map[string]string)
		for path := range cp.files {
			content, err := m.gitShow(commitSHA, path)
			if err != nil {
				internal.LogDebug("Failed to read %s:%s: %v", commitSHA[:8], path, err)
				continue
			}
			files[path] = content
		}

		result = append(result, CheckpointData{
			CommitSHA:    commitSHA,
			CheckpointID: cp.id,
			Files:        files,
		})
	}

	return result, nil
}

func (m *CheckpointMonitor) gitShow(commitSHA, path string) (string, error) {
	cmd := exec.Command("git", "-C", m.workDir, "show", commitSHA+":"+path)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// getCurrentBranchCheckpoints scans the current branch's commit messages for
// Entire-Checkpoint trailers and returns the set of checkpoint IDs.
func (m *CheckpointMonitor) getCurrentBranchCheckpoints() (map[string]bool, error) {
	cmd := exec.Command("git", "-C", m.workDir, "log", "--format=%B", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	ids := make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Entire-Checkpoint:") {
			id := strings.TrimSpace(strings.TrimPrefix(line, "Entire-Checkpoint:"))
			if id != "" {
				ids[id] = true
			}
		}
	}

	internal.LogDebug("Found %d Entire-Checkpoint trailers on current branch", len(ids))
	return ids, nil
}

// filterByBranch keeps only checkpoints whose ID matches a trailer on the current branch
func (m *CheckpointMonitor) filterByBranch(all []CheckpointData) []CheckpointData {
	branchCPs, err := m.getCurrentBranchCheckpoints()
	if err != nil {
		internal.LogError("Failed to get current branch checkpoints: %v", err)
		// If we can't determine the branch, return all (best effort)
		return all
	}

	if len(branchCPs) == 0 {
		internal.LogDebug("No Entire-Checkpoint trailers found on current branch")
		return nil
	}

	var filtered []CheckpointData
	for _, cp := range all {
		if branchCPs[cp.CheckpointID] {
			filtered = append(filtered, cp)
		}
	}
	return filtered
}

// loadState restores the last processed commit from .memo/entire_state.json
func (m *CheckpointMonitor) loadState() {
	path := filepath.Join(m.memoDir, "entire_state.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var state entireState
	if err := json.Unmarshal(data, &state); err != nil {
		return
	}

	m.lastCommit = state.LastCommit
	internal.LogDebug("Loaded entire state: lastCommit=%s", m.lastCommit)
}

// saveState persists the last processed commit to .memo/entire_state.json
func (m *CheckpointMonitor) saveState() {
	state := entireState{
		LastCommit: m.lastCommit,
		UpdatedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(state)
	if err != nil {
		internal.LogError("Failed to marshal entire state: %v", err)
		return
	}

	path := filepath.Join(m.memoDir, "entire_state.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		internal.LogError("Failed to save entire state: %v", err)
	}
}
