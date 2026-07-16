package codex

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const (
	ProviderRef = "provider:codex"
	// DefaultModelRef is a stable logical selector owned by the Codex adapter.
	// It does not claim which physical model/version the provider resolves.
	DefaultModelRef = "codex-default"
	AgentRef        = "agent:codex"
)

const (
	CodeUnavailable             = "codex.unavailable"
	CodeCapacityUnavailable     = "codex.capacity_unavailable"
	CodeExecutionConflict       = "codex.execution_conflict"
	CodeExecutionNotFound       = "codex.execution_not_found"
	CodeExecutionInterrupted    = "codex.execution_interrupted"
	CodeExecutionCanceled       = "codex.execution_canceled"
	CodeExecutionStopped        = "codex.execution_stopped"
	CodeExecutionTimeout        = "codex.execution_timeout"
	CodeProcessStartFailed      = "codex.process_start_failed"
	CodeProcessFailed           = "codex.process_failed"
	CodeProcessCleanupFailed    = "codex.process_cleanup_failed"
	CodeOutputMissing           = "codex.output_missing"
	CodeOutputTooLarge          = "codex.output_too_large"
	CodeOutputInvalid           = "codex.output_invalid"
	CodeStateInvalid            = "codex.state_invalid"
	CodeStatePersistenceFailed  = "codex.state_persistence_failed"
	CodeClockInvalid            = "codex.clock_invalid"
	CodeCommandRequired         = "codex.command_required"
	CodeCommandInvalid          = "codex.command_invalid"
	CodeCommandNotFound         = "codex.command_not_found"
	CodeWorkRootRequired        = "codex.work_root_required"
	CodeWorkRootCreateFailed    = "codex.work_root_create_failed"
	CodeWorkRootInvalid         = "codex.work_root_invalid"
	CodeWorkRootPermissions     = "codex.work_root_permissions"
	CodeWorkRootOpenFailed      = "codex.work_root_open_failed"
	CodeRuntimeScopeInvalid     = "codex.runtime_scope_invalid"
	CodeReasoningEffortInvalid  = "codex.reasoning_effort_invalid"
	CodeTimeoutInvalid          = "codex.timeout_invalid"
	CodeProcessPipeDrainInvalid = "codex.process_pipe_drain_delay_invalid"
	CodeDiagnosticLimitInvalid  = "codex.max_diagnostic_bytes_invalid"
	CodeMaxConcurrentInvalid    = "codex.max_concurrent_executions_invalid"
	CodeEnvironmentInvalid      = "codex.environment_invalid"
	CodeCredentialInvalid       = "codex.credential_invalid"
	CodeCredentialUnavailable   = "codex.credential_unavailable"
	CodeSecretLeak              = "codex.secret_leak"
	CodeControlUnsupported      = "codex.control_unsupported"
	CodeProcessOwnershipBusy    = "codex.process_ownership_busy"
	CodeProcessOwnershipInvalid = "codex.process_ownership_invalid"
	CodeProcessIdentityMismatch = "codex.process_identity_mismatch"
	CodeProcessInspectionFailed = "codex.process_inspection_failed"
	CodeProcessSignalFailed     = "codex.process_signal_failed"
	CodeStopConflict            = "codex.stop_conflict"
)

var (
	errExecutionTimeout            = errors.New(CodeExecutionTimeout)
	errExecutionCanceled           = errors.New(CodeExecutionCanceled)
	errExecutionStoppedCooperative = errors.New(CodeExecutionStopped + ".cooperative")
	errExecutionStoppedForced      = errors.New(CodeExecutionStopped + ".forced")
	errAdapterShutdown             = errors.New("codex.adapter_shutdown")
	errExecutionFinished           = errors.New("codex.execution_finished")
)

// Config is fully resolved by the composition root. Environment is the exact
// child-process environment; it is never merged with the parent environment.
type Config struct {
	Command  string
	WorkRoot string
	// RuntimeScope is an opaque, host-local identity supplied by bootstrap. It
	// must not be stored in the application database or copied by its backup.
	RuntimeScope            string
	Model                   string
	ReasoningEffort         string
	Timeout                 time.Duration
	ProcessPipeDrainDelay   time.Duration
	MaxDiagnosticBytes      int64
	MaxConcurrentExecutions int
	Environment             map[string]string
	CredentialStore         credentials.Store
	CredentialRef           credentials.CredentialRef
	Now                     func() time.Time
}

