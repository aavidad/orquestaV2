package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaagentprocessregistrymemory "orquesta/modulos/orquesta-agent-process-registry-memory"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func codexProgressSpecAndAckPathForTestV0(
	t *testing.T,
) (orquestaruntime.ExternalAgentLaunchSpecV0, string) {
	t.Helper()
	spec := codexDeliverySpecForTestV0()
	ackPath := filepath.Join(t.TempDir(), orquestaruntimecodex.CodexAgentAckFileNameV0)
	return spec, ackPath
}

func codexProgressSourceForTestV0(
	t *testing.T,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
	ackPath string,
) CodexProgressObservationSourceV0 {
	t.Helper()
	registry := orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0()
	if err := registry.RecordAgentProcessV0(context.Background(), codexProgressProcessRecordForTestV0(spec)); err != nil {
		t.Fatalf("RecordAgentProcessV0: %v", err)
	}
	return CodexProgressObservationSourceV0{
		Store:           NewInMemoryCodexReceiptDescriptorStoreV0(codexProgressDescriptorForTestV0(spec, ackPath)),
		ProcessRegistry: registry,
		State:           NewInMemoryCodexProgressStateStoreV0(),
		Policy: orquestaruntime.AgentProgressHeartbeatPolicyV0{
			StalledAfterNoProgressTicks: 1,
			LoopAfterRepeatedActions:    4,
		},
	}
}

func codexProgressRequestForTestV0(
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) orquestacionnucleoapp.AgentProgressObservationRequestV0 {
	return orquestacionnucleoapp.AgentProgressObservationRequestV0{
		Run:           codexProgressRunForTestV0(spec),
		CorrelationID: "corr-progress-001",
	}
}

func codexProgressRunForTestV0(
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	return orquestacoreworkflow.OrchestrationRunV0{
		RunID:         "run-ref-001",
		StartedAgents: []string{spec.RequestID},
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
	}
}

func codexProgressDescriptorForTestV0(
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
	ackPath string,
) CodexReceiptDescriptorV0 {
	return CodexReceiptDescriptorV0{
		DescriptorRef: "receipt-ref-progress-001",
		RunID:         "run-ref-001",
		AgentRef:      spec.RequestID,
		Spec:          spec,
		AckPath:       ackPath,
	}
}

func codexProgressProcessRecordForTestV0(
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) orquestacionnucleoapp.AgentProcessRegistryRecordV0 {
	return orquestacionnucleoapp.AgentProcessRegistryRecordV0{
		RunID:          "run-ref-001",
		AgentRequestID: spec.RequestID,
		ProcessRef:     "process-ref-progress-001",
		SessionRef:     "session-ref-progress-001",
		LaunchRef:      "launch-ref-progress-001",
		ReadinessRef:   "readiness-ref-progress-001",
		EvidenceRefs:   []string{"process-ref-progress-001", "session-ref-progress-001"},
	}
}

func codexProgressACKBytesForTestV0(
	t *testing.T,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) []byte {
	t.Helper()
	path := writeCodexDeliveryAckForTestV0(t, spec, codexDeliveryAckForTestV0(spec))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ack: %v", err)
	}
	return data
}

func appendCodexProgressTestLogV0(path string, value string) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(value)
	return err
}

func codexProgressObservationLeaksPathV0(
	observation orquestacionnucleoapp.AgentProgressObservationV0,
	path string,
) bool {
	values := append([]string{
		observation.CandidateRef,
		observation.PhaseID,
		observation.TaskRef,
		observation.DeliveryRef,
		observation.AssessmentRef,
		observation.QuestionID,
		observation.Report.ReportID,
		observation.Report.Summary,
	}, observation.EvidenceRefs...)
	values = append(values, observation.Report.EvidenceRefs...)
	for _, value := range values {
		if strings.Contains(value, path) {
			return true
		}
	}
	return false
}

func codexProgressAssessmentProjectionForTestV0(
	observation orquestacionnucleoapp.AgentProgressObservationV0,
) string {
	return orquestacoreworkflow.AgentAssessmentProjectionRefV0(
		orquestacoreworkflow.AgentWorkAssessedPayloadV0{
			AssessmentRef:  "assessment-ref-" + observation.Report.ReportID,
			PhaseID:        observation.PhaseID,
			AgentRequestID: observation.Report.AgentRequestID,
			TaskRef:        observation.TaskRef,
			Verdict:        orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
			Action:         orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
			Severity:       orquestacoreworkflow.AgentAssessmentSeverityMediumV0,
			Summary:        "Assessment fake ya materializado.",
			EvidenceRefs:   observation.EvidenceRefs,
		},
	)
}
