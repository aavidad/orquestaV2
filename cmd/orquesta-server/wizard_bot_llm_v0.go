package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	orquestaweb "orquesta/modulos/orquesta-web"
)

type serverWizardBotLLMConfigV0 struct {
	Enabled          bool
	Model            string
	ReasoningEffort  string
	DailyTokenBudget int
	ProjectWorkDir   string
}

type serverWizardBotLLMAssistantV0 struct {
	Config   serverWizardBotLLMConfigV0
	Protocol serverCodexAppServerProtocolPortV0
}

func serverWizardBotLLMAssistantFromConfigV0(
	serverConfigProjectWorkDir string,
	projectConfig serverProjectConfigFileV0,
	goalBackend serverCodexGoalBackendV0,
) orquestaweb.WizardBotLLMAssistPortV0 {
	config := serverWizardBotLLMConfigV0{
		Enabled:          wizardBotLLMEnabledFromProjectConfigFileV0(projectConfig),
		Model:            wizardBotModelFromProjectConfigFileV0(projectConfig),
		ReasoningEffort:  wizardBotReasoningEffortFromProjectConfigFileV0(projectConfig),
		DailyTokenBudget: wizardBotDailyTokenBudgetFromProjectConfigFileV0(projectConfig),
		ProjectWorkDir:   strings.TrimSpace(serverConfigProjectWorkDir),
	}
	if !config.Enabled {
		return nil
	}
	return serverWizardBotLLMAssistantV0{
		Config:   config,
		Protocol: serverWizardBotProtocolFromGoalBackendV0(goalBackend),
	}
}

func serverWizardBotProtocolFromGoalBackendV0(
	goalBackend serverCodexGoalBackendV0,
) serverCodexAppServerProtocolPortV0 {
	if starter, ok := goalBackend.Starter.(serverCodexGoalCostRoutingStarterV0); ok {
		return starter.Backend.Protocol
	}
	if backend, ok := goalBackend.Starter.(serverCodexAppServerGoalBackendV0); ok {
		return backend.Protocol
	}
	if backend, ok := goalBackend.Observer.(serverCodexAppServerGoalBackendV0); ok {
		return backend.Protocol
	}
	return nil
}

func (assistant serverWizardBotLLMAssistantV0) AssistWizardBotTurnV0(
	ctx context.Context,
	request orquestaweb.WizardBotLLMRequestV0,
) (orquestaweb.WizardBotLLMResultV0, error) {
	if !assistant.Config.Enabled {
		return orquestaweb.WizardBotLLMResultV0{Status: "disabled"}, nil
	}
	if assistant.Config.DailyTokenBudget <= 0 {
		return orquestaweb.WizardBotLLMResultV0{Status: "budget_exhausted"}, nil
	}
	if assistant.Protocol == nil {
		return orquestaweb.WizardBotLLMResultV0{Status: "provider_failed"}, errors.New("wizard_bot_provider_unavailable")
	}
	thread, err := assistant.Protocol.StartThreadV0(ctx, serverCodexAppServerThreadStartParamsV0{
		CWD:            assistant.Config.ProjectWorkDir,
		Ephemeral:      true,
		Model:          assistant.Config.Model,
		Sandbox:        "read-only",
		ApprovalPolicy: "never",
	})
	if err != nil {
		return orquestaweb.WizardBotLLMResultV0{Status: "provider_failed"}, err
	}
	threadID := strings.TrimSpace(thread.ID)
	if threadID == "" {
		return orquestaweb.WizardBotLLMResultV0{Status: "provider_failed"}, errors.New("wizard_bot_thread_id_missing")
	}
	if _, err := assistant.Protocol.StartTurnV0(ctx, serverCodexAppServerTurnStartParamsV0{
		ThreadID:       threadID,
		CWD:            assistant.Config.ProjectWorkDir,
		InputText:      serverWizardBotPromptV0(request),
		Model:          assistant.Config.Model,
		Effort:         assistant.Config.ReasoningEffort,
		ApprovalPolicy: "never",
	}); err != nil {
		return orquestaweb.WizardBotLLMResultV0{Status: "provider_failed"}, err
	}
	threadRead, err := assistant.Protocol.ReadThreadV0(ctx, threadID, true)
	if err != nil {
		return orquestaweb.WizardBotLLMResultV0{Status: "provider_failed"}, err
	}
	result, ok := serverWizardBotResultFromThreadV0(threadRead)
	if !ok {
		return orquestaweb.WizardBotLLMResultV0{Status: "provider_failed"}, errors.New("wizard_bot_result_missing")
	}
	if strings.TrimSpace(result.Status) == "" {
		result.Status = "ok"
	}
	result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-wizard-bot-llm-app-server")
	return result, nil
}

func serverWizardBotPromptV0(request orquestaweb.WizardBotLLMRequestV0) string {
	payload, _ := json.Marshal(struct {
		UserText          string                                    `json:"user_text"`
		Locale            string                                    `json:"locale,omitempty"`
		GroundingSnippets []orquestaweb.WizardBotGroundingSnippetV0 `json:"grounding_snippets"`
		AllowedAnswers    []orquestaweb.WizardBotAllowedAnswerV0    `json:"allowed_answers"`
	}{
		UserText:          request.UserText,
		Locale:            request.Locale,
		GroundingSnippets: request.GroundingSnippets,
		AllowedAnswers:    request.AllowedAnswers,
	})
	var b strings.Builder
	b.WriteString("Eres la capa LLM opt-in del bot del wizard de Orquesta. ")
	b.WriteString("Usa SOLO grounding_snippets como base. No inventes preguntas ni opciones. ")
	b.WriteString("Devuelve exclusivamente JSON con status, say, filled_answers y grounding_refs. ")
	b.WriteString("filled_answers solo puede contener pares presentes en allowed_answers. ")
	b.WriteString("Si el corpus no cubre la duda, status ok, filled_answers vacio y say honesto. ")
	b.WriteString("Payload: ")
	b.Write(payload)
	return b.String()
}

func serverWizardBotResultFromThreadV0(
	thread serverCodexAppServerThreadReadV0,
) (orquestaweb.WizardBotLLMResultV0, bool) {
	for i := len(thread.Turns) - 1; i >= 0; i-- {
		turn := thread.Turns[i]
		for j := len(turn.Items) - 1; j >= 0; j-- {
			text := strings.TrimSpace(turn.Items[j].Text)
			if text == "" {
				continue
			}
			var result orquestaweb.WizardBotLLMResultV0
			if err := json.Unmarshal([]byte(serverWizardBotJSONPayloadV0(text)), &result); err == nil {
				return result, true
			}
		}
	}
	return orquestaweb.WizardBotLLMResultV0{}, false
}

func serverWizardBotJSONPayloadV0(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
	}
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end >= start {
		return text[start : end+1]
	}
	return text
}
