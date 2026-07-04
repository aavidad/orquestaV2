package orquestamcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestMCPNuevaAppWizardDescriptorV0EsAdaptadorFino(t *testing.T) {
	descriptor := MCPNuevaAppWizardDescriptorV0()

	if descriptor.Name != MCPNuevaAppWizardToolNameV0 ||
		descriptor.Version != MCPNuevaAppWizardToolVersionV0 ||
		descriptor.ResourceURI != MCPNuevaAppWizardResourceURIV0 ||
		!strings.Contains(descriptor.InputSchema, "wizard_answers") ||
		!strings.Contains(descriptor.Output, "wizard") {
		t.Fatalf("descriptor inesperado: %+v", descriptor)
	}
	payload, err := json.Marshal(descriptor)
	if err != nil {
		t.Fatalf("marshal descriptor: %v", err)
	}
	text := string(payload)
	for _, forbidden := range []string{"runtime", "filesystem", "DB", "HOME"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("descriptor filtra detalle prohibido %q: %s", forbidden, text)
		}
	}
}

func TestNormalizeMCPNuevaAppWizardInputV0DerivaIdentidadYRecorta(t *testing.T) {
	input := NormalizeMCPNuevaAppWizardInputV0(MCPNuevaAppWizardToolInputV0{
		RequestID:     " request-ref-wizard-001 ",
		CorrelationID: " corr-wizard-001 ",
		SessionID:     " session-wizard-001 ",
		Need:          " quiero una app para una agenda ",
		ActionIDs:     []string{" review ", "", " review "},
		WizardAnswers: []MCPNuevaAppWizardAnswerV0{{
			QuestionRef: " wizard-r2-plataformas ",
			UserChoice:  " web ",
		}},
	})

	if input.RequestID != "request-ref-wizard-001" ||
		input.CorrelationID != "corr-wizard-001" ||
		input.SessionID != "session-wizard-001" ||
		input.Need != "quiero una app para una agenda" ||
		len(input.ActionIDs) != 1 ||
		input.WizardAnswers[0].QuestionRef != "wizard-r2-plataformas" ||
		input.WizardAnswers[0].UserChoice != "web" {
		t.Fatalf("input normalizado inesperado: %+v", input)
	}
}

func TestNewMCPNuevaAppWizardResultFromJSONV0ConservaWizardYContrastes(t *testing.T) {
	raw := []byte(`{
		"schema_version":"web_nueva_app_intake_guided_response.v0",
		"turn":{"schema_version":"web_nueva_app_intake_guided_turn.v0"},
		"session":{"session_id":"session-wizard-json"},
		"wizard":{
			"schema_version":"web_nueva_app_wizard_turn.v0",
			"questions":[{"question_ref":"wizard-r2-plataformas"}],
			"contrasts":[{"question_ref":"wizard-r2-plataformas","user_choice":"web","recommended":"web_mobile"}],
			"engineering_defaults":[{"area":"arquitectura","value":"hexagonal_puertos_adaptadores"}]
		}
	}`)

	result, err := NewMCPNuevaAppWizardResultFromJSONV0(MCPNuevaAppWizardToolInputV0{
		RequestID:     "request-ref-wizard-json",
		CorrelationID: "corr-wizard-json",
	}, raw)
	if err != nil {
		t.Fatalf("result from json: %v", err)
	}
	if result.Estado != MCPNuevaAppWizardEstadoOKV0 ||
		result.RequestID != "request-ref-wizard-json" ||
		result.CorrelationID != "corr-wizard-json" ||
		!strings.Contains(string(result.Wizard), "web_mobile") ||
		!strings.Contains(string(result.Session), "session-wizard-json") {
		t.Fatalf("result=%+v wizard=%s", result, result.Wizard)
	}
}

func TestMCPNuevaAppWizardTransportV0DelegaEnPuerto(t *testing.T) {
	fake := &fakeMCPNuevaAppWizardPortV0{}
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{NuevaAppWizard: fake}); err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(context.Background(), MCPNuevaAppWizardToolNameV0, MCPNuevaAppWizardToolInputV0{
		RequestID: "request-ref-wizard-transport",
		Need:      "quiero una app para una agenda",
		WizardAnswers: []MCPNuevaAppWizardAnswerV0{{
			QuestionRef: "wizard-r2-plataformas",
			UserChoice:  "web",
		}},
	})
	if err != nil {
		t.Fatalf("call wizard: %v", err)
	}
	if fake.input.Need != "quiero una app para una agenda" ||
		len(fake.input.WizardAnswers) != 1 ||
		fake.input.WizardAnswers[0].UserChoice != "web" {
		t.Fatalf("puerto no recibio input normalizado: %+v", fake.input)
	}
	var result MCPNuevaAppWizardToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != MCPNuevaAppWizardEstadoOKV0 ||
		!strings.Contains(string(result.Wizard), "wizard-r2-plataformas") {
		t.Fatalf("result=%+v wizard=%s", result, result.Wizard)
	}
}

type fakeMCPNuevaAppWizardPortV0 struct {
	input MCPNuevaAppWizardToolInputV0
}

func (fake *fakeMCPNuevaAppWizardPortV0) Execute(
	_ context.Context,
	input MCPNuevaAppWizardToolInputV0,
) (MCPNuevaAppWizardToolResultV0, error) {
	fake.input = input
	return MCPNuevaAppWizardToolResultV0{
		Estado:        MCPNuevaAppWizardEstadoOKV0,
		RequestID:     input.RequestID,
		CorrelationID: input.CorrelationID,
		Wizard:        json.RawMessage(`{"questions":[{"question_ref":"wizard-r2-plataformas"}]}`),
		Session:       json.RawMessage(`{"session_id":"session-wizard-transport"}`),
		Turn:          json.RawMessage(`{"messages":[]}`),
	}, nil
}
