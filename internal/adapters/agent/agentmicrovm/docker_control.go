package agentmicrovm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/ports"
)

const (
	CodeDockerStopRequestInvalid       = "agentmicrovm.docker_stop_request_invalid"
	CodeDockerStopCanceledBeforeSubmit = "agentmicrovm.docker_stop_canceled_before_submit"
	CodeDockerStopUnavailable          = "agentmicrovm.docker_stop_unavailable"
	CodeDockerStopRejected             = "agentmicrovm.docker_stop_rejected"
	CodeDockerStopResponseInvalid      = "agentmicrovm.docker_stop_response_invalid"
	dockerStopOperationDomain          = "orquesta.agentmicrovm.docker-stop-operation.v1\x00"
)

func (adapter *DockerAdapter) ControlCapabilities(ctx context.Context) (ports.AgentControlCapabilities, error) {
	if adapter == nil || nilInterface(adapter.client) || nilInterface(ctx) {
		return ports.AgentControlCapabilities{}, fail(CodeConfigurationInvalid, nil)
	}
	if err := adapter.negotiateDocker(ctx); err != nil {
		return ports.AgentControlCapabilities{}, err
	}
	return ports.AgentControlCapabilities{ForcedStop: true}, nil
}

func (adapter *DockerAdapter) Stop(ctx context.Context, request ports.AgentStopRequest) (ports.AgentStopReceipt, error) {
	launch, err := adapter.resolveDockerStopLaunch(ctx, request)
	if err != nil {
		return ports.AgentStopReceipt{}, err
	}
	if request.Mode != ports.AgentStopForced {
		return adapter.unsupportedDockerStop(request)
	}
	stored, found, err := adapter.resolveDockerStop(ctx, request, launch)
	if err != nil {
		return ports.AgentStopReceipt{}, err
	}
	if !found {
		body, encodeErr := microvm.CodificarSolicitudDetenerContenedorV1(microvm.SolicitudDetenerContenedorV1{
			RevisionEsperada: launch.LaunchBindingRevision, Cerca: request.StopActionFence,
		})
		if encodeErr != nil {
			return ports.AgentStopReceipt{}, fail(CodeDockerStopRequestInvalid, encodeErr)
		}
		stored, err = adapter.recordDockerStop(ctx, request, launch, body)
		if err != nil {
			return ports.AgentStopReceipt{}, err
		}
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentStopReceipt{}, fail(CodeDockerStopCanceledBeforeSubmit, err)
	}
	response, err := adapter.client.DetenerContenedorCodificada(
		ctx, stored.IdempotencyKey, stored.TargetRef, stored.Body,
	)
	if err != nil {
		if contextErr := preserveContextError(ctx, err); contextErr != nil {
			return ports.AgentStopReceipt{}, contextErr
		}
		return ports.AgentStopReceipt{}, classifyMutationError(err, CodeDockerStopRejected, CodeDockerStopUnavailable)
	}
	return adapter.completeDockerStop(request, launch, stored, response)
}

func (adapter *DockerAdapter) unsupportedDockerStop(request ports.AgentStopRequest) (ports.AgentStopReceipt, error) {
	receipt := dockerStopReceipt(request, ports.AgentStopUnsupported, "", adapter.now())
	if err := ports.ValidateAgentStopReceipt(request, receipt); err != nil {
		return ports.AgentStopReceipt{}, fail(CodeDockerStopResponseInvalid, err)
	}
	return receipt, nil
}

func (adapter *DockerAdapter) completeDockerStop(request ports.AgentStopRequest, launch ports.AgentProviderRequest,
	stored ports.AgentProviderStopRequest, response microvm.RespuestaContenedorV1) (ports.AgentStopReceipt, error) {
	if !adapter.validDockerStoppedShape(request, launch, response) || !validDockerStopIdentity(request, response) {
		return ports.AgentStopReceipt{}, fail(CodeDockerStopResponseInvalid, nil)
	}
	receipt := dockerStopReceipt(request, ports.AgentStopped, "agentmicrovm-docker-stop:"+stored.IdempotencyKey, adapter.now())
	if err := ports.ValidateAgentStopReceipt(request, receipt); err != nil {
		return ports.AgentStopReceipt{}, fail(CodeDockerStopResponseInvalid, err)
	}
	return receipt, nil
}

