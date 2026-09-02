package bootstrap

import (
	"context"
	"crypto/ed25519"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/adapters/agent/codex"
	credentiallocal "orquesta/internal/adapters/credentials/local"
	localruntime "orquesta/internal/adapters/system/local"
	"orquesta/internal/application"
	"orquesta/internal/config"
	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type clienteRuntimeIsolationMicroVM struct {
	clienteFactoriaAgentMicroVM
	antesCapacidades func()
	falloCapacidades error
	mutarCapacidades func(*microvm.RespuestaCapacidades)
	consultas        atomic.Int64
	observada        string
	claveDetencion   string
	referencia       string
	detencion        microvm.SolicitudDetencion
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
	respuesta, err := cliente.clienteFactoriaAgentMicroVM.Capacidades(ctx)
	if err == nil && cliente.mutarCapacidades != nil {
		cliente.mutarCapacidades(&respuesta)
	}
	return respuesta, err
}

func (cliente *clienteRuntimeIsolationMicroVM) Observar(_ context.Context, referencia string) (microvm.RespuestaEjecucion, error) {
	cliente.observada = referencia
	return microvm.RespuestaEjecucion{Referencia: referencia, Estado: "disponible", Revision: 17, Cerca: 23}, nil
}

func (cliente *clienteRuntimeIsolationMicroVM) Detener(_ context.Context, clave, referencia string, solicitud microvm.SolicitudDetencion) (microvm.RespuestaDetencion, error) {
	cliente.claveDetencion, cliente.referencia, cliente.detencion = clave, referencia, solicitud
	modo := microvm.ModoDetencionEfectivoForzada
	receipt, confirmada := "detencion:"+strings.Repeat("e", 64), uint64(1_786_905_600_000)
	return microvm.RespuestaDetencion{
		Ejecucion:         microvm.RespuestaEjecucion{Referencia: referencia, Estado: "detenida", Revision: 18, Cerca: solicitud.Cerca},
		ClaveIdempotencia: clave, Estado: microvm.EstadoDetencionConfirmada,
		ModoSolicitado: solicitud.Modo, ModoEfectivo: &modo, ReceiptRef: &receipt, ConfirmadaUnixMS: &confirmada,
	}, nil
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
	configPath, socketPath, brokerPath := writeRuntimeIsolationMicroVMConfig(t, root, true)
	provisionRuntimeIsolationMicroVMCredential(t, root)
	statePath := filepath.Join(root, "state", "orquesta.sqlite")

	client := &clienteRuntimeIsolationMicroVM{}
	var constructorCalls atomic.Int64
	var connectionCloses atomic.Int64
	var brokerCerradoAntesDelCliente atomic.Bool
	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath,
		constructorClienteAgentMicroVM: func(gotSocket string) (recursoClienteAgentMicroVM, error) {
			constructorCalls.Add(1)
			if info, err := os.Lstat(statePath); err != nil || !info.Mode().IsRegular() {
				t.Fatalf("constructor microVM anterior al repositorio: info=%v err=%v", info, err)
			}
			if gotSocket != socketPath {
				t.Fatalf("microvm socket=%q want=%q", gotSocket, socketPath)
			}
			return recursoClienteAgentMicroVM{
				cliente: client,
				liberarConexiones: func() error {
					_, statErr := os.Lstat(brokerPath)
					brokerCerradoAntesDelCliente.Store(errors.Is(statErr, os.ErrNotExist))
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
	controller, ok := runtime.agent.(application.AgentController)
	if !ok {
		t.Fatalf("microvm agent omitted AgentController: %T", runtime.agent)
	}
	controlCapabilities, err := controller.ControlCapabilities(context.Background())
	if err != nil || !controlCapabilities.CooperativeStop || !controlCapabilities.ForcedStop {
		t.Fatalf("microvm control capabilities=%+v err=%v", controlCapabilities, err)
	}
	executionRef, _ := goal.NewExecutionRef("execution:runtime-stop")
	goalRef, _ := goal.NewGoalRef("goal:runtime-stop")
	workItemRef, _ := goal.NewWorkItemRef("work-item:runtime-stop")
	stop := ports.AgentStopRequest{
		ExecutionRef: executionRef, GoalRef: goalRef, WorkItemRef: workItemRef,
		PlanGeneration: 2, AppSpecGeneration: 3, ExecutionAttempt: 1, SpecHash: strings.Repeat("d", 64),
		ProviderRef: codex.ProviderRef, ModelRef: codex.DefaultModelRef, AgentRef: codex.AgentRef,
		ExternalRef: "ejecucion:" + strings.Repeat("a", 64), Mode: ports.AgentStopForced,
		IdempotencyKey: "stop:runtime-isolation",
	}
	stopReceipt, err := controller.Stop(context.Background(), stop)
	wantPhysicalStop := microvm.SolicitudDetencion{RevisionEsperada: 17, Cerca: 23, Modo: microvm.ModoDetencionForzada}
	if err != nil || client.observada != stop.ExternalRef || client.claveDetencion != stop.IdempotencyKey ||
		client.referencia != stop.ExternalRef || client.detencion != wantPhysicalStop ||
		stopReceipt.Status != ports.AgentStopped || stopReceipt.ReceiptRef != "detencion:"+strings.Repeat("e", 64) ||
		ports.ValidateAgentStopReceipt(stop, stopReceipt) != nil {
		t.Fatalf("physical stop observation=%q key=%q ref=%q request=%+v receipt=%+v err=%v",
			client.observada, client.claveDetencion, client.referencia, client.detencion, stopReceipt, err)
	}
	if runtime.workspace == nil {
		t.Fatal("configured repository did not compose workspace")
	}
	brokerInfo, err := os.Lstat(brokerPath)
	if err != nil || brokerInfo.Mode().Type() != os.ModeSocket || brokerInfo.Mode().Perm() != 0o600 {
		t.Fatalf("credential broker info=%v err=%v", brokerInfo, err)
	}
	descriptors, err := runtime.agent.(catalogoCapacidadColocacionAgente).DescribirCapacidadColocaciones()
	if err != nil || len(descriptors) != 1 || descriptors[0].Plazas != 16 ||
		descriptors[0].PlacementRef.String() != "placement:codex:microvm-runtime" {
		t.Fatalf("physical microvm capacity=%+v err=%v", descriptors, err)
	}
	if constructorCalls.Load() != 1 || client.consultas.Load() != 4 || connectionCloses.Load() != 0 {
		t.Fatalf("constructor=%d capabilities=%d cleanup=%d before shutdown",
			constructorCalls.Load(), client.consultas.Load(), connectionCloses.Load())
	}
	if err := runtime.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown(microvm) error=%v", err)
	}
	if connectionCloses.Load() != 1 || !brokerCerradoAntesDelCliente.Load() {
		t.Fatalf("connection cleanup=%d broker before=%t want=1/true",
			connectionCloses.Load(), brokerCerradoAntesDelCliente.Load())
	}
}

func TestBuildMicroVMCapabilitiesFailureCleansClientAfterOpeningDurableState(t *testing.T) {
	root := t.TempDir()
	configPath, socketPath, brokerPath := writeRuntimeIsolationMicroVMConfig(t, root, false)
	provisionRuntimeIsolationMicroVMCredential(t, root)
	want := errors.New("runtime_isolation.capabilities_failed")
	statePath := filepath.Join(root, "state", "orquesta.sqlite")
	client := &clienteRuntimeIsolationMicroVM{
		falloCapacidades: want,
		antesCapacidades: func() {
			if info, err := os.Lstat(statePath); err != nil || !info.Mode().IsRegular() {
				t.Fatalf("Capabilities ran before SQLite state: info=%v err=%v", info, err)
			}
		},
	}
	var connectionCloses atomic.Int64
	var brokerCerradoAntesDelCliente atomic.Bool

	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath,
		constructorClienteAgentMicroVM: func(gotSocket string) (recursoClienteAgentMicroVM, error) {
			if gotSocket != socketPath {
				t.Fatalf("microvm socket=%q want=%q", gotSocket, socketPath)
			}
			if info, statErr := os.Lstat(statePath); statErr != nil || !info.Mode().IsRegular() {
				t.Fatalf("constructor microVM anterior al repositorio: info=%v err=%v", info, statErr)
			}
			return recursoClienteAgentMicroVM{
				cliente: client,
				liberarConexiones: func() error {
					_, statErr := os.Lstat(brokerPath)
					brokerCerradoAntesDelCliente.Store(errors.Is(statErr, os.ErrNotExist))
					connectionCloses.Add(1)
					return nil
				},
			}, nil
		},
	})
	if runtime != nil || !errors.Is(err, want) {
		t.Fatalf("capabilities failure: runtime=%v err=%v", runtime, err)
	}
	if client.consultas.Load() != 1 || connectionCloses.Load() != 1 || !brokerCerradoAntesDelCliente.Load() {
		t.Fatalf("capabilities=%d cleanup=%d broker before=%t want=1/1/true",
			client.consultas.Load(), connectionCloses.Load(), brokerCerradoAntesDelCliente.Load())
	}
	if info, statErr := os.Lstat(statePath); statErr != nil || !info.Mode().IsRegular() {
		t.Fatalf("failed Capabilities lost durable state: info=%v err=%v", info, statErr)
	}
	for _, path := range []string{
		filepath.Join(root, "artifacts"),
		filepath.Join(root, "effective_config.json"),
		filepath.Join(root, "work"),
	} {
		if _, statErr := os.Lstat(path); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("failed Capabilities created %s: %v", path, statErr)
		}
	}
}

