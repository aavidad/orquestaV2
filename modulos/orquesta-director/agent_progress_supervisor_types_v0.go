package orquestadirector

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

const (
	ErrDirectorAgentProgressSupervisionInvalidaV0 = "director_agent_progress_supervision_invalida"
	agentProgressSupervisorSourceGroupV0          = "orquesta-director"
	stalledHighNoProgressTicksV0                  = 5
	stalledHighRepeatedActionCountV0              = 3
)

type AgentProgressSupervisionInputV0 struct {
	CommandMeta   orquestacoreworkflow.OrchestrationCommandMetaV0 `json:"command_meta"`
	Report        orquestaruntime.AgentProgressReportV0           `json:"report"`
	PhaseID       string                                          `json:"phase_id"`
	TaskRef       string                                          `json:"task_ref,omitempty"`
	DeliveryRef   string                                          `json:"delivery_ref,omitempty"`
	AssessmentRef string                                          `json:"assessment_ref"`
	QuestionID    string                                          `json:"question_id,omitempty"`
	StopAllowed   *bool                                           `json:"stop_allowed,omitempty"`
}

type AgentProgressSupervisionResultV0 struct {
	AssessCommand      orquestacoreworkflow.OrchestrationCommandV0  `json:"assess_command"`
	AskDirectorCommand *orquestacoreworkflow.OrchestrationCommandV0 `json:"ask_director_command,omitempty"`
}

type AgentProgressSupervisionErrorV0 struct {
	Code          string   `json:"code"`
	Message       string   `json:"message"`
	Field         string   `json:"field,omitempty"`
	Retryable     bool     `json:"retryable"`
	Evidence      []string `json:"evidence,omitempty"`
	CorrelationID string   `json:"correlation_id,omitempty"`
}

func (err AgentProgressSupervisionErrorV0) Error() string {
	return err.Code
}
