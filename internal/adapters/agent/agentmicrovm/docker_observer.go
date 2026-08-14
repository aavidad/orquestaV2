package agentmicrovm

import (
	"context"
	"encoding/base64"
	"errors"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/ports"
)

const maxDockerObservationPages = 4096

// dockerObservationClient is read-only: ObserveAgent cannot reach a mutation.
type dockerObservationClient interface {
	ObservarContenedor(context.Context, string) (microvm.RespuestaContenedorV1, error)
	LeerEventosSesionContenedor(context.Context, string, string, microvm.ConsultaEventosSesionContenedorV1) (microvm.RespuestaEventosSesionContenedorV1, error)
}

// ObserveAgent rebuilds from cursor zero; physical liveness is never terminal work.
func (adapter *DockerAdapter) ObserveAgent(ctx context.Context, request ports.AgentObserveRequest) (ports.AgentObservation, error) {
	if nilInterface(ctx) || adapter == nil || nilInterface(adapter.observer) {
		return ports.AgentObservation{}, fail(CodeConfigurationInvalid, nil)
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentObservation{}, err
	}
	if err := adapter.validateDockerObserveRequest(request); err != nil {
		return ports.AgentObservation{}, err
	}
	launch, err := adapter.resolveDockerObservationLaunch(ctx, request)
	if err != nil {
		return ports.AgentObservation{}, err
	}
	physical, err := adapter.observer.ObservarContenedor(ctx, request.ExternalRef)
	if err != nil {
		if contextErr := preserveContextError(ctx, err); contextErr != nil {
			return ports.AgentObservation{}, contextErr
		}
		return ports.AgentObservation{}, classifyDockerObservationError(err, false)
	}
	if !adapter.validDockerObservedContainer(request, launch, physical) {
		return ports.AgentObservation{}, fail(CodeObservationResponseInvalid, nil)
	}
	return adapter.observeDockerSession(ctx, request, physical)
}

func (adapter *DockerAdapter) resolveDockerObservationLaunch(ctx context.Context, request ports.AgentObserveRequest) (ports.AgentProviderRequest, error) {
	key := ports.AgentProviderRequestKey{ExecutionRef: request.ExecutionRef, ActionFence: request.LaunchActionFence, Stage: ports.AgentProviderRequestLaunch}
	stored, found, err := adapter.journal.ResolveAgentProviderRequest(ctx, key)
	if contextErr := preserveContextError(ctx, err); contextErr != nil {
		return ports.AgentProviderRequest{}, contextErr
	}
	if err != nil {
		return ports.AgentProviderRequest{}, fail(CodeObservationUnavailable, err)
	}
	if !found || stored.Key != key || ports.ValidateAgentProviderRequest(stored) != nil ||
		stored.ProviderRef != DockerProviderRef || stored.LaunchBindingRef != request.ExternalRef ||
		stored.LaunchBindingRevision == 0 {
		return ports.AgentProviderRequest{}, fail(CodeObservationResponseInvalid, nil)
	}
	return stored, nil
}

func (adapter *DockerAdapter) validateDockerObserveRequest(request ports.AgentObserveRequest) error {
	if err := ports.ValidateAgentObserveRequest(request); err != nil ||
		request.SessionRef.String() == "" || !validPhysicalExecutionRef(request.ExternalRef) ||
		request.ProviderRef != adapter.capabilities.ProviderRef ||
		request.ModelRef != adapter.capabilities.ModelRef ||
		request.AgentRef != adapter.capabilities.AgentRef {
		return fail(CodeObservationRequestInvalid, err)
	}
	return nil
}

func (adapter *DockerAdapter) validDockerObservedContainer(request ports.AgentObserveRequest, launch ports.AgentProviderRequest, physical microvm.RespuestaContenedorV1) bool {
	return physical.Referencia == launch.LaunchBindingRef && physical.Revision == launch.LaunchBindingRevision &&
		physical.Cerca == request.LaunchActionFence && physical.VCPU == adapter.vcpu &&
		physical.MemoriaMiB == adapter.memoryMiB && validDockerObservedIdentity(request, physical) &&
		validDockerObservedLiveness(physical)
}

func validDockerObservedIdentity(request ports.AgentObserveRequest, physical microvm.RespuestaContenedorV1) bool {
	identity := physical.Identidad
	return identity == nil || identity.PIDObservado != 0 && identity.ExecutionLabel == physical.Referencia &&
		identity.Generation == request.LaunchActionFence && identity.RunLabel == request.ExecutionRef.String()
}

func validDockerObservedLiveness(physical microvm.RespuestaContenedorV1) bool {
	if physical.RecursoVivo == nil || physical.EstadoMotor == nil {
		return false
	}
	switch physical.Estado {
	case "activo":
		return physical.Identidad != nil && *physical.RecursoVivo && *physical.EstadoMotor == "running"
	case "detenido":
		return !*physical.RecursoVivo && *physical.EstadoMotor == "removed"
	default:
		return false
	}
}

