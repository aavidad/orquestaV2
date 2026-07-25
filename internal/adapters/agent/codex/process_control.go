package codex

import (
	"context"
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
	return adapter != nil && adapter.config.RuntimeScope != "" && platformProcessControlSupported() &&
		(adapter.cgroups != nil || adapter.config.allowLegacyProcessControlForTests)
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
	treeGone, inspectErr := adapter.inspectProcessTree(record)
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
	intent, intentFound, intentErr := adapter.loadWinningStopSignalIntent(state.runPath)
	if record.SchemaVersion == cgroupProcessSchemaVersion {
		identity, identityErr := platformInspectProcess(record)
		populated, populatedErr := adapter.cgroups.populated(record)
		if identityErr != nil || populatedErr != nil {
			adapter.releaseProcessOwnershipLocked(state)
			return false, errors.Join(identityErr, populatedErr)
		}
		if identity == processIdentityGone && intentErr == nil && intentFound &&
			intent.Mode == ports.AgentStopForced {
			intentProven := proofFound &&
				intent.RequestHash == proof.RequestHash &&
				intent.Idempotency == proof.Idempotency &&
				intent.Mode == proof.Mode && intent.Sequence == proof.Sequence
			if !intentProven {
				if err := adapter.signalForcedIntentLocked(
					state, record, state.runPath, intent,
				); err != nil {
					adapter.releaseProcessOwnershipLocked(state)
					return false, err
				}
				proof, proofFound = *state.stopProof.Load(), true
			}
			// A restart must finish the exact forced intent whether the previous
			// owner crashed before or after cgroup.kill and whether the leaf was
			// already empty when adoption began.
			if err := adapter.cgroups.drain(record, adapter.config.SupervisorStartTimeout); err != nil {
				adapter.releaseProcessOwnershipLocked(state)
				return false, err
			}
			if !proofFound {
				adapter.releaseProcessOwnershipLocked(state)
				return false, &Error{Code: CodeStopConflict}
			}
			if err := adapter.finishStoppedProcessLocked(state, proof); err != nil {
				adapter.releaseProcessOwnershipLocked(state)
				return false, err
			}
			return false, nil
		}
		if identity == processIdentityGone && populated {
			if err := adapter.cgroups.drain(record, adapter.config.SupervisorStartTimeout); err != nil {
				adapter.releaseProcessOwnershipLocked(state)
				return false, err
			}
			if err := adapter.recoverInterruptedExecutionLocked(state); err != nil {
				adapter.releaseProcessOwnershipLocked(state)
				return false, err
			}
			if err := adapter.cleanupTerminalCgroup(state); err != nil {
				return false, err
			}
			return false, nil
		}
	}
	if treeGone {
		if proofFound {
			requireProof := adapter.requireDurableStopProof
			if record.SchemaVersion == cgroupProcessSchemaVersion {
				requireProof = adapter.requireWinningStopProof
			}
			if err := requireProof(state.runPath, proof); err != nil {
				if intentErr != nil || !intentFound || intent.Mode != ports.AgentStopForced {
					adapter.releaseProcessOwnershipLocked(state)
					return false, errors.Join(err, intentErr)
				}
				if err := adapter.signalForcedIntentLocked(
					state, record, state.runPath, intent,
				); err != nil {
					adapter.releaseProcessOwnershipLocked(state)
					return false, err
				}
				proof = *state.stopProof.Load()
			}
			if err := adapter.finishStoppedProcessLocked(state, proof); err != nil {
				adapter.releaseProcessOwnershipLocked(state)
				return false, err
			}
			return false, nil
		}
		hasStopRequest, requestErr := adapter.hasDurableStopRequest(state.runPath)
		if requestErr != nil {
			adapter.releaseProcessOwnershipLocked(state)
			return false, requestErr
		}
		if hasStopRequest {
			if completion, result, found := adapter.loadCompletionProof(state); found &&
				completion.Cause == supervisorCauseStop {
				clearBytes(result)
				intent, intentFound, intentErr := adapter.loadWinningStopSignalIntent(state.runPath)
				if intentErr != nil || !intentFound {
					adapter.releaseProcessOwnershipLocked(state)
					return false, errors.Join(intentErr, &Error{Code: CodeStopConflict})
				}
				proof, persistErr := adapter.persistStopSignalProof(state.runPath, intent)
				if persistErr != nil {
					adapter.releaseProcessOwnershipLocked(state)
					return false, persistErr
				}
				state.stopProof.Store(&proof)
				if err := adapter.finishStoppedProcessLocked(state, proof); err != nil {
					adapter.releaseProcessOwnershipLocked(state)
					return false, err
				}
				return false, nil
			}
			adapter.releaseProcessOwnershipLocked(state)
			return false, nil
		}
		if _, result, completionFound := adapter.loadCompletionProof(state); completionFound {
			clearBytes(result)
			if err := adapter.finishSupervisedProcessLocked(state); err != nil {
				adapter.releaseProcessOwnershipLocked(state)
				return false, err
			}
			return false, nil
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
	if identity == processIdentityGone && record.SupervisorInstance != "" {
		// A recycled process group is not proof that it still belongs to this
		// execution. Schema V2 delegates exact descendant drainage to the
		// identity-bound supervisor and never signals a PGID after that leader
		// is gone.
		return true, nil
	}
	members, err := processGroupMemberCount(record.PGID)
	if err != nil {
		return false, err
	}
	return members == 0, nil
}

func (adapter *Adapter) inspectProcessTree(record processRecord) (bool, error) {
	if record.SchemaVersion != cgroupProcessSchemaVersion {
		if adapter == nil || !adapter.config.allowLegacyProcessControlForTests {
			return false, &Error{Code: CodeControlUnsupported}
		}
		return inspectProcessTree(record)
	}
	if adapter == nil || adapter.cgroups == nil {
		return false, &Error{Code: CodeCgroupRootRequired}
	}
	identity, err := platformInspectProcess(record)
	if err != nil {
		return false, &Error{Code: CodeProcessInspectionFailed, Cause: err}
	}
	if identity == processIdentityMismatch {
		return false, &Error{Code: CodeProcessIdentityMismatch}
	}
	populated, err := adapter.cgroups.populated(record)
	if err != nil {
		// cgroupfs refuses rmdir while populated. Therefore an exact missing
		// leaf, together with the already verified gone supervisor identity,
		// is equivalent to populated=0 and closes the cleanup race with Stop.
		if identity == processIdentityGone && errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		return false, &Error{Code: ErrorCode(err), Cause: errors.Join(errors.New("inspect cgroup population"), err)}
	}
	return identity == processIdentityGone && !populated, nil
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

func (adapter *Adapter) signalProcessTree(record processRecord, mode ports.AgentStopMode) error {
	if record.SchemaVersion != cgroupProcessSchemaVersion {
		if adapter == nil || !adapter.config.allowLegacyProcessControlForTests {
			return &Error{Code: CodeControlUnsupported}
		}
		return signalProcessTree(record, mode)
	}
	if adapter == nil || adapter.cgroups == nil {
		return &Error{Code: CodeCgroupRootRequired}
	}
	if _, err := adapter.cgroups.populated(record); err != nil {
		return &Error{Code: ErrorCode(err), Cause: errors.Join(errors.New("signal cgroup population"), err)}
	}
	err := platformSignalCgroupSupervisor(record, mode)
	if errors.Is(err, os.ErrProcessDone) && mode == ports.AgentStopForced {
		// The syscall effect is all that belongs under adapter.mu. Stop then
		// waits for exact populated=0 through inspectProcessTree after unlock.
		return adapter.cgroups.kill(record)
	}
	return err
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

func (adapter *Adapter) discoverPersistedProcessesLocked(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	names, err := adapter.stopJournalNames("executions")
	if err != nil {
		return err
	}
	sort.Strings(names)
	discoveryErrors := make([]error, 0)
	for _, name := range names {
		runPath := path.Join("executions", name)
		if err := adapter.discoverPersistedProcessLocked(ctx, runPath); err != nil {
			discoveryErrors = append(discoveryErrors, err)
		}
	}
	return errors.Join(discoveryErrors...)
}

// discoverShutdownProcessesLocked acquires only exact process ownership for
// termination. It deliberately does not settle provider output: recovery of
// credential and session guards belongs to the normal runtime-scope activation
// path, while shutdown must remain able to kill durable processes even when
// their external authority is no longer available.
func (adapter *Adapter) discoverShutdownProcessesLocked(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	names, err := adapter.stopJournalNames("executions")
	if err != nil {
		return err
	}
	sort.Strings(names)
	discoveryErrors := make([]error, 0)
	for _, name := range names {
		runPath := path.Join("executions", name)
		if err := adapter.discoverShutdownProcessLocked(ctx, runPath); err != nil {
			discoveryErrors = append(discoveryErrors, err)
		}
	}
	return errors.Join(discoveryErrors...)
}

func (adapter *Adapter) discoverShutdownProcessLocked(ctx context.Context, runPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	process, found, err := adapter.readProcessRecord(runPath)
	if err != nil {
		return err
	}
	if !found {
		return adapter.cleanupOrphanedCgroup(runPath, adapter.config.SupervisorStartTimeout)
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
	if _, terminalFound, err := adapter.loadCausalTerminal(
		runPath, launch.RequestHash, launch.SpecHash, launch.MaxOutputBytes,
	); err != nil {
		return err
	} else if terminalFound {
		return adapter.cleanupOrphanedCgroup(runPath, adapter.config.SupervisorStartTimeout)
	}
	receipt, err := launch.receipt(executionRef)
	if err != nil {
		return err
	}
	state := &executionState{
		requestHash: launch.RequestHash, terminalRequestHash: launch.RequestHash,
		receipt: receipt, maxOutput: launch.MaxOutputBytes, runPath: runPath, status: ports.AgentPending,
	}
	if _, _, err := adapter.ownProcessLocked(state); err != nil {
		return err
	}
	adapter.executions[executionRef.String()] = state
	return nil
}

func (adapter *Adapter) discoverPersistedProcessLocked(ctx context.Context, runPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	process, found, err := adapter.readProcessRecord(runPath)
	if err != nil {
		return err
	}
	if !found {
		return adapter.cleanupOrphanedCgroup(runPath, adapter.config.SupervisorStartTimeout)
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
		return adapter.cleanupOrphanedCgroup(runPath, adapter.config.SupervisorStartTimeout)
	}
	state := &executionState{
		requestHash: launch.RequestHash, terminalRequestHash: launch.RequestHash,
		receipt: receipt, maxOutput: launch.MaxOutputBytes, runPath: runPath, status: ports.AgentPending,
	}
	if recoveryErr := adapter.recoverExecutionGuards(ctx, launch, state); recoveryErr != nil {
		if recoveryAuthorityFailure(recoveryErr) {
			recoveryErr = adapter.quarantineRecoveryFailureLocked(ctx, state, recoveryErr)
		}
		return recoveryErr
	}
	adopted, err := adapter.adoptPersistedProcessLocked(state)
	if err != nil {
		destroyExecutionGuards(state)
		return err
	}
	if adopted {
		adapter.executions[executionRef.String()] = state
	} else {
		destroyExecutionGuards(state)
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
