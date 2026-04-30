package agentesapp

import (
	"sync"
	"time"
)

type PanelBuildDiagnostics struct {
	Generated string           `json:"generated"`
	TotalMS   int64            `json:"total_ms"`
	Slow      bool             `json:"slow"`
	PhasesMS  map[string]int64 `json:"phases_ms,omitempty"`
}

var panelBuildDiagnosticsState struct {
	mu    sync.RWMutex
	value PanelBuildDiagnostics
}

func recordPanelBuildDiagnostics(total time.Duration, phases map[string]time.Duration) {
	diag := PanelBuildDiagnostics{
		Generated: time.Now().UTC().Format(time.RFC3339),
		TotalMS:   total.Milliseconds(),
		Slow:      total >= 500*time.Millisecond,
	}
	if len(phases) > 0 {
		diag.PhasesMS = make(map[string]int64, len(phases))
		for name, duration := range phases {
			diag.PhasesMS[name] = duration.Milliseconds()
		}
	}
	panelBuildDiagnosticsState.mu.Lock()
	panelBuildDiagnosticsState.value = diag
	panelBuildDiagnosticsState.mu.Unlock()
}

func GetPanelBuildDiagnostics() PanelBuildDiagnostics {
	panelBuildDiagnosticsState.mu.RLock()
	defer panelBuildDiagnosticsState.mu.RUnlock()
	diag := panelBuildDiagnosticsState.value
	if len(diag.PhasesMS) > 0 {
		cloned := make(map[string]int64, len(diag.PhasesMS))
		for name, value := range diag.PhasesMS {
			cloned[name] = value
		}
		diag.PhasesMS = cloned
	}
	return diag
}