func (adapter *DockerAdapter) observeDockerSession(ctx context.Context, request ports.AgentObserveRequest, physical microvm.RespuestaContenedorV1) (ports.AgentObservation, error) {
	state := dockerObservationState{}
	for pageCount := 0; pageCount < maxDockerObservationPages; pageCount++ {
		page, missing, err := adapter.readDockerObservationPage(ctx, request, physical, state.cursor)
		if err != nil {
			return ports.AgentObservation{}, err
		}
		if missing {
			return nonterminalObservation(request, ports.AgentPending), nil
		}
		if !validDockerObservationPage(page, request, physical.Cerca, state.cursor) {
			return ports.AgentObservation{}, fail(CodeObservationResponseInvalid, nil)
		}
		for _, event := range page.Eventos {
			content, valid := decodeDockerObservationEvent(event, request, state.cursor+1)
			if !valid {
				return ports.AgentObservation{}, fail(CodeObservationResponseInvalid, nil)
			}
			observation, terminal, consumeErr := state.consume(request, event, content)
			if terminal || consumeErr != nil {
				return observation, consumeErr
			}
		}
		observation, done, pageErr := dockerObservationAfterPage(request, physical, page, state)
		if done || pageErr != nil {
			return observation, pageErr
		}
	}
	return ports.AgentObservation{}, fail(CodeObservationResponseInvalid, nil)
}

type dockerObservationState struct {
	cursor                           uint64
	stdout                           []byte
	outputOverflow, durableTruncated bool
	sessionStarted                   bool
}

func (adapter *DockerAdapter) readDockerObservationPage(ctx context.Context, request ports.AgentObserveRequest,
	physical microvm.RespuestaContenedorV1, cursor uint64) (microvm.RespuestaEventosSesionContenedorV1, bool, error) {
	if err := ctx.Err(); err != nil {
		return microvm.RespuestaEventosSesionContenedorV1{}, false, err
	}
	page, err := adapter.observer.LeerEventosSesionContenedor(ctx, request.ExternalRef, request.SessionRef.String(),
		microvm.ConsultaEventosSesionContenedorV1{RevisionEsperada: physical.Revision, Cerca: physical.Cerca,
			DespuesDe: cursor, MaximoEventos: microvm.MaximoEventosPaginaSesionV1,
			MaximoBytes: microvm.MaximoEventosSesionBytesV1})
	if err == nil {
		return page, false, nil
	}
	if contextErr := preserveContextError(ctx, err); contextErr != nil {
		return page, false, contextErr
	}
	if cursor == 0 && physical.Estado == "activo" && isNotFound(err) {
		return page, true, nil
	}
	return page, false, classifyDockerObservationError(err, true)
}

func (state *dockerObservationState) consume(request ports.AgentObserveRequest, event microvm.EventoSesionContenedorV1,
	content []byte) (ports.AgentObservation, bool, error) {
	state.cursor++
	switch event.Tipo {
	case microvm.TipoEventoSesionContenedorIniciada, microvm.TipoEventoSesionContenedorStderr:
		state.sessionStarted = true
	case microvm.TipoEventoSesionContenedorStdout:
		state.sessionStarted = true
		if int64(len(content)) > request.MaxOutputBytes-int64(len(state.stdout)) {
			state.outputOverflow = true
		} else if !state.outputOverflow {
			state.stdout = append(state.stdout, content...)
		}
	case microvm.TipoEventoSesionContenedorSalidaTruncada:
		state.sessionStarted, state.durableTruncated = true, true
	case microvm.TipoEventoSesionContenedorFinalizada:
		observation, err := terminalDockerObservation(
			request, event, state.stdout, state.outputOverflow, state.durableTruncated)
		return observation, true, err
	}
	return ports.AgentObservation{}, false, nil
}

func dockerObservationAfterPage(request ports.AgentObserveRequest, physical microvm.RespuestaContenedorV1,
	page microvm.RespuestaEventosSesionContenedorV1, state dockerObservationState) (ports.AgentObservation, bool, error) {
	if page.Terminal || page.SiguienteCursor != state.cursor {
		return ports.AgentObservation{}, true, fail(CodeObservationResponseInvalid, nil)
	}
	if len(page.Eventos) != 0 {
		return ports.AgentObservation{}, false, nil
	}
	if physical.Estado == "detenido" {
		return ports.AgentObservation{}, true, fail(CodeObservationResponseInvalid, nil)
	}
	status := ports.AgentPending
	if state.sessionStarted {
		status = ports.AgentRunning
	}
	return nonterminalObservation(request, status), true, nil
}

