package orquestadirectorsupervisedburst

import "strings"

import (
	"context"

	orquestadirectorcycle "orquesta/modulos/orquesta-director-cycle"
	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

const (
	ErrDirectorSupervisedBurstInvalidoV0   = "director_supervised_burst_invalido"
	ErrDirectorSupervisedBurstStepInputV0  = "director_supervised_burst_step_input"
	ErrDirectorSupervisedBurstStepV0       = "director_supervised_burst_step"
	ErrDirectorSupervisedBurstSupervisorV0 = "director_supervised_burst_supervisor"
)

type DirectorCycleStepInputBuilderPortV0 interface {
	BuildDirectorCycleStepInputV0(
		ctx context.Context,
		request DirectorSupervisedBurstStepRequestV0,
	) (orquestadirectorcycle.DirectorCycleStepInputV0, error)
}

type DirectorCycleStepExecutorPortV0 interface {
	ExecuteDirectorCycleStepV0(
		ctx context.Context,
		input orquestadirectorcycle.DirectorCycleStepInputV0,
	) (orquestadirectorcycle.DirectorCycleStepResultV0, error)
}

type DirectorSupervisorPolicyPortV0 interface {
	DecideDirectorSupervisorNextActionV0(
		input orquestadirectorsupervisor.DirectorSupervisorDecisionInputV0,
	) (orquestadirectorsupervisor.DirectorSupervisorDecisionV0, error)
}

type DirectorSupervisedBurstInputV0 struct {
	StepInputBuilder DirectorCycleStepInputBuilderPortV0 `json:"-"`
	StepExecutor     DirectorCycleStepExecutorPortV0     `json:"-"`
	Supervisor       DirectorSupervisorPolicyPortV0      `json:"-"`
	RunRef           string                              `json:"run_ref"`
	MaxSteps         int                                 `json:"max_steps"`
	CorrelationID    string                              `json:"correlation_id,omitempty"`
	EvidenceRefs     []string                            `json:"evidence_refs,omitempty"`
}

type DirectorSupervisedBurstStepRequestV0 struct {
	RunRef             string                                                   `json:"run_ref"`
	StepNumber         int                                                      `json:"step_number"`
	MaxSteps           int                                                      `json:"max_steps"`
	PreviousStepResult *orquestadirectorcycle.DirectorCycleStepResultV0         `json:"previous_step_result,omitempty"`
	PreviousDecision   *orquestadirectorsupervisor.DirectorSupervisorDecisionV0 `json:"previous_decision,omitempty"`
	CorrelationID      string                                                   `json:"correlation_id,omitempty"`
	EvidenceRefs       []string                                                 `json:"evidence_refs,omitempty"`
}

type DirectorSupervisedBurstResultV0 struct {
	RunRef        string                                                `json:"run_ref"`
	MaxSteps      int                                                   `json:"max_steps"`
	ExecutedSteps int                                                   `json:"executed_steps"`
	FinalAction   orquestadirectorsupervisor.DirectorSupervisorActionV0 `json:"final_action,omitempty"`
	Steps         []DirectorSupervisedBurstStepResultV0                 `json:"steps,omitempty"`
	Issues        []DirectorSupervisedBurstIssueV0                      `json:"issues,omitempty"`
	EvidenceRefs  []string                                              `json:"evidence_refs,omitempty"`
}

type DirectorSupervisedBurstStepResultV0 struct {
	StepNumber   int                                                   `json:"step_number"`
	CycleRef     string                                                `json:"cycle_ref,omitempty"`
	TickRef      string                                                `json:"tick_ref,omitempty"`
	CycleStatus  string                                                `json:"cycle_status,omitempty"`
	Action       orquestadirectorsupervisor.DirectorSupervisorActionV0 `json:"action,omitempty"`
	ErrorCode    string                                                `json:"error_code,omitempty"`
	ShouldRepeat bool                                                  `json:"should_repeat"`
}

type DirectorSupervisedBurstIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

type DirectorSupervisedBurstErrorV0 struct {
	Code          string                           `json:"code"`
	Message       string                           `json:"message"`
	Field         string                           `json:"field,omitempty"`
	Retryable     bool                             `json:"retryable"`
	Issues        []DirectorSupervisedBurstIssueV0 `json:"issues,omitempty"`
	CorrelationID string                           `json:"correlation_id,omitempty"`
}

func (err DirectorSupervisedBurstErrorV0) Error() string {
	parts := []string{strings.TrimSpace(err.Code)}
	if field := strings.TrimSpace(err.Field); field != "" {
		parts = append(parts, "field="+field)
	}
	if message := strings.TrimSpace(err.Message); message != "" {
		parts = append(parts, message)
	}
	if len(err.Issues) > 0 {
		issue := mostSpecificBurstIssueV0(err.Issues)
		detail := compactBurstErrorIssueV0([]string{
			strings.TrimSpace(issue.Code),
			strings.TrimSpace(issue.Field),
			strings.TrimSpace(issue.Message),
		})
		if detail != "" {
			parts = append(parts, "issue="+detail)
		}
	}
	return compactBurstErrorIssueV0(parts)
}

func compactBurstErrorIssueV0(parts []string) string {
	return strings.Join(compactBurstStringsV0(parts), ": ")
}

func mostSpecificBurstIssueV0(issues []DirectorSupervisedBurstIssueV0) DirectorSupervisedBurstIssueV0 {
	for index := len(issues) - 1; index >= 0; index-- {
		if strings.TrimSpace(issues[index].Code) != "" {
			return issues[index]
		}
	}
	return DirectorSupervisedBurstIssueV0{}
}
