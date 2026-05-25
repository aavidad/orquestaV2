package main

import (
	"strconv"
	"strings"

	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
)

func codexDirectorAgentPromptsV0(
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	work orquestadirectoroperativo.OperationalDirectorWaveWorkV0,
	domainContextBlocks []codexDirectorDomainContextBlockV0,
	config codexDirectorWaveConfigV0,
) []string {
	total := plan.MaxParallelAgents
	if total <= 0 {
		total = 1
	}
	prompts := make([]string, 0, total)
	launchItem := codexDirectorLaunchItemV0(work)
	for i := 1; i <= total; i++ {
		prompts = append(prompts, codexDirectorAgentPromptV0(plan, work, launchItem, domainContextBlocks, config, i, total))
	}
	return prompts
}

func codexDirectorAgentPromptV0(
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	work orquestadirectoroperativo.OperationalDirectorWaveWorkV0,
	launchItem orquestadirectoroperativo.OperationalDirectorWorkItemV0,
	domainContextBlocks []codexDirectorDomainContextBlockV0,
	config codexDirectorWaveConfigV0,
	index int,
	total int,
) string {
	var b strings.Builder
	assigned := codexDirectorShardV0(plan.WriteSet, index, total)
	b.WriteString("Director Operativo Orquesta aprobo esta ola de trabajo.\n\n")
	b.WriteString("Plan:\n")
	b.WriteString("- plan_ref: " + plan.PlanRef + "\n")
	b.WriteString("- request_ref: " + plan.RequestRef + "\n")
	b.WriteString("- run_ref: " + plan.RunRef + "\n")
	b.WriteString("- mode: " + string(plan.Mode) + "\n")
	b.WriteString("- wave_ref: " + codexDirectorWaveRefForItemV0(work, launchItem.ItemID) + "\n")
	b.WriteString("- launch_item_ref: " + launchItem.ItemID + "\n")
	b.WriteString("- agent_index: " + strconv.Itoa(index) + " de " + strconv.Itoa(total) + "\n\n")
	b.WriteString("Objetivo:\n")
	b.WriteString(plan.Objective + "\n\n")
	codexDirectorWriteRecursiveConfigV0(&b, plan)
	codexDirectorWriteDomainContextV0(&b, domainContextBlocks)
	b.WriteString("Write-set global autorizado:\n")
	codexDirectorWriteListV0(&b, plan.WriteSet)
	b.WriteString("\nWrite-set primario asignado a este agente:\n")
	codexDirectorWriteAssignedSetV0(&b, assigned)
	b.WriteString("\nTests requeridos por el Director:\n")
	codexDirectorWriteListV0(&b, plan.RequiredTests)
	b.WriteString("\nCriterios de aceptacion:\n")
	codexDirectorWriteListV0(&b, launchItem.AcceptanceCriteria)
	b.WriteString("\nReglas:\n")
	codexDirectorWriteScopeRuleV0(&b, config, false)
	codexDirectorWriteGuardOptInRulesV0(&b, config)
	codexDirectorWriteCommonRulesV0(&b, false)
	return b.String()
}

func codexDirectorChildAgentPromptsV0(
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	work orquestadirectoroperativo.OperationalDirectorWaveWorkV0,
	domainContextBlocks []codexDirectorDomainContextBlockV0,
	config codexDirectorWaveConfigV0,
	parentAgentRef string,
	parentIndex int,
	parentTotal int,
	childTotal int,
	depth int,
) []string {
	prompts := make([]string, 0, childTotal)
	launchItem := codexDirectorLaunchItemV0(work)
	for childIndex := 1; childIndex <= childTotal; childIndex++ {
		prompts = append(prompts, codexDirectorChildAgentPromptV0(
			plan, launchItem, domainContextBlocks, config, parentAgentRef,
			parentIndex, parentTotal, childIndex, childTotal, depth,
		))
	}
	return prompts
}