func validDockerObservationPage(page microvm.RespuestaEventosSesionContenedorV1, request ports.AgentObserveRequest,
	fence, cursor uint64) bool {
	checks := []bool{
		page.Referencia == request.ExternalRef, page.SesionRef == request.SessionRef.String(),
		page.Cerca == fence, page.Eventos != nil, page.CodigoError == nil,
		len(page.Eventos) <= microvm.MaximoEventosPaginaSesionV1,
		cursor <= maxDurableCounter-uint64(len(page.Eventos)),
		page.SiguienteCursor == cursor+uint64(len(page.Eventos)), page.SiguienteCursor <= maxDurableCounter,
	}
	for _, valid := range checks {
		if !valid {
			return false
		}
	}
	if len(page.Eventos) == 0 {
		return !page.Terminal
	}
	for _, event := range page.Eventos[:len(page.Eventos)-1] {
		if event.Tipo == microvm.TipoEventoSesionContenedorFinalizada {
			return false
		}
	}
	lastTerminal := page.Eventos[len(page.Eventos)-1].Tipo == microvm.TipoEventoSesionContenedorFinalizada
	return page.Terminal == lastTerminal
}

func decodeDockerObservationEvent(event microvm.EventoSesionContenedorV1, request ports.AgentObserveRequest,
	sequence uint64) ([]byte, bool) {
	if event.SesionRef != request.SessionRef.String() || event.Secuencia != sequence {
		return nil, false
	}
	content, err := base64.StdEncoding.Strict().DecodeString(event.ContenidoBase64)
	if err != nil || base64.StdEncoding.EncodeToString(content) != event.ContenidoBase64 || len(content) > 8192 {
		return nil, false
	}
	return content, validDockerObservationEventPayload(event, content)
}

func validDockerObservationEventPayload(event microvm.EventoSesionContenedorV1, content []byte) bool {
	switch event.Tipo {
	case microvm.TipoEventoSesionContenedorIniciada:
		return len(content) == 0 && dockerEventHasNoOutcome(event)
	case microvm.TipoEventoSesionContenedorStdout, microvm.TipoEventoSesionContenedorStderr:
		return dockerEventHasNoOutcome(event)
	case microvm.TipoEventoSesionContenedorSalidaTruncada:
		return len(content) == 0 && validDockerTruncatedEvent(event)
	case microvm.TipoEventoSesionContenedorFinalizada:
		return len(content) == 0 && !(event.CodigoSalida != nil && event.Senal != nil)
	default:
		return false
	}
}

func validDockerTruncatedEvent(event microvm.EventoSesionContenedorV1) bool {
	return event.CodigoError != nil && event.CodigoSalida == nil && event.Senal == nil && !event.Agotada
}

func dockerEventHasNoOutcome(event microvm.EventoSesionContenedorV1) bool {
	return event.CodigoSalida == nil && event.Senal == nil && !event.Agotada && event.CodigoError == nil
}

func terminalDockerObservation(request ports.AgentObserveRequest, event microvm.EventoSesionContenedorV1,
	stdout []byte, overflow, durableTruncated bool) (ports.AgentObservation, error) {
	if event.CodigoSalida == nil && event.Senal == nil && !event.Agotada && event.CodigoError == nil {
		return ports.AgentObservation{}, fail(CodeObservationResponseInvalid, nil)
	}
	status, code := dockerTerminalStatus(event, stdout, overflow, durableTruncated)
	observation := baseObservation(request, status)
	if status == ports.AgentCompleted {
		observation.MediaType = request.ArtifactMediaType
		observation.Content = append([]byte(nil), stdout...)
	} else {
		observation.ErrorCode = code
	}
	if err := ports.ValidateAgentObservation(observation, request.MaxOutputBytes); err != nil {
		return ports.AgentObservation{}, fail(CodeObservationResponseInvalid, err)
	}
	return observation, nil
}

func dockerTerminalStatus(event microvm.EventoSesionContenedorV1, stdout []byte,
	overflow, durableTruncated bool) (ports.AgentStatus, string) {
	switch {
	case event.CodigoError != nil:
		return ports.AgentFailed, CodeSessionFailed
	case durableTruncated:
		return ports.AgentFailed, CodeSessionOutputTruncated
	case event.Agotada:
		return ports.AgentFailed, CodeSessionExhausted
	case event.Senal != nil:
		return ports.AgentFailed, CodeSessionSignaled
	case event.CodigoSalida != nil && *event.CodigoSalida != 0:
		return ports.AgentFailed, CodeSessionExitNonzero
	case overflow:
		return ports.AgentFailed, CodeSessionOutputTooLarge
	case len(stdout) == 0:
		return ports.AgentFailed, CodeSessionOutputMissing
	case event.CodigoSalida == nil:
		return ports.AgentFailed, CodeSessionFailed
	default:
		return ports.AgentCompleted, ""
	}
}

func classifyDockerObservationError(err error, session bool) error {
	var containerError *microvm.ErrorContenedorV1
	if errors.As(err, &containerError) {
		return fail(CodeObservationResponseInvalid, err)
	}
	return classifyReadOnlyObservationError(err, session)
}
