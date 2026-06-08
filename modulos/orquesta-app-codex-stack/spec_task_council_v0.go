package orquestaappcodexstack

import (
	"encoding/json"
	"strings"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadecisioncouncil "orquesta/modulos/orquesta-decision-council"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func councilTaskV0(
	task orquestacoreworkflow.WorkflowTaskV0,
	payload orquestaruntime.LaunchRuntimeAgentRequestV0,
) orquestaruntime.AgentStartTaskV0 {
	role := decisionCouncilRoleFromTaskV0(task, payload)
	return orquestaruntime.AgentStartTaskV0{
		TaskRef:         task.TaskID,
		Priority:        "alta",
		Title:           task.Title,
		Objective:       councilObjectiveV0(task, role),
		WriteSet:        append([]string(nil), task.WriteSet...),
		SkillRefs:       compactStringsV0(append(append([]string(nil), payload.SkillRefs...), task.SkillRefs...)),
		ParentTaskRef:   task.ParentTaskRef,
		CohortRef:       task.CohortRef,
		WaveRef:         task.WaveRef,
		DelegationDepth: task.DelegationDepth,
		MaxChildAgents:  task.MaxChildAgents,
		ChildTaskRefs:   append([]string(nil), task.ChildTaskRefs...),
		RequiredTests:   append([]string(nil), task.RequiredTests...),
		DoneCriteria: compactStringsV0(append(
			[]string{
				"agent_ack.json escrito con status completed.",
				"Artefacto esperado entregado: " + councilTaskCriteriaValueV0(task, "expected_artifact:"),
				"Usa refs estructuradas del contexto; no dependas de texto libre ni de nombres inventados.",
			},
			task.AcceptanceCriteria...,
		)),
	}
}

func councilObjectiveV0(
	task orquestacoreworkflow.WorkflowTaskV0,
	role string,
) string {
	lines := []string{
		strings.TrimSpace(task.Summary),
		"Trabajo de consejo residente multiagente de Orquesta.",
		"Rol estructurado: " + role + ".",
		"Assignment: " + councilTaskCriteriaValueV0(task, "assignment_ref:") + ".",
		"Agente/familia canonicos: " + councilTaskCriteriaValueV0(task, "agent_ref:") + " / " + councilTaskCriteriaValueV0(task, "family_ref:") + ".",
		"Artefacto esperado: " + councilTaskCriteriaValueV0(task, "expected_artifact:") + ".",
		"Conserva evidencia durable por refs compactas y entrega ACK normal de Orquesta.",
	}
	switch role {
	case orquestadecisioncouncil.CouncilRoleCritiqueV0:
		lines = append(lines,
			"Critica la propuesta asignada desde sus refs y aporta mejoras, riesgos y disenso util si aplica.",
			"No sustituyas la propuesta por una decision global; entrega una critica cruzada verificable.",
		)
	case orquestadecisioncouncil.CouncilRoleVoteV0:
		lines = append(lines,
			"Emite un voto estructurado `architecture_vote.v0` con task_ref, vote_ref, option_ref, position y evidence_refs.",
			"Si no hay evidencia suficiente, conserva el trabajo como voto razonado de rechazo, bloqueo o abstencion; no inventes consenso.",
		)
	default:
		lines = append(lines,
			"Propon una opcion independiente con resumen, razones, riesgos, tradeoffs y evidencia por refs.",
			"No cierres la arquitectura global; entrega una propuesta para que otros agentes la critiquen y voten.",
		)
	}
	return strings.Join(compactStringsV0(lines), "\n")
}

func decisionCouncilContextEntriesV0(
	area string,
	taskRef string,
	task orquestacoreworkflow.WorkflowTaskV0,
) []orquestacontext.ContextMaterializedEntryV0 {
	role := decisionCouncilRoleFromTaskV0(task, orquestaruntime.LaunchRuntimeAgentRequestV0{})
	if role == "" {
		return nil
	}
	content := decisionCouncilContextContentV0(task, role)
	if content == "" {
		return nil
	}
	refSuffix := packetRefSuffixV0(taskRef, area, "decision-council")
	return []orquestacontext.ContextMaterializedEntryV0{{
		EntryRef:  "entry-ref-app-stack-council-" + refSuffix,
		Layer:     orquestacontext.ContextLayerTaskContextV0,
		Kind:      orquestacontext.ContextEntryDocRefV0,
		SourceRef: "source-ref-app-stack-council-" + refSuffix,
		Mode:      orquestacontext.ContextMaterializationModeContentV0,
		Content:   content,
		Bytes:     len(content),
		Required:  true,
	}}
}

func decisionCouncilContextContentV0(
	task orquestacoreworkflow.WorkflowTaskV0,
	role string,
) string {
	data := map[string]any{
		"schema_version":        "decision_council_context.v0",
		"task_ref":              task.TaskID,
		"run_ref":               task.RunID,
		"phase_id":              string(task.PhaseID),
		"decision_council_role": role,
		"assignment_ref":        councilTaskCriteriaValueV0(task, "assignment_ref:"),
		"agent_ref":             councilTaskCriteriaValueV0(task, "agent_ref:"),
		"family_ref":            councilTaskCriteriaValueV0(task, "family_ref:"),
		"expected_artifact":     councilTaskCriteriaValueV0(task, "expected_artifact:"),
		"gate_ref":              councilTaskCriteriaValueV0(task, "gate_ref:"),
		"context_policy":        councilTaskCriteriaValueV0(task, "context_policy:"),
		"context_refs":          compactStringsV0(task.ContextRefs),
		"depends_on":            compactStringsV0(task.DependsOn),
		"acceptance_criteria":   compactStringsV0(task.AcceptanceCriteria),
		"cohort_ref":            strings.TrimSpace(task.CohortRef),
		"wave_ref":              strings.TrimSpace(task.WaveRef),
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(raw)
}

func decisionCouncilRoleFromTaskV0(
	task orquestacoreworkflow.WorkflowTaskV0,
	payload orquestaruntime.LaunchRuntimeAgentRequestV0,
) string {
	if role := councilRoleFromContextRefsV0(task.ContextRefs); role != "" {
		return role
	}
	if role := councilRoleFromPayloadV0(payload.Role); role != "" {
		return role
	}
	if role := councilTaskCriteriaValueV0(task, "decision_council_role:"); validCouncilPayloadRoleV0(role) {
		return role
	}
	return councilTaskRefRoleV0(task.TaskID)
}

func councilTaskCriteriaValueV0(
	task orquestacoreworkflow.WorkflowTaskV0,
	prefix string,
) string {
	for _, criterion := range task.AcceptanceCriteria {
		if value, ok := strings.CutPrefix(strings.TrimSpace(criterion), prefix); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func councilRoleFromPayloadV0(role string) string {
	role = strings.TrimSpace(role)
	if validCouncilPayloadRoleV0(role) {
		return role
	}
	return ""
}

func councilRoleFromContextRefsV0(refs []string) string {
	for _, ref := range refs {
		switch strings.TrimSpace(ref) {
		case "decision-council-role-p":
			return orquestadecisioncouncil.CouncilRoleProposalV0
		case "decision-council-role-c":
			return orquestadecisioncouncil.CouncilRoleCritiqueV0
		case "decision-council-role-v":
			return orquestadecisioncouncil.CouncilRoleVoteV0
		}
	}
	return ""
}

func councilTaskRefRoleV0(taskRef string) string {
	taskRef = strings.TrimSpace(taskRef)
	switch {
	case strings.HasPrefix(taskRef, "task-council-p-"):
		return orquestadecisioncouncil.CouncilRoleProposalV0
	case strings.HasPrefix(taskRef, "task-council-c-"):
		return orquestadecisioncouncil.CouncilRoleCritiqueV0
	case strings.HasPrefix(taskRef, "task-council-v-"):
		return orquestadecisioncouncil.CouncilRoleVoteV0
	default:
		return ""
	}
}

func validCouncilPayloadRoleV0(role string) bool {
	switch strings.TrimSpace(role) {
	case orquestadecisioncouncil.CouncilRoleProposalV0,
		orquestadecisioncouncil.CouncilRoleCritiqueV0,
		orquestadecisioncouncil.CouncilRoleVoteV0:
		return true
	default:
		return false
	}
}