func (adapter *DockerAdapter) resolveDockerStopLaunch(
	ctx context.Context,
	request ports.AgentStopRequest,
) (ports.AgentProviderRequest, error) {
	if adapter == nil || nilInterface(ctx) {
		return ports.AgentProviderRequest{}, fail(CodeConfigurationInvalid, nil)
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentProviderRequest{}, err
	}
	if err := ports.ValidateAgentStopRequest(request); err != nil {
		return ports.AgentProviderRequest{}, fail(CodeDockerStopRequestInvalid, err)
	}
	if !adapter.validDockerStopAuthority(request) {
		return ports.AgentProviderRequest{}, fail(CodeDockerStopRequestInvalid, nil)
	}
	key := ports.AgentProviderRequestKey{ExecutionRef: request.ExecutionRef,
		ActionFence: request.LaunchActionFence, Stage: ports.AgentProviderRequestLaunch}
	stored, found, err := adapter.journal.ResolveAgentProviderRequest(ctx, key)
	if contextErr := preserveContextError(ctx, err); contextErr != nil {
		return ports.AgentProviderRequest{}, contextErr
	}
	if err != nil {
		return ports.AgentProviderRequest{}, fail(CodeDockerJournalFailed, err)
	}
	if !found || !validDockerStopLaunch(stored, key, request) {
		return ports.AgentProviderRequest{}, fail(CodeDockerStopRequestInvalid, nil)
	}
	return stored, nil
}

func (adapter *DockerAdapter) recordDockerStop(
	ctx context.Context,
	request ports.AgentStopRequest,
	launch ports.AgentProviderRequest,
	body []byte,
) (ports.AgentProviderStopRequest, error) {
	prepared := ports.AgentProviderStopRequest{
		Key: ports.AgentProviderStopRequestKey{ExecutionRef: request.ExecutionRef,
			LaunchActionFence: request.LaunchActionFence, StopActionFence: request.StopActionFence},
		StopEffectAttemptRef: request.StopEffectAttemptRef, ProviderRef: DockerProviderRef,
		IdempotencyKey: dockerStopOperationRef(request), TargetRef: launch.LaunchBindingRef,
		ExpectedRevision: launch.LaunchBindingRevision, Body: append([]byte(nil), body...),
		BodySHA256: ports.AgentProviderRequestBodySHA256(body),
	}
	stored, err := adapter.stopJournal.RecordAgentProviderStopRequest(ctx, prepared)
	if err != nil || ports.ValidateAgentProviderStopRequest(stored) != nil ||
		!reflect.DeepEqual(stored, prepared) {
		return ports.AgentProviderStopRequest{}, fail(CodeDockerJournalFailed, err)
	}
	return stored, nil
}

func (adapter *DockerAdapter) resolveDockerStop(ctx context.Context, request ports.AgentStopRequest,
	launch ports.AgentProviderRequest) (ports.AgentProviderStopRequest, bool, error) {
	key := ports.AgentProviderStopRequestKey{ExecutionRef: request.ExecutionRef,
		LaunchActionFence: request.LaunchActionFence, StopActionFence: request.StopActionFence}
	stored, found, err := adapter.stopJournal.ResolveAgentProviderStopRequest(ctx, key)
	if contextErr := preserveContextError(ctx, err); contextErr != nil {
		return ports.AgentProviderStopRequest{}, false, contextErr
	}
	if err != nil {
		return ports.AgentProviderStopRequest{}, false, fail(CodeDockerJournalFailed, err)
	}
	if !found {
		return ports.AgentProviderStopRequest{}, false, nil
	}
	if !validDockerStoredStop(request, launch, key, stored) {
		return ports.AgentProviderStopRequest{}, false, fail(CodeDockerJournalFailed, nil)
	}
	return stored, true, nil
}

