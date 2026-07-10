package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	codexWaveProgressBudgetSchemaVersionV0  = "orquesta_codex_wave_progress_budget.v0"
	codexWaveProgressReceiptSchemaVersionV0 = "orquesta_codex_wave_progress_receipt.v0"
	codexWaveNoProgressReasonV0             = "no_progress_diagnostic_budget_exhausted"
)

// codexWaveProgressBudgetV0 is composition-only policy. It deliberately
// observes runtime files rather than leaking process or log details into core.
type codexWaveProgressBudgetV0 struct {
	SchemaVersion           string                              `json:"schema_version"`
	NoProgressBudgetSeconds int                                 `json:"no_progress_budget_seconds"`
	DiagnosticBudgetBytes   int64                               `json:"diagnostic_budget_bytes"`
	WriteSet                []string                            `json:"write_set,omitempty"`
	StartedAt               string                              `json:"started_at"`
	Baseline                map[string]codexWaveWriteSetStateV0 `json:"baseline,omitempty"`
}

type codexWaveWriteSetStateV0 struct {
	Exists           bool  `json:"exists"`
	SizeBytes        int64 `json:"size_bytes,omitempty"`
	ModifiedUnixNano int64 `json:"modified_unix_nano,omitempty"`
}

type codexWaveProgressReceiptV0 struct {
	SchemaVersion string `json:"schema_version"`
	ReceiptRef    string `json:"receipt_ref"`
	Reason        string `json:"reason"`
	TriggeredAt   string `json:"triggered_at"`
}

func codexWaveProgressBudgetFromFlagsV0(seconds int, diagnosticBytes int64, rawWriteSet string) (*codexWaveProgressBudgetV0, error) {
	if seconds == 0 && diagnosticBytes == 0 && strings.TrimSpace(rawWriteSet) == "" {
		return nil, nil
	}
	if seconds <= 0 || diagnosticBytes <= 0 {
		return nil, errors.New("progress_budget_incomplete")
	}
	writeSet, err := codexWaveProgressWriteSetV0(rawWriteSet)
	if err != nil {
		return nil, err
	}
	return &codexWaveProgressBudgetV0{
		SchemaVersion:           codexWaveProgressBudgetSchemaVersionV0,
		NoProgressBudgetSeconds: seconds,
		DiagnosticBudgetBytes:   diagnosticBytes,
		WriteSet:                writeSet,
	}, nil
}

func codexWaveProgressWriteSetV0(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	seen := make(map[string]bool, len(parts))
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		value := filepath.Clean(strings.TrimSpace(part))
		if value == "." || value == "" || filepath.IsAbs(value) ||
			strings.HasPrefix(value, ".."+string(filepath.Separator)) || value == ".." ||
			strings.Contains(value, "\\") || seen[value] {
			return nil, errors.New("progress_write_set_invalid")
		}
		seen[value] = true
		result = append(result, value)
	}
	return result, nil
}

func codexWaveInitializeProgressBudgetV0(summary *codexWaveLaunchSummaryV0) error {
	if summary == nil || summary.ProgressBudget == nil {
		return nil
	}
	budget := summary.ProgressBudget
	if budget.SchemaVersion != codexWaveProgressBudgetSchemaVersionV0 ||
		budget.NoProgressBudgetSeconds <= 0 || budget.DiagnosticBudgetBytes <= 0 {
		return errors.New("progress_budget_invalid")
	}
	startedAt, err := time.Parse(time.RFC3339, summary.CreatedAt)
	if err != nil {
		return errors.New("progress_budget_started_at_invalid")
	}
	budget.StartedAt = startedAt.UTC().Format(time.RFC3339)
	baseline, err := codexWaveWriteSetStateV0ForPaths(summary.ProjectWorkDir, budget.WriteSet)
	if err != nil {
		return err
	}
	budget.Baseline = baseline
	return nil
}

