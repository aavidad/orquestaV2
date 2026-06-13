package orquestaappcodexstack

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
)

type ProjectPlanBootstrapDirectorDecisionSourceV0 struct {
	ProjectWorkDir string
}

var _ orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0 = ProjectPlanBootstrapDirectorDecisionSourceV0{}

func (source ProjectPlanBootstrapDirectorDecisionSourceV0) ListDirectorAgentDecisionsV0(
	_ context.Context,
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	if !projectPlanBootstrapShouldEmitV0(request) ||
		!source.projectPlanBootstrapHasPlanningSignalV0(request.Run) {
		return nil, nil
	}
	slug := projectPlanBootstrapSlugV0(source.ProjectWorkDir, request.Run)
	return projectPlanBootstrapDecisionsV0(request.Run, slug), nil
}

func projectPlanBootstrapShouldEmitV0(
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
) bool {
	if codexStackDecisionSourceRecoveryRequestedV0(request.RequestedBy) {
		return false
	}
	run := request.Run
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 ||
		strings.TrimSpace(run.RunID) == "" ||
		len(compactStringsV0(run.Tasks)) > 0 ||
		drainRunHasPendingExternalAgentsV0(run, nil) {
		return false
	}
	if orquestafactory.NormalizeRequestKindV0(request.RequestKind) != orquestafactory.RequestKindCrearAppCompletaV0 {
		return false
	}
	switch run.CurrentPhase {
	case "",
		orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0,
		orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0,
		orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
		orquestacoreworkflow.OrchestrationPhaseProgramacionV0:
		return true
	default:
		return false
	}
}

func (source ProjectPlanBootstrapDirectorDecisionSourceV0) projectPlanBootstrapHasPlanningSignalV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	projectWorkDir := strings.TrimSpace(source.ProjectWorkDir)
	if projectWorkDir == "" {
		return len(compactStringsV0(run.PhaseArtifacts)) > 0 ||
			len(compactStringsV0(run.Deliveries)) > 0 ||
			len(compactStringsV0(run.DeliveredAgents)) > 0
	}
	for _, rel := range []string{
		"docs/plan_tareas.md",
		"docs/plan_microtareas.md",
		"docs/arquitectura.md",
		"docs/decisiones.md",
	} {
		path := filepath.Join(projectWorkDir, filepath.FromSlash(rel))
		if info, err := os.Stat(path); err == nil && !info.IsDir() && projectPlanBootstrapActionablePlanFileV0(path) {
			return true
		}
	}
	return false
}

func projectPlanBootstrapActionablePlanFileV0(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return false
	}
	if len(data) > 64*1024 {
		data = data[:64*1024]
	}
	return projectPlanBootstrapHasStructuredBacklogV0(string(data))
}

func projectPlanBootstrapHasStructuredBacklogV0(markdown string) bool {
	lines := strings.Split(markdown, "\n")
	inBacklog := false
	inBootstrap := false
	hasOutput := false
	hasValidation := false
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		switch {
		case strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "### "):
			inBacklog = projectPlanBootstrapBacklogHeadingV0(lower)
			inBootstrap = false
		case inBacklog && strings.HasPrefix(line, "### "):
			inBootstrap = projectPlanBootstrapTaskHeadingV0(lower)
		case inBacklog && inBootstrap && strings.HasPrefix(line, "-"):
			hasOutput = hasOutput || projectPlanBootstrapOutputLineV0(lower)
			hasValidation = hasValidation || projectPlanBootstrapValidationLineV0(lower)
		}
		if inBacklog && inBootstrap && hasOutput && hasValidation {
			return true
		}
	}
	return false
}

func projectPlanBootstrapBacklogHeadingV0(lowerLine string) bool {
	return strings.Contains(lowerLine, "backlog") ||
		strings.Contains(lowerLine, "microtareas") ||
		strings.Contains(lowerLine, "plan de tareas")
}

func projectPlanBootstrapTaskHeadingV0(lowerLine string) bool {
	return strings.Contains(lowerLine, "t01") &&
		strings.Contains(lowerLine, "bootstrap") &&
		strings.Contains(lowerLine, "vertical")
}

