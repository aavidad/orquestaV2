//go:build linux

package codex

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const (
	poolTestProfileA = "CodexPoolA"
	poolTestProfileB = "CodexPoolB"
)

func TestPoolUsesDistinctPersistentHomesAndSpillsCapacity(t *testing.T) {
	config, authPaths := poolTestFixture(t)
	pool := openTestPool(t, config)
	descriptores, err := pool.DescribirCapacidadColocaciones()
	if err != nil || len(descriptores) != len(pool.profiles) {
		t.Fatalf("catálogo capacidad=%+v error=%v", descriptores, err)
	}
	for indice, descriptor := range descriptores {
		if descriptor.Plazas != 1 || descriptor.BaseMedicion != application.BaseMedicionCapacidadBruta ||
			descriptor.PlacementRef.String() != "placement:"+pool.profiles[indice].ref ||
			strings.Contains(descriptor.PlacementRef.String(), poolTestProfileA) || strings.Contains(descriptor.PlacementRef.String(), poolTestProfileB) {
			t.Fatalf("descriptor no opaco: %+v", descriptor)
		}
	}

	firstContext, cancelFirst := context.WithCancel(context.Background())
	defer cancelFirst()
	first := testRequest(t, "pool-capacity-a", "helper:fd-audit-block helper:account-home", 1024)
	firstReceipt, err := pool.Launch(firstContext, first)
	if err != nil {
		t.Fatalf("Launch(first) error = %v", err)
	}
	secondContext, cancelSecond := context.WithCancel(context.Background())
	defer cancelSecond()
	second := testRequest(t, "pool-capacity-b", "helper:fd-audit-block helper:account-home", 1024)
	secondReceipt, err := pool.Launch(secondContext, second)
	if err != nil {
		t.Fatalf("Launch(second) error = %v", err)
	}
	firstOwner := poolExecutionOwner(t, pool, first.ExecutionRef)
	secondOwner := poolExecutionOwner(t, pool, second.ExecutionRef)
	if firstOwner == secondOwner || firstOwner.ref == secondOwner.ref ||
		firstOwner.workRoot == secondOwner.workRoot {
		t.Fatalf("capacity did not spill across disjoint profiles: first=%+v second=%+v",
			firstOwner, secondOwner)
	}
	for _, profile := range pool.profiles {
		if profile.adapter.config.MaxConcurrentExecutions != 1 {
			t.Fatalf("profile %q capacity = %d", profile.ref, profile.adapter.config.MaxConcurrentExecutions)
		}
	}

	third := testRequest(t, "pool-capacity-full", "helper:account-home helper:success", 1024)
	if _, err := pool.Launch(context.Background(), third); ErrorCode(err) != CodeCapacityUnavailable {
		t.Fatalf("Launch(full pool) error=%v code=%q", err, ErrorCode(err))
	}

	for profile, expected := range map[string]string{
		poolTestProfileA: `{"tokens":"alpha"}`,
		poolTestProfileB: `{"tokens":"beta"}`,
	} {
		payload, err := os.ReadFile(authPaths[profile])
		if err != nil || string(payload) != expected {
			t.Fatalf("auth %s payload=%q error=%v", profile, payload, err)
		}
	}
	if err := filepath.WalkDir(config.Adapter.WorkRoot, func(filePath string, entry fs.DirEntry, err error) error {
		if err == nil && entry.Name() == accountAuthFileName {
			t.Errorf("auth.json copied into pool work root: %s", filePath)
		}
		return err
	}); err != nil {
		t.Fatalf("WalkDir(pool work root) error = %v", err)
	}

	cancelFirst()
	firstTerminal := awaitPoolTerminal(t, pool, first.ExecutionRef)
	if firstTerminal.Status != ports.AgentFailed || firstTerminal.ErrorCode != CodeExecutionCanceled {
		t.Fatalf("first terminal = %+v", firstTerminal)
	}
	if _, err := pool.Launch(context.Background(), third); err != nil {
		t.Fatalf("Launch(after released slot) error = %v", err)
	}
	if terminal := awaitPoolTerminal(t, pool, third.ExecutionRef); terminal.Status != ports.AgentCompleted {
		t.Fatalf("third terminal = %+v", terminal)
	}

	cancelSecond()
	if terminal := awaitPoolTerminal(t, pool, second.ExecutionRef); terminal.Status != ports.AgentFailed {
		t.Fatalf("second terminal = %+v", terminal)
	}
	if firstReceipt.ExternalRef == "" || secondReceipt.ExternalRef == "" {
		t.Fatalf("launch receipts missing external refs: first=%+v second=%+v", firstReceipt, secondReceipt)
	}
}

