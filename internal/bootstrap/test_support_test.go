package bootstrap

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/config"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

type bearerRoundTripper struct {
	token string
	base  http.RoundTripper
}

func (transport bearerRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	cloned := request.Clone(request.Context())
	cloned.Header = request.Header.Clone()
	cloned.Header.Set("Authorization", "Bearer "+transport.token)
	base := transport.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(cloned)
}

func authorizedHTTPClient(t *testing.T, tokenPath string) *http.Client {
	t.Helper()
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("read local auth token: %v", err)
	}
	return &http.Client{Transport: bearerRoundTripper{token: string(token)}}
}

func writeTestConfig(t *testing.T, root string) string {
	t.Helper()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatalf("make config directory private: %v", err)
	}
	configPath := root + "/orquesta.toml"
	content := fmt.Sprintf(`governance.global_process_slots_budget = 4
[server]
listen = "127.0.0.1:0"
shutdown_timeout = "2s"

[state.sqlite]
path = %s
busy_timeout = "1s"
max_open_connections = 4

[artifact.filesystem]
root = %s

[credentials.local]
path = %s
max_document_bytes = 1048576

[runtime]
max_output_bytes = 65536

[runtime.codex]
timeout = "1s"
supervisor_start_timeout = "500ms"
max_concurrent_executions = 4
work_root = %s
cache_root = %s

[identity]
local_token_path = %s

[scheduler]
poll_interval = "10ms"
observation_interval = "10ms"
claim_lease = "1s"
max_execution_attempts = 3
execution_timeout = "10s"

[config]
effective_path = %s
	`, strconv.Quote(root+"/state/orquesta.sqlite"), strconv.Quote(root+"/artifacts"),
		strconv.Quote(root+"/secrets/credentials.json"), strconv.Quote(root+"/work"),
		strconv.Quote(root+"/cache/codex-go"),
		strconv.Quote(root+"/secrets/local-owner.token"),
		strconv.Quote(root+"/effective_config.json"))
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}

