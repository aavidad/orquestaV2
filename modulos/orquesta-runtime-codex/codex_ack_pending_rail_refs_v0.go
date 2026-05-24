package orquestaruntimecodex

import "strings"

func CodexAgentAckPendingRailEvidenceRefsV0(ack CodexAgentAckV0) []string {
	if !CodexAgentAckHasPendingRailV0(ack) {
		return nil
	}
	refs := []string{CodexAgentAckPendingRailEvidenceRefV0}
	for _, category := range codexAckPendingRailCategoriesV0(ack) {
		refs = append(refs, "ack-pending-rail:"+category)
	}
	return compactCodexDeliveryObservationRefsV0(refs)
}

func codexAckPendingRailCategoriesV0(ack CodexAgentAckV0) []string {
	categories := []string{}
	categories = append(categories, codexAckPendingRailCategoriesForFieldValuesV0(ack.Files)...)
	categories = append(categories, codexAckPendingRailCategoriesForFieldValuesV0(ack.Tests)...)
	categories = append(categories, codexAckPendingRailCategoriesForNotesV0(ack.Notes)...)
	return compactCodexDeliveryObservationRefsV0(categories)
}

func codexAckPendingRailCategoriesForFieldValuesV0(values []string) []string {
	categories := []string{}
	for _, value := range values {
		if !codexAckFieldValueHasPendingRailV0(value) {
			continue
		}
		categories = append(categories, codexAckPendingRailCategoriesForTextV0(value)...)
	}
	return categories
}

func codexAckPendingRailCategoriesForNotesV0(values []string) []string {
	categories := []string{}
	for _, value := range values {
		if !codexAckValueHasPendingRailV0(value) {
			continue
		}
		categories = append(categories, codexAckPendingRailCategoriesForTextV0(value)...)
	}
	return categories
}

func codexAckPendingRailCategoriesForTextV0(value string) []string {
	lower := strings.ToLower(strings.ReplaceAll(value, `\/`, "/"))
	categories := []string{}
	for _, rule := range codexAckPendingRailCategoryRulesV0() {
		if codexAckPendingRailTextMatchesAnyV0(lower, rule.markers) {
			categories = append(categories, rule.category)
		}
	}
	return categories
}

type codexAckPendingRailCategoryRuleV0 struct {
	category string
	markers  []string
}

func codexAckPendingRailCategoryRulesV0() []codexAckPendingRailCategoryRuleV0 {
	return []codexAckPendingRailCategoryRuleV0{
		{"token", []string{"token", "access_token", "access-token", "refresh_token", "refresh-token", "authorization", "bearer"}},
		{"secret", []string{"secret", "secreto", "client_secret", "client-secret", "private key"}},
		{"credential", []string{"credential", "credencial", "password", "passwd", "api_key", "api-key", "apikey"}},
		{"prompt", []string{"prompt"}},
		{"completion", []string{"completion"}},
		{"transcript", []string{"transcript"}},
		{"home", []string{"home", "code_home", "codex_home", "/home/", "/users/", "$home", "~/"}},
		{"provider", []string{"provider", "proveedor"}},
		{"model", []string{"model", "modelo"}},
		{"oauth", []string{"oauth"}},
		{"runtime", []string{"runtime", "codex", "claude", "ollama", "vllm"}},
		{"storage", []string{"db", "database", "sql", "dsn", "filesystem", "postgres", "sqlite"}},
		{"tooling", []string{"git", "docker", "tmux"}},
		{"adapter", []string{"adapter", "adaptador"}},
	}
}

func codexAckPendingRailTextMatchesAnyV0(lower string, markers []string) bool {
	for _, marker := range markers {
		if codexDeliveryObservationContainsFragmentV0(lower, marker) {
			return true
		}
	}
	return false
}