func TestPoolHonorsAggregateCapacityAndReplayDoesNotConsumeSlot(t *testing.T) {
	config, _ := poolTestFixture(t)
	config.Adapter.MaxConcurrentExecutions = 1
	pool := openTestPool(t, config)

	firstContext, cancelFirst := context.WithCancel(context.Background())
	defer cancelFirst()
	first := testRequest(t, "pool-aggregate-first", "helper:fd-audit-block helper:account-home", 1024)
	firstReceipt, err := pool.Launch(firstContext, first)
	if err != nil {
		t.Fatalf("Launch(first) error = %v", err)
	}
	secondContext, cancelSecond := context.WithCancel(context.Background())
	defer cancelSecond()
	second := testRequest(t, "pool-aggregate-second", "helper:fd-audit-block helper:account-home", 1024)
	if _, err := pool.Launch(secondContext, second); ErrorCode(err) != CodeCapacityUnavailable ||
		!capacityDefinitelyNotApplied(err) {
		t.Fatalf("Launch(second) error=%v code=%q definitely_not_applied=%v",
			err, ErrorCode(err), capacityDefinitelyNotApplied(err))
	}

	cancelFirst()
	if terminal := awaitPoolTerminal(t, pool, first.ExecutionRef); terminal.Status != ports.AgentFailed {
		t.Fatalf("first terminal = %+v", terminal)
	}
	replayed, err := pool.Launch(context.Background(), first)
	if err != nil || replayed != firstReceipt {
		t.Fatalf("Launch(terminal replay)=%+v error=%v want=%+v", replayed, err, firstReceipt)
	}
	if _, err := pool.Launch(secondContext, second); err != nil {
		t.Fatalf("Launch(after terminal replay) error = %v", err)
	}
	cancelSecond()
	if terminal := awaitPoolTerminal(t, pool, second.ExecutionRef); terminal.Status != ports.AgentFailed {
		t.Fatalf("second terminal = %+v", terminal)
	}
}

func TestPoolConcurrentIdempotentLaunchPublishesOneJournalAndProcess(t *testing.T) {
	config, _ := poolTestFixture(t)
	pool := openTestPool(t, config)
	request := testRequest(t, "pool-idempotent-concurrent", "helper:account-home helper:success", 1024)

	const callers = 12
	receipts := make(chan ports.AgentLaunchReceipt, callers)
	errs := make(chan error, callers)
	var calls sync.WaitGroup
	calls.Add(callers)
	for index := 0; index < callers; index++ {
		go func() {
			defer calls.Done()
			receipt, err := pool.Launch(context.Background(), request)
			receipts <- receipt
			errs <- err
		}()
	}
	calls.Wait()
	close(receipts)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent Launch error = %v", err)
		}
	}
	var first ports.AgentLaunchReceipt
	for receipt := range receipts {
		if first.ExecutionRef.String() == "" {
			first = receipt
		} else if receipt != first {
			t.Fatalf("non-idempotent receipt: first=%+v got=%+v", first, receipt)
		}
	}
	if terminal := awaitPoolTerminal(t, pool, request.ExecutionRef); terminal.Status != ports.AgentCompleted {
		t.Fatalf("terminal = %+v", terminal)
	}
	owner := poolExecutionOwner(t, pool, request.ExecutionRef)
	invocations, err := os.ReadFile(filepath.Join(
		owner.workRoot, filepath.FromSlash(executionPath(request.ExecutionRef)), "helper-invocations",
	))
	if err != nil || string(invocations) != "1" {
		t.Fatalf("helper invocations=%q error=%v", invocations, err)
	}
}

