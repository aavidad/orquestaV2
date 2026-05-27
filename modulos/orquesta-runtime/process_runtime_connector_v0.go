package orquestaruntime

import (
	"context"
	"io"
	"os/exec"
	"sync"
	"time"
)

type ProcessRuntimeConnectorV0 struct {
	mu           sync.Mutex
	nextProcess  uint64
	nextSession  uint64
	nextLaunch   uint64
	nextStop     uint64
	processes    map[string]*processRuntimeRecordV0
	effectPolicy ProcessRuntimeEffectPolicyV0
}

func NewProcessRuntimeConnectorV0() *ProcessRuntimeConnectorV0 {
	return &ProcessRuntimeConnectorV0{
		processes:    map[string]*processRuntimeRecordV0{},
		effectPolicy: normalizeProcessRuntimeEffectPolicyV0(ProcessRuntimeEffectPolicyV0{}),
	}
}

func (c *ProcessRuntimeConnectorV0) LaunchV0(
	ctx context.Context,
	req ProcessRuntimeLaunchRequestV0,
) (ProcessRuntimeSnapshotV0, error) {
	policy := c.normalizedEffectPolicyV0()
	ctx, cancel := processRuntimeEffectContextV0(ctx, policy.LaunchTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return ProcessRuntimeSnapshotV0{}, processRuntimeErrorV0(ProcessRuntimeContextDoneV0, "context")
	}
	if err := validateProcessRuntimeLaunchRequestV0(req); err != nil {
		return ProcessRuntimeSnapshotV0{}, err
	}
	receipt := processRuntimeFinalizeLaunchReceiptForRequestV0(req)

	cmd := exec.Command(req.CommandPath, req.Args...)
	cmd.Dir = req.WorkingDir
	cmd.Env = processRuntimeExecEnvV0(req.Env)
	cmd.Stdin = nil
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		return ProcessRuntimeSnapshotV0{}, processRuntimeErrorV0(ProcessRuntimeLaunchFallidoV0, "command_path")
	}

	c.mu.Lock()
	c.ensureProcessesLocked()
	record := &processRuntimeRecordV0{
		cmd:           cmd,
		process:       cmd.Process,
		pid:           cmd.Process.Pid,
		processRef:    c.nextProcessRefLocked(),
		sessionRef:    c.nextSessionRefLocked(),
		launchRef:     c.nextLaunchRefLocked(),
		status:        ProcessRuntimeRunningV0,
		launchReceipt: &receipt,
		done:          make(chan struct{}),
	}
	c.processes[record.processRef] = record
	snapshot := record.snapshot()
	c.mu.Unlock()

	go c.waitProcessV0(record)

	return snapshot, nil
}

func processRuntimeExecEnvV0(env []string) []string {
	clean := make([]string, len(env))
	copy(clean, env)
	return clean
}

func processRuntimeFinalizeLaunchReceiptForRequestV0(
	req ProcessRuntimeLaunchRequestV0,
) ProcessRuntimeLaunchReceiptV0 {
	if req.LaunchReceipt == nil {
		return processRuntimeDirectLaunchReceiptV0(req)
	}
	return finalizeProcessRuntimeLaunchReceiptV0(*req.LaunchReceipt)
}

func (c *ProcessRuntimeConnectorV0) StopV0(
	ctx context.Context,
	processRef string,
) (ProcessRuntimeSnapshotV0, error) {
	policy := c.normalizedEffectPolicyV0()
	ctx, cancel := processRuntimeEffectContextV0(ctx, policy.StopTimeout)
	defer cancel()
	if err := validateProcessRuntimeRefV0(processRef); err != nil {
		return ProcessRuntimeSnapshotV0{}, err
	}
	graceDeadline := processRuntimeContextDeadlineV0(ctx, policy.StopTimeout)

	record, snapshot, alreadyStopped, err := c.recordForStopV0(processRef, graceDeadline)
	if err != nil {
		return ProcessRuntimeSnapshotV0{}, err
	}
	if alreadyStopped {
		return snapshot, nil
	}

	reason, escalate := c.signalProcessStopV0(record, policy.StopSignal)
	c.markStopReasonV0(record, reason)
	if escalate {
		return c.escalateProcessStopV0(record, policy.KillWait)
	}

	select {
	case <-record.done:
		return c.SnapshotV0(processRef)
	case <-ctx.Done():
		c.markStopReasonV0(record, ProcessRuntimeStopGraceTimeoutV0)
		return c.escalateProcessStopV0(record, policy.KillWait)
	}
}

func (c *ProcessRuntimeConnectorV0) SnapshotV0(
	processRef string,
) (ProcessRuntimeSnapshotV0, error) {
	if err := validateProcessRuntimeRefV0(processRef); err != nil {
		return ProcessRuntimeSnapshotV0{}, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.ensureProcessesLocked()
	record, ok := c.processes[processRef]
	if !ok {
		return ProcessRuntimeSnapshotV0{}, processRuntimeErrorV0(ProcessRuntimeNoEncontradoV0, "process_ref")
	}
	return record.snapshot(), nil
}

func (c *ProcessRuntimeConnectorV0) recordForStopV0(
	processRef string,
	graceDeadline time.Time,
) (*processRuntimeRecordV0, ProcessRuntimeSnapshotV0, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ensureProcessesLocked()
	record, ok := c.processes[processRef]
	if !ok {
		return nil, ProcessRuntimeSnapshotV0{}, false,
			processRuntimeErrorV0(ProcessRuntimeNoEncontradoV0, "process_ref")
	}
	if record.stopRef == "" {
		record.stopRef = c.nextStopRefLocked()
	}
	if record.status == ProcessRuntimeStoppedV0 {
		if record.stopReason == "" {
			record.stopReason = ProcessRuntimeStopAlreadyStoppedV0
		}
		snapshot := record.snapshot()
		return record, snapshot, true, nil
	}
	record.status = ProcessRuntimeStoppingV0
	record.stopReason = ProcessRuntimeStopCooperativeSignalSentV0
	record.stopDeadline = graceDeadline
	snapshot := record.snapshot()
	return record, snapshot, false, nil
}
