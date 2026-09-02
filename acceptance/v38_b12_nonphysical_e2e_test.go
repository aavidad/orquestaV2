package acceptance_test

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/adapters/agent/agentmicrovm"
	"orquesta/internal/application"
	"orquesta/internal/credentials"
	"orquesta/internal/ports"
)

type v38B12AuthorityReader struct{}

type v38B12SessionBroker struct{}

func (v38B12SessionBroker) Ensure(_ context.Context, request ports.ExecutionSessionEnsureRequest) (ports.ExecutionSessionReceipt, error) {
	authority, err := application.DeriveExecutionSessionAuthority(request, "execution_token")
	return ports.ExecutionSessionReceipt{Authority: authority, EnsuredAt: time.Unix(1, 0).UTC()}, err
}

func (v38B12SessionBroker) Revoke(context.Context, ports.ExecutionSessionEnsureRequest) error {
	return nil
}

func (v38B12AuthorityReader) DescribeUseAuthority(_ context.Context, request credentials.DescribeUseAuthorityRequest) (credentials.DescribedUseAuthority, error) {
	return credentials.DescribedUseAuthority{CredentialRef: request.CredentialRef, OwnerRef: request.OwnerRef, ScopeRef: request.ScopeRef, PurposeRef: request.PurposeRef, Version: 1}, nil
}

type v38B12Signer struct {
	signer interface {
		Preparar(microvm.ContextoAutorizado, microvm.PlanLanzamiento, time.Time, time.Duration) (microvm.SolicitudLanzamiento, error)
	}
}

func (s v38B12Signer) Preparar(_ context.Context, _ ports.AgentLaunchRequest, authority microvm.ContextoAutorizado, plan microvm.PlanLanzamiento, at time.Time, validity time.Duration) (microvm.SolicitudLanzamiento, error) {
	return s.signer.Preparar(authority, plan, at, validity)
}

type v38B12Renderer string

func (r v38B12Renderer) RenderAgentPrompt(ports.AgentPrompt) (string, error) { return string(r), nil }

type v38B12RecordingLauncher struct {
	adapter *agentmicrovm.Adapter
	request ports.AgentLaunchRequest
	receipt ports.AgentLaunchReceipt
	err     error
}

func (launcher *v38B12RecordingLauncher) Launch(ctx context.Context, request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
	receipt, err := launcher.adapter.Launch(ctx, request)
	launcher.err = err
	if err == nil {
		launcher.request, launcher.receipt = request, receipt
	}
	return receipt, err
}

func (launcher *v38B12RecordingLauncher) Capabilities(ctx context.Context) (ports.AgentCapabilities, error) {
	return launcher.adapter.Capabilities(ctx)
}