// Error exposes only a stable machine code. Cause is retained for local
// diagnosis, but process stderr is never attached to an error returned through
// the agent port.
type Error struct {
	Code             string
	Cause            error
	TemporaryFailure bool
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func (err *Error) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Cause
}

func (err *Error) Temporary() bool {
	return err != nil && err.TemporaryFailure
}

func ErrorCode(err error) string {
	var adapterError *Error
	if errors.As(err, &adapterError) {
		return adapterError.Code
	}
	return ""
}

type Adapter struct {
	config                Config
	command               string
	environment           []string
	rootPath              string
	root                  *os.Root
	syncDirectoryFn       func(*os.Root, string) error
	processCleanup        func(*exec.Cmd) error
	credentialOutputScrub func(string) error
	lifecycle             context.Context
	cancelLifecycle       context.CancelCauseFunc

	mu           sync.Mutex
	closed       bool
	executions   map[string]*executionState
	waitGroup    sync.WaitGroup
	shutdownOnce sync.Once
	shutdownDone chan struct{}
	shutdownErr  error
}

type executionState struct {
	requestHash         string
	terminalRequestHash string
	receipt             ports.AgentLaunchReceipt
	maxOutput           int64
	runPath             string
	status              ports.AgentStatus
	terminal            *terminalRecord
	terminalDurable     bool
	cancel              context.CancelCauseFunc
	credentialGuard     *credentials.LeakGuard
	process             *processRecord
	ownerLock           *os.File
	settled             chan struct{}
	stopProof           atomic.Pointer[stopSignalProof]
}

func New(config Config) (*Adapter, error) {
	validated, command, environment, rootPath, root, err := prepareConfig(config)
	if err != nil {
		return nil, err
	}
	lifecycle, cancelLifecycle := context.WithCancelCause(context.Background())
	adapter := &Adapter{
		config:          validated,
		command:         command,
		environment:     environment,
		rootPath:        rootPath,
		root:            root,
		syncDirectoryFn: syncCodexDirectory,
		processCleanup:  cleanupProcessGroup,
		lifecycle:       lifecycle,
		cancelLifecycle: cancelLifecycle,
		executions:      make(map[string]*executionState),
		shutdownDone:    make(chan struct{}),
	}
	adapter.credentialOutputScrub = adapter.scrubCredentialOutput
	return adapter, nil
}

// BindRuntimeScope completes the host-local process-control identity after the
// composition root has opened its durable state file. Construction deliberately
// permits an empty scope so executable/work-root preflight can still happen
// before any durable composition state is created.
func (adapter *Adapter) BindRuntimeScope(scope string) error {
	if adapter == nil {
		return &Error{Code: CodeUnavailable}
	}
	if scope == "" || strings.TrimSpace(scope) != scope || strings.ContainsRune(scope, '\x00') {
		return &Error{Code: CodeRuntimeScopeInvalid}
	}
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	if adapter.closed {
		return &Error{Code: CodeUnavailable}
	}
	if adapter.config.RuntimeScope == scope {
		return nil
	}
	if adapter.config.RuntimeScope != "" || len(adapter.executions) != 0 {
		return &Error{Code: CodeRuntimeScopeInvalid}
	}
	adapter.config.RuntimeScope = scope
	return nil
}

func (adapter *Adapter) Capabilities(ctx context.Context) (ports.AgentCapabilities, error) {
	if adapter == nil {
		return ports.AgentCapabilities{}, &Error{Code: CodeUnavailable}
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentCapabilities{}, err
	}
	adapter.mu.Lock()
	closed := adapter.closed
	adapter.mu.Unlock()
	if closed {
		return ports.AgentCapabilities{}, &Error{Code: CodeUnavailable}
	}
	return ports.AgentCapabilities{
		ProviderRef:  ProviderRef,
		ModelRef:     adapter.modelRef(),
		AgentRef:     AgentRef,
		Unrestricted: true,
	}, nil
}

