package codex

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type processIdentityState uint8

const (
	processIdentityGone processIdentityState = iota
	processIdentityAlive
	processIdentityMismatch
)

var errOwnerLockBusy = errors.New(CodeProcessOwnershipBusy)

func (adapter *Adapter) processControlsEnabled() bool {
	return adapter != nil && adapter.config.RuntimeScope != "" && platformProcessControlSupported()
}

func (adapter *Adapter) acquireOwnerLock(runPath string) (*os.File, error) {
	lockPath := path.Join(runPath, ownerLockFileName)
	if info, err := adapter.root.Lstat(lockPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
			return nil, &Error{Code: CodeProcessOwnershipInvalid}
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, &Error{Code: CodeProcessOwnershipInvalid, Cause: err}
	}
	file, err := adapter.root.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, &Error{Code: CodeProcessOwnershipInvalid, Cause: err}
	}
	fail := func(cause error) (*os.File, error) {
		_ = file.Close()
		return nil, cause
	}
	if err := file.Chmod(0o600); err != nil {
		return fail(&Error{Code: CodeProcessOwnershipInvalid, Cause: err})
	}
	locked, err := platformTryOwnerLock(file)
	if err != nil {
		return fail(&Error{Code: CodeProcessOwnershipInvalid, Cause: err})
	}
	if !locked {
		return fail(&Error{Code: CodeProcessOwnershipBusy, Cause: errOwnerLockBusy})
	}
	if !platformOwnerLockCLOEXEC(file) {
		platformUnlockOwner(file)
		return fail(&Error{Code: CodeProcessOwnershipInvalid})
	}
	return file, nil
}

func releaseOwnerLock(file *os.File) {
	if file == nil {
		return
	}
	platformUnlockOwner(file)
	_ = file.Close()
}

func (adapter *Adapter) releaseProcessOwnershipLocked(state *executionState) {
	if state == nil {
		return
	}
	releaseOwnerLock(state.ownerLock)
	state.ownerLock = nil
}

func (adapter *Adapter) adoptPersistedProcessLocked(state *executionState) (bool, error) {
	if !adapter.processControlsEnabled() {
		return false, nil
	}
	record, found, err := adapter.processRecordForState(state)
	if err != nil || !found {
		return false, err
	}
	owner, err := adapter.acquireOwnerLock(state.runPath)
	if err != nil {
		return false, err
	}
	treeGone, inspectErr := inspectProcessTree(record)
	if inspectErr != nil {
		releaseOwnerLock(owner)
		return false, inspectErr
	}
	state.process, state.ownerLock = &record, owner
	proof, proofFound, proofErr := adapter.loadWinningStopSignalProof(state.runPath)
	if proofErr != nil {
		adapter.releaseProcessOwnershipLocked(state)
		return false, proofErr
	}
	if proofFound {
		state.stopProof.Store(&proof)
	}
	if treeGone {
		if proofFound {
			if err := adapter.finishStoppedProcessLocked(state, proof); err != nil {
				adapter.releaseProcessOwnershipLocked(state)
				return false, err
			}
			return false, nil
		}
		// Presence of a request is not evidence that its signal happened. Do
		// not collapse unreadable journal state into absence either.
		if _, requestErr := adapter.hasDurableStopRequest(state.runPath); requestErr != nil {
			adapter.releaseProcessOwnershipLocked(state)
			return false, requestErr
		}
		adapter.releaseProcessOwnershipLocked(state)
		return false, nil
	}
	state.status = ports.AgentRunning
	return true, nil
}

func inspectProcessTree(record processRecord) (bool, error) {
	identity, err := platformInspectProcess(record)
	if err != nil {
		return false, &Error{Code: CodeProcessInspectionFailed, Cause: err}
	}
	if identity == processIdentityMismatch {
		return false, &Error{Code: CodeProcessIdentityMismatch}
	}
	members, err := processGroupMemberCount(record.PGID)
	if err != nil {
		return false, err
	}
	return members == 0, nil
}

func signalProcessTree(record processRecord, mode ports.AgentStopMode) error {
	gone, err := inspectProcessTree(record)
	if err != nil {
		return err
	}
	if gone {
		return os.ErrProcessDone
	}
	signal := syscall.Signal(15)
	if mode == ports.AgentStopForced {
		signal = syscall.Signal(9)
	}
	process, err := os.FindProcess(-record.PGID)
	if err == nil {
		err = process.Signal(signal)
	}
	if errors.Is(err, os.ErrProcessDone) {
		return os.ErrProcessDone
	}
	if err != nil {
		return &Error{Code: CodeProcessSignalFailed, Cause: err}
	}
	return nil
}