func TestBuildMicroVMRejectsDifferentProtocolOrVersionWithoutFallback(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*microvm.RespuestaCapacidades)
	}{
		{name: "protocol", mutate: func(response *microvm.RespuestaCapacidades) {
			response.Protocolo = "agentmicrovm.local.v0"
		}},
		{name: "version", mutate: func(response *microvm.RespuestaCapacidades) {
			response.Version = "0.0.0-incompatible"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			configPath, socketPath, brokerPath := writeRuntimeIsolationMicroVMConfig(t, root, false)
			provisionRuntimeIsolationMicroVMCredential(t, root)
			client := &clienteRuntimeIsolationMicroVM{mutarCapacidades: test.mutate}
			var constructorCalls, connectionCloses atomic.Int64

			runtime, err := Build(context.Background(), Options{
				ConfigPath: configPath,
				constructorClienteAgentMicroVM: func(gotSocket string) (recursoClienteAgentMicroVM, error) {
					constructorCalls.Add(1)
					if gotSocket != socketPath {
						t.Fatalf("microvm socket=%q want=%q", gotSocket, socketPath)
					}
					return recursoClienteAgentMicroVM{
						cliente:           client,
						liberarConexiones: func() error { connectionCloses.Add(1); return nil },
					}, nil
				},
			})
			if runtime != nil || err == nil {
				t.Fatalf("incompatible %s admitted: runtime=%v err=%v", test.name, runtime, err)
			}
			if constructorCalls.Load() != 1 || client.consultas.Load() > 1 || connectionCloses.Load() != 1 {
				t.Fatalf("incompatible %s fallback/cleanup: constructor=%d capabilities=%d closes=%d",
					test.name, constructorCalls.Load(), client.consultas.Load(), connectionCloses.Load())
			}
			if _, statErr := os.Lstat(brokerPath); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("incompatible %s retained broker socket: %v", test.name, statErr)
			}
			for _, path := range []string{filepath.Join(root, "artifacts"), filepath.Join(root, "effective_config.json")} {
				if _, statErr := os.Lstat(path); !errors.Is(statErr, os.ErrNotExist) {
					t.Fatalf("incompatible %s fell back and created %s: %v", test.name, path, statErr)
				}
			}
		})
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
		dependenciasAutoridadFisicaAgentMicroVM{},
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

