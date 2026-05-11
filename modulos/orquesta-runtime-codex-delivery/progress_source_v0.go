package orquestaruntimecodexdelivery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

type CodexProcessSnapshotSourcePortV0 interface {
	SnapshotV0(processRef string) (orquestaruntime.ProcessRuntimeSnapshotV0, error)
}

type CodexProgressObservationSourceV0 struct {
	Store                      CodexReceiptDescriptorStorePortV0
	ProcessRegistry            orquestacionnucleoapp.AgentProcessRegistryPortV0
	SnapshotSource             CodexProcessSnapshotSourcePortV0
	State                      CodexProgressStateStorePortV0
	Policy                     orquestaruntime.AgentProgressHeartbeatPolicyV0
	BudgetPolicy               CodexBudgetActivityPolicyV0
	MinUnchangedSampleInterval time.Duration
}

var _ orquestacionnucleoapp.AgentProgressObservationProviderPortV0 = CodexProgressObservationSourceV0{}

func (source CodexProgressObservationSourceV0) BuildAgentProgressObservationsV0(
	ctx context.Context,
	request orquestacionnucleoapp.AgentProgressObservationRequestV0,
) ([]orquestacionnucleoapp.AgentProgressObservationV0, error) {
	if err := source.validateV0(); err != nil {
		return nil, err
	}
	descriptors, err := source.Store.ListCodexReceiptDescriptorsV0(
		ctx,
		codexReceiptDescriptorRequestFromProgressV0(request),
	)
	if err != nil {
		return nil, err
	}
	observations := make([]orquestacionnucleoapp.AgentProgressObservationV0, 0, len(descriptors))
	for _, descriptor := range descriptors {
		observation, ready, err := source.observationFromDescriptorV0(ctx, request, descriptor)
		if err != nil {
			return nil, err
		}
		if !ready {
			continue
		}
		observations = append(observations, observation)
	}
	return observations, nil
}

func (source CodexProgressObservationSourceV0) validateV0() error {
	switch {
	case source.Store == nil:
		return fmt.Errorf("codex_progress_observation_source: store requerido")
	case source.ProcessRegistry == nil:
		return fmt.Errorf("codex_progress_observation_source: process_registry requerido")
	case source.State == nil:
		return fmt.Errorf("codex_progress_observation_source: state requerido")
	default:
		return nil
	}
}

func codexReceiptDescriptorRequestFromProgressV0(
	request orquestacionnucleoapp.AgentProgressObservationRequestV0,
) CodexReceiptDescriptorRequestV0 {
	return CodexReceiptDescriptorRequestV0{
		RunID:          strings.TrimSpace(request.Run.RunID),
		StartedAgents:  compactCodexDeliveryRefsV0(request.Run.StartedAgents),
		Deliveries:     compactCodexDeliveryRefsV0(request.Run.Deliveries),
		PhaseArtifacts: compactCodexDeliveryRefsV0(request.Run.PhaseArtifacts),
		CorrelationID:  strings.TrimSpace(request.CorrelationID),
		EvidenceRefs:   compactCodexDeliveryRefsV0(request.EvidenceRefs),
	}
}

func (source CodexProgressObservationSourceV0) observationFromDescriptorV0(
	ctx context.Context,
	request orquestacionnucleoapp.AgentProgressObservationRequestV0,
	descriptor CodexReceiptDescriptorV0,
) (orquestacionnucleoapp.AgentProgressObservationV0, bool, error) {
	ackReady, err := codexProgressAckReadyV0(descriptor)
	if err != nil || ackReady {
		return orquestacionnucleoapp.AgentProgressObservationV0{}, false, err
	}
	record, err := source.ProcessRegistry.ResolveAgentProcessV0(
		ctx,
		strings.TrimSpace(descriptor.RunID),
		strings.TrimSpace(descriptor.AgentRef),
	)
	if err != nil {
		return orquestacionnucleoapp.AgentProgressObservationV0{}, false, err
	}
	snapshot, err := source.snapshotV0(record)
	if err != nil {
		return orquestacionnucleoapp.AgentProgressObservationV0{}, false, err
	}
	state, err := source.State.ObserveCodexProgressV0(ctx, CodexProgressSampleV0{
		RunID:                record.RunID,
		AgentRequestID:       record.AgentRequestID,
		ProcessRef:           record.ProcessRef,
		Signature:            codexProgressSignatureV0(descriptor),
		EvidenceRefs:         codexProgressEvidenceRefsV0(record),
		ObservedAt:           time.Now().UTC(),
		MinUnchangedInterval: source.MinUnchangedSampleInterval,
	})
	if err != nil {
		return orquestacionnucleoapp.AgentProgressObservationV0{}, false, err
	}
	if !state.SampleAccepted {
		return orquestacionnucleoapp.AgentProgressObservationV0{}, false, nil
	}
	report, issues := orquestaruntime.BuildAgentProgressReportFromHeartbeatV0(
		codexProgressReportIDV0(state.Current),
		snapshot,
		state.Previous,
		state.Current,
		source.Policy,
	)
	if len(issues) > 0 {
		return orquestacionnucleoapp.AgentProgressObservationV0{}, false, fmt.Errorf(
			"codex_progress_observation: %s:%s",
			issues[0].Code,
			issues[0].Field,
		)
	}
	report = source.reportWithBudgetV0(report, state)
	report = codexProgressReportWithProcessFailureContextV0(descriptor, report)
	decisionRequired := codexProgressReportDecisionRequiredV0(report)
	if report.Status == orquestaruntime.AgentProgressingV0 &&
		!codexProgressBudgetPolicyEnabledV0(source.BudgetPolicy) &&
		!decisionRequired {
		return orquestacionnucleoapp.AgentProgressObservationV0{}, false, nil
	}
	if decisionRequired && codexProgressReportAlreadyHandledV0(request.Run, report) {
		return orquestacionnucleoapp.AgentProgressObservationV0{}, false, nil
	}
	if decisionRequired {
		if err := source.State.MarkCodexProgressReportedV0(ctx, CodexProgressReportMarkV0{
			RunID:          record.RunID,
			AgentRequestID: record.AgentRequestID,
			ProcessRef:     record.ProcessRef,
			Signature:      state.Signature,
			Status:         report.Status,
		}); err != nil {
			return orquestacionnucleoapp.AgentProgressObservationV0{}, false, err
		}
	}
	return codexProgressObservationV0(descriptor, report, decisionRequired), true, nil
}