func projectPlanBootstrapOutputLineV0(lowerLine string) bool {
	for _, required := range []string{"go.mod", "cmd/server", "internal", "web", "i18n"} {
		if strings.Contains(lowerLine, required) {
			return true
		}
	}
	return false
}

func projectPlanBootstrapValidationLineV0(lowerLine string) bool {
	return strings.Contains(lowerLine, "validacion:") &&
		strings.Contains(lowerLine, "go test ./...")
}

func projectPlanBootstrapDecisionsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	projectSlug string,
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	projectSlug = projectBacklogSlugV0(projectSlug)
	if projectSlug == "" {
		projectSlug = "app"
	}
	current := strings.TrimSpace(string(run.CurrentPhase))
	if current == "" {
		current = string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0)
	}
	decisions := []orquestadirectoragent.DirectorAgentDecisionV0{}
	contract := projectPlanBootstrapExistingContractV0(run)
	if contract.ContractRef == "" {
		decisions, current = projectPlanBootstrapAppendOpenPhaseV0(
			decisions,
			run,
			projectSlug,
			current,
			orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0,
			"Rescate autonomo: abrir decision arquitectonica minima para no quedar en planificacion.",
		)
		voteRef := projectPlanBootstrapRefV0(run, projectSlug, "vote")
		decisionRef := projectPlanBootstrapRefV0(run, projectSlug, "accepted")
		decisions = append(decisions,
			projectPlanBootstrapVoteDecisionV0(run, projectSlug, voteRef),
			projectPlanBootstrapAcceptDecisionV0(run, projectSlug, voteRef, decisionRef),
		)
		decisions, current = projectPlanBootstrapAppendOpenPhaseV0(
			decisions,
			run,
			projectSlug,
			current,
			orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
			"Rescate autonomo: abrir planificacion para publicar contrato y bootstrap ejecutable.",
		)
		contract = orquestadirectoragent.DirectorAgentFunctionContractRefV0{
			ContractRef:  "contract:function:" + projectSlug + ":bootstrap:v0",
			FunctionName: "BuildCompleteAppBootstrapV0",
		}
		decisions = append(decisions, projectPlanBootstrapPublishContractDecisionV0(
			run,
			projectSlug,
			decisionRef,
			contract,
		))
	} else if !projectPlanBootstrapCreateMicrotaskPhaseCurrentV0(current) {
		decisions, current = projectPlanBootstrapAppendOpenPhaseV0(
			decisions,
			run,
			projectSlug,
			current,
			orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
			"Rescate autonomo: reabrir planificacion para materializar bootstrap pendiente.",
		)
	}
	decisions = append(decisions, projectPlanBootstrapMicrotaskDecisionV0(run, projectSlug, contract))
	decisions, _ = projectPlanBootstrapAppendOpenPhaseV0(
		decisions,
		run,
		projectSlug,
		current,
		orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		"Rescate autonomo: abrir programacion para ejecutar bootstrap de app completa.",
	)
	return decisions
}

func projectPlanBootstrapExistingContractV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) orquestadirectoragent.DirectorAgentFunctionContractRefV0 {
	if len(compactStringsV0(run.FunctionContracts)) == 0 {
		return orquestadirectoragent.DirectorAgentFunctionContractRefV0{}
	}
	return orquestadirectoragent.DirectorAgentFunctionContractRefV0{
		ContractRef:  compactStringsV0(run.FunctionContracts)[0],
		FunctionName: "BuildCompleteAppBootstrapV0",
	}
}

func projectPlanBootstrapCreateMicrotaskPhaseCurrentV0(current string) bool {
	current = strings.TrimSpace(current)
	return current == string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0) ||
		current == string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0)
}

