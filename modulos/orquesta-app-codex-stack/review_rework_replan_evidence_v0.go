package orquestaappcodexstack

import (
	"crypto/sha1"
	"fmt"
	"strings"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func reviewReworkReplanEvidenceRefsV0(
	request orquestacionnucleoapp.ReviewReworkReplanPlanRequestV0,
	rework reviewReworkProjectionV0,
	result reviewResultProjectionV0,
	fallback bool,
	missingTargets []string,
) []string {
	refs := []string{
		"evidence-ref-review-rework-replan",
		rework.ReworkRequestRef,
		result.ReviewResultRef,
		rework.DeliveryRef,
	}
	if fallback {
		refs = append(refs, "evidence-ref-review-rework-task-fallback")
	}
	for _, target := range reviewReworkTargetEvidenceRefsV0(missingTargets) {
		refs = append(refs, target)
	}
	return compactStringsV0(append(refs, reviewReworkReplanNeutralEvidenceRefsV0(request.EvidenceRefs)...))
}

func reviewReworkReplanSummaryV0(missingTargets []string) string {
	missingTargets = compactStringsV0(missingTargets)
	if len(missingTargets) == 0 {
		return "Repetir tarea tras revision no aceptada."
	}
	return "Corregir entrega tras revision; conservar lo valido y completar faltantes: " +
		strings.Join(reviewReworkSummaryTargetsV0(missingTargets), ", ") + "."
}

func reviewReworkReplanAgentSuffixV0(
	rework reviewReworkProjectionV0,
	taskRef string,
) string {
	sum := sha1.Sum([]byte(strings.Join([]string{
		rework.ReworkRequestRef,
		rework.ReviewResultRef,
		rework.ReviewRequestID,
		rework.DeliveryRef,
		taskRef,
	}, "\x00")))
	return reviewReworkReplanAgentStemV0(taskRef) + "-" + fmt.Sprintf("%x", sum[:])[:12]
}

func reviewReworkReplanTaskRetryAgentPrefixV0(taskRef string) string {
	return "agent-ref-" + reviewReworkReplanAgentStemV0(taskRef) + "-"
}

func reviewReworkReplanAgentStemV0(taskRef string) string {
	stem := reviewReworkReplanSafeRefV0(taskRef)
	if stem == "" || stem == "sin-ref" {
		stem = "review-rework"
	}
	if len(stem) > 48 {
		stem = stem[:48]
	}
	return stem
}

func reviewReworkSummaryTargetsV0(targets []string) []string {
	targets = compactStringsV0(targets)
	if len(targets) <= 6 {
		return targets
	}
	return append(append([]string(nil), targets[:6]...), "mas")
}

func reviewReworkTargetEvidenceRefsV0(targets []string) []string {
	targets = compactStringsV0(targets)
	refs := make([]string, 0, len(targets))
	for index, target := range targets {
		if index >= 8 {
			break
		}
		refs = append(refs, "review-rework-missing-"+reviewReworkReplanSafeOpaqueTargetV0(target))
	}
	return refs
}

func reviewReworkReplanSafeRefV0(value string) string {
	value = strings.TrimSpace(value)
	replacer := strings.NewReplacer("\\", "-", "/", "-", " ", "-", "#", "-", ":", "-")
	value = strings.Trim(replacer.Replace(value), "-")
	if value == "" {
		return "sin-ref"
	}
	return value
}

func reviewReworkReplanNeutralEvidenceRefsV0(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range compactStringsV0(values) {
		if reviewReworkReplanEvidenceRefIsNeutralV0(value) {
			out = append(out, value)
		}
	}
	return out
}

func reviewReworkReplanEvidenceRefIsNeutralV0(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return false
	}
	for _, token := range reviewReworkReplanForbiddenEvidenceTokensV0() {
		if reviewReworkReplanHasEvidenceTokenV0(lower, token) {
			return false
		}
	}
	return true
}

func reviewReworkReplanForbiddenEvidenceTokensV0() []string {
	return []string{
		"db", "database", "sql", "dsn",
		"runtime", "provider", "proveedor", "model", "modelo",
		"home", "oauth", "codex", "claude", "ollama", "vllm",
		"adapter", "adaptador", "filesystem", "git", "docker", "tmux",
		"secret", "secreto", "token", "password", "credential", "credencial", "api_key",
	}
}

func reviewReworkReplanHasEvidenceTokenV0(value string, token string) bool {
	token = strings.ToLower(strings.TrimSpace(token))
	if token == "" {
		return false
	}
	start := 0
	for {
		index := strings.Index(value[start:], token)
		if index < 0 {
			return false
		}
		absolute := start + index
		if reviewReworkReplanTokenBoundaryV0(value, absolute, absolute+len(token)) {
			return true
		}
		start = absolute + len(token)
	}
}

func reviewReworkReplanTokenBoundaryV0(value string, start int, end int) bool {
	return (start == 0 || !reviewReworkReplanTokenCharV0(value[start-1])) &&
		(end >= len(value) || !reviewReworkReplanTokenCharV0(value[end]))
}

func reviewReworkReplanTokenCharV0(value byte) bool {
	return (value >= 'a' && value <= 'z') ||
		(value >= '0' && value <= '9') ||
		value == '_'
}
