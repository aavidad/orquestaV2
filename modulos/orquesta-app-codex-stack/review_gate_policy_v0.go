package orquestaappcodexstack

import (
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

type ReviewGateDeliveryPolicyPortV0 interface {
	StrictGoLineBudgetForDeliveryV0(
		descriptor orquestaruntimecodex.CodexAgentAckV0,
		packet orquestaruntime.AgentStartPacketV0,
	) bool
	EffectiveWriteSetV0(writeSet []string) []string
	MaxGoFileLinesV0() int
}

type codexStackReviewGatePolicyV0 struct {
	StrictGoLineBudget bool
	MaxGoFileLines     int
}

func codexStackReviewGatePolicyFromConfigV0(
	config ReviewGateConfigV0,
) ReviewGateDeliveryPolicyPortV0 {
	if config.Policy != nil {
		return config.Policy
	}
	return codexStackReviewGatePolicyV0{
		StrictGoLineBudget: config.StrictGoLineBudget,
		MaxGoFileLines:     orquestaruntimeworktree.WorktreeGoFileLineBudgetLimitV0(config.MaxLinesPerFile),
	}
}

func (policy codexStackReviewGatePolicyV0) StrictGoLineBudgetForDeliveryV0(
	descriptor orquestaruntimecodex.CodexAgentAckV0,
	packet orquestaruntime.AgentStartPacketV0,
) bool {
	if !policy.StrictGoLineBudget {
		return false
	}
	if codexStackReviewGatePacketProfileExcludesGoBudgetV0(packet.Policies) {
		return false
	}
	if codexStackReviewGateHasGoTestV0(packet.Task.RequiredTests) ||
		codexStackReviewGateHasGoTestV0(descriptor.Tests) {
		return true
	}
	return codexStackReviewGatePathListTouchesGoV0(packet.Task.WriteSet) ||
		codexStackReviewGatePathListTouchesGoV0(descriptor.Files)
}

func (policy codexStackReviewGatePolicyV0) EffectiveWriteSetV0(writeSet []string) []string {
	out := make([]string, 0, len(writeSet))
	for _, entry := range compactStringsV0(writeSet) {
		out = append(out, entry)
		out = append(out, codexStackReviewGateWriteSetAliasesV0(entry)...)
	}
	return compactStringsV0(out)
}

func (policy codexStackReviewGatePolicyV0) MaxGoFileLinesV0() int {
	return policy.MaxGoFileLines
}

func codexStackReviewGatePacketProfileExcludesGoBudgetV0(policies []string) bool {
	for _, policy := range compactStringsV0(policies) {
		key, value := codexStackReviewGatePolicyPairV0(policy)
		if !codexStackReviewGateProfilePolicyKeyV0(key) {
			continue
		}
		switch strings.TrimSpace(value) {
		case "documentation", "docs", "documentacion", "domain_work", "review", "no_go", "non_go":
			return true
		}
	}
	return false
}

func codexStackReviewGatePolicyPairV0(policy string) (string, string) {
	policy = strings.ToLower(strings.TrimSpace(policy))
	for _, separator := range []string{":", "="} {
		if key, value, ok := strings.Cut(policy, separator); ok {
			return strings.TrimSpace(key), strings.TrimSpace(value)
		}
	}
	return "", policy
}

func codexStackReviewGateProfilePolicyKeyV0(key string) bool {
	switch strings.ReplaceAll(strings.TrimSpace(key), "-", "_") {
	case "", "profile", "task_profile", "work_profile", "work_profile_kind", "work_kind":
		return true
	default:
		return false
	}
}

func codexStackReviewGateWriteSetAliasesV0(entry string) []string {
	switch strings.Trim(strings.TrimSpace(entry), "/") {
	case "web":
		return []string{"internal/web", "internal/webadmin", "frontend"}
	case "api":
		return []string{"internal/api", "cmd/api"}
	case "docs", "documentacion":
		return []string{"docs", "*/docs", "**/docs/**"}
	default:
		return nil
	}
}

func codexStackReviewGateHasGoTestV0(commands []string) bool {
	for _, command := range commands {
		parts := strings.Fields(strings.TrimSpace(command))
		if len(parts) >= 2 && parts[0] == "go" && parts[1] == "test" {
			return true
		}
	}
	return false
}

func codexStackReviewGatePathListTouchesGoV0(paths []string) bool {
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if strings.HasSuffix(path, ".go") {
			return true
		}
	}
	return false
}
