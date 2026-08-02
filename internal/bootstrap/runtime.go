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
	"time"

	"orquesta/internal/adapters/agent/codex"
	"orquesta/internal/adapters/agent/staticcapacity"
	"orquesta/internal/adapters/artifact/filesystem"
	"orquesta/internal/adapters/attestor/bubblewrap"
	"orquesta/internal/adapters/attestor/firecrackerclient"
	"orquesta/internal/adapters/attestor/firecrackerlauncher"
	"orquesta/internal/adapters/auth/bearer"
	"orquesta/internal/adapters/auth/executiontoken"
	"orquesta/internal/adapters/auth/localtoken"
	"orquesta/internal/adapters/auth/oidc"
	configtoml "orquesta/internal/adapters/config/toml"
	credentiallocal "orquesta/internal/adapters/credentials/local"
	statesqlite "orquesta/internal/adapters/state/sqlite"
	"orquesta/internal/adapters/system/local"
	gitlocal "orquesta/internal/adapters/workspace/gitlocal"
	"orquesta/internal/application"
	commandcore "orquesta/internal/commands"
	"orquesta/internal/config"
	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/i18n"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
	launcherprotocol "orquesta/internal/testattestorprotocol/launcher"
)

type AgentAdapter interface {
	application.AgentLauncher
	application.AgentObserver
	Shutdown(context.Context) error
}

type catalogoCapacidadColocacionAgente interface {
	DescribirCapacidadColocaciones() ([]application.DescriptorCapacidadColocacionAgente, error)
}

func componerFuentesCapacidadAgente(agente AgentAdapter, vigencia time.Duration, ahora func() time.Time, requerido bool) ([]application.FuenteCapacidadColocacionAgente, error) {
	catalogo, disponible := agente.(catalogoCapacidadColocacionAgente)
	if !disponible {
		if requerido {
			return nil, errors.New("bootstrap.agent_capacity_catalog_required")
		}
		return nil, nil
	}
	descriptores, err := catalogo.DescribirCapacidadColocaciones()
	if err != nil {
		return nil, err
	}
	if len(descriptores) == 0 {
		return nil, nil
	}
	resultado := make([]application.FuenteCapacidadColocacionAgente, 0, len(descriptores))
	for _, descriptor := range descriptores {
		if descriptor.BaseMedicion != application.BaseMedicionCapacidadBruta {
			return nil, errors.New("bootstrap.agent_capacity_measurement_invalid")
		}
		observador, err := staticcapacity.New(staticcapacity.Config{ReferenciaFuente: descriptor.SourceRef,
			ReferenciaPool: descriptor.PoolRef, Plazas: descriptor.Plazas, Vigencia: vigencia, Ahora: ahora})
		if err != nil {
			return nil, err
		}
		resultado = append(resultado, application.FuenteCapacidadColocacionAgente{descriptor.PlacementRef,
			descriptor.SourceRef, descriptor.PoolRef, descriptor.BaseMedicion, observador})
	}
	return resultado, nil
}

type AgentFactory func(config.Snapshot, application.Clock) (AgentAdapter, error)

type Options struct {
	ConfigPath                    string
	Version                       string
	Listener                      net.Listener
	AgentFactory                  AgentFactory
	IdentityHTTPClient            *http.Client
	CommandExecutionResolver      commandcore.ExecutionAuthorityResolver
	ReportError                   func(error)
	codexGoToolchainTrustForTests codexGoToolchainTrust
}

type identityRuntimeComposition struct {
	provider       identity.IdentityProvider
	localPrincipal identity.Principal
	localHierarchy identity.ProjectHierarchy
	provisionLocal bool
}

