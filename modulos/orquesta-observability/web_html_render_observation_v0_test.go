package orquestaobservability

import "testing"

func TestWebHTMLRenderObservationV0ValidaContratoCompacto(t *testing.T) {
	observation := NewWebHTMLRenderObservationV0(
		"route-ref-web-ops",
		WebHTMLRenderFailedReasonV0,
		WebHTMLRenderStageRenderV0,
		500,
		"es",
	)

	if err := ValidateWebHTMLRenderObservationV0(observation); err != nil {
		t.Fatalf("ValidateWebHTMLRenderObservationV0: %v", err)
	}
	if observation.CounterKey != "web_html_render_failed_total" || observation.Count != 1 {
		t.Fatalf("observation=%+v", observation)
	}
}

func TestWebHTMLRenderObservationV0RechazaDatosNoCompactos(t *testing.T) {
	observation := NewWebHTMLRenderObservationV0(
		"/ops?token=secret",
		"web_html_render_failed",
		WebHTMLRenderStageRenderV0,
		500,
		"es",
	)

	if err := ValidateWebHTMLRenderObservationV0(observation); err == nil {
		t.Fatalf("error esperado")
	}
}

func TestInMemoryWebHTMLRenderObserverV0CuentaPorReason(t *testing.T) {
	observer := NewInMemoryWebHTMLRenderObserverV0()
	observer.ObserveWebHTMLRenderV0(NewWebHTMLRenderObservationV0(
		"route-ref-web-ops",
		WebHTMLResponseWriteFailedReasonV0,
		WebHTMLRenderStageWriteV0,
		200,
		"es",
	))

	observations, counters := observer.SnapshotV0()
	if len(observations) != 1 ||
		counters["web_response_write_failed_total"] != 1 {
		t.Fatalf("observations=%+v counters=%+v", observations, counters)
	}
}
