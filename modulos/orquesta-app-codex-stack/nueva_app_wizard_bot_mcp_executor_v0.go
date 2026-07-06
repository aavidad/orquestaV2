package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaweb "orquesta/modulos/orquesta-web"
)

type CodexStackNuevaAppWizardBotExecutorV0 struct {
	Assistant orquestaweb.WizardBotLLMAssistPortV0
}

func NewCodexStackNuevaAppWizardBotExecutorV0(
	assistant orquestaweb.WizardBotLLMAssistPortV0,
) CodexStackNuevaAppWizardBotExecutorV0 {
	return CodexStackNuevaAppWizardBotExecutorV0{Assistant: assistant}
}

func (executor CodexStackNuevaAppWizardBotExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPNuevaAppWizardBotToolInputV0,
) (orquestamcp.MCPNuevaAppWizardBotToolResultV0, error) {
	input = orquestamcp.NormalizeMCPNuevaAppWizardBotInputV0(input)
	session, err := codexStackWizardBotSessionFromMCPInputV0(input)
	if err != nil {
		return orquestamcp.NewMCPNuevaAppWizardBotErrorResultV0(input, "nueva_app_wizard_bot_session_invalida", "session"), nil
	}
	session, reply := orquestaweb.NewWebNuevaAppWizardBotReplyWithLLMV0(ctx, session, orquestaweb.WizardBotTurnV0{
		SessionRef: input.SessionRef,
		UserText:   input.UserText,
		Locale:     input.Locale,
	}, executor.Assistant)
	rawReply, err := json.Marshal(reply)
	if err != nil {
		return orquestamcp.MCPNuevaAppWizardBotToolResultV0{}, err
	}
	rawSession, err := json.Marshal(session)
	if err != nil {
		return orquestamcp.MCPNuevaAppWizardBotToolResultV0{}, err
	}
	return orquestamcp.NormalizeMCPNuevaAppWizardBotResultV0(input, orquestamcp.MCPNuevaAppWizardBotToolResultV0{
		Estado:  orquestamcp.MCPNuevaAppWizardBotEstadoOKV0,
		Reply:   rawReply,
		Session: rawSession,
	}), nil
}

func codexStackWizardBotSessionFromMCPInputV0(
	input orquestamcp.MCPNuevaAppWizardBotToolInputV0,
) (orquestaweb.WebNuevaAppIntakeSessionV0, error) {
	if len(input.Session) > 0 && string(input.Session) != "null" {
		var session orquestaweb.WebNuevaAppIntakeSessionV0
		if err := json.Unmarshal(input.Session, &session); err != nil {
			return orquestaweb.WebNuevaAppIntakeSessionV0{}, err
		}
		return codexStackNormalizeWizardBotSessionV0(session, input), nil
	}
	return codexStackNormalizeWizardBotSessionV0(
		orquestaweb.NewWebNuevaAppIntakeSessionV0(input.SessionRef, input.Locale, "", ""),
		input,
	), nil
}

func codexStackNormalizeWizardBotSessionV0(
	session orquestaweb.WebNuevaAppIntakeSessionV0,
	input orquestamcp.MCPNuevaAppWizardBotToolInputV0,
) orquestaweb.WebNuevaAppIntakeSessionV0 {
	if strings.TrimSpace(session.SessionRef) == "" {
		session.SessionRef = input.SessionRef
	}
	if strings.TrimSpace(session.SessionID) == "" {
		session.SessionID = input.SessionRef
	}
	if strings.TrimSpace(input.Locale) != "" {
		session.Form.Locale = input.Locale
		session.Locale = input.Locale
	}
	return session
}
