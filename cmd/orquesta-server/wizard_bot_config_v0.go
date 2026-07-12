package main

import (
	"strings"
)

type serverProjectConfigWizardBotV0 struct {
	LLMEnabled       *bool   `json:"llm_enabled,omitempty"`
	Model            *string `json:"model,omitempty"`
	ReasoningEffort  *string `json:"reasoning_effort,omitempty"`
	DailyTokenBudget *int    `json:"daily_token_budget,omitempty"`
}

type serverProjectConfigServerResidentV0 struct {
	MaxActions *int `json:"max_actions,omitempty"`
}

func wizardBotLLMEnabledFromProjectConfigFileV0(fileConfig serverProjectConfigFileV0) bool {
	if fileConfig.WizardBot.LLMEnabled == nil {
		return false
	}
	return *fileConfig.WizardBot.LLMEnabled
}

func wizardBotModelFromProjectConfigFileV0(fileConfig serverProjectConfigFileV0) string {
	if fileConfig.WizardBot.Model != nil {
		if value := strings.TrimSpace(*fileConfig.WizardBot.Model); value != "" {
			return value
		}
	}
	return "gpt-5.6"
}

func wizardBotReasoningEffortFromProjectConfigFileV0(fileConfig serverProjectConfigFileV0) string {
	if fileConfig.WizardBot.ReasoningEffort != nil {
		value := strings.ToLower(strings.TrimSpace(*fileConfig.WizardBot.ReasoningEffort))
		switch value {
		case "medium", "high", "xhigh":
			return value
		}
	}
	return "high"
}

func wizardBotDailyTokenBudgetFromProjectConfigFileV0(fileConfig serverProjectConfigFileV0) int {
	if fileConfig.WizardBot.DailyTokenBudget != nil && *fileConfig.WizardBot.DailyTokenBudget > 0 {
		return *fileConfig.WizardBot.DailyTokenBudget
	}
	return 20000
}
