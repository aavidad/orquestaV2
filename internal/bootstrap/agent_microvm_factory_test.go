package bootstrap

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/adapters/agent/agentmicrovm"
	"orquesta/internal/adapters/agent/codex"
	"orquesta/internal/agentprotocol/codexwork"
	"orquesta/internal/config"
	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

type rendererFactoriaAgentMicroVM struct{}

func (rendererFactoriaAgentMicroVM) RenderAgentPrompt(ports.AgentPrompt) (string, error) {
	return "trabajo sellado", nil
}

type rendererTipadoNuloFactoriaAgentMicroVM struct{}

func (*rendererTipadoNuloFactoriaAgentMicroVM) RenderAgentPrompt(ports.AgentPrompt) (string, error) {
	return "", nil
}

type storeFactoriaAgentMicroVM struct {
	cierres       atomic.Int64
	material      []byte
	usos          []credentials.UseRequest
	secretos      []credentials.Secret
	descripciones []credentials.DescribeUseAuthorityRequest
}

func (*storeFactoriaAgentMicroVM) Create(context.Context, credentials.CreateRequest) (credentials.MutationResult, error) {
	return credentials.MutationResult{}, nil
}
func (store *storeFactoriaAgentMicroVM) Use(_ context.Context, request credentials.UseRequest, callback func(credentials.Secret) error) (credentials.Receipt, error) {
	store.usos = append(store.usos, request)
	secret, err := credentials.NewSecret(store.material)
	if err != nil {
		return credentials.Receipt{}, err
	}
	store.secretos = append(store.secretos, secret)
	if err := callback(secret); err != nil {
		return credentials.Receipt{}, err
	}
	return credentials.Receipt{
		CredentialRef: request.CredentialRef, OwnerRef: request.OwnerRef,
		ScopeRef: request.ScopeRef, PurposeRef: request.PurposeRef,
		Version: 1, RequestRef: request.RequestRef, ActorRef: request.ActorRef, Operation: "use",
	}, nil
}
func (*storeFactoriaAgentMicroVM) Rotate(context.Context, credentials.RotateRequest) (credentials.MutationResult, error) {
	return credentials.MutationResult{}, nil
}
func (*storeFactoriaAgentMicroVM) Revoke(context.Context, credentials.RevokeRequest) (credentials.MutationResult, error) {
	return credentials.MutationResult{}, nil
}
func (store *storeFactoriaAgentMicroVM) DescribeUseAuthority(
	_ context.Context,
	request credentials.DescribeUseAuthorityRequest,
) (credentials.DescribedUseAuthority, error) {
	store.descripciones = append(store.descripciones, request)
	return credentials.DescribedUseAuthority{
		CredentialRef: request.CredentialRef,
		OwnerRef:      request.OwnerRef,
		ScopeRef:      request.ScopeRef,
		PurposeRef:    request.PurposeRef,
		Version:       1,
	}, nil
}
func (store *storeFactoriaAgentMicroVM) UseOnce(
	_ context.Context,
	request credentials.OneShotUseRequest,
	consume func(credentials.Secret) error,
) (credentials.OneShotUseResult, error) {
	secret, err := credentials.NewSecret(store.material)
	if err != nil {
		return credentials.OneShotUseResult{}, err
	}
	defer secret.Destroy()
	if err := consume(secret); err != nil {
		return credentials.OneShotUseResult{}, err
	}
	return credentials.OneShotUseResult{Receipt: credentials.Receipt{
		CredentialRef: request.CredentialRef,
		OwnerRef:      request.OwnerRef,
		ScopeRef:      request.ScopeRef,
		PurposeRef:    request.PurposeRef,
		Version:       request.Version,
		RequestRef:    request.RequestRef,
		ActorRef:      request.ActorRef,
		Operation:     credentials.OperationUseOnce,
		OccurredAt:    time.Unix(1, 0).UTC(),
	}}, nil
}
func (store *storeFactoriaAgentMicroVM) Close() error {
	store.cierres.Add(1)
	return nil
}

type registroLanzamientosFactoriaAgentMicroVM struct {
	autoridad ports.MicroVMHostLaunchAuthorityV1
	prepares  atomic.Int64
	bindings  atomic.Int64
}

func (registro *registroLanzamientosFactoriaAgentMicroVM) Prepare(
	_ context.Context,
	autoridad ports.MicroVMHostLaunchAuthorityV1,
) (ports.MicroVMHostLaunchAuthorityV1, error) {
	registro.prepares.Add(1)
	registro.autoridad = ports.CloneMicroVMHostLaunchAuthorityV1(autoridad)
	return ports.CloneMicroVMHostLaunchAuthorityV1(registro.autoridad), nil
}

func (registro *registroLanzamientosFactoriaAgentMicroVM) BindExternal(
	_ context.Context,
	key ports.MicroVMHostLaunchAuthorityKey,
	externalRef string,
) (ports.MicroVMHostLaunchAuthorityV1, error) {
	registro.bindings.Add(1)
	if registro.autoridad.Key != key {
		return ports.MicroVMHostLaunchAuthorityV1{}, errors.New("registro factoria: key cruzada")
	}
	registro.autoridad.ExternalRef = externalRef
	return ports.CloneMicroVMHostLaunchAuthorityV1(registro.autoridad), nil
}

func (registro *registroLanzamientosFactoriaAgentMicroVM) Resolve(
	_ context.Context,
	key ports.MicroVMHostLaunchAuthorityKey,
) (ports.MicroVMHostLaunchAuthorityV1, error) {
	if registro.autoridad.Key != key {
		return ports.MicroVMHostLaunchAuthorityV1{}, errors.New("registro factoria: autoridad ausente")
	}
	return ports.CloneMicroVMHostLaunchAuthorityV1(registro.autoridad), nil
}

