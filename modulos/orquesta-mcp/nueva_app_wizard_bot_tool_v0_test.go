package orquestamcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestMCPNuevaAppWizardBotDescriptorV0EsAdaptadorFino(t *testing.T) {
	descriptor := MCPNuevaAppWizardBotDescriptorV0()

	if descriptor.Name != MCPNuevaAppWizardBotToolNameV0 ||
		descriptor.Version != MCPNuevaAppWizardBotToolVersionV0 ||
		descriptor.ResourceURI != MCPNuevaAppWizardBotResourceURIV0 ||
		!strings.Contains(descriptor.InputSchema, "user_text") ||
		!strings.Contains(descriptor.InputSchema, "session") ||
		!strings.Contains(descriptor.Output, "reply") {
		t.Fatalf("descriptor bot inesperado: %+v", descriptor)
	}
	payload, err := json.Marshal(descriptor)
	if err != nil {
		t.Fatalf("marshal descriptor: %v", err)
	}
	text := string(payload)
	for _, forbidden := range []string{"filesystem", "DB", "HOME", "token"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("descriptor filtra detalle prohibido %q: %s", forbidden, text)
		}
	}
}

func TestNormalizeMCPNuevaAppWizardBotInputV0DerivaIdentidadYRecorta(t *testing.T) {
	input := NormalizeMCPNuevaAppWizardBotInputV0(MCPNuevaAppWizardBotToolInputV0{
		RequestID:     " request-ref-wizard-bot-001 ",
		CorrelationID: " corr-wizard-bot-001 ",
		SessionRef:    " session-wizard-bot-001 ",
		UserText:      " que es CalDAV? ",
		Locale:        " es-ES ",
	})

	if input.RequestID != "request-ref-wizard-bot-001" ||
		input.CorrelationID != "corr-wizard-bot-001" ||
		input.SessionRef != "session-wizard-bot-001" ||
		input.UserText != "que es CalDAV?" ||
		input.Locale != "es-ES" {
		t.Fatalf("input bot normalizado inesperado: %+v", input)
	}
}

func TestMCPNuevaAppWizardBotTransportV0DelegaEnPuerto(t *testing.T) {
	fake := &fakeMCPNuevaAppWizardBotPortV0{}
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{NuevaAppWizardBot: fake}); err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(context.Background(), MCPNuevaAppWizardBotToolNameV0, MCPNuevaAppWizardBotToolInputV0{
		RequestID:  "request-ref-wizard-bot-transport",
		SessionRef: "session-wizard-bot-transport",
		UserText:   "que es web?",
		Locale:     "es-ES",
	})
	if err != nil {
		t.Fatalf("call wizard bot: %v", err)
	}
	if fake.input.UserText != "que es web?" || fake.input.SessionRef != "session-wizard-bot-transport" {
		t.Fatalf("puerto bot no recibio input normalizado: %+v", fake.input)
	}
	var result MCPNuevaAppWizardBotToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != MCPNuevaAppWizardBotEstadoOKV0 ||
		!strings.Contains(string(result.Reply), "wizard_bot_reply") ||
		!strings.Contains(string(result.Session), "session-wizard-bot-transport") {
		t.Fatalf("result=%+v reply=%s session=%s", result, result.Reply, result.Session)
	}
}

type fakeMCPNuevaAppWizardBotPortV0 struct {
	input MCPNuevaAppWizardBotToolInputV0
}

func (fake *fakeMCPNuevaAppWizardBotPortV0) Execute(
	_ context.Context,
	input MCPNuevaAppWizardBotToolInputV0,
) (MCPNuevaAppWizardBotToolResultV0, error) {
	fake.input = input
	return MCPNuevaAppWizardBotToolResultV0{
		Estado:  MCPNuevaAppWizardBotEstadoOKV0,
		Reply:   json.RawMessage(`{"schema_version":"web_nueva_app_wizard_bot_reply.v0","say":"ok"}`),
		Session: json.RawMessage(`{"session_ref":"session-wizard-bot-transport"}`),
	}, nil
}
