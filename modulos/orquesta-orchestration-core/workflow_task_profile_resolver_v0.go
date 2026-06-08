package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadecisioncouncil "orquesta/modulos/orquesta-decision-council"
)

type WorkflowTaskProfileResolverPortV0 interface {
	ResolveWorkflowTaskProfileV0(
		ctx context.Context,
		request WorkflowTaskProfileRequestV0,
	) (WorkflowTaskProfileResolutionV0, error)
}

type WorkflowTaskProfileRequestV0 struct {
	Run             orquestacoreworkflow.OrchestrationRunV0
	Task            orquestacoreworkflow.WorkflowTaskV0
	DefaultCapacity orquestacoreworkflow.OrchestrationCapacityRecommendationV0
}

type WorkflowTaskProfileResolutionV0 struct {
	ProfileKind                orquestacoreworkflow.WorkProfileKindV0
	Role                       string
	ReasonCode                 string
	CapacitySummary            string
	AgentSummary               string
	MinimumRecommendedCapacity orquestacoreworkflow.OrchestrationCapacityRecommendationV0
	EvidenceRefs               []string
	SkillRefs                  []string
}

type DefaultWorkflowTaskProfileResolverV0 struct{}

var _ WorkflowTaskProfileResolverPortV0 = DefaultWorkflowTaskProfileResolverV0{}

func (DefaultWorkflowTaskProfileResolverV0) ResolveWorkflowTaskProfileV0(
	_ context.Context,
	request WorkflowTaskProfileRequestV0,
) (WorkflowTaskProfileResolutionV0, error) {
	if councilRole := decisionCouncilRoleFromContextRefsV0(request.Task.ContextRefs); councilRole != "" {
		return workflowTaskCouncilProfileResolutionV0(councilRole, request.DefaultCapacity), nil
	}
	kind := workflowTaskProfileKindV0(request.Task)
	return workflowTaskProfileResolutionForKindV0(kind, request.DefaultCapacity), nil
}

func normalizeWorkflowTaskProfileResolutionV0(
	request WorkflowTaskProfileRequestV0,
	resolution WorkflowTaskProfileResolutionV0,
) (WorkflowTaskProfileResolutionV0, error) {
	kind := orquestacoreworkflow.NormalizeWorkProfileKindV0(resolution.ProfileKind)
	if strings.TrimSpace(string(kind)) == "" {
		kind = workflowTaskProfileKindV0(request.Task)
	}
	if _, ok := orquestacoreworkflow.LookupWorkProfileDefinitionV0(kind); !ok {
		return WorkflowTaskProfileResolutionV0{}, errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"workflow_task_profile",
			"perfil de trabajo no soportado",
		)
	}
	defaults := workflowTaskProfileResolutionForKindV0(kind, request.DefaultCapacity)
	resolution.ProfileKind = kind
	resolution.Role = firstNonEmptyV0(resolution.Role, defaults.Role)
	resolution.ReasonCode = firstNonEmptyV0(resolution.ReasonCode, defaults.ReasonCode)
	resolution.CapacitySummary = firstNonEmptyV0(resolution.CapacitySummary, defaults.CapacitySummary)
	resolution.AgentSummary = firstNonEmptyV0(resolution.AgentSummary, defaults.AgentSummary)
	if strings.TrimSpace(string(resolution.MinimumRecommendedCapacity)) == "" {
		resolution.MinimumRecommendedCapacity = defaults.MinimumRecommendedCapacity
	}
	resolution.EvidenceRefs = compactStringsV0(resolution.EvidenceRefs)
	resolution.SkillRefs = compactStringsV0(append(
		append([]string(nil), defaults.SkillRefs...),
		append(resolution.SkillRefs, request.Task.SkillRefs...)...,
	))
	return resolution, nil
}

func workflowTaskProfileKindV0(
	task orquestacoreworkflow.WorkflowTaskV0,
) orquestacoreworkflow.WorkProfileKindV0 {
	kind := orquestacoreworkflow.NormalizeWorkProfileKindV0(task.WorkProfileKind)
	if _, ok := orquestacoreworkflow.LookupWorkProfileDefinitionV0(kind); ok {
		return kind
	}
	switch task.PhaseID {
	case orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0:
		return orquestacoreworkflow.WorkProfileCodeStudyV0
	case orquestacoreworkflow.OrchestrationPhaseDocumentacionV0:
		return orquestacoreworkflow.WorkProfileDocumentationV0
	case orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0:
		return orquestacoreworkflow.WorkProfileReviewV0
	default:
		return orquestacoreworkflow.WorkProfileImplementationV0
	}
}

func workflowTaskProfileResolutionForKindV0(
	kind orquestacoreworkflow.WorkProfileKindV0,
	defaultCapacity orquestacoreworkflow.OrchestrationCapacityRecommendationV0,
) WorkflowTaskProfileResolutionV0 {
	capacity := defaultCapacity
	if strings.TrimSpace(string(capacity)) == "" {
		capacity = workflowTaskProfileDefaultCapacityV0(kind)
	}
	return WorkflowTaskProfileResolutionV0{
		ProfileKind:                kind,
		Role:                       workflowTaskProfileRoleV0(kind),
		ReasonCode:                 workflowTaskProfileReasonCodeV0(kind),
		CapacitySummary:            workflowTaskProfileCapacitySummaryV0(kind),
		AgentSummary:               workflowTaskProfileAgentSummaryV0(kind),
		MinimumRecommendedCapacity: capacity,
		SkillRefs:                  workflowTaskProfileSkillRefsV0(kind),
	}
}

