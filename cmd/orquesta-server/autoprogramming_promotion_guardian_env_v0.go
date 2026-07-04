package main

import "strings"

const autoprogrammingPromotionGuardianChildEnvScopeV0 = "child_process"

const (
	envAutoprogrammingPromotionGuardianProjectDirV0                 = "ORQUESTA_GUARDIAN_PROJECT_DIR"
	envAutoprogrammingPromotionGuardianStateDirV0                   = "ORQUESTA_GUARDIAN_STATE_DIR"
	envAutoprogrammingPromotionGuardianCurrentBinV0                 = "ORQUESTA_GUARDIAN_CURRENT_BIN"
	envAutoprogrammingPromotionGuardianCandidateBinV0               = "ORQUESTA_GUARDIAN_CANDIDATE_BIN"
	envAutoprogrammingPromotionGuardianLastGoodBinV0                = "ORQUESTA_GUARDIAN_LAST_GOOD_BIN"
	envAutoprogrammingPromotionGuardianArtifactRootV0               = "ORQUESTA_GUARDIAN_ARTIFACT_ROOT"
	envAutoprogrammingPromotionGuardianBuildCommandV0               = "ORQUESTA_GUARDIAN_BUILD_COMMAND"
	envAutoprogrammingPromotionGuardianTestCommandsV0               = "ORQUESTA_GUARDIAN_TEST_COMMANDS"
	envAutoprogrammingPromotionGuardianHealthTimeoutV0              = "ORQUESTA_GUARDIAN_HEALTH_TIMEOUT"
	envAutoprogrammingPromotionGuardianCommandTimeoutV0             = "ORQUESTA_GUARDIAN_COMMAND_TIMEOUT"
	envAutoprogrammingPromotionGuardianArtifactMaxBytesV0           = "ORQUESTA_GUARDIAN_ARTIFACT_MAX_BYTES"
	envAutoprogrammingPromotionGuardianRepairCommandV0              = "ORQUESTA_GUARDIAN_REPAIR_COMMAND"
	envAutoprogrammingPromotionGuardianRepairCodexV0                = "ORQUESTA_GUARDIAN_REPAIR_CODEX"
	envAutoprogrammingPromotionGuardianRepairCodexWriteSetV0        = "ORQUESTA_GUARDIAN_REPAIR_CODEX_WRITE_SET"
	envAutoprogrammingPromotionGuardianRepairCodexRequiredTestsV0   = "ORQUESTA_GUARDIAN_REPAIR_CODEX_REQUIRED_TESTS"
	envAutoprogrammingPromotionGuardianRepairCodexWorktreeRefV0     = "ORQUESTA_GUARDIAN_REPAIR_CODEX_WORKTREE_REF"
	envAutoprogrammingPromotionGuardianRepairCodexBranchRefV0       = "ORQUESTA_GUARDIAN_REPAIR_CODEX_BRANCH_REF"
	envAutoprogrammingPromotionGuardianRepairCodexRunRefV0          = "ORQUESTA_GUARDIAN_REPAIR_CODEX_RUN_REF"
	envAutoprogrammingPromotionGuardianRepairCodexPromotionRefV0    = "ORQUESTA_GUARDIAN_REPAIR_CODEX_PROMOTION_REF"
	envAutoprogrammingPromotionGuardianRepairCodexSandboxV0         = "ORQUESTA_GUARDIAN_REPAIR_CODEX_SANDBOX"
	envAutoprogrammingPromotionGuardianRepairCodexReasoningEffortV0 = "ORQUESTA_GUARDIAN_REPAIR_CODEX_REASONING_EFFORT"
	envAutoprogrammingPromotionGuardianRepairCodexRuntimeDirV0      = "ORQUESTA_GUARDIAN_REPAIR_CODEX_RUNTIME_DIR"
	envAutoprogrammingPromotionGuardianRepairCodexAllowBroadV0      = "ORQUESTA_GUARDIAN_REPAIR_CODEX_ALLOW_BROAD_SANDBOX"
	envAutoprogrammingPromotionGuardianRepairCodexSandboxEvidenceV0 = "ORQUESTA_GUARDIAN_REPAIR_CODEX_SANDBOX_EVIDENCE_REF"
	envAutoprogrammingPromotionGuardianCommandEffectEvidenceRefsV0  = "ORQUESTA_GUARDIAN_COMMAND_EFFECT_EVIDENCE_REFS"
	envAutoprogrammingPromotionGuardianSkipHealthEvidenceRefsV0     = "ORQUESTA_GUARDIAN_SKIP_HEALTH_EVIDENCE_REFS"
	envAutoprogrammingPromotionGuardianPromotionRefV0               = "ORQUESTA_GUARDIAN_PROMOTION_REF"
	envAutoprogrammingPromotionGuardianRunRefV0                     = "ORQUESTA_GUARDIAN_RUN_REF"
	envAutoprogrammingPromotionGuardianWorktreeRefV0                = "ORQUESTA_GUARDIAN_WORKTREE_REF"
	envAutoprogrammingPromotionGuardianBranchRefV0                  = "ORQUESTA_GUARDIAN_BRANCH_REF"
	envAutoprogrammingPromotionGuardianPromotionEvidenceV0          = "ORQUESTA_GUARDIAN_PROMOTION_EVIDENCE"
)

