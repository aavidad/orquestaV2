package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"orquesta/internal/adapters/agent/codex"
	"orquesta/internal/adapters/artifact/filesystem"
	"orquesta/internal/adapters/auth/localtoken"
	statesqlite "orquesta/internal/adapters/state/sqlite"
	"orquesta/internal/adapters/system/local"
	"orquesta/internal/application"
	"orquesta/internal/config"
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
	ConfigPath   string
	Version      string
	Listener     net.Listener
	AgentFactory AgentFactory
	ReportError  func(error)
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
	snapshot, err := config.Load(config.LoadOptions{FilePath: options.ConfigPath})
	if err != nil {
		return nil, err
	}
	if err := validateSnapshot(snapshot, options.ConfigPath); err != nil {
		return nil, err
	}
	authenticator, err := localtoken.Open(snapshot.Identity.LocalTokenPath)
	if err != nil {
		return nil, err
	}
	if err := snapshot.WriteEffective(); err != nil {
		return nil, err
	}

	clock := local.Clock{}
	repository, err := statesqlite.Open(ctx, statesqlite.Options{
		Path: snapshot.State.SQLite.Path, BusyTimeout: snapshot.State.SQLite.BusyTimeout,
		MaxOpenConnections: int(snapshot.State.SQLite.MaxOpenConnections),
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

	artifacts, err := filesystem.Open(snapshot.Artifact.Filesystem.Root)
	if err != nil {
		return nil, err
	}
	cleanupArtifacts := true
	defer func() {
		if cleanupArtifacts {
			_ = artifacts.Close()
		}
	}()

	actorRef, err := goal.NewActorRef(snapshot.Identity.LocalActor)
	if err != nil {
		return nil, err
	}
	projectRef, err := goal.NewProjectRef(snapshot.Project.Default)
	if err != nil {
		return nil, err
	}
	identityProvider, err := identity.NewLocalOwnerProvider(actorRef, projectRef)
	if err != nil {
		return nil, err
	}
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
			shutdownCtx, cancel := context.WithTimeout(context.Background(), snapshot.Server.ShutdownTimeout)
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

	orchestrator, err := application.New(application.Dependencies{
		State: repository, Launcher: agent, Observer: agent, Artifacts: artifacts,
		Clock: clock, IDs: local.IDGenerator{},
		MaxOutputBytes:    snapshot.Runtime.MaxOutputBytes,
		MaxActionAttempts: uint64(snapshot.Scheduler.MaxActionAttempts),
		ClaimLease:        snapshot.Scheduler.ClaimLease,
		ObservationDelay:  snapshot.Scheduler.ObservationInterval,
		ExecutionTimeout:  snapshot.Scheduler.ExecutionTimeout,
	})
	if err != nil {
		return nil, err
	}
	catalog, err := i18n.LoadBundled()
	if err != nil {
		return nil, err
	}
	version := strings.TrimSpace(options.Version)
	if version == "" {
		version = "dev"
	}
	interfaceServer, err := mcpiface.New(mcpiface.Config{
		Orchestrator: orchestrator, Identity: identityProvider, Catalog: catalog,
		Locale: snapshot.API.Locale, MaxListLimit: int(snapshot.API.MaxListLimit),
		MaxRequestBytes: snapshot.Server.MaxRequestBytes, Version: version,
	})
	if err != nil {
		return nil, err
	}
	listener := options.Listener
	if listener == nil {
		listener, err = net.Listen("tcp", snapshot.Server.Listen)
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
	mux := http.NewServeMux()
	mux.Handle(snapshot.Server.MCPPath, authenticator.Middleware(interfaceServer.Handler()))
	lifecycleCtx, cancelLifecycle := context.WithCancel(context.Background())
	cleanupLifecycle := true
	defer func() {
		if cleanupLifecycle {
			cancelLifecycle()
		}
	}()
	httpServer := &http.Server{
		Handler: mux, ReadTimeout: snapshot.Server.ReadTimeout,
		WriteTimeout: snapshot.Server.WriteTimeout, IdleTimeout: snapshot.Server.IdleTimeout,
		BaseContext: func(net.Listener) context.Context { return lifecycleCtx },
	}
	workerRef, err := local.IDGenerator{}.NewID(ctx, "worker")
	if err != nil {
		return nil, err
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
			pollInterval: runtime.config.Scheduler.PollInterval, report: runtime.reportError,
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
			shutdownCtx, cancel := context.WithTimeout(context.Background(), runtime.config.Server.ShutdownTimeout)
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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), runtime.config.Server.ShutdownTimeout)
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
	return "http://" + runtime.Address() + runtime.config.Server.MCPPath
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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), runtime.config.Server.ShutdownTimeout)
		defer cancel()
		return errors.Join(err, runtime.Shutdown(shutdownCtx))
	}
	serveErr := runtime.Wait()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), runtime.config.Server.ShutdownTimeout)
	defer cancel()
	return errors.Join(serveErr, runtime.Shutdown(shutdownCtx))
}

