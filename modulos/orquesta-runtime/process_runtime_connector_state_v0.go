package orquestaruntime

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

type processRuntimeRecordV0 struct {
	cmd           *exec.Cmd
	process       *os.Process
	pid           int
	processRef    string
	sessionRef    string
	launchRef     string
	stopRef       string
	status        ProcessRuntimeStatusV0
	stopReason    ProcessRuntimeStopReasonCodeV0
	stopDeadline  time.Time
	launchReceipt *ProcessRuntimeLaunchReceiptV0
	done          chan struct{}
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
	receiptRefs := ProcessRuntimeLaunchReceiptRefsV0(r.launchReceipt)
	return ProcessRuntimeSnapshotV0{
		SchemaVersion:     ProcessRuntimeConnectorVersionV0,
		ProcessRef:        r.processRef,
		SessionRef:        r.sessionRef,
		LaunchRef:         r.launchRef,
		PID:               r.pid,
		StopRef:           r.stopRef,
		Status:            r.status,
		LaunchReceipt:     r.launchReceipt,
		LaunchReceiptRefs: receiptRefs,
		StopReasonCode:    r.stopReason,
		StopGraceDeadline: processRuntimeDeadlineStringV0(r.stopDeadline),
	}
}

func processRuntimeDeadlineStringV0(deadline time.Time) string {
	if deadline.IsZero() {
		return ""
	}
	return deadline.UTC().Format(time.RFC3339Nano)
}