func projectPlanBootstrapAppendOpenPhaseV0(
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	projectSlug string,
	current string,
	target orquestacoreworkflow.OrchestrationPhaseIDV0,
	reason string,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, string) {
	targetID := string(target)
	if strings.TrimSpace(current) == targetID {
		return decisions, targetID
	}
	part := "open-" + string(target)
	ref := projectPlanBootstrapRefV0(run, projectSlug, part)
	decision := orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-project-plan-bootstrap-" + projectSlug + "-" + part + "-" + ref,
		RunID:         run.RunID,
		PhaseID:       projectPlanBootstrapCurrentPhaseV0(current),
		CommandType:   orquestadirectoragent.DirectorAgentCommandOpenPhaseV0,
		CommandRef:    "command-ref-project-plan-bootstrap-" + projectSlug + "-" + part + "-" + ref,
		Summary:       reason,
		EvidenceRefs:  []string{"evidence-ref-project-plan-bootstrap-" + projectSlug},
		OpenPhase: &orquestadirectoragent.DirectorAgentOpenPhaseCommandV0{
			PhaseID: targetID,
			Reason:  reason,
		},
	}
	return append(decisions, decision), targetID
}

func projectPlanBootstrapCurrentPhaseV0(current string) string {
	current = strings.TrimSpace(current)
	if current == "" {
		return string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0)
	}
	return current
}

func projectPlanBootstrapVoteDecisionV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	projectSlug string,
	voteRef string,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-project-plan-bootstrap-" + projectSlug + "-vote-" + voteRef,
		RunID:         run.RunID,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandRequestVoteV0,
		CommandRef:    "command-ref-project-plan-bootstrap-" + projectSlug + "-vote-" + voteRef,
		Summary:       "Rescate autonomo: fijar opcion tecnica minima para bootstrap vertical.",
		EvidenceRefs:  []string{"evidence-ref-project-plan-bootstrap-" + projectSlug},
		RequestVote: &orquestadirectoragent.DirectorAgentVoteCommandV0{
			VoteRequestID:              voteRef,
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
			DecisionTopicRef:           "topic-ref-project-plan-bootstrap-" + projectSlug,
			BrainstormRef:              "brainstorm-ref-project-plan-bootstrap-" + projectSlug,
			Summary:                    "Elegir bootstrap vertical Go con puertos, web minima, i18n y tests.",
			MinimumRecommendedCapacity: orquestadirectoragent.DirectorAgentCapacityHighV0,
			EvidenceRefs:               []string{"evidence-ref-project-plan-bootstrap-" + projectSlug},
		},
	}
}

func projectPlanBootstrapAcceptDecisionV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	projectSlug string,
	voteRef string,
	decisionRef string,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-project-plan-bootstrap-" + projectSlug + "-accept-" + decisionRef,
		RunID:         run.RunID,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandAcceptDecisionV0,
		CommandRef:    "command-ref-project-plan-bootstrap-" + projectSlug + "-accept-" + decisionRef,
		Summary:       "Rescate autonomo: aceptar bootstrap vertical minimo para continuar.",
		EvidenceRefs:  []string{"evidence-ref-project-plan-bootstrap-" + projectSlug},
		AcceptDecision: &orquestadirectoragent.DirectorAgentAcceptDecisionCommandV0{
			DecisionRef:       decisionRef,
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
			VoteRef:           voteRef,
			AcceptedOptionRef: "option-ref-project-plan-bootstrap-" + projectSlug,
			Summary:           "Bootstrap vertical con arquitectura hexagonal, i18n y go test ./....",
			EvidenceRefs:      []string{"evidence-ref-project-plan-bootstrap-" + projectSlug},
		},
	}
}

func projectPlanBootstrapPublishContractDecisionV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	projectSlug string,
	decisionRef string,
	contract orquestadirectoragent.DirectorAgentFunctionContractRefV0,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	ref := projectPlanBootstrapRefV0(run, projectSlug, "contract")
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-project-plan-bootstrap-" + projectSlug + "-contract-" + ref,
		RunID:         run.RunID,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandPublishContractV0,
		CommandRef:    "command-ref-project-plan-bootstrap-" + projectSlug + "-contract-" + ref,
		Summary:       "Publicar contrato minimo recuperado para app completa.",
		EvidenceRefs:  []string{"evidence-ref-project-plan-bootstrap-" + projectSlug},
		PublishContract: &orquestadirectoragent.DirectorAgentPublishContractCommandV0{
			ContractRef:   contract.ContractRef,
			PhaseID:       string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0),
			DecisionRef:   decisionRef,
			Summary:       "Contrato para bootstrap vertical de app completa.",
			FunctionNames: []string{contract.FunctionName},
			EvidenceRefs:  []string{"evidence-ref-project-plan-bootstrap-" + projectSlug},
		},
	}
}

