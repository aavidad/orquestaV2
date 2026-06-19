package orquestaappcodexstack

import (
	"sort"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func codexStackOperationalClosureDeliveriesForTaskV0(
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
	trace codexStackOperationalClosureTraceV0,
	task orquestacoreworkflow.WorkflowTaskV0,
) []orquestacoreworkflow.DeliveryRegisteredPayloadV0 {
	deliveries := make([]orquestacoreworkflow.DeliveryRegisteredPayloadV0, 0)
	for _, deliveryRef := range request.Run.Deliveries {
		delivery, ok := trace.Deliveries[strings.TrimSpace(deliveryRef)]
		if !ok {
			continue
		}
		if strings.TrimSpace(delivery.TaskID) != strings.TrimSpace(task.TaskID) ||
			!codexStackOperationalClosureContainsV0(request.Run.Deliveries, delivery.DeliveryRef) {
			continue
		}
		deliveries = append(deliveries, delivery)
	}
	closedTasks := codexStackOperationalClosureSetV0(request.Run.ClosedTasks)
	for _, childRef := range codexStackOperationalClosureChildTaskRefsV0(request.Run, task) {
		if !closedTasks[childRef] {
			continue
		}
		for _, deliveryRef := range request.Run.Deliveries {
			delivery, ok := trace.Deliveries[strings.TrimSpace(deliveryRef)]
			if !ok || strings.TrimSpace(delivery.TaskID) != childRef ||
				!codexStackOperationalClosureContainsV0(request.Run.Deliveries, delivery.DeliveryRef) {
				continue
			}
			deliveries = append(deliveries, delivery)
		}
	}
	return deliveries
}

func codexStackOperationalClosureAcceptedReviewForDeliveryV0(
	request orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0,
	trace codexStackOperationalClosureTraceV0,
	deliveryRef string,
) (
	orquestacoreworkflow.ReviewRequestedPayloadV0,
	orquestacoreworkflow.ReviewAcceptedPayloadV0,
	orquestacoreworkflow.ReviewResultV0,
	bool,
) {
	deliveryRef = strings.TrimSpace(deliveryRef)
	for _, acceptedReviewRef := range request.Run.AcceptedReviews {
		accepted, ok := trace.AcceptedReviews[strings.TrimSpace(acceptedReviewRef)]
		if !ok {
			continue
		}
		if strings.TrimSpace(accepted.DeliveryRef) != deliveryRef ||
			!codexStackOperationalClosureContainsV0(request.Run.AcceptedReviews, accepted.AcceptedReviewRef) {
			continue
		}
		reviewRequest, ok := trace.ReviewRequests[strings.TrimSpace(accepted.ReviewRequestID)]
		if !ok || strings.TrimSpace(reviewRequest.DeliveryRef) != deliveryRef {
			continue
		}
		result, ok := codexStackOperationalClosureAcceptedResultV0(trace, accepted)
		if ok {
			return reviewRequest, accepted, result, true
		}
	}
	return orquestacoreworkflow.ReviewRequestedPayloadV0{}, orquestacoreworkflow.ReviewAcceptedPayloadV0{}, orquestacoreworkflow.ReviewResultV0{}, false
}

// codexStackOperationalClosureAcceptedResultV0 empareja una review aceptada con
// su resultado de review aceptado de forma determinista. Cuando una misma tarea
// pasa por varios ciclos de rework existen varios resultados aceptados para el
// mismo (review_request_id, delivery_ref); recorrer el map de resultados elegía
// uno arbitrario y, si no era el que registró la evidencia de tests de esta
// review aceptada, el cierre fallaba con required_test_evidence_refs aunque la
// evidencia durable existiera.
//
// Para emparejar accepted-review-ref-X -> review-result-ref-X usamos como ancla
// el conjunto de evidencias: la review aceptada y su resultado comparten las
// mismas evidence_refs (provienen de la misma observación de gate). Preferimos
// el resultado cuyo ReviewResultRef o EvidenceRefs estén referenciados por la
// review aceptada; solo si no hay match directo recurrimos al primer resultado
// aceptado en orden estable (trace.ReviewResultRefs).
func codexStackOperationalClosureAcceptedResultV0(
	trace codexStackOperationalClosureTraceV0,
	accepted orquestacoreworkflow.ReviewAcceptedPayloadV0,
) (orquestacoreworkflow.ReviewResultV0, bool) {
	acceptedEvidence := codexStackOperationalClosureSetV0(accepted.EvidenceRefs)
	var fallback orquestacoreworkflow.ReviewResultV0
	haveFallback := false
	for _, resultRef := range codexStackOperationalClosureAcceptedResultRefsV0(trace) {
		result, ok := trace.ReviewResults[resultRef]
		if !ok {
			continue
		}
		if strings.TrimSpace(result.ReviewRequestID) != strings.TrimSpace(accepted.ReviewRequestID) ||
			strings.TrimSpace(result.DeliveryRef) != strings.TrimSpace(accepted.DeliveryRef) ||
			result.Status != orquestacoreworkflow.ReviewResultStatusAcceptedV0 {
			continue
		}
		if codexStackOperationalClosureResultMatchesAcceptedV0(result, acceptedEvidence) {
			return result, true
		}
		if !haveFallback {
			fallback = result
			haveFallback = true
		}
	}
	return fallback, haveFallback
}

// codexStackOperationalClosureResultMatchesAcceptedV0 indica si el resultado de
// review corresponde directamente a la review aceptada: bien porque la review
// aceptada referencia el propio ReviewResultRef en sus evidencias, bien porque
// alguna de las evidencias del resultado coincide con las de la review aceptada.
func codexStackOperationalClosureResultMatchesAcceptedV0(
	result orquestacoreworkflow.ReviewResultV0,
	acceptedEvidence map[string]bool,
) bool {
	if len(acceptedEvidence) == 0 {
		return false
	}
	if acceptedEvidence[strings.TrimSpace(result.ReviewResultRef)] {
		return true
	}
	for _, evidenceRef := range result.EvidenceRefs {
		if acceptedEvidence[strings.TrimSpace(evidenceRef)] {
			return true
		}
	}
	return false
}

// codexStackOperationalClosureAcceptedResultRefsV0 devuelve los refs de
// resultados de review en orden estable. Usa el slice ordenado trace.ReviewResultRefs
// como fuente de determinismo y degrada a las claves del map (recorrido no
// determinista) solo si el slice no está poblado.
func codexStackOperationalClosureAcceptedResultRefsV0(
	trace codexStackOperationalClosureTraceV0,
) []string {
	if len(trace.ReviewResultRefs) > 0 {
		return trace.ReviewResultRefs
	}
	refs := make([]string, 0, len(trace.ReviewResults))
	for ref := range trace.ReviewResults {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	return refs
}
