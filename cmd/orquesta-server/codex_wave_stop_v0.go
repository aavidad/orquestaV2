package main

import (
	"errors"
	"path/filepath"
	"strings"
	"time"

	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func codexWaveLoadTrustedRegistryV0(
	config codexWaveControlConfigV0,
) (codexWaveLaunchSummaryV0, error) {
	summary, err := codexWaveLoadRegistryV0(config.RuntimeWorkDir)
	if err != nil {
		return codexWaveLaunchSummaryV0{}, err
	}
	if strings.TrimSpace(summary.WaveRef) == "" ||
		(config.WaveRef != "" && summary.WaveRef != config.WaveRef) {
		return codexWaveLaunchSummaryV0{}, errors.New("wave_ref_mismatch")
	}
	if !sameCleanPathV0(summary.RuntimeWorkDir, config.RuntimeWorkDir) {
		return codexWaveLaunchSummaryV0{}, errors.New("runtime_scope_mismatch")
	}
	if !codexWaveRuntimeUnderAllowedRootV0(summary.RuntimeWorkDir, summary.ProjectWorkDir) {
		return codexWaveLaunchSummaryV0{}, errors.New("blocked_registry_untrusted")
	}
	return summary, nil
}

func codexWaveApplyStopRequestV0(
	summary *codexWaveLaunchSummaryV0,
	config codexWaveControlConfigV0,
	now time.Time,
) {
	if summary == nil {
		return
	}
	if err := codexWaveValidateStopPolicyV0(*summary, config); err != nil {
		summary.Errors = append(summary.Errors, codexWavePublicErrorV0{Code: err.Error()})
		return
	}
	matched := false
	for i := range summary.Agents {
		agent := &summary.Agents[i]
		if config.AgentRef != "" && agent.AgentRef != config.AgentRef {
			continue
		}
		matched = true
		codexWaveStopOneAgentV0(summary, agent, config, now)
	}
	if !matched {
		summary.Errors = append(summary.Errors, codexWavePublicErrorV0{Code: "agent_ref_not_found"})
	}
}

func codexWaveValidateStopPolicyV0(
	summary codexWaveLaunchSummaryV0,
	config codexWaveControlConfigV0,
) error {
	if strings.TrimSpace(config.ConfirmStop) != strings.TrimSpace(summary.WaveRef) {
		return errors.New("stop_confirmation_required")
	}
	if strings.TrimSpace(config.Reason) == "" {
		return errors.New("stop_reason_required")
	}
	if config.ForceStop {
		return nil
	}
	return nil
}

func codexWaveStopOneAgentV0(
	summary *codexWaveLaunchSummaryV0,
	agent *codexWaveAgentSummaryV0,
	config codexWaveControlConfigV0,
	now time.Time,
) {
	if agent.PID <= 0 || !processAliveV0(agent.PID) {
		return
	}
	if err := codexWaveValidateAgentProcessProofV0(*summary, *agent); err != nil {
		summary.Errors = append(summary.Errors, codexWaveProofErrorV0(agent.AgentRef, err))
		return
	}
	if !config.ForceStop {
		if err := codexWaveRequestCooperativeStopV0(*summary, *agent, config, now); err != nil {
			summary.Errors = append(summary.Errors, codexWavePublicErrorV0{
				AgentRef: agent.AgentRef,
				Code:     "cooperative_stop_request_failed",
			})
			return
		}
		agent.StopRequestedAt = now.UTC().Format(time.RFC3339)
		agent.Status = "stop_requested"
		return
	}
	if err := signalProcessGroupV0(agent.PID); err != nil {
		summary.Errors = append(summary.Errors, codexWavePublicErrorV0{
			AgentRef: agent.AgentRef,
			Code:     "stop_failed",
		})
		return
	}
	agent.StopRequestedAt = now.UTC().Format(time.RFC3339)
	agent.Status = "stop_requested"
}

func codexWaveRequestCooperativeStopV0(
	summary codexWaveLaunchSummaryV0,
	agent codexWaveAgentSummaryV0,
	config codexWaveControlConfigV0,
	now time.Time,
) error {
	request := orquestaruntimecodex.CodexShutdownRequestV0{
		SchemaVersion:      orquestaruntimecodex.CodexShutdownRequestSchemaVersionV0,
		RunRef:             "run-ref-" + safeCodexWavePurgeRefPartV0(summary.WaveRef),
		AgentRef:           agent.AgentRef,
		CorrelationID:      "corr-codex-wave-stop-" + safeCodexWavePurgeRefPartV0(agent.AgentRef),
		RequestedBy:        "codex-wave-stop",
		Reason:             config.Reason,
		ShutdownAttemptRef: codexWaveStopAttemptRefV0(summary, agent, config, now),
		CheckpointRef:      "checkpoint-ref-" + safeCodexWavePurgeRefPartV0(agent.AgentRef),
		EvidenceRefs:       []string{"codex-wave-stop-cooperative-ref-" + safeCodexWavePurgeRefPartV0(summary.WaveRef)},
	}
	path := filepath.Join(agent.RuntimeWorkDir, orquestaruntimecodex.CodexShutdownRequestFileNameV0)
	if issues := orquestaruntimecodex.WriteCodexShutdownRequestFileV0(path, request); len(issues) > 0 {
		return errors.New("shutdown_request_invalid")
	}
	return nil
}
