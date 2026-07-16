package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"orquesta/internal/adapters/agent/codex"
	"orquesta/internal/adapters/artifact/filesystem"
	"orquesta/internal/adapters/auth/bearer"
	"orquesta/internal/adapters/auth/localtoken"
	"orquesta/internal/adapters/auth/oidc"
	configtoml "orquesta/internal/adapters/config/toml"
	credentiallocal "orquesta/internal/adapters/credentials/local"
	statesqlite "orquesta/internal/adapters/state/sqlite"
	"orquesta/internal/adapters/system/local"
	"orquesta/internal/application"
	"orquesta/internal/config"
	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/i18n"
	"orquesta/internal/identity"
	mcpiface "orquesta/internal/interfaces/mcp"
	"orquesta/internal/ports"
)

type AgentAdapter interface {
	application.AgentLauncher
	application.AgentObserver
	Shutdown(context.Context) error
}

type AgentFactory func(config.Snapshot, application.Clock) (AgentAdapter, error)

type Options struct {
	ConfigPath         string
	Version            string
	Listener           net.Listener
	AgentFactory       AgentFactory
	IdentityHTTPClient *http.Client
	ReportError        func(error)
}

type identityRuntimeComposition struct {
	provider       identity.IdentityProvider
	localPrincipal identity.Principal
	localHierarchy identity.ProjectHierarchy
	provisionLocal bool
}

type Runtime struct {
	config          config.Snapshot
	orchestrator    *application.Orchestrator
	repository      *statesqlite.Repository
	artifacts       *filesystem.Store
	agent           AgentAdapter
	listener        net.Listener
	httpServer      *http.Server
	workerRef       string
	reportError     func(error)
	lifecycleCtx    context.Context
	cancelLifecycle context.CancelFunc

	mu              sync.Mutex
	started         bool
	stopping        bool
	stopped         bool
	schedulerCancel context.CancelFunc
	schedulerDone   chan struct{}
	serveDone       chan struct{}
	serveErr        error
	shutdownOnce    sync.Once
	shutdownDone    chan struct{}
	shutdownErr     error
}

