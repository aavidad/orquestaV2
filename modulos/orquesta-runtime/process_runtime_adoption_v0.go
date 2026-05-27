package orquestaruntime

import (
	"context"
	"os"
	"strings"
)

func (c *ProcessRuntimeConnectorV0) AdoptProcessV0(
	ctx context.Context,
	snapshot ProcessRuntimeSnapshotV0,
) (ProcessRuntimeSnapshotV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return ProcessRuntimeSnapshotV0{}, processRuntimeErrorV0(ProcessRuntimeContextDoneV0, "context")
	}
	snapshot = normalizeProcessRuntimeAdoptionSnapshotV0(snapshot)
	if err := validateProcessRuntimeAdoptionSnapshotV0(snapshot); err != nil {
		return ProcessRuntimeSnapshotV0{}, err
	}
	process, err := os.FindProcess(snapshot.PID)
	if err != nil {
		return ProcessRuntimeSnapshotV0{}, processRuntimeErrorV0(ProcessRuntimeNoEncontradoV0, "pid")
	}
	status := ProcessRuntimeRunningV0
	if !processRuntimeProcessAliveV0(process) {
		status = ProcessRuntimeStoppedV0
	}
	c.mu.Lock()
	c.ensureProcessesLocked()
	if existing, ok := c.processes[snapshot.ProcessRef]; ok {
		current := existing.snapshot()
		c.mu.Unlock()
		return current, nil
	}
	record := &processRuntimeRecordV0{
		process:       process,
		pid:           snapshot.PID,
		processRef:    snapshot.ProcessRef,
		sessionRef:    snapshot.SessionRef,
		launchRef:     snapshot.LaunchRef,
		status:        status,
		launchReceipt: snapshot.LaunchReceipt,
		done:          make(chan struct{}),
	}
	if record.status == ProcessRuntimeStoppedV0 {
		close(record.done)
	} else {
		go c.watchAdoptedProcessV0(record)
	}
	c.processes[record.processRef] = record
	out := record.snapshot()
	c.mu.Unlock()
	return out, nil
}

func normalizeProcessRuntimeAdoptionSnapshotV0(
	snapshot ProcessRuntimeSnapshotV0,
) ProcessRuntimeSnapshotV0 {
	snapshot.SchemaVersion = strings.TrimSpace(snapshot.SchemaVersion)
	if snapshot.SchemaVersion == "" {
		snapshot.SchemaVersion = ProcessRuntimeConnectorVersionV0
	}
	snapshot.ProcessRef = strings.TrimSpace(snapshot.ProcessRef)
	snapshot.SessionRef = strings.TrimSpace(snapshot.SessionRef)
	snapshot.LaunchRef = strings.TrimSpace(snapshot.LaunchRef)
	if snapshot.Status == "" {
		snapshot.Status = ProcessRuntimeRunningV0
	}
	return snapshot
}

func validateProcessRuntimeAdoptionSnapshotV0(
	snapshot ProcessRuntimeSnapshotV0,
) error {
	if snapshot.PID <= 0 {
		return processRuntimeErrorV0(ProcessRuntimeRefInvalidaV0, "pid")
	}
	if issues := validateProcessRuntimeSnapshotV0(snapshot); len(issues) > 0 {
		return issues[0]
	}
	return nil
}

func processRuntimeProcessHandleV0(record *processRuntimeRecordV0) *os.Process {
	if record == nil {
		return nil
	}
	if record.process != nil {
		return record.process
	}
	if record.cmd != nil && record.cmd.Process != nil {
		return record.cmd.Process
	}
	return nil
}