func writeRuntimeIsolationMicroVMConfig(t *testing.T, root string, withRepository bool) (string, string, string) {
	t.Helper()
	_, descriptorRaw := descriptorFactoriaAgentMicroVM(t)
	descriptorPath := filepath.Join(root, "perfil-microvm.json")
	if err := os.WriteFile(descriptorPath, descriptorRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	socketPath := filepath.Join(root, "agente-microvm.sock")
	brokerPath := filepath.Join(root, "b.sock")
	configPath := writeTestConfig(t, root)
	replaceTestConfigValue(t, configPath, "[runtime]\n", "[runtime]\nprovider = \"codex\"\nisolation = \"microvm\"\n")
	replaceTestConfigValue(t, configPath, "[runtime.codex]\n", `[runtime.microvm]
placement_ref = "placement:codex:microvm-runtime"
socket_path = `+strconv.Quote(socketPath)+`
profile_descriptor_path = `+strconv.Quote(descriptorPath)+`
expected_profile_descriptor_sha256 = "`+digestRawFactoriaAgentMicroVM(descriptorRaw)+`"
launch_grant_key_id = "clave-publica:runtime-isolation"
launch_grant_signing_credential_ref = "credential:microvm-launch-signing"
expired_launch_continuation_authority_signing_credential_ref = "credential:microvm-continuation-signing"
expired_launch_continuation_authority_key_id = "continuation-runtime-isolation"
expired_launch_continuation_authority_key_epoch = 1
expired_launch_continuation_authority_trust_revision = 1
expired_launch_continuation_authority_public_key_sha256 = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
expired_launch_continuation_authority_validity = "2m"
credential_broker_socket_path = `+strconv.Quote(brokerPath)+`
credential_broker_peer_uid = `+strconv.Itoa(os.Geteuid())+`
credential_broker_exchange_timeout = "1s"
credential_broker_max_connections = 16

[runtime.codex]
model = "gpt-5.6"
credential_ref = "credential:codex-account-runtime"
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
	return configPath, socketPath, brokerPath
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
	authSecret, authSecretErr := credentials.NewSecret([]byte(`{"auth":"fixture"}`))
	if authSecretErr != nil {
		_ = store.Close()
		t.Fatal(authSecretErr)
	}
	_, createAuthErr := store.Create(context.Background(), credentials.CreateRequest{
		ActorRef:      "actor:local-owner",
		RequestRef:    "request:runtime-isolation-microvm-create-codex-account",
		CredentialRef: "credential:codex-account-runtime",
		OwnerRef:      "actor:local-owner",
		ScopeRefs:     []credentials.ScopeRef{"project:default"},
		PurposeRef:    credentials.PurposeRef(codex.ProviderRef),
		Material:      authSecret,
	})
	authSecret.Destroy()
	closeErr := store.Close()
	if createErr != nil || createAuthErr != nil || closeErr != nil {
		t.Fatalf("provision microvm credential: signing=%v auth=%v close=%v", createErr, createAuthErr, closeErr)
	}
}