func codexProgressAckReadyV0(
	descriptor CodexReceiptDescriptorV0,
) (bool, error) {
	_, ready, err := (CodexDeliveryObservationSourceV0{}).observationFromDescriptorV0(context.Background(), descriptor)
	return ready, err
}

func (source CodexProgressObservationSourceV0) snapshotV0(
	record orquestacionnucleoapp.AgentProcessRegistryRecordV0,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	if source.SnapshotSource != nil {
		return source.SnapshotSource.SnapshotV0(record.ProcessRef)
	}
	return orquestaruntime.ProcessRuntimeSnapshotV0{
		SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
		ProcessRef:    strings.TrimSpace(record.ProcessRef),
		SessionRef:    strings.TrimSpace(record.SessionRef),
		LaunchRef:     strings.TrimSpace(record.LaunchRef),
		Status:        orquestaruntime.ProcessRuntimeRunningV0,
	}, nil
}

func codexProgressObservationV0(
	descriptor CodexReceiptDescriptorV0,
	report orquestaruntime.AgentProgressReportV0,
	decisionRequired bool,
) orquestacionnucleoapp.AgentProgressObservationV0 {
	task := descriptor.Spec.AgentPacket.Task
	return orquestacionnucleoapp.AgentProgressObservationV0{
		CandidateRef:     "progress-candidate-ref-" + report.ReportID,
		Report:           report,
		PhaseID:          descriptor.Spec.AgentPacket.Phase,
		TaskRef:          strings.TrimSpace(task.TaskRef),
		DecisionRequired: decisionRequired,
		EvidenceRefs:     compactCodexDeliveryRefsV0(report.EvidenceRefs),
	}
}

func codexProgressReportDecisionRequiredV0(report orquestaruntime.AgentProgressReportV0) bool {
	if report.DecisionRequired {
		return true
	}
	return report.Status == orquestaruntime.AgentStalledV0 ||
		report.Status == orquestaruntime.AgentLoopDetectedV0 ||
		report.Status == orquestaruntime.AgentStoppedV0
}

func codexProgressSignatureV0(
	descriptor CodexReceiptDescriptorV0,
) string {
	dir := filepath.Dir(strings.TrimSpace(descriptor.AckPath))
	files := []string{
		orquestaruntimecodex.CodexStdoutFileNameV0,
		orquestaruntimecodex.CodexStderrFileNameV0,
		orquestaruntimecodex.CodexLastMessageFileNameV0,
		orquestaruntimecodex.CodexAgentAckFileNameV0,
	}
	var builder strings.Builder
	for _, name := range files {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil {
			builder.WriteString(name)
			builder.WriteString(":missing;")
			continue
		}
		builder.WriteString(name)
		builder.WriteString(":")
		builder.WriteString(strconv.FormatInt(info.Size(), 10))
		builder.WriteString(":")
		builder.WriteString(strconv.FormatInt(info.ModTime().UnixNano(), 10))
		builder.WriteString(";")
	}
	return builder.String()
}

func codexProgressEvidenceRefsV0(
	record orquestacionnucleoapp.AgentProcessRegistryRecordV0,
) []string {
	return compactCodexDeliveryRefsV0([]string{
		"progress-evidence-ref-" + strings.TrimSpace(record.AgentRequestID),
		strings.TrimSpace(record.SessionRef),
	})
}

func codexProgressReportIDV0(
	current orquestaruntime.AgentProgressHeartbeatV0,
) string {
	return fmt.Sprintf("agent-progress-report-ref-%s-%06d", current.AgentRequestID, current.TickCounter)
}
