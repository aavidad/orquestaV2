package main

import (
	orquestatelegram "orquesta/modulos/orquesta-operator-telegram"
)

type telegramOperatorWiringResultV0 struct {
	Enabled       bool
	Ready         bool
	BlockedCode   string
	MissingFields []string
	Adapter       orquestatelegram.AdapterV0
}

func telegramOperatorAdapterFromProjectConfigFileV0(
	config serverProjectConfigFileV0,
	ports orquestatelegram.OperatorPortsV0,
) telegramOperatorWiringResultV0 {
	telegramConfig := telegramOperatorConfigFromProjectConfigFileV0(config)
	if !telegramConfig.Enabled {
		return telegramOperatorWiringResultV0{}
	}
	adapter, issues := orquestatelegram.NewAdapterV0(telegramConfig, ports)
	if len(issues) == 0 {
		return telegramOperatorWiringResultV0{Enabled: true, Ready: true, Adapter: adapter}
	}
	return telegramOperatorWiringResultV0{
		Enabled:       true,
		Ready:         false,
		BlockedCode:   orquestatelegram.ErrTelegramLinkMissingV0,
		MissingFields: telegramOperatorMissingFieldsV0(issues),
	}
}

func telegramOperatorMissingFieldsV0(issues []orquestatelegram.IssueV0) []string {
	out := make([]string, 0, len(issues))
	for _, issue := range issues {
		if issue.Code != orquestatelegram.ErrTelegramLinkMissingV0 || issue.Field == "" {
			continue
		}
		out = append(out, issue.Field)
	}
	return out
}