func Build(ctx context.Context, options Options) (*Runtime, error) {
	snapshot, err := loadConfigSnapshot(ctx, options.ConfigPath)
	if err != nil {
		return nil, err
	}
	if err := validateSnapshot(snapshot, options.ConfigPath); err != nil {
		return nil, err
	}
	clock := local.Clock{}
	catalog, err := i18n.LoadBundled()
	if err != nil {
		return nil, err
	}
	workerRef, err := local.IDGenerator{}.NewID(ctx, "worker")
	if err != nil {
		return nil, err
	}

	// Validate or acquire the reversible network boundary before creating any
	// durable composition state. Injected listeners remain owned by Build and
	// are closed on every failed build.
	listener := options.Listener
	if listener == nil {
		listener, err = net.Listen("tcp", snapshot.ServerListen())
		if err != nil {
			return nil, fmt.Errorf("bootstrap.listen_failed: %w", err)
		}
	}
	cleanupListener := true
	defer func() {
		if cleanupListener {
			_ = listener.Close()
		}
	}()
	if err := validateListener(listener); err != nil {
		return nil, err
	}

	// Agent validation may inspect its executable and allocate its isolated
	// work root, but it must succeed before token/config/state/artifact writes.
	factory := options.AgentFactory
	if factory == nil {
		factory = productionAgentFactory
	}
	agent, err := factory(snapshot, clock)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, errors.New("bootstrap.agent_factory_returned_nil")
	}
	cleanupAgent := true
	defer func() {
		if cleanupAgent {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), snapshot.ServerShutdownTimeout())
			defer cancel()
			_ = agent.Shutdown(shutdownCtx)
		}
	}()
	capabilities, err := agent.Capabilities(ctx)
	if err != nil {
		return nil, err
	}
	if err := ports.ValidateAgentCapabilities(capabilities); err != nil {
		return nil, err
	}

	identityComposition, err := composeIdentityRuntime(ctx, snapshot, options.IdentityHTTPClient)
	if err != nil {
		return nil, err
	}
	authenticator, err := bearer.New(identityComposition.provider)
	if err != nil {
		return nil, err
	}
	if err := writeEffectiveSnapshot(ctx, snapshot); err != nil {
		return nil, err
	}

	repository, err := statesqlite.Open(ctx, statesqlite.Options{
		Path: snapshot.StateSQLitePath(), BusyTimeout: snapshot.StateSQLiteBusyTimeout(),
		MaxOpenConnections: int(snapshot.StateSQLiteMaxOpenConnections()),
		Now:                clock.Now,
	})
	if err != nil {
		return nil, err
	}
	cleanupRepository := true
	defer func() {
		if cleanupRepository {
			_ = repository.Close()
		}
	}()
	if identityComposition.provisionLocal {
		if err := repository.ProvisionLocalAccess(
			ctx, identityComposition.localPrincipal, identityComposition.localHierarchy,
			identity.RoleProjectOwner, clock.Now(),
		); err != nil {
			return nil, err
		}
	}

	artifacts, err := filesystem.Open(snapshot.ArtifactFilesystemRoot())
	if err != nil {
		return nil, err
	}
	cleanupArtifacts := true
	defer func() {
		if cleanupArtifacts {
			_ = artifacts.Close()
		}
	}()

	orchestrator, err := application.New(application.Dependencies{
		State: repository, Access: repository,
		Launcher: agent, Observer: agent, Artifacts: artifacts,
		Clock: clock, IDs: local.IDGenerator{},
		MaxOutputBytes:          snapshot.RuntimeMaxOutputBytes(),
		MaxMailboxEnvelopeBytes: snapshot.MailboxMaxEnvelopeBytes(),
		MaxExecutionAttempts:    uint64(snapshot.SchedulerMaxExecutionAttempts()),
		ClaimLease:              snapshot.SchedulerClaimLease(),
		DirectorLeaseDuration:   snapshot.DirectorLeaseDuration(),
		ObservationDelay:        snapshot.SchedulerObservationInterval(),
		ExecutionTimeout:        snapshot.SchedulerExecutionTimeout(),
		AgentCapabilities:       capabilities,
	})
	if err != nil {
		return nil, err
	}
	version := strings.TrimSpace(options.Version)
	if version == "" {
		version = "dev"
	}
	interfaceServer, err := mcpiface.New(mcpiface.Config{
		Orchestrator: orchestrator, Identity: identity.ContextProvider{}, Catalog: catalog,
		Locale: snapshot.APILocale(), MaxListLimit: int(snapshot.APIMaxListLimit()),
		MaxRequestBytes: snapshot.ServerMaxRequestBytes(), Version: version,
	})
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.Handle(snapshot.ServerMCPPath(), authenticator.Wrap(interfaceServer.Handler()))
	lifecycleCtx, cancelLifecycle := context.WithCancel(context.Background())
	cleanupLifecycle := true
	defer func() {
		if cleanupLifecycle {
			cancelLifecycle()
		}
	}()
	httpServer := &http.Server{
		Handler: mux, ReadTimeout: snapshot.ServerReadTimeout(),
		WriteTimeout: snapshot.ServerWriteTimeout(), IdleTimeout: snapshot.ServerIdleTimeout(),
		BaseContext: func(net.Listener) context.Context { return lifecycleCtx },
	}
	runtime := &Runtime{
		config: snapshot, orchestrator: orchestrator, repository: repository,
		artifacts: artifacts, agent: agent, listener: listener, httpServer: httpServer,
		workerRef: workerRef, reportError: options.ReportError,
		lifecycleCtx: lifecycleCtx, cancelLifecycle: cancelLifecycle,
		shutdownDone: make(chan struct{}),
	}
	cleanupRepository = false
	cleanupArtifacts = false
	cleanupAgent = false
	cleanupListener = false
	cleanupLifecycle = false
	return runtime, nil
}

