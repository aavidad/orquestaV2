package agentmicrovm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/ports"
)

const (
	DockerProviderRef              = "provider:docker"
	CodeDockerAuthorityUnsupported = "agentmicrovm.docker_authority_unsupported"
	CodeDockerJournalFailed        = "agentmicrovm.docker_journal_failed"
	CodeDockerRecoveryInvalid      = "agentmicrovm.docker_recovery_invalid"
	CodeDockerRecoveryPending      = "agentmicrovm.docker_recovery_pending"
	CodeDockerResponseInvalid      = "agentmicrovm.docker_response_invalid"
	dockerReceiptDomain            = "orquesta.agentmicrovm.docker-launch-receipt.v1\x00"
	dockerProviderRequestKeyDomain = "orquesta.agentmicrovm.docker-provider-request.v1\x00"
)

type DockerClient interface {
	NegociarDocker(context.Context) (microvm.RespuestaCapacidades, error)
	LanzarORecuperarContenedorCodificada(context.Context, string, json.RawMessage) (microvm.RespuestaContenedorV1, error)
	ConsultarOperacionContenedor(context.Context, string) (microvm.RespuestaOperacionContenedorV1, error)
	IniciarSesionContenedorCodificada(context.Context, string, string, json.RawMessage) (microvm.RespuestaInicioSesionContenedorV1, error)
	EnviarEntradaSesionContenedorCodificada(context.Context, string, string, string, json.RawMessage) (microvm.RespuestaEntradaSesionContenedorV1, error)
	DetenerContenedorCodificada(context.Context, string, string, json.RawMessage) (microvm.RespuestaContenedorV1, error)
}

type DockerSigner interface {
	PrepararContenedor(
		context.Context,
		ports.AgentLaunchRequest,
		DockerPhysicalBinding,
		DockerCompilation,
	) (microvm.SolicitudLanzarORecuperarContenedorV1, error)
}

var _ DockerClient = (*microvm.Cliente)(nil)
var _ DockerSigner = (*CredentialSigner)(nil)
var _ dockerObservationClient = (*microvm.Cliente)(nil)

type DockerConfig struct {
	Client         DockerClient
	Signer         DockerSigner
	Journal        ports.AgentProviderRequestJournal
	StopJournal    ports.AgentProviderStopRequestJournal
	Capabilities   ports.AgentCapabilities
	PlacementRef   ports.AgentPlacementRef
	ProviderModel  string
	PromptRenderer PromptRenderer
	ImageRef       string
	ExecutorRef    string
	VCPU           uint8
	MemoryMiB      uint32
	MaxPIDs        uint32
	Now            func() time.Time
}

// DockerAdapter is opt-in composition behind AgentLauncher. It only talks to
// the published sibling client and never owns Docker transport or lifecycle.
type DockerAdapter struct {
	client           DockerClient
	observer         dockerObservationClient
	signer           DockerSigner
	journal          ports.AgentProviderRequestJournal
	stopJournal      ports.AgentProviderStopRequestJournal
	capabilities     ports.AgentCapabilities
	placement        ports.AgentPlacementRef
	model            string
	renderer         PromptRenderer
	imageRef         string
	executorRef      string
	vcpu             uint8
	memoryMiB        uint32
	maxPIDs          uint32
	now              func() time.Time
	physicalCapacity physicalCapacityProjection
}

func NewDockerAdapter(config DockerConfig) (*DockerAdapter, error) {
	if nilInterface(config.Client) || nilInterface(config.Signer) || nilInterface(config.Journal) ||
		nilInterface(config.StopJournal) || nilInterface(config.Now) ||
		nilInterface(config.PromptRenderer) || config.PlacementRef.String() == "" ||
		!validWorkPacketToken(config.ProviderModel, 128) ||
		ports.ValidateAgentCapabilities(config.Capabilities) != nil ||
		config.Capabilities.ProviderRef != DockerProviderRef ||
		config.Capabilities.RequierePreservacionEntorno || !validDockerPhysicalConfig(config) {
		return nil, fail(CodeConfigurationInvalid, nil)
	}
	observer, ok := config.Client.(dockerObservationClient)
	if !ok || nilInterface(observer) {
		return nil, fail(CodeObservationClientInvalid, nil)
	}
	return &DockerAdapter{
		client: config.Client, observer: observer, signer: config.Signer, journal: config.Journal,
		stopJournal:  config.StopJournal,
		capabilities: cloneCapabilities(config.Capabilities), placement: config.PlacementRef,
		model: config.ProviderModel, renderer: config.PromptRenderer,
		imageRef: config.ImageRef, executorRef: config.ExecutorRef,
		vcpu: config.VCPU, memoryMiB: config.MemoryMiB, maxPIDs: config.MaxPIDs, now: config.Now,
		physicalCapacity: newPhysicalCapacityProjection(),
	}, nil
}