type autoprogrammingPromotionGuardianChildEnvMetadataV0 struct {
	Scope          string
	Classification string
	Label          string
	Description    string
}

type autoprogrammingPromotionGuardianChildEnvPublicMetadataV0 struct {
	Key            string `json:"key"`
	Scope          string `json:"scope"`
	Classification string `json:"classification"`
	Label          string `json:"label"`
	Description    string `json:"description"`
}

type autoprogrammingPromotionGuardianEnvBindingV0 struct {
	Key   string
	Value string
}

var autoprogrammingPromotionGuardianChildEnvRegistryV0 = map[string]autoprogrammingPromotionGuardianChildEnvMetadataV0{
	envAutoprogrammingPromotionGuardianProjectDirV0:                 guardianChildEnvMetadataV0("local_path", "Proyecto", "Directorio de trabajo del proceso guardian."),
	envAutoprogrammingPromotionGuardianStateDirV0:                   guardianChildEnvMetadataV0("local_path", "Estado guardian", "Directorio de estado aislado del guardian."),
	envAutoprogrammingPromotionGuardianCurrentBinV0:                 guardianChildEnvMetadataV0("local_path", "Binario actual", "Ruta del binario activo que puede restaurarse o comparar."),
	envAutoprogrammingPromotionGuardianCandidateBinV0:               guardianChildEnvMetadataV0("local_path", "Binario candidato", "Ruta del binario candidato que se valida antes de promocion."),
	envAutoprogrammingPromotionGuardianLastGoodBinV0:                guardianChildEnvMetadataV0("local_path", "Ultimo binario valido", "Ruta del ultimo binario valido conocido."),
	envAutoprogrammingPromotionGuardianArtifactRootV0:               guardianChildEnvMetadataV0("local_path", "Artefactos guardian", "Raiz local donde el guardian escribe artefactos propios."),
	envAutoprogrammingPromotionGuardianBuildCommandV0:               guardianChildEnvMetadataV0("command_template", "Build", "Comando de build entregado al guardian."),
	envAutoprogrammingPromotionGuardianTestCommandsV0:               guardianChildEnvMetadataV0("command_template", "Tests", "Comandos de test entregados al guardian."),
	envAutoprogrammingPromotionGuardianHealthTimeoutV0:              guardianChildEnvMetadataV0("duration_policy", "Timeout health", "Timeout de healthcheck entregado al guardian."),
	envAutoprogrammingPromotionGuardianCommandTimeoutV0:             guardianChildEnvMetadataV0("duration_policy", "Timeout comando", "Timeout de comandos ejecutados por guardian."),
	envAutoprogrammingPromotionGuardianArtifactMaxBytesV0:           guardianChildEnvMetadataV0("byte_budget", "Limite artefacto", "Limite maximo de bytes de artefactos guardian."),
	envAutoprogrammingPromotionGuardianRepairCommandV0:              guardianChildEnvMetadataV0("command_template", "Repair", "Comando de reparacion entregado al guardian."),
	envAutoprogrammingPromotionGuardianRepairCodexV0:                guardianChildEnvMetadataV0("boolean_flag", "Repair Codex", "Activa reparacion Codex gestionada por guardian."),
	envAutoprogrammingPromotionGuardianRepairCodexWriteSetV0:        guardianChildEnvMetadataV0("write_set", "Write-set repair", "Write-set permitido para reparacion Codex."),
	envAutoprogrammingPromotionGuardianRepairCodexRequiredTestsV0:   guardianChildEnvMetadataV0("command_template", "Tests repair", "Tests requeridos para reparacion Codex."),
	envAutoprogrammingPromotionGuardianRepairCodexWorktreeRefV0:     guardianChildEnvMetadataV0("opaque_ref", "Worktree repair", "Ref opaca de worktree de reparacion Codex."),
	envAutoprogrammingPromotionGuardianRepairCodexBranchRefV0:       guardianChildEnvMetadataV0("opaque_ref", "Branch repair", "Ref opaca de rama de reparacion Codex."),
	envAutoprogrammingPromotionGuardianRepairCodexRunRefV0:          guardianChildEnvMetadataV0("opaque_ref", "Run repair", "Ref opaca de run de reparacion Codex."),
	envAutoprogrammingPromotionGuardianRepairCodexPromotionRefV0:    guardianChildEnvMetadataV0("opaque_ref", "Promocion repair", "Ref opaca de promocion de reparacion Codex."),
	envAutoprogrammingPromotionGuardianRepairCodexSandboxV0:         guardianChildEnvMetadataV0("runtime_policy", "Sandbox repair", "Politica sandbox entregada al guardian para reparacion Codex."),
	envAutoprogrammingPromotionGuardianRepairCodexReasoningEffortV0: guardianChildEnvMetadataV0("runtime_policy", "Reasoning repair", "Esfuerzo de razonamiento entregado al guardian para reparacion Codex."),
	envAutoprogrammingPromotionGuardianRepairCodexRuntimeDirV0:      guardianChildEnvMetadataV0("local_path", "Runtime repair", "Directorio runtime aislado de reparacion Codex."),
	envAutoprogrammingPromotionGuardianRepairCodexAllowBroadV0:      guardianChildEnvMetadataV0("boolean_flag", "Sandbox amplio repair", "Breakglass para permitir sandbox amplio en reparacion Codex."),
	envAutoprogrammingPromotionGuardianRepairCodexSandboxEvidenceV0: guardianChildEnvMetadataV0("evidence_ref", "Evidencia sandbox", "Ref de evidencia asociada a sandbox de reparacion Codex."),
	envAutoprogrammingPromotionGuardianCommandEffectEvidenceRefsV0:  guardianChildEnvMetadataV0("evidence_refs", "Evidencias efecto", "Refs de evidencia de efectos externos autorizados."),
	envAutoprogrammingPromotionGuardianSkipHealthEvidenceRefsV0:     guardianChildEnvMetadataV0("evidence_refs", "Evidencias skip health", "Refs de evidencia para omitir healthcheck."),
	envAutoprogrammingPromotionGuardianPromotionRefV0:               guardianChildEnvMetadataV0("opaque_ref", "Promocion", "Ref opaca de promocion supervisada."),
	envAutoprogrammingPromotionGuardianRunRefV0:                     guardianChildEnvMetadataV0("opaque_ref", "Run", "Ref opaca del run causal."),
	envAutoprogrammingPromotionGuardianWorktreeRefV0:                guardianChildEnvMetadataV0("opaque_ref", "Worktree", "Ref opaca del worktree causal."),
	envAutoprogrammingPromotionGuardianBranchRefV0:                  guardianChildEnvMetadataV0("opaque_ref", "Branch", "Ref opaca de branch causal."),
	envAutoprogrammingPromotionGuardianPromotionEvidenceV0:          guardianChildEnvMetadataV0("evidence_refs", "Evidencias promocion", "Refs de evidencia de la promocion."),
}