func projectPlanBootstrapMicrotaskDecisionV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	projectSlug string,
	contract orquestadirectoragent.DirectorAgentFunctionContractRefV0,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	taskID := "task-" + projectSlug + "-bootstrap-vertical"
	ref := projectPlanBootstrapRefV0(run, projectSlug, "task")
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-project-plan-bootstrap-" + projectSlug + "-task-" + ref,
		RunID:         run.RunID,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0,
		CommandRef:    "command-ref-project-plan-bootstrap-" + projectSlug + "-task-" + ref,
		Summary:       "Materializar bootstrap vertical porque la planificacion inicial no creo microtareas.",
		EvidenceRefs:  []string{"evidence-ref-project-plan-bootstrap-" + projectSlug},
		CreateMicrotask: &orquestadirectoragent.DirectorAgentCreateMicrotaskCommandV0{
			Task: orquestadirectoragent.DirectorAgentMicrotaskV0{
				SchemaVersion:        orquestadirectoragent.DirectorAgentMicrotaskSchemaVersionV0,
				TaskID:               taskID,
				RunID:                run.RunID,
				PhaseID:              string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
				WorkProfileKind:      string(orquestacoreworkflow.WorkProfileImplementationV0),
				Title:                "Bootstrap vertical de app completa",
				Summary:              "Crear una app ejecutable de cero con estructura Go, API HTTP, web minima, i18n, README y pruebas.",
				WriteSet:             []string{"go.mod", "cmd/server", "internal", "web", "i18n", "docs", "README.md"},
				AcceptanceCriteria:   projectPlanBootstrapAcceptanceCriteriaV0(projectSlug),
				RequiredTests:        []string{"go test ./..."},
				ContextRefs:          []string{"context-ref-project-plan-bootstrap-" + projectSlug},
				CohortRef:            "cohort-ref-project-plan-bootstrap-" + projectSlug,
				WaveRef:              "wave-ref-project-plan-bootstrap-" + projectSlug,
				MaxChildAgents:       projectBacklogMaxSubagentsV0,
				FunctionContractRefs: []orquestadirectoragent.DirectorAgentFunctionContractRefV0{contract},
			},
		},
	}
}

func projectPlanBootstrapAcceptanceCriteriaV0(projectSlug string) []string {
	return []string{
		"objetivo_actual: crear app completa " + projectSlug + " bootstrap vertical",
		"crear estructura Go ejecutable con go.mod y cmd/server",
		"separar dominio, aplicacion y adaptadores en internal",
		"exponer API HTTP minima y web usable cuando aplique",
		"usar catalogos i18n es/en para texto visible basico",
		"documentar ejecucion local, decisiones y pruebas en README.md/docs",
		"validacion obligatoria: go test ./...",
		"no borrar ni mover trabajo existente sin evidencia y permiso explicito",
	}
}

func projectPlanBootstrapSlugV0(
	projectWorkDir string,
	run orquestacoreworkflow.OrchestrationRunV0,
) string {
	for _, value := range []string{
		filepath.Base(strings.TrimSpace(projectWorkDir)),
		run.ProjectRef,
		run.AppSpecRef,
		run.RunID,
	} {
		if slug := projectBacklogSlugV0(value); slug != "" && slug != "." {
			return slug
		}
	}
	return "app"
}

func projectPlanBootstrapRefV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	projectSlug string,
	part string,
) string {
	return codexStackDeterministicDigestV0(run.RunID + ":" + projectSlug + ":project-plan-bootstrap:" + strings.TrimSpace(part))
}
