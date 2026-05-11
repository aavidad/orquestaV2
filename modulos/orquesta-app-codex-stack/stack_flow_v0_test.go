package orquestaappcodexstack

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildStackV0RequiresOptInAndExplicitPorts(t *testing.T) {
	if _, err := BuildStackV0(ConfigV0{}); err == nil {
		t.Fatalf("esperaba opt-in requerido")
	}
}

func TestCodexStackV0GatewayAPIYWebArrancanEquipoDirectorConRuntimeInyectado(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	if stack.Ports.ReviewGateSource == nil {
		t.Fatalf("review gate source no conectado")
	}

	director := postDirectorAPIV0(t, stack)
	if len(director.StartedAgents) != 4 || runtime.launchCountV0() != 4 {
		t.Fatalf("director=%+v launches=%d", director, runtime.launchCountV0())
	}
	assertStatsForRunV0(t, stack.Handler, director.RunRef, 4)
	assertPhaseArtifactEventsForRunV0(t, stack, director.RunRef, 4)

	form := codexStackFormValuesV0()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/nueva-app", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	stack.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Director arrancado") {
		t.Fatalf("web status=%d body=%s", rec.Code, rec.Body.String())
	}
	if runtime.launchCountV0() != 8 {
		t.Fatalf("launches tras web=%d", runtime.launchCountV0())
	}
}