func (adapter *Adapter) modelRef() string {
	if adapter.config.Model != "" {
		return adapter.config.Model
	}
	return DefaultModelRef
}

func (adapter *Adapter) Launch(ctx context.Context, request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
	if adapter == nil {
		return ports.AgentLaunchReceipt{}, &Error{Code: CodeUnavailable}
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	if err := ports.ValidateAgentLaunchRequest(request); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	requestHash, err := hashLaunchRequest(request)
	if err != nil {
		return ports.AgentLaunchReceipt{}, &Error{Code: CodeStateInvalid, Cause: err}
	}
	executionKey := request.ExecutionRef.String()

	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	if adapter.closed {
		return ports.AgentLaunchReceipt{}, &Error{Code: CodeUnavailable}
	}
	if existing, found := adapter.executions[executionKey]; found {
		if existing.requestHash != requestHash {
			return ports.AgentLaunchReceipt{}, &Error{Code: CodeExecutionConflict}
		}
		return existing.receipt, nil
	}

	record, runPath, found, err := adapter.loadLaunchRecord(request.ExecutionRef)
	if err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	if found {
		return adapter.resumeLaunchRecordLocked(ctx, request, requestHash, record, runPath, false, nil, nil)
	}
	if adapter.activeExecutionCountLocked() >= adapter.config.MaxConcurrentExecutions {
		return ports.AgentLaunchReceipt{}, &Error{
			Code:             CodeCapacityUnavailable,
			TemporaryFailure: true,
		}
	}

	if adapter.config.CredentialStore != nil {
		return adapter.launchWithCredentialLocked(ctx, request, requestHash)
	}
	record, runPath, recordCreated, err := adapter.ensureLaunchRecord(request, requestHash)
	if err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	return adapter.resumeLaunchRecordLocked(ctx, request, requestHash, record, runPath, recordCreated, adapter.environment, nil)
}

func (adapter *Adapter) resumeLaunchRecordLocked(
	ctx context.Context,
	request ports.AgentLaunchRequest,
	requestHash string,
	record launchRecord,
	runPath string,
	recordCreated bool,
	environment []string,
	credentialGuard *credentials.LeakGuard,
) (ports.AgentLaunchReceipt, error) {
	defer func() {
		if credentialGuard != nil {
			credentialGuard.Destroy()
		}
	}()
	executionKey := request.ExecutionRef.String()
	terminalRequestHash := record.RequestHash
	if record.SchemaVersion == legacyStateSchemaVersion {
		upgraded, err := adapter.bindLegacyLaunchRecord(runPath, record, request, requestHash)
		if err != nil {
			return ports.AgentLaunchReceipt{}, err
		}
		record = upgraded
	} else if record.RequestHash != requestHash {
		return ports.AgentLaunchReceipt{}, &Error{Code: CodeExecutionConflict}
	}
	receipt, err := record.receipt(request.ExecutionRef)
	if err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	if err := ports.ValidateAgentLaunchReceipt(request, receipt); err != nil {
		return ports.AgentLaunchReceipt{}, &Error{Code: CodeStateInvalid, Cause: err}
	}
	state := &executionState{
		requestHash:         requestHash,
		terminalRequestHash: terminalRequestHash,
		receipt:             receipt,
		maxOutput:           record.MaxOutputBytes,
		runPath:             runPath,
		status:              ports.AgentPending,
	}
	if terminal, found, loadErr := adapter.loadCausalTerminal(runPath, terminalRequestHash, record.SpecHash, record.MaxOutputBytes); loadErr != nil {
		return ports.AgentLaunchReceipt{}, loadErr
	} else if found {
		state.status = terminal.Status
		state.terminal = &terminal
		state.terminalDurable = true
		adapter.executions[executionKey] = state
		return receipt, nil
	}
	if !recordCreated {
		adopted, adoptErr := adapter.adoptPersistedProcessLocked(state)
		if adoptErr != nil {
			return ports.AgentLaunchReceipt{}, adoptErr
		}
		if adopted || state.terminal != nil {
			adapter.executions[executionKey] = state
			return receipt, nil
		}
		if err := adapter.recoverInterruptedExecutionLocked(state); err != nil {
			return ports.AgentLaunchReceipt{}, err
		}
		adapter.executions[executionKey] = state
		return receipt, nil
	}

	adapter.executions[executionKey] = state
	state.credentialGuard = credentialGuard
	credentialGuard = nil
	adapter.startExecutionLocked(ctx, request, state, environment)
	return receipt, nil
}

func (adapter *Adapter) activeExecutionCountLocked() int {
	active := 0
	for _, state := range adapter.executions {
		if state.terminal == nil {
			active++
		}
	}
	return active
}

func (adapter *Adapter) Observe(ctx context.Context, executionRef goal.ExecutionRef) (ports.AgentObservation, error) {
	if adapter == nil {
		return ports.AgentObservation{}, &Error{Code: CodeUnavailable}
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentObservation{}, err
	}
	if executionRef.String() == "" {
		return ports.AgentObservation{}, &ports.AgentContractError{Code: "agent.execution_ref_required"}
	}
	executionKey := executionRef.String()

	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	if adapter.closed {
		return ports.AgentObservation{}, &Error{Code: CodeUnavailable}
	}
	if state, found := adapter.executions[executionKey]; found {
		return adapter.observeStateLocked(executionKey, executionRef, state)
	}

	record, runPath, found, err := adapter.loadLaunchRecord(executionRef)
	if err != nil {
		return ports.AgentObservation{}, err
	}
	if !found {
		return ports.AgentObservation{}, &Error{Code: CodeExecutionNotFound}
	}
	receipt, err := adapter.observationReceipt(record, executionRef)
	if err != nil {
		return ports.AgentObservation{}, err
	}
	state := &executionState{
		requestHash:         record.RequestHash,
		terminalRequestHash: record.RequestHash,
		receipt:             receipt,
		maxOutput:           record.MaxOutputBytes,
		runPath:             runPath,
		status:              ports.AgentPending,
	}
	if terminal, terminalFound, loadErr := adapter.loadCausalTerminal(runPath, record.RequestHash, record.SpecHash, record.MaxOutputBytes); loadErr != nil {
		return ports.AgentObservation{}, loadErr
	} else if terminalFound {
		state.status = terminal.Status
		state.terminal = &terminal
		state.terminalDurable = true
	} else {
		adopted, adoptErr := adapter.adoptPersistedProcessLocked(state)
		if adoptErr != nil {
			return ports.AgentObservation{}, adoptErr
		}
		if !adopted && state.terminal == nil {
			if recoveryErr := adapter.recoverInterruptedExecutionLocked(state); recoveryErr != nil {
				return ports.AgentObservation{}, recoveryErr
			}
		}
	}
	adapter.executions[executionKey] = state
	return adapter.observeStateLocked(executionKey, executionRef, state)
}

func (adapter *Adapter) recoverInterruptedExecutionLocked(state *executionState) error {
	if err := adapter.credentialOutputScrub(state.runPath); err != nil {
		return err
	}
	terminal := terminalRecord{
		SchemaVersion: stateSchemaVersion,
		RequestHash:   state.terminalRequestHash,
		Status:        ports.AgentFailed,
		ErrorCode:     CodeExecutionInterrupted,
		ObservedAt:    adapter.terminalTime(state.receipt.AcceptedAt),
	}
	persisted, err := adapter.persistTerminal(state.runPath, terminal, state.receipt.SpecHash, state.maxOutput)
	if err != nil {
		return err
	}
	state.status = persisted.Status
	state.terminal = &persisted
	state.terminalDurable = true
	adapter.releaseProcessOwnershipLocked(state)
	return nil
}

func (adapter *Adapter) observeStateLocked(executionKey string, executionRef goal.ExecutionRef, state *executionState) (ports.AgentObservation, error) {
	if state.terminal == nil && state.process != nil && state.cancel == nil {
		treeGone, err := inspectProcessTree(*state.process)
		if err != nil {
			return ports.AgentObservation{}, err
		}
		if treeGone {
			if proof := state.stopProof.Load(); proof != nil {
				if err := adapter.finishStoppedProcessLocked(state, *proof); err != nil {
					return ports.AgentObservation{}, err
				}
			} else {
				// A durable request only proves intent. Read it so journal I/O
				// failures remain visible, then recover the unknowable exit.
				if _, err := adapter.hasDurableStopRequest(state.runPath); err != nil {
					return ports.AgentObservation{}, err
				}
				if err := adapter.recoverInterruptedExecutionLocked(state); err != nil {
					return ports.AgentObservation{}, err
				}
			}
		}
	}
	if state.terminal != nil && !state.terminalDurable {
		if err := adapter.credentialOutputScrub(state.runPath); err != nil {
			return ports.AgentObservation{}, err
		}
		persisted, err := adapter.persistTerminal(state.runPath, *state.terminal, state.receipt.SpecHash, state.maxOutput)
		if err != nil {
			return ports.AgentObservation{}, err
		}
		state.terminal = &persisted
		state.status = persisted.Status
		state.terminalDurable = true
	}
	observation, err := adapter.observationForState(executionRef, state)
	if err == nil && state.terminal != nil && state.terminalDurable {
		delete(adapter.executions, executionKey)
	}
	return observation, err
}

func (adapter *Adapter) observationForState(executionRef goal.ExecutionRef, state *executionState) (ports.AgentObservation, error) {
	if state.terminal != nil {
		observation := state.terminal.observation(executionRef, state.receipt.SpecHash)
		if err := ports.ValidateAgentObservation(observation, state.maxOutput); err != nil {
			return ports.AgentObservation{}, &Error{Code: CodeStateInvalid, Cause: err}
		}
		return observation, nil
	}
	observedAt := adapter.config.Now()
	if observedAt.IsZero() {
		return ports.AgentObservation{}, &Error{Code: CodeClockInvalid}
	}
	return ports.AgentObservation{
		ExecutionRef: executionRef,
		SpecHash:     state.receipt.SpecHash,
		Status:       state.status,
		ObservedAt:   observedAt.UTC(),
	}, nil
}

func (adapter *Adapter) Shutdown(ctx context.Context) error {
	if adapter == nil {
		return nil
	}
	adapter.mu.Lock()
	if !adapter.closed {
		adapter.closed = true
		adapter.cancelLifecycle(errAdapterShutdown)
	}
	adapter.shutdownOnce.Do(func() {
		adopted, discoveryErr := adapter.adoptedProcessesLocked()
		go func() {
			adoptedErr := adapter.stopAdoptedProcesses(adopted)
			adapter.waitGroup.Wait()
			adapter.shutdownErr = errors.Join(discoveryErr, adoptedErr, adapter.root.Close())
			close(adapter.shutdownDone)
		}()
	})
	done := adapter.shutdownDone
	adapter.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return adapter.shutdownErr
	}
}