func composeIdentityRuntime(
	ctx context.Context,
	snapshot config.Snapshot,
	httpClient *http.Client,
) (identityRuntimeComposition, error) {
	switch snapshot.IdentityProvider() {
	case localtoken.AuthenticationMethod:
		principal, hierarchy, err := localIdentityComposition(snapshot)
		if err != nil {
			return identityRuntimeComposition{}, err
		}
		credential, err := localtoken.Open(snapshot.IdentityLocalTokenPath())
		if err != nil {
			return identityRuntimeComposition{}, err
		}
		provider, err := credential.ForPrincipal(principal)
		if err != nil {
			return identityRuntimeComposition{}, err
		}
		return identityRuntimeComposition{
			provider: provider, localPrincipal: principal, localHierarchy: hierarchy, provisionLocal: true,
		}, nil
	case oidc.AuthenticationMethod:
		provider, err := oidc.New(ctx, oidc.Options{
			Issuer: snapshot.IdentityOIDCIssuer(), Audience: snapshot.IdentityOIDCAudience(),
			RequiredGroups: snapshot.IdentityOIDCRequiredGroups(), ClockSkew: snapshot.IdentityOIDCClockSkew(),
			UpstreamTimeout: snapshot.IdentityOIDCUpstreamTimeout(), HTTPClient: httpClient,
		})
		if err != nil {
			return identityRuntimeComposition{}, err
		}
		return identityRuntimeComposition{provider: provider}, nil
	default:
		return identityRuntimeComposition{}, errors.New("bootstrap.identity_provider_unsupported")
	}
}

func localIdentityComposition(snapshot config.Snapshot) (identity.Principal, identity.ProjectHierarchy, error) {
	actorRef, err := goal.NewActorRef(snapshot.IdentityLocalActor())
	if err != nil {
		return identity.Principal{}, identity.ProjectHierarchy{}, err
	}
	principalRef, err := identity.NewPrincipalRef(actorRef.String())
	if err != nil {
		return identity.Principal{}, identity.ProjectHierarchy{}, err
	}
	principal, err := identity.NewPrincipal(
		principalRef, actorRef, identity.PrincipalKindHuman, localtoken.AuthenticationMethod,
	)
	if err != nil {
		return identity.Principal{}, identity.ProjectHierarchy{}, err
	}

	projectValue := snapshot.ProjectDefault()
	workspaceRef, err := identity.NewWorkspaceRef(projectValue)
	if err != nil {
		return identity.Principal{}, identity.ProjectHierarchy{}, err
	}
	groupRef, err := identity.NewGroupRef(projectValue)
	if err != nil {
		return identity.Principal{}, identity.ProjectHierarchy{}, err
	}
	projectRef, err := goal.NewProjectRef(projectValue)
	if err != nil {
		return identity.Principal{}, identity.ProjectHierarchy{}, err
	}
	repositoryRef, err := identity.NewRepositoryRef(projectValue)
	if err != nil {
		return identity.Principal{}, identity.ProjectHierarchy{}, err
	}
	hierarchy, err := identity.NewProjectHierarchy(identity.ProjectHierarchyInput{
		WorkspaceRef: workspaceRef, GroupRef: groupRef, GroupParentWorkspaceRef: workspaceRef,
		ProjectRef: projectRef, ProjectParentGroupRef: groupRef,
		RepositoryRef: repositoryRef, RepositoryParentProjectRef: projectRef,
	})
	if err != nil {
		return identity.Principal{}, identity.ProjectHierarchy{}, err
	}
	return principal, hierarchy, nil
}

