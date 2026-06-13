package main

import (
	"context"
	"strings"
	"sync"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaserver "orquesta/modulos/orquesta-server"
)

type serverSupervisorWakeupRelayV0 struct {
	mu      sync.RWMutex
	request func(string) bool
}

func (relay *serverSupervisorWakeupRelayV0) bindRuntimeV0(runtime *orquestaserver.RuntimeV0) {
	if relay == nil {
		return
	}
	relay.mu.Lock()
	defer relay.mu.Unlock()
	if runtime == nil {
		relay.request = nil
		return
	}
	relay.request = runtime.RequestSupervisorWakeupV0
}

func (relay *serverSupervisorWakeupRelayV0) requestV0(cause string) bool {
	if relay == nil {
		return false
	}
	relay.mu.RLock()
	request := relay.request
	relay.mu.RUnlock()
	if request == nil {
		return false
	}
	return request(cause)
}

type serverWakeupRunStoreV0 struct {
	inner  orquestacionnucleoapp.RunStorePortV0
	wakeup *serverSupervisorWakeupRelayV0
}

func (store serverWakeupRunStoreV0) LoadRunV0(
	ctx context.Context,
	runRef string,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	return store.inner.LoadRunV0(ctx, runRef)
}

func (store serverWakeupRunStoreV0) SaveRunV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) error {
	if err := store.inner.SaveRunV0(ctx, run); err != nil {
		return err
	}
	store.wakeup.requestV0("run_store_saved")
	return nil
}

type serverWakeupRunQueueV0 struct {
	inner  orquestarunqueue.RunQueuePortV0
	wakeup *serverSupervisorWakeupRelayV0
}

func (queue serverWakeupRunQueueV0) ListRunSchedulingCandidatesV0(
	ctx context.Context,
	request orquestarunqueue.RunQueueReadRequestV0,
) ([]orquestarunqueue.RunSchedulingCandidateV0, error) {
	return queue.inner.ListRunSchedulingCandidatesV0(ctx, request)
}

func (queue serverWakeupRunQueueV0) SetRunPriorityV0(
	ctx context.Context,
	command orquestarunqueue.RunQueuePriorityCommandV0,
) (orquestarunqueue.RunSchedulingCandidateV0, error) {
	candidate, err := queue.inner.SetRunPriorityV0(ctx, command)
	if err != nil {
		return candidate, err
	}
	if serverRunQueueCandidateShouldWakeSupervisorV0(candidate) {
		queue.wakeup.requestV0("run_queue_ready")
	}
	return candidate, nil
}

func serverRunQueueCandidateShouldWakeSupervisorV0(
	candidate orquestarunqueue.RunSchedulingCandidateV0,
) bool {
	status := strings.ToLower(strings.TrimSpace(candidate.Status))
	return status == "" || status == orquestarunqueue.RunStatusReadyV0
}

type serverWakeupRunControlV0 struct {
	inner  orquestaruncontrol.RunControlPortV0
	wakeup *serverSupervisorWakeupRelayV0
}

func (control serverWakeupRunControlV0) ReadRunControlStateV0(
	ctx context.Context,
	request orquestaruncontrol.RunControlReadRequestV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	return control.inner.ReadRunControlStateV0(ctx, request)
}

func (control serverWakeupRunControlV0) PauseRunV0(
	ctx context.Context,
	command orquestaruncontrol.PauseRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	return control.inner.PauseRunV0(ctx, command)
}

func (control serverWakeupRunControlV0) ResumeRunV0(
	ctx context.Context,
	command orquestaruncontrol.ResumeRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	state, err := control.inner.ResumeRunV0(ctx, command)
	if err == nil {
		control.wakeup.requestV0("run_control_resume")
	}
	return state, err
}

func (control serverWakeupRunControlV0) StopRunV0(
	ctx context.Context,
	command orquestaruncontrol.StopRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	state, err := control.inner.StopRunV0(ctx, command)
	if err == nil {
		control.wakeup.requestV0("run_control_stop")
	}
	return state, err
}

func (control serverWakeupRunControlV0) CancelRunV0(
	ctx context.Context,
	command orquestaruncontrol.CancelRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	state, err := control.inner.CancelRunV0(ctx, command)
	if err == nil {
		control.wakeup.requestV0("run_control_cancel")
	}
	return state, err
}

func (control serverWakeupRunControlV0) RecordRunCheckpointV0(
	ctx context.Context,
	command orquestaruncontrol.RecordRunCheckpointCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	state, err := control.inner.RecordRunCheckpointV0(ctx, command)
	if err == nil {
		control.wakeup.requestV0("run_control_checkpoint")
	}
	return state, err
}

func (control serverWakeupRunControlV0) CompleteRunControlV0(
	ctx context.Context,
	command orquestaruncontrol.CompleteRunControlCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	return control.inner.CompleteRunControlV0(ctx, command)
}
