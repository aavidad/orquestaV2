package orquestacoreworkflow

import "strings"

const WorkProfileSchemaVersionV0 = "work_profile.v0"

const (
	WorkProfileCodeStudyV0      WorkProfileKindV0 = "code_study"
	WorkProfileImplementationV0 WorkProfileKindV0 = "implementation"
	WorkProfileRefactorV0       WorkProfileKindV0 = "refactor"
	WorkProfileRequiredTestsV0  WorkProfileKindV0 = "required_tests"
	WorkProfileDocumentationV0  WorkProfileKindV0 = "documentation"
	WorkProfileReviewV0         WorkProfileKindV0 = "review"
	WorkProfileDomainWorkV0     WorkProfileKindV0 = "domain_work"
)

const (
	ErrWorkProfileInvalidoV0     = "work_profile_invalido"
	ErrWorkProfileTaskInvalidaV0 = "work_profile_task_invalida"
)

type WorkProfileKindV0 string

type WorkProfileV0 struct {
	SchemaVersion        string                          `json:"schema_version"`
	ProfileRef           string                          `json:"profile_ref"`
	ProfileKind          WorkProfileKindV0               `json:"profile_kind"`
	TaskRef              string                          `json:"task_ref"`
	RunRef               string                          `json:"run_ref"`
	PhaseID              OrchestrationPhaseIDV0          `json:"phase_id,omitempty"`
	Title                string                          `json:"title"`
	Objective            string                          `json:"objective"`
	Summary              string                          `json:"summary,omitempty"`
	ScopeRefs            []string                        `json:"scope_refs"`
	AcceptanceCriteria   []string                        `json:"acceptance_criteria,omitempty"`
	RequiredTests        []string                        `json:"required_tests,omitempty"`
	DependsOn            []string                        `json:"depends_on,omitempty"`
	ParentTaskRef        string                          `json:"parent_task_ref,omitempty"`
	CohortRef            string                          `json:"cohort_ref,omitempty"`
	WaveRef              string                          `json:"wave_ref,omitempty"`
	DelegationDepth      int                             `json:"delegation_depth,omitempty"`
	MaxChildAgents       int                             `json:"max_child_agents,omitempty"`
	ChildTaskRefs        []string                        `json:"child_task_refs,omitempty"`
	FunctionContractRefs []WorkflowFunctionContractRefV0 `json:"function_contract_refs,omitempty"`
}

type WorkProfileDefinitionV0 struct {
	Kind                  WorkProfileKindV0      `json:"kind"`
	DefaultPhaseID        OrchestrationPhaseIDV0 `json:"default_phase_id"`
	RequiredTestsRequired bool                   `json:"required_tests_required"`
	DefaultCriteria       []string               `json:"default_criteria,omitempty"`
}

type WorkProfileErrorV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

func (err WorkProfileErrorV0) Error() string {
	return err.Code
}

func NewWorkProfileV0(profile WorkProfileV0) (WorkProfileV0, error) {
	normalized := NormalizeWorkProfileV0(profile)
	if err := ValidateWorkProfileV0(normalized); err != nil {
		return WorkProfileV0{}, err
	}
	return normalized, nil
}

func NormalizeWorkProfileV0(profile WorkProfileV0) WorkProfileV0 {
	kind := NormalizeWorkProfileKindV0(profile.ProfileKind)
	phaseID := profile.PhaseID
	if phaseID == "" {
		if definition, ok := LookupWorkProfileDefinitionV0(kind); ok {
			phaseID = definition.DefaultPhaseID
		}
	}
	return WorkProfileV0{
		SchemaVersion:        strings.TrimSpace(profile.SchemaVersion),
		ProfileRef:           strings.TrimSpace(profile.ProfileRef),
		ProfileKind:          kind,
		TaskRef:              strings.TrimSpace(profile.TaskRef),
		RunRef:               strings.TrimSpace(profile.RunRef),
		PhaseID:              phaseID,
		Title:                strings.TrimSpace(profile.Title),
		Objective:            strings.TrimSpace(profile.Objective),
		Summary:              strings.TrimSpace(profile.Summary),
		ScopeRefs:            compactStringsV0(profile.ScopeRefs),
		AcceptanceCriteria:   compactStringsV0(profile.AcceptanceCriteria),
		RequiredTests:        compactStringsV0(profile.RequiredTests),
		DependsOn:            compactStringsV0(profile.DependsOn),
		ParentTaskRef:        strings.TrimSpace(profile.ParentTaskRef),
		CohortRef:            strings.TrimSpace(profile.CohortRef),
		WaveRef:              strings.TrimSpace(profile.WaveRef),
		DelegationDepth:      profile.DelegationDepth,
		MaxChildAgents:       profile.MaxChildAgents,
		ChildTaskRefs:        compactStringsV0(profile.ChildTaskRefs),
		FunctionContractRefs: normalizeWorkflowFunctionContractRefsV0(profile.FunctionContractRefs),
	}
}