func autoridadFisicaFactoriaAgentMicroVM(
	lector credentials.UseAuthorityReader,
	registro ports.MicroVMHostLaunchAuthorityRegistry,
) dependenciasAutoridadFisicaAgentMicroVM {
	almacenOneShot, _ := lector.(credentials.OneShotStore)
	return dependenciasAutoridadFisicaAgentMicroVM{
		lectorCredencial: lector, almacenOneShot: almacenOneShot, registroLanzamientos: registro,
	}
}

type clienteFactoriaAgentMicroVM struct{}

func (*clienteFactoriaAgentMicroVM) Capacidades(context.Context) (microvm.RespuestaCapacidades, error) {
	return microvm.RespuestaCapacidades{
		Protocolo: microvm.ProtocoloLocal,
		Version:   "0.1.0",
		Operaciones: []string{
			"salud", "capacidades", "crear_ejecucion", "consultar_ejecucion",
			"consultar_revision_trabajo", "iniciar_sesion", "enviar_entrada_sesion",
			"leer_eventos_sesion", "reconciliar_entrada_sesion",
		},
		KVMDisponible: true, FirecrackerConfigurado: true, FirecrackerEjecutable: true,
		MaximoEjecuciones: 16,
	}, nil
}
func (*clienteFactoriaAgentMicroVM) Lanzar(context.Context, string, microvm.SolicitudLanzamiento) (microvm.RespuestaEjecucion, error) {
	return microvm.RespuestaEjecucion{}, nil
}
func (*clienteFactoriaAgentMicroVM) RevisionTrabajo(context.Context, string) (microvm.RespuestaRevisionTrabajo, error) {
	return microvm.RespuestaRevisionTrabajo{}, nil
}
func (*clienteFactoriaAgentMicroVM) IniciarSesion(context.Context, string, string, microvm.SolicitudIniciarSesionTrabajoV1) (microvm.RespuestaSesionTrabajoV1, error) {
	return microvm.RespuestaSesionTrabajoV1{}, nil
}
func (*clienteFactoriaAgentMicroVM) EnviarEntradaSesion(context.Context, string, string, string, microvm.SolicitudEntradaSesionTrabajoV1) (microvm.RespuestaSesionTrabajoV1, error) {
	return microvm.RespuestaSesionTrabajoV1{}, nil
}
func (*clienteFactoriaAgentMicroVM) ReconciliarEntradaSesion(context.Context, string, string, string, uint64) (microvm.RespuestaReconciliacionEntradaSesionTrabajoV1, error) {
	return microvm.RespuestaReconciliacionEntradaSesionTrabajoV1{}, nil
}
func (*clienteFactoriaAgentMicroVM) Observar(context.Context, string) (microvm.RespuestaEjecucion, error) {
	return microvm.RespuestaEjecucion{}, nil
}
func (*clienteFactoriaAgentMicroVM) LeerEventosSesion(context.Context, string, string, microvm.ConsultaEventosSesionTrabajoV1) (microvm.PaginaEventosSesionTrabajoV1, error) {
	return microvm.PaginaEventosSesionTrabajoV1{}, nil
}

type clienteLanzamientoFactoriaAgentMicroVM struct {
	solicitud        ports.AgentLaunchRequest
	descriptor       microvm.DescriptorPerfilLanzamientoV1
	claveLanzamiento string
	lanzamiento      microvm.SolicitudLanzamiento
	inicio           microvm.SolicitudIniciarSesionTrabajoV1
	entrada          microvm.SolicitudEntradaSesionTrabajoV1
	reconciliaciones int
}

