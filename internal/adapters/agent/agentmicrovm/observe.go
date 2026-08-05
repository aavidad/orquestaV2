package agentmicrovm

import (
	"context"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

const (
	CodeObservationRequestInvalid  = "agentmicrovm.observation_request_invalid"
	CodeObservationClientInvalid   = "agentmicrovm.observation_client_invalid"
	CodeObservationResponseInvalid = "agentmicrovm.observation_response_invalid"
	CodeSessionFailed              = "agentmicrovm.session_failed"
	CodeSessionExhausted           = "agentmicrovm.session_exhausted"
	CodeSessionSignaled            = "agentmicrovm.session_signaled"
	CodeSessionExitNonzero         = "agentmicrovm.session_exit_nonzero"
	CodeSessionOutputTruncated     = "agentmicrovm.output_truncated"
	CodeSessionOutputTooLarge      = "agentmicrovm.output_too_large"
	CodeSessionOutputMissing       = "agentmicrovm.output_missing"
)

// observationClient is the read-only part of the public sibling connector.
// Keeping it separate from Client makes mutation from ObserveAgent impossible.
type observationClient interface {
	Observar(context.Context, string) (microvm.RespuestaEjecucion, error)
	LeerEventosSesion(
		context.Context,
		string,
		string,
		microvm.ConsultaEventosSesionTrabajoV1,
	) (microvm.PaginaEventosSesionTrabajoV1, error)
}

// ObserveAgent reconstructs one observation from the sibling's durable event
// stream. It retains no cursor: restart and replay begin at zero and recover
// the same terminal proof from the public API.
func (adapter *Adapter) ObserveAgent(
	ctx context.Context,
	request ports.AgentObserveRequest,
) (ports.AgentObservation, error) {
	if err := ctx.Err(); err != nil {
		return ports.AgentObservation{}, err
	}
	if adapter == nil || adapter.client == nil || adapter.observer == nil {
		return ports.AgentObservation{}, fail(CodeConfigurationInvalid, nil)
	}
	if err := adapter.validateObserveRequest(request); err != nil {
		return ports.AgentObservation{}, err
	}
	physical, err := adapter.observer.Observar(ctx, request.ExternalRef)
	if err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return ports.AgentObservation{}, contextErr
		}
		return ports.AgentObservation{}, classifyPhysicalObservationError(err)
	}
	if physical.Referencia != request.ExternalRef || physical.Revision == 0 ||
		physical.Revision > maxDurableCounter || physical.Cerca == 0 || physical.Cerca > maxDurableCounter {
		return ports.AgentObservation{}, fail(CodeObservationResponseInvalid, nil)
	}

	return adapter.observeSession(ctx, adapter.observer, request, physical.Cerca)
}

func (adapter *Adapter) validateObserveRequest(request ports.AgentObserveRequest) error {
	if err := ports.ValidateAgentObserveRequest(request); err != nil {
		return fail(CodeObservationRequestInvalid, err)
	}
	if request.SessionRef.String() == "" || !validPhysicalExecutionRef(request.ExternalRef) {
		return fail(CodeObservationRequestInvalid, nil)
	}
	checks := []bool{
		request.ProviderRef == adapter.capabilities.ProviderRef,
		request.ModelRef == adapter.capabilities.ModelRef,
		request.AgentRef == adapter.capabilities.AgentRef,
	}
	for _, matches := range checks {
		if !matches {
			return fail(CodeObservationRequestInvalid, nil)
		}
	}
	return nil
}

func (adapter *Adapter) observeSession(
	ctx context.Context,
	client observationClient,
	request ports.AgentObserveRequest,
	fence uint64,
) (ports.AgentObservation, error) {
	var (
		cursor           uint64
		stdout           []byte
		outputOverflow   bool
		durableTruncated bool
		previousRevision uint64
		previousWork     uint64
		previousSession  uint64
	)
	for {
		if err := ctx.Err(); err != nil {
			return ports.AgentObservation{}, err
		}
		page, err := client.LeerEventosSesion(ctx, request.ExternalRef, request.SessionRef.String(),
			microvm.ConsultaEventosSesionTrabajoV1{
				Cerca: fence, DespuesDe: cursor, MaximoEventos: microvm.MaximoEventosPaginaSesionV1,
			})
		if err != nil {
			if contextErr := ctx.Err(); contextErr != nil {
				return ports.AgentObservation{}, contextErr
			}
			if cursor == 0 && isNotFound(err) {
				return nonterminalObservation(request, ports.AgentPending), nil
			}
			return ports.AgentObservation{}, classifySessionObservationError(err)
		}
		if !validObservationPage(page, request, fence, cursor) {
			return ports.AgentObservation{}, fail(CodeObservationResponseInvalid, nil)
		}

		for index := range page.Eventos {
			event := page.Eventos[index]
			if !validObservationEvent(
				event, request, fence, cursor+uint64(index)+1,
				previousRevision, previousWork, previousSession,
			) {
				return ports.AgentObservation{}, fail(CodeObservationResponseInvalid, nil)
			}
			previousRevision, previousWork, previousSession =
				event.Revision, event.RevisionTrabajo, event.RevisionSesion
			if event.Tipo == microvm.TipoEventoSesionStdout {
				content, decodeErr := base64.StdEncoding.Strict().DecodeString(event.ContenidoBase64)
				if decodeErr != nil || len(content) == 0 {
					return ports.AgentObservation{}, fail(CodeObservationResponseInvalid, decodeErr)
				}
				if int64(len(content)) > request.MaxOutputBytes-int64(len(stdout)) {
					outputOverflow = true
				} else if !outputOverflow {
					stdout = append(stdout, content...)
				}
			} else if event.Tipo == microvm.TipoEventoSesionSalidaTruncada {
				durableTruncated = true
			}
		}

		if page.Terminal {
			return terminalObservation(request, page, stdout, outputOverflow, durableTruncated)
		}
		if page.SiguienteCursor == cursor {
			return nonterminalObservation(request, translateNonterminalStatus(page.Estado)), nil
		}
		cursor = page.SiguienteCursor
	}
}