type Runtime struct {
	config             config.Snapshot
	orchestrator       *application.Orchestrator
	dispatcher         *commandcore.Dispatcher
	repository         *statesqlite.Repository
	artifacts          *filesystem.Store
	testAttestor       interface{ Close() error }
	workspace          *gitlocal.Adapter
	credentialStore    interface{ Close() error }
	agent              AgentAdapter
	controladoresCuota []application.ControladorCuotaAgente
	listener           net.Listener
	httpServer         *http.Server
	workerRef          string
	reportError        func(error)
	lifecycleCtx       context.Context
	cancelLifecycle    context.CancelFunc

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
	setup, err := prepareBuildSetup(ctx, options.ConfigPath)
	if err != nil {
		return nil, err
	}
	if options.AgentFactory == nil {
		if err := validateProductionAgentSelection(setup.snapshot); err != nil {
			return nil, err
		}
	}
	var cleanup buildCleanup
	defer cleanup.run()
	listener, err := openBuildListener(setup.snapshot, options.Listener)
	if err != nil {
		return nil, err
	}
	cleanup.add(func() { _ = listener.Close() })
	promptRenderer, err := newCatalogCodexPromptRenderer(setup.catalog, setup.snapshot.APILocale())
	if err != nil {
		return nil, err
	}
	var (
		agent               AgentAdapter
		capabilities        ports.AgentCapabilities
		controller          application.AgentController
		identityComposition identityRuntimeComposition
		credentialStore     *credentiallocal.Store
	)
	openAgent := func() error {
		agent, capabilities, controller, err = openBuildAgent(
			ctx,
			setup.snapshot,
			setup.clock,
			options.AgentFactory,
			promptRenderer,
			credentialStore,
			options.codexGoToolchainTrustForTests,
		)
		if err == nil {
			cleanup.add(func() { shutdownBuildAgent(agent, setup.snapshot.ServerShutdownTimeout()) })
		}
		return err
	}
	if options.AgentFactory != nil {
		if err := openAgent(); err != nil {
			return nil, err
		}
	}
	identityComposition, err = composeIdentityRuntime(ctx, setup.snapshot, options.IdentityHTTPClient)
	if err != nil {
		return nil, err
	}
	credentialStore, err = credentiallocal.Open(credentiallocal.Options{
		Path: setup.snapshot.CredentialsLocalPath(), OwnerUID: os.Geteuid(),
		MaxStoreBytes: setup.snapshot.CredentialsLocalMaxDocumentBytes(), Now: setup.clock.Now,
	})
	if err != nil {
		return nil, err
	}
	cleanup.add(func() { _ = credentialStore.Close() })
	if options.AgentFactory == nil {
		if err := openAgent(); err != nil {
			return nil, err
		}
	}
	var workspace *gitlocal.Adapter
	if setup.snapshot.RepositoryLocalSeedPath() != "" {
		workspace, err = openBuildWorkspace(ctx, setup.snapshot, setup.clock, identityComposition)
		if err != nil {
			return nil, err
		}
		cleanup.add(func() { _ = workspace.Close() })
		if err := bindAgentWorkspaceResolver(agent, workspace); err != nil {
			return nil, err
		}
	}
	testAttestor, err := openBuildTestAttestor(setup.snapshot, setup.clock, workspace)
	if err != nil {
		return nil, err
	}
	if testAttestor.closer != nil {
		cleanup.add(func() { _ = testAttestor.closer.Close() })
	}
	if err := writeEffectiveSnapshot(ctx, setup.snapshot); err != nil {
		return nil, err
	}
	repository, err := openBuildRepository(ctx, setup, identityComposition)
	if err != nil {
		return nil, err
	}
	cleanup.add(func() { _ = repository.Close() })
	if workspace != nil {
		if err := workspace.BindDurableWorkspaceBindingResolver(
			sqliteWorkspaceBindingResolver{repository: repository},
		); err != nil {
			return nil, err
		}
	}
	executionBroker, err := executiontoken.New(credentialStore, repository)
	if err != nil {
		return nil, err
	}
	identityProvider, err := executiontoken.NewRoutedProvider(identityComposition.provider, executionBroker)
	if err != nil {
		return nil, err
	}
	authenticator, err := bearer.New(identityProvider)
	if err != nil {
		return nil, err
	}
	mcpEndpoint := "http://" + listener.Addr().String() + setup.snapshot.ServerMCPPath()
	if options.AgentFactory == nil {
		sessionResolver, err := newCodexExecutionSessionResolver(repository, executionBroker, mcpEndpoint)
		if err != nil {
			return nil, err
		}
		if err := bindCodexAgentAuthority(ctx, agent, repository, sessionResolver); err != nil {
			return nil, err
		}
	} else if err := bindAgentRuntimeScope(ctx, agent, repository); err != nil {
		return nil, err
	}
	postArtifactMailbox, err := newLoopbackPostArtifactMailboxAdmitter(executionBroker, mcpEndpoint)
	if err != nil {
		return nil, err
	}
	artifacts, err := filesystem.Open(setup.snapshot.ArtifactFilesystemRoot())
	if err != nil {
		return nil, err
	}
	cleanup.add(func() { _ = artifacts.Close() })
	controladoresCuota, err := abrirControladoresCuota(ctx, setup, agent, repository, artifacts)
	if err != nil {
		return nil, err
	}
	cleanup.add(func() {
		_ = cerrarControladoresCuota(controladoresCuota, setup.snapshot.ServerShutdownTimeout())
	})
	fuentesCapacidad, err := componerFuentesCapacidadAgente(
		agent, setup.snapshot.RuntimeCapacityObservationTTL(), setup.clock.Now, options.AgentFactory == nil,
	)
	if err != nil {
		return nil, err
	}
	orchestrator, err := newBuildOrchestrator(
		setup, repository, artifacts, agent, controller, capabilities, workspace, testAttestor,
		executionRuntimeComposition{
			sessions: executionBroker, postArtifactMailbox: postArtifactMailbox,
			capacitySources: fuentesCapacidad,
		},
	)
	if err != nil {
		return nil, err
	}
	runtime, err := newBuildRuntime(
		setup, options, listener, agent, repository, artifacts, orchestrator, authenticator,
		testAttestor.closer, workspace, credentialStore, controladoresCuota,
	)
	if err != nil {
		return nil, err
	}
	cleanup.add(runtime.cancelLifecycle)
	cleanup.release()
	return runtime, nil
}

func prepareBuildSetup(ctx context.Context, configPath string) (buildSetup, error) {
	snapshot, err := loadConfigSnapshot(ctx, configPath)
	if err != nil {
		return buildSetup{}, err
	}
	if err := validateSnapshot(snapshot, configPath); err != nil {
		return buildSetup{}, err
	}
	clock := local.Clock{}
	policy, err := buildBudgetPolicy(snapshot, clock.Now())
	if err != nil {
		return buildSetup{}, err
	}
	catalog, err := i18n.LoadBundled()
	if err != nil {
		return buildSetup{}, err
	}
	workerRef, err := local.IDGenerator{}.NewID(ctx, "worker")
	if err != nil {
		return buildSetup{}, err
	}
	return buildSetup{
		snapshot: snapshot, clock: clock, budgetPolicy: policy, catalog: catalog, workerRef: workerRef,
	}, nil
}

type buildSetup struct {
	snapshot     config.Snapshot
	clock        local.Clock
	budgetPolicy application.BudgetPolicy
	catalog      *i18n.Catalog
	workerRef    string
}

const (
	testAttestorDisabled   = "disabled"
	testAttestorBubblewrap = "bubblewrap"
	testAttestorMicroVM    = "microvm"
)

type buildTestAttestorComposition struct {
	attestor application.TestAttestor
	policy   application.TestAttestationPolicy
	closer   interface{ Close() error }
}

