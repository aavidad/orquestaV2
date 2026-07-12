package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaweb "orquesta/modulos/orquesta-web"
)

func TestWizardBotConfigDesdeFicheroCanonicoV0(t *testing.T) {
	configFile := `{"schema_version":"orquesta_config.v0","wizard_bot":{"llm_enabled":true,"model":"gpt-5.5","reasoning_effort":"high","daily_token_budget":1234}}`
	config := loadProjectConfigFromTextForTestV0(t, configFile)

	if !wizardBotLLMEnabledFromProjectConfigFileV0(config) ||
		wizardBotModelFromProjectConfigFileV0(config) != "gpt-5.5" ||
		wizardBotReasoningEffortFromProjectConfigFileV0(config) != "high" ||
		wizardBotDailyTokenBudgetFromProjectConfigFileV0(config) != 1234 {
		t.Fatalf("wizard_bot config inesperada")
	}
}

func TestWizardBotConfigApagadoPorDefectoYLowSubeAHighV0(t *testing.T) {
	config := loadProjectConfigFromTextForTestV0(t, `{"schema_version":"orquesta_config.v0","wizard_bot":{"reasoning_effort":"low"}}`)

	if wizardBotLLMEnabledFromProjectConfigFileV0(config) {
		t.Fatalf("wizard_bot llm debe estar apagado por defecto")
	}
	if got := wizardBotReasoningEffortFromProjectConfigFileV0(config); got != "high" {
		t.Fatalf("reasoning_effort=%q want high", got)
	}
}

func TestWizardBotLLMUsaAppServerConModeloYReasoningHighV0(t *testing.T) {
	protocol := &fakeWizardBotAppServerProtocolV0{
		read: serverCodexAppServerThreadReadV0{Turns: []serverCodexAppServerReadTurnV0{{
			Items: []serverCodexAppServerReadItemV0{{
				Text: `{"status":"ok","say":"Registrado desde catalogo","filled_answers":[{"question_ref":"wizard-t2-identidad-corporativa","user_choice":"ldap_bind"}],"grounding_refs":["wizard-corpus:question:wizard-t2-identidad-corporativa"]}`,
			}},
		}}},
	}
	assistant := serverWizardBotLLMAssistantV0{
		Config: serverWizardBotLLMConfigV0{
			Enabled:          true,
			Model:            "gpt-5.5",
			ReasoningEffort:  "high",
			DailyTokenBudget: 20000,
			ProjectWorkDir:   ".",
		},
		Protocol: protocol,
	}

	result, err := assistant.AssistWizardBotTurnV0(context.Background(), orquestaweb.WizardBotLLMRequestV0{
		UserText: "login corporativo",
		GroundingSnippets: []orquestaweb.WizardBotGroundingSnippetV0{{
			Ref:  "wizard-corpus:question:wizard-t2-identidad-corporativa",
			Kind: "question",
			Text: "Debe integrarse con identidad corporativa?",
		}},
		AllowedAnswers: []orquestaweb.WizardBotAllowedAnswerV0{{
			QuestionRef: "wizard-t2-identidad-corporativa",
			UserChoice:  "ldap_bind",
		}},
	})
	if err != nil {
		t.Fatalf("AssistWizardBotTurnV0: %v", err)
	}
	if result.Status != "ok" || len(result.FilledAnswers) != 1 {
		t.Fatalf("result inesperado: %+v", result)
	}
	if protocol.thread.Model != "gpt-5.5" || protocol.turn.Model != "gpt-5.5" || protocol.turn.Effort != "high" {
		t.Fatalf("modelo/reasoning no propagados: thread=%+v turn=%+v", protocol.thread, protocol.turn)
	}
	if strings.Contains(protocol.turn.Effort, "low") {
		t.Fatalf("reasoning low prohibido: %+v", protocol.turn)
	}
}

func TestWizardBotLLMDegradaSinProveedorOPresupuestoV0(t *testing.T) {
	noProvider := serverWizardBotLLMAssistantV0{Config: serverWizardBotLLMConfigV0{Enabled: true, DailyTokenBudget: 1}}
	result, err := noProvider.AssistWizardBotTurnV0(context.Background(), orquestaweb.WizardBotLLMRequestV0{})
	if err == nil || result.Status != "provider_failed" {
		t.Fatalf("sin proveedor result=%+v err=%v", result, err)
	}

	noBudget := serverWizardBotLLMAssistantV0{Config: serverWizardBotLLMConfigV0{Enabled: true, DailyTokenBudget: 0}, Protocol: &fakeWizardBotAppServerProtocolV0{}}
	result, err = noBudget.AssistWizardBotTurnV0(context.Background(), orquestaweb.WizardBotLLMRequestV0{})
	if err != nil || result.Status != "budget_exhausted" {
		t.Fatalf("sin presupuesto result=%+v err=%v", result, err)
	}
}

func loadProjectConfigFromTextForTestV0(t *testing.T, text string) serverProjectConfigFileV0 {
	t.Helper()
	path := filepath.Join(t.TempDir(), "orquesta.config.json")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	config, ok, err := loadServerProjectConfigPathV0(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !ok {
		t.Fatalf("config not loaded")
	}
	return config
}

type fakeWizardBotAppServerProtocolV0 struct {
	thread serverCodexAppServerThreadStartParamsV0
	turn   serverCodexAppServerTurnStartParamsV0
	read   serverCodexAppServerThreadReadV0
}

func (fake *fakeWizardBotAppServerProtocolV0) StartThreadV0(
	_ context.Context,
	params serverCodexAppServerThreadStartParamsV0,
) (serverCodexAppServerThreadV0, error) {
	fake.thread = params
	return serverCodexAppServerThreadV0{ID: "thread-wizard-bot-001"}, nil
}

func (fake *fakeWizardBotAppServerProtocolV0) UpdateThreadSettingsV0(
	context.Context,
	serverCodexAppServerThreadSettingsUpdateParamsV0,
) error {
	return nil
}

func (fake *fakeWizardBotAppServerProtocolV0) SetGoalV0(
	context.Context,
	serverCodexAppServerThreadGoalSetParamsV0,
) (serverCodexAppServerThreadGoalV0, error) {
	return serverCodexAppServerThreadGoalV0{}, nil
}

func (fake *fakeWizardBotAppServerProtocolV0) StartTurnV0(
	_ context.Context,
	params serverCodexAppServerTurnStartParamsV0,
) (serverCodexAppServerTurnV0, error) {
	fake.turn = params
	return serverCodexAppServerTurnV0{ID: "turn-wizard-bot-001", Status: "completed"}, nil
}

func (fake *fakeWizardBotAppServerProtocolV0) GetGoalV0(
	context.Context,
	string,
) (*serverCodexAppServerThreadGoalV0, error) {
	return nil, nil
}

func (fake *fakeWizardBotAppServerProtocolV0) ReadThreadV0(
	context.Context,
	string,
	bool,
) (serverCodexAppServerThreadReadV0, error) {
	return fake.read, nil
}

// El modelo por defecto del wizard bot debe ser gpt-5.6: sin este guard, el
// default puede derivar en silencio a una familia de modelos retirada.
func TestWizardBotModeloPorDefectoEsGPT56V0(t *testing.T) {
	if model := wizardBotModelFromProjectConfigFileV0(serverProjectConfigFileV0{}); model != "gpt-5.6" {
		t.Fatalf("modelo por defecto del wizard bot=%q, esperado gpt-5.6", model)
	}
}