func validObservationPage(
	page microvm.PaginaEventosSesionTrabajoV1,
	request ports.AgentObserveRequest,
	fence uint64,
	cursor uint64,
) bool {
	if page.EjecucionRef != request.ExternalRef || page.SesionRef != request.SessionRef.String() ||
		page.Cerca != fence || page.DespuesDe != cursor || page.Revision == 0 ||
		page.RevisionTrabajo == 0 || page.RevisionSesion == 0 || page.Revision > maxDurableCounter ||
		page.RevisionTrabajo > maxDurableCounter || page.RevisionSesion > maxDurableCounter ||
		page.Eventos == nil || len(page.Eventos) > microvm.MaximoEventosPaginaSesionV1 ||
		page.SiguienteCursor < cursor || page.SiguienteCursor > maxDurableCounter ||
		page.SiguienteCursor != cursor+uint64(len(page.Eventos)) {
		return false
	}
	switch page.Estado {
	case microvm.EstadoSesionAdmitida, microvm.EstadoSesionIniciando, microvm.EstadoSesionActiva,
		microvm.EstadoSesionAmbigua, microvm.EstadoSesionFinalizada, microvm.EstadoSesionFallida:
	default:
		return false
	}
	if page.Terminal != (page.Estado == microvm.EstadoSesionFinalizada || page.Estado == microvm.EstadoSesionFallida) {
		return false
	}
	if len(page.Eventos) == 0 {
		return !page.Terminal
	}
	return page.Eventos[len(page.Eventos)-1].Terminal == page.Terminal
}

func validObservationEvent(
	event microvm.EventoSesionTrabajoV1,
	request ports.AgentObserveRequest,
	fence uint64,
	sequence uint64,
	previousRevision uint64,
	previousWork uint64,
	previousSession uint64,
) bool {
	if event.SesionRef != request.SessionRef.String() || event.Cerca != fence || event.Secuencia != sequence ||
		event.Revision == 0 || event.RevisionTrabajo == 0 || event.RevisionSesion == 0 ||
		event.Revision > maxDurableCounter || event.RevisionTrabajo > maxDurableCounter ||
		event.RevisionSesion > maxDurableCounter || event.Revision < previousRevision ||
		event.RevisionTrabajo < previousWork || event.RevisionSesion <= previousSession {
		return false
	}
	terminal := event.Tipo == microvm.TipoEventoSesionFinalizada
	if event.Terminal != terminal || event.Terminal != (event.Resultado != nil) {
		return false
	}
	switch event.Tipo {
	case microvm.TipoEventoSesionIniciada, microvm.TipoEventoSesionSalidaTruncada:
		return event.ContenidoBase64 == ""
	case microvm.TipoEventoSesionStdout, microvm.TipoEventoSesionStderr:
		content, err := base64.StdEncoding.Strict().DecodeString(event.ContenidoBase64)
		return err == nil && len(content) > 0 && len(content) <= 8_192
	case microvm.TipoEventoSesionFinalizada:
		return event.ContenidoBase64 == "" && event.Resultado != nil
	default:
		return false
	}
}

func terminalObservation(
	request ports.AgentObserveRequest,
	page microvm.PaginaEventosSesionTrabajoV1,
	stdout []byte,
	overflow bool,
	durableTruncated bool,
) (ports.AgentObservation, error) {
	last := page.Eventos[len(page.Eventos)-1]
	result := last.Resultado
	if result == nil {
		return ports.AgentObservation{}, fail(CodeObservationResponseInvalid, nil)
	}
	if !validTerminalResult(page.Estado, result) || result.SalidaTruncada != durableTruncated {
		return ports.AgentObservation{}, fail(CodeObservationResponseInvalid, nil)
	}
	status, failureCode := terminalStatus(page.Estado, result, stdout, overflow)
	observation := baseObservation(request, status)
	if status == ports.AgentCompleted {
		observation.MediaType = request.ArtifactMediaType
		observation.Content = append([]byte(nil), stdout...)
	} else {
		observation.ErrorCode = failureCode
	}
	if err := ports.ValidateAgentObservation(observation, request.MaxOutputBytes); err != nil {
		return ports.AgentObservation{}, fail(CodeObservationResponseInvalid, err)
	}
	return observation, nil
}