func productionAgentFactory(snapshot config.Snapshot, clock application.Clock) (AgentAdapter, error) {
	if snapshot.Runtime.Provider != "codex" {
		return nil, errors.New("bootstrap.runtime_provider_unsupported")
	}
	if snapshot.Runtime.Codex.CredentialRef != "" {
		return nil, errors.New("bootstrap.credential_resolver_unavailable")
	}
	environment, err := config.ResolveChildEnvironment(snapshot.Runtime.Codex.EnvAllowlist)
	if err != nil {
		return nil, err
	}
	return codex.New(codex.Config{
		Command:                 snapshot.Runtime.Codex.Command,
		WorkRoot:                snapshot.Runtime.Codex.WorkRoot,
		Model:                   snapshot.Runtime.Codex.Model,
		ReasoningEffort:         snapshot.Runtime.Codex.Reasoning,
		Timeout:                 snapshot.Runtime.Codex.Timeout,
		ProcessPipeDrainDelay:   snapshot.Runtime.Codex.ProcessPipeDrainDelay,
		MaxDiagnosticBytes:      snapshot.Runtime.Codex.MaxDiagnosticBytes,
		MaxConcurrentExecutions: int(snapshot.Runtime.Codex.MaxConcurrentExecutions),
		Environment:             environment, Now: clock.Now,
	})
}

func validateSnapshot(snapshot config.Snapshot, sourceConfigPath string) error {
	host, _, err := net.SplitHostPort(snapshot.Server.Listen)
	if err != nil {
		return errors.New("bootstrap.listen_invalid")
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("bootstrap.listen_must_be_loopback")
	}
	if !isLiteralMCPPath(snapshot.Server.MCPPath) {
		return errors.New("bootstrap.mcp_path_invalid")
	}
	if snapshot.Runtime.Codex.Timeout >= snapshot.Scheduler.ExecutionTimeout {
		return errors.New("bootstrap.execution_timeout_invalid")
	}
	if err := validateRuntimePaths(snapshot, sourceConfigPath); err != nil {
		return err
	}
	return nil
}

func isLiteralMCPPath(value string) bool {
	if !strings.HasPrefix(value, "/") || value == "/" || path.Clean(value) != value {
		return false
	}
	for _, character := range value {
		switch {
		case character >= 'a' && character <= 'z':
		case character >= 'A' && character <= 'Z':
		case character >= '0' && character <= '9':
		case strings.ContainsRune("/-._~", character):
		default:
			return false
		}
	}
	return true
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
	statePath, err := canonicalRuntimePath(snapshot.State.SQLite.Path)
	if err != nil {
		return errors.New("bootstrap.state_path_invalid")
	}
	artifactRoot, err := canonicalRuntimePath(snapshot.Artifact.Filesystem.Root)
	if err != nil {
		return errors.New("bootstrap.artifact_path_invalid")
	}
	workRoot, err := canonicalRuntimePath(snapshot.Runtime.Codex.WorkRoot)
	if err != nil {
		return errors.New("bootstrap.work_path_invalid")
	}
	effectivePath, err := canonicalRuntimePath(snapshot.Effective.Path)
	if err != nil {
		return errors.New("bootstrap.effective_path_invalid")
	}
	tokenPath, err := canonicalRuntimePath(snapshot.Identity.LocalTokenPath)
	if err != nil {
		return errors.New("bootstrap.local_token_path_invalid")
	}
	tokenDirectory := filepath.Dir(tokenPath)
	if pathsOverlap(filepath.Dir(statePath), artifactRoot) ||
		pathsOverlap(filepath.Dir(statePath), workRoot) || pathsOverlap(artifactRoot, workRoot) ||
		pathsOverlap(effectivePath, statePath) || pathsOverlap(effectivePath, artifactRoot) ||
		pathsOverlap(effectivePath, workRoot) || pathsOverlap(tokenDirectory, filepath.Dir(statePath)) ||
		pathsOverlap(tokenDirectory, artifactRoot) || pathsOverlap(tokenDirectory, workRoot) ||
		pathsOverlap(tokenDirectory, effectivePath) {
		return errors.New("bootstrap.runtime_paths_overlap")
	}
	if strings.TrimSpace(sourceConfigPath) != "" {
		configPath, err := canonicalRuntimePath(sourceConfigPath)
		if err != nil {
			return errors.New("bootstrap.config_path_invalid")
		}
		if pathsOverlap(configPath, statePath) || pathsOverlap(configPath, artifactRoot) ||
			pathsOverlap(configPath, workRoot) || pathsOverlap(configPath, effectivePath) ||
			pathsOverlap(configPath, tokenDirectory) {
			return errors.New("bootstrap.runtime_paths_overlap")
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
