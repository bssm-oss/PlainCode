package runtime

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	ModeAuto    = "auto"
	ModeProcess = "process"
	ModeDocker  = "docker"

	StatusRunning = "running"
	StatusStopped = "stopped"
	StatusUnknown = "unknown"

	HealthHealthy   = "healthy"
	HealthUnhealthy = "unhealthy"
	HealthUnknown   = "unknown"
)

// State captures the last known runtime state for a spec-managed service.
type State struct {
	SpecID         string            `json:"spec_id"`
	Mode           string            `json:"mode"`
	Status         string            `json:"status"`
	Health         string            `json:"health"`
	Command        []string          `json:"command,omitempty"`
	WorkingDir     string            `json:"working_dir,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	HealthcheckURL string            `json:"healthcheck_url,omitempty"`
	LogPath        string            `json:"log_path,omitempty"`
	PID            int               `json:"pid,omitempty"`
	PGID           int               `json:"pgid,omitempty"`
	Image          string            `json:"image,omitempty"`
	ContainerName  string            `json:"container_name,omitempty"`
	ContainerID    string            `json:"container_id,omitempty"`
	Ports          []string          `json:"ports,omitempty"`
	StartedAt      time.Time         `json:"started_at,omitempty"`
	StoppedAt      time.Time         `json:"stopped_at,omitempty"`
	LastCheckAt    time.Time         `json:"last_check_at,omitempty"`
	UpdatedAt      time.Time         `json:"updated_at,omitempty"`
	LastEvent      string            `json:"last_event,omitempty"`
	Error          string            `json:"error,omitempty"`
	StatePath      string            `json:"state_path,omitempty"`
	EventPath      string            `json:"event_path,omitempty"`
}

// Event captures a notable runtime lifecycle step for debugging.
type Event struct {
	Timestamp time.Time         `json:"timestamp"`
	Kind      string            `json:"kind"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
}

// Store persists runtime state under .plaincode/runs/.
type Store struct {
	dir string
}

// NewStore creates a runtime state store rooted at the given state directory.
func NewStore(stateDir string) *Store {
	return &Store{dir: filepath.Join(stateDir, "runs")}
}

// Save writes a runtime state to disk.
func (s *Store) Save(state *State) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("creating runtime state directory: %w", err)
	}
	state.UpdatedAt = time.Now()
	state.StatePath = s.statePath(state.SpecID)
	state.EventPath = s.EventPath(state.SpecID)
	if state.LogPath == "" {
		state.LogPath = s.LogPath(state.SpecID)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling runtime state: %w", err)
	}

	if err := os.WriteFile(s.statePath(state.SpecID), data, 0o644); err != nil {
		return fmt.Errorf("writing runtime state: %w", err)
	}
	return nil
}

// Load reads runtime state for a spec ID.
func (s *Store) Load(specID string) (*State, error) {
	data, err := os.ReadFile(s.statePath(specID))
	if err != nil {
		return nil, fmt.Errorf("reading runtime state: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing runtime state: %w", err)
	}
	state.StatePath = s.statePath(specID)
	state.EventPath = s.EventPath(specID)
	if state.LogPath == "" {
		state.LogPath = s.LogPath(specID)
	}
	return &state, nil
}

// Delete removes runtime state for a spec ID.
func (s *Store) Delete(specID string) error {
	if err := os.Remove(s.statePath(specID)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing runtime state: %w", err)
	}
	if err := os.Remove(s.EventPath(specID)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing runtime events: %w", err)
	}
	if err := os.Remove(s.LogPath(specID)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing runtime log: %w", err)
	}
	return nil
}

// ListAll returns every stored runtime state, sorted by spec ID.
func (s *Store) ListAll() ([]*State, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading runtime state directory: %w", err)
	}

	var states []*State
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		specID, err := url.PathUnescape(strings.TrimSuffix(entry.Name(), ".json"))
		if err != nil {
			continue
		}
		state, err := s.Load(specID)
		if err != nil {
			continue
		}
		states = append(states, state)
	}

	sort.Slice(states, func(i, j int) bool {
		return states[i].SpecID < states[j].SpecID
	})
	return states, nil
}

// LogPath returns the preferred log path for a spec's process runtime.
func (s *Store) LogPath(specID string) string {
	return filepath.Join(s.dir, specFileName(specID)+".log")
}

// EventPath returns the preferred event log path for a spec runtime.
func (s *Store) EventPath(specID string) string {
	return filepath.Join(s.dir, specFileName(specID)+".events.jsonl")
}

// AppendEvent records a lifecycle event to a JSONL file.
func (s *Store) AppendEvent(specID string, event Event) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("creating runtime state directory: %w", err)
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling runtime event: %w", err)
	}
	f, err := os.OpenFile(s.EventPath(specID), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("opening runtime event log: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("writing runtime event: %w", err)
	}
	return nil
}

// ReadEvents loads runtime events from disk. If limit > 0, only the most recent events are returned.
func (s *Store) ReadEvents(specID string, limit int) ([]Event, error) {
	data, err := os.ReadFile(s.EventPath(specID))
	if err != nil {
		return nil, fmt.Errorf("reading runtime events: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var events []Event
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var event Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("parsing runtime event: %w", err)
		}
		events = append(events, event)
	}

	if limit > 0 && len(events) > limit {
		events = events[len(events)-limit:]
	}
	return events, nil
}

// WriteLog stores a complete runtime log snapshot for a spec.
func (s *Store) WriteLog(specID string, data []byte) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("creating runtime state directory: %w", err)
	}
	if err := os.WriteFile(s.LogPath(specID), data, 0o644); err != nil {
		return fmt.Errorf("writing runtime log: %w", err)
	}
	return nil
}

func (s *Store) statePath(specID string) string {
	return filepath.Join(s.dir, specFileName(specID)+".json")
}

func specFileName(specID string) string {
	return url.PathEscape(specID)
}