func (adapter *DockerAdapter) Capabilities(ctx context.Context) (ports.AgentCapabilities, error) {
	if adapter == nil || nilInterface(adapter.client) || nilInterface(ctx) {
		return ports.AgentCapabilities{}, fail(CodeConfigurationInvalid, nil)
	}
	if err := adapter.negotiateDocker(ctx); err != nil {
		return ports.AgentCapabilities{}, err
	}
	return cloneCapabilities(adapter.capabilities), nil
}

func (adapter *DockerAdapter) NegotiatedPhysicalCapacity() (NegotiatedPhysicalCapacity, error) {
	if adapter == nil {
		return NegotiatedPhysicalCapacity{}, fail(CodeNegotiatedPhysicalCapacityUnavailable, nil)
	}
	return adapter.physicalCapacity.read()
}

func (adapter *DockerAdapter) Launch(
	ctx context.Context,
	request ports.AgentLaunchRequest,
) (ports.AgentLaunchReceipt, error) {
	return adapter.launchDocker(ctx, request, false)
}

// ReconcileLaunch never invokes the launch mutation. It resolves the exact
// journaled bytes and queries their operation key before completing sessions.
func (adapter *DockerAdapter) ReconcileLaunch(
	ctx context.Context,
	request ports.AgentLaunchRequest,
) (ports.AgentLaunchReceipt, error) {
	return adapter.launchDocker(ctx, request, true)
}

