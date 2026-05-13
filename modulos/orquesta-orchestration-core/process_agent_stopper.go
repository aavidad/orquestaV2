package orquestacionnucleoapp

import (
	"context"
	"errors"
	"strings"

	orquestaagentprocessregistry "orquesta/modulos/orquesta-agent-process-registry"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type ProcessRuntimeStopPortV0 interface {
	StopV0(context.Context, string) (orquestaruntime.ProcessRuntimeSnapshotV0, error)
}

type ProcessRuntimeIdentitySnapshotPortV0 interface {
	SnapshotV0(string) (orquestaruntime.ProcessRuntimeSnapshotV0, error)
}

type ProcessAgentStopperV0 struct {
	Registry AgentProcessRegistryPortV0
	Runtime  ProcessRuntimeStopPortV0
}

var _ AgentStopperPortV0 = ProcessAgentStopperV0{}

func (stopper ProcessAgentStopperV0) StopAgentV0(
	ctx context.Context,
	inbound orquestaruntime.AgentStopperInboundV0,
) (AgentStopResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return AgentStopResultV0{}, err
	}
	if err := stopper.validateV0(inbound); err != nil {
		return AgentStopResultV0{}, err
	}
	payload := inbound.Payload
	record, err := stopper.Registry.ResolveAgentProcessV0(
		ctx,
		payload.RunID,
		payload.AgentRequestID,
	)
	if err != nil {
		return AgentStopResultV0{}, processAgentStopRegistryErrorV0(err)
	}
	if err := stopper.validateRegisteredProcessIdentityV0(record); err != nil {
		return AgentStopResultV0{}, err
	}
	snapshot, err := stopper.Runtime.StopV0(ctx, record.ProcessRef)
	if err != nil {
		return AgentStopResultV0{}, err
	}
	if snapshot.Status != orquestaruntime.ProcessRuntimeStoppedV0 {
		return AgentStopResultV0{}, errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"process_runtime.status",
			"process_runtime no confirmado stopped",
		)
	}
	if strings.TrimSpace(snapshot.StopRef) == "" {
		return AgentStopResultV0{}, errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"process_runtime.stop_ref",
			"stop_ref requerido",
		)
	}
	return processAgentStopResultV0(record, snapshot), nil
}

func processAgentStopRegistryErrorV0(err error) error {
	if err == nil {
		return nil
	}
	var coreErr ErrorV0
	if errors.As(err, &coreErr) {
		return err
	}
	var registryErr orquestaagentprocessregistry.ErrorV0
	if errors.As(err, &registryErr) {
		code := ErrNucleoOrquestacionStoreV0
		if registryErr.Code == orquestaagentprocessregistry.ErrAgentProcessRegistryInvalidV0 {
			code = ErrNucleoOrquestacionInvalidoV0
		}
		return errorV0(code, registryErr.Field, registryErr.Message)
	}
	return errorV0(ErrNucleoOrquestacionStoreV0, "agent_process_registry", err.Error())
}

func (stopper ProcessAgentStopperV0) validateV0(
	inbound orquestaruntime.AgentStopperInboundV0,
) error {
	if stopper.Registry == nil {
		return errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"agent_process_registry",
			"agent_process_registry requerido",
		)
	}
	if stopper.Runtime == nil {
		return errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"process_runtime",
			"process_runtime requerido",
		)
	}
	if issues := orquestaruntime.ValidateAgentStopperInboundV0(inbound); len(issues) > 0 {
		return errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"agent_stopper_inbound."+issues[0].Field,
			string(issues[0].Code),
		)
	}
	return nil
}

func (stopper ProcessAgentStopperV0) validateRegisteredProcessIdentityV0(
	record AgentProcessRecordV0,
) error {
	snapshotPort, ok := stopper.Runtime.(ProcessRuntimeIdentitySnapshotPortV0)
	if !ok {
		return nil
	}
	snapshot, err := snapshotPort.SnapshotV0(record.ProcessRef)
	if err != nil {
		return err
	}
	if strings.TrimSpace(snapshot.ProcessRef) != strings.TrimSpace(record.ProcessRef) {
		return processAgentStopIdentityMismatchV0("process_ref")
	}
	if strings.TrimSpace(record.SessionRef) != "" &&
		strings.TrimSpace(snapshot.SessionRef) != strings.TrimSpace(record.SessionRef) {
		return processAgentStopIdentityMismatchV0("session_ref")
	}
	if strings.TrimSpace(record.LaunchRef) != "" &&
		strings.TrimSpace(snapshot.LaunchRef) != "" &&
		strings.TrimSpace(snapshot.LaunchRef) != strings.TrimSpace(record.LaunchRef) {
		return processAgentStopIdentityMismatchV0("launch_ref")
	}
	return nil
}

func processAgentStopIdentityMismatchV0(field string) error {
	return errorV0(
		ErrNucleoOrquestacionInvalidoV0,
		"agent_process."+field,
		"agent_process identity mismatch",
	)
}

func processAgentStopResultV0(
	record AgentProcessRecordV0,
	snapshot orquestaruntime.ProcessRuntimeSnapshotV0,
) AgentStopResultV0 {
	return AgentStopResultV0{
		AgentRequestID:  strings.TrimSpace(record.AgentRequestID),
		ConfirmationRef: strings.TrimSpace(snapshot.StopRef),
		EvidenceRefs: compactStringsV0([]string{
			record.ProcessRef,
			record.SessionRef,
			record.LaunchRef,
			record.ReadinessRef,
			snapshot.ProcessRef,
			snapshot.SessionRef,
			snapshot.LaunchRef,
			snapshot.StopRef,
		}),
	}
}
