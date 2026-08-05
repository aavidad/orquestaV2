package bootstrap

import (
	"context"
	"crypto/ed25519"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/adapters/agent/codex"
	credentiallocal "orquesta/internal/adapters/credentials/local"
	localruntime "orquesta/internal/adapters/system/local"
	"orquesta/internal/application"
	"orquesta/internal/config"
	"orquesta/internal/credentials"
)

type clienteRuntimeIsolationMicroVM struct {
	clienteFactoriaAgentMicroVM
	antesCapacidades func()
	falloCapacidades error
	consultas        atomic.Int64
}

type agenteRuntimeIsolationCatalogoTrasBind struct {
	*countingAgent
	workspaceVinculado atomic.Bool
	scopeVinculado     atomic.Bool
	consultasCatalogo  atomic.Int64
}

func (agente *agenteRuntimeIsolationCatalogoTrasBind) BindWorkspacePathResolver(codex.WorkspacePathResolver) error {
	agente.workspaceVinculado.Store(true)
	return nil
}

func (agente *agenteRuntimeIsolationCatalogoTrasBind) BindRuntimeScope(ctx context.Context, scope string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if scope == "" {
		return errors.New("runtime_isolation.scope_required")
	}
	agente.scopeVinculado.Store(true)
	return nil
}

func (agente *agenteRuntimeIsolationCatalogoTrasBind) DescribirCapacidadColocaciones() ([]application.DescriptorCapacidadColocacionAgente, error) {
	agente.consultasCatalogo.Add(1)
	if !agente.workspaceVinculado.Load() || !agente.scopeVinculado.Load() {
		return nil, errors.New("runtime_isolation.capacity_before_bindings")
	}
	return capacidadAgentePrueba{}.DescribirCapacidadColocaciones()
}

func (cliente *clienteRuntimeIsolationMicroVM) Capacidades(ctx context.Context) (microvm.RespuestaCapacidades, error) {
	cliente.consultas.Add(1)
	if cliente.antesCapacidades != nil {
		cliente.antesCapacidades()
	}
	if cliente.falloCapacidades != nil {
		return microvm.RespuestaCapacidades{}, cliente.falloCapacidades
	}
	return cliente.clienteFactoriaAgentMicroVM.Capacidades(ctx)
}

func TestBuildRejectsPartialMicroVMConfigBeforeCompositionState(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	replaceTestConfigValue(t, configPath, "[runtime]\n", "[runtime]\nisolation = \"microvm\"\n")

	runtime, err := Build(context.Background(), Options{ConfigPath: configPath})
	if runtime != nil || !config.HasErrorCode(err, config.ErrorCrossValidation) {
		t.Fatalf("partial microvm config: runtime=%v err=%v", runtime, err)
	}
	assertNoCompositionState(t, root)
}

func TestBuildComposesProductionMicroVMAfterCapabilitiesWithoutHostBinders(t *testing.T) {
	root := t.TempDir()
	configPath, socketPath := writeRuntimeIsolationMicroVMConfig(t, root, true)
	provisionRuntimeIsolationMicroVMCredential(t, root)

	client := &clienteRuntimeIsolationMicroVM{}
	var constructorCalls atomic.Int64
	var connectionCloses atomic.Int64
	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath,
		constructorClienteAgentMicroVM: func(gotSocket string) (recursoClienteAgentMicroVM, error) {
			constructorCalls.Add(1)
			if gotSocket != socketPath {
				t.Fatalf("microvm socket=%q want=%q", gotSocket, socketPath)
			}
			return recursoClienteAgentMicroVM{
				cliente: client,
				liberarConexiones: func() error {
					connectionCloses.Add(1)
					return nil
				},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("Build(microvm) error=%v", err)
	}
	if runtime == nil {
		t.Fatal("Build(microvm) returned nil runtime")
	}
	if _, ok := runtime.agent.(*agenteMicroVM); !ok {
		t.Fatalf("production agent type=%T want=*bootstrap.agenteMicroVM", runtime.agent)
	}
	if _, ok := runtime.agent.(application.AgentController); ok {
		t.Fatalf("microvm agent invented AgentController: %T", runtime.agent)
	}
	if runtime.workspace == nil {
		t.Fatal("configured repository did not compose workspace")
	}
	descriptors, err := runtime.agent.(catalogoCapacidadColocacionAgente).DescribirCapacidadColocaciones()
	if err != nil || len(descriptors) != 1 || descriptors[0].Plazas != 16 ||
		descriptors[0].PlacementRef.String() != "placement:codex:microvm-runtime" {
		t.Fatalf("physical microvm capacity=%+v err=%v", descriptors, err)
	}
	if constructorCalls.Load() != 1 || client.consultas.Load() != 1 || connectionCloses.Load() != 0 {
		t.Fatalf("constructor=%d capabilities=%d cleanup=%d before shutdown",
			constructorCalls.Load(), client.consultas.Load(), connectionCloses.Load())
	}
	if err := runtime.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown(microvm) error=%v", err)
	}
	if connectionCloses.Load() != 1 {
		t.Fatalf("connection cleanup=%d want=1", connectionCloses.Load())
	}
}

