package orquestadirector

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type progressiveRuntimeFakeOutboxDispatcherV0 struct {
	t           *testing.T
	runtimeFake *orquestaruntime.RuntimeFakeLifecycleV0
}

func newProgressiveRuntimeFakeOutboxDispatcherV0(t *testing.T) *progressiveRuntimeFakeOutboxDispatcherV0 {
	t.Helper()
	return &progressiveRuntimeFakeOutboxDispatcherV0{
		t:           t,
		runtimeFake: orquestaruntime.NewRuntimeFakeLifecycleV0(),
	}
}

func (d *progressiveRuntimeFakeOutboxDispatcherV0) mustDispatchOnly(
	outbox []orquestacoreworkflow.OutboxMessageV0,
) orquestaruntime.RuntimeFakeLifecycleSnapshotV0 {
	d.t.Helper()
	snapshot, err := d.dispatchOnly(outbox)
	if err != nil {
		d.t.Fatalf("dispatch runtime fake outbox: %v", err)
	}
	return snapshot
}

func (d *progressiveRuntimeFakeOutboxDispatcherV0) dispatchOnly(
	outbox []orquestacoreworkflow.OutboxMessageV0,
) (orquestaruntime.RuntimeFakeLifecycleSnapshotV0, error) {
	if len(outbox) != 1 {
		return orquestaruntime.RuntimeFakeLifecycleSnapshotV0{}, progressiveOutboxLenErrorV0(len(outbox))
	}
	return d.dispatch(outbox[0])
}

func (d *progressiveRuntimeFakeOutboxDispatcherV0) dispatch(
	message orquestacoreworkflow.OutboxMessageV0,
) (orquestaruntime.RuntimeFakeLifecycleSnapshotV0, error) {
	switch message.MessageType {
	case orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0:
		if err := validateProgressiveRuntimeOutboxEnvelopeV0(message); err != nil {
			return orquestaruntime.RuntimeFakeLifecycleSnapshotV0{}, err
		}
		inbound, err := progressiveLaunchInboundFromOutboxV0(message)
		if err != nil {
			return orquestaruntime.RuntimeFakeLifecycleSnapshotV0{}, err
		}
		return d.launch(inbound, message)
	case orquestacoreworkflow.OutboxMessageStopRuntimeAgentV0:
		if err := validateProgressiveRuntimeOutboxEnvelopeV0(message); err != nil {
			return orquestaruntime.RuntimeFakeLifecycleSnapshotV0{}, err
		}
		inbound, err := progressiveStopInboundFromOutboxV0(message)
		if err != nil {
			return orquestaruntime.RuntimeFakeLifecycleSnapshotV0{}, err
		}
		return d.stop(inbound, message)
	default:
		return orquestaruntime.RuntimeFakeLifecycleSnapshotV0{}, progressiveRuntimeFakeOutboxDispatchErrorV0{
			Code:        progressiveRuntimeFakeOutboxTipoNoSoportadoV0,
			Field:       "message_type",
			MessageType: message.MessageType,
			TargetPort:  message.TargetPort,
		}
	}
}

func (d *progressiveRuntimeFakeOutboxDispatcherV0) mustReportProgress(
	report orquestaruntime.AgentProgressReportV0,
) orquestaruntime.RuntimeFakeLifecycleSnapshotV0 {
	d.t.Helper()
	snapshot, err := d.runtimeFake.ReportProgressV0(report)
	if err != nil {
		d.t.Fatalf("ReportProgressV0: %v", err)
	}
	return snapshot
}

func (d *progressiveRuntimeFakeOutboxDispatcherV0) launch(
	inbound orquestaruntime.AgentLauncherInboundV0,
	message orquestacoreworkflow.OutboxMessageV0,
) (orquestaruntime.RuntimeFakeLifecycleSnapshotV0, error) {
	snapshot, err := d.runtimeFake.LaunchAgentV0(inbound)
	if err != nil {
		return orquestaruntime.RuntimeFakeLifecycleSnapshotV0{}, progressiveDispatchFailedErrorV0(message, "launch", err)
	}
	return snapshot, nil
}

func (d *progressiveRuntimeFakeOutboxDispatcherV0) stop(
	inbound orquestaruntime.AgentStopperInboundV0,
	message orquestacoreworkflow.OutboxMessageV0,
) (orquestaruntime.RuntimeFakeLifecycleSnapshotV0, error) {
	snapshot, err := d.runtimeFake.StopAgentV0(inbound)
	if err != nil {
		return orquestaruntime.RuntimeFakeLifecycleSnapshotV0{}, progressiveDispatchFailedErrorV0(message, "stop", err)
	}
	return snapshot, nil
}