func (runtime *Runtime) Start(parent context.Context) error {
	if runtime == nil {
		return errors.New("bootstrap.runtime_unavailable")
	}
	if parent == nil {
		return errors.New("bootstrap.parent_context_required")
	}
	runtime.mu.Lock()
	if runtime.stopping || runtime.stopped {
		runtime.mu.Unlock()
		return errors.New("bootstrap.runtime_stopped")
	}
	if runtime.started {
		runtime.mu.Unlock()
		return errors.New("bootstrap.runtime_already_started")
	}
	runtime.started = true
	schedulerCtx, cancelScheduler := context.WithCancel(runtime.lifecycleCtx)
	runtime.schedulerCancel = cancelScheduler
	runtime.schedulerDone = make(chan struct{})
	runtime.serveDone = make(chan struct{})
	runtime.mu.Unlock()

	go func() {
		defer close(runtime.schedulerDone)
		scheduler{
			orchestrator: runtime.orchestrator, workerRef: runtime.workerRef,
			pollInterval: runtime.config.SchedulerPollInterval(), report: runtime.reportError,
		}.run(schedulerCtx)
	}()
	go func() {
		err := runtime.httpServer.Serve(runtime.listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		runtime.mu.Lock()
		runtime.serveErr = err
		runtime.mu.Unlock()
		runtime.cancelLifecycle()
		close(runtime.serveDone)
	}()
	go func() {
		select {
		case <-parent.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), runtime.config.ServerShutdownTimeout())
			defer cancel()
			_ = runtime.Shutdown(shutdownCtx)
		case <-runtime.shutdownDone:
		}
	}()
	return nil
}

func (runtime *Runtime) Wait() error {
	if runtime == nil {
		return errors.New("bootstrap.runtime_unavailable")
	}
	runtime.mu.Lock()
	serveDone := runtime.serveDone
	started := runtime.started
	runtime.mu.Unlock()
	if !started || serveDone == nil {
		return errors.New("bootstrap.runtime_not_started")
	}
	<-serveDone
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	return runtime.serveErr
}

func (runtime *Runtime) Shutdown(ctx context.Context) error {
	if runtime == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("bootstrap.shutdown_context_required")
	}
	runtime.shutdownOnce.Do(func() {
		runtime.mu.Lock()
		runtime.stopping = true
		runtime.mu.Unlock()
		go runtime.performShutdown()
	})
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-runtime.shutdownDone:
		return runtime.shutdownErr
	}
}

func (runtime *Runtime) performShutdown() {
	defer func() {
		runtime.mu.Lock()
		runtime.stopped = true
		runtime.stopping = false
		runtime.mu.Unlock()
		close(runtime.shutdownDone)
	}()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), runtime.config.ServerShutdownTimeout())
	defer cancel()
	var failures []error
	runtime.cancelLifecycle()
	runtime.mu.Lock()
	cancelScheduler := runtime.schedulerCancel
	schedulerDone := runtime.schedulerDone
	runtime.mu.Unlock()
	if cancelScheduler != nil {
		cancelScheduler()
	}
	if err := runtime.httpServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		failures = append(failures, err)
		if closeErr := runtime.httpServer.Close(); closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
			failures = append(failures, closeErr)
		}
	}
	if err := runtime.agent.Shutdown(shutdownCtx); err != nil {
		failures = append(failures, err)
	}
	if schedulerDone != nil {
		select {
		case <-schedulerDone:
		case <-shutdownCtx.Done():
			failures = append(failures, shutdownCtx.Err())
		}
	}
	if err := runtime.listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		failures = append(failures, err)
	}
	if err := runtime.artifacts.Close(); err != nil {
		failures = append(failures, err)
	}
	if err := runtime.repository.Close(); err != nil {
		failures = append(failures, err)
	}
	runtime.shutdownErr = errors.Join(failures...)
}

func (runtime *Runtime) Address() string {
	if runtime == nil || runtime.listener == nil {
		return ""
	}
	return runtime.listener.Addr().String()
}

func (runtime *Runtime) MCPURL() string {
	if runtime == nil {
		return ""
	}
	return "http://" + runtime.Address() + runtime.config.ServerMCPPath()
}

func (runtime *Runtime) Orchestrator() *application.Orchestrator {
	if runtime == nil {
		return nil
	}
	return runtime.orchestrator
}

func Run(ctx context.Context, options Options) error {
	runtime, err := Build(ctx, options)
	if err != nil {
		return err
	}
	if err := runtime.Start(ctx); err != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), runtime.config.ServerShutdownTimeout())
		defer cancel()
		return errors.Join(err, runtime.Shutdown(shutdownCtx))
	}
	serveErr := runtime.Wait()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), runtime.config.ServerShutdownTimeout())
	defer cancel()
	return errors.Join(serveErr, runtime.Shutdown(shutdownCtx))
}