func TestBuildMicroVMCapabilitiesFailureCleansClientBeforeDurableState(t *testing.T) {
	root := t.TempDir()
	configPath, socketPath := writeRuntimeIsolationMicroVMConfig(t, root, false)
	provisionRuntimeIsolationMicroVMCredential(t, root)
	want := errors.New("runtime_isolation.capabilities_failed")
	statePath := filepath.Join(root, "state", "orquesta.sqlite")
	client := &clienteRuntimeIsolationMicroVM{
		falloCapacidades: want,
		antesCapacidades: func() {
			if _, err := os.Lstat(statePath); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("Capabilities observed SQLite state: %v", err)
			}
		},
	}
	var connectionCloses atomic.Int64

	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath,
		constructorClienteAgentMicroVM: func(gotSocket string) (recursoClienteAgentMicroVM, error) {
			if gotSocket != socketPath {
				t.Fatalf("microvm socket=%q want=%q", gotSocket, socketPath)
			}
			return recursoClienteAgentMicroVM{
				cliente:           client,
				liberarConexiones: func() error { connectionCloses.Add(1); return nil },
			}, nil
		},
	})
	if runtime != nil || !errors.Is(err, want) {
		t.Fatalf("capabilities failure: runtime=%v err=%v", runtime, err)
	}
	if client.consultas.Load() != 1 || connectionCloses.Load() != 1 {
		t.Fatalf("capabilities=%d cleanup=%d want=1/1", client.consultas.Load(), connectionCloses.Load())
	}
	for _, path := range []string{
		statePath,
		filepath.Join(root, "artifacts"),
		filepath.Join(root, "effective_config.json"),
		filepath.Join(root, "work"),
	} {
		if _, statErr := os.Lstat(path); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("failed Capabilities created %s: %v", path, statErr)
		}
	}
}

func TestOpenBuildAgentProcessDoesNotCallMicroVMConstructor(t *testing.T) {
	configPath := writeTestConfig(t, t.TempDir())
	snapshot, err := loadConfigSnapshot(context.Background(), configPath)
	if err != nil {
		t.Fatal(err)
	}
	var constructorCalls atomic.Int64
	agent, _, _, err := openBuildAgent(
		context.Background(),
		snapshot,
		localruntime.Clock{},
		nil,
		rendererFactoriaAgentMicroVM{},
		nil,
		codexGoToolchainOwnerTrusted,
		func(string) (recursoClienteAgentMicroVM, error) {
			constructorCalls.Add(1)
			return recursoClienteAgentMicroVM{}, errors.New("microvm constructor must not run")
		},
	)
	if err != nil {
		t.Fatalf("openBuildAgent(process) error=%v", err)
	}
	if constructorCalls.Load() != 0 {
		t.Fatalf("process called microvm constructor %d times", constructorCalls.Load())
	}
	if _, ok := agent.(*agenteMicroVM); ok {
		t.Fatalf("process selected microvm agent: %T", agent)
	}
	shutdownBuildAgent(agent, snapshot.ServerShutdownTimeout())
}