var autoprogrammingPromotionGuardianChildEnvRegistryKeysV0 = []string{
	envAutoprogrammingPromotionGuardianProjectDirV0,
	envAutoprogrammingPromotionGuardianStateDirV0,
	envAutoprogrammingPromotionGuardianCurrentBinV0,
	envAutoprogrammingPromotionGuardianCandidateBinV0,
	envAutoprogrammingPromotionGuardianLastGoodBinV0,
	envAutoprogrammingPromotionGuardianArtifactRootV0,
	envAutoprogrammingPromotionGuardianBuildCommandV0,
	envAutoprogrammingPromotionGuardianTestCommandsV0,
	envAutoprogrammingPromotionGuardianHealthTimeoutV0,
	envAutoprogrammingPromotionGuardianCommandTimeoutV0,
	envAutoprogrammingPromotionGuardianArtifactMaxBytesV0,
	envAutoprogrammingPromotionGuardianRepairCommandV0,
	envAutoprogrammingPromotionGuardianRepairCodexV0,
	envAutoprogrammingPromotionGuardianRepairCodexWriteSetV0,
	envAutoprogrammingPromotionGuardianRepairCodexRequiredTestsV0,
	envAutoprogrammingPromotionGuardianRepairCodexWorktreeRefV0,
	envAutoprogrammingPromotionGuardianRepairCodexBranchRefV0,
	envAutoprogrammingPromotionGuardianRepairCodexRunRefV0,
	envAutoprogrammingPromotionGuardianRepairCodexPromotionRefV0,
	envAutoprogrammingPromotionGuardianRepairCodexSandboxV0,
	envAutoprogrammingPromotionGuardianRepairCodexReasoningEffortV0,
	envAutoprogrammingPromotionGuardianRepairCodexRuntimeDirV0,
	envAutoprogrammingPromotionGuardianRepairCodexAllowBroadV0,
	envAutoprogrammingPromotionGuardianRepairCodexSandboxEvidenceV0,
	envAutoprogrammingPromotionGuardianCommandEffectEvidenceRefsV0,
	envAutoprogrammingPromotionGuardianSkipHealthEvidenceRefsV0,
	envAutoprogrammingPromotionGuardianPromotionRefV0,
	envAutoprogrammingPromotionGuardianRunRefV0,
	envAutoprogrammingPromotionGuardianWorktreeRefV0,
	envAutoprogrammingPromotionGuardianBranchRefV0,
	envAutoprogrammingPromotionGuardianPromotionEvidenceV0,
}

