package main

import "strings"

func autoprogrammingPromotionGuardianEnvV0(
	base []string,
	request serverAutoprogrammingPromotionGuardianRequestV0,
) []string {
	env := append([]string(nil), base...)
	pairs := map[string]string{
		"ORQUESTA_GUARDIAN_PROJECT_DIR":                       request.ProjectDir,
		"ORQUESTA_GUARDIAN_STATE_DIR":                         request.StateDir,
		"ORQUESTA_GUARDIAN_CURRENT_BIN":                       request.CurrentBin,
		"ORQUESTA_GUARDIAN_CANDIDATE_BIN":                     request.CandidateBin,
		"ORQUESTA_GUARDIAN_LAST_GOOD_BIN":                     request.LastGoodBin,
		"ORQUESTA_GUARDIAN_ARTIFACT_ROOT":                     request.ArtifactRoot,
		"ORQUESTA_GUARDIAN_BUILD_COMMAND":                     request.BuildCommand,
		"ORQUESTA_GUARDIAN_TEST_COMMANDS":                     strings.Join(request.TestCommands, "\n"),
		"ORQUESTA_GUARDIAN_HEALTH_TIMEOUT":                    request.HealthTimeout,
		"ORQUESTA_GUARDIAN_COMMAND_TIMEOUT":                   request.CommandTimeout,
		"ORQUESTA_GUARDIAN_ARTIFACT_MAX_BYTES":                request.ArtifactMaxBytes,
		"ORQUESTA_GUARDIAN_REPAIR_COMMAND":                    request.RepairCommand,
		"ORQUESTA_GUARDIAN_REPAIR_CODEX_WRITE_SET":            strings.Join(request.RepairCodexWriteSet, ","),
		"ORQUESTA_GUARDIAN_REPAIR_CODEX_REQUIRED_TESTS":       strings.Join(request.RepairCodexRequiredTests, "\n"),
		"ORQUESTA_GUARDIAN_REPAIR_CODEX_WORKTREE_REF":         request.RepairCodexWorktreeRef,
		"ORQUESTA_GUARDIAN_REPAIR_CODEX_BRANCH_REF":           request.RepairCodexBranchRef,
		"ORQUESTA_GUARDIAN_REPAIR_CODEX_RUN_REF":              request.RepairCodexRunRef,
		"ORQUESTA_GUARDIAN_REPAIR_CODEX_PROMOTION_REF":        request.RepairCodexPromotionRef,
		"ORQUESTA_GUARDIAN_REPAIR_CODEX_SANDBOX":              request.RepairCodexSandbox,
		"ORQUESTA_GUARDIAN_REPAIR_CODEX_REASONING_EFFORT":     request.RepairCodexReasoning,
		"ORQUESTA_GUARDIAN_REPAIR_CODEX_RUNTIME_DIR":          request.RepairCodexRuntimeDir,
		"ORQUESTA_GUARDIAN_REPAIR_CODEX_SANDBOX_EVIDENCE_REF": request.RepairCodexSandboxEvidence,
		"ORQUESTA_GUARDIAN_COMMAND_EFFECT_EVIDENCE_REFS": strings.Join(
			request.CommandEffectEvidenceRefs,
			",",
		),
		"ORQUESTA_GUARDIAN_SKIP_HEALTH_EVIDENCE_REFS": strings.Join(request.SkipHealthEvidenceRefs, ","),
		"ORQUESTA_GUARDIAN_PROMOTION_REF":             request.PromotionRef,
		"ORQUESTA_GUARDIAN_RUN_REF":                   request.RunRef,
		"ORQUESTA_GUARDIAN_WORKTREE_REF":              request.WorktreeRef,
		"ORQUESTA_GUARDIAN_BRANCH_REF":                request.BranchRef,
		"ORQUESTA_GUARDIAN_PROMOTION_EVIDENCE":        strings.Join(request.EvidenceRefs, ","),
	}
	for key, value := range pairs {
		if strings.TrimSpace(value) != "" {
			env = append(env, key+"="+value)
		}
	}
	if request.RepairCodex {
		env = append(env, "ORQUESTA_GUARDIAN_REPAIR_CODEX=1")
	}
	if request.RepairCodexAllowBroad {
		env = append(env, "ORQUESTA_GUARDIAN_REPAIR_CODEX_ALLOW_BROAD_SANDBOX=1")
	}
	return env
}