func productionAgentFactory(snapshot config.Snapshot, clock application.Clock) (AgentAdapter, error) {
	if snapshot.RuntimeProvider() != "codex" {
		return nil, errors.New("bootstrap.runtime_provider_unsupported")
	}
	environment, err := config.ResolveChildEnvironment(snapshot.RuntimeCodexEnvAllowlist())
	if err != nil {
		return nil, err
	}
	adapterConfig := codex.Config{
		Command:                 snapshot.RuntimeCodexCommand(),
		WorkRoot:                snapshot.RuntimeCodexWorkRoot(),
		Model:                   snapshot.RuntimeCodexModel(),
		ReasoningEffort:         snapshot.RuntimeCodexReasoning(),
		Timeout:                 snapshot.RuntimeCodexTimeout(),
		ProcessPipeDrainDelay:   snapshot.RuntimeCodexProcessPipeDrainDelay(),
		MaxDiagnosticBytes:      snapshot.RuntimeCodexMaxDiagnosticBytes(),
		MaxConcurrentExecutions: int(snapshot.RuntimeCodexMaxConcurrentExecutions()),
		Environment:             environment, Now: clock.Now,
	}
	credentialRef := snapshot.RuntimeCodexCredentialRef()
	if credentialRef == "" {
		return codex.New(adapterConfig)
	}
	store, err := credentiallocal.Open(credentiallocal.Options{
		Path: snapshot.CredentialsLocalPath(), OwnerUID: os.Geteuid(),
		MaxStoreBytes: snapshot.CredentialsLocalMaxDocumentBytes(), Now: clock.Now,
	})
	if err != nil {
		return nil, err
	}
	adapterConfig.CredentialStore = store
	adapterConfig.CredentialRef = credentials.CredentialRef(credentialRef)
	agent, err := codex.New(adapterConfig)
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	return &credentialAgent{AgentAdapter: agent, closeStore: store.Close}, nil
}

type credentialAgent struct {
	AgentAdapter
	closeStore func() error
	closeOnce  sync.Once
	closeErr   error
}

func (agent *credentialAgent) Shutdown(ctx context.Context) error {
	agentErr := agent.AgentAdapter.Shutdown(ctx)
	agent.closeOnce.Do(func() { agent.closeErr = agent.closeStore() })
	return errors.Join(agentErr, agent.closeErr)
}

func validateSnapshot(snapshot config.Snapshot, sourceConfigPath string) error {
	if err := validateRuntimePaths(snapshot, sourceConfigPath); err != nil {
		return err
	}
	for _, reserved := range configtoml.ReservedPaths(sourceConfigPath) {
		if err := validateRuntimePaths(snapshot, reserved); err != nil {
			return err
		}
	}
	return nil
}

func validateListener(listener net.Listener) error {
	if listener == nil {
		return errors.New("bootstrap.listener_required")
	}
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok || address.IP == nil || !address.IP.IsLoopback() {
		return errors.New("bootstrap.listener_must_be_loopback")
	}
	return nil
}