func (*clienteLanzamientoFactoriaAgentMicroVM) Capacidades(ctx context.Context) (microvm.RespuestaCapacidades, error) {
	return (&clienteFactoriaAgentMicroVM{}).Capacidades(ctx)
}
func (cliente *clienteLanzamientoFactoriaAgentMicroVM) Lanzar(
	_ context.Context,
	clave string,
	solicitud microvm.SolicitudLanzamiento,
) (microvm.RespuestaEjecucion, error) {
	cliente.claveLanzamiento = clave
	cliente.lanzamiento = solicitud
	return microvm.RespuestaEjecucion{
		Referencia: "ejecucion:" + strings.Repeat("a", 64),
		Estado:     "disponible", Revision: 3,
		Cerca: cliente.solicitud.EffectAuthority.ActionFence,
		VCPU:  cliente.descriptor.VCPU, MemoriaMiB: cliente.descriptor.MemoriaMiB,
		Identidad: &microvm.IdentidadProceso{PID: 1234, InicioTicks: 5678},
	}, nil
}
func (*clienteLanzamientoFactoriaAgentMicroVM) RevisionTrabajo(
	_ context.Context,
	referencia string,
) (microvm.RespuestaRevisionTrabajo, error) {
	return microvm.RespuestaRevisionTrabajo{Referencia: referencia, RevisionTrabajo: 1}, nil
}
func (cliente *clienteLanzamientoFactoriaAgentMicroVM) IniciarSesion(
	_ context.Context,
	_ string,
	referencia string,
	solicitud microvm.SolicitudIniciarSesionTrabajoV1,
) (microvm.RespuestaSesionTrabajoV1, error) {
	cliente.inicio = solicitud
	return microvm.RespuestaSesionTrabajoV1{
		EjecucionRef: referencia, SesionRef: cliente.solicitud.SessionRef.String(),
		Estado: microvm.EstadoSesionActiva, Revision: 4, RevisionTrabajo: 2,
		RevisionSesion: 1, Cerca: cliente.solicitud.EffectAuthority.ActionFence,
	}, nil
}
func (cliente *clienteLanzamientoFactoriaAgentMicroVM) EnviarEntradaSesion(
	_ context.Context,
	_ string,
	referencia string,
	_ string,
	solicitud microvm.SolicitudEntradaSesionTrabajoV1,
) (microvm.RespuestaSesionTrabajoV1, error) {
	cliente.entrada = solicitud
	return microvm.RespuestaSesionTrabajoV1{
		EjecucionRef: referencia, SesionRef: cliente.solicitud.SessionRef.String(),
		Estado: microvm.EstadoSesionActiva, Revision: 5, RevisionTrabajo: 3,
		RevisionSesion: 2, Cerca: cliente.solicitud.EffectAuthority.ActionFence,
	}, nil
}
func (cliente *clienteLanzamientoFactoriaAgentMicroVM) ReconciliarEntradaSesion(
	context.Context,
	string,
	string,
	string,
	uint64,
) (microvm.RespuestaReconciliacionEntradaSesionTrabajoV1, error) {
	cliente.reconciliaciones++
	return microvm.RespuestaReconciliacionEntradaSesionTrabajoV1{}, &microvm.ErrorRespuesta{
		Estado: 404, Codigo: "entrada.ausente",
	}
}
func (cliente *clienteLanzamientoFactoriaAgentMicroVM) Observar(
	context.Context,
	string,
) (microvm.RespuestaEjecucion, error) {
	return microvm.RespuestaEjecucion{}, nil
}
func (*clienteLanzamientoFactoriaAgentMicroVM) LeerEventosSesion(
	context.Context,
	string,
	string,
	microvm.ConsultaEventosSesionTrabajoV1,
) (microvm.PaginaEventosSesionTrabajoV1, error) {
	return microvm.PaginaEventosSesionTrabajoV1{}, &microvm.ErrorRespuesta{
		Estado: 404, Codigo: "sesion.ausente",
	}
}

func TestProductionAgentMicroVMComponeBindingCapacidadYCierraSoloConexiones(t *testing.T) {
	snapshot, _, rutaSocket := fixtureFactoriaAgentMicroVM(t)
	rutaBroker := snapshot.RuntimeMicroVMCredentialBrokerSocketPath()
	store := &storeFactoriaAgentMicroVM{}
	registro := &registroLanzamientosFactoriaAgentMicroVM{}
	var conexiones atomic.Int64
	var brokerCerradoAntesDelCliente atomic.Bool
	agente, err := productionAgentMicroVMConConstructor(
		snapshot,
		rendererFactoriaAgentMicroVM{},
		store,
		autoridadFisicaFactoriaAgentMicroVM(store, registro),
		func(ruta string) (recursoClienteAgentMicroVM, error) {
			if ruta != rutaSocket {
				t.Fatalf("socket=%q want=%q", ruta, rutaSocket)
			}
			return recursoClienteAgentMicroVM{
				cliente: &clienteFactoriaAgentMicroVM{},
				liberarConexiones: func() error {
					_, statErr := os.Lstat(rutaBroker)
					brokerCerradoAntesDelCliente.Store(errors.Is(statErr, os.ErrNotExist))
					conexiones.Add(1)
					return nil
				},
			}, nil
		},
	)
	if err != nil || agente == nil {
		t.Fatalf("productionAgentMicroVM() agente=%v error=%v", agente, err)
	}
	infoBroker, err := os.Lstat(rutaBroker)
	if err != nil || infoBroker.Mode().Type() != os.ModeSocket || infoBroker.Mode().Perm() != 0o600 {
		t.Fatalf("broker publicado info=%v error=%v", infoBroker, err)
	}
	capacidades, err := agente.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities() error=%v", err)
	}
	if capacidades.ProviderRef != codex.ProviderRef || capacidades.ModelRef != codex.DefaultModelRef ||
		capacidades.AgentRef != codex.AgentRef || !capacidades.Unrestricted || !capacidades.RequierePreservacionEntorno {
		t.Fatalf("capacidades=%+v", capacidades)
	}
	descriptores, err := agente.DescribirCapacidadColocaciones()
	if err != nil || len(descriptores) != 1 ||
		descriptores[0].PlacementRef.String() != "placement:codex:microvm-prueba" || descriptores[0].Plazas != 16 {
		t.Fatalf("capacidad=%+v error=%v", descriptores, err)
	}
	if err := agente.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error=%v", err)
	}
	if err := agente.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() repetido error=%v", err)
	}
	if conexiones.Load() != 1 || !brokerCerradoAntesDelCliente.Load() || store.cierres.Load() != 0 {
		t.Fatalf("conexiones=%d broker antes=%t cierres store=%d",
			conexiones.Load(), brokerCerradoAntesDelCliente.Load(), store.cierres.Load())
	}
}

