package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
)

func TestMCPRequestAppChangeHTTPV0InyectaAppRefDesdeRuta(t *testing.T) {
	executor := &fakeMCPRequestAppChangeExecutorV0{
		result: MCPRequestAppChangeToolResultV0{
			Estado:              MCPRequestAppChangeEstadoOKV0,
			RunRef:              "run-ref-http-change-001",
			AppRef:              "agenda-equipo",
			ChangeRef:           "change-ref-http-001",
			DirectorQuestionRef: "question-ref-http-change-001",
		},
	}
	handler := NewMCPRequestAppChangeHTTPHandlerV0(executor)
	body := new(bytes.Buffer)
	_ = json.NewEncoder(body).Encode(MCPRequestAppChangeToolInputV0{
		AppChangeRequest: validMCPRequestAppChangeHTTPInputV0(),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v0/apps/agenda-equipo/changes", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.input.AppChangeRequest.AppRef != "agenda-equipo" {
		t.Fatalf("app_ref=%q", executor.input.AppChangeRequest.AppRef)
	}
}

func TestMCPRequestAppChangeHTTPV0Devuelve400SiContratoInvalido(t *testing.T) {
	handler := NewMCPRequestAppChangeHTTPHandlerV0(&fakeMCPRequestAppChangeExecutorV0{
		result: MCPRequestAppChangeToolResultV0{
			Estado: MCPRequestAppChangeEstadoErrorV0,
			Errores: []MCPAppChangeIssueV0{{
				Code:  "app_change_user_intent_required",
				Field: "user_intent",
			}},
		},
	})
	body := new(bytes.Buffer)
	_ = json.NewEncoder(body).Encode(MCPRequestAppChangeToolInputV0{})
	req := httptest.NewRequest(http.MethodPost, MCPRequestAppChangeHTTPPathV0, body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func validMCPRequestAppChangeHTTPInputV0() orquestaappchange.AppChangeRequestV0 {
	return orquestaappchange.AppChangeRequestV0{
		RunRef:     "run-ref-http-change-001",
		ChangeRef:  "change-ref-http-001",
		UserIntent: "Cambiar la web para mostrar vista semanal.",
	}
}

type fakeMCPRequestAppChangeExecutorV0 struct {
	input  MCPRequestAppChangeToolInputV0
	result MCPRequestAppChangeToolResultV0
}

func (executor *fakeMCPRequestAppChangeExecutorV0) Execute(
	_ context.Context,
	input MCPRequestAppChangeToolInputV0,
) (MCPRequestAppChangeToolResultV0, error) {
	executor.input = input
	return executor.result, nil
}