func validateRuntimePaths(snapshot config.Snapshot, sourceConfigPath string) error {
	statePath, err := canonicalRuntimePath(snapshot.StateSQLitePath())
	if err != nil {
		return errors.New("bootstrap.state_path_invalid")
	}
	artifactRoot, err := canonicalRuntimePath(snapshot.ArtifactFilesystemRoot())
	if err != nil {
		return errors.New("bootstrap.artifact_path_invalid")
	}
	workRoot, err := canonicalRuntimePath(snapshot.RuntimeCodexWorkRoot())
	if err != nil {
		return errors.New("bootstrap.work_path_invalid")
	}
	effectivePath, err := canonicalRuntimePath(snapshot.ConfigEffectivePath())
	if err != nil {
		return errors.New("bootstrap.effective_path_invalid")
	}
	tokenPath, err := canonicalRuntimePath(snapshot.IdentityLocalTokenPath())
	if err != nil {
		return errors.New("bootstrap.local_token_path_invalid")
	}
	credentialPath, err := canonicalRuntimePath(snapshot.CredentialsLocalPath())
	if err != nil {
		return errors.New("bootstrap.credential_path_invalid")
	}
	rawCredentialReservedPaths := credentiallocal.ReservedPaths(snapshot.CredentialsLocalPath())
	credentialReservedPaths := make([]string, 0, len(rawCredentialReservedPaths))
	for _, raw := range rawCredentialReservedPaths {
		reservedPath, err := canonicalRuntimePath(raw)
		if err != nil {
			return errors.New("bootstrap.credential_path_invalid")
		}
		credentialReservedPaths = append(credentialReservedPaths, reservedPath)
	}
	tokenDirectory := filepath.Dir(tokenPath)
	if pathsOverlap(filepath.Dir(statePath), artifactRoot) ||
		pathsOverlap(filepath.Dir(statePath), workRoot) || pathsOverlap(artifactRoot, workRoot) ||
		pathsOverlap(effectivePath, statePath) || pathsOverlap(effectivePath, artifactRoot) ||
		pathsOverlap(effectivePath, workRoot) || pathsOverlap(tokenDirectory, filepath.Dir(statePath)) ||
		pathsOverlap(tokenDirectory, artifactRoot) || pathsOverlap(tokenDirectory, workRoot) ||
		pathsOverlap(tokenDirectory, effectivePath) ||
		pathsOverlap(credentialPath, filepath.Dir(statePath)) || pathsOverlap(credentialPath, artifactRoot) ||
		pathsOverlap(credentialPath, workRoot) || pathsOverlap(credentialPath, effectivePath) ||
		pathsOverlap(credentialPath, tokenPath) {
		return errors.New("bootstrap.runtime_paths_overlap")
	}
	for _, reservedPath := range credentialReservedPaths {
		if pathsOverlap(reservedPath, filepath.Dir(statePath)) || pathsOverlap(reservedPath, artifactRoot) ||
			pathsOverlap(reservedPath, workRoot) || pathsOverlap(reservedPath, effectivePath) ||
			pathsOverlap(reservedPath, tokenPath) {
			return errors.New("bootstrap.runtime_paths_overlap")
		}
	}
	if strings.TrimSpace(sourceConfigPath) != "" {
		configPath, err := canonicalRuntimePath(sourceConfigPath)
		if err != nil {
			return errors.New("bootstrap.config_path_invalid")
		}
		if pathsOverlap(configPath, statePath) || pathsOverlap(configPath, artifactRoot) ||
			pathsOverlap(configPath, workRoot) || pathsOverlap(configPath, effectivePath) ||
			pathsOverlap(configPath, tokenDirectory) || pathsOverlap(configPath, credentialPath) {
			return errors.New("bootstrap.runtime_paths_overlap")
		}
		for _, reservedPath := range credentialReservedPaths {
			if pathsOverlap(configPath, reservedPath) {
				return errors.New("bootstrap.runtime_paths_overlap")
			}
		}
	}
	return nil
}

func canonicalRuntimePath(raw string) (string, error) {
	absolute, err := filepath.Abs(filepath.Clean(raw))
	if err != nil {
		return "", err
	}
	existing := absolute
	missing := make([]string, 0)
	for {
		_, statErr := os.Lstat(existing)
		if statErr == nil {
			break
		}
		if !errors.Is(statErr, os.ErrNotExist) {
			return "", statErr
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return "", statErr
		}
		missing = append(missing, filepath.Base(existing))
		existing = parent
	}
	resolved, err := filepath.EvalSymlinks(existing)
	if err != nil {
		return "", err
	}
	for index := len(missing) - 1; index >= 0; index-- {
		resolved = filepath.Join(resolved, missing[index])
	}
	return filepath.Clean(resolved), nil
}

func pathsOverlap(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if left == right {
		return true
	}
	leftRelative, leftErr := filepath.Rel(left, right)
	rightRelative, rightErr := filepath.Rel(right, left)
	return (leftErr == nil && leftRelative != ".." && !strings.HasPrefix(leftRelative, ".."+string(filepath.Separator))) ||
		(rightErr == nil && rightRelative != ".." && !strings.HasPrefix(rightRelative, ".."+string(filepath.Separator)))
}