func (adapter *DockerAdapter) launchDocker(
	ctx context.Context,
	request ports.AgentLaunchRequest,
	reconcile bool,
) (ports.AgentLaunchReceipt, error) {
	if err := adapter.validateDockerRequest(ctx, request); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	packet, packetJSON, err := buildProviderWorkPacketV1(request, adapter.model, adapter.renderer)
	if err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	binding := adapter.dockerPhysicalBinding()
	compiled, err := CompileDockerLaunch(request, binding)
	if err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	plan, issuedAt := compiled.Plan, compiled.IssuedAt
	if err := adapter.negotiateDocker(ctx); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}

	var body []byte
	var response microvm.RespuestaContenedorV1
	if reconcile {
		stored, signed, concessionRef, resolveErr := adapter.resolveDockerLaunch(ctx, request, plan)
		if resolveErr != nil {
			return ports.AgentLaunchReceipt{}, resolveErr
		}
		body = stored.Body
		operation, callErr := adapter.client.ConsultarOperacionContenedor(ctx, stored.IdempotencyKey)
		if callErr != nil {
			if contextErr := preserveContextError(ctx, callErr); contextErr != nil {
				return ports.AgentLaunchReceipt{}, contextErr
			}
			return ports.AgentLaunchReceipt{}, fail(CodeDockerRecoveryPending, callErr)
		}
		planSHA, digestErr := microvm.CalcularSHA256PlanLanzamientoContenedorV1(signed.Plan)
		if digestErr != nil || operation.OperacionRef != stored.IdempotencyKey ||
			operation.PlanSHA256 != planSHA || operation.ConcesionRef != concessionRef ||
			!validDockerResponse(plan, operation.Contenedor, false) {
			return ports.AgentLaunchReceipt{}, fail(CodeDockerResponseInvalid, digestErr)
		}
		response = operation.Contenedor
	} else {
		signed, signingErr := adapter.signer.PrepararContenedor(ctx, request, binding, compiled)
		if signingErr != nil {
			if contextErr := preserveContextError(ctx, signingErr); contextErr != nil {
				return ports.AgentLaunchReceipt{}, contextErr
			}
			return ports.AgentLaunchReceipt{}, fail(CodeSigningFailed, signingErr)
		}
		if !reflect.DeepEqual(signed.Plan, plan) {
			return ports.AgentLaunchReceipt{}, fail(CodeSigningFailed, nil)
		}
		body, err = microvm.CodificarSolicitudLanzarORecuperarContenedorV1(signed)
		if err != nil {
			return ports.AgentLaunchReceipt{}, fail(CodeSigningFailed, err)
		}
		stored, recordErr := adapter.recordDockerStage(ctx, request, ports.AgentProviderRequestLaunch, "", 0, body)
		if recordErr != nil {
			return ports.AgentLaunchReceipt{}, recordErr
		}
		body = stored.Body
		if err := ctx.Err(); err != nil {
			return ports.AgentLaunchReceipt{}, fail(CodeLaunchCanceledBeforeSubmit, err)
		}
		response, err = adapter.client.LanzarORecuperarContenedorCodificada(ctx, stored.IdempotencyKey, stored.Body)
		if err != nil {
			if contextErr := preserveContextError(ctx, err); contextErr != nil {
				return ports.AgentLaunchReceipt{}, contextErr
			}
			return ports.AgentLaunchReceipt{}, classifyLaunchError(err)
		}
		if !validDockerResponse(plan, response, true) {
			return ports.AgentLaunchReceipt{}, fail(CodeDockerResponseInvalid, nil)
		}
	}

	launchKey := dockerJournalKey(request, ports.AgentProviderRequestLaunch)
	bound, err := adapter.journal.BindAgentProviderLaunch(
		context.WithoutCancel(ctx), launchKey, response.Referencia, response.Revision,
	)
	if err != nil || bound.LaunchBindingRef != response.Referencia ||
		bound.LaunchBindingRevision != response.Revision || !bytes.Equal(bound.Body, body) {
		return ports.AgentLaunchReceipt{}, fail(CodeDockerJournalFailed, err)
	}
	if err := adapter.deliverDockerSession(ctx, request, bound, packet.TimeBudgetMS, packetJSON); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	receipt := ports.AgentLaunchReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, LaunchActionFence: request.EffectAuthority.ActionFence,
		SpecHash:    request.SpecHash,
		ProviderRef: adapter.capabilities.ProviderRef, ModelRef: adapter.capabilities.ModelRef,
		AgentRef: adapter.capabilities.AgentRef, ExternalRef: bound.LaunchBindingRef,
		IdempotencyKey: request.IdempotencyKey,
		ReceiptRef:     "agentmicrovm-docker-launch:sha256:" + dockerDigest(dockerReceiptDomain, body),
		AcceptedAt:     issuedAt, RequierePreservacionEntorno: request.RequierePreservacionEntorno,
	}
	if err := ports.ValidateAgentLaunchReceipt(request, receipt); err != nil {
		return ports.AgentLaunchReceipt{}, fail(CodeDockerResponseInvalid, err)
	}
	return receipt, nil
}

