package db

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"orquesta/internal/controlruntime"
	"orquesta/runtimeagente"
)

func runtimeOrderSendInstructionLoadWorkerSnapshotFresh(raw string) (*runtimeagente.WorkerSnapshot, error) {
	type workerMetadataPaths struct {
		ManifestPath  string `json:"worker_manifest_path"`
		StatusPath    string `json:"worker_status_path"`
		HeartbeatPath string `json:"worker_heartbeat_path"`
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var meta workerMetadataPaths
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(meta.ManifestPath) == "" && strings.TrimSpace(meta.StatusPath) == "" && strings.TrimSpace(meta.HeartbeatPath) == "" {
		return nil, nil
	}
	snap := &runtimeagente.WorkerSnapshot{
		ManifestPath:  strings.TrimSpace(meta.ManifestPath),
		StatusPath:    strings.TrimSpace(meta.StatusPath),
		HeartbeatPath: strings.TrimSpace(meta.HeartbeatPath),
	}
	if data, err := os.ReadFile(snap.ManifestPath); err == nil {
		var manifest runtimeagente.WorkerManifest
		if json.Unmarshal(data, &manifest) == nil {
			snap.Manifest = &manifest
		}
	}
	if data, err := os.ReadFile(snap.StatusPath); err == nil {
		var status runtimeagente.WorkerStatus
		if json.Unmarshal(data, &status) == nil {
			snap.Status = &status
		}
	}
	if data, err := os.ReadFile(snap.HeartbeatPath); err == nil {
		var heartbeat runtimeagente.WorkerHeartbeat
		if json.Unmarshal(data, &heartbeat) == nil {
			snap.Heartbeat = &heartbeat
		}
	}
	return snap, nil
}

func runtimeOrderSendInstructionPermiteReceiptLastOutput(handle *RuntimeHandle, snap *runtimeagente.WorkerSnapshot, payload map[string]any) bool {
	if handle == nil || snap == nil {
		return false
	}
	if runtimeOrderSendInstructionUsaTMUXPaneActivityEvidence(handle, payload) {
		return false
	}
	if runtimeOrderSendInstructionEsPipelinePremium(payload) {
		return false
	}
	if RuntimeHandleMailboxDeliveryMode(handle) != runtimeagente.MailboxDeliveryInteractive {
		return true
	}
	view := snap.View(time.Now().UTC(), time.Minute)
	if view == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(view.State)) {
	case "ready", "running":
		return true
	default:
		return false
	}
}

