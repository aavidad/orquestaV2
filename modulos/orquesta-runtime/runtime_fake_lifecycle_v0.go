package orquestaruntime

import "fmt"

type RuntimeFakeLifecycleStatusV0 string

const (
	RuntimeFakeLifecycleLaunchedV0     RuntimeFakeLifecycleStatusV0 = "launched"
	RuntimeFakeLifecycleProgressingV0  RuntimeFakeLifecycleStatusV0 = "progressing"
	RuntimeFakeLifecycleLoopDetectedV0 RuntimeFakeLifecycleStatusV0 = "loop_detected"
	RuntimeFakeLifecycleStoppedV0      RuntimeFakeLifecycleStatusV0 = "stopped"
)

type RuntimeFakeLifecycleErrorCodeV0 string

const (
	RuntimeFakeLifecycleInboundInvalidoV0 RuntimeFakeLifecycleErrorCodeV0 = "inbound_invalido"
	RuntimeFakeLifecycleAgentMissingV0    RuntimeFakeLifecycleErrorCodeV0 = "agent_missing"
	RuntimeFakeLifecycleRunMismatchV0     RuntimeFakeLifecycleErrorCodeV0 = "run_mismatch"
	RuntimeFakeLifecycleProgressInvalidV0 RuntimeFakeLifecycleErrorCodeV0 = "progress_invalid"
)

type RuntimeFakeLifecycleErrorV0 struct {
	Code       RuntimeFakeLifecycleErrorCodeV0 `json:"code"`
	MessageKey string                          `json:"message_key"`
	Field      string                          `json:"field,omitempty"`
	Retryable  bool                            `json:"retryable"`
	Evidence   []string                        `json:"evidence,omitempty"`
}

func (e RuntimeFakeLifecycleErrorV0) Error() string {
	if e.Field == "" {
		return string(e.Code)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Field)
}

type RuntimeFakeLifecycleSnapshotV0 struct {
	AgentRequestID  string                       `json:"agent_request_id"`
	RunID           string                       `json:"run_id"`
	Status          RuntimeFakeLifecycleStatusV0 `json:"status"`
	LaunchRef       string                       `json:"launch_ref,omitempty"`
	LastReportRef   string                       `json:"last_report_ref,omitempty"`
	StopRef         string                       `json:"stop_ref,omitempty"`
	ProgressReports int                          `json:"progress_reports"`
}

type RuntimeFakeLifecycleV0 struct {
	agents map[string]runtimeFakeLifecycleAgentV0
}

type runtimeFakeLifecycleAgentV0 struct {
	agentRequestID  string
	runID           string
	status          RuntimeFakeLifecycleStatusV0
	launchRef       string
	lastReportRef   string
	stopRef         string
	progressReports int
}

func NewRuntimeFakeLifecycleV0() *RuntimeFakeLifecycleV0 {
	return &RuntimeFakeLifecycleV0{
		agents: map[string]runtimeFakeLifecycleAgentV0{},
	}
}

func (r *RuntimeFakeLifecycleV0) LaunchAgentV0(
	inbound AgentLauncherInboundV0,
) (RuntimeFakeLifecycleSnapshotV0, error) {
	if issues := ValidateAgentLauncherInboundV0(inbound); len(issues) != 0 {
		return RuntimeFakeLifecycleSnapshotV0{}, runtimeFakeLifecycleErrorV0(
			RuntimeFakeLifecycleInboundInvalidoV0,
			"launch",
			agentLauncherErrorEvidenceV0(issues),
		)
	}

	r.ensureAgents()
	if agent, ok := r.agents[inbound.Payload.AgentRequestID]; ok {
		if agent.runID != inbound.Payload.RunID {
			return RuntimeFakeLifecycleSnapshotV0{}, runtimeFakeLifecycleErrorV0(
				RuntimeFakeLifecycleRunMismatchV0,
				"run_id",
				nil,
			)
		}
		return agent.snapshot(), nil
	}

	agent := runtimeFakeLifecycleAgentV0{
		agentRequestID: inbound.Payload.AgentRequestID,
		runID:          inbound.Payload.RunID,
		status:         RuntimeFakeLifecycleLaunchedV0,
		launchRef:      inbound.CorrelationID,
	}
	r.agents[agent.agentRequestID] = agent

	return agent.snapshot(), nil
}

