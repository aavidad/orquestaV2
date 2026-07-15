package codex

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"sync"
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
	CodeReasoningEffortInvalid  = "codex.reasoning_effort_invalid"
	CodeTimeoutInvalid          = "codex.timeout_invalid"
	CodeProcessPipeDrainInvalid = "codex.process_pipe_drain_delay_invalid"
	CodeDiagnosticLimitInvalid  = "codex.max_diagnostic_bytes_invalid"
	CodeMaxConcurrentInvalid    = "codex.max_concurrent_executions_invalid"
	CodeEnvironmentInvalid      = "codex.environment_invalid"
	CodeCredentialInvalid       = "codex.credential_invalid"
	CodeCredentialUnavailable   = "codex.credential_unavailable"
	CodeSecretLeak              = "codex.secret_leak"
)

var (
	errExecutionTimeout  = errors.New(CodeExecutionTimeout)
	errExecutionCanceled = errors.New(CodeExecutionCanceled)
	errAdapterShutdown   = errors.New("codex.adapter_shutdown")
	errExecutionFinished = errors.New("codex.execution_finished")
)

// Config is fully resolved by the composition root. Environment is the exact
// child-process environment; it is never merged with the parent environment.
type Config struct {
	Command                 string
	WorkRoot                string
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
	if terminal, found, loadErr := adapter.loadTerminal(runPath, terminalRequestHash, record.SpecHash, record.MaxOutputBytes); loadErr != nil {
		return ports.AgentLaunchReceipt{}, loadErr
	} else if found {
		state.status = terminal.Status
		state.terminal = &terminal
		state.terminalDurable = true
		adapter.executions[executionKey] = state
		return receipt, nil
	}
	if !recordCreated {
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
	if terminal, terminalFound, loadErr := adapter.loadTerminal(runPath, record.RequestHash, record.SpecHash, record.MaxOutputBytes); loadErr != nil {
		return ports.AgentObservation{}, loadErr
	} else if terminalFound {
		state.status = terminal.Status
		state.terminal = &terminal
		state.terminalDurable = true
	} else if recoveryErr := adapter.recoverInterruptedExecutionLocked(state); recoveryErr != nil {
		return ports.AgentObservation{}, recoveryErr
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
	return nil
}

func (adapter *Adapter) observeStateLocked(executionKey string, executionRef goal.ExecutionRef, state *executionState) (ports.AgentObservation, error) {
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
		go func() {
			adapter.waitGroup.Wait()
			adapter.shutdownErr = adapter.root.Close()
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

func (adapter *Adapter) Close() error {
	return adapter.Shutdown(context.Background())
}

var _ application.AgentLauncher = (*Adapter)(nil)
var _ application.AgentObserver = (*Adapter)(nil)