type buildCleanup []func()

func (cleanup *buildCleanup) add(operation func()) { *cleanup = append(*cleanup, operation) }
func (cleanup *buildCleanup) release()             { *cleanup = nil }
func (cleanup *buildCleanup) run() {
	for index := len(*cleanup) - 1; index >= 0; index-- {
		(*cleanup)[index]()
	}
}

func openBuildListener(snapshot config.Snapshot, listener net.Listener) (net.Listener, error) {
	var err error
	if listener == nil {
		listener, err = net.Listen("tcp", snapshot.ServerListen())
		if err != nil {
			return nil, fmt.Errorf("bootstrap.listen_failed: %w", err)
		}
	}
	if err := validateListener(listener); err != nil {
		_ = listener.Close()
		return nil, err
	}
	return listener, nil
}

func openBuildAgent(
	ctx context.Context, snapshot config.Snapshot, clock local.Clock, factory AgentFactory,
	promptRenderer codex.PromptRenderer,
	credentialStore credentials.Store,
	toolchainTrust codexGoToolchainTrust,
) (AgentAdapter, ports.AgentCapabilities, application.AgentController, error) {
	if factory == nil {
		if toolchainTrust == nil {
			toolchainTrust = codexGoToolchainOwnerTrusted
		}
		factory = func(snapshot config.Snapshot, clock application.Clock) (AgentAdapter, error) {
			return productionAgentAdapterWithGoToolchainTrust(
				snapshot,
				clock,
				promptRenderer,
				credentialStore,
				toolchainTrust,
			)
		}
	}
	agent, err := factory(snapshot, clock)
	if err != nil {
		return nil, ports.AgentCapabilities{}, nil, err
	}
	if agent == nil {
		return nil, ports.AgentCapabilities{}, nil, errors.New("bootstrap.agent_factory_returned_nil")
	}
	capabilities, err := agent.Capabilities(ctx)
	if err == nil {
		err = ports.ValidateAgentCapabilities(capabilities)
	}
	if err != nil {
		shutdownBuildAgent(agent, snapshot.ServerShutdownTimeout())
		return nil, ports.AgentCapabilities{}, nil, err
	}
	controller, _ := agent.(application.AgentController)
	return agent, capabilities, controller, nil
}

type workspaceResolverBinder interface {
	BindWorkspacePathResolver(codex.WorkspacePathResolver) error
}

// localRepositoryLocator is deliberately a one-binding composition adapter.
// The runtime's bootstrap project is the only repository V16 may expose; an
// empty seed leaves non-code Goals usable while write work fails structurally.
type localRepositoryLocator struct {
	repositoryRef identity.RepositoryRef
	seedPath      string
	targetRef     string
}

type sqliteWorkspaceBindingResolver struct {
	repository *statesqlite.Repository
}

func (resolver sqliteWorkspaceBindingResolver) ResolveDurableWorkspaceBinding(
	ctx context.Context,
	ref ports.ExecutionWorkspaceRef,
) (gitlocal.DurableWorkspaceBinding, bool, error) {
	if resolver.repository == nil {
		return gitlocal.DurableWorkspaceBinding{}, false, errors.New("bootstrap.workspace_state_unavailable")
	}
	recovery, found, err := resolver.repository.WorkspaceBinding(ctx, ref)
	if err != nil || !found {
		return gitlocal.DurableWorkspaceBinding{}, found, err
	}
	binding := recovery.Binding
	if application.ValidateWorkspaceBinding(binding) != nil || binding.Ref != ref ||
		binding.AdapterRef != gitlocal.AdapterReference() {
		return gitlocal.DurableWorkspaceBinding{}, false, errors.New("bootstrap.workspace_binding_invalid")
	}
	request := ports.WorkspacePrepareRequest{
		WorkspaceRef: binding.Ref, PrincipalRef: binding.PrincipalRef,
		ActorRef: binding.ActorRef, ProjectRef: binding.ProjectRef, RepositoryRef: binding.RepositoryRef,
		GoalRef: binding.GoalRef, WorkItemRef: binding.WorkItemRef, ExecutionRef: binding.ExecutionRef,
		ExecutionAttempt: binding.ExecutionAttempt, PlanGeneration: binding.PlanGeneration,
		AppSpecGeneration: binding.AppSpecGeneration, AppSpecHash: binding.SpecHash,
		WriteSet: append([]string(nil), binding.WriteSet...), WriteSetDigest: binding.WriteSetDigest,
		// Application prepare requests canonically leave TargetRef empty; the
		// composition-owned locator supplies the configured target.
		TargetRef: "", IntentRef: binding.EffectIntentRef, AttemptRef: binding.EffectAttemptRef,
		ActionFence: binding.EffectFence, IdempotencyKey: recovery.PrepareIdempotencyKey,
		PreparedAt: binding.PreparedAt,
	}
	prepared := ports.WorkspacePrepared{
		WorkspaceRef: binding.Ref, RepositoryRef: binding.RepositoryRef, ExecutionRef: binding.ExecutionRef,
		TargetRef: binding.TargetRef, BaseOID: binding.BaseOID, ObjectFormat: binding.ObjectFormat,
		WriteSetDigest: binding.WriteSetDigest, AdapterRef: binding.AdapterRef,
		ReceiptRef: recovery.PreparedReceiptRef, PreparedAt: binding.PreparedAt,
	}
	if ports.ValidateWorkspacePrepareRequest(request) != nil ||
		ports.ValidateWorkspacePrepared(request, prepared) != nil {
		return gitlocal.DurableWorkspaceBinding{}, false, errors.New("bootstrap.workspace_binding_invalid")
	}
	return gitlocal.DurableWorkspaceBinding{Request: request, Prepared: prepared}, true, nil
}

