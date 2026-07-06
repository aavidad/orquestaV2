package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestahttpgateway "orquesta/modulos/orquesta-http-gateway"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaweb "orquesta/modulos/orquesta-web"
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

func TestBuildStackV0CableaDomainWorkOptIn(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	domainWork := &fakeCodexStackDomainWorkExecutorV0{}
	stack := mustBuildCodexStackWithDomainWorkForTestV0(t, runtime, domainWork)
	input := orquestamcp.MCPDomainWorkToolInputV0{
		RequestID:     "request-ref-codex-stack-domain-001",
		CorrelationID: "corr-codex-stack-domain-001",
		Action:        orquestamcp.MCPDomainWorkActionCreateJobV0,
		JobRequest: orquestadomainwork.DomainWorkJobRequestV0{
			SchemaVersion:  orquestadomainwork.DomainWorkJobRequestSchemaV0,
			RequestID:      "request-ref-codex-stack-domain-001",
			CorrelationID:  "corr-codex-stack-domain-001",
			IdempotencyKey: "idem-codex-stack-domain-001",
			RequestedBy:    "orquesta",
			DomainRef:      "opes",
			WorkKind:       "draft_content_block",
			Objective:      "crear bloque documental",
		},
	}
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/domain-work", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	stack.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if domainWork.called != 1 || domainWork.input.JobRequest.WorkKind != "draft_content_block" {
		t.Fatalf("domain work no delegado=%+v", domainWork)
	}
}