func TestV38B12NonphysicalE2EPersistsLaunchChainAndFailsClosedAtQuiesce(t *testing.T) {
	base := nuevoSistemaCompuertaAV38(t, 1)
	client, calls := v38B12UDSPeer(t)
	placement := base.colocacion
	resolver, err := agentmicrovm.NewCredentialClaimResolver(v38B12AuthorityReader{}, "purpose:microvm-launch", []agentmicrovm.CredentialClaimBinding{{PlacementRef: placement, CredentialRef: "credential:v38-b12"}})
	if err != nil {
		t.Fatal(err)
	}
	descriptor := v38B12Descriptor(t)
	grantSigner, err := microvm.NuevoFirmanteConcesiones("clave-publica:v38-b12", ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)))
	if err != nil {
		t.Fatal(err)
	}
	capabilities := ports.AgentCapabilities{ProviderRef: "provider:codex", ModelRef: "model:codex-microvm", AgentRef: "agent:codex-microvm", RequierePreservacionEntorno: true, RoleKeys: []string{"role:worker"}, SkillRefs: []string{"skill:go"}, ToolRefs: []string{"tool:test"}, CapabilityRefs: []string{"capability:code"}}
	adapter, err := agentmicrovm.New(agentmicrovm.Config{Client: client, Signer: v38B12Signer{grantSigner}, ClaimResolver: resolver, LaunchAuthorityRegistry: base.repositorio, Capabilities: capabilities, ModelBinding: agentmicrovm.ProviderModelBinding{ModelRef: capabilities.ModelRef, ProviderModel: "gpt-5", Profile: agentmicrovm.ProfileBinding{PlacementRef: placement, Descriptor: descriptor}}, PromptRenderer: v38B12Renderer("execute exact work")})
	if err != nil {
		t.Fatal(err)
	}
	launcher := &v38B12RecordingLauncher{adapter: adapter}
	negotiated, err := adapter.Capabilities(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	orchestrator, err := application.New(application.Dependencies{State: base.repositorio, Access: base.repositorio, ExecutionSessions: v38B12SessionBroker{}, Launcher: launcher, Observer: adapter, Controller: adapter, Artifacts: base.artefactos, Clock: base.reloj, IDs: base.identidades, MaxOutputBytes: 1 << 20, MaxMailboxEnvelopeBytes: 64 << 10, MaxExecutionAttempts: 3, MaxChildrenPerParent: 6, ClaimLease: time.Minute, AttestTestClaimLease: time.Minute, DirectorLeaseDuration: 2 * time.Minute, EffectApprovalTTL: base.politica.EffectApprovalTTL, BudgetPolicy: base.politica, ObservationDelay: time.Second, ExecutionTimeout: time.Hour, AgentCapabilities: negotiated, CapacityObservationWait: time.Second, CapacitySources: []application.FuenteCapacidadColocacionAgente{{PlacementRef: placement, SourceRef: "capacity-source:v38-b12", PoolRef: "capacity-pool:v38-b12", BaseMedicion: application.BaseMedicionCapacidadBruta, Observer: base.observador}}, AgentLifecycle: &application.AgentEnvironmentLifecycleComposition{Store: base.repositorio, Physical: adapter, Reconciler: adapter, HistoricalResolver: base.repositorio, HistoricalPreserver: adapter, BuildPreservation: func(context.Context, ports.AgentPreserveReceipt, application.GoalRecord, time.Time) (application.ComprobantePreservacionEntornoAgente, error) {
		return application.ComprobantePreservacionEntornoAgente{}, errors.New("not reached")
	}}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := orchestrator.Submit(context.Background(), v06Access(t), application.SubmitRequest{RequestRef: "request:v38-b12-e2e", Statement: "exercise nonphysical B12", Confirm: true, Plan: planCompuertaAV38(1)})
	if err != nil {
		t.Fatal(err)
	}
	claim, found, err := orchestrator.ClaimNextAction(context.Background(), "worker:v38-b12", application.ActionClaimSelection{})
	if err != nil || !found {
		t.Fatalf("claim found=%v err=%v", found, err)
	}
	if _, err := orchestrator.ProcessClaim(context.Background(), claim); err != nil {
		t.Fatalf("process=%v adapter=%v code=%s", err, launcher.err, agentmicrovm.ErrorCode(launcher.err))
	}
	record, err := base.repositorio.GetGoal(context.Background(), result.Record.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	if len(record.EffectAttempts) != 1 || len(record.EffectReceipts) != 1 || record.EffectReceipts[0].Status != application.EffectStatusAccepted {
		t.Fatalf("launch chain incomplete: attempts=%d receipts=%+v", len(record.EffectAttempts), record.EffectReceipts)
	}
	key := ports.AgentHistoricalRuntimeAuthorityKey{ExecutionRef: record.EffectAttempts[0].Subject.ExecutionRef, ActionFence: record.EffectAttempts[0].ActionFence}
	subject := ports.AgentEnvironmentLifecycleSubject{ExecutionRef: launcher.receipt.ExecutionRef, GoalRef: launcher.receipt.GoalRef, WorkItemRef: launcher.receipt.WorkItemRef, PlanGeneration: launcher.receipt.PlanGeneration, AppSpecGeneration: launcher.receipt.AppSpecGeneration, ExecutionAttempt: launcher.receipt.ExecutionAttempt, SpecHash: launcher.receipt.SpecHash, ProviderRef: launcher.receipt.ProviderRef, ModelRef: launcher.receipt.ModelRef, AgentRef: launcher.receipt.AgentRef, ExternalRef: launcher.receipt.ExternalRef}
	token, _ := ports.NewAgentPhysicalToken(subject.ExternalRef)
	revision, _ := ports.NewAgentPhysicalRevision("3")
	fence, _ := ports.NewAgentPhysicalFence(strconv.FormatUint(key.ActionFence, 10))
	preserveRequest := ports.AgentPreserveRequest{Subject: subject, ExpectedToken: ports.AgentEnvironmentLifecycleToken{PhysicalToken: token, Revision: revision, Fence: fence, State: ports.AgentEnvironmentQuiesced}, IdempotencyKey: "preserve:v38-b12"}
	if _, preserveErr := application.PreserveAgentEnvironmentWithHistoricalAuthority(context.Background(), base.repositorio, key, preserveRequest, adapter); agentmicrovm.ErrorCode(preserveErr) != agentmicrovm.CodePreserveResponseInsufficient {
		runtime, runtimeErr := base.repositorio.ResolveRuntime(context.Background(), ports.MicroVMHostLaunchAuthorityKey{RunRef: key.ExecutionRef, ActionFence: key.ActionFence})
		t.Fatalf("authority-aware Preserve must reach UDS then fail closed, got %v runtime=%+v runtimeErr=%v request=%+v attempts=%+v receipts=%+v executions=%+v", preserveErr, runtime, runtimeErr, launcher.request, record.EffectAttempts, record.EffectReceipts, record.Executions)
	}
	_, err = adapter.Quiesce(context.Background(), ports.AgentQuiesceRequest{Subject: subject, ExpectedToken: ports.AgentEnvironmentLifecycleToken{PhysicalToken: token, Revision: revision, Fence: fence, State: ports.AgentEnvironmentActive}, IdempotencyKey: "quiesce:v38-b12"})
	if err == nil || agentmicrovm.ErrorCode(err) != agentmicrovm.CodeLifecycleReceiptUnavailable {
		t.Fatalf("Quiesce must fail closed, got %v", err)
	}
	if *calls < 5 {
		t.Fatalf("UDS calls=%d", *calls)
	}
}

func v38B12Descriptor(t *testing.T) microvm.DescriptorPerfilLanzamientoV1 {
	d := microvm.DescriptorPerfilLanzamientoV1{Esquema: microvm.EsquemaDescriptorPerfilLanzamientoV1, VCPU: 2, MemoriaMiB: 512, KernelSHA256: strings.Repeat("1", 64), InitramfsSHA256: strings.Repeat("2", 64), PerfilSHA256: strings.Repeat("3", 64), ServiciosDisponibles: []microvm.ServicioVsock{{Papel: "control_broker", ServicioRef: "servicio:control", Puerto: 10001, IdentidadRef: "identidad-servicio:control", IdentidadSHA256: strings.Repeat("4", 64)}}}
	ref, err := microvm.ConstruirEjecutorRefPerfilV1(d.PerfilSHA256)
	if err != nil {
		t.Fatal(err)
	}
	d.EjecutorRef = ref
	digest, err := microvm.CalcularSHA256DescriptorPerfilLanzamientoV1(d)
	if err != nil {
		t.Fatal(err)
	}
	d.DescriptorRef = "perfil-lanzamiento:sha256:" + digest
	return d
}

func v38B12UDSPeer(t *testing.T) (*microvm.Cliente, *int) {
	t.Helper()
	socket := filepath.Join(t.TempDir(), "agentmicrovm.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	calls := new(int)
	external := "ejecucion:" + strings.Repeat("a", 64)
	manifest := []byte(fmt.Sprintf(
		`{"protocolo":"agentmicrovm.preservacion.v1","contexto":{"referencia":%q,"cerca":1,"revision_trabajo":2}}`,
		external,
	))
	manifestSHA256 := fmt.Sprintf("%x", sha256.Sum256(manifest))
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*calls++
		w.Header().Set(microvm.CabeceraProtocolo, microvm.ProtocoloLocal)
		w.Header().Set("content-type", "application/json")
		switch {
		case r.URL.Path == "/v1/capacidades":
			json.NewEncoder(w).Encode(microvm.RespuestaCapacidades{Protocolo: microvm.ProtocoloLocal, Version: "0.1.0", Operaciones: []string{"salud", "capacidades", "crear_ejecucion", "reconciliar_lanzamiento", "consultar_ejecucion", "consultar_revision_trabajo", "iniciar_sesion", "enviar_entrada_sesion", "leer_eventos_sesion", "reconciliar_entrada_sesion", "detener_ejecucion"}, KVMDisponible: true, FirecrackerConfigurado: true, FirecrackerEjecutable: true, MaximoEjecuciones: 1})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/ejecuciones":
			var request microvm.SolicitudLanzamiento
			_ = json.NewDecoder(r.Body).Decode(&request)
			var plan microvm.PlanLanzamiento
			_ = json.Unmarshal(request.Plan, &plan)
			json.NewEncoder(w).Encode(microvm.RespuestaEjecucion{Referencia: external, Estado: "disponible", Revision: 3, Cerca: plan.Cerca, VCPU: 2, MemoriaMiB: 512, Identidad: &microvm.IdentidadProceso{PID: 123, InicioTicks: 456}})
		case r.URL.Path == "/v1/ejecuciones/"+external+"/trabajo":
			json.NewEncoder(w).Encode(microvm.RespuestaRevisionTrabajo{Referencia: external, RevisionTrabajo: 1})
		case strings.Contains(r.URL.Path, "/entradas/reconciliacion"):
			session := strings.Split(r.URL.Path, "/")[5]
			var fence uint64
			_, _ = fmt.Sscan(r.URL.Query().Get("cerca"), &fence)
			receipt := microvm.RespuestaSesionTrabajoV1{EjecucionRef: external, SesionRef: session, Estado: microvm.EstadoSesionActiva, Revision: 4, RevisionTrabajo: 2, RevisionSesion: 2, Cerca: fence}
			json.NewEncoder(w).Encode(microvm.RespuestaReconciliacionEntradaSesionTrabajoV1{EjecucionRef: external, SesionRef: session, Cerca: fence, Estado: microvm.EstadoReconciliacionEntradaResuelta, Comprobante: &receipt})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/ejecuciones/"+external:
			json.NewEncoder(w).Encode(microvm.RespuestaEjecucion{Referencia: external, Estado: "disponible", Revision: 3, Cerca: 1, VCPU: 2, MemoriaMiB: 512, Identidad: &microvm.IdentidadProceso{PID: 123, InicioTicks: 456}})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/ejecuciones/"+external+"/preservacion":
			var request microvm.SolicitudPreservacion
			_ = json.NewDecoder(r.Body).Decode(&request)
			json.NewEncoder(w).Encode(microvm.RespuestaPreservacion{Ejecucion: microvm.RespuestaEjecucion{Referencia: external, Estado: "preservada", Revision: request.RevisionEsperada + 1, Cerca: request.Cerca}, RevisionTrabajo: 2, ManifiestoSHA256: manifestSHA256, ManifiestoBytes: uint64(len(manifest))})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/ejecuciones/"+external+"/preservacion/manifiesto":
			json.NewEncoder(w).Encode(microvm.RespuestaManifiestoPreservacion{
				Referencia: external, Cerca: 1, RevisionTrabajo: 2,
				ManifiestoRef: manifestSHA256, ManifiestoSHA256: manifestSHA256,
				ManifiestoBytes: uint64(len(manifest)), SelladaUnixMS: 1_786_905_600_000,
				ContenidoBase64: base64.StdEncoding.EncodeToString(manifest),
			})
		default:
			http.NotFound(w, r)
		}
	})}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("serve UDS: %v", err)
		}
	}()
	t.Cleanup(func() { _ = server.Close() })
	client, err := microvm.Nuevo(socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.LiberarConexiones)
	return client, calls
}
