package cmd

import (
	"runtime"
	"sync"
	"time"

	"orquesta/agentesapp"
)

type serverPhaseDiagnostics struct {
	Generated string           `json:"generated"`
	TotalMS   int64            `json:"total_ms"`
	PhasesMS  map[string]int64 `json:"phases_ms,omitempty"`
	Slow      bool             `json:"slow"`
}

type serverHotPathDiagnostics struct {
	Generated      string                            `json:"generated"`
	LastSelfHeal   *serverPhaseDiagnostics           `json:"last_self_heal,omitempty"`
	LastPanelBuild *agentesapp.PanelBuildDiagnostics `json:"last_panel_build,omitempty"`
}

type serverRuntimeHealth struct {
	Generated      string                   `json:"generated"`
	UptimeSeconds  int64                    `json:"uptime_seconds"`
	Goroutines     int                      `json:"goroutines"`
	HeapAllocBytes uint64                   `json:"heap_alloc_bytes"`
	HeapObjects    uint64                   `json:"heap_objects"`
	SysBytes       uint64                   `json:"sys_bytes"`
	NumGC          uint32                   `json:"num_gc"`
	LastGCPauseMS  int64                    `json:"last_gc_pause_ms"`
	HotPaths       serverHotPathDiagnostics `json:"hot_paths"`
}

var (
	serverProcessStartedAt                = time.Now().UTC()
	serverRuntimeHealthPanelDiagnosticsFn = agentesapp.GetPanelBuildDiagnostics
	serverDiagnosticsState                struct {
		mu           sync.RWMutex
		lastSelfHeal serverPhaseDiagnostics
	}
)

func recordServerSelfHealDiagnostics(total time.Duration, phases map[string]time.Duration) {
	diag := serverPhaseDiagnostics{
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
	serverDiagnosticsState.mu.Lock()
	serverDiagnosticsState.lastSelfHeal = diag
	serverDiagnosticsState.mu.Unlock()
}

func readServerHotPathDiagnostics() serverHotPathDiagnostics {
	out := serverHotPathDiagnostics{Generated: time.Now().UTC().Format(time.RFC3339)}

	serverDiagnosticsState.mu.RLock()
	lastSelfHeal := serverDiagnosticsState.lastSelfHeal
	serverDiagnosticsState.mu.RUnlock()
	if lastSelfHeal.Generated != "" {
		cloned := lastSelfHeal
		if len(lastSelfHeal.PhasesMS) > 0 {
			clonedMap := make(map[string]int64, len(lastSelfHeal.PhasesMS))
			for name, value := range lastSelfHeal.PhasesMS {
				clonedMap[name] = value
			}
			cloned.PhasesMS = clonedMap
		}
		out.LastSelfHeal = &cloned
	}
	panel := serverRuntimeHealthPanelDiagnosticsFn()
	if panel.Generated != "" {
		out.LastPanelBuild = &panel
	}
	return out
}

func buildServerRuntimeHealth() serverRuntimeHealth {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	lastPause := int64(0)
	if mem.NumGC > 0 {
		lastPause = int64(mem.PauseNs[(mem.NumGC-1)%uint32(len(mem.PauseNs))] / uint64(time.Millisecond))
	}
	return serverRuntimeHealth{
		Generated:      time.Now().UTC().Format(time.RFC3339),
		UptimeSeconds:  int64(time.Since(serverProcessStartedAt).Seconds()),
		Goroutines:     runtime.NumGoroutine(),
		HeapAllocBytes: mem.HeapAlloc,
		HeapObjects:    mem.HeapObjects,
		SysBytes:       mem.Sys,
		NumGC:          mem.NumGC,
		LastGCPauseMS:  lastPause,
		HotPaths:       readServerHotPathDiagnostics(),
	}
}