func TestBuildStackHTTPHandlerV0WiresNuevaAppIntakeAssistant(t *testing.T) {
	config := codexStackBaseConfigForTestV0(t, newFakeCodexStackRuntimeV0(), nil, nil)
	config.AppIntakeAssistant = fakeCodexStackNuevaAppIntakeAssistantV0{}
	stack, err := BuildStackV0(config)
	if err != nil {
		t.Fatalf("BuildStackV0: %v", err)
	}
	body := `{"session_id":"session-codex-stack-guided-assistant","locale":"es","need":"Necesito coordinar avisos clinicos"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, orquestahttpgateway.RouteAppIntakeGuidedTurnV0, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	stack.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out orquestaweb.WebNuevaAppIntakeGuidedResponseV0
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Session.Form.Nombre != "Turnos clinicos stack" ||
		out.Session.Form.TipoApp != "web" ||
		len(out.Session.Form.Integraciones) == 0 ||
		out.Session.Form.Integraciones[0].Tipo != "messaging" {
		t.Fatalf("assistant no cableado en stack: %+v", out.Session.Form)
	}
}

func TestBuildStackV0CableaNuevaAppWizardMCPRico(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	result, err := stack.MCPTransportBindings.NuevaAppWizard.Execute(context.Background(), orquestamcp.MCPNuevaAppWizardToolInputV0{
		RequestID: "request-ref-stack-wizard-001",
		SessionID: "session-stack-wizard-001",
		Session:   json.RawMessage(`{"schema_version":"web_nueva_app_intake_session.v0","session_id":"session-stack-wizard-001","form":{"request_id":"request-ref-stack-wizard-001","objetivo":"quiero una app para una agenda"}}`),
	})
	if err != nil {
		t.Fatalf("execute wizard: %v", err)
	}
	if result.Estado != orquestamcp.MCPNuevaAppWizardEstadoOKV0 ||
		!strings.Contains(string(result.Wizard), "wizard-r1-uso-personal-compartido") ||
		!strings.Contains(string(result.Wizard), "hexagonal_puertos_adaptadores") {
		t.Fatalf("wizard inicial incompleto: estado=%s wizard=%s", result.Estado, result.Wizard)
	}
	dataTurn, err := stack.MCPTransportBindings.NuevaAppWizard.Execute(context.Background(), orquestamcp.MCPNuevaAppWizardToolInputV0{
		RequestID: "request-ref-stack-wizard-data-001",
		SessionID: "session-stack-wizard-data-001",
		Session:   json.RawMessage(`{"schema_version":"web_nueva_app_intake_session.v0","session_id":"session-stack-wizard-data-001","form":{"request_id":"request-ref-stack-wizard-data-001","locale":"es-ES","nombre":"Agenda","objetivo":"quiero una app para una agenda","descripcion":"flujo operativo de agenda","tipo_app":"web","usuarios_objetivo":["usuarios autenticados"],"plataformas":["web","mobile"],"agentes":{"autonomia":"media"}}}`),
	})
	if err != nil {
		t.Fatalf("execute wizard data: %v", err)
	}
	wizardData := string(dataTurn.Wizard)
	hasDomainCalendar := strings.Contains(wizardData, "wizard-r3-integracion-agenda") &&
		strings.Contains(wizardData, "calendar_google_workspace")
	hasUniversalCalendar := strings.Contains(wizardData, "wizard-u7-integraciones") &&
		strings.Contains(wizardData, "integracion_calendario")
	if !hasDomainCalendar && !hasUniversalCalendar {
		t.Fatalf("wizard data no ofrece integracion agenda: %s", dataTurn.Wizard)
	}

	next, err := stack.MCPTransportBindings.NuevaAppWizard.Execute(context.Background(), orquestamcp.MCPNuevaAppWizardToolInputV0{
		RequestID:        "request-ref-stack-wizard-002",
		Need:             "quiero una app para una agenda",
		GlossaryExpanded: true,
		WizardAnswers: []orquestamcp.MCPNuevaAppWizardAnswerV0{{
			QuestionRef:   "wizard-r2-plataformas",
			UserChoice:    "web",
			Justification: "solo oficina",
		}},
	})
	if err != nil {
		t.Fatalf("execute wizard answer: %v", err)
	}
	if !strings.Contains(string(next.Wizard), `"recommended":"web_mobile"`) ||
		!strings.Contains(string(next.Wizard), `"user_choice":"web"`) ||
		!strings.Contains(string(next.Wizard), `"justification":"solo oficina"`) ||
		!strings.Contains(string(next.Wizard), `"glossary_expanded":true`) {
		t.Fatalf("wizard no conserva contraste: %s", next.Wizard)
	}

	glossary, err := stack.MCPTransportBindings.NuevaAppWizard.Execute(context.Background(), orquestamcp.MCPNuevaAppWizardToolInputV0{
		RequestID: "request-ref-stack-wizard-glossary-001",
		Need:      "quiero una app para una agenda",
		WizardAnswers: []orquestamcp.MCPNuevaAppWizardAnswerV0{{
			ComprehensionQuery: "que es web?",
		}},
	})
	if err != nil {
		t.Fatalf("execute wizard glossary: %v", err)
	}
	if !strings.Contains(string(glossary.Wizard), `"glossary_response"`) ||
		!strings.Contains(string(glossary.Wizard), "tipo_app.web") {
		t.Fatalf("wizard MCP no propaga consulta de comprension: %s", glossary.Wizard)
	}
}

type fakeCodexStackNuevaAppIntakeAssistantV0 struct{}

func (fakeCodexStackNuevaAppIntakeAssistantV0) BuildNuevaAppIntakeGuidedTurnV0(
	context.Context,
	orquestaweb.WebNuevaAppIntakeGuidedRequestV0,
	orquestaweb.WebNuevaAppIntakeSessionV0,
) (orquestaweb.WebNuevaAppIntakeGuidedTurnV0, error) {
	return orquestaweb.WebNuevaAppIntakeGuidedTurnV0{
		Decisions: []orquestaweb.WebNuevaAppIntakeDecisionV0{
			{Field: "nombre", Value: "Turnos clinicos stack"},
			{Field: "tipo_app", Value: "web"},
			{Field: "integraciones.0.tipo", Value: "messaging"},
		},
	}, nil
}

type fakeCodexStackDomainWorkExecutorV0 struct {
	called int
	input  orquestamcp.MCPDomainWorkToolInputV0
	inputs []orquestamcp.MCPDomainWorkToolInputV0
}

func (executor *fakeCodexStackDomainWorkExecutorV0) Execute(
	_ context.Context,
	input orquestamcp.MCPDomainWorkToolInputV0,
) (orquestamcp.MCPDomainWorkToolResultV0, error) {
	executor.called++
	executor.input = input
	executor.inputs = append(executor.inputs, input)
	if input.Action == orquestamcp.MCPDomainWorkActionSubmitArtifactV0 {
		return orquestamcp.MCPDomainWorkToolResultV0{
			Estado:        orquestamcp.MCPDomainWorkEstadoOKV0,
			RequestID:     input.RequestID,
			CorrelationID: input.CorrelationID,
			Action:        input.Action,
			Receipt: &orquestadomainwork.DomainWorkArtifactReceiptV0{
				SchemaVersion: orquestadomainwork.DomainWorkArtifactReceiptSchemaV0,
				Status:        orquestadomainwork.DomainWorkStatusAcceptedV0,
				JobRef:        input.ArtifactSubmission.JobRef,
				ArtifactRef:   input.ArtifactSubmission.ArtifactRef,
				ReceiptRef:    "receipt-ref-" + input.ArtifactSubmission.ArtifactRef,
			},
		}, nil
	}
	return orquestamcp.MCPDomainWorkToolResultV0{
		Estado:        orquestamcp.MCPDomainWorkEstadoOKV0,
		RequestID:     input.RequestID,
		CorrelationID: input.CorrelationID,
		Action:        input.Action,
		Job: &orquestadomainwork.DomainWorkJobV0{
			SchemaVersion: orquestadomainwork.DomainWorkJobSchemaV0,
			Status:        orquestadomainwork.DomainWorkStatusAcceptedV0,
			JobRef:        "job-ref-codex-stack-domain-001",
			DomainRef:     input.JobRequest.DomainRef,
			WorkKind:      input.JobRequest.WorkKind,
		},
	}, nil
}
