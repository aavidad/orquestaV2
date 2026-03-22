package localrpc

import "time"

const (
	DefaultHost      = "127.0.0.1"
	DefaultPort      = "17899"
	DefaultStateFile = "orquesta-localrpc.json"

	HealthPath = "/__orquesta/healthz"
	ExecPath   = "/__orquesta/cli/exec"
)

type State struct {
	Addr      string    `json:"addr"`
	PID       int       `json:"pid"`
	Kind      string    `json:"kind,omitempty"`
	ScopeID   string    `json:"scope_id,omitempty"`
	DBPath    string    `json:"db_path,omitempty"`
	StartedAt time.Time `json:"started_at"`
	Version   string    `json:"version,omitempty"`
}

type HealthResponse struct {
	OK        bool      `json:"ok"`
	Addr      string    `json:"addr"`
	PID       int       `json:"pid"`
	Kind      string    `json:"kind,omitempty"`
	ScopeID   string    `json:"scope_id,omitempty"`
	DBPath    string    `json:"db_path,omitempty"`
	StartedAt time.Time `json:"started_at"`
	Version   string    `json:"version,omitempty"`
}

type ExecRequest struct {
	Args []string `json:"args"`
}

type ExecResponse struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}
