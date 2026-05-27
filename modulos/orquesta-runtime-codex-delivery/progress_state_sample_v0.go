package orquestaruntimecodexdelivery

import (
	"fmt"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func normalizeCodexProgressSampleV0(sample CodexProgressSampleV0) CodexProgressSampleV0 {
	sample.RunID = strings.TrimSpace(sample.RunID)
	sample.AgentRequestID = strings.TrimSpace(sample.AgentRequestID)
	sample.ProcessRef = strings.TrimSpace(sample.ProcessRef)
	sample.Signature = strings.TrimSpace(sample.Signature)
	sample.ActionSignature = strings.TrimSpace(sample.ActionSignature)
	sample.EvidenceRefs = compactCodexDeliveryRefsV0(sample.EvidenceRefs)
	if sample.ObservedAt.IsZero() {
		sample.ObservedAt = orquestaruntime.NowUTCV0(nil)
	}
	if sample.MinUnchangedInterval < 0 {
		sample.MinUnchangedInterval = 0
	}
	return sample
}

func normalizeCodexProgressReportMarkV0(mark CodexProgressReportMarkV0) CodexProgressReportMarkV0 {
	mark.RunID = strings.TrimSpace(mark.RunID)
	mark.AgentRequestID = strings.TrimSpace(mark.AgentRequestID)
	mark.ProcessRef = strings.TrimSpace(mark.ProcessRef)
	mark.Signature = strings.TrimSpace(mark.Signature)
	return mark
}

func validateCodexProgressSampleV0(sample CodexProgressSampleV0) error {
	switch {
	case sample.RunID == "":
		return fmt.Errorf("codex_progress_sample: run_id requerido")
	case sample.AgentRequestID == "":
		return fmt.Errorf("codex_progress_sample: agent_request_id requerido")
	case sample.ProcessRef == "":
		return fmt.Errorf("codex_progress_sample: process_ref requerido")
	case sample.Signature == "":
		return fmt.Errorf("codex_progress_sample: signature requerida")
	default:
		return nil
	}
}

func validateCodexProgressReportMarkV0(mark CodexProgressReportMarkV0) error {
	if err := validateCodexProgressSampleV0(CodexProgressSampleV0{
		RunID:          mark.RunID,
		AgentRequestID: mark.AgentRequestID,
		ProcessRef:     mark.ProcessRef,
		Signature:      mark.Signature,
	}); err != nil {
		return err
	}
	switch mark.Status {
	case orquestaruntime.AgentProgressingV0,
		orquestaruntime.AgentStalledV0,
		orquestaruntime.AgentLoopDetectedV0,
		orquestaruntime.AgentStoppedV0:
		return nil
	default:
		return fmt.Errorf("codex_progress_report_mark: status requerido")
	}
}

func codexProgressSampleKeyV0(sample CodexProgressSampleV0) string {
	return sample.RunID + "\x00" + sample.AgentRequestID + "\x00" + sample.ProcessRef
}

func codexProgressReportMarkKeyV0(mark CodexProgressReportMarkV0) string {
	return mark.RunID + "\x00" + mark.AgentRequestID + "\x00" + mark.ProcessRef
}

func codexProgressHeartbeatRefV0(sample CodexProgressSampleV0, tick int) string {
	return fmt.Sprintf("heartbeat-ref-%s-%06d", sample.AgentRequestID, tick)
}