func TestProductionAgentMicroVMLaunchFirmaYEntregaBindingExactoOffline(t *testing.T) {
	snapshot, _, _ := fixtureFactoriaAgentMicroVM(t)
	descriptor, _ := descriptorFactoriaAgentMicroVM(t)
	solicitud := solicitudLanzamientoFactoriaAgentMicroVM(t)
	privada := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{7}, ed25519.SeedSize))
	store := &storeFactoriaAgentMicroVM{material: append([]byte(nil), privada...)}
	registro := &registroLanzamientosFactoriaAgentMicroVM{}
	cliente := &clienteLanzamientoFactoriaAgentMicroVM{solicitud: solicitud, descriptor: descriptor}
	agente, err := productionAgentMicroVMConConstructor(
		snapshot,
		rendererFactoriaAgentMicroVM{},
		store,
		autoridadFisicaFactoriaAgentMicroVM(store, registro),
		func(string) (recursoClienteAgentMicroVM, error) {
			return recursoClienteAgentMicroVM{
				cliente:           cliente,
				liberarConexiones: func() error { return nil },
			}, nil
		},
	)
	if err != nil {
		t.Fatalf("productionAgentMicroVM() error=%v", err)
	}
	defer agente.Shutdown(context.Background())
	recibo, err := agente.Launch(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("Launch() error=%v", err)
	}
	if recibo.ProviderRef != codex.ProviderRef || recibo.ModelRef != codex.DefaultModelRef ||
		recibo.AgentRef != codex.AgentRef || recibo.ExternalRef == "" ||
		!recibo.RequierePreservacionEntorno {
		t.Fatalf("recibo=%+v", recibo)
	}

	wantUse := credentials.UseRequest{
		ActorRef:      solicitud.ActorRef.String(),
		RequestRef:    "request:microvm-launch:" + solicitud.ExecutionRef.String(),
		CredentialRef: "credential:microvm-launch-signing",
		OwnerRef:      credentials.OwnerRef(solicitud.ActorRef.String()),
		ScopeRef:      credentials.ScopeRef(solicitud.ProjectRef.String()),
		PurposeRef:    "orquesta.microvm-launch-grant.v1",
		Version:       0,
	}
	if len(store.usos) != 1 || store.usos[0] != wantUse {
		t.Fatalf("CredentialStore.Use=%+v want=%+v", store.usos, wantUse)
	}
	if len(store.descripciones) != 1 ||
		store.descripciones[0].CredentialRef != "credential:codex-account-1" ||
		store.descripciones[0].PurposeRef != credentials.PurposeRef(codex.ProviderRef) {
		t.Fatalf("DescribeUseAuthority=%+v", store.descripciones)
	}
	if registro.prepares.Load() != 1 || registro.bindings.Load() != 1 ||
		registro.autoridad.OneShotClaim.CredentialRef != "credential:codex-account-1" ||
		registro.autoridad.OneShotClaim.PurposeRef != credentials.PurposeRef(codex.ProviderRef) ||
		registro.autoridad.OneShotClaim.OwnerRef.String() != solicitud.ActorRef.String() ||
		registro.autoridad.OneShotClaim.ScopeRef.String() != solicitud.ProjectRef.String() ||
		registro.autoridad.ExternalRef != recibo.ExternalRef {
		t.Fatalf("registro autoridad=%+v prepare=%d bind=%d",
			registro.autoridad, registro.prepares.Load(), registro.bindings.Load())
	}
	if len(store.secretos) != 1 || !bytes.Equal(store.secretos[0].Bytes(), make([]byte, ed25519.PrivateKeySize)) {
		t.Fatal("la clave privada no quedó destruida al salir del callback")
	}

	var plan microvm.PlanLanzamiento
	if err := json.Unmarshal(cliente.lanzamiento.Plan, &plan); err != nil {
		t.Fatalf("plan firmado inválido: %v", err)
	}
	if plan.RunRef != solicitud.ExecutionRef.String() || plan.Cerca != solicitud.EffectAuthority.ActionFence ||
		plan.VCPU != descriptor.VCPU || plan.MemoriaMiB != descriptor.MemoriaMiB ||
		plan.KernelSHA256 != descriptor.KernelSHA256 || plan.InitramfsSHA256 != descriptor.InitramfsSHA256 ||
		plan.PerfilSHA256 == nil || *plan.PerfilSHA256 != descriptor.PerfilSHA256 ||
		!reflect.DeepEqual(plan.Servicios, descriptor.ServiciosDisponibles) {
		t.Fatalf("plan=%+v descriptor=%+v", plan, descriptor)
	}
	if cliente.claveLanzamiento != solicitud.IdempotencyKey {
		t.Fatalf("clave lanzamiento=%q want=%q", cliente.claveLanzamiento, solicitud.IdempotencyKey)
	}
	concesion := verificarConcesionFactoriaAgentMicroVM(
		t,
		cliente.lanzamiento,
		privada.Public().(ed25519.PublicKey),
	)
	if concesion.ClaveID != "clave-publica:orquesta-prueba" ||
		concesion.RunRef != solicitud.ExecutionRef.String() || concesion.Cerca != solicitud.EffectAuthority.ActionFence {
		t.Fatalf("concesión=%+v", concesion)
	}

	if cliente.inicio.SesionRef != solicitud.SessionRef.String() ||
		cliente.inicio.EjecutorRef != descriptor.EjecutorRef ||
		cliente.inicio.Cerca != solicitud.EffectAuthority.ActionFence || cliente.reconciliaciones != 2 {
		t.Fatalf("inicio=%+v reconciliaciones=%d", cliente.inicio, cliente.reconciliaciones)
	}
	entradaRaw, err := base64.StdEncoding.Strict().DecodeString(cliente.entrada.ContenidoBase64)
	if err != nil || !cliente.entrada.CerrarStdin {
		t.Fatalf("entrada=%+v error=%v", cliente.entrada, err)
	}
	paquete, err := codexwork.DecodeWorkPacketV1(entradaRaw)
	if err != nil || paquete.Model != "gpt-5.6" || paquete.Schema != codexwork.WorkPacketSchemaV1 ||
		paquete.ExecutionRef != solicitud.ExecutionRef.String() ||
		paquete.EffectAttemptRef != solicitud.EffectAuthority.EffectAttemptRef {
		t.Fatalf("paquete=%+v error=%v", paquete, err)
	}
	descriptores, err := agente.DescribirCapacidadColocaciones()
	if err != nil || len(descriptores) != 1 ||
		descriptores[0].PlacementRef != solicitud.ReferenciaColocacion {
		t.Fatalf("placement negociado=%+v error=%v", descriptores, err)
	}
}

