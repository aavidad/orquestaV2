package observabilidadruntime

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type genericProcessAdapter struct{}

type genericProcessPayload struct {
	ObservedAt string         `json:"observed_at"`
	RuntimeRef string         `json:"runtime_ref"`
	SessionRef string         `json:"session_ref"`
	PID        *int64         `json:"pid"`
	Command    string         `json:"command"`
	Argv       []string       `json:"argv"`
	CWD        string         `json:"cwd"`
	State      string         `json:"state"`
	Running    *bool          `json:"running"`
	ExitCode   *int           `json:"exit_code"`
	CPUPercent *float64       `json:"cpu_pct"`
	RSSBytes   *int64         `json:"rss_bytes"`
	Details    map[string]any `json:"details"`
}

func (genericProcessAdapter) Provider() string {
	return "generic_process"
}

func (genericProcessAdapter) Normalize(payload json.RawMessage) (*Sample, error) {
	var in genericProcessPayload
	if err := json.Unmarshal(payload, &in); err != nil {
		return nil, fmt.Errorf("payload generic_process invalido: %w", err)
	}

	observedAt, err := parseObservedAt(in.ObservedAt)
	if err != nil {
		return nil, err
	}
	state := strings.TrimSpace(in.State)
	if state == "" {
		switch {
		case in.Running != nil && *in.Running:
			state = "running"
		case in.ExitCode != nil:
			state = "exited"
		case in.Running != nil && !*in.Running:
			state = "stopped"
		default:
			state = "unknown"
		}
	}

	details := cloneDetails(in.Details)
	if len(in.Argv) > 0 {
		details["argv"] = in.Argv
	}
	if in.CPUPercent != nil {
		details["cpu_pct"] = *in.CPUPercent
	}
	if in.RSSBytes != nil {
		details["rss_bytes"] = *in.RSSBytes
	}

	return &Sample{
		Provider:   "generic_process",
		Kind:       "process_snapshot",
		ObservedAt: observedAt,
		RuntimeRef: strings.TrimSpace(in.RuntimeRef),
		SessionRef: strings.TrimSpace(in.SessionRef),
		State:      state,
		PID:        in.PID,
		Command:    strings.TrimSpace(in.Command),
		CWD:        strings.TrimSpace(in.CWD),
		ExitCode:   in.ExitCode,
		Details:    details,
	}, nil
}

func parseObservedAt(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Now().UTC(), nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("observed_at invalido: %w", err)
	}
	return t.UTC(), nil
}

func cloneDetails(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