func (locator localRepositoryLocator) LocateLocalRepository(_ context.Context, ref identity.RepositoryRef) (gitlocal.LocalRepositoryBinding, error) {
	if locator.repositoryRef.String() == "" || ref != locator.repositoryRef || locator.seedPath == "" {
		return gitlocal.LocalRepositoryBinding{}, errors.New("bootstrap.repository_seed_unavailable")
	}
	return gitlocal.LocalRepositoryBinding{Path: locator.seedPath, TargetRef: locator.targetRef}, nil
}

func openBuildWorkspace(
	ctx context.Context, snapshot config.Snapshot, clock application.Clock, composition identityRuntimeComposition,
) (*gitlocal.Adapter, error) {
	seedPath := snapshot.RepositoryLocalSeedPath()
	if seedPath == "" {
		return nil, errors.New("bootstrap.repository_seed_unavailable")
	}
	environment, err := config.ResolveChildEnvironment(snapshot.RuntimeCodexEnvAllowlist())
	if err != nil {
		return nil, err
	}
	gitCommand, err := resolveBootstrapExecutable("git", environment)
	if err != nil {
		return nil, err
	}
	seedPath, err = filepath.Abs(seedPath)
	if err != nil {
		return nil, err
	}
	return gitlocal.New(gitlocal.Config{
		Root: snapshot.WorkspaceLocalRoot(), GitCommand: gitCommand, Now: clock.Now,
		MaxSnapshotBytes:   snapshot.TestAttestorMaxSubjectBytes(),
		MaxSnapshotEntries: bubblewrap.SubjectEntryLimit(snapshot.TestAttestorMaxSubjectBytes()),
		Locator: localRepositoryLocator{
			repositoryRef: composition.localHierarchy.RepositoryRef(), seedPath: seedPath,
			targetRef: snapshot.RepositoryLocalTargetRef(),
		},
	})
}

func openBuildTestAttestor(
	snapshot config.Snapshot,
	clock application.Clock,
	workspace *gitlocal.Adapter,
) (buildTestAttestorComposition, error) {
	switch snapshot.TestAttestorProvider() {
	case testAttestorDisabled:
		return buildTestAttestorComposition{}, nil
	case testAttestorBubblewrap:
		if workspace == nil {
			return buildTestAttestorComposition{}, errors.New("bootstrap.test_attestor_workspace_required")
		}
		config := bubblewrapConfig(snapshot, workspace, clock.Now)
		attestor, err := bubblewrap.New(config)
		if err != nil {
			return buildTestAttestorComposition{}, err
		}
		identity := attestor.PolicyIdentity()
		policy := application.TestAttestationPolicy{Ref: identity.Ref, Digest: identity.Digest}
		if err := application.ValidateTestAttestationPolicy(policy); err != nil {
			_ = attestor.Close()
			return buildTestAttestorComposition{}, err
		}
		return buildTestAttestorComposition{attestor: attestor, policy: policy, closer: attestor}, nil
	case testAttestorMicroVM:
		if workspace == nil {
			return buildTestAttestorComposition{}, errors.New("bootstrap.test_attestor_workspace_required")
		}
		launcher, err := firecrackerlauncher.NewClient(
			snapshot.TestAttestorMicroVMLauncherSocket(),
		)
		if err != nil {
			return buildTestAttestorComposition{}, err
		}
		attestor, err := firecrackerclient.New(firecrackerConfig(snapshot, workspace, launcher, clock.Now))
		if err != nil {
			return buildTestAttestorComposition{}, err
		}
		identity := attestor.PolicyIdentity()
		policy := application.TestAttestationPolicy{Ref: identity.Ref, Digest: identity.Digest}
		if err := application.ValidateTestAttestationPolicy(policy); err != nil {
			_ = attestor.Close()
			return buildTestAttestorComposition{}, err
		}
		return buildTestAttestorComposition{attestor: attestor, policy: policy, closer: attestor}, nil
	default:
		return buildTestAttestorComposition{}, errors.New("bootstrap.test_attestor_provider_unsupported")
	}
}

func firecrackerConfig(
	snapshot config.Snapshot,
	source firecrackerclient.SnapshotStreamSource,
	launcher launcherprotocol.Client,
	now func() time.Time,
) firecrackerclient.Config {
	return firecrackerclient.Config{
		Launcher: launcher, SnapshotSource: source, Now: now,
		ExpectedAssetDigest: snapshot.TestAttestorMicroVMExpectedAssetDigest(),
		Limits: firecrackerclient.Limits{
			Timeout: snapshot.TestAttestorTimeout(), CleanupTimeout: snapshot.ServerShutdownTimeout(),
			MaxOutputBytes:    uint64(snapshot.RuntimeMaxOutputBytes()),
			MaxSubjectBytes:   snapshot.TestAttestorMaxSubjectBytes(),
			MaxConcurrentRuns: int(snapshot.TestAttestorMaxConcurrentRuns()),
			GuestMemoryMiB:    uint32(snapshot.TestAttestorMicroVMGuestMemoryMiB()),
			MemoryMaxBytes:    uint64(snapshot.TestAttestorMemoryMaxBytes()),
			PIDsMax:           uint32(snapshot.TestAttestorPIDsMax()),
			CPUQuotaMicros:    uint64(snapshot.TestAttestorCPUQuotaMicros()),
		},
	}
}

