package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type GenericProcessSnapshot struct {
	PID          int64          `json:"pid"`
	PPID         int64          `json:"ppid"`
	RawState     string         `json:"raw_state"`
	LogicalState string         `json:"logical_state"`
	Threads      int            `json:"threads"`
	ChildCount   int            `json:"child_count"`
	OpenFDs      int            `json:"open_fds"`
	RSSBytes     int64          `json:"rss_bytes"`
	MemBytes     int64          `json:"mem_bytes"`
	Extra        map[string]any `json:"extra,omitempty"`
}

func TomarMuestraGenericProcess(runtimeInst *RuntimeInstance) (*RuntimeTelemetrySample, error) {
	snap, err := leerGenericProcessSnapshot(runtimeInst)
	if err != nil {
		return nil, err
	}
	sampleJSON, _ := json.Marshal(snap)
	return &RuntimeTelemetrySample{
		CPUPct:       0,
		MemBytes:     snap.MemBytes,
		RSSBytes:     snap.RSSBytes,
		OpenFDs:      int64(snap.OpenFDs),
		ChildCount:   snap.ChildCount,
		ThreadCount:  snap.Threads,
		LogicalState: snap.LogicalState,
		Source:       "generic_process",
		SampleJSON:   string(sampleJSON),
	}, nil
}