func autoprogrammingPromotionGuardianEnvV0(
	base []string,
	request serverAutoprogrammingPromotionGuardianRequestV0,
) []string {
	env := append([]string(nil), base...)
	for _, binding := range autoprogrammingPromotionGuardianEnvBindingsV0(request) {
		if strings.TrimSpace(binding.Value) != "" {
			env = append(env, binding.Key+"="+binding.Value)
		}
	}
	return env
}

func autoprogrammingPromotionGuardianEnvBindingsV0(
	request serverAutoprogrammingPromotionGuardianRequestV0,
) []autoprogrammingPromotionGuardianEnvBindingV0 {
	return []autoprogrammingPromotionGuardianEnvBindingV0{
		{Key: envAutoprogrammingPromotionGuardianProjectDirV0, Value: request.ProjectDir},
		{Key: envAutoprogrammingPromotionGuardianStateDirV0, Value: request.StateDir},
		{Key: envAutoprogrammingPromotionGuardianCurrentBinV0, Value: request.CurrentBin},
		{Key: envAutoprogrammingPromotionGuardianCandidateBinV0, Value: request.CandidateBin},
		{Key: envAutoprogrammingPromotionGuardianLastGoodBinV0, Value: request.LastGoodBin},
		{Key: envAutoprogrammingPromotionGuardianArtifactRootV0, Value: request.ArtifactRoot},
		{Key: envAutoprogrammingPromotionGuardianBuildCommandV0, Value: request.BuildCommand},
		{Key: envAutoprogrammingPromotionGuardianTestCommandsV0, Value: strings.Join(request.TestCommands, "\n")},
		{Key: envAutoprogrammingPromotionGuardianHealthTimeoutV0, Value: request.HealthTimeout},
		{Key: envAutoprogrammingPromotionGuardianCommandTimeoutV0, Value: request.CommandTimeout},
		{Key: envAutoprogrammingPromotionGuardianArtifactMaxBytesV0, Value: request.ArtifactMaxBytes},
		{Key: envAutoprogrammingPromotionGuardianRepairCommandV0, Value: request.RepairCommand},
		{Key: envAutoprogrammingPromotionGuardianRepairCodexV0, Value: guardianBooleanChildEnvValueV0(request.RepairCodex)},
		{Key: envAutoprogrammingPromotionGuardianRepairCodexWriteSetV0, Value: strings.Join(request.RepairCodexWriteSet, ",")},
		{Key: envAutoprogrammingPromotionGuardianRepairCodexRequiredTestsV0, Value: strings.Join(request.RepairCodexRequiredTests, "\n")},
		{Key: envAutoprogrammingPromotionGuardianRepairCodexWorktreeRefV0, Value: request.RepairCodexWorktreeRef},
		{Key: envAutoprogrammingPromotionGuardianRepairCodexBranchRefV0, Value: request.RepairCodexBranchRef},
		{Key: envAutoprogrammingPromotionGuardianRepairCodexRunRefV0, Value: request.RepairCodexRunRef},
		{Key: envAutoprogrammingPromotionGuardianRepairCodexPromotionRefV0, Value: request.RepairCodexPromotionRef},
		{Key: envAutoprogrammingPromotionGuardianRepairCodexSandboxV0, Value: request.RepairCodexSandbox},
		{Key: envAutoprogrammingPromotionGuardianRepairCodexReasoningEffortV0, Value: request.RepairCodexReasoning},
		{Key: envAutoprogrammingPromotionGuardianRepairCodexRuntimeDirV0, Value: request.RepairCodexRuntimeDir},
		{Key: envAutoprogrammingPromotionGuardianRepairCodexAllowBroadV0, Value: guardianBooleanChildEnvValueV0(request.RepairCodexAllowBroad)},
		{Key: envAutoprogrammingPromotionGuardianRepairCodexSandboxEvidenceV0, Value: request.RepairCodexSandboxEvidence},
		{Key: envAutoprogrammingPromotionGuardianCommandEffectEvidenceRefsV0, Value: strings.Join(request.CommandEffectEvidenceRefs, ",")},
		{Key: envAutoprogrammingPromotionGuardianSkipHealthEvidenceRefsV0, Value: strings.Join(request.SkipHealthEvidenceRefs, ",")},
		{Key: envAutoprogrammingPromotionGuardianPromotionRefV0, Value: request.PromotionRef},
		{Key: envAutoprogrammingPromotionGuardianRunRefV0, Value: request.RunRef},
		{Key: envAutoprogrammingPromotionGuardianWorktreeRefV0, Value: request.WorktreeRef},
		{Key: envAutoprogrammingPromotionGuardianBranchRefV0, Value: request.BranchRef},
		{Key: envAutoprogrammingPromotionGuardianPromotionEvidenceV0, Value: strings.Join(request.EvidenceRefs, ",")},
	}
}

