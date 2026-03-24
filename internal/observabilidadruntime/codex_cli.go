package observabilidadruntime

import (
	"encoding/json"
	"fmt"
	"strings"
)

type codexCLIAdapter struct{}

type codexCLIPayload struct {
	ObservedAt string         `json:"observed_at"`
	RuntimeRef string         `json:"runtime_ref"`
	SessionID  string         `json:"session_id"`
	Model      string         `json:"model"`
	CWD        string         `json:"cwd"`
	Command    string         `json:"command"`
	State      string         `json:"state"`
	Active     *bool          `json:"active"`
	PID        *int64         `json:"pid"`
	ExitCode   *int           `json:"exit_code"`
	Details    map[string]any `json:"details"`
}

func (codexCLIAdapter) Provider() string {
	return "codex_cli"
}

func (codexCLIAdapter) Normalize(payload json.RawMessage) (*Sample, error) {
	var in codexCLIPayload
	if err := json.Unmarshal(payload, &in); err != nil {
		return nil, fmt.Errorf("payload codex_cli invalido: %w", err)
	}

	observedAt, err := parseObservedAt(in.ObservedAt)
	if err != nil {
		return nil, err
	}
	state := strings.TrimSpace(in.State)
	if state == "" {
		switch {
		case in.Active != nil && *in.Active:
			state = "running"
		case in.ExitCode != nil:
			state = "exited"
		case in.Active != nil && !*in.Active:
			state = "idle"
		default:
			state = "unknown"
		}
	}

	return &Sample{
		Provider:   "codex_cli",
		Kind:       "cli_session_snapshot",
		ObservedAt: observedAt,
		RuntimeRef: strings.TrimSpace(in.RuntimeRef),
		SessionRef: strings.TrimSpace(in.SessionID),
		State:      state,
		PID:        in.PID,
		Command:    strings.TrimSpace(in.Command),
		Model:      strings.TrimSpace(in.Model),
		CWD:        strings.TrimSpace(in.CWD),
		ExitCode:   in.ExitCode,
		Details:    cloneDetails(in.Details),
	}, nil
}