func bubblewrapConfig(snapshot config.Snapshot, source bubblewrap.SnapshotStreamSource, now func() time.Time) bubblewrap.Config {
	return bubblewrap.Config{
		BubblewrapCommand: snapshot.TestAttestorBubblewrapCommand(),
		ToolchainRoot:     snapshot.TestAttestorGoToolchainRoot(), CgroupRoot: snapshot.TestAttestorCgroupRoot(),
		SnapshotSource: source, Now: now,
		Limits: bubblewrap.Limits{
			Timeout: snapshot.TestAttestorTimeout(), CleanupTimeout: snapshot.ServerShutdownTimeout(),
			MaxOutputBytes: snapshot.RuntimeMaxOutputBytes(), MaxSubjectBytes: snapshot.TestAttestorMaxSubjectBytes(),
			MaxConcurrentRuns: snapshot.TestAttestorMaxConcurrentRuns(),
			MemoryMaxBytes:    snapshot.TestAttestorMemoryMaxBytes(), PIDsMax: snapshot.TestAttestorPIDsMax(),
			CPUQuotaMicros: snapshot.TestAttestorCPUQuotaMicros(),
		},
	}
}

func resolveBootstrapExecutable(name string, environment map[string]string) (string, error) {
	if name == "" || filepath.IsAbs(name) || strings.ContainsRune(name, filepath.Separator) {
		return "", errors.New("bootstrap.executable_invalid")
	}
	searchPath := environment["PATH"]
	for _, directory := range filepath.SplitList(searchPath) {
		if directory == "" || !filepath.IsAbs(directory) {
			continue
		}
		candidate := filepath.Join(directory, name)
		info, err := os.Stat(candidate)
		if err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0 {
			return candidate, nil
		}
	}
	return "", errors.New("bootstrap.executable_not_found")
}

func bindAgentWorkspaceResolver(agent AgentAdapter, resolver codex.WorkspacePathResolver) error {
	binder, ok := agent.(workspaceResolverBinder)
	if !ok {
		return errors.New("bootstrap.agent_workspace_resolver_unsupported")
	}
	return binder.BindWorkspacePathResolver(resolver)
}

func shutdownBuildAgent(agent AgentAdapter, timeout time.Duration) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_ = agent.Shutdown(shutdownCtx)
}

func openBuildRepository(
	ctx context.Context, setup buildSetup, composition identityRuntimeComposition,
) (*statesqlite.Repository, error) {
	repository, err := statesqlite.Open(ctx, statesqlite.Options{
		Path: setup.snapshot.StateSQLitePath(), BusyTimeout: setup.snapshot.StateSQLiteBusyTimeout(),
		MaxOpenConnections: int(setup.snapshot.StateSQLiteMaxOpenConnections()), Now: setup.clock.Now,
	})
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*statesqlite.Repository, error) { _ = repository.Close(); return nil, err }
	if composition.provisionLocal {
		err = repository.ProvisionLocalAccess(
			ctx, composition.localPrincipal, composition.localHierarchy, identity.RoleProjectOwner, setup.clock.Now(),
		)
		if err != nil {
			return fail(err)
		}
	}
	return repository, nil
}

func abrirControladoresCuota(
	ctx context.Context, setup buildSetup, agente AgentAdapter,
	estado application.StateRepository, artefactos application.ArtifactStore,
) ([]application.ControladorCuotaAgente, error) {
	iniciador, disponible := agente.(application.IniciadorControladoresCuotaAgente)
	if !disponible {
		return nil, nil
	}
	controladores, err := iniciador.IniciarControladoresCuota(context.Background(), application.ConfiguracionControladoresCuotaAgente{
		VigenciaObservacion: setup.snapshot.RuntimeCapacityObservationTTL(),
		DemoraReconexion:    setup.budgetPolicy.QuotaRetryDelay, Ahora: setup.clock.Now,
		Sumidero: func(ctx context.Context, observacion application.AgentQuotaObservation, evidencia []byte) error {
			return application.RegistrarObservacionCuota(ctx, estado, artefactos, observacion, evidencia)
		},
	})
	if err != nil {
		return nil, err
	}
	espera, cancelar := context.WithTimeout(ctx, setup.snapshot.RuntimeCapacityObservationTimeout())
	defer cancelar()
	for _, controlador := range controladores {
		if err := controlador.EsperarInicial(espera); err != nil {
			_ = cerrarControladoresCuota(controladores, setup.snapshot.ServerShutdownTimeout())
			return nil, err
		}
	}
	return controladores, nil
}

func cerrarControladoresCuota(controladores []application.ControladorCuotaAgente, limite time.Duration) (err error) {
	ctx, cancelar := context.WithTimeout(context.Background(), limite)
	defer cancelar()
	for _, controlador := range controladores {
		err = errors.Join(err, controlador.Cerrar(ctx))
	}
	return err
}

func newBuildOrchestrator(
	setup buildSetup, repository *statesqlite.Repository, artifacts *filesystem.Store,
	agent AgentAdapter, controller application.AgentController, capabilities ports.AgentCapabilities, workspace *gitlocal.Adapter,
	testAttestor buildTestAttestorComposition,
	execution ...executionRuntimeComposition,
) (*application.Orchestrator, error) {
	return application.New(buildOrchestratorDependencies(
		setup, repository, artifacts, agent, controller, capabilities, workspace, testAttestor, execution...,
	))
}

type executionRuntimeComposition struct {
	sessions            ports.ExecutionSessionBroker
	postArtifactMailbox application.PostArtifactMailboxAdmitter
	capacitySources     []application.FuenteCapacidadColocacionAgente
}

