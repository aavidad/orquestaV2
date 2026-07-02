package orquestaautoprogramming

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	autoprogrammingStartupLockContractRefV0       = "contract:codex-startup-lock:v0"
	autoprogrammingStartupLockIncidentRefV0       = "incidencia:docs/incidencia_opes_codex_startup_lock_stale_timeout_tractorista_2026-06-22.md"
	autoprogrammingStartupLockRemoteIncidentRefV0 = "incidencia:docs/incidencias/incidencia_orquesta_remoto_reabre_startup_lock_timeout_solo_2026-07-01.md"
	autoprogrammingStartupLockBugRefV0            = "bug:BUG-ORQ-20260701-092"
	autoprogrammingStartupLockRegressionTestV0    = "go test -count=1 ./modulos/orquesta-runtime-codex -run 'TestCodexWrapperV0(CualquierVariableStartupLockCuentaComoExplicitaV0|RespetaStartupLockExplicitoAunqueDefaultSeaCeroV0|RespetaStartupLockTimeoutSoloAunqueDefaultSeaCeroV0|RespetaStartupLockTimeoutExplicitoAunqueDefaultSeaCeroV0|RetiraStartupLockObsoletoVacio|StaleLockPorDefectoNoSuperaTimeout)'"
)

type autoprogrammingContractGuardV0 struct {
	triggerRefs        []string
	contractRef        string
	requiredTests      []string
	acceptanceCriteria []string
}

func autoprogrammingKnownContractGuardsV0() []autoprogrammingContractGuardV0 {
	return []autoprogrammingContractGuardV0{
		{
			triggerRefs: []string{
				autoprogrammingStartupLockContractRefV0,
				autoprogrammingStartupLockIncidentRefV0,
				"docs/incidencia_opes_codex_startup_lock_stale_timeout_tractorista_2026-06-22.md",
				autoprogrammingStartupLockRemoteIncidentRefV0,
				"docs/incidencias/incidencia_orquesta_remoto_reabre_startup_lock_timeout_solo_2026-07-01.md",
				autoprogrammingStartupLockBugRefV0,
			},
			contractRef:   autoprogrammingStartupLockContractRefV0,
			requiredTests: []string{autoprogrammingStartupLockRegressionTestV0},
			acceptanceCriteria: []string{
				"startup-lock: enumerar ORQUESTA_CODEX_STARTUP_LOCK_SECONDS.",
				"startup-lock: enumerar ORQUESTA_CODEX_STARTUP_LOCK_TIMEOUT_SECONDS.",
				"startup-lock: enumerar ORQUESTA_CODEX_STARTUP_LOCK_STALE_SECONDS.",
				"startup-lock: cualquier ORQUESTA_CODEX_STARTUP_LOCK_* explicita activa lock.",
				"startup-lock: stale efectivo no supera timeout efectivo.",
			},
		},
	}
}

func autoprogrammingContractGuardsForGroupV0(
	group AutoprogrammingTaskGroupV0,
) []autoprogrammingContractGuardV0 {
	return autoprogrammingContractGuardsForRefsV0(
		autoprogrammingContractTriggerRefsForGroupV0(group),
	)
}

func autoprogrammingContractGuardsForRefsV0(
	refs []string,
) []autoprogrammingContractGuardV0 {
	guards := autoprogrammingKnownContractGuardsV0()
	out := make([]autoprogrammingContractGuardV0, 0, len(guards))
	for _, guard := range guards {
		if autoprogrammingContractGuardTriggeredV0(refs, guard.triggerRefs) {
			out = append(out, guard)
		}
	}
	return out
}

func autoprogrammingContractTriggerRefsForGroupV0(
	group AutoprogrammingTaskGroupV0,
) []string {
	var refs []string
	for _, task := range group.Tasks {
		refs = append(refs, task.ContextRefs...)
	}
	return compactStringsV0(refs)
}

func autoprogrammingContractGuardTriggeredV0(
	refs []string,
	triggers []string,
) bool {
	for _, ref := range refs {
		ref = strings.ToLower(strings.TrimSpace(ref))
		for _, trigger := range triggers {
			if ref == strings.ToLower(strings.TrimSpace(trigger)) {
				return true
			}
		}
	}
	return false
}

func autoprogrammingContractRequiredTestsForGroupV0(
	group AutoprogrammingTaskGroupV0,
) []string {
	var out []string
	for _, guard := range autoprogrammingContractGuardsForGroupV0(group) {
		out = append(out, guard.requiredTests...)
	}
	return compactStringsV0(out)
}

func autoprogrammingContractAcceptanceCriteriaForGroupV0(
	group AutoprogrammingTaskGroupV0,
) []string {
	var out []string
	for _, guard := range autoprogrammingContractGuardsForGroupV0(group) {
		out = append(out, guard.acceptanceCriteria...)
	}
	return compactStringsV0(out)
}

func autoprogrammingContractRefsForGroupV0(
	group AutoprogrammingTaskGroupV0,
) []string {
	var out []string
	for _, guard := range autoprogrammingContractGuardsForGroupV0(group) {
		if strings.TrimSpace(guard.contractRef) != "" {
			out = append(out, guard.contractRef)
		}
	}
	return compactStringsV0(out)
}

func autoprogrammingWorkflowFunctionContractRefsForGroupV0(
	group AutoprogrammingTaskGroupV0,
) []orquestacoreworkflow.WorkflowFunctionContractRefV0 {
	var out []orquestacoreworkflow.WorkflowFunctionContractRefV0
	for _, guard := range autoprogrammingContractGuardsForGroupV0(group) {
		if strings.TrimSpace(guard.contractRef) != "" {
			out = append(out, orquestacoreworkflow.WorkflowFunctionContractRefV0{
				ContractRef: guard.contractRef,
			})
		}
	}
	return out
}

func autoprogrammingGoalContractRuleRefsForGroupV0(
	group AutoprogrammingProgrammableGroupV0,
) []orquestagoal.GoalRuleRefV0 {
	var out []orquestagoal.GoalRuleRefV0
	for _, guard := range autoprogrammingContractGuardsForRefsV0(group.Task.ContextRefs) {
		if strings.TrimSpace(guard.contractRef) != "" {
			out = append(out, orquestagoal.GoalRuleRefV0{
				Kind:        "contract",
				Ref:         guard.contractRef,
				Enforcement: orquestagoal.GoalRuleEnforcementHardV0,
			})
		}
	}
	return out
}
