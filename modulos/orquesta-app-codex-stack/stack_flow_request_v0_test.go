package orquestaappcodexstack

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaweb "orquesta/modulos/orquesta-web"
)

func postDirectorAPIV0(
	t *testing.T,
	stack StackV0,
) orquestamcp.MCPArrancarDirectorAppToolResultV0 {
	t.Helper()
	body := bytes.NewBuffer(nil)
	err := json.NewEncoder(body).Encode(orquestamcp.MCPArrancarDirectorAppToolInputV0{
		RequestID:            "request-ref-app-stack-api-001",
		CorrelationID:        "corr-app-stack-api-001",
		AppSpecRequest:       codexStackAppSpecRequestV0(),
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxExternalWaits:     4,
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/apps/director", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("director status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPArrancarDirectorAppToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode director: %v", err)
	}
	if result.Estado != orquestamcp.MCPArrancarDirectorAppEstadoOKV0 {
		t.Fatalf("director result=%+v", result)
	}
	return result
}

func assertStatsForRunV0(t *testing.T, handler http.Handler, runRef string, agents int) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodGet,
		"/director-stats?run_ref="+url.QueryEscape(runRef),
		nil,
	)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("stats status=%d body=%s", rec.Code, rec.Body.String())
	}
	var page orquestaweb.WebDirectorStatsPageV0
	if err := json.NewDecoder(rec.Body).Decode(&page); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	if page.ViewModel.Counts.AgentsStarted != agents ||
		page.ViewModel.Counts.Brainstorms != agents ||
		page.ViewModel.Counts.TasksTotal != 0 {
		t.Fatalf("stats=%+v", page.ViewModel.Counts)
	}
}

func assertPhaseArtifactEventsForRunV0(t *testing.T, stack StackV0, runRef string, want int) {
	t.Helper()
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.PhaseArtifacts) < want {
		t.Fatalf("phase_artifacts=%v want>=%d", run.PhaseArtifacts, want)
	}
	sink, ok := stack.Stores.EventSink.(*orquestacionnucleoapp.InMemoryEventSinkV0)
	if !ok {
		return
	}
	got := codexStackRealSmokeEventCountV0(sink.EventsV0(), "PhaseArtifactRegistered")
	if got < want {
		t.Fatalf("PhaseArtifactRegistered=%d want>=%d artifacts=%v", got, want, run.PhaseArtifacts)
	}
}

func codexStackAppSpecRequestV0() orquestafactory.AppSpecRequestV0 {
	enabled := true
	return orquestafactory.AppSpecRequestV0{
		SchemaVersion: orquestafactory.AppSpecRequestSchemaV0,
		RequestID:     "request-ref-app-stack-app-001",
		Source:        "orquesta-app-stack-test",
		Locale:        "es-ES",
		RequestKind:   orquestafactory.RequestKindCrearAppCompletaV0,
		ExecutionMode: orquestafactory.ExecutionModeNormalV0,
		Nombre:        "Agenda API Web",
		Objetivo:      "Gestionar una agenda con API REST en Go y web de administracion.",
		TipoApp:       "mixed",
		Plataformas:   []string{"server", "web"},
		PreferenciasTecnicas: orquestafactory.PreferenciasTecnicasV0{
			Lenguaje:     "go",
			Arquitectura: "hexagonal",
		},
		Datos: orquestafactory.DatosRequestV0{
			DBRequired:         true,
			NecesidadFuncional: "Persistir contactos y citas mediante puerto y conector elegido por el usuario.",
		},
		Calidad: orquestafactory.CalidadRequestV0{
			Pruebas:        "alta",
			Accesibilidad:  "basica",
			Observabilidad: &enabled,
		},
		I18N: orquestafactory.I18NRequestV0{
			Enabled:       &enabled,
			DefaultLocale: "es-ES",
			Locales:       []string{"es-ES", "en-US"},
		},
		Agentes: orquestafactory.AgentesRequestV0{Autonomia: "alta"},
	}
}

func codexStackFormValuesV0() url.Values {
	values := url.Values{}
	values.Set("request_id", "request-ref-app-stack-web-001")
	values.Set("locale", "es-ES")
	values.Set("request_kind", "crear_app_completa")
	values.Set("execution_mode", "normal")
	values.Set("nombre", "Agenda API Web")
	values.Set("objetivo", "Gestionar agenda con API REST en Go y web de administracion.")
	values.Set("tipo_app", "mixed")
	values.Set("plataformas", "server,web")
	values.Set("preferencias_tecnicas.lenguaje", "go")
	values.Set("preferencias_tecnicas.arquitectura", "hexagonal")
	values.Set("datos.db_required", "true")
	values.Set("datos.necesidad_funcional", "Persistir contactos y citas por puerto.")
	values.Set("calidad.pruebas", "alta")
	values.Set("calidad.accesibilidad", "basica")
	values.Set("calidad.observabilidad", "true")
	values.Set("i18n.enabled", "true")
	values.Set("i18n.default_locale", "es-ES")
	values.Set("i18n.locales", "es-ES,en-US")
	values.Set("agentes.autonomia", "alta")
	return values
}