func buildOrchestratorDependencies(
	setup buildSetup, repository *statesqlite.Repository, artifacts *filesystem.Store,
	agent AgentAdapter, controller application.AgentController, capabilities ports.AgentCapabilities, workspace *gitlocal.Adapter,
	testAttestor buildTestAttestorComposition,
	execution ...executionRuntimeComposition,
) application.Dependencies {
	var composition executionRuntimeComposition
	if len(execution) == 1 {
		composition = execution[0]
	}
	return application.Dependencies{
		State: repository, WizardGapsStore: repository,
		IntakeDossierStore: repository, Access: repository,
		Launcher: agent, Observer: agent, Controller: controller, Artifacts: artifacts,
		WorkspaceManager: workspace, VersionControl: workspace,
		TestAttestor: testAttestor.attestor, TestAttestationPolicy: testAttestor.policy,
		Clock: setup.clock, IDs: local.IDGenerator{},
		MaxOutputBytes: setup.snapshot.RuntimeMaxOutputBytes(), MaxMailboxEnvelopeBytes: setup.snapshot.MailboxMaxEnvelopeBytes(),
		MaxExecutionAttempts:  uint64(setup.snapshot.SchedulerMaxExecutionAttempts()),
		MaxChildrenPerParent:  int(setup.snapshot.SchedulerMaxChildrenPerParent()),
		ClaimLease:            setup.snapshot.SchedulerClaimLease(),
		AttestTestClaimLease:  setup.snapshot.SchedulerAttestTestClaimLease(),
		DirectorLeaseDuration: setup.snapshot.DirectorLeaseDuration(),
		EffectApprovalTTL:     setup.snapshot.GovernanceEffectApprovalTTL(), BudgetPolicy: setup.budgetPolicy,
		ObservationDelay: setup.snapshot.SchedulerObservationInterval(), ExecutionTimeout: setup.snapshot.SchedulerExecutionTimeout(),
		AgentCapabilities: capabilities,
		ExecutionSessions: composition.sessions, PostArtifactMailbox: composition.postArtifactMailbox,
		CapacitySources: composition.capacitySources, CapacityObservationWait: setup.snapshot.RuntimeCapacityObservationTimeout(),
	}
}