func autoprogrammingPromotionGuardianChildEnvPublicRegistryV0() []autoprogrammingPromotionGuardianChildEnvPublicMetadataV0 {
	records := make([]autoprogrammingPromotionGuardianChildEnvPublicMetadataV0, 0, len(autoprogrammingPromotionGuardianChildEnvRegistryKeysV0))
	for _, key := range autoprogrammingPromotionGuardianChildEnvRegistryKeysV0 {
		metadata := autoprogrammingPromotionGuardianChildEnvRegistryV0[key]
		records = append(records, autoprogrammingPromotionGuardianChildEnvPublicMetadataV0{
			Key:            key,
			Scope:          metadata.Scope,
			Classification: metadata.Classification,
			Label:          metadata.Label,
			Description:    metadata.Description,
		})
	}
	return records
}

func guardianChildEnvMetadataV0(
	classification string,
	label string,
	description string,
) autoprogrammingPromotionGuardianChildEnvMetadataV0 {
	return autoprogrammingPromotionGuardianChildEnvMetadataV0{
		Scope:          autoprogrammingPromotionGuardianChildEnvScopeV0,
		Classification: classification,
		Label:          label,
		Description:    description,
	}
}

func guardianBooleanChildEnvValueV0(enabled bool) string {
	if enabled {
		return "1"
	}
	return ""
}