func TestPoolRebuildsExactRoutingFromJournalsAcrossRestart(t *testing.T) {
	config, _ := poolTestFixture(t)
	pool := openTestPool(t, config)
	request := testRequest(t, "pool-restart-routing", "helper:account-home helper:success", 1024)
	receipt, err := pool.Launch(context.Background(), request)
	if err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	if terminal := awaitPoolTerminal(t, pool, request.ExecutionRef); terminal.Status != ports.AgentCompleted {
		t.Fatalf("terminal = %+v", terminal)
	}
	ownerRef := poolExecutionOwner(t, pool, request.ExecutionRef).ref
	if err := pool.Close(); err != nil {
		t.Fatalf("Close(first pool) error = %v", err)
	}

	reopened := openTestPool(t, config)
	if owner := poolExecutionOwner(t, reopened, request.ExecutionRef); owner.ref != ownerRef {
		t.Fatalf("route changed across restart: got=%q want=%q", owner.ref, ownerRef)
	}
	observation, err := reopened.Observe(context.Background(), request.ExecutionRef)
	if err != nil || observation.Status != ports.AgentCompleted {
		t.Fatalf("Observe(restart)=%+v error=%v", observation, err)
	}
	replayed, err := reopened.Launch(context.Background(), request)
	if err != nil || replayed != receipt {
		t.Fatalf("Launch(replay)=%+v error=%v want=%+v", replayed, err, receipt)
	}
	owner := poolExecutionOwner(t, reopened, request.ExecutionRef)
	invocations, err := os.ReadFile(filepath.Join(
		owner.workRoot, filepath.FromSlash(executionPath(request.ExecutionRef)), "helper-invocations",
	))
	if err != nil || string(invocations) != "1" {
		t.Fatalf("restart re-executed process: invocations=%q error=%v", invocations, err)
	}
}

func TestPoolStopRoutesToExactJournalAndShutdownCoversAllProfiles(t *testing.T) {
	config, _ := poolTestFixture(t)
	pool := openTestPool(t, config)
	first := testRequest(t, "pool-stop-first", "helper:fd-audit-block helper:account-home", 1024)
	firstReceipt, err := pool.Launch(context.Background(), first)
	if err != nil {
		t.Fatalf("Launch(first) error = %v", err)
	}
	second := testRequest(t, "pool-stop-second", "helper:fd-audit-block helper:account-home", 1024)
	if _, err := pool.Launch(context.Background(), second); err != nil {
		t.Fatalf("Launch(second) error = %v", err)
	}
	if poolExecutionOwner(t, pool, first.ExecutionRef) == poolExecutionOwner(t, pool, second.ExecutionRef) {
		t.Fatal("blocking executions share one capacity-one profile")
	}

	stopContext, cancelStop := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelStop()
	stopped, err := pool.Stop(
		stopContext,
		stopRequestForLaunch(firstReceipt, ports.AgentStopForced, "stop:pool-first"),
	)
	if err != nil || stopped.Status != ports.AgentStopped {
		t.Fatalf("Stop(first)=%+v error=%v", stopped, err)
	}
	secondObservation, err := pool.Observe(context.Background(), second.ExecutionRef)
	if err != nil {
		t.Fatalf("Observe(second) error = %v", err)
	}
	if secondObservation.Status == ports.AgentCompleted || secondObservation.Status == ports.AgentFailed {
		t.Fatalf("stop crossed profile boundary: second=%+v", secondObservation)
	}

	if err := pool.Close(); err != nil {
		t.Fatalf("Close(pool) error = %v", err)
	}
	reopened := openTestPool(t, config)
	secondTerminal, err := reopened.Observe(context.Background(), second.ExecutionRef)
	if err != nil || secondTerminal.Status != ports.AgentFailed {
		t.Fatalf("shutdown did not settle second execution: observation=%+v error=%v", secondTerminal, err)
	}
}

