package orquestaweb

import (
	"strings"

	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

type WebArrancarDirectorAppResultV0 struct {
	Estado                string                            `json:"estado"`
	RequestID             string                            `json:"request_id,omitempty"`
	CorrelationID         string                            `json:"correlation_id,omitempty"`
	RunRef                string                            `json:"run_ref,omitempty"`
	PhaseID               string                            `json:"phase_id,omitempty"`
	DirectorExecutionMode string                            `json:"director_execution_mode,omitempty"`
	DirectorTask          WebNuevaAppDirectorTaskV0         `json:"director_task,omitempty"`
	DirectorTasks         []WebNuevaAppDirectorTaskV0       `json:"director_tasks,omitempty"`
	LoopStatus            string                            `json:"loop_status,omitempty"`
	StartedAgents         []string                          `json:"started_agents,omitempty"`
	GoalRef               string                            `json:"goal_ref,omitempty"`
	ExternalGoalRef       string                            `json:"external_goal_ref,omitempty"`
	GoalStatus            string                            `json:"goal_status,omitempty"`
	GoalLaunchReceipt     *orquestagoal.GoalLaunchReceiptV0 `json:"goal_launch_receipt,omitempty"`
	Errores               []WebNuevaAppIssueV0              `json:"errores_publicos,omitempty"`
	EvidenceRefs          []string                          `json:"evidence_refs,omitempty"`
}

func NewWebNuevaAppDirectorViewModelV0(
	form WebNuevaAppFormV0,
	result WebArrancarDirectorAppResultV0,
) WebNuevaAppViewModelV0 {
	request := form.ToAppSpecRequestV0()
	tasks := compactDirectorTasksV0(result.DirectorTask, result.DirectorTasks)
	return WebNuevaAppViewModelV0{
		RequestID:         firstNuevaAppValueV0(result.RequestID, request.RequestID),
		Locale:            request.Locale,
		Estado:            WebNuevaAppEstadoDirector,
		ResumenApp:        resumenFromAppSpecRequestV0(request),
		BacklogPreview:    emptyBacklogPreviewV0(),
		PreguntasAbiertas: []string{},
		Fases:             []WebNuevaAppFaseV0{},
		Microtareas:       []WebNuevaAppMicrotareaV0{},
		Director: &WebNuevaAppDirectorV0{
			RunRef:                trimV0(result.RunRef),
			PhaseID:               trimV0(result.PhaseID),
			DirectorExecutionMode: trimV0(result.DirectorExecutionMode),
			LoopStatus:            trimV0(result.LoopStatus),
			DirectorTasks:         tasks,
			StartedAgents:         compactStringsV0(result.StartedAgents),
			GoalRef:               trimV0(result.GoalRef),
			ExternalGoalRef:       trimV0(result.ExternalGoalRef),
			GoalStatus:            trimV0(result.GoalStatus),
			GoalLaunchReceipt:     result.GoalLaunchReceipt,
			EvidenceRefs:          compactStringsV0(result.EvidenceRefs),
		},
	}
}

func NewWebNuevaAppDirectorErrorViewModelV0(
	form WebNuevaAppFormV0,
	result WebArrancarDirectorAppResultV0,
) WebNuevaAppViewModelV0 {
	request := form.ToAppSpecRequestV0()
	return WebNuevaAppViewModelV0{
		RequestID:           firstNuevaAppValueV0(result.RequestID, request.RequestID),
		Locale:              request.Locale,
		Estado:              WebNuevaAppEstadoInvalida,
		ResumenApp:          resumenFromAppSpecRequestV0(request),
		BacklogPreview:      emptyBacklogPreviewV0(),
		ErroresPublicos:     compactNuevaAppIssuesV0(result.Errores, request.Locale),
		PreguntasAbiertas:   []string{},
		Fases:               []WebNuevaAppFaseV0{},
		Microtareas:         []WebNuevaAppMicrotareaV0{},
		ContratosRequeridos: []string{},
		Riesgos:             []string{},
	}
}

func NewWebNuevaAppGoalPreviewViewModelV0(
	form WebNuevaAppFormV0,
	result WebPreviewDirectorAppResultV0,
) WebNuevaAppViewModelV0 {
	request := form.ToAppSpecRequestV0()
	summary := result.GoalSpecSummary
	return WebNuevaAppViewModelV0{
		RequestID:         firstNuevaAppValueV0(result.RequestID, request.RequestID),
		Locale:            request.Locale,
		Estado:            WebNuevaAppEstadoGoalPreview,
		ResumenApp:        resumenFromAppSpecRequestV0(request),
		BacklogPreview:    emptyBacklogPreviewV0(),
		PreguntasAbiertas: []string{},
		Fases:             []WebNuevaAppFaseV0{},
		Microtareas:       []WebNuevaAppMicrotareaV0{},
		GoalPreview: &WebNuevaAppGoalPreviewV0{
			RunRef:                trimV0(firstNuevaAppValueV0(result.RunRef, summary.RunRef)),
			GoalRef:               trimV0(summary.GoalRef),
			DirectorExecutionMode: trimV0(result.DirectorExecutionMode),
			DirectorKind:          trimV0(summary.DirectorKind),
			SpecHash:              trimV0(summary.SpecHash),
			ContextRefs:           compactStringsV0(summary.ContextRefs),
			RuleRefs:              compactStringsV0(summary.RuleRefs),
			RequiredTestRefs:      compactStringsV0(summary.RequiredTestRefs),
			ArtifactTypes:         compactStringsV0(summary.ArtifactTypes),
			SpecSummary:           summary,
			Estimate:              result.Estimate,
			EvidenceRefs:          compactStringsV0(result.EvidenceRefs),
		},
	}
}

func NewWebNuevaAppGoalPreviewErrorViewModelV0(
	form WebNuevaAppFormV0,
	result WebPreviewDirectorAppResultV0,
) WebNuevaAppViewModelV0 {
	request := form.ToAppSpecRequestV0()
	return WebNuevaAppViewModelV0{
		RequestID:           firstNuevaAppValueV0(result.RequestID, request.RequestID),
		Locale:              request.Locale,
		Estado:              WebNuevaAppEstadoInvalida,
		ResumenApp:          resumenFromAppSpecRequestV0(request),
		BacklogPreview:      emptyBacklogPreviewV0(),
		ErroresPublicos:     compactNuevaAppIssuesV0(append(result.Errores, goalIssuesToNuevaAppIssuesV0(result.GoalSpecIssues)...), request.Locale),
		PreguntasAbiertas:   []string{},
		Fases:               []WebNuevaAppFaseV0{},
		Microtareas:         []WebNuevaAppMicrotareaV0{},
		ContratosRequeridos: []string{},
		Riesgos:             []string{},
	}
}

func goalIssuesToNuevaAppIssuesV0(values []orquestagoal.GoalWorkIssueV0) []WebNuevaAppIssueV0 {
	out := make([]WebNuevaAppIssueV0, 0, len(values))
	for _, value := range values {
		code := trimV0(value.Code)
		if code == "" {
			continue
		}
		out = append(out, WebNuevaAppIssueV0{
			Code:    code,
			Field:   trimV0(value.Field),
			Message: code,
		})
	}
	if out == nil {
		return []WebNuevaAppIssueV0{}
	}
	return out
}

func resumenFromAppSpecRequestV0(request orquestafactory.AppSpecRequestV0) WebNuevaAppResumenV0 {
	return WebNuevaAppResumenV0{
		Nombre:        trimV0(request.Nombre),
		Objetivo:      trimV0(request.Objetivo),
		Descripcion:   trimV0(request.Descripcion),
		TipoApp:       trimV0(request.TipoApp),
		RequestKind:   trimV0(request.RequestKind),
		ExecutionMode: trimV0(request.ExecutionMode),
		Locale:        trimV0(request.Locale),
		DefaultLocale: trimV0(request.I18N.DefaultLocale),
		I18NEnabled:   request.I18N.Enabled == nil || *request.I18N.Enabled,
		Plataformas:   compactStringsV0(request.Plataformas),
		DeployTarget:  trimV0(request.Deploy.Target),
	}
}

func compactDirectorTasksV0(
	first WebNuevaAppDirectorTaskV0,
	values []WebNuevaAppDirectorTaskV0,
) []WebNuevaAppDirectorTaskV0 {
	out := make([]WebNuevaAppDirectorTaskV0, 0, len(values)+1)
	for _, value := range append([]WebNuevaAppDirectorTaskV0{first}, values...) {
		value = normalizeDirectorTaskV0(value)
		if value.AgentRequestID == "" && value.TaskRef == "" {
			continue
		}
		out = append(out, value)
	}
	if out == nil {
		return []WebNuevaAppDirectorTaskV0{}
	}
	return out
}

func normalizeDirectorTaskV0(value WebNuevaAppDirectorTaskV0) WebNuevaAppDirectorTaskV0 {
	return WebNuevaAppDirectorTaskV0{
		TaskRef:        trimV0(value.TaskRef),
		BrainstormRef:  trimV0(value.BrainstormRef),
		AgentRequestID: trimV0(value.AgentRequestID),
		Capacity:       trimV0(value.Capacity),
	}
}

func compactNuevaAppIssuesV0(values []WebNuevaAppIssueV0, locale string) []WebNuevaAppIssueV0 {
	out := make([]WebNuevaAppIssueV0, 0, len(values))
	for _, value := range values {
		code := trimV0(value.Code)
		if code == "" {
			continue
		}
		out = append(out, WebNuevaAppIssueV0{
			Code:    code,
			Field:   trimV0(value.Field),
			Message: nuevaAppPublicIssueMessageV0(locale, code, value.Message),
		})
	}
	if out == nil {
		return []WebNuevaAppIssueV0{}
	}
	return out
}

func firstNuevaAppValueV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
