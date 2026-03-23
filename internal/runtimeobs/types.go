package runtimeobs

import "time"

type Sample struct {
	Provider   string         `json:"provider"`
	Kind       string         `json:"kind"`
	ObservedAt time.Time      `json:"observed_at"`
	RuntimeRef string         `json:"runtime_ref,omitempty"`
	SessionRef string         `json:"session_ref,omitempty"`
	State      string         `json:"state,omitempty"`
	PID        *int64         `json:"pid,omitempty"`
	Command    string         `json:"command,omitempty"`
	Model      string         `json:"model,omitempty"`
	CWD        string         `json:"cwd,omitempty"`
	ExitCode   *int           `json:"exit_code,omitempty"`
	Details    map[string]any `json:"details,omitempty"`
}