func TestPoolFansOutCompositionBindingsAndControls(t *testing.T) {
	config, _ := poolTestFixture(t)
	config.Adapter.RuntimeScope = ""
	pool := openTestPool(t, config)
	sessionResolver := &poolSessionResolver{}
	workspaceResolver := &poolWorkspaceResolver{}

	if err := pool.BindSessionResolver(sessionResolver); err != nil {
		t.Fatalf("BindSessionResolver() error = %v", err)
	}
	if err := pool.BindWorkspacePathResolver(workspaceResolver); err != nil {
		t.Fatalf("BindWorkspacePathResolver() error = %v", err)
	}
	if err := pool.BindRuntimeScope(context.Background(), "runtime-scope:pool-test"); err != nil {
		t.Fatalf("BindRuntimeScope() error = %v", err)
	}
	for _, profile := range pool.profiles {
		if profile.adapter.config.SessionResolver != sessionResolver ||
			profile.adapter.workspaceResolver != workspaceResolver ||
			profile.adapter.config.RuntimeScope != "runtime-scope:pool-test" ||
			!profile.adapter.runtimeScopeReady {
			t.Fatalf("bindings missing on profile %q", profile.ref)
		}
	}
	capabilities, err := pool.ControlCapabilities(context.Background())
	if err != nil || !capabilities.CooperativeStop || !capabilities.ForcedStop {
		t.Fatalf("ControlCapabilities()=%+v error=%v", capabilities, err)
	}
	agentCapabilities, err := pool.Capabilities(context.Background())
	if err != nil || agentCapabilities.ProviderRef != ProviderRef ||
		agentCapabilities.ModelRef != DefaultModelRef || agentCapabilities.AgentRef != AgentRef {
		t.Fatalf("Capabilities()=%+v error=%v", agentCapabilities, err)
	}
}

func TestPoolShutdownCancelsBlockedLaunchAuthorityWithoutDeadlock(t *testing.T) {
	config, _ := poolTestFixture(t)
	pool := openTestPool(t, config)
	resolver := &poolBlockingSessionResolver{started: make(chan struct{})}
	if err := pool.BindSessionResolver(resolver); err != nil {
		t.Fatalf("BindSessionResolver() error = %v", err)
	}
	request := testRequest(t, "pool-blocked-resolver", "helper:account-home helper:success", 1024)
	request.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:pool-blocked-resolver")
	launchDone := make(chan error, 1)
	go func() {
		_, err := pool.Launch(context.Background(), request)
		launchDone <- err
	}()
	select {
	case <-resolver.started:
	case <-time.After(time.Second):
		t.Fatal("session resolver did not start")
	}

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelShutdown()
	if err := pool.Shutdown(shutdownContext); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	select {
	case err := <-launchDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("blocked Launch error = %v, want context cancellation", err)
		}
	case <-time.After(time.Second):
		t.Fatal("blocked Launch did not cooperate with Pool shutdown")
	}
}

func TestPoolShutdownTimeoutDoesNotPublishCompletionBeforeAdapterRelease(t *testing.T) {
	config, _ := poolTestFixture(t)
	config.Adapter.SupervisorStartTimeout = 25 * time.Millisecond
	pool, err := NewPool(config)
	if err != nil {
		t.Fatalf("NewPool() error = %v", err)
	}
	_, releaseAdapter, err := pool.profiles[0].adapter.beginOperation(context.Background())
	if err != nil {
		t.Fatalf("beginOperation(adapter) error = %v", err)
	}
	released := false
	defer func() {
		if !released {
			releaseAdapter()
		}
	}()

	timeoutContext, cancelTimeout := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancelTimeout()
	if err := pool.Shutdown(timeoutContext); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown(timeout) error = %v", err)
	}
	time.Sleep(2 * config.Adapter.SupervisorStartTimeout)
	select {
	case <-pool.shutdownDone:
		t.Fatal("Pool published shutdown before child adapter released")
	default:
	}

	releaseAdapter()
	released = true
	if err := pool.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown(after release) error = %v", err)
	}
	reopened, err := NewPool(config)
	if err != nil {
		t.Fatalf("NewPool(after real shutdown) error = %v", err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatalf("Close(reopened) error = %v", err)
	}
}

