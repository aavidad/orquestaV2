package main

import (
	"testing"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func TestServerWebHTMLRenderObserverV0AceptaContratoCompacto(t *testing.T) {
	observer := serverWebHTMLRenderObserverV0()
	observer.ObserveWebHTMLRenderV0(orquestaobservability.NewWebHTMLRenderObservationV0(
		"route-ref-web-ops",
		orquestaobservability.WebHTMLRenderFailedReasonV0,
		orquestaobservability.WebHTMLRenderStageRenderV0,
		500,
		"es",
	))

	store, ok := observer.(*orquestaobservability.InMemoryWebHTMLRenderObserverV0)
	if !ok {
		t.Fatalf("observer type=%T", observer)
	}
	observations, counters := store.SnapshotV0()
	if len(observations) != 1 || counters["web_html_render_failed_total"] != 1 {
		t.Fatalf("observations=%+v counters=%+v", observations, counters)
	}
}