func (r *RuntimeFakeLifecycleV0) ReportProgressV0(
	report AgentProgressReportV0,
) (RuntimeFakeLifecycleSnapshotV0, error) {
	if issues := ValidateAgentProgressReportV0(report); len(issues) != 0 {
		return RuntimeFakeLifecycleSnapshotV0{}, runtimeFakeLifecycleErrorV0(
			RuntimeFakeLifecycleProgressInvalidV0,
			"progress",
			agentProgressErrorEvidenceV0(issues),
		)
	}

	r.ensureAgents()
	agent, ok := r.agents[report.AgentRequestID]
	if !ok {
		return RuntimeFakeLifecycleSnapshotV0{}, runtimeFakeLifecycleErrorV0(
			RuntimeFakeLifecycleAgentMissingV0,
			"agent_request_id",
			nil,
		)
	}
	if agent.runID != report.RunID {
		return RuntimeFakeLifecycleSnapshotV0{}, runtimeFakeLifecycleErrorV0(
			RuntimeFakeLifecycleRunMismatchV0,
			"run_id",
			nil,
		)
	}

	agent.status = runtimeFakeLifecycleStatusFromReportV0(agent.status, report.Status)
	agent.lastReportRef = report.ReportID
	agent.progressReports++
	r.agents[agent.agentRequestID] = agent

	return agent.snapshot(), nil
}

func (r *RuntimeFakeLifecycleV0) StopAgentV0(
	inbound AgentStopperInboundV0,
) (RuntimeFakeLifecycleSnapshotV0, error) {
	if issues := ValidateAgentStopperInboundV0(inbound); len(issues) != 0 {
		return RuntimeFakeLifecycleSnapshotV0{}, runtimeFakeLifecycleErrorV0(
			RuntimeFakeLifecycleInboundInvalidoV0,
			"stop",
			agentStopperErrorEvidenceV0(issues),
		)
	}

	r.ensureAgents()
	agent, ok := r.agents[inbound.Payload.AgentRequestID]
	if !ok {
		return RuntimeFakeLifecycleSnapshotV0{}, runtimeFakeLifecycleErrorV0(
			RuntimeFakeLifecycleAgentMissingV0,
			"agent_request_id",
			nil,
		)
	}
	if agent.runID != inbound.Payload.RunID {
		return RuntimeFakeLifecycleSnapshotV0{}, runtimeFakeLifecycleErrorV0(
			RuntimeFakeLifecycleRunMismatchV0,
			"run_id",
			nil,
		)
	}

	if agent.stopRef == "" {
		agent.stopRef = inbound.CorrelationID
	}
	agent.status = RuntimeFakeLifecycleStoppedV0
	r.agents[agent.agentRequestID] = agent

	return agent.snapshot(), nil
}

func (r *RuntimeFakeLifecycleV0) ensureAgents() {
	if r.agents == nil {
		r.agents = map[string]runtimeFakeLifecycleAgentV0{}
	}
}

func (a runtimeFakeLifecycleAgentV0) snapshot() RuntimeFakeLifecycleSnapshotV0 {
	return RuntimeFakeLifecycleSnapshotV0{
		AgentRequestID:  a.agentRequestID,
		RunID:           a.runID,
		Status:          a.status,
		LaunchRef:       a.launchRef,
		LastReportRef:   a.lastReportRef,
		StopRef:         a.stopRef,
		ProgressReports: a.progressReports,
	}
}

func runtimeFakeLifecycleStatusFromReportV0(
	current RuntimeFakeLifecycleStatusV0,
	status AgentProgressStatusV0,
) RuntimeFakeLifecycleStatusV0 {
	if current == RuntimeFakeLifecycleStoppedV0 {
		return RuntimeFakeLifecycleStoppedV0
	}
	switch status {
	case AgentLoopDetectedV0:
		return RuntimeFakeLifecycleLoopDetectedV0
	case AgentStoppedV0:
		return RuntimeFakeLifecycleStoppedV0
	default:
		return RuntimeFakeLifecycleProgressingV0
	}
}

func runtimeFakeLifecycleErrorV0(
	code RuntimeFakeLifecycleErrorCodeV0,
	field string,
	evidence []string,
) RuntimeFakeLifecycleErrorV0 {
	return RuntimeFakeLifecycleErrorV0{
		Code:       code,
		MessageKey: "orquesta.runtime.fake_lifecycle." + string(code),
		Field:      field,
		Retryable:  false,
		Evidence:   evidence,
	}
}

func agentLauncherErrorEvidenceV0(issues []AgentLauncherInboundErrorV0) []string {
	evidence := make([]string, 0, len(issues))
	for _, issue := range issues {
		evidence = append(evidence, string(issue.Code))
	}
	return evidence
}

func agentStopperErrorEvidenceV0(issues []AgentStopperInboundErrorV0) []string {
	evidence := make([]string, 0, len(issues))
	for _, issue := range issues {
		evidence = append(evidence, string(issue.Code))
	}
	return evidence
}

func agentProgressErrorEvidenceV0(issues []AgentProgressReportErrorV0) []string {
	evidence := make([]string, 0, len(issues))
	for _, issue := range issues {
		evidence = append(evidence, string(issue.Code))
	}
	return evidence
}