func TestPoolAllowsAdditionButRejectsMaterializedProfileWithdrawal(t *testing.T) {
	config, _ := poolTestFixture(t)
	firstConfig := config
	firstConfig.AccountHomes = append([]AccountHome(nil), config.AccountHomes[:1]...)
	first, err := NewPool(firstConfig)
	if err != nil {
		t.Fatalf("NewPool(first profile) error = %v", err)
	}
	request := testRequest(t, "pool-safe-addition", "helper:account-home helper:success", 1024)
	if _, err := first.Launch(context.Background(), request); err != nil {
		t.Fatalf("Launch(first profile) error = %v", err)
	}
	if terminal := awaitPoolTerminal(t, first, request.ExecutionRef); terminal.Status != ports.AgentCompleted {
		t.Fatalf("terminal = %+v", terminal)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close(first profile) error = %v", err)
	}

	expanded, err := NewPool(config)
	if err != nil {
		t.Fatalf("NewPool(safe addition) error = %v", err)
	}
	if observation, err := expanded.Observe(context.Background(), request.ExecutionRef); err != nil ||
		observation.Status != ports.AgentCompleted {
		t.Fatalf("Observe(after addition)=%+v error=%v", observation, err)
	}
	if err := expanded.Close(); err != nil {
		t.Fatalf("Close(expanded) error = %v", err)
	}

	withdrawn := config
	withdrawn.AccountHomes = append([]AccountHome(nil), config.AccountHomes[1:]...)
	if pool, err := NewPool(withdrawn); pool != nil || ErrorCode(err) != CodePoolProfileWithdrawal {
		t.Fatalf("NewPool(withdrawn)=%v error=%v code=%q", pool, err, ErrorCode(err))
	}
}

func TestPoolAllowsEmptyProfileRetirementWithoutDeletingWorkRoot(t *testing.T) {
	config, _ := poolTestFixture(t)
	pool, err := NewPool(config)
	if err != nil {
		t.Fatalf("NewPool() error = %v", err)
	}
	retiredRef, err := deriveAccountProfileRef(
		config.AccountHomes[1].Root,
		config.AccountHomes[1].Profile,
	)
	if err != nil {
		t.Fatal(err)
	}
	retiredDirectory, err := poolProfileDirectoryName(retiredRef)
	if err != nil {
		t.Fatal(err)
	}
	retiredWorkRoot := filepath.Join(
		config.Adapter.WorkRoot, poolProfilesDirectory, retiredDirectory,
	)
	if err := pool.Close(); err != nil {
		t.Fatalf("Close(pool) error = %v", err)
	}

	rotated := config
	rotated.AccountHomes = append([]AccountHome(nil), config.AccountHomes[:1]...)
	remaining, err := NewPool(rotated)
	if err != nil {
		t.Fatalf("NewPool(empty retirement) error = %v", err)
	}
	if _, err := os.Stat(retiredWorkRoot); err != nil {
		t.Fatalf("retired empty work root was removed: %v", err)
	}
	if err := remaining.Close(); err != nil {
		t.Fatalf("Close(remaining) error = %v", err)
	}

	readded, err := NewPool(config)
	if err != nil {
		t.Fatalf("NewPool(re-add empty profile) error = %v", err)
	}
	if err := readded.Close(); err != nil {
		t.Fatalf("Close(readded) error = %v", err)
	}
}

