package orquestaruntime

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

type ProcessRuntimeConnectorV0 struct {
	mu          sync.Mutex
	nextProcess uint64
	nextSession uint64
	nextLaunch  uint64
	nextStop    uint64
	processes   map[string]*processRuntimeRecordV0
}

type processRuntimeRecordV0 struct {
	cmd        *exec.Cmd
	processRef string
	sessionRef string
	launchRef  string
	stopRef    string
	status     ProcessRuntimeStatusV0
	done       chan struct{}
}

func NewProcessRuntimeConnectorV0() *ProcessRuntimeConnectorV0 {
	return &ProcessRuntimeConnectorV0{
		processes: map[string]*processRuntimeRecordV0{},
	}
}

func (c *ProcessRuntimeConnectorV0) LaunchV0(
	ctx context.Context,
	req ProcessRuntimeLaunchRequestV0,
) (ProcessRuntimeSnapshotV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return ProcessRuntimeSnapshotV0{}, processRuntimeErrorV0(ProcessRuntimeContextDoneV0, "context")
	}
	if err := validateProcessRuntimeLaunchRequestV0(req); err != nil {
		return ProcessRuntimeSnapshotV0{}, err
	}

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
		cmd:        cmd,
		processRef: c.nextProcessRefLocked(),
		sessionRef: c.nextSessionRefLocked(),
		launchRef:  c.nextLaunchRefLocked(),
		status:     ProcessRuntimeRunningV0,
		done:       make(chan struct{}),
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

func (c *ProcessRuntimeConnectorV0) StopV0(
	ctx context.Context,
	processRef string,
) (ProcessRuntimeSnapshotV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateProcessRuntimeRefV0(processRef); err != nil {
		return ProcessRuntimeSnapshotV0{}, err
	}

	record, snapshot, alreadyStopped, err := c.recordForStopV0(processRef)
	if err != nil {
		return ProcessRuntimeSnapshotV0{}, err
	}
	if alreadyStopped {
		return snapshot, nil
	}

	c.signalProcessStopV0(record)

	select {
	case <-record.done:
		return c.SnapshotV0(processRef)
	case <-ctx.Done():
		c.killProcessV0(record)
		select {
		case <-record.done:
			return c.SnapshotV0(processRef)
		default:
			return ProcessRuntimeSnapshotV0{}, processRuntimeErrorV0(ProcessRuntimeContextDoneV0, "context")
		}
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
	snapshot := record.snapshot()
	return record, snapshot, record.status == ProcessRuntimeStoppedV0, nil
}

func (c *ProcessRuntimeConnectorV0) waitProcessV0(record *processRuntimeRecordV0) {
	_ = record.cmd.Wait()

	c.mu.Lock()
	record.status = ProcessRuntimeStoppedV0
	c.mu.Unlock()
	close(record.done)
}

func (c *ProcessRuntimeConnectorV0) signalProcessStopV0(record *processRuntimeRecordV0) {
	if record.cmd == nil || record.cmd.Process == nil {
		return
	}
	if err := record.cmd.Process.Signal(os.Interrupt); err != nil {
		c.killProcessV0(record)
	}
}

func (c *ProcessRuntimeConnectorV0) killProcessV0(record *processRuntimeRecordV0) {
	if record.cmd == nil || record.cmd.Process == nil {
		return
	}
	_ = record.cmd.Process.Kill()
}

func (c *ProcessRuntimeConnectorV0) ensureProcessesLocked() {
	if c.processes == nil {
		c.processes = map[string]*processRuntimeRecordV0{}
	}
}

func (c *ProcessRuntimeConnectorV0) nextProcessRefLocked() string {
	c.nextProcess++
	return fmt.Sprintf("process-ref-v0-%06d", c.nextProcess)
}

func (c *ProcessRuntimeConnectorV0) nextLaunchRefLocked() string {
	c.nextLaunch++
	return fmt.Sprintf("launch-ref-v0-%06d", c.nextLaunch)
}

func (c *ProcessRuntimeConnectorV0) nextSessionRefLocked() string {
	c.nextSession++
	return fmt.Sprintf("session-ref-v0-%06d", c.nextSession)
}

func (c *ProcessRuntimeConnectorV0) nextStopRefLocked() string {
	c.nextStop++
	return fmt.Sprintf("stop-ref-v0-%06d", c.nextStop)
}

func (r *processRuntimeRecordV0) snapshot() ProcessRuntimeSnapshotV0 {
	return ProcessRuntimeSnapshotV0{
		SchemaVersion: ProcessRuntimeConnectorVersionV0,
		ProcessRef:    r.processRef,
		SessionRef:    r.sessionRef,
		LaunchRef:     r.launchRef,
		StopRef:       r.stopRef,
		Status:        r.status,
	}
}
