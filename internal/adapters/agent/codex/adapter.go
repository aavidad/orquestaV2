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
	CodeUnavailable                       = "codex.unavailable"
	CodeCapacityUnavailable               = "codex.capacity_unavailable"
	CodeExecutionConflict                 = "codex.execution_conflict"
	CodeExecutionNotFound                 = "codex.execution_not_found"
	CodeLegacyExecutionRequiresNewAttempt = "codex.legacy_execution_requires_new_attempt"
	CodeLegacyControlMetadataUnknown      = "codex.legacy_control_metadata_unknown"
	CodeExecutionInterrupted              = "codex.execution_interrupted"
	CodeExecutionCanceled                 = "codex.execution_canceled"
	CodeExecutionStopped                  = "codex.execution_stopped"
	CodeExecutionTimeout                  = "codex.execution_timeout"
	CodeProcessStartFailed                = "codex.process_start_failed"
	CodeProcessFailed                     = "codex.process_failed"
	CodeProcessCleanupFailed              = "codex.process_cleanup_failed"
	CodeOutputMissing                     = "codex.output_missing"
	CodeOutputTooLarge                    = "codex.output_too_large"
	CodeOutputInvalid                     = "codex.output_invalid"
	CodeStateInvalid                      = "codex.state_invalid"
	CodeStatePersistenceFailed            = "codex.state_persistence_failed"
	CodeClockInvalid                      = "codex.clock_invalid"
	CodeCommandRequired                   = "codex.command_required"
	CodeCommandInvalid                    = "codex.command_invalid"
	CodeCommandNotFound                   = "codex.command_not_found"
	CodeWorkRootRequired                  = "codex.work_root_required"
	CodeWorkRootCreateFailed              = "codex.work_root_create_failed"
	CodeWorkRootInvalid                   = "codex.work_root_invalid"
	CodeWorkRootPermissions               = "codex.work_root_permissions"
	CodeWorkRootOpenFailed                = "codex.work_root_open_failed"
	CodeRuntimeScopeInvalid               = "codex.runtime_scope_invalid"
	CodeReasoningEffortInvalid            = "codex.reasoning_effort_invalid"
	CodeTimeoutInvalid                    = "codex.timeout_invalid"
	CodeProcessPipeDrainInvalid           = "codex.process_pipe_drain_delay_invalid"
	CodeSupervisorStartTimeoutInvalid     = "codex.supervisor_start_timeout_invalid"
	CodeDiagnosticLimitInvalid            = "codex.max_diagnostic_bytes_invalid"
	CodeMaxConcurrentInvalid              = "codex.max_concurrent_executions_invalid"
	CodeEnvironmentInvalid                = "codex.environment_invalid"
	CodeAccountProfileInvalid             = "codex.account_profile_invalid"
	CodeAccountProfileUnavailable         = "codex.account_profile_unavailable"
	CodeCredentialInvalid                 = "codex.credential_invalid"
	CodeCredentialUnavailable             = "codex.credential_unavailable"
	CodeCredentialOutputUnverifiable      = "codex.credential_output_unverifiable"
	CodePromptRendererInvalid             = "codex.prompt_renderer_invalid"
	CodePromptRenderFailed                = "codex.prompt_render_failed"
	CodeSessionInvalid                    = "codex.session_invalid"
	CodeSessionUnavailable                = "codex.session_unavailable"
	CodeWorkspaceResolverInvalid          = "codex.workspace_resolver_invalid"
	CodeWorkspaceUnavailable              = "codex.workspace_unavailable"
	CodeWorkspaceUnsafe                   = "codex.workspace_unsafe"
	CodeSecretLeak                        = "codex.secret_leak"
	CodeControlUnsupported                = "codex.control_unsupported"
	CodeProcessOwnershipBusy              = "codex.process_ownership_busy"
	CodeProcessOwnershipInvalid           = "codex.process_ownership_invalid"
	CodeProcessIdentityMismatch           = "codex.process_identity_mismatch"
	CodeProcessInspectionFailed           = "codex.process_inspection_failed"
	CodeProcessSignalFailed               = "codex.process_signal_failed"
	CodeStopConflict                      = "codex.stop_conflict"
	CodeCgroupRootRequired                = "codex.cgroup_root_required"
	CodeCgroupRootInvalid                 = "codex.cgroup_root_invalid"
	CodeCgroupCreateFailed                = "codex.cgroup_create_failed"
	CodeCgroupIdentityMismatch            = "codex.cgroup_identity_mismatch"
	CodeCgroupDrainFailed                 = "codex.cgroup_drain_failed"
)

// WorkspacePathResolver is deliberately adapter-local.  It resolves the
// opaque execution binding only at the process boundary; neither the agent
// port nor any Codex receipt gains a physical workspace path.
type WorkspacePathResolver interface {
	ResolveExecutionWorkspace(context.Context, ports.ExecutionWorkspaceRef) (string, error)
}