type adoptedProcess struct {
	state  *executionState
	record processRecord
}

func (adapter *Adapter) adoptedProcessesLocked() ([]adoptedProcess, error) {
	discoveryErr := adapter.discoverPersistedProcessesLocked()
	processes := make([]adoptedProcess, 0)
	for _, state := range adapter.executions {
		if state.process == nil || state.ownerLock == nil || state.cancel != nil {
			continue
		}
		processes = append(processes, adoptedProcess{state: state, record: *state.process})
	}
	return processes, discoveryErr
}

func (adapter *Adapter) stopAdoptedProcesses(processes []adoptedProcess) error {
	errorsByProcess := make([]error, len(processes))
	for index, process := range processes {
		err := signalProcessTree(process.record, ports.AgentStopForced)
		if err != nil && !errors.Is(err, os.ErrProcessDone) {
			errorsByProcess[index] = err
		}
	}
	for index, process := range processes {
		observedGone := false
		if errorsByProcess[index] == nil {
			observedGone, errorsByProcess[index] = waitForExactProcess(context.Background(), process.record, nil)
		}
		if observedGone {
			adapter.mu.Lock()
			if process.state.process != nil && *process.state.process == process.record {
				adapter.releaseProcessOwnershipLocked(process.state)
			}
			adapter.mu.Unlock()
		}
	}
	return errors.Join(errorsByProcess...)
}

func (adapter *Adapter) Close() error {
	return adapter.Shutdown(context.Background())
}

var _ application.AgentLauncher = (*Adapter)(nil)
var _ application.AgentObserver = (*Adapter)(nil)
var _ application.AgentController = (*Adapter)(nil)
