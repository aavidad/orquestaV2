package orquestaappcodexstack

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

// TestCodexStackOperationalClosureAcceptedResultDeterministicPairingV0 cubre el
// bug de emparejamiento no determinista: cuando una tarea pasa por varios ciclos
// de rework existen varios resultados aceptados para el mismo
// (review_request_id, delivery_ref). El cierre debe elegir el resultado cuya
// evidencia coincide con la review aceptada concreta, no uno arbitrario del map.
func TestCodexStackOperationalClosureAcceptedResultDeterministicPairingV0(t *testing.T) {
	const (
		requestID  = "review-request-g03"
		deliveryRef = "delivery-ref-g03"
	)
	resultRework1 := orquestacoreworkflow.ReviewResultV0{
		ReviewResultRef: "review-result-ref-rework-1",
		ReviewRequestID: requestID,
		DeliveryRef:     deliveryRef,
		Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
		EvidenceRefs:    []string{"evidence-rework-1"},
	}
	resultRework2 := orquestacoreworkflow.ReviewResultV0{
		ReviewResultRef: "review-result-ref-rework-2",
		ReviewRequestID: requestID,
		DeliveryRef:     deliveryRef,
		Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
		EvidenceRefs:    []string{"evidence-rework-2"},
	}
	trace := codexStackOperationalClosureTraceV0{
		ReviewResults: map[string]orquestacoreworkflow.ReviewResultV0{
			resultRework1.ReviewResultRef: resultRework1,
			resultRework2.ReviewResultRef: resultRework2,
		},
		ReviewResultRefs: []string{resultRework1.ReviewResultRef, resultRework2.ReviewResultRef},
	}

	// La review aceptada comparte evidencias con el segundo resultado de rework.
	accepted := orquestacoreworkflow.ReviewAcceptedPayloadV0{
		AcceptedReviewRef: "accepted-review-ref-g03",
		ReviewRequestID:   requestID,
		DeliveryRef:       deliveryRef,
		EvidenceRefs:      []string{"evidence-rework-2"},
	}

	got, ok := codexStackOperationalClosureAcceptedResultV0(trace, accepted)
	if !ok {
		t.Fatalf("se esperaba emparejar un resultado aceptado")
	}
	if got.ReviewResultRef != resultRework2.ReviewResultRef {
		t.Fatalf("emparejamiento no determinista: got %q, want %q", got.ReviewResultRef, resultRework2.ReviewResultRef)
	}
}

// TestCodexStackOperationalClosureAcceptedResultMatchByResultRefV0 cubre el caso
// en el que la review aceptada referencia directamente el review_result_ref en
// sus evidencias.
func TestCodexStackOperationalClosureAcceptedResultMatchByResultRefV0(t *testing.T) {
	const (
		requestID   = "review-request-g03b"
		deliveryRef = "delivery-ref-g03b"
	)
	resultA := orquestacoreworkflow.ReviewResultV0{
		ReviewResultRef: "review-result-ref-a",
		ReviewRequestID: requestID,
		DeliveryRef:     deliveryRef,
		Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
	}
	resultB := orquestacoreworkflow.ReviewResultV0{
		ReviewResultRef: "review-result-ref-b",
		ReviewRequestID: requestID,
		DeliveryRef:     deliveryRef,
		Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
	}
	trace := codexStackOperationalClosureTraceV0{
		ReviewResults: map[string]orquestacoreworkflow.ReviewResultV0{
			resultA.ReviewResultRef: resultA,
			resultB.ReviewResultRef: resultB,
		},
		ReviewResultRefs: []string{resultA.ReviewResultRef, resultB.ReviewResultRef},
	}
	accepted := orquestacoreworkflow.ReviewAcceptedPayloadV0{
		AcceptedReviewRef: "accepted-review-ref-g03b",
		ReviewRequestID:   requestID,
		DeliveryRef:       deliveryRef,
		EvidenceRefs:      []string{resultB.ReviewResultRef},
	}

	got, ok := codexStackOperationalClosureAcceptedResultV0(trace, accepted)
	if !ok {
		t.Fatalf("se esperaba emparejar un resultado aceptado")
	}
	if got.ReviewResultRef != resultB.ReviewResultRef {
		t.Fatalf("emparejamiento por result-ref incorrecto: got %q, want %q", got.ReviewResultRef, resultB.ReviewResultRef)
	}
}

// TestCodexStackOperationalClosureAcceptedResultFallbackStableV0 verifica que,
// si no hay match directo por evidencia, el fallback es estable (primer
// resultado aceptado en orden de trace.ReviewResultRefs) y no depende del
// recorrido del map.
func TestCodexStackOperationalClosureAcceptedResultFallbackStableV0(t *testing.T) {
	const (
		requestID   = "review-request-g03c"
		deliveryRef = "delivery-ref-g03c"
	)
	first := orquestacoreworkflow.ReviewResultV0{
		ReviewResultRef: "review-result-ref-aaa",
		ReviewRequestID: requestID,
		DeliveryRef:     deliveryRef,
		Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
	}
	second := orquestacoreworkflow.ReviewResultV0{
		ReviewResultRef: "review-result-ref-zzz",
		ReviewRequestID: requestID,
		DeliveryRef:     deliveryRef,
		Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
	}
	trace := codexStackOperationalClosureTraceV0{
		ReviewResults: map[string]orquestacoreworkflow.ReviewResultV0{
			first.ReviewResultRef:  first,
			second.ReviewResultRef: second,
		},
		ReviewResultRefs: []string{first.ReviewResultRef, second.ReviewResultRef},
	}
	// Sin evidencias compartidas: cae al fallback ordenado.
	accepted := orquestacoreworkflow.ReviewAcceptedPayloadV0{
		AcceptedReviewRef: "accepted-review-ref-g03c",
		ReviewRequestID:   requestID,
		DeliveryRef:       deliveryRef,
	}

	for i := 0; i < 10; i++ {
		got, ok := codexStackOperationalClosureAcceptedResultV0(trace, accepted)
		if !ok {
			t.Fatalf("se esperaba emparejar un resultado aceptado")
		}
		if got.ReviewResultRef != first.ReviewResultRef {
			t.Fatalf("fallback no determinista en iteración %d: got %q, want %q", i, got.ReviewResultRef, first.ReviewResultRef)
		}
	}
}
