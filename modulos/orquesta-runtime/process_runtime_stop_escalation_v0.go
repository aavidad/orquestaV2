package orquestaruntime

import (
	"context"
	"errors"
	"os"
	"time"
)

func (c *ProcessRuntimeConnectorV0) signalProcessStopV0(
	record *processRuntimeRecordV0,
	signal os.Signal,
) (ProcessRuntimeStopReasonCodeV0, bool) {
	process := processRuntimeProcessHandleV0(record)
	if process == nil {
		return ProcessRuntimeStopSignalNotSupportedV0, true
	}
	if signal == nil {
		signal = os.Interrupt
	}
	if err := process.Signal(signal); err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			return ProcessRuntimeStopAlreadyStoppedV0, false
		}
		return ProcessRuntimeStopSignalNotSupportedV0, true
	}
	return ProcessRuntimeStopCooperativeSignalSentV0, false
}

func (c *ProcessRuntimeConnectorV0) killProcessV0(record *processRuntimeRecordV0) error {
	process := processRuntimeProcessHandleV0(record)
	if process == nil {
		return nil
	}
	if err := process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}
	return nil
}

func (c *ProcessRuntimeConnectorV0) escalateProcessStopV0(
	record *processRuntimeRecordV0,
	killWait time.Duration,
) (ProcessRuntimeSnapshotV0, error) {
	if err := c.killProcessV0(record); err != nil {
		c.markStopReasonV0(record, ProcessRuntimeStopKillFailedV0)
		return ProcessRuntimeSnapshotV0{}, processRuntimeErrorV0(ProcessRuntimeKillFallidoV0, "process")
	}
	timer := time.NewTimer(killWait)
	defer timer.Stop()
	select {
	case <-record.done:
		return c.SnapshotV0(record.processRef)
	case <-timer.C:
		return ProcessRuntimeSnapshotV0{}, processRuntimeErrorV0(ProcessRuntimeStopFallidoV0, "kill_wait")
	}
}

func (c *ProcessRuntimeConnectorV0) markStopReasonV0(
	record *processRuntimeRecordV0,
	reason ProcessRuntimeStopReasonCodeV0,
) {
	if reason == "" {
		return
	}
	c.mu.Lock()
	if record.status != ProcessRuntimeStoppedV0 {
		record.stopReason = reason
	}
	c.mu.Unlock()
}

func processRuntimeContextDeadlineV0(ctx context.Context, fallback time.Duration) time.Time {
	if deadline, ok := ctx.Deadline(); ok {
		return deadline
	}
	return time.Now().Add(fallback)
}