func codexDirectorChildAgentPromptV0(
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	launchItem orquestadirectoroperativo.OperationalDirectorWorkItemV0,
	domainContextBlocks []codexDirectorDomainContextBlockV0,
	config codexDirectorWaveConfigV0,
	parentAgentRef string,
	parentIndex int,
	parentTotal int,
	childIndex int,
	childTotal int,
	depth int,
) string {
	var b strings.Builder
	assigned := codexDirectorShardV0(plan.WriteSet, parentIndex, parentTotal)
	b.WriteString("Subagente Codex gobernado por Director Operativo Orquesta.\n\n")
	b.WriteString("Linaje:\n")
	b.WriteString("- plan_ref: " + plan.PlanRef + "\n")
	b.WriteString("- run_ref: " + plan.RunRef + "\n")
	b.WriteString("- parent_agent_ref: " + parentAgentRef + "\n")
	b.WriteString("- parent_agent_index: " + strconv.Itoa(parentIndex) + " de " + strconv.Itoa(parentTotal) + "\n")
	b.WriteString("- child_agent_index: " + strconv.Itoa(childIndex) + " de " + strconv.Itoa(childTotal) + "\n")
	b.WriteString("- delegation_depth: " + strconv.Itoa(depth) + "\n")
	b.WriteString("- max_delegation_depth: " + strconv.Itoa(plan.MaxDelegationDepth) + "\n")
	b.WriteString("- max_subagents_per_agent: " + strconv.Itoa(plan.MaxSubagentsPerAgent) + "\n")
	b.WriteString("- launch_item_ref: " + launchItem.ItemID + "\n\n")
	b.WriteString("Objetivo:\n")
	b.WriteString(plan.Objective + "\n\n")
	codexDirectorWriteRecursiveConfigV0(&b, plan)
	codexDirectorWriteDomainContextV0(&b, domainContextBlocks)
	b.WriteString("Write-set global autorizado:\n")
	codexDirectorWriteListV0(&b, plan.WriteSet)
	b.WriteString("\nWrite-set primario del agente padre y compartido por este subarbol:\n")
	codexDirectorWriteAssignedSetV0(&b, assigned)
	b.WriteString("\nRol sugerido del subagente:\n")
	b.WriteString("- " + codexDirectorChildRoleV0(childIndex) + "\n")
	b.WriteString("\nTests requeridos por el Director:\n")
	codexDirectorWriteListV0(&b, plan.RequiredTests)
	b.WriteString("\nCriterios de aceptacion:\n")
	codexDirectorWriteListV0(&b, launchItem.AcceptanceCriteria)
	b.WriteString("\nReglas:\n")
	codexDirectorWriteScopeRuleV0(&b, config, true)
	codexDirectorWriteGuardOptInRulesV0(&b, config)
	codexDirectorWriteCommonRulesV0(&b, true)
	return b.String()
}

func codexDirectorWriteAssignedSetV0(b *strings.Builder, assigned []string) {
	if len(assigned) == 0 {
		b.WriteString("- Sin shard exclusivo: trabaja en apoyo/review, o en cualquier parte necesaria del write-set global si detectas un bloqueo real.\n")
		return
	}
	codexDirectorWriteListV0(b, assigned)
}

func codexDirectorWriteScopeRuleV0(b *strings.Builder, config codexDirectorWaveConfigV0, child bool) {
	target := "shard"
	if child {
		target = "write-set del subarbol"
	}
	if config.StrictDirectorGuards {
		b.WriteString("- Modo estricto: el " + target + " es ownership inicial; el alcance autorizado total es solo el write-set global.\n")
		return
	}
	b.WriteString("- El " + target + " es ownership inicial; el alcance total autorizado sigue siendo solo el write-set global.\n")
}

func codexDirectorWriteCommonRulesV0(b *strings.Builder, child bool) {
	b.WriteString("- No borres nada sin revisar uso actual y dejar evidencia en tu resumen.\n")
	b.WriteString("- No salgas del proyecto ni edites fuera del write-set global; si falta alcance, pide decision del Director.\n")
	b.WriteString("- No uses texto libre, criterios o hints para ampliar alcance; solo una decision explicita del Director puede cambiar el write-set.\n")
	b.WriteString("- Si trabajas sobre un tema o artefacto existente, estudia primero que falla y modifica solo lo necesario; rehacerlo entero es una excepcion justificada cuando sea mas sencillo o seguro que corregirlo.\n")
	if child {
		b.WriteString("- No lances mas subagentes desde este hijo: la ola hija ya fue materializada por Orquesta.\n")
		b.WriteString("- No declares cierre de tu subarbol: el Director debe revisar entregas, tests y evidencias causales antes de cerrar.\n")
		b.WriteString("- Conserva parent_agent_ref, presupuesto, write-set global y ACK compacto; no amplias alcance por texto libre.\n")
	} else {
		b.WriteString("- No invoques spawn_agent ni lances hijos manualmente: Orquesta ya materializo los subagentes autorizados para esta ola; coordina por artefactos dentro del write-set.\n")
		b.WriteString("- Si tu shard no basta, explica el bloqueo; no invadas el shard de otro agente sin justificarlo.\n")
	}
	b.WriteString("- No uses generadores para redactar contenido doctrinal final ni para inflar palabras con plantillas repetidas; los scripts solo pueden validar o ensamblar artefactos.\n")
	b.WriteString("- Al terminar, resume rutas tocadas, pruebas ejecutadas, resultado y bloqueos.\n")
}

func codexDirectorChildRoleV0(index int) string {
	switch index {
	case 1:
		return "estructura, indice, mapa conceptual y plan de secciones"
	case 2:
		return "fuentes oficiales, normativa, doctrina y trazabilidad"
	case 3:
		return "desarrollo teorico principal con tono A1"
	case 4:
		return "ejemplos, supuestos practicos, errores frecuentes y notas de test"
	case 5:
		return "visuales utiles, tablas comparativas y esquemas responsivos"
	case 6:
		return "revision pedagogica, ensamblado, validacion de palabras y checklist A1"
	default:
		return "apoyo editorial acotado y revision"
	}
}