// validTerminalResult rejects incomplete or contradictory terminal receipts.
// In particular, only an explicit zero exit code can prove successful work;
// absence of every process outcome is not a failure category we may invent.
func validTerminalResult(
	state microvm.EstadoSesionTrabajoV1,
	result *microvm.ResultadoTerminalSesionTrabajoV1,
) bool {
	if result == nil || result.CodigoSalida != nil && result.Senal != nil {
		return false
	}
	switch state {
	case microvm.EstadoSesionFallida:
		return result.CodigoError != nil
	case microvm.EstadoSesionFinalizada:
		if result.CodigoError != nil {
			return false
		}
		return result.CodigoSalida != nil || result.Senal != nil || result.Agotada
	default:
		return false
	}
}

func terminalStatus(
	state microvm.EstadoSesionTrabajoV1,
	result *microvm.ResultadoTerminalSesionTrabajoV1,
	stdout []byte,
	overflow bool,
) (ports.AgentStatus, string) {
	switch {
	case state == microvm.EstadoSesionFallida || result.CodigoError != nil:
		return ports.AgentFailed, CodeSessionFailed
	case result.SalidaTruncada:
		return ports.AgentFailed, CodeSessionOutputTruncated
	case result.Agotada:
		return ports.AgentFailed, CodeSessionExhausted
	case result.Senal != nil:
		return ports.AgentFailed, CodeSessionSignaled
	case result.CodigoSalida != nil && *result.CodigoSalida != 0:
		return ports.AgentFailed, CodeSessionExitNonzero
	case overflow:
		return ports.AgentFailed, CodeSessionOutputTooLarge
	case len(stdout) == 0:
		return ports.AgentFailed, CodeSessionOutputMissing
	case result.CodigoSalida == nil || *result.CodigoSalida != 0:
		// validTerminalResult guarantees this branch cannot be reached for an
		// accepted success. Keep the guard local so future ordering changes do
		// not turn an incomplete receipt into AgentCompleted.
		return ports.AgentFailed, CodeSessionFailed
	default:
		return ports.AgentCompleted, ""
	}
}

func translateNonterminalStatus(state microvm.EstadoSesionTrabajoV1) ports.AgentStatus {
	switch state {
	case microvm.EstadoSesionActiva, microvm.EstadoSesionAmbigua:
		return ports.AgentRunning
	default:
		return ports.AgentPending
	}
}

func nonterminalObservation(
	request ports.AgentObserveRequest,
	status ports.AgentStatus,
) ports.AgentObservation {
	return baseObservation(request, status)
}

func baseObservation(request ports.AgentObserveRequest, status ports.AgentStatus) ports.AgentObservation {
	return ports.AgentObservation{
		ExecutionRef: request.ExecutionRef,
		SpecHash:     request.SpecHash,
		Status:       status,
		Usage: governance.ResourceUsage{
			Quality: governance.UsageQualityUnknown,
		},
		ObservedAt: time.Now().UTC(),
	}
}

func classifyPhysicalObservationError(err error) error {
	return classifyReadOnlyObservationError(err, false)
}

func classifySessionObservationError(err error) error {
	return classifyReadOnlyObservationError(err, true)
}

func classifyReadOnlyObservationError(err error, session bool) error {
	var protocolError *microvm.ErrorProtocolo
	if errors.As(err, &protocolError) {
		return fail(CodeProtocolIncompatible, err)
	}
	var configurationError *microvm.ErrorConfiguracion
	var responseTooLarge *microvm.ErrorRespuestaGrande
	var sessionError *microvm.ErrorSesionTrabajoV1
	if errors.As(err, &configurationError) || errors.As(err, &responseTooLarge) ||
		errors.As(err, &sessionError) {
		return fail(CodeObservationResponseInvalid, err)
	}
	var responseError *microvm.ErrorRespuesta
	if errors.As(err, &responseError) {
		if responseError.Estado == http.StatusConflict || responseError.Estado == http.StatusRequestTimeout ||
			responseError.Estado == http.StatusTooManyRequests || responseError.Estado >= http.StatusInternalServerError ||
			(responseError.Estado == http.StatusNotFound && !session) {
			return fail(CodeObservationUnavailable, err)
		}
		return fail(CodeObservationResponseInvalid, err)
	}
	var networkError net.Error
	if errors.As(err, &networkError) {
		return fail(CodeObservationUnavailable, err)
	}
	return fail(CodeObservationUnavailable, err)
}

func isNotFound(err error) bool {
	var responseError *microvm.ErrorRespuesta
	return errors.As(err, &responseError) && responseError.Estado == http.StatusNotFound
}