func runtimeOrderSendInstructionPremiumActivityEvidence(runtime *RuntimeInstance, handle *RuntimeHandle, snap *runtimeagente.WorkerSnapshot, payload map[string]any, baseline time.Time) (bool, string, time.Time, error) {
	if handle == nil || snap == nil {
		return false, "", time.Time{}, nil
	}
	usaPipelinePremium := runtimeOrderSendInstructionEsPipelinePremium(payload)
	usaTMUXPaneEvidence := runtimeOrderSendInstructionUsaTMUXPaneActivityEvidence(handle, payload)
	if !usaPipelinePremium && !usaTMUXPaneEvidence {
		return false, "", time.Time{}, nil
	}
	view := snap.View(time.Now().UTC(), time.Minute)
	if view == nil || view.HeartbeatStale || !view.Alive {
		return false, "", time.Time{}, nil
	}
	switch strings.ToLower(strings.TrimSpace(view.State)) {
	case "ready", "idle", "running":
	default:
		return false, "", time.Time{}, nil
	}
	driver := strings.ToLower(strings.TrimSpace(view.Driver))
	transport := strings.ToLower(strings.TrimSpace(view.Transport))
	if driver == "process_pty_cli" || transport == "pty_broker" || transport == "cli" {
		if !usaPipelinePremium {
			return false, "", time.Time{}, nil
		}
		for _, candidate := range []*time.Time{
			snap.LastProgressTime(),
			snap.LastOutputTime(),
		} {
			if candidate == nil || candidate.IsZero() {
				continue
			}
			receiptAt := candidate.UTC()
			if runtimeOrderSendInstructionReceiptAtOrAfter(receiptAt, baseline) {
				return true, "worker_activity", receiptAt, nil
			}
		}
		return false, "", time.Time{}, nil
	}
	if !RuntimeHandlePermiteSendInputInteractivo(handle) && !usaTMUXPaneEvidence {
		return false, "", time.Time{}, nil
	}
	known, captured, err := controlruntime.CaptureTMUXPaneMetadata(strings.TrimSpace(handle.MetadataJSON))
	if err != nil {
		return false, "", time.Time{}, nil
	}
	if !known {
		for _, candidate := range []*time.Time{
			snap.LastProgressTime(),
			snap.LastOutputTime(),
		} {
			if candidate == nil || candidate.IsZero() {
				continue
			}
			receiptAt := candidate.UTC()
			if runtimeOrderSendInstructionReceiptAtOrAfter(receiptAt, baseline) {
				return true, "tmux_pane_activity", receiptAt, nil
			}
		}
		return runtimeOrderSendInstructionTMUXTranscriptActivityEvidence(runtime, handle, baseline)
	}
	if !known || !runtimeTMUXPaneShowsInteractiveWork(captured) {
		return runtimeOrderSendInstructionTMUXTranscriptActivityEvidence(runtime, handle, baseline)
	}
	if usaPipelinePremium {
		return runtimeOrderSendInstructionTMUXTranscriptActivityEvidence(runtime, handle, baseline)
	}
	for _, candidate := range []*time.Time{
		snap.LastProgressTime(),
		snap.LastOutputTime(),
	} {
		if candidate == nil || candidate.IsZero() {
			continue
		}
		receiptAt := candidate.UTC()
		if runtimeOrderSendInstructionReceiptAtOrAfter(receiptAt, baseline) {
			return true, "tmux_pane_activity", receiptAt, nil
		}
	}
	return runtimeOrderSendInstructionTMUXTranscriptActivityEvidence(runtime, handle, baseline)
}

func runtimeTMUXPaneShowsInteractiveWork(captured string) bool {
	normalized := strings.ToLower(strings.TrimSpace(normalizarTextoTranscript(captured)))
	if normalized == "" {
		return false
	}
	return strings.Contains(normalized, "esc to interrupt") ||
		strings.Contains(normalized, "background terminal running") ||
		strings.Contains(normalized, "shift+tab to accept edits")
}

func runtimeOrderSendInstructionTMUXTranscriptActivityEvidence(runtime *RuntimeInstance, handle *RuntimeHandle, baseline time.Time) (bool, string, time.Time, error) {
	if runtime == nil || handle == nil {
		return false, "", time.Time{}, nil
	}
	filter := FiltroRuntimeTranscript{
		RuntimeID: &runtime.ID,
		HandleID:  &handle.ID,
		Limit:     64,
	}
	entries, err := ListarRuntimeTranscript(filter)
	if err != nil {
		return false, "", time.Time{}, err
	}
	for idx := len(entries) - 1; idx >= 0; idx-- {
		entry := entries[idx]
		if entry == nil || !runtimeOrderSendInstructionReceiptAtOrAfter(entry.CreatedAt.UTC(), baseline) {
			continue
		}
		if !runtimeTranscriptLooksLikeTMUXInteractiveActivity(entry) {
			continue
		}
		return true, "tmux_transcript_activity", entry.CreatedAt.UTC(), nil
	}
	return false, "", time.Time{}, nil
}