func RegistrarMuestraGenericProcess(runtimeID int64, runtimeInst *RuntimeInstance) (*RuntimeTelemetrySample, error) {
	snap, err := leerGenericProcessSnapshot(runtimeInst)
	if err != nil {
		return nil, err
	}
	observedProcessState := ""
	if logicalState, processState, ok := runtimeObservedStateForGenericProcess(runtimeInst); ok {
		snap.LogicalState = logicalState
		observedProcessState = processState
	}
	sampleJSON, _ := json.Marshal(snap)
	sample := &RuntimeTelemetrySample{
		CPUPct:       0,
		MemBytes:     snap.MemBytes,
		RSSBytes:     snap.RSSBytes,
		OpenFDs:      int64(snap.OpenFDs),
		ChildCount:   snap.ChildCount,
		ThreadCount:  snap.Threads,
		LogicalState: snap.LogicalState,
		Source:       "generic_process",
		SampleJSON:   string(sampleJSON),
		RuntimeID:    runtimeID,
	}
	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	id, err := registrarRuntimeTelemetrySampleTx(tx, sample)
	if err != nil {
		return nil, err
	}
	sample.ID = id
	rawProcessState := snap.RawState
	if strings.TrimSpace(rawProcessState) == "" {
		rawProcessState = "desconocido"
	}
	if strings.TrimSpace(observedProcessState) != "" {
		rawProcessState = observedProcessState
	}
	if _, err := tx.Exec(`
		UPDATE runtime_instances
		SET pid = COALESCE(?, pid),
		    ppid = COALESCE(?, ppid),
		    child_count = ?,
		    thread_count = ?,
		    logical_state = ?,
		    process_state = ?,
		    last_event_at = CURRENT_TIMESTAMP,
		    last_heartbeat_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		int64PtrOrNil(snap.PID), int64PtrOrNil(snap.PPID),
		sample.ChildCount, sample.ThreadCount, sample.LogicalState, rawProcessState, runtimeID,
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return sample, nil
}

func runtimeObservedStateForGenericProcess(runtimeInst *RuntimeInstance) (string, string, bool) {
	if runtimeInst == nil || runtimeInst.SesionID == nil || *runtimeInst.SesionID <= 0 {
		return "", "", false
	}
	handle, err := GetRuntimeHandleBySesionID(*runtimeInst.SesionID)
	if err != nil || handle == nil {
		return "", "", false
	}
	return runtimeObservedLogicalStateFromStructuredWorker(handle)
}

func leerGenericProcessSnapshot(runtimeInst *RuntimeInstance) (*GenericProcessSnapshot, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("generic_process solo esta soportado en linux")
	}
	if runtimeInst == nil || runtimeInst.PID == nil || *runtimeInst.PID <= 0 {
		return nil, fmt.Errorf("generic_process requiere un runtime con PID real")
	}
	pid := *runtimeInst.PID
	statusPath := filepath.Join("/proc", strconv.FormatInt(pid, 10), "status")
	data, err := os.ReadFile(statusPath)
	if err != nil {
		return nil, err
	}

	snap := &GenericProcessSnapshot{
		PID:          pid,
		PPID:         0,
		RawState:     "unknown",
		LogicalState: logicalStateFromProcState("unknown"),
		Threads:      0,
		ChildCount:   0,
		OpenFDs:      0,
		RSSBytes:     0,
		MemBytes:     0,
		Extra:        map[string]any{},
	}

	for _, line := range strings.Split(string(data), "\n") {
		switch {
		case strings.HasPrefix(line, "State:"):
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				snap.RawState = fields[1]
				snap.LogicalState = logicalStateFromProcState(fields[1])
			}
		case strings.HasPrefix(line, "PPid:"):
			if v, err := strconv.ParseInt(strings.TrimSpace(strings.TrimPrefix(line, "PPid:")), 10, 64); err == nil {
				snap.PPID = v
			}
		case strings.HasPrefix(line, "Threads:"):
			if v, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Threads:"))); err == nil {
				snap.Threads = v
			}
		case strings.HasPrefix(line, "VmRSS:"):
			if v, err := parseLinuxMemLine(line); err == nil {
				snap.RSSBytes = v
			}
		case strings.HasPrefix(line, "VmSize:"):
			if v, err := parseLinuxMemLine(line); err == nil {
				snap.MemBytes = v
			}
		}
	}
	if snap.MemBytes == 0 {
		snap.MemBytes = snap.RSSBytes
	}
	if snap.RSSBytes == 0 {
		snap.RSSBytes = snap.MemBytes
	}
	if n, err := contarFDsProceso(pid); err == nil {
		snap.OpenFDs = n
	}
	if n, err := contarHijosProceso(pid); err == nil {
		snap.ChildCount = n
	}
	snap.Extra["pid"] = snap.PID
	snap.Extra["ppid"] = snap.PPID
	snap.Extra["raw_state"] = snap.RawState
	return snap, nil
}

func parseLinuxMemLine(line string) (int64, error) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0, fmt.Errorf("línea de memoria inválida")
	}
	v, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return 0, err
	}
	if len(fields) >= 3 && strings.EqualFold(fields[2], "kB") {
		v *= 1024
	}
	return v, nil
}

func contarFDsProceso(pid int64) (int, error) {
	entries, err := os.ReadDir(filepath.Join("/proc", strconv.FormatInt(pid, 10), "fd"))
	if err != nil {
		return 0, err
	}
	return len(entries), nil
}

func contarHijosProceso(pid int64) (int, error) {
	childrenPath := filepath.Join("/proc", strconv.FormatInt(pid, 10), "task", strconv.FormatInt(pid, 10), "children")
	data, err := os.ReadFile(childrenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	return len(strings.Fields(string(data))), nil
}

func logicalStateFromProcState(state string) string {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case "R":
		return "pensando"
	case "S", "I":
		return "esperando_io"
	case "D":
		return "bloqueado"
	case "T", "t":
		return "pausado"
	case "Z", "X":
		return "cerrado"
	default:
		return "disponible"
	}
}

func rawProcessStateFromLogical(logical string) string {
	switch strings.ToLower(strings.TrimSpace(logical)) {
	case "pensando":
		return "running"
	case "esperando_io":
		return "sleeping"
	case "bloqueado":
		return "blocked"
	case "pausado":
		return "stopped"
	case "cerrado":
		return "zombie"
	default:
		return "unknown"
	}
}

func int64PtrOrNil(v int64) *int64 {
	if v <= 0 {
		return nil
	}
	return &v
}