func configurarPerfilCuotaCodexPrueba(t *testing.T, raiz, configuracion, comando string, cantidad int) {
	t.Helper()
	if cantidad < 1 {
		t.Fatal("cantidad de perfiles inválida")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	base, err := os.MkdirTemp(home, ".orquesta-cuota-codex-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(base) })
	raizCuentas := filepath.Join(base, "cuentas-codex")
	perfiles := make([]string, cantidad)
	for indice := range perfiles {
		perfiles[indice] = fmt.Sprintf("CodexTest%d", indice+1)
		perfil := filepath.Join(raizCuentas, perfiles[indice])
		if err := os.MkdirAll(perfil, 0o700); err != nil {
			t.Fatal(err)
		}
		for _, ruta := range []string{raizCuentas, perfil} {
			if err := os.Chmod(ruta, 0o700); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(perfil, "auth.json"), []byte(`{"tokens":"test"}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	puente := filepath.Join(raiz, "codex-con-cuota.sh")
	ejecutable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	guion := "#!/bin/sh\n" +
		"if [ \"${1-}\" = app-server ]; then exec " + strconv.Quote(ejecutable) + " -test.run=^TestAyudanteCuotaCodexBootstrap$ -- ayudante-cuota-codex-bootstrap; fi\n" +
		"exec " + strconv.Quote(comando) + " \"$@\"\n"
	if err := os.WriteFile(puente, []byte(guion), 0o700); err != nil {
		t.Fatal(err)
	}
	perfilesTOML := make([]string, len(perfiles))
	for indice, perfil := range perfiles {
		perfilesTOML[indice] = strconv.Quote(perfil)
	}
	replaceTestConfigValue(t, configuracion, "[runtime.codex]\n", "[runtime.capacity]\nobservation_timeout = \"5s\"\n\n[runtime.codex]\naccount_home_root = "+strconv.Quote(raizCuentas)+"\naccount_profiles = ["+strings.Join(perfilesTOML, ", ")+"]\n")
	replaceTestConfigValue(t, configuracion, "max_concurrent_executions = 4", "max_concurrent_executions = "+strconv.Itoa(cantidad))
	replaceTestConfigValue(t, configuracion, "timeout = \"5s\"", "timeout = \"10s\"")
	replaceTestConfigValue(t, configuracion, "command = "+strconv.Quote(comando), "command = "+strconv.Quote(puente))
}

func TestAyudanteCuotaCodexBootstrap(t *testing.T) {
	if len(os.Args) == 0 || os.Args[len(os.Args)-1] != "ayudante-cuota-codex-bootstrap" {
		return
	}
	lector := bufio.NewScanner(os.Stdin)
	if !lector.Scan() {
		os.Exit(2)
	}
	fmt.Println(`{"id":"init","result":{"userAgent":"test","codexHome":"/test","platformFamily":"unix","platformOs":"linux"}}`)
	if !lector.Scan() || !lector.Scan() {
		os.Exit(2)
	}
	fmt.Println(`{"id":1,"result":{"rateLimits":{"credits":null,"individualLimit":null,"limitId":"test","limitName":"test","planType":"plus","primary":{"usedPercent":25,"windowDurationMins":300,"resetsAt":4102444800},"rateLimitReachedType":null,"secondary":null,"spendControlReached":false},"rateLimitResetCredits":null,"rateLimitsByLimitId":null}}`)
	for lector.Scan() {
	}
	os.Exit(0)
}

func colocacionAgentePrueba(t *testing.T, agente AgentAdapter) ports.AgentPlacementRef {
	t.Helper()
	descriptores, err := agente.(catalogoCapacidadColocacionAgente).DescribirCapacidadColocaciones()
	if err != nil || len(descriptores) != 1 {
		t.Fatalf("colocaciones=%+v error=%v", descriptores, err)
	}
	return descriptores[0].PlacementRef
}

type countingAgent struct {
	capacidadAgentePrueba
	now      func() time.Time
	launches *atomic.Int64
	content  []byte
	mu       sync.Mutex
	requests map[goal.ExecutionRef]ports.AgentLaunchRequest
	closed   bool
}

func newCountingAgent(clock application.Clock, launches *atomic.Int64) *countingAgent {
	return &countingAgent{now: clock.Now, launches: launches, requests: make(map[goal.ExecutionRef]ports.AgentLaunchRequest)}
}

func (agent *countingAgent) Capabilities(context.Context) (ports.AgentCapabilities, error) {
	return ports.AgentCapabilities{ProviderRef: "provider:test", ModelRef: "model:test", AgentRef: "agent:test", Unrestricted: true}, nil
}

func (agent *countingAgent) Launch(ctx context.Context, request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
	if err := ctx.Err(); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	if err := ports.ValidateAgentLaunchRequest(request); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	agent.mu.Lock()
	defer agent.mu.Unlock()
	if agent.closed {
		return ports.AgentLaunchReceipt{}, errors.New("test_agent.closed")
	}
	if existing, ok := agent.requests[request.ExecutionRef]; ok {
		if !reflect.DeepEqual(existing, request) {
			return ports.AgentLaunchReceipt{}, errors.New("test_agent.conflict")
		}
	} else {
		agent.requests[request.ExecutionRef] = request
		agent.launches.Add(1)
	}
	return ports.AgentLaunchReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: request.SpecHash,
		ProviderRef: "provider:test", ModelRef: "model:test", AgentRef: "agent:test",
		ExternalRef:    "test:" + request.ExecutionRef.String(),
		IdempotencyKey: request.IdempotencyKey, AcceptedAt: agent.now(),
		ReceiptRef: "test-launch:" + request.ExecutionRef.String(),
	}, nil
}

func (agent *countingAgent) Observe(ctx context.Context, executionRef goal.ExecutionRef) (ports.AgentObservation, error) {
	if err := ctx.Err(); err != nil {
		return ports.AgentObservation{}, err
	}
	agent.mu.Lock()
	request, ok := agent.requests[executionRef]
	closed := agent.closed
	agent.mu.Unlock()
	if closed || !ok {
		return ports.AgentObservation{}, errors.New("test_agent.not_found")
	}
	content := agent.content
	if len(content) == 0 {
		content = []byte("artifact:" + request.Objective)
	}
	return ports.AgentObservation{
		ExecutionRef: executionRef, SpecHash: request.SpecHash, Status: ports.AgentCompleted,
		MediaType: "text/plain", Content: append([]byte(nil), content...),
		Usage: unknownTestUsage(), ObservedAt: agent.now(),
	}, nil
}

func (agent *countingAgent) Shutdown(context.Context) error {
	agent.mu.Lock()
	agent.closed = true
	agent.mu.Unlock()
	return nil
}

func countingFactory(launches *atomic.Int64) AgentFactory {
	return func(_ config.Snapshot, clock application.Clock) (AgentAdapter, error) {
		return newCountingAgent(clock, launches), nil
	}
}

func constantContentFactory(launches *atomic.Int64, content []byte) AgentFactory {
	return func(_ config.Snapshot, clock application.Clock) (AgentAdapter, error) {
		agent := newCountingAgent(clock, launches)
		agent.content = append([]byte(nil), content...)
		return agent, nil
	}
}

func submitTestGoal(t *testing.T, runtime *Runtime, requestRef string) goal.GoalRef {
	t.Helper()
	access := testRuntimeAccess(t, runtime)
	result, err := runtime.Orchestrator().Submit(context.Background(), access, application.SubmitRequest{
		RequestRef: requestRef, Statement: "produce restart evidence",
		Confirm: true,
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	return result.Record.Goal.Ref()
}

func waitTerminalGoal(t *testing.T, runtime *Runtime, ref goal.GoalRef) application.GoalRecord {
	t.Helper()
	access := testRuntimeAccess(t, runtime)
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		record, err := runtime.Orchestrator().GetGoal(context.Background(), access, ref)
		if err == nil && record.Goal.IsTerminal() {
			return record
		}
		time.Sleep(10 * time.Millisecond)
	}
	record, err := runtime.Orchestrator().GetGoal(context.Background(), access, ref)
	t.Fatalf("goal %s did not become terminal: state=%s executions=%+v attempts=%d receipts=%d error=%v", ref.String(), record.Goal.State(), record.Executions, len(record.EffectAttempts), len(record.EffectReceipts), err)
	return application.GoalRecord{}
}

func testRuntimeAccess(t *testing.T, runtime *Runtime) application.Access {
	t.Helper()
	principal, hierarchy, err := localIdentityComposition(runtime.config)
	if err != nil {
		t.Fatalf("compose runtime identity: %v", err)
	}
	access, err := application.NewAccess(principal, hierarchy.ProjectRef())
	if err != nil {
		t.Fatalf("compose runtime access: %v", err)
	}
	return access
}

const sleeperMarker = "bootstrap-helper-sleep"

func TestBootstrapSleeper(t *testing.T) {
	marked := false
	for _, argument := range os.Args {
		if argument == sleeperMarker {
			marked = true
			break
		}
	}
	if !marked {
		return
	}
	time.Sleep(30 * time.Second)
}

type processAgent struct {
	capacidadAgentePrueba
	ctx       context.Context
	cancel    context.CancelFunc
	now       func() time.Time
	started   chan int
	mu        sync.Mutex
	execution goal.ExecutionRef
	request   ports.AgentLaunchRequest
	command   *exec.Cmd
	done      chan struct{}
	wait      sync.WaitGroup
	closed    bool
}

func newProcessAgent(clock application.Clock) *processAgent {
	ctx, cancel := context.WithCancel(context.Background())
	return &processAgent{ctx: ctx, cancel: cancel, now: clock.Now, started: make(chan int, 1)}
}

func (agent *processAgent) Capabilities(context.Context) (ports.AgentCapabilities, error) {
	return ports.AgentCapabilities{ProviderRef: "provider:process-test", ModelRef: "model:process-test", AgentRef: "agent:process-test", Unrestricted: true}, nil
}

func (agent *processAgent) Launch(_ context.Context, request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
	agent.mu.Lock()
	defer agent.mu.Unlock()
	if agent.closed {
		return ports.AgentLaunchReceipt{}, errors.New("process_agent.closed")
	}
	if agent.command != nil {
		if agent.execution != request.ExecutionRef || agent.request.IdempotencyKey != request.IdempotencyKey {
			return ports.AgentLaunchReceipt{}, errors.New("process_agent.conflict")
		}
		return agent.receipt(request), nil
	}
	command := exec.CommandContext(agent.ctx, os.Args[0], "-test.run=^TestBootstrapSleeper$", "--", sleeperMarker)
	if err := command.Start(); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	agent.command = command
	agent.execution = request.ExecutionRef
	agent.request = request
	agent.done = make(chan struct{})
	agent.wait.Add(1)
	agent.started <- command.Process.Pid
	go func() {
		defer agent.wait.Done()
		_ = command.Wait()
		close(agent.done)
	}()
	return agent.receipt(request), nil
}

func (agent *processAgent) receipt(request ports.AgentLaunchRequest) ports.AgentLaunchReceipt {
	return ports.AgentLaunchReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: request.SpecHash,
		ProviderRef: "provider:process-test", ModelRef: "model:process-test", AgentRef: "agent:process-test",
		ExternalRef: "pid-owned", IdempotencyKey: request.IdempotencyKey, AcceptedAt: agent.now(),
		ReceiptRef: "process-launch:" + request.ExecutionRef.String(),
	}
}

func (agent *processAgent) Observe(context.Context, goal.ExecutionRef) (ports.AgentObservation, error) {
	return ports.AgentObservation{
		ExecutionRef: agent.execution, SpecHash: agent.request.SpecHash, Status: ports.AgentRunning,
		Usage: unknownTestUsage(), ObservedAt: agent.now(),
	}, nil
}

func testBudgetDemand(ref string) governance.BudgetDemand {
	return governance.BudgetDemand{Ref: "budget-demand:" + ref}
}

func unknownTestUsage() governance.ResourceUsage {
	return governance.ResourceUsage{Quality: governance.UsageQualityUnknown}
}

type capacidadAgentePrueba struct{}

func (capacidadAgentePrueba) DescribirCapacidadColocaciones() ([]application.DescriptorCapacidadColocacionAgente, error) {
	colocacion, _ := ports.NewAgentPlacementRef("placement:test")
	return []application.DescriptorCapacidadColocacionAgente{{PlacementRef: colocacion, SourceRef: "source:test", PoolRef: "pool:test", BaseMedicion: application.BaseMedicionCapacidadBruta, Plazas: 1_000}}, nil
}

func (capacidadAgentePrueba) IniciarControladoresCuota(ctx context.Context, configuracion application.ConfiguracionControladoresCuotaAgente) ([]application.ControladorCuotaAgente, error) {
	ahora := configuracion.Ahora()
	colocacion, _ := ports.NewAgentPlacementRef("placement:test")
	err := configuracion.Sumidero(ctx, application.AgentQuotaObservation{PlacementRef: colocacion, WindowRef: "window:test", Status: application.AgentQuotaAvailable, Quality: governance.UsageQualityExact, ObservedAt: ahora, ExpiresAt: ahora.Add(configuracion.VigenciaObservacion)}, nil)
	return nil, err
}

func (agent *processAgent) Shutdown(ctx context.Context) error {
	agent.mu.Lock()
	if !agent.closed {
		agent.closed = true
		agent.cancel()
	}
	agent.mu.Unlock()
	finished := make(chan struct{})
	go func() {
		agent.wait.Wait()
		close(finished)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-finished:
		return nil
	}
}
