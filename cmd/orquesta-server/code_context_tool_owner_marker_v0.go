package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	orquestacontext "orquesta/modulos/orquesta-context"
)

const (
	serverCodeContextToolOwnerMarkerSchemaVersionV0 = "server_code_context_tool_owner_marker.v0"
	serverCodeContextToolOwnerMarkerDirV0           = "code-context-tool-owners"
	defaultCodeContextToolOwnerTermGraceV0          = 2 * time.Second
	defaultCodeContextToolOwnerKillGraceV0          = time.Second
)

type serverCodeContextToolOwnerMarkerV0 struct {
	SchemaVersion   string   `json:"schema_version"`
	OwnerRef        string   `json:"owner_ref"`
	ToolRef         string   `json:"tool_ref"`
	ProviderKind    string   `json:"provider_kind"`
	PID             int      `json:"pid,omitempty"`
	StartedAt       string   `json:"started_at,omitempty"`
	LastHeartbeatAt string   `json:"last_heartbeat_at,omitempty"`
	CPUPercent      int      `json:"cpu_percent,omitempty"`
	ActiveRequests  int      `json:"active_requests,omitempty"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
}

type serverFileCodeContextToolOwnerRegistryV0 struct {
	dir string
}

func newServerFileCodeContextToolOwnerRegistryV0(dir string) serverFileCodeContextToolOwnerRegistryV0 {
	return serverFileCodeContextToolOwnerRegistryV0{dir: strings.TrimSpace(dir)}
}

func (registry serverFileCodeContextToolOwnerRegistryV0) WriteCodeContextToolOwnerMarkerV0(
	marker serverCodeContextToolOwnerMarkerV0,
) error {
	marker = normalizeServerCodeContextToolOwnerMarkerV0(marker)
	if err := validateServerCodeContextToolOwnerMarkerV0(marker); err != nil {
		return err
	}
	path, err := registry.markerPathV0(marker.OwnerRef)
	if err != nil {
		return err
	}
	return writeServerCodeContextJSONFileV0(path, marker)
}

func (registry serverFileCodeContextToolOwnerRegistryV0) ReadCodeContextToolOwnerMarkerV0(
	ownerRef string,
) (serverCodeContextToolOwnerMarkerV0, error) {
	path, err := registry.markerPathV0(ownerRef)
	if err != nil {
		return serverCodeContextToolOwnerMarkerV0{}, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return serverCodeContextToolOwnerMarkerV0{}, err
	}
	var marker serverCodeContextToolOwnerMarkerV0
	if err := json.Unmarshal(raw, &marker); err != nil {
		return serverCodeContextToolOwnerMarkerV0{}, err
	}
	marker = normalizeServerCodeContextToolOwnerMarkerV0(marker)
	if err := validateServerCodeContextToolOwnerMarkerV0(marker); err != nil {
		return serverCodeContextToolOwnerMarkerV0{}, err
	}
	if marker.OwnerRef != strings.TrimSpace(ownerRef) {
		return serverCodeContextToolOwnerMarkerV0{}, errors.New("code_context_owner_marker_mismatch")
	}
	return marker, nil
}

func (registry serverFileCodeContextToolOwnerRegistryV0) markerPathV0(ownerRef string) (string, error) {
	ownerRef = strings.TrimSpace(ownerRef)
	if registry.dir == "" || ownerRef == "" || !isSafeServerCodeContextOwnerRefV0(ownerRef) {
		return "", errors.New("code_context_owner_marker_invalid")
	}
	return filepath.Join(registry.dir, serverCodeContextToolOwnerMarkerDirV0, ownerRef+".json"), nil
}

type serverFileCodeContextToolOwnerObserverV0 struct {
	Registry serverFileCodeContextToolOwnerRegistryV0
}

func (observer serverFileCodeContextToolOwnerObserverV0) ObserveCodeContextToolOwnerV0(
	ctx context.Context,
	lease orquestacontext.CodeContextToolLeaseV0,
	now time.Time,
) (orquestacontext.CodeContextToolLeaseObservationV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return orquestacontext.CodeContextToolLeaseObservationV0{}, err
	}
	marker, err := observer.Registry.ReadCodeContextToolOwnerMarkerV0(lease.OwnerRef)
	if err != nil {
		return orquestacontext.CodeContextToolLeaseObservationV0{}, err
	}
	evidence := append([]string{}, marker.EvidenceRefs...)
	if marker.PID > 0 && !processAliveV0(marker.PID) {
		evidence = append(evidence, "evidence-ref-code-context-owner-process-not-alive")
	}
	if marker.LastHeartbeatAt != "" {
		evidence = append(evidence, "evidence-ref-code-context-owner-heartbeat-present")
	}
	return orquestacontext.CodeContextToolLeaseObservationV0{
		ObservationRef: "observation-ref-code-context-owner-" + marker.OwnerRef,
		ObservedAt:     now.UTC().Format(time.RFC3339),
		Lease:          lease,
		LastRequestAt:  marker.LastHeartbeatAt,
		CPUPercent:     nonNegativeEnvlessIntV0(marker.CPUPercent),
		ActiveRequests: nonNegativeEnvlessIntV0(marker.ActiveRequests),
		CPUHighPercent: 75,
		EvidenceRefs:   compactEnvlessStringsV0(evidence),
	}, nil
}

type serverFileCodeContextToolOwnerStopperV0 struct {
	Registry serverFileCodeContextToolOwnerRegistryV0
	Signal   func(int, syscall.Signal) error
	Alive    func(int) bool
	TermWait time.Duration
	KillWait time.Duration
}

func (stopper serverFileCodeContextToolOwnerStopperV0) StopCodeContextToolOwnerV0(
	ctx context.Context,
	ownerRef string,
) ([]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	marker, err := stopper.Registry.ReadCodeContextToolOwnerMarkerV0(ownerRef)
	if err != nil {
		return nil, err
	}
	if marker.ProviderKind != orquestacontext.CodeContextProviderKindCodebaseMCPV0 {
		return nil, errors.New("code_context_owner_marker_provider_not_codebase")
	}
	if marker.PID <= 0 {
		return []string{"evidence-ref-code-context-owner-no-pid"}, nil
	}
	alive := stopper.Alive
	if alive == nil {
		alive = processAliveV0
	}
	if !alive(marker.PID) {
		return []string{"evidence-ref-code-context-owner-already-stopped"}, nil
	}
	signal := stopper.Signal
	if signal == nil {
		signal = serverSignalCodeContextToolOwnerV0
	}
	if err := signal(marker.PID, syscall.SIGTERM); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return nil, err
	}
	evidence := []string{"evidence-ref-code-context-owner-stop-signal-sent"}
	termWait := stopper.TermWait
	if termWait <= 0 {
		termWait = defaultCodeContextToolOwnerTermGraceV0
	}
	if waitServerCodeContextToolOwnerDownV0(ctx, marker.PID, termWait, alive) {
		return append(evidence, "evidence-ref-code-context-owner-stopped-after-term"), nil
	}
	if err := signal(marker.PID, syscall.SIGKILL); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return evidence, err
	}
	evidence = append(evidence, "evidence-ref-code-context-owner-kill-signal-sent")
	killWait := stopper.KillWait
	if killWait <= 0 {
		killWait = defaultCodeContextToolOwnerKillGraceV0
	}
	if !waitServerCodeContextToolOwnerDownV0(ctx, marker.PID, killWait, alive) {
		return evidence, errors.New("code_context_owner_stop_timeout")
	}
	return append(evidence, "evidence-ref-code-context-owner-stopped-after-kill"), nil
}

func normalizeServerCodeContextToolOwnerMarkerV0(
	marker serverCodeContextToolOwnerMarkerV0,
) serverCodeContextToolOwnerMarkerV0 {
	marker.SchemaVersion = strings.TrimSpace(marker.SchemaVersion)
	if marker.SchemaVersion == "" {
		marker.SchemaVersion = serverCodeContextToolOwnerMarkerSchemaVersionV0
	}
	marker.OwnerRef = strings.TrimSpace(marker.OwnerRef)
	marker.ToolRef = strings.TrimSpace(marker.ToolRef)
	marker.ProviderKind = strings.TrimSpace(marker.ProviderKind)
	marker.StartedAt = strings.TrimSpace(marker.StartedAt)
	marker.LastHeartbeatAt = strings.TrimSpace(marker.LastHeartbeatAt)
	marker.CPUPercent = nonNegativeEnvlessIntV0(marker.CPUPercent)
	marker.ActiveRequests = nonNegativeEnvlessIntV0(marker.ActiveRequests)
	marker.EvidenceRefs = compactEnvlessStringsV0(marker.EvidenceRefs)
	return marker
}

func validateServerCodeContextToolOwnerMarkerV0(marker serverCodeContextToolOwnerMarkerV0) error {
	if marker.SchemaVersion != serverCodeContextToolOwnerMarkerSchemaVersionV0 ||
		marker.OwnerRef == "" ||
		marker.ToolRef == "" ||
		marker.ProviderKind == "" ||
		!isSafeServerCodeContextOwnerRefV0(marker.OwnerRef) {
		return errors.New("code_context_owner_marker_invalid")
	}
	if marker.StartedAt != "" {
		if _, err := time.Parse(time.RFC3339, marker.StartedAt); err != nil {
			return errors.New("code_context_owner_marker_invalid")
		}
	}
	if marker.LastHeartbeatAt != "" {
		if _, err := time.Parse(time.RFC3339, marker.LastHeartbeatAt); err != nil {
			return errors.New("code_context_owner_marker_invalid")
		}
	}
	return nil
}

func isSafeServerCodeContextOwnerRefV0(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value, "..") {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' ||
			r == '_' {
			continue
		}
		return false
	}
	return true
}

func serverSignalProcessV0(pid int, signal syscall.Signal) error {
	if pid <= 0 {
		return errors.New("code_context_owner_pid_invalid")
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Signal(signal)
}

func serverSignalCodeContextToolOwnerV0(pid int, signal syscall.Signal) error {
	if signal == syscall.SIGTERM {
		return signalProcessGroupV0(pid)
	}
	if signal == syscall.SIGKILL {
		return signalProcessGroupKillV0(pid)
	}
	return serverSignalProcessV0(pid, signal)
}

func waitServerCodeContextToolOwnerDownV0(
	ctx context.Context,
	pid int,
	timeout time.Duration,
	alive func(int) bool,
) bool {
	if ctx == nil {
		ctx = context.Background()
	}
	if alive == nil {
		alive = processAliveV0
	}
	if pid <= 0 || !alive(pid) {
		return true
	}
	if timeout <= 0 {
		timeout = 100 * time.Millisecond
	}
	deadline := time.Now().Add(timeout)
	for {
		if !alive(pid) {
			return true
		}
		if err := ctx.Err(); err != nil {
			return false
		}
		if !time.Now().Before(deadline) {
			return !alive(pid)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func nonNegativeEnvlessIntV0(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func compactEnvlessStringsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