func runtimeTranscriptLooksLikeTMUXInteractiveActivity(entry *RuntimeTranscriptEntry) bool {
	if entry == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(entry.Stream), "pty_out") {
		return false
	}
	normalized := strings.TrimSpace(strings.ToLower(entry.NormalizedText))
	if normalized == "" {
		normalized = strings.TrimSpace(strings.ToLower(normalizarTextoTranscript(entry.Text)))
	}
	if normalized == "" {
		return false
	}
	switch strings.TrimSpace(strings.ToLower(entry.Classification)) {
	case "runtime_panic", "runtime_failure_signal":
		return false
	case "tool_execution", "tool_exploration":
		return true
	case "bootstrap_guidance":
		return false
	}
	markers := []string{
		"working",
		"ran ",
		"explored",
		"edited ",
		"waited for background terminal",
		"aplique un slice",
		"apliqué un slice",
		"gofmt",
		"go test",
		"git diff",
	}
	for _, marker := range markers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func runtimeOrderSendInstructionTranscriptPatchEvidence(runtime *RuntimeInstance, handle *RuntimeHandle, baseline time.Time) (bool, string, time.Time, error) {
	if runtime == nil || handle == nil {
		return false, "", time.Time{}, nil
	}
	filter := FiltroRuntimeTranscript{
		RuntimeID: &runtime.ID,
		HandleID:  &handle.ID,
		Limit:     64,
	}
	entries, err := ListarRuntimeTranscript(filter)
	if err != nil {
		return false, "", time.Time{}, err
	}
	for idx := len(entries) - 1; idx >= 0; idx-- {
		entry := entries[idx]
		if entry == nil || !runtimeOrderSendInstructionReceiptAtOrAfter(entry.CreatedAt.UTC(), baseline) {
			continue
		}
		if !runtimeTranscriptLooksLikeMicroprogramacionPatch(entry) {
			continue
		}
		return true, "transcript_patch", entry.CreatedAt.UTC(), nil
	}
	windowText := strings.Builder{}
	var latestAt time.Time
	for idx := len(entries) - 1; idx >= 0; idx-- {
		entry := entries[idx]
		if entry == nil || !runtimeOrderSendInstructionReceiptAtOrAfter(entry.CreatedAt.UTC(), baseline) {
			continue
		}
		if strings.TrimSpace(entry.Text) == "" {
			continue
		}
		if windowText.Len() > 0 {
			windowText.WriteString("\n")
		}
		windowText.WriteString(entry.Text)
		if entry.CreatedAt.UTC().After(latestAt) {
			latestAt = entry.CreatedAt.UTC()
		}
	}
	if latestAt.IsZero() {
		return false, "", time.Time{}, nil
	}
	if runtimeMicroprogramacionPatchDetected(windowText.String(), "") {
		return true, "transcript_patch", latestAt, nil
	}
	return false, "", time.Time{}, nil
}

func runtimeOrderSendInstructionTMUXPanePatchEvidence(handle *RuntimeHandle, snap *runtimeagente.WorkerSnapshot, baseline time.Time) (bool, string, time.Time, error) {
	if handle == nil || snap == nil {
		return false, "", time.Time{}, nil
	}
	known, captured, err := controlruntime.CaptureTMUXPaneMetadata(strings.TrimSpace(handle.MetadataJSON))
	if err != nil {
		return false, "", time.Time{}, nil
	}
	if !known || !runtimeMicroprogramacionPatchDetected(captured, "") {
		return false, "", time.Time{}, nil
	}
	for _, candidate := range []*time.Time{
		snap.LastOutputTime(),
		snap.LastProgressTime(),
		snap.HeartbeatTime(),
		snap.UpdatedTime(),
		snap.ReadyTime(),
	} {
		if candidate == nil || candidate.IsZero() {
			continue
		}
		receiptAt := candidate.UTC()
		if baseline.IsZero() || receiptAt.After(baseline) {
			return true, "tmux_pane_patch", receiptAt, nil
		}
	}
	return true, "tmux_pane_patch", time.Now().UTC(), nil
}

func runtimeOrderSendInstructionTranscriptBlockedByAgent(runtime *RuntimeInstance, handle *RuntimeHandle, baseline time.Time) (bool, string, time.Time, error) {
	if runtime == nil || handle == nil {
		return false, "", time.Time{}, nil
	}
	filter := FiltroRuntimeTranscript{
		RuntimeID: &runtime.ID,
		HandleID:  &handle.ID,
		Limit:     64,
	}
	entries, err := ListarRuntimeTranscript(filter)
	if err != nil {
		return false, "", time.Time{}, err
	}
	for idx := len(entries) - 1; idx >= 0; idx-- {
		entry := entries[idx]
		if entry == nil || !runtimeOrderSendInstructionReceiptAtOrAfter(entry.CreatedAt.UTC(), baseline) {
			continue
		}
		if !runtimeTranscriptLooksLikeMicroprogramacionBlockedResponse(entry.Text, entry.NormalizedText) {
			continue
		}
		return true, strings.TrimSpace(entry.Text), entry.CreatedAt.UTC(), nil
	}
	return false, "", time.Time{}, nil
}