func TestBuildCustomAgentComposesCapacityAfterHistoricalBindings(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	seedPath := filepath.Join(root, "repository")
	if err := os.Mkdir(seedPath, 0o700); err != nil {
		t.Fatal(err)
	}
	replaceTestConfigValue(t, configPath, "[identity]\n", `[workspace.local]
root = `+strconv.Quote(filepath.Join(root, "workspaces"))+`

[repository.local]
seed_path = `+strconv.Quote(seedPath)+`
target_ref = "refs/heads/main"

[identity]
`)
	var agent *agenteRuntimeIsolationCatalogoTrasBind
	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath,
		AgentFactory: func(_ config.Snapshot, clock application.Clock) (AgentAdapter, error) {
			agent = &agenteRuntimeIsolationCatalogoTrasBind{countingAgent: newCountingAgent(clock, &atomic.Int64{})}
			return agent, nil
		},
	})
	if err != nil {
		t.Fatalf("Build(custom agent) error=%v", err)
	}
	if agent == nil {
		t.Fatal("custom factory did not retain its agent")
	}
	if !agent.workspaceVinculado.Load() || !agent.scopeVinculado.Load() || agent.consultasCatalogo.Load() != 1 {
		t.Fatalf("custom bindings: workspace=%t scope=%t catalog=%d",
			agent.workspaceVinculado.Load(), agent.scopeVinculado.Load(), agent.consultasCatalogo.Load())
	}
	if err := runtime.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown(custom agent) error=%v", err)
	}
}

func writeRuntimeIsolationMicroVMConfig(t *testing.T, root string, withRepository bool) (string, string) {
	t.Helper()
	_, descriptorRaw := descriptorFactoriaAgentMicroVM(t)
	descriptorPath := filepath.Join(root, "perfil-microvm.json")
	if err := os.WriteFile(descriptorPath, descriptorRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	socketPath := filepath.Join(root, "agente-microvm.sock")
	configPath := writeTestConfig(t, root)
	replaceTestConfigValue(t, configPath, "[runtime]\n", "[runtime]\nprovider = \"codex\"\nisolation = \"microvm\"\n")
	replaceTestConfigValue(t, configPath, "[runtime.codex]\n", `[runtime.microvm]
placement_ref = "placement:codex:microvm-runtime"
socket_path = `+strconv.Quote(socketPath)+`
profile_descriptor_path = `+strconv.Quote(descriptorPath)+`
expected_profile_descriptor_sha256 = "`+digestRawFactoriaAgentMicroVM(descriptorRaw)+`"
launch_grant_key_id = "clave-publica:runtime-isolation"
launch_grant_signing_credential_ref = "credential:microvm-launch-signing"

[runtime.codex]
model = "gpt-5.6"
`)
	if withRepository {
		seedPath := filepath.Join(root, "repository")
		if err := os.Mkdir(seedPath, 0o700); err != nil {
			t.Fatal(err)
		}
		replaceTestConfigValue(t, configPath, "[identity]\n", `[workspace.local]
root = `+strconv.Quote(filepath.Join(root, "workspaces"))+`

[repository.local]
seed_path = `+strconv.Quote(seedPath)+`
target_ref = "refs/heads/main"

[identity]
`)
	}
	return configPath, socketPath
}

func provisionRuntimeIsolationMicroVMCredential(t *testing.T, root string) {
	t.Helper()
	if err := os.Mkdir(filepath.Join(root, "secrets"), 0o700); err != nil {
		t.Fatal(err)
	}
	store, err := credentiallocal.Open(credentiallocal.Options{
		Path: filepath.Join(root, "secrets", "credentials.json"), OwnerUID: os.Geteuid(),
		MaxStoreBytes: 1 << 20, Now: localruntime.Clock{}.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	privateKey := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	secret, err := credentials.NewSecret(privateKey)
	clear(privateKey)
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	_, createErr := store.Create(context.Background(), credentials.CreateRequest{
		ActorRef:      "actor:local-owner",
		RequestRef:    "request:runtime-isolation-microvm-create",
		CredentialRef: "credential:microvm-launch-signing",
		OwnerRef:      "actor:local-owner",
		ScopeRefs:     []credentials.ScopeRef{"project:default"},
		PurposeRef:    "orquesta.microvm-launch-grant.v1",
		Material:      secret,
	})
	secret.Destroy()
	closeErr := store.Close()
	if createErr != nil || closeErr != nil {
		t.Fatalf("provision microvm credential: create=%v close=%v", createErr, closeErr)
	}
}
