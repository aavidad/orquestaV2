package agentmicrovm

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net"
	"net/http"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/ports"
)

const (
	CodeWorkRevisionUnavailable   = "agentmicrovm.work_revision_unavailable"
	CodeWorkRevisionInvalid       = "agentmicrovm.work_revision_invalid"
	CodeSessionStartUnavailable   = "agentmicrovm.session_start_unavailable"
	CodeSessionStartRejected      = "agentmicrovm.session_start_rejected"
	CodeSessionResponseInvalid    = "agentmicrovm.session_response_invalid"
	CodeSessionPending            = "agentmicrovm.session_pending"
	CodeSessionAmbiguous          = "agentmicrovm.session_ambiguous"
	CodeSessionInconsistent       = "agentmicrovm.session_inconsistent"
	CodeSessionInputUnavailable   = "agentmicrovm.session_input_unavailable"
	CodeSessionInputRejected      = "agentmicrovm.session_input_rejected"
	CodeSessionInputPending       = "agentmicrovm.session_input_pending"
	CodeSessionReconcileInvalid   = "agentmicrovm.session_reconcile_invalid"
	CodeSessionObserveUnavailable = "agentmicrovm.session_observe_unavailable"
)

const (
	sessionStartKeyPrefix = "orquesta-session-start:sha256:"
	sessionInputKeyPrefix = "orquesta-session-input:sha256:"
	sessionStartStage     = "session-start"
	sessionInputStage     = "session-input"
	remoteSessionPending  = "sesiones.operacion_pendiente"
)

type sessionRevisions struct {
	execution uint64
	work      uint64
	session   uint64
}