func codexWaveEnforceProgressBudgetV0(summary *codexWaveLaunchSummaryV0, now time.Time) {
	if summary == nil || summary.ProgressBudget == nil || summary.DryRun {
		return
	}
	budget := summary.ProgressBudget
	startedAt, err := time.Parse(time.RFC3339, budget.StartedAt)
	if err != nil || now.Before(startedAt.Add(time.Duration(budget.NoProgressBudgetSeconds)*time.Second)) {
		return
	}
	writeSetChanged, err := codexWaveWriteSetChangedV0(summary.ProjectWorkDir, budget)
	if err != nil {
		summary.Errors = append(summary.Errors, codexWavePublicErrorV0{Code: "progress_budget_write_set_unavailable"})
		return
	}
	for index := range summary.Agents {
		agent := &summary.Agents[index]
		if agent.ProgressReceipt != nil || !processAliveV0(agent.PID) ||
			agent.LastMessageBytes > 0 || writeSetChanged || agent.StderrBytes < budget.DiagnosticBudgetBytes {
			continue
		}
		if err := codexWaveValidateAgentProcessProofV0(*summary, *agent); err != nil {
			summary.Errors = append(summary.Errors, codexWaveProofErrorV0(agent.AgentRef, err))
			continue
		}
		control := codexWaveControlConfigV0{WaveRef: summary.WaveRef, ConfirmStop: summary.WaveRef, Reason: codexWaveNoProgressReasonV0}
		if err := codexWaveRequestCooperativeStopV0(*summary, *agent, control, now); err != nil {
			summary.Errors = append(summary.Errors, codexWavePublicErrorV0{AgentRef: agent.AgentRef, Code: "progress_budget_stop_request_failed"})
			continue
		}
		triggeredAt := now.UTC().Format(time.RFC3339)
		agent.StopRequestedAt = triggeredAt
		agent.ReworkRef = codexWaveOpaqueRefV0("rework", summary.WaveRef, codexWaveNoProgressReasonV0+":"+agent.AgentRef)
		agent.ProgressReceipt = &codexWaveProgressReceiptV0{
			SchemaVersion: codexWaveProgressReceiptSchemaVersionV0,
			ReceiptRef:    codexWaveOpaqueRefV0("progress-receipt", summary.WaveRef, agent.AgentRef+":"+triggeredAt),
			Reason:        codexWaveNoProgressReasonV0,
			TriggeredAt:   triggeredAt,
		}
		agent.Status = "stop_requested"
	}
}

func codexWaveWriteSetChangedV0(projectWorkDir string, budget *codexWaveProgressBudgetV0) (bool, error) {
	if budget == nil || len(budget.WriteSet) == 0 {
		return false, nil
	}
	current, err := codexWaveWriteSetStateV0ForPaths(projectWorkDir, budget.WriteSet)
	if err != nil {
		return false, err
	}
	for path, state := range current {
		if budget.Baseline[path] != state {
			return true, nil
		}
	}
	return false, nil
}

func codexWaveWriteSetStateV0ForPaths(projectWorkDir string, writeSet []string) (map[string]codexWaveWriteSetStateV0, error) {
	result := make(map[string]codexWaveWriteSetStateV0, len(writeSet))
	for _, relativePath := range writeSet {
		if _, err := codexWaveProgressWriteSetV0(relativePath); err != nil {
			return nil, errors.New("progress_write_set_invalid")
		}
		path := filepath.Join(projectWorkDir, relativePath)
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			result[relativePath] = codexWaveWriteSetStateV0{}
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("progress_write_set_stat_failed")
		}
		result[relativePath] = codexWaveWriteSetStateV0{Exists: true, SizeBytes: info.Size(), ModifiedUnixNano: info.ModTime().UnixNano()}
	}
	return result, nil
}

func codexWaveProgressBudgetCopyV0(value *codexWaveProgressBudgetV0) *codexWaveProgressBudgetV0 {
	if value == nil {
		return nil
	}
	copyValue := *value
	copyValue.WriteSet = append([]string(nil), value.WriteSet...)
	copyValue.Baseline = make(map[string]codexWaveWriteSetStateV0, len(value.Baseline))
	for path, state := range value.Baseline {
		copyValue.Baseline[path] = state
	}
	return &copyValue
}

func codexWaveProgressBudgetPublicCopyV0(value *codexWaveProgressBudgetV0) *codexWaveProgressBudgetV0 {
	return codexWaveProgressBudgetCopyV0(value)
}

func codexWaveProgressReceiptCopyV0(value *codexWaveProgressReceiptV0) *codexWaveProgressReceiptV0 {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}