func TestProductionAgentMicroVMConstruyeClientePublicoSinMarcarSocketNiKVM(t *testing.T) {
	snapshot, _, rutaSocket := fixtureFactoriaAgentMicroVM(t)
	store := &storeFactoriaAgentMicroVM{}
	registro := &registroLanzamientosFactoriaAgentMicroVM{}
	agente, err := productionAgentMicroVM(
		snapshot,
		rendererFactoriaAgentMicroVM{},
		store,
		autoridadFisicaFactoriaAgentMicroVM(store, registro),
	)
	if err != nil || agente == nil {
		t.Fatalf("productionAgentMicroVM() agente=%v error=%v", agente, err)
	}
	if _, err := os.Lstat(rutaSocket); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("la composición creó o tocó el socket: %v", err)
	}
	if err := agente.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error=%v", err)
	}
	if store.cierres.Load() != 0 {
		t.Fatalf("store compartido cerrado %d veces", store.cierres.Load())
	}
}

func TestCargarDescriptorPerfilAgentMicroVMRechazaFilesystemDigestYJSONInseguros(t *testing.T) {
	_, raw := descriptorFactoriaAgentMicroVM(t)
	digest := digestRawFactoriaAgentMicroVM(raw)
	root := t.TempDir()
	escribir := func(nombre string, contenido []byte, modo os.FileMode) string {
		t.Helper()
		ruta := filepath.Join(root, nombre)
		if err := os.WriteFile(ruta, contenido, modo); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(ruta, modo); err != nil {
			t.Fatal(err)
		}
		return ruta
	}
	valido := escribir("valido.json", raw, 0o600)
	if descriptor, err := cargarDescriptorPerfilAgentMicroVM(valido, digest, os.Geteuid()); err != nil || descriptor.DescriptorRef == "" {
		t.Fatalf("descriptor válido=%+v error=%v", descriptor, err)
	}

	modo := escribir("modo.json", raw, 0o622)
	directorio := filepath.Join(root, "directorio")
	if err := os.Mkdir(directorio, 0o700); err != nil {
		t.Fatal(err)
	}
	invalidoJSON := escribir("invalido.json", []byte(`{"esquema":"desconocido"}`), 0o600)
	grande := escribir("grande.json", []byte(strings.Repeat("x", int(maximoDescriptorPerfilMicroVMBytes+1))), 0o600)
	hardlink := filepath.Join(root, "hardlink.json")
	if err := os.Link(valido, hardlink); err != nil {
		t.Fatal(err)
	}
	symlink := filepath.Join(root, "symlink.json")
	if err := os.Symlink(valido, symlink); err != nil {
		t.Skipf("symlink no soportado: %v", err)
	}
	tests := []struct {
		nombre string
		ruta   string
		digest string
		owner  int
	}{
		{"ruta relativa", "valido.json", digest, os.Geteuid()},
		{"digest distinto", valido, strings.Repeat("0", 64), os.Geteuid()},
		{"digest mayúsculas", valido, strings.ToUpper(digest), os.Geteuid()},
		{"symlink", symlink, digest, os.Geteuid()},
		{"modo escribible", modo, digest, os.Geteuid()},
		{"no regular", directorio, digest, os.Geteuid()},
		{"owner distinto", valido, digest, os.Geteuid() + 1},
		{"hardlink", hardlink, digest, os.Geteuid()},
		{"json inválido", invalidoJSON, digestRawFactoriaAgentMicroVM([]byte(`{"esquema":"desconocido"}`)), os.Geteuid()},
		{"sobredimensionado", grande, digestRawFactoriaAgentMicroVM([]byte(strings.Repeat("x", int(maximoDescriptorPerfilMicroVMBytes+1)))), os.Geteuid()},
	}
	for _, test := range tests {
		t.Run(test.nombre, func(t *testing.T) {
			got, err := cargarDescriptorPerfilAgentMicroVM(test.ruta, test.digest, test.owner)
			if !errors.Is(err, errFactoriaAgentMicroVMDescriptorInvalido) || got.DescriptorRef != "" {
				t.Fatalf("descriptor=%+v error=%v", got, err)
			}
		})
	}
}