// launchSession proves that one exact packet is durably accepted by an active
// session. It retains no replay state: restart recovery belongs to B10.5 and
// must cross an application-owned durable attempt rather than adapter memory.
func (adapter *Adapter) launchSession(
	ctx context.Context,
	request ports.AgentLaunchRequest,
	physical microvm.RespuestaEjecucion,
	timeBudgetMS uint64,
	packetJSON []byte,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	startKey := sessionIdempotencyKey(sessionStartKeyPrefix, sessionStartStage, request)
	inputKey := sessionIdempotencyKey(sessionInputKeyPrefix, sessionInputStage, request)

	resolved, absent, err := adapter.reconcileSessionInput(ctx, request, physical, inputKey)
	if err != nil || resolved {
		return err
	}
	if !absent {
		return fail(CodeSessionInputPending, nil)
	}

	revisions, sessionAbsent, err := adapter.observeSessionBeforeInput(ctx, request, physical)
	if err != nil {
		return err
	}
	if sessionAbsent {
		revisions, err = adapter.startSession(ctx, request, physical, startKey, timeBudgetMS)
		if err != nil {
			return err
		}
	}

	resolved, absent, err = adapter.reconcileSessionInput(ctx, request, physical, inputKey)
	if err != nil || resolved {
		return err
	}
	if !absent {
		return fail(CodeSessionInputPending, nil)
	}

	input := microvm.SolicitudEntradaSesionTrabajoV1{
		RevisionEsperada: revisions.execution, RevisionTrabajoEsperada: revisions.work,
		RevisionSesionEsperada: revisions.session, Cerca: physical.Cerca,
		ContenidoBase64: base64.StdEncoding.EncodeToString(packetJSON), CerrarStdin: true,
	}
	response, callErr := adapter.client.EnviarEntradaSesion(
		ctx, inputKey, physical.Referencia, request.SessionRef.String(), input,
	)
	if callErr != nil {
		if contextErr := preserveContextError(ctx, callErr); contextErr != nil {
			return contextErr
		}
		return classifySessionMutationError(
			callErr, CodeSessionInputUnavailable, CodeSessionInputRejected, CodeSessionInputPending,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !validActiveSessionResponse(response, physical.Referencia, request, physical.Cerca, revisions) ||
		response.RevisionSesion <= revisions.session {
		return fail(CodeSessionResponseInvalid, nil)
	}
	return nil
}

func (adapter *Adapter) startSession(
	ctx context.Context,
	request ports.AgentLaunchRequest,
	physical microvm.RespuestaEjecucion,
	key string,
	timeBudgetMS uint64,
) (sessionRevisions, error) {
	work, err := adapter.client.RevisionTrabajo(ctx, physical.Referencia)
	if err != nil {
		if contextErr := preserveContextError(ctx, err); contextErr != nil {
			return sessionRevisions{}, contextErr
		}
		return sessionRevisions{}, classifyReadOnlySessionError(err, CodeWorkRevisionUnavailable)
	}
	if err := ctx.Err(); err != nil {
		return sessionRevisions{}, err
	}
	if work.Referencia != physical.Referencia || work.RevisionTrabajo == 0 ||
		work.RevisionTrabajo > maxDurableCounter {
		return sessionRevisions{}, fail(CodeWorkRevisionInvalid, nil)
	}
	start := microvm.SolicitudIniciarSesionTrabajoV1{
		SesionRef: request.SessionRef.String(), RevisionEsperada: physical.Revision,
		RevisionTrabajoEsperada: work.RevisionTrabajo, Cerca: physical.Cerca,
		EjecutorRef: adapter.profile.Descriptor.EjecutorRef, DirectorioTrabajo: ".",
		PlazoTotalMilisegundos: timeBudgetMS,
		MaximoEventosBytes:     microvm.MaximoEventosSesionBytesV1,
	}
	response, err := adapter.client.IniciarSesion(ctx, key, physical.Referencia, start)
	if err != nil {
		if contextErr := preserveContextError(ctx, err); contextErr != nil {
			return sessionRevisions{}, contextErr
		}
		return sessionRevisions{}, classifySessionMutationError(
			err, CodeSessionStartUnavailable, CodeSessionStartRejected, CodeSessionPending,
		)
	}
	if err := ctx.Err(); err != nil {
		return sessionRevisions{}, err
	}
	minimum := sessionRevisions{execution: physical.Revision, work: work.RevisionTrabajo}
	if !validSessionResponse(response, physical.Referencia, request, physical.Cerca, minimum) {
		return sessionRevisions{}, fail(CodeSessionResponseInvalid, nil)
	}
	return classifySessionState(response.Estado, response.Terminal, response.Resultado,
		sessionRevisions{execution: response.Revision, work: response.RevisionTrabajo, session: response.RevisionSesion})
}

func (adapter *Adapter) observeSessionBeforeInput(
	ctx context.Context,
	request ports.AgentLaunchRequest,
	physical microvm.RespuestaEjecucion,
) (sessionRevisions, bool, error) {
	page, err := adapter.observer.LeerEventosSesion(ctx, physical.Referencia, request.SessionRef.String(),
		microvm.ConsultaEventosSesionTrabajoV1{
			Cerca: physical.Cerca, DespuesDe: 0, MaximoEventos: microvm.MaximoEventosPaginaSesionV1,
		})
	if err != nil {
		if contextErr := preserveContextError(ctx, err); contextErr != nil {
			return sessionRevisions{}, false, contextErr
		}
		if isNotFound(err) {
			return sessionRevisions{}, true, nil
		}
		return sessionRevisions{}, false, classifyReadOnlySessionError(err, CodeSessionObserveUnavailable)
	}
	if err := ctx.Err(); err != nil {
		return sessionRevisions{}, false, err
	}
	observeRequest := ports.AgentObserveRequest{
		ExternalRef: physical.Referencia, SessionRef: request.SessionRef,
		LaunchActionFence: request.EffectAuthority.ActionFence,
	}
	if !validObservationPage(page, observeRequest, physical.Cerca, 0) {
		return sessionRevisions{}, false, fail(CodeSessionResponseInvalid, nil)
	}
	var previousRevision, previousWork, previousSession uint64
	for index := range page.Eventos {
		event := page.Eventos[index]
		if !validObservationEvent(event, observeRequest, physical.Cerca, uint64(index)+1,
			previousRevision, previousWork, previousSession) {
			return sessionRevisions{}, false, fail(CodeSessionResponseInvalid, nil)
		}
		previousRevision, previousWork, previousSession = event.Revision, event.RevisionTrabajo, event.RevisionSesion
	}
	revisions, stateErr := classifySessionState(page.Estado, page.Terminal, nil,
		sessionRevisions{execution: page.Revision, work: page.RevisionTrabajo, session: page.RevisionSesion})
	return revisions, false, stateErr
}

func (adapter *Adapter) reconcileSessionInput(
	ctx context.Context,
	request ports.AgentLaunchRequest,
	physical microvm.RespuestaEjecucion,
	key string,
) (resolved bool, absent bool, err error) {
	response, callErr := adapter.client.ReconciliarEntradaSesion(
		ctx, key, physical.Referencia, request.SessionRef.String(), physical.Cerca,
	)
	if callErr != nil {
		if contextErr := preserveContextError(ctx, callErr); contextErr != nil {
			return false, false, contextErr
		}
		if isNotFound(callErr) {
			return false, true, nil
		}
		return false, false, classifyReadOnlySessionError(callErr, CodeSessionInputUnavailable)
	}
	if err := ctx.Err(); err != nil {
		return false, false, err
	}
	if response.EjecucionRef != physical.Referencia || response.SesionRef != request.SessionRef.String() ||
		response.Cerca != physical.Cerca {
		return false, false, fail(CodeSessionReconcileInvalid, nil)
	}
	switch response.Estado {
	case microvm.EstadoReconciliacionEntradaPendiente:
		if response.Comprobante != nil {
			return false, false, fail(CodeSessionReconcileInvalid, nil)
		}
		return false, false, nil
	case microvm.EstadoReconciliacionEntradaResuelta:
		if response.Comprobante == nil ||
			!validActiveSessionResponse(
				*response.Comprobante, physical.Referencia, request, physical.Cerca,
				sessionRevisions{execution: physical.Revision},
			) {
			return false, false, fail(CodeSessionReconcileInvalid, nil)
		}
		return true, false, nil
	default:
		return false, false, fail(CodeSessionReconcileInvalid, nil)
	}
}

func validSessionResponse(
	response microvm.RespuestaSesionTrabajoV1,
	externalRef string,
	request ports.AgentLaunchRequest,
	fence uint64,
	minimum sessionRevisions,
) bool {
	return response.EjecucionRef == externalRef &&
		response.SesionRef == request.SessionRef.String() && response.Cerca == fence &&
		response.Revision > 0 && response.Revision <= maxDurableCounter &&
		response.RevisionTrabajo > 0 && response.RevisionTrabajo <= maxDurableCounter &&
		response.RevisionSesion > 0 && response.RevisionSesion <= maxDurableCounter &&
		response.Revision >= minimum.execution && response.RevisionTrabajo >= minimum.work &&
		response.RevisionSesion >= minimum.session
}

func validActiveSessionResponse(
	response microvm.RespuestaSesionTrabajoV1,
	externalRef string,
	request ports.AgentLaunchRequest,
	fence uint64,
	minimum sessionRevisions,
) bool {
	return validSessionResponse(response, externalRef, request, fence, minimum) &&
		response.Estado == microvm.EstadoSesionActiva && !response.Terminal && response.Resultado == nil
}

func classifySessionState(
	state microvm.EstadoSesionTrabajoV1,
	terminal bool,
	result *microvm.ResultadoTerminalSesionTrabajoV1,
	revisions sessionRevisions,
) (sessionRevisions, error) {
	switch state {
	case microvm.EstadoSesionActiva:
		if terminal || result != nil {
			return sessionRevisions{}, fail(CodeSessionResponseInvalid, nil)
		}
		return revisions, nil
	case microvm.EstadoSesionAdmitida, microvm.EstadoSesionIniciando:
		if terminal || result != nil {
			return sessionRevisions{}, fail(CodeSessionResponseInvalid, nil)
		}
		return sessionRevisions{}, fail(CodeSessionPending, nil)
	case microvm.EstadoSesionAmbigua:
		return sessionRevisions{}, fail(CodeSessionAmbiguous, nil)
	case microvm.EstadoSesionFinalizada, microvm.EstadoSesionFallida:
		return sessionRevisions{}, fail(CodeSessionInconsistent, nil)
	default:
		return sessionRevisions{}, fail(CodeSessionResponseInvalid, nil)
	}
}

func sessionIdempotencyKey(prefix string, stage string, request ports.AgentLaunchRequest) string {
	digest := sha256.New()
	for _, field := range []string{
		request.IdempotencyKey, request.ExecutionRef.String(), request.SessionRef.String(),
		request.SpecHash, request.EffectAuthority.EffectAttemptRef, stage,
	} {
		appendDigestField(digest, []byte(field))
	}
	return prefix + hex.EncodeToString(digest.Sum(nil))
}

func classifyReadOnlySessionError(err error, unavailableCode string) error {
	var protocolError *microvm.ErrorProtocolo
	var configurationError *microvm.ErrorConfiguracion
	var responseTooLarge *microvm.ErrorRespuestaGrande
	var sessionError *microvm.ErrorSesionTrabajoV1
	if errors.As(err, &protocolError) || errors.As(err, &configurationError) ||
		errors.As(err, &responseTooLarge) || errors.As(err, &sessionError) {
		return fail(CodeSessionResponseInvalid, err)
	}
	var responseError *microvm.ErrorRespuesta
	if errors.As(err, &responseError) && responseError.Estado != http.StatusRequestTimeout &&
		responseError.Estado != http.StatusTooManyRequests && responseError.Estado < http.StatusInternalServerError {
		return fail(CodeSessionResponseInvalid, err)
	}
	return fail(unavailableCode, err)
}

func classifySessionMutationError(
	err error,
	unavailableCode string,
	rejectedCode string,
	pendingCode string,
) error {
	var responseError *microvm.ErrorRespuesta
	if errors.As(err, &responseError) && responseError.Estado == http.StatusConflict &&
		responseError.Codigo == remoteSessionPending {
		return fail(pendingCode, err)
	}
	if errors.As(err, &responseError) && responseError.Estado != http.StatusRequestTimeout &&
		responseError.Estado != http.StatusTooManyRequests && responseError.Estado < http.StatusInternalServerError {
		return fail(rejectedCode, err)
	}
	var protocolError *microvm.ErrorProtocolo
	var configurationError *microvm.ErrorConfiguracion
	var responseTooLarge *microvm.ErrorRespuestaGrande
	var sessionError *microvm.ErrorSesionTrabajoV1
	if errors.As(err, &protocolError) || errors.As(err, &configurationError) ||
		errors.As(err, &responseTooLarge) || errors.As(err, &sessionError) {
		return fail(rejectedCode, err)
	}
	var networkError net.Error
	if errors.As(err, &networkError) || responseError != nil {
		return fail(unavailableCode, err)
	}
	return fail(unavailableCode, err)
}