func NormalizeWorkProfileKindV0(kind WorkProfileKindV0) WorkProfileKindV0 {
	value := strings.ToLower(strings.TrimSpace(string(kind)))
	value = strings.ReplaceAll(value, "-", "_")
	switch value {
	case "study_code", "code_analysis", "estudio_codigo", "analisis_codigo":
		return WorkProfileCodeStudyV0
	case "programming", "programacion", "implementacion":
		return WorkProfileImplementationV0
	case "refactorizacion":
		return WorkProfileRefactorV0
	case "tests", "testing", "pruebas", "run_tests":
		return WorkProfileRequiredTestsV0
	case "docs", "documentacion":
		return WorkProfileDocumentationV0
	case "revision":
		return WorkProfileReviewV0
	case "external_work", "trabajo_dominio":
		return WorkProfileDomainWorkV0
	default:
		return WorkProfileKindV0(value)
	}
}

func WorkProfileDefinitionsV0() []WorkProfileDefinitionV0 {
	return []WorkProfileDefinitionV0{
		{
			Kind:           WorkProfileCodeStudyV0,
			DefaultPhaseID: OrchestrationPhaseBrainstormingArquitecturaV0,
			DefaultCriteria: []string{
				"Mapa de componentes, riesgos y puntos de cambio documentado.",
				"Alcance de modificacion posterior expresado por refs compactas.",
			},
		},
		{
			Kind:                  WorkProfileImplementationV0,
			DefaultPhaseID:        OrchestrationPhaseProgramacionV0,
			RequiredTestsRequired: true,
			DefaultCriteria: []string{
				"Cambio implementado dentro del write-set declarado.",
				"ACK declara pruebas focales ejecutadas y resultado.",
			},
		},
		{
			Kind:                  WorkProfileRefactorV0,
			DefaultPhaseID:        OrchestrationPhaseProgramacionV0,
			RequiredTestsRequired: true,
			DefaultCriteria: []string{
				"Comportamiento publico conservado salvo cambio aceptado.",
				"El cambio reduce complejidad sin ampliar alcance no solicitado.",
			},
		},
		{
			Kind:                  WorkProfileRequiredTestsV0,
			DefaultPhaseID:        OrchestrationPhaseProgramacionV0,
			RequiredTestsRequired: true,
			DefaultCriteria: []string{
				"Evidencia durable de pruebas requerida para cierre.",
				"Fallo de prueba queda explicado con refs causales.",
			},
		},
		{
			Kind:           WorkProfileDocumentationV0,
			DefaultPhaseID: OrchestrationPhaseDocumentacionV0,
			DefaultCriteria: []string{
				"Documentacion actualizada con alcance y evidencias compactas.",
				"No introduce decisiones de entorno concreto.",
			},
		},
		{
			Kind:           WorkProfileReviewV0,
			DefaultPhaseID: OrchestrationPhaseRevisionV0,
			DefaultCriteria: []string{
				"Revision registra aceptacion o cambios requeridos con evidencias.",
				"Hallazgos priorizados por riesgo e impacto.",
			},
		},
		{
			Kind:           WorkProfileDomainWorkV0,
			DefaultPhaseID: OrchestrationPhaseProgramacionV0,
			DefaultCriteria: []string{
				"Trabajo de dominio resuelto solo por refs y contrato externo.",
				"Artefacto devuelto sin exponer internals de la app propietaria.",
			},
		},
	}
}

func LookupWorkProfileDefinitionV0(kind WorkProfileKindV0) (WorkProfileDefinitionV0, bool) {
	normalized := NormalizeWorkProfileKindV0(kind)
	for _, definition := range WorkProfileDefinitionsV0() {
		if definition.Kind == normalized {
			return definition, true
		}
	}
	return WorkProfileDefinitionV0{}, false
}