func TestProductionAgentMicroVMRechazaDependenciasNulasAntesDeAbrirCliente(t *testing.T) {
	snapshot, _, _ := fixtureFactoriaAgentMicroVM(t)
	store := &storeFactoriaAgentMicroVM{}
	registro := &registroLanzamientosFactoriaAgentMicroVM{}
	var construcciones atomic.Int64
	constructor := func(string) (recursoClienteAgentMicroVM, error) {
		construcciones.Add(1)
		return recursoClienteAgentMicroVM{
			cliente:           &clienteFactoriaAgentMicroVM{},
			liberarConexiones: func() error { return nil },
		}, nil
	}
	sinOneShot := autoridadFisicaFactoriaAgentMicroVM(store, registro)
	sinOneShot.almacenOneShot = nil
	tests := []struct {
		nombre    string
		renderer  codex.PromptRenderer
		store     credentials.Store
		autoridad dependenciasAutoridadFisicaAgentMicroVM
	}{
		{"renderer nil", nil, store, autoridadFisicaFactoriaAgentMicroVM(store, registro)},
		{"renderer tipado nil", (*rendererTipadoNuloFactoriaAgentMicroVM)(nil), store, autoridadFisicaFactoriaAgentMicroVM(store, registro)},
		{"store nil", rendererFactoriaAgentMicroVM{}, nil, autoridadFisicaFactoriaAgentMicroVM(store, registro)},
		{"store tipado nil", rendererFactoriaAgentMicroVM{}, (*storeFactoriaAgentMicroVM)(nil), autoridadFisicaFactoriaAgentMicroVM(store, registro)},
		{"lector nil", rendererFactoriaAgentMicroVM{}, store, autoridadFisicaFactoriaAgentMicroVM(nil, registro)},
		{"lector tipado nil", rendererFactoriaAgentMicroVM{}, store, autoridadFisicaFactoriaAgentMicroVM((*storeFactoriaAgentMicroVM)(nil), registro)},
		{"one shot nil", rendererFactoriaAgentMicroVM{}, store, sinOneShot},
		{"registro nil", rendererFactoriaAgentMicroVM{}, store, autoridadFisicaFactoriaAgentMicroVM(store, nil)},
		{"registro tipado nil", rendererFactoriaAgentMicroVM{}, store, autoridadFisicaFactoriaAgentMicroVM(store, (*registroLanzamientosFactoriaAgentMicroVM)(nil))},
	}
	for _, test := range tests {
		t.Run(test.nombre, func(t *testing.T) {
			antes := construcciones.Load()
			agente, err := productionAgentMicroVMConConstructor(
				snapshot, test.renderer, test.store,
				test.autoridad, constructor,
			)
			if agente != nil || !errors.Is(err, errFactoriaAgentMicroVMAutoridadInvalida) {
				t.Fatalf("agente=%v error=%v", agente, err)
			}
			if construcciones.Load() != antes {
				t.Fatalf("constructor invocado: antes=%d después=%d", antes, construcciones.Load())
			}
			if store.cierres.Load() != 0 {
				t.Fatalf("store compartido cerrado %d veces", store.cierres.Load())
			}
		})
	}
}