func newBuildRuntime(
	setup buildSetup, options Options, listener net.Listener, agent AgentAdapter,
	repository *statesqlite.Repository, artifacts *filesystem.Store, orchestrator *application.Orchestrator,
	authenticator *bearer.Middleware,
	testAttestor interface{ Close() error },
	workspace *gitlocal.Adapter,
	credentialStore interface{ Close() error },
	controladoresCuota []application.ControladorCuotaAgente,
) (*Runtime, error) {
	version := strings.TrimSpace(options.Version)
	if version == "" {
		version = "dev"
	}
	executionResolver := options.CommandExecutionResolver
	if executionResolver == nil {
		executionResolver = repository
	}
	surfaces, err := newCommandSurfaces(
		setup, version, orchestrator, repository, executionResolver,
	)
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.Handle(setup.snapshot.ServerMCPPath(), authenticator.Wrap(surfaces.mcpHandler))
	mux.Handle(commandHTTPPrefix, authenticator.Wrap(surfaces.httpHandler))
	lifecycleCtx, cancelLifecycle := context.WithCancel(context.Background())
	httpServer := &http.Server{
		Handler: mux, ReadTimeout: setup.snapshot.ServerReadTimeout(),
		WriteTimeout: setup.snapshot.ServerWriteTimeout(), IdleTimeout: setup.snapshot.ServerIdleTimeout(),
		BaseContext: func(net.Listener) context.Context { return lifecycleCtx },
	}
	return &Runtime{
		config: setup.snapshot, orchestrator: orchestrator, dispatcher: surfaces.dispatcher, repository: repository,
		artifacts: artifacts, testAttestor: testAttestor, workspace: workspace, credentialStore: credentialStore,
		agent: agent, controladoresCuota: controladoresCuota, listener: listener, httpServer: httpServer,
		workerRef: setup.workerRef, reportError: options.ReportError,
		lifecycleCtx: lifecycleCtx, cancelLifecycle: cancelLifecycle,
		shutdownDone: make(chan struct{}),
	}, nil
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
		if snapshot.IdentityLocalPrincipalsManifestPath() != "" {
			secondaries, manifestErr := localtoken.OpenManifest(localtoken.ManifestOptions{
				Path: snapshot.IdentityLocalPrincipalsManifestPath(), OwnerUID: os.Geteuid(),
				MaxDocumentBytes: snapshot.IdentityLocalPrincipalsManifestMaxBytes(),
				MaxEntries:       int(snapshot.IdentityLocalPrincipalsManifestMaxEntries()),
				Reserved:         []identity.Principal{principal},
			})
			if manifestErr != nil {
				return identityRuntimeComposition{}, manifestErr
			}
			provider, err = localtoken.Combine(provider, secondaries)
			if err != nil {
				return identityRuntimeComposition{}, err
			}
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
			pollInterval:          runtime.config.SchedulerPollInterval(),
			maxConcurrentLaunches: dispatcherProcessSlotLimit(runtime.config),
			report:                runtime.reportError,
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

func dispatcherProcessSlotLimit(snapshot config.Snapshot) int64 {
	return snapshot.GovernanceGlobalProcessSlotsBudget()
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
	for _, controlador := range runtime.controladoresCuota {
		if err := controlador.Cerrar(shutdownCtx); err != nil {
			failures = append(failures, err)
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
	if runtime.testAttestor != nil {
		if err := runtime.testAttestor.Close(); err != nil {
			failures = append(failures, err)
		}
	}
	if runtime.workspace != nil {
		if err := runtime.workspace.Close(); err != nil {
			failures = append(failures, err)
		}
	}
	if runtime.credentialStore != nil {
		if err := runtime.credentialStore.Close(); err != nil {
			failures = append(failures, err)
		}
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
	runErr := runtime.Start(ctx)
	if runErr == nil {
		runErr = runtime.Wait()
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), runtime.config.ServerShutdownTimeout())
	defer cancel()
	return errors.Join(runErr, runtime.Shutdown(shutdownCtx))
}

func productionAgentFactory(
	snapshot config.Snapshot, clock application.Clock, renderers ...codex.PromptRenderer,
) (AgentAdapter, error) {
	if err := validateProductionAgentSelection(snapshot); err != nil {
		return nil, err
	}
	var promptRenderer codex.PromptRenderer
	if len(renderers) == 1 {
		promptRenderer = renderers[0]
	} else if len(renderers) != 0 {
		return nil, errors.New("bootstrap.codex_prompt_renderer_invalid")
	} else {
		catalog, err := i18n.LoadBundled()
		if err != nil {
			return nil, err
		}
		promptRenderer, err = newCatalogCodexPromptRenderer(catalog, snapshot.APILocale())
		if err != nil {
			return nil, err
		}
	}
	credentialRef := snapshot.RuntimeCodexCredentialRef()
	if credentialRef == "" {
		return productionAgentAdapterWithGoToolchainTrust(
			snapshot,
			clock,
			promptRenderer,
			nil,
			codexGoToolchainOwnerTrusted,
		)
	}
	store, err := credentiallocal.Open(credentiallocal.Options{
		Path: snapshot.CredentialsLocalPath(), OwnerUID: os.Geteuid(),
		MaxStoreBytes: snapshot.CredentialsLocalMaxDocumentBytes(), Now: clock.Now,
	})
	if err != nil {
		return nil, err
	}
	agent, err := productionAgentAdapter(snapshot, clock, promptRenderer, store)
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	return &credentialAgent{Adapter: agent, closeStore: store.Close}, nil
}

func productionAgentAdapter(
	snapshot config.Snapshot, clock application.Clock, promptRenderer codex.PromptRenderer, credentialStore credentials.Store,
) (*codex.Adapter, error) {
	adapterConfig, err := productionCodexAdapterConfigWithGoToolchainTrust(
		snapshot,
		clock,
		promptRenderer,
		credentialStore,
		codexGoToolchainOwnerTrusted,
	)
	if err != nil {
		return nil, err
	}
	return codex.New(adapterConfig)
}

func productionAgentAdapterWithGoToolchainTrust(
	snapshot config.Snapshot,
	clock application.Clock,
	promptRenderer codex.PromptRenderer,
	credentialStore credentials.Store,
	ownerTrusted codexGoToolchainTrust,
) (AgentAdapter, error) {
	adapterConfig, err := productionCodexAdapterConfigWithGoToolchainTrust(
		snapshot,
		clock,
		promptRenderer,
		credentialStore,
		ownerTrusted,
	)
	if err != nil {
		return nil, err
	}
	profiles := snapshot.RuntimeCodexAccountProfiles()
	if len(profiles) == 0 {
		return codex.New(adapterConfig)
	}
	accountHomes := make([]codex.AccountHome, len(profiles))
	for index, profile := range profiles {
		accountHomes[index] = codex.AccountHome{
			Root:    adapterConfig.AccountHomeRoot,
			Profile: profile,
		}
	}
	adapterConfig.AccountHomeRoot = ""
	adapterConfig.AccountProfile = ""
	return codex.NewPool(codex.PoolConfig{
		Adapter:      adapterConfig,
		AccountHomes: accountHomes,
	})
}

func productionCodexAdapterConfigWithGoToolchainTrust(
	snapshot config.Snapshot,
	clock application.Clock,
	promptRenderer codex.PromptRenderer,
	credentialStore credentials.Store,
	ownerTrusted codexGoToolchainTrust,
) (codex.Config, error) {
	if err := validateProductionAgentSelection(snapshot); err != nil {
		return codex.Config{}, err
	}
	environment, err := config.ResolveChildEnvironment(snapshot.RuntimeCodexEnvAllowlist())
	if err != nil {
		return codex.Config{}, err
	}
	environment, err = prepareCodexGoEnvironmentWithTrust(
		environment,
		snapshot.RuntimeCodexCacheRoot(),
		snapshot.RuntimeCodexGoToolchainRoot(),
		ownerTrusted,
	)
	if err != nil {
		return codex.Config{}, err
	}
	accountHomeRoot := snapshot.RuntimeCodexAccountHomeRoot()
	if accountHomeRoot != "" {
		accountHomeRoot, err = canonicalRuntimePath(accountHomeRoot)
		if err != nil {
			return codex.Config{}, errors.New("bootstrap.account_home_path_invalid")
		}
	}
	adapterConfig := codex.Config{
		Command:                     snapshot.RuntimeCodexCommand(),
		WorkRoot:                    snapshot.RuntimeCodexWorkRoot(),
		CgroupRoot:                  snapshot.RuntimeCodexCgroupRoot(),
		Model:                       snapshot.RuntimeCodexModel(),
		ReasoningEffort:             snapshot.RuntimeCodexReasoning(),
		Timeout:                     snapshot.RuntimeCodexTimeout(),
		ProcessPipeDrainDelay:       snapshot.RuntimeCodexProcessPipeDrainDelay(),
		SupervisorStartTimeout:      snapshot.RuntimeCodexSupervisorStartTimeout(),
		MaxDiagnosticBytes:          snapshot.RuntimeCodexMaxDiagnosticBytes(),
		AppServerMaxFrameBytes:      int(snapshot.RuntimeCodexAppServerMaxFrameBytes()),
		MaxConcurrentExecutions:     int(snapshot.RuntimeCodexMaxConcurrentExecutions()),
		MCPBearerTokenEnvVar:        snapshot.RuntimeCodexMCPBearerTokenEnvVar(),
		AccountHomeRoot:             accountHomeRoot,
		AccountProfile:              snapshot.RuntimeCodexAccountProfile(),
		AccountAuthMaxDocumentBytes: snapshot.RuntimeCodexAccountAuthMaxDocumentBytes(),
		PromptRenderer:              promptRenderer,
		Environment:                 environment, Now: clock.Now,
	}
	credentialRef := snapshot.RuntimeCodexCredentialRef()
	if credentialRef != "" {
		if credentialStore == nil {
			return codex.Config{}, errors.New("bootstrap.credential_store_required")
		}
		adapterConfig.CredentialStore = credentialStore
		adapterConfig.CredentialRef = credentials.CredentialRef(credentialRef)
	}
	return adapterConfig, nil
}

func validateProductionAgentSelection(snapshot config.Snapshot) error {
	if snapshot.RuntimeProvider() != "codex" {
		return errors.New("bootstrap.runtime_provider_unsupported")
	}
	if snapshot.RuntimeIsolation() != "process" {
		return errors.New("bootstrap.runtime_isolation_not_composed")
	}
	return nil
}

type credentialAgent struct {
	*codex.Adapter
	closeStore func() error
	closeOnce  sync.Once
	closeErr   error
}

func (agent *credentialAgent) Shutdown(ctx context.Context) error {
	agentErr := agent.Adapter.Shutdown(ctx)
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
	paths := []struct{ raw, code, value string }{
		{raw: snapshot.StateSQLitePath(), code: "bootstrap.state_path_invalid"},
		{raw: snapshot.ArtifactFilesystemRoot(), code: "bootstrap.artifact_path_invalid"},
		{raw: snapshot.RuntimeCodexWorkRoot(), code: "bootstrap.work_path_invalid"},
		{raw: snapshot.RuntimeCodexCacheRoot(), code: codexGoCacheInvalid},
		{raw: snapshot.ConfigEffectivePath(), code: "bootstrap.effective_path_invalid"},
		{raw: snapshot.IdentityLocalTokenPath(), code: "bootstrap.local_token_path_invalid"},
		{raw: snapshot.CredentialsLocalPath(), code: "bootstrap.credential_path_invalid"},
		{raw: snapshot.WorkspaceLocalRoot(), code: "bootstrap.workspace_path_invalid"},
	}
	for index := range paths {
		var err error
		if paths[index].value, err = canonicalRuntimePath(paths[index].raw); err != nil {
			return errors.New(paths[index].code)
		}
	}
	statePath, artifactRoot, workRoot, cacheRoot := paths[0].value, paths[1].value, paths[2].value, paths[3].value
	effectivePath, tokenPath, credentialPath, workspaceRoot := paths[4].value, paths[5].value, paths[6].value, paths[7].value
	accountHomeRoot := ""
	if strings.TrimSpace(snapshot.RuntimeCodexAccountHomeRoot()) != "" {
		var err error
		accountHomeRoot, err = canonicalRuntimePath(snapshot.RuntimeCodexAccountHomeRoot())
		if err != nil {
			return errors.New("bootstrap.account_home_path_invalid")
		}
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
	stateDirectory := filepath.Dir(statePath)
	if overlapsAny(stateDirectory, artifactRoot, workRoot, cacheRoot) ||
		overlapsAny(artifactRoot, workRoot, cacheRoot) ||
		overlapsAny(workRoot, cacheRoot) ||
		overlapsAny(effectivePath, statePath, artifactRoot, workRoot, cacheRoot) ||
		overlapsAny(tokenDirectory, stateDirectory, artifactRoot, workRoot, cacheRoot, effectivePath) ||
		overlapsAny(credentialPath, stateDirectory, artifactRoot, workRoot, cacheRoot, effectivePath, tokenPath) ||
		overlapsAny(workspaceRoot, stateDirectory, artifactRoot, workRoot, cacheRoot, effectivePath, tokenDirectory, credentialPath) {
		return errors.New("bootstrap.runtime_paths_overlap")
	}
	if accountHomeRoot != "" &&
		overlapsAny(accountHomeRoot, stateDirectory, artifactRoot, workRoot, cacheRoot,
			effectivePath, tokenDirectory, credentialPath, workspaceRoot) {
		return errors.New("bootstrap.runtime_paths_overlap")
	}
	if snapshot.IdentityLocalPrincipalsManifestPath() != "" {
		manifestPath, err := canonicalRuntimePath(snapshot.IdentityLocalPrincipalsManifestPath())
		if err != nil {
			return errors.New("bootstrap.local_principals_manifest_path_invalid")
		}
		if manifestPath == tokenPath ||
			overlapsAny(manifestPath, stateDirectory, artifactRoot, workRoot, cacheRoot,
				effectivePath, credentialPath, workspaceRoot, accountHomeRoot) {
			return errors.New("bootstrap.runtime_paths_overlap")
		}
		for _, reservedPath := range credentialReservedPaths {
			if pathsOverlap(manifestPath, reservedPath) {
				return errors.New("bootstrap.runtime_paths_overlap")
			}
		}
	}
	for _, reservedPath := range credentialReservedPaths {
		if overlapsAny(reservedPath, stateDirectory, artifactRoot, workRoot, cacheRoot,
			effectivePath, tokenPath, workspaceRoot, accountHomeRoot) {
			return errors.New("bootstrap.runtime_paths_overlap")
		}
	}
	if strings.TrimSpace(sourceConfigPath) != "" {
		configPath, err := canonicalRuntimePath(sourceConfigPath)
		if err != nil {
			return errors.New("bootstrap.config_path_invalid")
		}
		if overlapsAny(configPath, statePath, artifactRoot, workRoot, cacheRoot,
			effectivePath, tokenDirectory, credentialPath, workspaceRoot, accountHomeRoot) {
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

func overlapsAny(path string, others ...string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	for _, other := range others {
		if strings.TrimSpace(other) == "" {
			continue
		}
		if pathsOverlap(path, other) {
			return true
		}
	}
	return false
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