func validDockerStoredStop(
	request ports.AgentStopRequest,
	launch ports.AgentProviderRequest,
	key ports.AgentProviderStopRequestKey,
	stored ports.AgentProviderStopRequest,
) bool {
	identity := [4]string{stored.StopEffectAttemptRef, stored.ProviderRef, stored.IdempotencyKey, stored.TargetRef}
	wantIdentity := [4]string{request.StopEffectAttemptRef, DockerProviderRef,
		dockerStopOperationRef(request), launch.LaunchBindingRef}
	if stored.Key != key || ports.ValidateAgentProviderStopRequest(stored) != nil ||
		identity != wantIdentity || stored.ExpectedRevision != launch.LaunchBindingRevision {
		return false
	}
	var physical microvm.SolicitudDetenerContenedorV1
	if json.Unmarshal(stored.Body, &physical) != nil || physical.RevisionEsperada != stored.ExpectedRevision ||
		physical.Cerca != request.StopActionFence {
		return false
	}
	canonical, err := microvm.CodificarSolicitudDetenerContenedorV1(physical)
	return err == nil && bytes.Equal(canonical, stored.Body)
}

func (adapter *DockerAdapter) validDockerStopAuthority(request ports.AgentStopRequest) bool {
	return request.ProviderRef == adapter.capabilities.ProviderRef &&
		request.ModelRef == adapter.capabilities.ModelRef && request.AgentRef == adapter.capabilities.AgentRef &&
		validPhysicalExecutionRef(request.ExternalRef)
}

func validDockerStopLaunch(
	stored ports.AgentProviderRequest,
	key ports.AgentProviderRequestKey,
	request ports.AgentStopRequest,
) bool {
	return stored.Key == key && ports.ValidateAgentProviderRequest(stored) == nil &&
		stored.ProviderRef == DockerProviderRef && stored.LaunchBindingRef == request.ExternalRef &&
		stored.LaunchBindingRevision != 0
}

func (adapter *DockerAdapter) validDockerStoppedShape(
	request ports.AgentStopRequest,
	launch ports.AgentProviderRequest,
	response microvm.RespuestaContenedorV1,
) bool {
	return response.Referencia == launch.LaunchBindingRef && response.Revision == launch.LaunchBindingRevision+2 &&
		response.Cerca == request.StopActionFence && response.Estado == "detenido" && response.VCPU == adapter.vcpu &&
		response.MemoriaMiB == adapter.memoryMiB && response.RecursoVivo != nil && !*response.RecursoVivo &&
		response.EstadoMotor != nil && *response.EstadoMotor == "removed"
}

func validDockerStopIdentity(request ports.AgentStopRequest, response microvm.RespuestaContenedorV1) bool {
	identity := response.Identidad
	return identity == nil || identity.PIDObservado != 0 && identity.ExecutionLabel == response.Referencia &&
		identity.RunLabel == request.ExecutionRef.String() && identity.Generation == request.StopActionFence
}

func dockerStopOperationRef(request ports.AgentStopRequest) string {
	digest := sha256.New()
	_, _ = digest.Write([]byte(dockerStopOperationDomain))
	for _, value := range []string{request.ExecutionRef.String(), request.StopEffectAttemptRef,
		request.IdempotencyKey, request.ExternalRef} {
		appendDigestField(digest, []byte(value))
	}
	_, _ = digest.Write(binary.BigEndian.AppendUint64(nil, request.LaunchActionFence))
	_, _ = digest.Write(binary.BigEndian.AppendUint64(nil, request.StopActionFence))
	return "orquesta-docker-stop:sha256:" + hex.EncodeToString(digest.Sum(nil))
}

func dockerStopReceipt(request ports.AgentStopRequest, status ports.AgentStopStatus,
	receiptRef string, confirmedAt time.Time) ports.AgentStopReceipt {
	receipt := ports.AgentStopReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration, ExecutionAttempt: request.ExecutionAttempt, StopEffectAttemptRef: request.StopEffectAttemptRef,
		StopActionFence: request.StopActionFence, SpecHash: request.SpecHash,
		ProviderRef: request.ProviderRef, ModelRef: request.ModelRef, AgentRef: request.AgentRef,
		ExternalRef: request.ExternalRef, Mode: request.Mode, IdempotencyKey: request.IdempotencyKey,
		Status: status,
	}
	if status != ports.AgentStopUnsupported && status != ports.AgentStopPending {
		receipt.ReceiptRef, receipt.ConfirmedAt = receiptRef, confirmedAt.UTC()
	}
	return receipt
}
