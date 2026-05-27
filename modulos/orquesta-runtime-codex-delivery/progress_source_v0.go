package orquestaruntimecodexdelivery

import (
	"context"
	"fmt"
	"strings"
	"time"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
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
	EmitProgressing            bool
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

func (source CodexProgressObservationSourceV0) observationFromDescriptorV0(
	ctx context.Context,
	request orquestacionnucleoapp.AgentProgressObservationRequestV0,
	descriptor CodexReceiptDescriptorV0,
) (orquestacionnucleoapp.AgentProgressObservationV0, bool, error) {
	if observation, ready, err := codexProgressFailedAckObservationV0(descriptor); err != nil || ready {
		if err != nil || codexProgressReportAlreadyHandledV0(request.Run, observation.Report) {
			return orquestacionnucleoapp.AgentProgressObservationV0{}, false, err
		}
		return codexProgressObservationForCurrentRunPhaseV0(request, observation), true, nil
	}
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
		if codexProgressAgentProcessMissingV0(err) {
			return orquestacionnucleoapp.AgentProgressObservationV0{}, false, nil
		}
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
		ActionSignature:      codexProgressActionSignatureV0(descriptor),
		EvidenceRefs:         codexProgressEvidenceRefsV0(record),
		ObservedAt:           orquestaruntime.NowUTCV0(nil),
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
	report = codexProgressReportWithRunningNoVisibleSignalV0(descriptor, snapshot, report)
	report = codexProgressReportWithProcessFailureContextV0(descriptor, report)
	report = codexProgressReportWithRepeatedActionEvidenceV0(report)
	decisionRequired := codexProgressReportDecisionRequiredV0(report)
	if report.Status == orquestaruntime.AgentProgressingV0 &&
		!codexProgressBudgetPolicyEnabledV0(source.BudgetPolicy) &&
		!source.EmitProgressing &&
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
	return codexProgressObservationForCurrentRunPhaseV0(
		request,
		codexProgressObservationV0(descriptor, report, decisionRequired),
	), true, nil
}