var (
	errExecutionTimeout            = errors.New(CodeExecutionTimeout)
	errExecutionCanceled           = errors.New(CodeExecutionCanceled)
	errExecutionStoppedCooperative = errors.New(CodeExecutionStopped + ".cooperative")
	errAdapterShutdown             = errors.New("codex.adapter_shutdown")
	errExecutionFinished           = errors.New("codex.execution_finished")
)

// Config is fully resolved by the composition root. Environment is the exact
// child-process environment; it is never merged with the parent environment.
type Config struct {
	Command  string
	WorkRoot string
	// CgroupRoot is the composition-delegated cgroup v2 directory used only
	// for Codex execution boundaries. It is distinct from attestor resources.
	CgroupRoot string
	// RuntimeScope is an opaque, host-local identity supplied by bootstrap. It
	// must not be stored in the application database or copied by its backup.
	RuntimeScope            string
	Model                   string
	ReasoningEffort         string
	Timeout                 time.Duration
	ProcessPipeDrainDelay   time.Duration
	SupervisorStartTimeout  time.Duration
	MaxDiagnosticBytes      int64
	MaxConcurrentExecutions int
	MCPBearerTokenEnvVar    string
	Environment             map[string]string
	// AccountHomeRoot and AccountProfile opt into one account-bound Codex
	// daemon. The selected persistent profile is projected directly as both
	// HOME and CODEX_HOME so Codex can persist credential refreshes. A
	// lifetime lease and MaxConcurrentExecutions=1 prevent concurrent use.
	// Pool composes several homogeneous account-bound adapters behind one
	// server and derives a disjoint WorkRoot for every opaque profile binding.
	AccountHomeRoot                   string
	AccountProfile                    string
	AccountAuthMaxDocumentBytes       int64
	CredentialStore                   credentials.Store
	CredentialRef                     credentials.CredentialRef
	PromptRenderer                    PromptRenderer
	SessionResolver                   SessionResolver
	WorkspacePathResolver             WorkspacePathResolver
	Now                               func() time.Time
	allowLegacyProcessControlForTests bool
	launchFailureStageForTests        string
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

// DefinitelyNotApplied proves the capacity rejection happened before Codex
// created durable launch state or started a process. Other temporary failures
// remain ambiguous and retain their reservation for idempotent reconciliation.
func (err *Error) DefinitelyNotApplied() bool {
	return err != nil && err.Code == CodeCapacityUnavailable
}

func ErrorCode(err error) string {
	var adapterError *Error
	if errors.As(err, &adapterError) {
		return adapterError.Code
	}
	return ""
}

type Adapter struct {
	config                   Config
	command                  string
	environment              []string
	rootPath                 string
	root                     *os.Root
	cgroups                  *codexCgroupRoot
	accountHomePath          string
	accountProfileBindingRef string
	accountProfileLock       *os.File
	syncDirectoryFn          func(*os.Root, string) error
	processCleanup           func(*exec.Cmd) error
	credentialOutputScrub    func(string) error
	beforeCgroupCleanup      func(string) error
	shutdownSignal           func(processRecord, ports.AgentStopMode) error
	shutdownInspect          func(processRecord) (bool, error)
	workspaceResolver        WorkspacePathResolver
	lifecycle                context.Context
	cancelLifecycle          context.CancelCauseFunc

	mu                sync.Mutex
	closed            bool
	runtimeScopeReady bool
	executions        map[string]*executionState
	activeWaits       int
	waitsDone         chan struct{}
	operations        int
	operationsDone    chan struct{}
	shutdownOnce      sync.Once
	shutdownDone      chan struct{}
	shutdownErr       error
}

type executionState struct {
	requestHash         string
	terminalRequestHash string
	receipt             ports.AgentLaunchReceipt
	maxOutput           int64
	artifactMediaType   string
	runPath             string
	status              ports.AgentStatus
	terminal            *terminalRecord
	terminalDurable     bool
	cancel              context.CancelCauseFunc
	credentialGuard     *credentials.LeakGuard
	sessionGuard        *credentials.LeakGuard
	process             *processRecord
	ownerLock           *os.File
	settled             chan struct{}
	starting            *executionStart
	startupRollback     bool
	startupErrorCode    string
	stopProof           atomic.Pointer[stopSignalProof]
	quarantine          *quarantineOperation
}

func New(config Config) (*Adapter, error) {
	validated, command, environment, rootPath, root, err := prepareConfig(config)
	if err != nil {
		return nil, err
	}
	accountProfileLock, accountHomePath, err := openAccountProfileLease(validated)
	if err != nil {
		_ = root.Close()
		return nil, err
	}
	accountProfileBindingRef, err := deriveAccountProfileRef(validated.AccountHomeRoot, validated.AccountProfile)
	if err != nil {
		_ = closeAccountProfileLease(accountProfileLock)
		_ = root.Close()
		return nil, err
	}
	var cgroups *codexCgroupRoot
	if validated.CgroupRoot != "" {
		cgroups, err = openCodexCgroupRoot(validated.CgroupRoot)
		if err != nil {
			_ = closeAccountProfileLease(accountProfileLock)
			_ = root.Close()
			return nil, err
		}
	} else if platformCgroupRequired() && validated.RuntimeScope != "" &&
		!validated.allowLegacyProcessControlForTests {
		_ = closeAccountProfileLease(accountProfileLock)
		_ = root.Close()
		return nil, &Error{Code: CodeCgroupRootRequired}
	}
	lifecycle, cancelLifecycle := context.WithCancelCause(context.Background())
	waitsDone := make(chan struct{})
	close(waitsDone)
	operationsDone := make(chan struct{})
	close(operationsDone)
	adapter := &Adapter{
		config:                   validated,
		command:                  command,
		environment:              environment,
		rootPath:                 rootPath,
		root:                     root,
		cgroups:                  cgroups,
		accountHomePath:          accountHomePath,
		accountProfileBindingRef: accountProfileBindingRef,
		accountProfileLock:       accountProfileLock,
		workspaceResolver:        validated.WorkspacePathResolver,
		syncDirectoryFn:          syncCodexDirectory,
		processCleanup:           cleanupProcessGroup,
		lifecycle:                lifecycle,
		cancelLifecycle:          cancelLifecycle,
		executions:               make(map[string]*executionState),
		waitsDone:                waitsDone,
		operationsDone:           operationsDone,
		shutdownDone:             make(chan struct{}),
	}
	adapter.shutdownSignal = adapter.signalProcessTree
	adapter.shutdownInspect = adapter.inspectProcessTree
	adapter.credentialOutputScrub = adapter.scrubCredentialOutput
	return adapter, nil
}

// BindRuntimeScope completes the host-local process-control identity after the
// composition root has opened its durable state file. Construction deliberately
// permits an empty scope so executable/work-root preflight can still happen
// before any durable composition state is created.
func (adapter *Adapter) BindRuntimeScope(ctx context.Context, scope string) error {
	if adapter == nil {
		return &Error{Code: CodeUnavailable}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if scope == "" || strings.TrimSpace(scope) != scope || strings.ContainsRune(scope, '\x00') {
		return &Error{Code: CodeRuntimeScopeInvalid}
	}
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	if adapter.closed {
		return &Error{Code: CodeUnavailable}
	}
	if adapter.config.RuntimeScope == scope && adapter.runtimeScopeReady {
		return nil
	}
	if adapter.config.RuntimeScope != "" && adapter.config.RuntimeScope != scope {
		return &Error{Code: CodeRuntimeScopeInvalid}
	}
	if adapter.config.RuntimeScope == "" && len(adapter.executions) != 0 {
		return &Error{Code: CodeRuntimeScopeInvalid}
	}
	if platformCgroupRequired() && adapter.cgroups == nil {
		if !adapter.config.allowLegacyProcessControlForTests {
			return &Error{Code: CodeCgroupRootRequired}
		}
	}
	adapter.config.RuntimeScope = scope
	if err := adapter.discoverPersistedProcessesLocked(ctx); err != nil {
		return err
	}
	adapter.runtimeScopeReady = true
	return nil
}

// BindWorkspacePathResolver attaches the composition-owned local resolver only
// after it has built the authorised workspace adapter. It is intentionally not
// persisted with the Codex journal and may not be replaced while executions
// are live.
func (adapter *Adapter) BindWorkspacePathResolver(resolver WorkspacePathResolver) error {
	if adapter == nil || resolver == nil {
		return &Error{Code: CodeWorkspaceResolverInvalid}
	}
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	if adapter.closed || len(adapter.executions) != 0 {
		return &Error{Code: CodeWorkspaceResolverInvalid}
	}
	adapter.workspaceResolver = resolver
	return nil
}

func (adapter *Adapter) Capabilities(ctx context.Context) (ports.AgentCapabilities, error) {
	if adapter == nil {
		return ports.AgentCapabilities{}, &Error{Code: CodeUnavailable}
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentCapabilities{}, err
	}
	select {
	case <-adapter.lifecycle.Done():
		return ports.AgentCapabilities{}, &Error{Code: CodeUnavailable}
	default:
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
	callerContext := ctx
	operationContext, endOperation, err := adapter.beginOperation(ctx)
	if err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	defer endOperation()
	ctx = operationContext
	requestHash, err := adapter.hashLaunchRequest(request)
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
			return adapter.retireCachedLegacyLaunchLocked(ctx, existing, request, requestHash)
		}
		return existing.receipt, nil
	}

	record, runPath, found, err := adapter.loadLaunchRecord(request.ExecutionRef)
	if err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	if found {
		if record.SchemaVersion != stateSchemaVersion {
			return adapter.retireLegacyLaunchRecordLocked(ctx, record, runPath, request, requestHash)
		}
		sourceSchema := record.SchemaVersion
		validatedRecord, receipt, trustedReplay, replayErr := adapter.validateLaunchReplay(runPath, record, request, requestHash)
		if replayErr != nil {
			return ports.AgentLaunchReceipt{}, replayErr
		}
		if sourceSchema == stateSchemaVersion {
			record = validatedRecord
		}
		terminal, terminalFound, terminalErr := adapter.loadCausalTerminal(runPath, record.RequestHash, record.SpecHash, record.MaxOutputBytes)
		if terminalErr != nil {
			return ports.AgentLaunchReceipt{}, terminalErr
		}
		if terminalFound && trustedReplay {
			return adapter.replayTerminalLaunchLocked(executionKey, requestHash, record, receipt, runPath, terminal), nil
		}
		session, sessionErr := adapter.resolveSession(ctx, request)
		if sessionErr != nil {
			if ctx.Err() != nil {
				return ports.AgentLaunchReceipt{}, ctx.Err()
			}
			if trustedReplay && recoveryAuthorityFailure(sessionErr) {
				sessionErr = adapter.quarantineLaunchRecoveryLocked(ctx, request, validatedRecord, record.RequestHash, runPath, sessionErr)
			}
			return ports.AgentLaunchReceipt{}, sessionErr
		}
		defer session.destroy()
		if preflightErr := adapter.preflightSessionLaunch(session, request); preflightErr != nil {
			if trustedReplay && recoveryAuthorityFailure(preflightErr) {
				preflightErr = adapter.quarantineLaunchRecoveryLocked(ctx, request, validatedRecord, record.RequestHash, runPath, preflightErr)
			}
			return ports.AgentLaunchReceipt{}, preflightErr
		}
		if !trustedReplay && session == nil && adapter.config.CredentialStore == nil &&
			record.AccountProfileRef == "" {
			return ports.AgentLaunchReceipt{}, &Error{Code: CodeSessionUnavailable}
		}
		if adapter.config.CredentialStore != nil {
			if terminalFound {
				if credentialErr := adapter.preflightCredentialAuthority(ctx, request, session); credentialErr != nil {
					return ports.AgentLaunchReceipt{}, credentialErr
				}
				receipt, replayErr = adapter.bindLegacyTerminalReceipt(runPath, record, request, requestHash)
				if replayErr != nil {
					return ports.AgentLaunchReceipt{}, replayErr
				}
				return adapter.replayTerminalLaunchLocked(executionKey, requestHash, record, receipt, runPath, terminal), nil
			}
			receipt, credentialErr := adapter.launchWithCredentialLocked(ctx, callerContext, request, requestHash, session)
			if credentialErr != nil {
				if trustedReplay && recoveryAuthorityFailure(credentialErr) {
					credentialErr = adapter.quarantineLaunchRecoveryLocked(ctx, request, validatedRecord, record.RequestHash, runPath, credentialErr)
				}
				return ports.AgentLaunchReceipt{}, credentialErr
			}
			return receipt, nil
		}
		if terminalFound {
			receipt, replayErr = adapter.bindLegacyTerminalReceipt(runPath, record, request, requestHash)
			if replayErr != nil {
				return ports.AgentLaunchReceipt{}, replayErr
			}
			return adapter.replayTerminalLaunchLocked(executionKey, requestHash, record, receipt, runPath, terminal), nil
		}
		return adapter.resumeLaunchRecordLocked(ctx, callerContext, request, requestHash, record, runPath, false, nil, nil, session)
	}
	if adapter.activeExecutionCountLocked() >= adapter.config.MaxConcurrentExecutions {
		return ports.AgentLaunchReceipt{}, &Error{
			Code:             CodeCapacityUnavailable,
			TemporaryFailure: true,
		}
	}

	session, err := adapter.resolveSession(ctx, request)
	if err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	defer session.destroy()
	if err := adapter.preflightSessionLaunch(session, request); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	if adapter.config.CredentialStore != nil {
		return adapter.launchWithCredentialLocked(ctx, callerContext, request, requestHash, session)
	}
	record, runPath, recordCreated, err := adapter.ensureLaunchRecord(request, requestHash)
	if err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	environment := adapter.environmentWithSession(adapter.environment, session)
	defer clearEnvironment(environment)
	return adapter.resumeLaunchRecordLocked(ctx, callerContext, request, requestHash, record, runPath, recordCreated, environment, nil, session)
}

func (adapter *Adapter) retireCachedLegacyLaunchLocked(
	ctx context.Context,
	state *executionState,
	request ports.AgentLaunchRequest,
	requestHash string,
) (ports.AgentLaunchReceipt, error) {
	legacy, runPath, found, err := adapter.loadLaunchRecord(request.ExecutionRef)
	if err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	if !found || legacy.SchemaVersion == stateSchemaVersion {
		return ports.AgentLaunchReceipt{}, &Error{Code: CodeExecutionConflict}
	}
	source, untrustedRequestHash, upgraded, err := adapter.loadLegacyLaunchBinding(runPath, legacy)
	if err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	durableState := state.requestHash == legacy.RequestHash ||
		state.requestHash == source.RequestHash ||
		upgraded && state.requestHash == untrustedRequestHash
	if state.runPath != runPath || state.terminalRequestHash != legacy.RequestHash || !durableState {
		return ports.AgentLaunchReceipt{}, &Error{Code: CodeStateInvalid}
	}
	if err := adapter.validateLegacyLaunchRequest(source, request); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	if state.terminal != nil {
		return ports.AgentLaunchReceipt{}, &Error{Code: CodeLegacyExecutionRequiresNewAttempt}
	}
	return ports.AgentLaunchReceipt{}, adapter.quarantineExecutionLocked(
		ctx, state, &Error{Code: CodeLegacyExecutionRequiresNewAttempt},
	)
}

func (adapter *Adapter) retireLegacyLaunchRecordLocked(
	ctx context.Context,
	legacy launchRecord,
	runPath string,
	request ports.AgentLaunchRequest,
	requestHash string,
) (ports.AgentLaunchReceipt, error) {
	source, untrustedRequestHash, upgraded, err := adapter.loadLegacyLaunchBinding(runPath, legacy)
	if err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	if err := adapter.validateLegacyLaunchRequest(source, request); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	process, processFound, err := adapter.readProcessRecord(runPath)
	if err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	receipt, err := adapter.observationReceipt(source, request.ExecutionRef)
	if err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	stateRequestHash := source.RequestHash
	if upgraded && processFound && process.RequestHash == untrustedRequestHash {
		stateRequestHash = process.RequestHash
	}
	state := &executionState{
		requestHash: stateRequestHash, terminalRequestHash: legacy.RequestHash,
		receipt: receipt, maxOutput: source.MaxOutputBytes,
		artifactMediaType: request.ArtifactMediaType,
		runPath:           runPath, status: ports.AgentPending,
	}
	if terminal, found, loadErr := adapter.loadCausalTerminal(
		runPath, legacy.RequestHash, source.SpecHash, source.MaxOutputBytes,
	); loadErr != nil {
		return ports.AgentLaunchReceipt{}, loadErr
	} else if found {
		state.status, state.terminal, state.terminalDurable = terminal.Status, &terminal, true
		adapter.executions[request.ExecutionRef.String()] = state
		return ports.AgentLaunchReceipt{}, &Error{Code: CodeLegacyExecutionRequiresNewAttempt}
	}
	if !processFound {
		if recoveryErr := adapter.recoverInterruptedExecutionLocked(state); recoveryErr != nil {
			return ports.AgentLaunchReceipt{}, recoveryErr
		}
		adapter.executions[request.ExecutionRef.String()] = state
		return ports.AgentLaunchReceipt{}, &Error{Code: CodeLegacyExecutionRequiresNewAttempt}
	}
	if recoveryErr := adapter.recoverExecutionGuards(ctx, source, state); recoveryErr != nil {
		if recoveryAuthorityFailure(recoveryErr) {
			recoveryErr = adapter.quarantineExecutionLocked(
				ctx, state, &Error{
					Code: CodeLegacyExecutionRequiresNewAttempt, Cause: recoveryErr,
				},
			)
		}
		adapter.executions[request.ExecutionRef.String()] = state
		if state.terminal != nil && state.terminalDurable {
			return ports.AgentLaunchReceipt{}, &Error{
				Code: CodeLegacyExecutionRequiresNewAttempt, Cause: recoveryErr,
			}
		}
		return ports.AgentLaunchReceipt{}, recoveryErr
	}
	adapter.executions[request.ExecutionRef.String()] = state
	return ports.AgentLaunchReceipt{}, adapter.quarantineExecutionLocked(
		ctx, state, &Error{Code: CodeLegacyExecutionRequiresNewAttempt},
	)
}

func (adapter *Adapter) validateLaunchReplay(runPath string, record launchRecord, request ports.AgentLaunchRequest, requestHash string) (launchRecord, ports.AgentLaunchReceipt, bool, error) {
	var err error
	trusted := record.SchemaVersion == stateSchemaVersion
	if record.SchemaVersion != stateSchemaVersion {
		record, trusted, err = adapter.previewLegacyLaunchRecord(runPath, record, request, requestHash)
	} else if record.RequestHash != requestHash {
		return record, ports.AgentLaunchReceipt{}, false, &Error{Code: CodeExecutionConflict}
	}
	if err != nil {
		return record, ports.AgentLaunchReceipt{}, false, err
	}
	if err := adapter.validateAccountProfileBinding(record); err != nil {
		return record, ports.AgentLaunchReceipt{}, false, err
	}
	receipt, err := record.receipt(request.ExecutionRef)
	if err == nil {
		err = ports.ValidateAgentLaunchReceipt(request, receipt)
	}
	if err != nil {
		return record, ports.AgentLaunchReceipt{}, false, &Error{Code: CodeStateInvalid, Cause: err}
	}
	return record, receipt, trusted, nil
}

func (adapter *Adapter) bindLegacyTerminalReceipt(
	runPath string,
	record launchRecord,
	request ports.AgentLaunchRequest,
	requestHash string,
) (ports.AgentLaunchReceipt, error) {
	bound, err := adapter.bindLegacyLaunchRecord(runPath, record, request, requestHash)
	if err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	_, receipt, _, err := adapter.validateLaunchReplay(runPath, bound, request, requestHash)
	return receipt, err
}

func (adapter *Adapter) replayTerminalLaunchLocked(
	executionKey, requestHash string,
	record launchRecord,
	receipt ports.AgentLaunchReceipt,
	runPath string,
	terminal terminalRecord,
) ports.AgentLaunchReceipt {
	adapter.executions[executionKey] = &executionState{
		requestHash: requestHash, terminalRequestHash: record.RequestHash, receipt: receipt,
		maxOutput: record.MaxOutputBytes, runPath: runPath, status: terminal.Status,
		terminal: &terminal, terminalDurable: true,
	}
	return receipt
}

func (adapter *Adapter) resumeLaunchRecordLocked(
	ctx context.Context,
	callerContext context.Context,
	request ports.AgentLaunchRequest,
	requestHash string,
	record launchRecord,
	runPath string,
	recordCreated bool,
	environment []string,
	credentialGuard *credentials.LeakGuard,
	session *resolvedSession,
) (ports.AgentLaunchReceipt, error) {
	defer func() {
		if credentialGuard != nil {
			credentialGuard.Destroy()
		}
		if session != nil {
			session.destroy()
		}
	}()
	executionKey := request.ExecutionRef.String()
	terminalRequestHash := record.RequestHash
	var err error
	if record.SchemaVersion != stateSchemaVersion {
		record, err = adapter.bindLegacyLaunchRecord(runPath, record, request, requestHash)
		if err != nil {
			return ports.AgentLaunchReceipt{}, err
		}
	}
	record, receipt, _, err := adapter.validateLaunchReplay(runPath, record, request, requestHash)
	if err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	launchEnvironment := environment
	if recordCreated && record.AccountProfileRef != "" {
		launchEnvironment, err = adapter.accountExecutionEnvironment(environment)
		if err != nil {
			return ports.AgentLaunchReceipt{}, err
		}
		defer clearEnvironment(launchEnvironment)
	}
	state := &executionState{requestHash: requestHash, terminalRequestHash: terminalRequestHash,
		receipt: receipt, maxOutput: record.MaxOutputBytes, artifactMediaType: request.ArtifactMediaType,
		runPath: runPath, status: ports.AgentPending}
	if terminal, found, loadErr := adapter.loadCausalTerminal(runPath, terminalRequestHash, record.SpecHash, record.MaxOutputBytes); loadErr != nil {
		return ports.AgentLaunchReceipt{}, loadErr
	} else if found {
		state.status, state.terminal, state.terminalDurable = terminal.Status, &terminal, true
		adapter.executions[executionKey] = state
		return receipt, nil
	}
	state.credentialGuard = credentialGuard
	credentialGuard = nil
	if session != nil {
		state.sessionGuard = session.guard
		session.guard = nil
	}
	if !recordCreated {
		adopted, adoptErr := adapter.adoptPersistedProcessLocked(state)
		if adoptErr != nil {
			destroyExecutionGuards(state)
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
	start := adapter.startExecutionLocked(ctx, callerContext, request, state, launchEnvironment, session)
	if start != nil {
		adapter.mu.Unlock()
		adapter.resolveExecutionStart(start)
		adapter.mu.Lock()
	}
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
	operationContext, endOperation, err := adapter.beginOperation(ctx)
	if err != nil {
		return ports.AgentObservation{}, err
	}
	defer endOperation()
	ctx = operationContext
	executionKey := executionRef.String()

	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	if adapter.closed {
		return ports.AgentObservation{}, &Error{Code: CodeUnavailable}
	}
	if state, found := adapter.executions[executionKey]; found {
		for state.starting != nil || state.quarantine != nil ||
			(state.startupRollback && state.terminal == nil && state.settled != nil) {
			starting := state.settled
			if state.starting != nil {
				starting = state.starting.resolved
			} else if state.quarantine != nil {
				starting = state.quarantine.done
			}
			adapter.mu.Unlock()
			select {
			case <-ctx.Done():
				adapter.mu.Lock()
				if errors.Is(context.Cause(ctx), errAdapterShutdown) {
					return ports.AgentObservation{}, &Error{Code: CodeUnavailable}
				}
				return ports.AgentObservation{}, ctx.Err()
			case <-starting:
			}
			adapter.mu.Lock()
			if ctx.Err() != nil {
				if errors.Is(context.Cause(ctx), errAdapterShutdown) {
					return ports.AgentObservation{}, &Error{Code: CodeUnavailable}
				}
				return ports.AgentObservation{}, ctx.Err()
			}
			if adapter.closed {
				return ports.AgentObservation{}, &Error{Code: CodeUnavailable}
			}
			current, stillFound := adapter.executions[executionKey]
			if !stillFound || current != state {
				break
			}
		}
		if current, stillFound := adapter.executions[executionKey]; stillFound && current == state {
			return adapter.observeStateLocked(executionKey, executionRef, state)
		}
	}

	record, runPath, found, err := adapter.loadLaunchRecord(executionRef)
	if err != nil {
		return ports.AgentObservation{}, err
	}
	if !found {
		return ports.AgentObservation{}, &Error{Code: CodeExecutionNotFound}
	}
	state, err := adapter.recoveryState(record, runPath, executionRef)
	if err != nil {
		return ports.AgentObservation{}, err
	}
	if terminal, terminalFound, loadErr := adapter.loadCausalTerminal(runPath, record.RequestHash, record.SpecHash, record.MaxOutputBytes); loadErr != nil {
		return ports.AgentObservation{}, loadErr
	} else if terminalFound {
		state.status, state.terminal, state.terminalDurable = terminal.Status, &terminal, true
	} else {
		process, processFound, processErr := adapter.readProcessRecord(runPath)
		if processErr != nil {
			return ports.AgentObservation{}, processErr
		}
		if processFound {
			if intent, intentFound, intentErr := adapter.loadQuarantineIntent(
				runPath, process,
			); intentErr != nil {
				return ports.AgentObservation{}, intentErr
			} else if intentFound {
				adapter.executions[executionKey] = state
				quarantineErr := adapter.quarantineExecutionLocked(
					ctx, state, &Error{Code: intent.ErrorCode},
				)
				if state.terminal == nil || !state.terminalDurable {
					return ports.AgentObservation{}, quarantineErr
				}
				return adapter.observeStateLocked(executionKey, executionRef, state)
			}
		}
		if recoveryErr := adapter.recoverExecutionGuards(ctx, record, state); recoveryErr != nil {
			if recoveryAuthorityFailure(recoveryErr) {
				quarantineCause := recoveryErr
				if record.SchemaVersion != stateSchemaVersion {
					quarantineCause = &Error{
						Code: CodeLegacyExecutionRequiresNewAttempt, Cause: recoveryErr,
					}
				}
				recoveryErr = adapter.quarantineExecutionLocked(ctx, state, quarantineCause)
			}
			return ports.AgentObservation{}, recoveryErr
		}
		adopted, adoptErr := adapter.adoptPersistedProcessLocked(state)
		if adoptErr != nil {
			destroyExecutionGuards(state)
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
	terminal := terminalRecord{SchemaVersion: stateSchemaVersion, RequestHash: state.terminalRequestHash,
		Status: ports.AgentFailed, ErrorCode: CodeExecutionInterrupted,
		ObservedAt: adapter.terminalTime(state.receipt.AcceptedAt)}
	var gateErr error
	terminal, gateErr = adapter.gateCredentialTerminalLocked(state, terminal)
	if gateErr != nil {
		return gateErr
	}
	if err := adapter.credentialOutputScrub(state.runPath); err != nil {
		return err
	}
	persisted, err := adapter.persistTerminal(state.runPath, terminal, state.receipt.SpecHash, state.maxOutput)
	if err != nil {
		return err
	}
	state.status, state.terminal, state.terminalDurable = persisted.Status, &persisted, true
	if err := adapter.cleanupTerminalCgroup(state); err != nil {
		return err
	}
	adapter.releaseProcessOwnershipLocked(state)
	return nil
}

func (adapter *Adapter) observeStateLocked(executionKey string, executionRef goal.ExecutionRef, state *executionState) (ports.AgentObservation, error) {
	if state.terminal == nil && state.process != nil && state.cancel == nil {
		treeGone, err := adapter.inspectProcessTree(*state.process)
		if err != nil {
			return ports.AgentObservation{}, err
		}
		if treeGone {
			if proof := state.stopProof.Load(); proof != nil {
				if err := adapter.finishStoppedProcessLocked(state, *proof); err != nil {
					return ports.AgentObservation{}, err
				}
			} else {
				hasStopRequest, requestErr := adapter.hasDurableStopRequest(state.runPath)
				if requestErr != nil {
					return ports.AgentObservation{}, requestErr
				}
				if hasStopRequest {
					if err := adapter.recoverInterruptedExecutionLocked(state); err != nil {
						return ports.AgentObservation{}, err
					}
				} else {
					if _, result, completionFound := adapter.loadCompletionProof(state); completionFound {
						clearBytes(result)
						if err := adapter.finishSupervisedProcessLocked(state); err != nil {
							return ports.AgentObservation{}, err
						}
					} else {
						if err := adapter.recoverInterruptedExecutionLocked(state); err != nil {
							return ports.AgentObservation{}, err
						}
					}
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
	if state.terminal != nil && state.terminalDurable {
		if err := adapter.cleanupTerminalCgroup(state); err != nil {
			return ports.AgentObservation{}, err
		}
		adapter.releaseProcessOwnershipLocked(state)
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
		Usage:        unknownCodexUsage(),
		ObservedAt:   observedAt.UTC(),
	}, nil
}

func (adapter *Adapter) Shutdown(ctx context.Context) error {
	if adapter == nil {
		return nil
	}
	// Cancellation is deliberately published before taking mu. A launch may be
	// outside the mutex waiting for supervisor READY, and an operation already
	// inside the mutex still receives the lifecycle cause immediately.
	adapter.cancelLifecycle(errAdapterShutdown)
	adapter.mu.Lock()
	if !adapter.closed {
		adapter.closed = true
	}
	adapter.shutdownOnce.Do(func() {
		shutdownCtx, cancel := context.WithTimeout(ctx, adapter.config.SupervisorStartTimeout)
		operationsDone := adapter.operationsDone
		go adapter.finalizeShutdown(shutdownCtx, cancel, operationsDone)
	})
	done := adapter.shutdownDone
	adapter.mu.Unlock()

	waitContext := ctx
	cancelWait := func() {}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		waitContext, cancelWait = context.WithTimeout(ctx, adapter.config.SupervisorStartTimeout)
	}
	defer cancelWait()
	select {
	case <-waitContext.Done():
		return waitContext.Err()
	case <-done:
		return adapter.shutdownErr
	}
}

func (adapter *Adapter) finalizeShutdown(
	shutdownCtx context.Context,
	cancel context.CancelFunc,
	operationsDone <-chan struct{},
) {
	defer cancel()
	// The coordinator is unique. Deadlines bound the caller and the stop phase,
	// but never authorize closing root/cgroup descriptors while an operation or
	// command waiter still owns them. Once those owners cooperate, this same
	// goroutine resumes finalization and closes shutdownDone exactly once.
	<-operationsDone
	adapter.mu.Lock()
	adopted, discoveryErr := adapter.adoptedProcessesLocked(context.WithoutCancel(shutdownCtx))
	adapter.mu.Unlock()
	adoptedErr := adapter.stopAdoptedProcesses(shutdownCtx, adopted)

	adapter.mu.Lock()
	waitsDone := adapter.waitsDone
	adapter.mu.Unlock()
	<-waitsDone

	shutdownErr := errors.Join(
		discoveryErr,
		adoptedErr,
		closeAccountProfileLease(adapter.accountProfileLock),
		adapter.root.Close(),
	)
	if adapter.cgroups != nil {
		shutdownErr = errors.Join(shutdownErr, adapter.cgroups.close())
	}
	adapter.mu.Lock()
	adapter.shutdownErr = shutdownErr
	close(adapter.shutdownDone)
	adapter.mu.Unlock()
}

func (adapter *Adapter) beginOperation(ctx context.Context) (context.Context, func(), error) {
	adapter.mu.Lock()
	if adapter.closed || context.Cause(adapter.lifecycle) != nil {
		adapter.mu.Unlock()
		return nil, nil, &Error{Code: CodeUnavailable}
	}
	if adapter.operations == 0 {
		adapter.operationsDone = make(chan struct{})
	}
	adapter.operations++
	adapter.mu.Unlock()

	operationContext, cancelOperation := context.WithCancelCause(ctx)
	stopLifecycle := context.AfterFunc(adapter.lifecycle, func() {
		cancelOperation(errAdapterShutdown)
	})
	var once sync.Once
	end := func() {
		once.Do(func() {
			stopLifecycle()
			cancelOperation(errExecutionFinished)
			adapter.mu.Lock()
			adapter.operations--
			if adapter.operations == 0 {
				close(adapter.operationsDone)
			}
			adapter.mu.Unlock()
		})
	}
	return operationContext, end, nil
}

func (adapter *Adapter) beginExecutionWaitLocked() {
	if adapter.activeWaits == 0 {
		adapter.waitsDone = make(chan struct{})
	}
	adapter.activeWaits++
}

func (adapter *Adapter) endExecutionWait() {
	adapter.mu.Lock()
	adapter.activeWaits--
	if adapter.activeWaits == 0 {
		close(adapter.waitsDone)
	}
	adapter.mu.Unlock()
}

type adoptedProcess struct {
	state  *executionState
	record processRecord
}

func (adapter *Adapter) adoptedProcessesLocked(ctx context.Context) ([]adoptedProcess, error) {
	discoveryErr := adapter.discoverShutdownProcessesLocked(ctx)
	processes := make([]adoptedProcess, 0)
	for _, state := range adapter.executions {
		if state.process == nil || state.ownerLock == nil || state.cancel != nil {
			continue
		}
		processes = append(processes, adoptedProcess{state: state, record: *state.process})
	}
	return processes, discoveryErr
}

func (adapter *Adapter) stopAdoptedProcesses(ctx context.Context, processes []adoptedProcess) error {
	defer adapter.releaseAdoptedProcessOwnership(processes)
	errorsByProcess := make([]error, len(processes))
	for index, process := range processes {
		err := adapter.shutdownSignal(process.record, ports.AgentStopForced)
		if err != nil && !errors.Is(err, os.ErrProcessDone) {
			errorsByProcess[index] = err
		}
	}
	for index, process := range processes {
		observedGone := false
		if errorsByProcess[index] == nil {
			observedGone, errorsByProcess[index] = waitForExactProcessWithInspector(ctx, process.record, nil, adapter.shutdownInspect)
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

func (adapter *Adapter) releaseAdoptedProcessOwnership(processes []adoptedProcess) {
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	for _, process := range processes {
		if process.state.process != nil && *process.state.process == process.record {
			adapter.releaseProcessOwnershipLocked(process.state)
		}
		destroyExecutionGuards(process.state)
	}
}

func (adapter *Adapter) Close() error {
	return adapter.Shutdown(context.Background())
}

var _ application.AgentLauncher = (*Adapter)(nil)
var _ application.AgentObserver = (*Adapter)(nil)
var _ application.AgentController = (*Adapter)(nil)