func TestProductionAgentMicroVMRechazaSeleccionClienteYRecursoIncompleto(t *testing.T) {
	process, err := config.Resolve(config.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	store := &storeFactoriaAgentMicroVM{}
	registro := &registroLanzamientosFactoriaAgentMicroVM{}
	if agente, err := productionAgentMicroVMConConstructor(
		process, rendererFactoriaAgentMicroVM{}, store,
		autoridadFisicaFactoriaAgentMicroVM(store, registro), nil,
	); agente != nil || !errors.Is(err, errFactoriaAgentMicroVMSeleccionInvalida) {
		t.Fatalf("selección process agente=%v error=%v", agente, err)
	}
	if recurso, err := nuevoRecursoClienteAgentMicroVM("socket-relativo"); err == nil || recurso.cliente != nil {
		t.Fatalf("cliente relativo=%+v error=%v", recurso, err)
	}

	snapshot, _, _ := fixtureFactoriaAgentMicroVM(t)
	var cierres atomic.Int64
	agente, err := productionAgentMicroVMConConstructor(
		snapshot, rendererFactoriaAgentMicroVM{}, store,
		autoridadFisicaFactoriaAgentMicroVM(store, registro),
		func(string) (recursoClienteAgentMicroVM, error) {
			return recursoClienteAgentMicroVM{liberarConexiones: func() error { cierres.Add(1); return nil }}, nil
		},
	)
	if agente != nil || !errors.Is(err, errFactoriaAgentMicroVMClienteInvalido) || cierres.Load() != 1 {
		t.Fatalf("recurso incompleto agente=%v error=%v cierres=%d", agente, err, cierres.Load())
	}
	falloCliente := errors.New("cliente no construible")
	agente, err = productionAgentMicroVMConConstructor(
		snapshot, rendererFactoriaAgentMicroVM{}, store,
		autoridadFisicaFactoriaAgentMicroVM(store, registro),
		func(string) (recursoClienteAgentMicroVM, error) { return recursoClienteAgentMicroVM{}, falloCliente },
	)
	if agente != nil || !errors.Is(err, errFactoriaAgentMicroVMClienteInvalido) || !errors.Is(err, falloCliente) {
		t.Fatalf("fallo cliente agente=%v error=%v", agente, err)
	}

	rutaBroker := snapshot.RuntimeMicroVMCredentialBrokerSocketPath()
	if err := os.WriteFile(rutaBroker, []byte("propiedad-ajena"), 0o600); err != nil {
		t.Fatal(err)
	}
	agente, err = productionAgentMicroVMConConstructor(
		snapshot, rendererFactoriaAgentMicroVM{}, store,
		autoridadFisicaFactoriaAgentMicroVM(store, registro),
		func(string) (recursoClienteAgentMicroVM, error) {
			return recursoClienteAgentMicroVM{
				cliente:           &clienteFactoriaAgentMicroVM{},
				liberarConexiones: func() error { cierres.Add(1); return nil },
			}, nil
		},
	)
	contenido, readErr := os.ReadFile(rutaBroker)
	if agente != nil || agentmicrovm.ErrorCode(err) != agentmicrovm.CodeCredentialBrokerSocketUnsafe ||
		cierres.Load() != 2 || readErr != nil || string(contenido) != "propiedad-ajena" {
		t.Fatalf("broker ocupado agente=%v error=%v cierres=%d contenido=%q read=%v",
			agente, err, cierres.Load(), contenido, readErr)
	}
}

func fixtureFactoriaAgentMicroVM(t *testing.T) (config.Snapshot, string, string) {
	t.Helper()
	_, raw := descriptorFactoriaAgentMicroVM(t)
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	rutaDescriptor := filepath.Join(root, "perfil.json")
	if err := os.WriteFile(rutaDescriptor, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	rutaSocket := filepath.Join(root, "agente-microvm.sock")
	rutaBroker := filepath.Join(root, "b.sock")
	toml := `[runtime]
provider = "codex"
isolation = "microvm"

[runtime.microvm]
placement_ref = "placement:codex:microvm-prueba"
socket_path = "` + rutaSocket + `"
profile_descriptor_path = "` + rutaDescriptor + `"
expected_profile_descriptor_sha256 = "` + digestRawFactoriaAgentMicroVM(raw) + `"
launch_grant_key_id = "clave-publica:orquesta-prueba"
launch_grant_signing_credential_ref = "credential:microvm-launch-signing"
credential_broker_socket_path = "` + rutaBroker + `"
credential_broker_peer_uid = ` + strconv.Itoa(os.Geteuid()) + `
credential_broker_exchange_timeout = "1s"
credential_broker_max_connections = 16

[runtime.codex]
model = "gpt-5.6"
credential_ref = "credential:codex-account-1"
`
	snapshot, err := config.Resolve(config.ResolveOptions{TOML: []byte(toml)})
	if err != nil {
		t.Fatalf("config.Resolve() error=%v", err)
	}
	return snapshot, rutaDescriptor, rutaSocket
}

func descriptorFactoriaAgentMicroVM(t *testing.T) (microvm.DescriptorPerfilLanzamientoV1, []byte) {
	t.Helper()
	perfilSHA := strings.Repeat("3", 64)
	ejecutorRef, err := microvm.ConstruirEjecutorRefPerfilV1(perfilSHA)
	if err != nil {
		t.Fatal(err)
	}
	descriptor := microvm.DescriptorPerfilLanzamientoV1{
		Esquema:         microvm.EsquemaDescriptorPerfilLanzamientoV1,
		EjecutorRef:     ejecutorRef,
		VCPU:            2,
		MemoriaMiB:      512,
		KernelSHA256:    strings.Repeat("1", 64),
		InitramfsSHA256: strings.Repeat("2", 64),
		PerfilSHA256:    perfilSHA,
		ServiciosDisponibles: []microvm.ServicioVsock{{
			Papel: "control_broker", ServicioRef: "servicio:control", Puerto: 10_001,
			IdentidadRef: "identidad-servicio:control", IdentidadSHA256: strings.Repeat("4", 64),
		}, {
			Papel: "controlled_egress_proxy", ServicioRef: "servicio:proxy", Puerto: 10_002,
			IdentidadRef: "identidad-servicio:proxy", IdentidadSHA256: strings.Repeat("5", 64),
		}},
	}
	digest, err := microvm.CalcularSHA256DescriptorPerfilLanzamientoV1(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	descriptor.DescriptorRef = "perfil-lanzamiento:sha256:" + digest
	raw, err := json.Marshal(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	return descriptor, raw
}

func solicitudLanzamientoFactoriaAgentMicroVM(t *testing.T) ports.AgentLaunchRequest {
	t.Helper()
	executionRef, _ := goal.NewExecutionRef("execution:microvm-factoria-1")
	goalRef, _ := goal.NewGoalRef("goal:microvm-factoria-1")
	workItemRef, _ := goal.NewWorkItemRef("work:microvm-factoria-1")
	actorRef, _ := goal.NewActorRef("actor:owner")
	projectRef, _ := goal.NewProjectRef("project:microvm")
	placementRef, _ := ports.NewAgentPlacementRef("placement:codex:microvm-prueba")
	sessionRef, _ := ports.NewExecutionSessionRef("execution-session:microvm-factoria-1")
	artifactRef, _ := ports.NewExecutionArtifactAccessRef("artifact-access:execution:sha256:" + strings.Repeat("a", 64))
	mcpRef, _ := ports.NewExecutionMCPAccessRef("mcp-access:execution:sha256:" + strings.Repeat("b", 64))
	mailboxRef, _ := ports.NewExecutionMailboxEndpointRef("mailbox-endpoint:execution:sha256:" + strings.Repeat("c", 64))
	request := ports.AgentLaunchRequest{
		ExecutionRef: executionRef, ReferenciaColocacion: placementRef,
		SessionRef: sessionRef,
		AccessAuthority: ports.AgentLaunchAccessAuthority{
			ArtifactAccessRef: artifactRef, MCPAccessRef: mcpRef, MailboxEndpointRef: mailboxRef,
		},
		GoalRef: goalRef, WorkItemRef: workItemRef,
		PlanGeneration: 2, AppSpecGeneration: 3, ExecutionAttempt: 1,
		SpecHash: strings.Repeat("d", 64), ActorRef: actorRef, ProjectRef: projectRef,
		Objective: "ejecutar trabajo exacto", PhaseRef: "phase-instance:build", PhaseKey: "phase:build",
		PhaseTemplateRef: "phase-template:program", PhaseInputRefs: []string{"input:spec"},
		PhaseCriterionRefs: []string{"criterion:green"}, RoleKey: "role:worker",
		SkillRefs: []string{"skill:go"}, ToolRefs: []string{"tool:test"},
		CapabilityRefs: []string{"capability:code"}, WriteSet: []string{"internal/bootstrap"},
		OutputContract: string(goal.OutputContractEvidenceBundle), ArtifactMediaType: "application/json",
		IdempotencyKey: "launch:microvm-factoria-1", MaxOutputBytes: 1 << 20,
		BudgetDemand: governance.BudgetDemand{
			Ref: "budget-demand:microvm-factoria-1",
			Resources: governance.ResourceVector{
				Tokens: 4_096, ActiveTimeNS: int64(90 * time.Second), ProcessSlots: 1, DiskBytes: 8_192,
			},
		},
		SecurityCriticality:         governance.SecurityCriticalityNormal,
		ReasoningEffort:             governance.ReasoningEffortMedium,
		RequierePreservacionEntorno: true,
		EffectAuthority: ports.AgentLaunchEffectAuthority{
			AuthorizationReceiptRef: "authorization:microvm-factoria-1",
			EffectApprovalRef:       "effect-approval:microvm-factoria-1",
			EffectAttemptRef:        "effect-attempt:microvm-factoria-1",
			ActionFence:             7, StartedAt: time.Unix(10, 0).UTC(),
			ClaimLeaseUntil: time.Unix(30, 0).UTC(), ApprovalExpiresAt: time.Unix(20, 0).UTC(),
		},
	}
	egress := microvm.ConcesionEgreso{
		Esquema: microvm.EsquemaConcesionEgreso, Referencia: "egreso:codex",
		Destinos:         []microvm.DestinoEgreso{{Host: "api.openai.com", Puertos: []uint16{443}}},
		MaximoConexiones: 4, LimiteTiempoMS: 60_000,
		LimiteSubidaBytes: 1 << 20, LimiteBajadaBytes: 8 << 20,
	}
	rawEgress, err := json.Marshal(egress)
	if err != nil {
		t.Fatal(err)
	}
	digestEgress := sha256.Sum256(rawEgress)
	request.EgressAuthority = ports.AgentLaunchEgressAuthority{
		PolicyRef: egress.Referencia, PayloadSHA256: hex.EncodeToString(digestEgress[:]),
		CanonicalPayload: rawEgress,
	}
	return request
}

type contenidoConcesionFactoriaAgentMicroVM struct {
	Esquema       string `json:"esquema"`
	Audiencia     string `json:"audiencia"`
	ConcesionRef  string `json:"concesion_ref"`
	RunRef        string `json:"run_ref"`
	Cerca         uint64 `json:"cerca"`
	PlanSHA256    string `json:"plan_sha256"`
	EmitidaUnixMS int64  `json:"emitida_unix_ms"`
	NoAntesUnixMS int64  `json:"no_antes_unix_ms"`
	ExpiraUnixMS  int64  `json:"expira_unix_ms"`
	ClaveID       string `json:"clave_id"`
	Algoritmo     string `json:"algoritmo"`
}

func verificarConcesionFactoriaAgentMicroVM(
	t *testing.T,
	solicitud microvm.SolicitudLanzamiento,
	publica ed25519.PublicKey,
) contenidoConcesionFactoriaAgentMicroVM {
	t.Helper()
	var concesion struct {
		Contenido contenidoConcesionFactoriaAgentMicroVM `json:"contenido"`
		Firma     string                                 `json:"firma_base64"`
	}
	if err := json.Unmarshal(solicitud.Concesion, &concesion); err != nil {
		t.Fatalf("concesión inválida: %v", err)
	}
	firma, err := base64.StdEncoding.Strict().DecodeString(concesion.Firma)
	if err != nil || !ed25519.Verify(publica, mensajeConcesionFactoriaAgentMicroVM(concesion.Contenido), firma) {
		t.Fatalf("firma de concesión no verificable: %v", err)
	}
	return concesion.Contenido
}

func mensajeConcesionFactoriaAgentMicroVM(contenido contenidoConcesionFactoriaAgentMicroVM) []byte {
	mensaje := append([]byte("agentmicrovm.concesion-lanzamiento.v1\x00"), campoConcesionFactoriaAgentMicroVM(contenido.Esquema)...)
	mensaje = append(mensaje, campoConcesionFactoriaAgentMicroVM(contenido.Audiencia)...)
	mensaje = append(mensaje, campoConcesionFactoriaAgentMicroVM(contenido.ConcesionRef)...)
	mensaje = append(mensaje, campoConcesionFactoriaAgentMicroVM(contenido.RunRef)...)
	mensaje = binary.BigEndian.AppendUint64(mensaje, contenido.Cerca)
	mensaje = append(mensaje, campoConcesionFactoriaAgentMicroVM(contenido.PlanSHA256)...)
	mensaje = binary.BigEndian.AppendUint64(mensaje, uint64(contenido.EmitidaUnixMS))
	mensaje = binary.BigEndian.AppendUint64(mensaje, uint64(contenido.NoAntesUnixMS))
	mensaje = binary.BigEndian.AppendUint64(mensaje, uint64(contenido.ExpiraUnixMS))
	mensaje = append(mensaje, campoConcesionFactoriaAgentMicroVM(contenido.ClaveID)...)
	return append(mensaje, campoConcesionFactoriaAgentMicroVM(contenido.Algoritmo)...)
}

func campoConcesionFactoriaAgentMicroVM(valor string) []byte {
	campo := binary.BigEndian.AppendUint32(nil, uint32(len(valor)))
	return append(campo, valor...)
}

func digestRawFactoriaAgentMicroVM(raw []byte) string {
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

var _ agentmicrovm.Client = (*clienteFactoriaAgentMicroVM)(nil)
var _ credentials.Store = (*storeFactoriaAgentMicroVM)(nil)