func TestPoolFailsClosedForDuplicateOrCorruptConfigurationAndJournals(t *testing.T) {
	t.Run("duplicate account", func(t *testing.T) {
		config, _ := poolTestFixture(t)
		config.AccountHomes = append(config.AccountHomes, config.AccountHomes[0])
		if pool, err := NewPool(config); pool != nil || ErrorCode(err) != CodePoolConfigInvalid {
			t.Fatalf("NewPool(duplicate)=%v error=%v code=%q", pool, err, ErrorCode(err))
		}
	})

	t.Run("corrupt auth", func(t *testing.T) {
		config, authPaths := poolTestFixture(t)
		mustWritePrivate(t, authPaths[poolTestProfileA], []byte(`{"broken"`))
		if pool, err := NewPool(config); pool != nil || ErrorCode(err) != CodeAccountProfileUnavailable {
			t.Fatalf("NewPool(corrupt auth)=%v error=%v code=%q", pool, err, ErrorCode(err))
		}
	})

	t.Run("corrupt request journal", func(t *testing.T) {
		config, _ := poolTestFixture(t)
		pool := openTestPool(t, config)
		request := testRequest(t, "pool-corrupt-journal", "helper:account-home helper:success", 1024)
		if _, err := pool.Launch(context.Background(), request); err != nil {
			t.Fatalf("Launch() error = %v", err)
		}
		if terminal := awaitPoolTerminal(t, pool, request.ExecutionRef); terminal.Status != ports.AgentCompleted {
			t.Fatalf("terminal = %+v", terminal)
		}
		owner := poolExecutionOwner(t, pool, request.ExecutionRef)
		requestPath := filepath.Join(
			owner.workRoot, filepath.FromSlash(executionPath(request.ExecutionRef)), requestFileName,
		)
		if err := pool.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		mustWritePrivate(t, requestPath, []byte(`{"broken"`))
		if reopened, err := NewPool(config); reopened != nil || ErrorCode(err) != CodeStateInvalid {
			t.Fatalf("NewPool(corrupt journal)=%v error=%v code=%q", reopened, err, ErrorCode(err))
		}
	})

	t.Run("duplicate request journal", func(t *testing.T) {
		config, _ := poolTestFixture(t)
		pool := openTestPool(t, config)
		request := testRequest(t, "pool-duplicate-journal", "helper:account-home helper:success", 1024)
		if _, err := pool.Launch(context.Background(), request); err != nil {
			t.Fatalf("Launch() error = %v", err)
		}
		if terminal := awaitPoolTerminal(t, pool, request.ExecutionRef); terminal.Status != ports.AgentCompleted {
			t.Fatalf("terminal = %+v", terminal)
		}
		owner := poolExecutionOwner(t, pool, request.ExecutionRef)
		var other *poolProfile
		for _, profile := range pool.profiles {
			if profile != owner {
				other = profile
				break
			}
		}
		requestHash, err := other.adapter.hashLaunchRequest(request)
		if err != nil {
			t.Fatalf("hashLaunchRequest(other) error = %v", err)
		}
		if _, _, created, err := other.adapter.ensureLaunchRecord(request, requestHash); err != nil || !created {
			t.Fatalf("ensureLaunchRecord(other) created=%v error=%v", created, err)
		}
		if err := pool.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		if reopened, err := NewPool(config); reopened != nil || ErrorCode(err) != CodePoolRoutingInvalid {
			t.Fatalf("NewPool(duplicate journal)=%v error=%v code=%q", reopened, err, ErrorCode(err))
		}
	})

	t.Run("execution directory without request journal", func(t *testing.T) {
		config, _ := poolTestFixture(t)
		pool := openTestPool(t, config)
		workRoot := pool.profiles[0].workRoot
		if err := pool.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		executionRef, _ := goal.NewExecutionRef("execution:pool-missing-request")
		runPath := filepath.Join(workRoot, filepath.FromSlash(executionPath(executionRef)))
		if err := os.MkdirAll(runPath, 0o700); err != nil {
			t.Fatalf("MkdirAll(incomplete execution) error = %v", err)
		}
		if reopened, err := NewPool(config); reopened != nil || ErrorCode(err) != CodePoolRoutingInvalid {
			t.Fatalf("NewPool(missing request)=%v error=%v code=%q", reopened, err, ErrorCode(err))
		}
	})

	t.Run("partial construction releases earlier profile lock", func(t *testing.T) {
		config, authPaths := poolTestFixture(t)
		first := config.AccountHomes[0]
		second := config.AccountHomes[1]
		firstRef, err := deriveAccountProfileRef(first.Root, first.Profile)
		if err != nil {
			t.Fatal(err)
		}
		secondRef, err := deriveAccountProfileRef(second.Root, second.Profile)
		if err != nil {
			t.Fatal(err)
		}
		corruptProfile := second.Profile
		if secondRef < firstRef {
			corruptProfile = first.Profile
		}
		mustWritePrivate(t, authPaths[corruptProfile], []byte(`{"broken"`))
		if pool, err := NewPool(config); pool != nil || ErrorCode(err) != CodeAccountProfileUnavailable {
			t.Fatalf("NewPool(partial)=%v error=%v code=%q", pool, err, ErrorCode(err))
		}
		payload := []byte(`{"tokens":"alpha"}`)
		if corruptProfile == poolTestProfileB {
			payload = []byte(`{"tokens":"beta"}`)
		}
		mustWritePrivate(t, authPaths[corruptProfile], payload)
		recovered, err := NewPool(config)
		if err != nil {
			t.Fatalf("NewPool(after partial cleanup) error = %v", err)
		}
		if err := recovered.Close(); err != nil {
			t.Fatalf("Close(recovered) error = %v", err)
		}
	})
}

