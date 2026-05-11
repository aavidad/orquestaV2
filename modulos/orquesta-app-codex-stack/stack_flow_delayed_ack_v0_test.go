package orquestaappcodexstack

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestCodexStackV0WebDrenaACKMultiagenteTardio(t *testing.T) {
	runtime := newDelayedAckCodexStackRuntimeV0(500 * time.Millisecond)
	stack := mustBuildCodexStackForTestV0(t, runtime)

	form := codexStackFormValuesV0()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/nueva-app", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	stack.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Director arrancado") {
		t.Fatalf("web status=%d body=%s", rec.Code, rec.Body.String())
	}
	store := stack.Stores.ReceiptStore.(*orquestaruntimecodexdelivery.InMemoryCodexReceiptDescriptorStoreV0)
	descriptors := codexStackRealSmokeDescriptorsV0(t, store)
	if len(descriptors) != 4 {
		t.Fatalf("descriptors=%d", len(descriptors))
	}
	drain, err := stack.DrainRunV0(req.Context(), DrainRunRequestV0{
		RunRef:           descriptors[0].RunID,
		MaxExternalWaits: 1000,
	})
	if err != nil {
		t.Fatalf("DrainRunV0: %v", err)
	}
	run, err := stack.Stores.RunStore.LoadRunV0(req.Context(), descriptors[0].RunID)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.PhaseArtifacts) < 4 {
		t.Fatalf(
			"phase_artifacts=%v started=%v failed=%v stopped=%v deliveries=%v drain_status=%s attempts=%d waits=%d",
			run.PhaseArtifacts,
			run.StartedAgents,
			run.FailedAgents,
			run.StoppedAgents,
			run.Deliveries,
			drain.Status,
			len(drain.Attempts),
			len(drain.ExternalWaits),
		)
	}
	assertPhaseArtifactEventsForRunV0(t, stack, descriptors[0].RunID, 4)
}