func (adapter *DockerAdapter) validateDockerRequest(ctx context.Context, request ports.AgentLaunchRequest) error {
	if adapter == nil || nilInterface(adapter.client) || nilInterface(adapter.signer) ||
		nilInterface(adapter.journal) || nilInterface(adapter.renderer) || nilInterface(ctx) {
		return fail(CodeConfigurationInvalid, nil)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ports.ValidateAgentLaunchRequest(request); err != nil {
		return fail(CodeLaunchRequestInvalid, err)
	}
	if request.SessionRef.String() == "" {
		return fail(CodeSessionRequired, nil)
	}
	access := request.AccessAuthority
	if !request.EgressAuthority.IsEmpty() || request.ExecutionWorkspaceRef.String() != "" ||
		access.ArtifactAccessRef.String() != "" || access.MCPAccessRef.String() != "" ||
		access.MailboxEndpointRef.String() != "" {
		return fail(CodeDockerAuthorityUnsupported, nil)
	}
	requirements := ports.AgentRequirements{RoleKey: request.RoleKey, SkillRefs: request.SkillRefs,
		ToolRefs: request.ToolRefs, CapabilityRefs: request.CapabilityRefs}
	if request.ReferenciaColocacion != adapter.placement || request.RequierePreservacionEntorno ||
		!ports.MatchAgentCapabilities(adapter.capabilities, requirements) {
		return fail(CodeCapabilityMismatch, nil)
	}
	return nil
}

func (adapter *DockerAdapter) negotiateDocker(ctx context.Context) error {
	generation, release, beginErr := adapter.physicalCapacity.begin(ctx)
	if beginErr != nil {
		return beginErr
	}
	defer release()
	response, err := adapter.client.NegociarDocker(ctx)
	if err != nil {
		if contextErr := preserveContextError(ctx, err); contextErr != nil {
			return contextErr
		}
		return classifyCapabilitiesError(err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if response.Protocolo != microvm.ProtocoloLocal || response.BackendEjecuciones != microvm.BackendEjecucionesDocker ||
		!response.DockerConfigurado || !response.DockerEngineDisponible || response.MaximoEjecuciones == 0 {
		return fail(CodeCapabilitiesRejected, nil)
	}
	adapter.physicalCapacity.publish(generation, NegotiatedPhysicalCapacity{
		PlacementRef: adapter.placement,
		Slots:        response.MaximoEjecuciones,
	})
	return nil
}

func (adapter *DockerAdapter) dockerPhysicalBinding() DockerPhysicalBinding {
	return DockerPhysicalBinding{
		PlacementRef: adapter.placement, ImageRef: adapter.imageRef,
		VCPU: adapter.vcpu, MemoryMiB: adapter.memoryMiB, MaxPIDs: adapter.maxPIDs,
	}
}

func (adapter *DockerAdapter) resolveDockerLaunch(
	ctx context.Context,
	request ports.AgentLaunchRequest,
	plan microvm.PlanLanzamientoContenedorV1,
) (ports.AgentProviderRequest, microvm.SolicitudLanzarORecuperarContenedorV1, string, error) {
	key := dockerJournalKey(request, ports.AgentProviderRequestLaunch)
	stored, found, err := adapter.journal.ResolveAgentProviderRequest(ctx, key)
	if err != nil {
		return ports.AgentProviderRequest{}, microvm.SolicitudLanzarORecuperarContenedorV1{}, "", fail(CodeDockerRecoveryPending, err)
	}
	if !found || stored.Key != key || ports.ValidateAgentProviderRequest(stored) != nil ||
		stored.ProviderRef != DockerProviderRef ||
		stored.EffectAttemptRef != request.EffectAuthority.EffectAttemptRef ||
		stored.IdempotencyKey != dockerLaunchOperationRef(request) {
		return ports.AgentProviderRequest{}, microvm.SolicitudLanzarORecuperarContenedorV1{}, "", fail(CodeDockerRecoveryInvalid, nil)
	}
	var signed microvm.SolicitudLanzarORecuperarContenedorV1
	if json.Unmarshal(stored.Body, &signed) != nil || !reflect.DeepEqual(signed.Plan, plan) {
		return ports.AgentProviderRequest{}, microvm.SolicitudLanzarORecuperarContenedorV1{}, "", fail(CodeDockerRecoveryInvalid, nil)
	}
	canonical, err := microvm.CodificarSolicitudLanzarORecuperarContenedorV1(signed)
	if err != nil || !bytes.Equal(canonical, stored.Body) {
		return ports.AgentProviderRequest{}, microvm.SolicitudLanzarORecuperarContenedorV1{}, "", fail(CodeDockerRecoveryInvalid, err)
	}
	var grant struct {
		Contenido struct {
			ConcesionRef string `json:"concesion_ref"`
		} `json:"contenido"`
	}
	if json.Unmarshal(signed.Concesion, &grant) != nil || grant.Contenido.ConcesionRef == "" {
		return ports.AgentProviderRequest{}, microvm.SolicitudLanzarORecuperarContenedorV1{}, "", fail(CodeDockerRecoveryInvalid, nil)
	}
	return stored, signed, grant.Contenido.ConcesionRef, nil
}

func (adapter *DockerAdapter) recordDockerStage(
	ctx context.Context,
	request ports.AgentLaunchRequest,
	stage ports.AgentProviderRequestStage,
	target string,
	revision uint64,
	body []byte,
) (ports.AgentProviderRequest, error) {
	prepared := ports.AgentProviderRequest{
		Key: dockerJournalKey(request, stage), EffectAttemptRef: request.EffectAuthority.EffectAttemptRef,
		ProviderRef: DockerProviderRef, IdempotencyKey: dockerStageKey(request, stage),
		TargetRef: target, ExpectedRevision: revision, Body: append([]byte(nil), body...),
		BodySHA256: ports.AgentProviderRequestBodySHA256(body),
	}
	stored, err := adapter.journal.RecordAgentProviderRequest(ctx, prepared)
	storedPrepared := stored
	if stage == ports.AgentProviderRequestLaunch {
		storedPrepared.LaunchBindingRef, storedPrepared.LaunchBindingRevision = "", 0
	}
	if err != nil || ports.ValidateAgentProviderRequest(stored) != nil ||
		!reflect.DeepEqual(storedPrepared, prepared) {
		return ports.AgentProviderRequest{}, fail(CodeDockerJournalFailed, err)
	}
	return stored, nil
}

func (adapter *DockerAdapter) deliverDockerSession(
	ctx context.Context,
	request ports.AgentLaunchRequest,
	launch ports.AgentProviderRequest,
	timeBudgetMS uint64,
	packet []byte,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	startBody, err := microvm.CodificarSolicitudInicioSesionContenedorV1(microvm.SolicitudInicioSesionContenedorV1{
		SesionRef: request.SessionRef.String(), RevisionEsperada: launch.LaunchBindingRevision,
		Cerca: request.EffectAuthority.ActionFence, EjecutorRef: adapter.executorRef,
		DirectorioTrabajo: ".", PlazoTotalMilisegundos: timeBudgetMS,
		MaximoEventosBytes: microvm.MaximoEventosSesionBytesV1,
	})
	if err != nil {
		return fail(CodeSessionResponseInvalid, err)
	}
	start, err := adapter.recordDockerStage(ctx, request, ports.AgentProviderRequestSessionStart,
		launch.LaunchBindingRef, launch.LaunchBindingRevision, startBody)
	if err != nil {
		return err
	}
	started, err := adapter.client.IniciarSesionContenedorCodificada(
		ctx, start.IdempotencyKey, launch.LaunchBindingRef, start.Body,
	)
	if err != nil {
		if contextErr := preserveContextError(ctx, err); contextErr != nil {
			return contextErr
		}
		return classifySessionMutationError(err, CodeSessionStartUnavailable, CodeSessionStartRejected, CodeSessionPending)
	}
	if started.Referencia != launch.LaunchBindingRef || started.SesionRef != request.SessionRef.String() ||
		started.Cerca != request.EffectAuthority.ActionFence || !started.Iniciada || started.CodigoError != nil {
		return fail(CodeSessionResponseInvalid, nil)
	}
	inputBody, err := microvm.CodificarSolicitudEntradaSesionContenedorV1(microvm.SolicitudEntradaSesionContenedorV1{
		RevisionEsperada: launch.LaunchBindingRevision, Cerca: request.EffectAuthority.ActionFence,
		ContenidoBase64: base64.StdEncoding.EncodeToString(packet), CerrarStdin: true,
	})
	if err != nil {
		return fail(CodeSessionResponseInvalid, err)
	}
	input, err := adapter.recordDockerStage(ctx, request, ports.AgentProviderRequestSessionInput,
		launch.LaunchBindingRef, launch.LaunchBindingRevision, inputBody)
	if err != nil {
		return err
	}
	accepted, err := adapter.client.EnviarEntradaSesionContenedorCodificada(
		ctx, input.IdempotencyKey, launch.LaunchBindingRef, request.SessionRef.String(), input.Body,
	)
	if err != nil {
		if contextErr := preserveContextError(ctx, err); contextErr != nil {
			return contextErr
		}
		return classifySessionMutationError(err, CodeSessionInputUnavailable, CodeSessionInputRejected, CodeSessionInputPending)
	}
	if accepted.Referencia != launch.LaunchBindingRef || accepted.SesionRef != request.SessionRef.String() ||
		accepted.EntradaRef != input.IdempotencyKey || accepted.Cerca != request.EffectAuthority.ActionFence ||
		!accepted.Aceptada || accepted.CodigoError != nil {
		return fail(CodeSessionResponseInvalid, nil)
	}
	return nil
}

func validDockerPhysicalConfig(config DockerConfig) bool {
	plan := microvm.PlanLanzamientoContenedorV1{
		Esquema: microvm.EsquemaPlanLanzamientoContenedorV1, OperacionRef: "docker-config",
		EjecucionRef: "ejecucion:config", RunRef: "execution:config", Cerca: 1,
		EspecificacionRef: "app-spec:config", ImagenRef: config.ImageRef,
		VCPU: config.VCPU, MemoriaMiB: config.MemoryMiB, MaximoPIDs: config.MaxPIDs,
		LimiteTiempoMS: 1, LimiteDiscoPicoBytes: 1, LimiteTokensAgente: 1,
	}
	if _, err := microvm.CodificarPlanLanzamientoContenedorV1(plan); err != nil {
		return false
	}
	_, err := microvm.CodificarSolicitudInicioSesionContenedorV1(microvm.SolicitudInicioSesionContenedorV1{
		SesionRef: "session:config", RevisionEsperada: 1, Cerca: 1, EjecutorRef: config.ExecutorRef,
		DirectorioTrabajo: ".", PlazoTotalMilisegundos: 1, MaximoEventosBytes: 1,
	})
	return err == nil
}

func validDockerResponse(
	plan microvm.PlanLanzamientoContenedorV1,
	response microvm.RespuestaContenedorV1,
	withLiveness bool,
) bool {
	if response.Referencia != plan.EjecucionRef || response.Estado != "activo" || response.Revision == 0 ||
		response.Revision > maxDurableCounter || response.Cerca != plan.Cerca ||
		response.VCPU != plan.VCPU || response.MemoriaMiB != plan.MemoriaMiB || response.Identidad == nil ||
		response.Identidad.ExecutionLabel != response.Referencia || response.Identidad.RunLabel != plan.RunRef ||
		response.Identidad.Generation != plan.Cerca || response.Identidad.PIDObservado == 0 {
		return false
	}
	if !withLiveness {
		return response.RecursoVivo == nil && response.EstadoMotor == nil
	}
	return response.RecursoVivo != nil && *response.RecursoVivo &&
		response.EstadoMotor != nil && *response.EstadoMotor == "running"
}

func dockerJournalKey(request ports.AgentLaunchRequest, stage ports.AgentProviderRequestStage) ports.AgentProviderRequestKey {
	return ports.AgentProviderRequestKey{ExecutionRef: request.ExecutionRef,
		ActionFence: request.EffectAuthority.ActionFence, Stage: stage}
}

func dockerStageKey(request ports.AgentLaunchRequest, stage ports.AgentProviderRequestStage) string {
	if stage == ports.AgentProviderRequestLaunch {
		return dockerLaunchOperationRef(request)
	}
	digest := sha256.New()
	_, _ = digest.Write([]byte(dockerProviderRequestKeyDomain))
	appendDigestField(digest, []byte(request.ExecutionRef.String()))
	appendDigestField(digest, []byte(request.EffectAuthority.EffectAttemptRef))
	appendDigestField(digest, []byte(request.IdempotencyKey))
	appendDigestField(digest, []byte(stage))
	_, _ = digest.Write(binary.BigEndian.AppendUint64(nil, request.EffectAuthority.ActionFence))
	return "orquesta-docker-" + string(stage) + ":sha256:" + hex.EncodeToString(digest.Sum(nil))
}

func dockerDigest(domain string, value []byte) string {
	digest := sha256.New()
	_, _ = digest.Write([]byte(domain))
	appendDigestField(digest, value)
	return hex.EncodeToString(digest.Sum(nil))
}