func processGroupMemberCount(pgid int) (int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, &Error{Code: CodeProcessInspectionFailed, Cause: err}
	}
	members := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}
		payload, err := os.ReadFile(path.Join("/proc", entry.Name(), "stat"))
		if err != nil {
			// Processes outside this adapter can disappear or be unreadable
			// during the scan. Owned group members run as this process and remain
			// readable; unrelated entries must not break exact-tree inspection.
			continue
		}
		state, processGroup, err := parseProcessStat(payload)
		if err != nil {
			continue
		}
		if processGroup == pgid && state != "Z" && state != "X" {
			members++
		}
	}
	return members, nil
}

func parseProcessStat(payload []byte) (string, int, error) {
	end := strings.LastIndex(string(payload), ")")
	if end < 0 || end+1 >= len(payload) {
		return "", 0, errors.New("invalid proc stat")
	}
	fields := strings.Fields(string(payload[end+1:]))
	if len(fields) <= 2 {
		return "", 0, errors.New("short proc stat")
	}
	pgid, err := strconv.Atoi(fields[2])
	if err != nil || pgid <= 0 {
		return "", 0, errors.New("invalid process group")
	}
	return fields[0], pgid, nil
}

func (adapter *Adapter) discoverPersistedProcessesLocked() error {
	names, err := adapter.stopJournalNames("executions")
	if err != nil {
		return err
	}
	sort.Strings(names)
	discoveryErrors := make([]error, 0)
	for _, name := range names {
		runPath := path.Join("executions", name)
		if err := adapter.discoverPersistedProcessLocked(runPath); err != nil {
			discoveryErrors = append(discoveryErrors, err)
		}
	}
	return errors.Join(discoveryErrors...)
}

func (adapter *Adapter) discoverPersistedProcessLocked(runPath string) error {
	process, found, err := adapter.readProcessRecord(runPath)
	if err != nil || !found {
		return err
	}
	executionRef, err := goal.NewExecutionRef(process.ExecutionRef)
	if err != nil {
		return &Error{Code: CodeProcessOwnershipInvalid, Cause: err}
	}
	if existing := adapter.executions[executionRef.String()]; existing != nil && existing.process != nil {
		return nil
	}
	launch, expectedPath, found, err := adapter.loadLaunchRecord(executionRef)
	if err != nil {
		return err
	}
	if !found || expectedPath != runPath || launch.RequestHash != process.RequestHash {
		return &Error{Code: CodeProcessOwnershipInvalid}
	}
	receipt, err := launch.receipt(executionRef)
	if err != nil {
		return err
	}
	if _, terminalFound, err := adapter.loadCausalTerminal(runPath, launch.RequestHash, launch.SpecHash, launch.MaxOutputBytes); err != nil {
		return err
	} else if terminalFound {
		return nil
	}
	state := &executionState{
		requestHash: launch.RequestHash, terminalRequestHash: launch.RequestHash,
		receipt: receipt, maxOutput: launch.MaxOutputBytes, runPath: runPath, status: ports.AgentPending,
	}
	adopted, err := adapter.adoptPersistedProcessLocked(state)
	if err != nil {
		return err
	}
	if adopted {
		adapter.executions[executionRef.String()] = state
	}
	return nil
}

func pendingStopReceipt(request ports.AgentStopRequest) ports.AgentStopReceipt {
	return newStopReceipt(request, ports.AgentStopPending, "", time.Time{})
}

func newStopReceipt(
	request ports.AgentStopRequest,
	status ports.AgentStopStatus,
	receiptRef string,
	confirmedAt time.Time,
) ports.AgentStopReceipt {
	return ports.AgentStopReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: request.SpecHash,
		ProviderRef: request.ProviderRef, ModelRef: request.ModelRef, AgentRef: request.AgentRef,
		ExternalRef: request.ExternalRef, Mode: request.Mode, IdempotencyKey: request.IdempotencyKey,
		Status: status, ReceiptRef: receiptRef, ConfirmedAt: confirmedAt,
	}
}

func unsupportedStopReceipt(request ports.AgentStopRequest) ports.AgentStopReceipt {
	receipt := pendingStopReceipt(request)
	receipt.Status = ports.AgentStopUnsupported
	return receipt
}
