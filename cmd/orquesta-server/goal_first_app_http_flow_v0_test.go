package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
)

func TestServerAppHTTPGoalFirstLanzaObservaYCierraV0(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envServerStateDirV0, stateDir)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envCodexCommandV0, filepath.Join(projectDir, "codex-bin"))
	t.Setenv(envOPESBaseURLV0, "")
	t.Setenv("OPES_BASE_URL", "")

	backend := &goalFirstHTTPBackendForTestV0{}
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvWithGoalBackendV0(config, serverCodexGoalBackendV0{
		Starter:  backend,
		Observer: backend,
	})
	if err != nil {
		t.Fatalf("buildStackFromEnvWithGoalBackendV0: %v", err)
	}
	handler, err := buildServerAppHandlerV0(stack)
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0: %v", err)
	}

	started := postGoalFirstStartForTestV0(t, handler)
	if started.Estado != orquestamcp.MCPArrancarDirectorAppEstadoOKV0 ||
		started.RunRef == "" ||
		started.GoalRef == "" ||
		started.ExternalGoalRef != "thread-ref-http-goal-first-001" ||
		started.GoalStatus != orquestagoal.GoalStatusRunningV0 {
		t.Fatalf("started=%+v", started)
	}
	if backend.packet.GoalRef != started.GoalRef || backend.packet.Objective == "" || len(backend.packet.ArtifactContracts) == 0 {
		t.Fatalf("packet no capturado: %+v started=%+v", backend.packet, started)
	}

	observed := postGoalFirstObserveForTestV0(t, handler, started.RunRef)
	if observed.Estado != orquestamcp.MCPObserveAppDirectorGoalEstadoOKV0 ||
		observed.GoalRef != started.GoalRef ||
		observed.GoalStatus != orquestagoal.GoalStatusCompleteV0 ||
		observed.RunStatus != "cerrada" ||
		observed.ClosureStatus != orquestagoal.GoalStatusAcceptedV0 ||
		!observed.ClosureAccepted {
		t.Fatalf("observed=%+v", observed)
	}
}

func postGoalFirstStartForTestV0(
	t *testing.T,
	handler http.Handler,
) orquestamcp.MCPArrancarDirectorAppToolResultV0 {
	t.Helper()
	body := []byte(`{
		"request_id":"req-http-goal-first-001",
		"correlation_id":"corr-http-goal-first-001",
		"app_spec_request":{
			"schema_version":"app_spec_request.v0",
			"request_id":"request-ref-http-goal-first-001",
			"source":"orquesta-web",
			"locale":"es-ES",
			"nombre":"Agenda",
			"objetivo":"Gestionar contactos y citas desde una API y una web.",
			"tipo_app":"mixed",
			"preferencias_tecnicas":{"lenguaje":"go","arquitectura":"hexagonal"},
			"calidad":{"pruebas":"media","accesibilidad":"basica","observabilidad":true}
		}
	}`)
	req := httptest.NewRequest(http.MethodPost, orquestamcp.MCPArrancarDirectorAppHTTPPathV0, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Correlation-ID", "corr-http-goal-first-001")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("start status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPArrancarDirectorAppToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode start: %v", err)
	}
	return result
}

func postGoalFirstObserveForTestV0(
	t *testing.T,
	handler http.Handler,
	runRef string,
) orquestamcp.MCPObserveAppDirectorGoalToolResultV0 {
	t.Helper()
	payload, err := json.Marshal(orquestamcp.MCPObserveAppDirectorGoalToolInputV0{
		RequestID:   "req-http-goal-first-observe-001",
		RunRef:      runRef,
		RequestedBy: "orquesta-server-test",
	})
	if err != nil {
		t.Fatalf("marshal observe: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, orquestamcp.MCPObserveAppDirectorGoalHTTPPathV0, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Correlation-ID", "corr-http-goal-first-observe-001")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("observe status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPObserveAppDirectorGoalToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode observe: %v", err)
	}
	return result
}

type goalFirstHTTPBackendForTestV0 struct {
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0
}

func (backend *goalFirstHTTPBackendForTestV0) StartCodexGoalV0(
	_ context.Context,
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
) (orquestaruntimecodexgoal.CodexGoalStartReceiptV0, error) {
	backend.packet = packet
	return orquestaruntimecodexgoal.CodexGoalStartReceiptV0{
		Status:          orquestagoal.GoalStatusRunningV0,
		GoalRef:         packet.GoalRef,
		ExternalGoalRef: "thread-ref-http-goal-first-001",
		EvidenceRefs:    []string{"evidence-ref-http-goal-first-launch"},
	}, nil
}

func (backend *goalFirstHTTPBackendForTestV0) ObserveCodexGoalV0(
	_ context.Context,
	request orquestaruntimecodexgoal.CodexGoalObservationRequestV0,
) (orquestaruntimecodexgoal.CodexGoalObservationReceiptV0, error) {
	return orquestaruntimecodexgoal.CodexGoalObservationReceiptV0{
		Status:          orquestagoal.GoalStatusCompleteV0,
		GoalRef:         request.GoalRef,
		ExternalGoalRef: request.ExternalGoalRef,
		Summary:         "goal-first fake completo",
		ArtifactRefs:    goalFirstHTTPRequiredArtifactRefsForTestV0(backend.packet),
		EvidenceRefs: append(
			[]string{"evidence-ref-http-goal-first-observed"},
			backend.packet.ClosurePolicy.RequiredEvidenceRefs...,
		),
	}, nil
}

func goalFirstHTTPRequiredArtifactRefsForTestV0(
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
) []string {
	refs := make([]string, 0, len(packet.ArtifactContracts))
	for _, contract := range packet.ArtifactContracts {
		if contract.Required && contract.ArtifactRef != "" {
			refs = append(refs, contract.ArtifactRef)
		}
	}
	return refs
}