type poolSessionResolver struct{}

func (*poolSessionResolver) ResolveCodexSession(
	context.Context,
	ports.AgentLaunchRequest,
) (Session, error) {
	return Session{}, errors.New("unused")
}

func (*poolSessionResolver) RecoverCodexSession(
	context.Context,
	ports.AgentLaunchRequest,
) (Session, error) {
	return Session{}, errors.New("unused")
}

type poolBlockingSessionResolver struct {
	started chan struct{}
	once    sync.Once
}

func (resolver *poolBlockingSessionResolver) ResolveCodexSession(
	ctx context.Context,
	_ ports.AgentLaunchRequest,
) (Session, error) {
	resolver.once.Do(func() { close(resolver.started) })
	<-ctx.Done()
	return Session{}, ctx.Err()
}

func (resolver *poolBlockingSessionResolver) RecoverCodexSession(
	ctx context.Context,
	request ports.AgentLaunchRequest,
) (Session, error) {
	return resolver.ResolveCodexSession(ctx, request)
}

type poolWorkspaceResolver struct{}

func (*poolWorkspaceResolver) ResolveExecutionWorkspace(
	context.Context,
	ports.ExecutionWorkspaceRef,
) (string, error) {
	return "", errors.New("unused")
}

func poolTestFixture(t *testing.T) (PoolConfig, map[string]string) {
	t.Helper()
	root, profileAPath := secureAccountFixture(t, poolTestProfileA)
	profileBPath := filepath.Join(root, poolTestProfileB)
	if err := os.Mkdir(profileBPath, 0o700); err != nil {
		t.Fatalf("Mkdir(profile B) error = %v", err)
	}
	authPaths := map[string]string{
		poolTestProfileA: filepath.Join(profileAPath, accountAuthFileName),
		poolTestProfileB: filepath.Join(profileBPath, accountAuthFileName),
	}
	mustWritePrivate(t, authPaths[poolTestProfileA], []byte(`{"tokens":"alpha"}`))
	mustWritePrivate(t, authPaths[poolTestProfileB], []byte(`{"tokens":"beta"}`))
	base := testConfig(t)
	base.AccountAuthMaxDocumentBytes = accountTestAuthMaximumBytes
	return PoolConfig{
		Adapter: base,
		AccountHomes: []AccountHome{
			{Root: root, Profile: poolTestProfileA},
			{Root: root, Profile: poolTestProfileB},
		},
	}, authPaths
}

func openTestPool(t *testing.T, config PoolConfig) *Pool {
	t.Helper()
	pool, err := NewPool(config)
	if err != nil {
		t.Fatalf("NewPool() error = %v", err)
	}
	t.Cleanup(func() {
		if err := pool.Close(); err != nil {
			t.Errorf("Close(pool) error = %v", err)
		}
	})
	return pool
}

func awaitPoolTerminal(
	t *testing.T,
	pool *Pool,
	executionRef goal.ExecutionRef,
) ports.AgentObservation {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		observation, err := pool.Observe(context.Background(), executionRef)
		if err != nil {
			t.Fatalf("Observe() error = %v", err)
		}
		if observation.Status == ports.AgentCompleted || observation.Status == ports.AgentFailed {
			return observation
		}
		if time.Now().After(deadline) {
			t.Fatalf("execution did not become terminal: %+v", observation)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func poolExecutionOwner(t *testing.T, pool *Pool, executionRef goal.ExecutionRef) *poolProfile {
	t.Helper()
	pool.mu.Lock()
	defer pool.mu.Unlock()
	owner, found, err := pool.scanExecutionRouteLocked(executionRef)
	if err != nil || !found {
		t.Fatalf("execution owner found=%v error=%v", found, err)
	}
	return owner
}
