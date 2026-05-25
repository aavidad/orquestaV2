package main

import (
	"strconv"
	"strings"

	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
)

func codexDirectorWriteRecursiveConfigV0(
	b *strings.Builder,
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
) {
	b.WriteString("Delegacion recursiva:\n")
	if !plan.RecursiveDelegation {
		b.WriteString("- recursive_delegation: false\n\n")
		return
	}
	b.WriteString("- recursive_delegation: true\n")
	b.WriteString("- max_delegation_depth: " + strconv.Itoa(plan.MaxDelegationDepth) + "\n")
	b.WriteString("- max_subagents_per_agent: " + strconv.Itoa(plan.MaxSubagentsPerAgent) + "\n")
	b.WriteString("- Orquesta materializa los subagentes autorizados con parent_agent_ref; no improvises hijos fuera de esos limites.\n\n")
}

func codexDirectorWriteDomainContextV0(
	b *strings.Builder,
	blocks []codexDirectorDomainContextBlockV0,
) {
	if len(blocks) == 0 {
		return
	}
	b.WriteString("Contexto de dominio inyectado por adaptador:\n")
	for _, block := range blocks {
		if strings.TrimSpace(block.Text) == "" {
			continue
		}
		sourceRef := strings.TrimSpace(block.SourceRef)
		if sourceRef == "" {
			sourceRef = "domain-context"
		}
		b.WriteString("- source_ref: " + sourceRef + "\n")
		b.WriteString(block.Text + "\n")
	}
	b.WriteString("\n")
}

func codexDirectorLaunchItemV0(
	work orquestadirectoroperativo.OperationalDirectorWaveWorkV0,
) orquestadirectoroperativo.OperationalDirectorWorkItemV0 {
	for _, wave := range work.Waves {
		for _, item := range wave.Items {
			if item.Kind == orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0 {
				return item
			}
		}
	}
	return orquestadirectoroperativo.OperationalDirectorWorkItemV0{}
}

func codexDirectorWaveRefForItemV0(
	work orquestadirectoroperativo.OperationalDirectorWaveWorkV0,
	itemRef string,
) string {
	for _, wave := range work.Waves {
		for _, item := range wave.Items {
			if item.ItemID == itemRef {
				return wave.WaveID
			}
		}
	}
	return ""
}

func codexDirectorShardV0(values []string, index int, total int) []string {
	if total <= 0 || index <= 0 {
		return nil
	}
	if codexDirectorWriteSetCoversWorkspaceV0(values) {
		return append([]string(nil), values...)
	}
	out := make([]string, 0)
	for i, value := range values {
		if i%total == index-1 {
			out = append(out, value)
		}
	}
	return out
}

func codexDirectorWriteSetCoversWorkspaceV0(values []string) bool {
	for _, value := range values {
		switch strings.TrimSpace(value) {
		case ".", "workspace":
			return true
		}
	}
	return false
}

func codexDirectorWorktreeRefV0(value string, waveRef string, strict bool) string {
	value = strings.TrimSpace(value)
	if value != "" || strict {
		return value
	}
	return codexDirectorDefaultRefV0(value, "worktree", waveRef)
}

func codexDirectorGuardOptInSummaryFromConfigV0(config codexDirectorWaveConfigV0) codexDirectorGuardOptInSummaryV0 {
	if !config.AllowGlobalWriteSet && !config.AllowPlaceholderTests {
		return codexDirectorGuardOptInSummaryV0{}
	}
	return codexDirectorGuardOptInSummaryV0{
		AllowGlobalWriteSet:      config.AllowGlobalWriteSet,
		AllowPlaceholderTests:    config.AllowPlaceholderTests,
		Reason:                   config.GuardOverrideReason,
		EvidenceRefs:             append([]string(nil), config.GuardOverrideEvidenceRefs...),
		NoDeleteGuard:            true,
		ProjectBoundaryGuard:     true,
		DirectorDecisionRequired: true,
	}
}

func codexDirectorStrictGuardIssuesV0(
	config codexDirectorWaveConfigV0,
) []orquestadirectoroperativo.OperationalDirectorIssueV0 {
	if !config.StrictDirectorGuards {
		return nil
	}
	var issues []orquestadirectoroperativo.OperationalDirectorIssueV0
	if codexDirectorWriteSetCoversWorkspaceV0(config.WriteSet) && !config.AllowGlobalWriteSet {
		issues = append(issues, codexDirectorIssueV0("global_write_set_requires_opt_in", "write_set", "write_set global requiere opt-in auditado"))
	}
	if codexDirectorRequiredTestsContainPlaceholderV0(config.RequiredTests) && !config.AllowPlaceholderTests {
		issues = append(issues, codexDirectorIssueV0("placeholder_tests_require_opt_in", "required_tests", "tests placeholder requieren opt-in auditado"))
	}
	if (config.AllowGlobalWriteSet || config.AllowPlaceholderTests) && !codexDirectorGuardOverrideAuditedV0(config) {
		issues = append(issues, codexDirectorIssueV0("guard_override_audit_missing", "guard_override", "opt-in requiere razon y evidence refs"))
	}
	return issues
}

func codexDirectorRequiredTestsContainPlaceholderV0(values []string) bool {
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		switch normalized {
		case "operator-validation-required", "operator_validation_required":
			return true
		}
		if strings.Contains(normalized, "placeholder") {
			return true
		}
	}
	return false
}

func codexDirectorGuardOverrideAuditedV0(config codexDirectorWaveConfigV0) bool {
	if strings.TrimSpace(config.GuardOverrideReason) == "" {
		return false
	}
	for _, ref := range config.GuardOverrideEvidenceRefs {
		if strings.TrimSpace(ref) != "" {
			return true
		}
	}
	return false
}

func codexDirectorIssueV0(code string, field string, message string) orquestadirectoroperativo.OperationalDirectorIssueV0 {
	return orquestadirectoroperativo.OperationalDirectorIssueV0{Code: code, Field: field, Message: message}
}

func codexDirectorWriteGuardOptInRulesV0(b *strings.Builder, config codexDirectorWaveConfigV0) {
	if !config.AllowGlobalWriteSet && !config.AllowPlaceholderTests {
		return
	}
	b.WriteString("- Opt-in auditado de guardas: razon y evidence_refs deben quedar en el summary de la ola.\n")
	b.WriteString("- Aunque haya opt-in, siguen prohibidos borrados no revisados y salir del proyecto.\n")
}

func codexDirectorWriteListV0(b *strings.Builder, values []string) {
	if len(values) == 0 {
		b.WriteString("- ninguno\n")
		return
	}
	for _, value := range values {
		b.WriteString("- " + value + "\n")
	}
}