func workflowTaskCouncilProfileResolutionV0(
	role string,
	defaultCapacity orquestacoreworkflow.OrchestrationCapacityRecommendationV0,
) WorkflowTaskProfileResolutionV0 {
	capacity := defaultCapacity
	if strings.TrimSpace(string(capacity)) == "" {
		capacity = orquestacoreworkflow.OrchestrationCapacityHighV0
	}
	kind := orquestacoreworkflow.WorkProfileCodeStudyV0
	if role == orquestadecisioncouncil.CouncilRoleVoteV0 {
		kind = orquestacoreworkflow.WorkProfileReviewV0
	}
	return WorkflowTaskProfileResolutionV0{
		ProfileKind:                kind,
		Role:                       role,
		ReasonCode:                 "decision_council_" + decisionCouncilRoleScopeV0(role),
		CapacitySummary:            "Capacidad para ronda de consejo multiagente.",
		AgentSummary:               "Ronda de consejo lista.",
		MinimumRecommendedCapacity: capacity,
		EvidenceRefs:               []string{decisionCouncilRoleContextRefV0(role)},
		SkillRefs:                  []string{"skill-ref-orquesta-revision-consejo-votacion-v0"},
	}
}

func workflowTaskProfileSkillRefsV0(kind orquestacoreworkflow.WorkProfileKindV0) []string {
	switch kind {
	case orquestacoreworkflow.WorkProfileCodeStudyV0:
		return []string{"skill-ref-orquesta-ordenacion-trabajo-v0"}
	case orquestacoreworkflow.WorkProfileRefactorV0:
		return []string{"skill-ref-orquesta-programacion-integracion-v0"}
	case orquestacoreworkflow.WorkProfileRequiredTestsV0:
		return []string{"skill-ref-orquesta-programacion-tests-v0"}
	case orquestacoreworkflow.WorkProfileDocumentationV0:
		return []string{"skill-ref-orquesta-artefacto-modular-v0"}
	case orquestacoreworkflow.WorkProfileReviewV0:
		return []string{"skill-ref-orquesta-programacion-revision-v0"}
	case orquestacoreworkflow.WorkProfileDomainWorkV0:
		return []string{"skill-ref-orquesta-ordenacion-trabajo-v0"}
	default:
		return []string{
			"skill-ref-orquesta-programacion-autonoma-v0",
			"skill-ref-orquesta-programacion-integracion-v0",
		}
	}
}

func workflowTaskProfileReasonCodeV0(kind orquestacoreworkflow.WorkProfileKindV0) string {
	if kind == orquestacoreworkflow.WorkProfileImplementationV0 {
		return "programacion_siguiente_paso"
	}
	return "work_profile_" + string(kind)
}

func workflowTaskProfileDefaultCapacityV0(
	kind orquestacoreworkflow.WorkProfileKindV0,
) orquestacoreworkflow.OrchestrationCapacityRecommendationV0 {
	switch kind {
	case orquestacoreworkflow.WorkProfileCodeStudyV0,
		orquestacoreworkflow.WorkProfileRefactorV0,
		orquestacoreworkflow.WorkProfileRequiredTestsV0,
		orquestacoreworkflow.WorkProfileReviewV0,
		orquestacoreworkflow.WorkProfileDomainWorkV0:
		return orquestacoreworkflow.OrchestrationCapacityHighV0
	default:
		return orquestacoreworkflow.OrchestrationCapacityMediumV0
	}
}

func workflowTaskProfileRoleV0(kind orquestacoreworkflow.WorkProfileKindV0) string {
	switch kind {
	case orquestacoreworkflow.WorkProfileCodeStudyV0:
		return "analisis"
	case orquestacoreworkflow.WorkProfileRefactorV0:
		return "refactor"
	case orquestacoreworkflow.WorkProfileRequiredTestsV0:
		return "pruebas"
	case orquestacoreworkflow.WorkProfileDocumentationV0:
		return "documentacion"
	case orquestacoreworkflow.WorkProfileReviewV0:
		return "revision"
	case orquestacoreworkflow.WorkProfileDomainWorkV0:
		return "dominio"
	default:
		return "implementacion"
	}
}

func workflowTaskProfileCapacitySummaryV0(kind orquestacoreworkflow.WorkProfileKindV0) string {
	switch kind {
	case orquestacoreworkflow.WorkProfileCodeStudyV0:
		return "Capacidad para estudio de codigo acotado."
	case orquestacoreworkflow.WorkProfileRefactorV0:
		return "Capacidad para refactor acotado."
	case orquestacoreworkflow.WorkProfileRequiredTestsV0:
		return "Capacidad para pruebas requeridas."
	case orquestacoreworkflow.WorkProfileDocumentationV0:
		return "Capacidad para documentacion acotada."
	case orquestacoreworkflow.WorkProfileReviewV0:
		return "Capacidad para revision acotada."
	case orquestacoreworkflow.WorkProfileDomainWorkV0:
		return "Capacidad para trabajo de dominio por refs."
	default:
		return "Capacidad para microtarea acotada."
	}
}

func workflowTaskProfileAgentSummaryV0(kind orquestacoreworkflow.WorkProfileKindV0) string {
	switch kind {
	case orquestacoreworkflow.WorkProfileCodeStudyV0:
		return "Estudio de codigo listo."
	case orquestacoreworkflow.WorkProfileRefactorV0:
		return "Refactor acotado listo."
	case orquestacoreworkflow.WorkProfileRequiredTestsV0:
		return "Pruebas requeridas listas."
	case orquestacoreworkflow.WorkProfileDocumentationV0:
		return "Documentacion acotada lista."
	case orquestacoreworkflow.WorkProfileReviewV0:
		return "Revision acotada lista."
	case orquestacoreworkflow.WorkProfileDomainWorkV0:
		return "Trabajo de dominio listo."
	default:
		return "Microtarea acotada lista."
	}
}

func firstNonEmptyV0(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return fallback
}
