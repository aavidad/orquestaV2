package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func (stack StackV0) recoverDurableAgentWorkAssessedBeforeReconciliationV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	command orquestacoreworkflow.OrchestrationCommandV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	if stack.Stores.RunStore == nil || strings.TrimSpace(command.CommandType) != orquestacoreworkflow.OrchestrationCommandAssessAgentWorkV0 {
		return command, nil
	}
	reader := stack.eventReaderForLaunchOutboxRecoveryV0()
	if reader == nil {
		return command, nil
	}
	eventID := agentWorkAssessedEventIDForCommandV0(command)
	if eventID == "" {
		return command, nil
	}
	var commandPayload orquestacoreworkflow.AssessAgentWorkCommandPayloadV0
	if err := json.Unmarshal(command.Payload, &commandPayload); err != nil {
		return command, err
	}
	events, err := reader.LoadRunEventsV0(ctx, run.RunID)
	if err != nil {
		return command, err
	}
	event, ok := durableEventByIDAndTypeV0(events, eventID, orquestacoreworkflow.OrchestrationEventAgentWorkAssessedV0)
	if !ok {
		event, ok = durableAgentWorkAssessedEventByAssessmentBaseV0(events, commandPayload.AssessmentRef)
	}
	if !ok {
		return command, nil
	}
	var payload orquestacoreworkflow.AgentWorkAssessedPayloadV0
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return command, err
	}
	if !codexStackStringInSetV0(run.AgentAssessments, orquestacoreworkflow.AgentAssessmentProjectionRefV0(payload)) {
		projected, err := orquestacoreworkflow.ApplyEventV0(run, event)
		if err != nil {
			return command, err
		}
		if err := stack.Stores.RunStore.SaveRunV0(ctx, projected); err != nil {
			return command, err
		}
	}
	recovered, err := orquestacoreworkflow.NewAssessAgentWorkCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      firstNonEmptyQueuedSourceV0(strings.TrimSpace(event.CausationID), strings.TrimSpace(command.CommandID)),
			RunID:          strings.TrimSpace(event.RunID),
			IdempotencyKey: strings.TrimSpace(event.IdempotencyKey),
			CorrelationID:  strings.TrimSpace(event.CorrelationID),
			RequestedBy:    "orquesta-app-codex-stack-live-agent-reconciliation-recovery",
			OccurredAt:     strings.TrimSpace(event.OccurredAt),
		},
		orquestacoreworkflow.AssessAgentWorkCommandPayloadV0(payload),
	)
	if err != nil {
		return command, err
	}
	return recovered, nil
}

func agentWorkAssessedEventIDForCommandV0(
	command orquestacoreworkflow.OrchestrationCommandV0,
) string {
	key := strings.TrimSpace(command.IdempotencyKey)
	if key == "" {
		return ""
	}
	return "evt-agentworkassessed-" + key
}

func durableEventByIDAndTypeV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
	eventID string,
	eventType string,
) (orquestacoreworkflow.OrchestrationEventV0, bool) {
	eventID = strings.TrimSpace(eventID)
	eventType = strings.TrimSpace(eventType)
	if eventID == "" || eventType == "" {
		return orquestacoreworkflow.OrchestrationEventV0{}, false
	}
	for _, event := range events {
		if strings.TrimSpace(event.EventID) == eventID &&
			strings.TrimSpace(event.EventType) == eventType {
			return event, true
		}
	}
	return orquestacoreworkflow.OrchestrationEventV0{}, false
}

func durableAgentWorkAssessedEventByAssessmentBaseV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
	assessmentRef string,
) (orquestacoreworkflow.OrchestrationEventV0, bool) {
	base := liveAgentReconciliationAssessmentBaseRefV0(assessmentRef)
	if base == "" {
		return orquestacoreworkflow.OrchestrationEventV0{}, false
	}
	var found orquestacoreworkflow.OrchestrationEventV0
	ok := false
	for _, event := range events {
		if strings.TrimSpace(event.EventType) != orquestacoreworkflow.OrchestrationEventAgentWorkAssessedV0 {
			continue
		}
		var payload orquestacoreworkflow.AgentWorkAssessedPayloadV0
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if liveAgentReconciliationAssessmentBaseRefV0(payload.AssessmentRef) != base {
			continue
		}
		if !ok || event.Sequence >= found.Sequence {
			found = event
			ok = true
		}
	}
	return found, ok
}

type liveAgentReconciliationAssessmentKeyPayloadV0 struct {
	ReportID             string   `json:"report_id"`
	RunID                string   `json:"run_id"`
	AgentRequestID       string   `json:"agent_request_id"`
	Status               string   `json:"status"`
	BudgetStatus         string   `json:"budget_status,omitempty"`
	BudgetReason         string   `json:"budget_reason,omitempty"`
	NoProgressTicks      int      `json:"no_progress_ticks,omitempty"`
	RepeatedActionCount  int      `json:"repeated_action_count,omitempty"`
	Summary              string   `json:"summary"`
	EvidenceRefs         []string `json:"evidence_refs,omitempty"`
	PhaseID              string   `json:"phase_id"`
	TaskRef              string   `json:"task_ref,omitempty"`
	DeliveryRef          string   `json:"delivery_ref,omitempty"`
	ObservationAssessRef string   `json:"observation_assessment_ref,omitempty"`
}

func liveAgentReconciliationAssessmentKeyV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	observation orquestacionnucleoapp.AgentProgressObservationV0,
) string {
	report := observation.Report
	payload := liveAgentReconciliationAssessmentKeyPayloadV0{
		ReportID:             strings.TrimSpace(report.ReportID),
		RunID:                strings.TrimSpace(report.RunID),
		AgentRequestID:       strings.TrimSpace(report.AgentRequestID),
		Status:               strings.TrimSpace(string(report.Status)),
		BudgetStatus:         strings.TrimSpace(string(report.BudgetStatus)),
		BudgetReason:         strings.TrimSpace(report.BudgetReason),
		NoProgressTicks:      report.NoProgressTicks,
		RepeatedActionCount:  report.RepeatedActionCount,
		Summary:              strings.TrimSpace(report.Summary),
		EvidenceRefs:         compactStringsV0(report.EvidenceRefs),
		PhaseID:              stoppedAgentObservationPhaseV0(run, observation),
		TaskRef:              strings.TrimSpace(observation.TaskRef),
		DeliveryRef:          strings.TrimSpace(observation.DeliveryRef),
		ObservationAssessRef: strings.TrimSpace(observation.AssessmentRef),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	digest := codexStackDeterministicDigestV0(string(data))
	if len(digest) > 24 {
		digest = digest[:24]
	}
	return digest
}

func liveAgentReconciliationAssessmentBaseRefV0(
	assessmentRef string,
) string {
	assessmentRef = strings.TrimSpace(assessmentRef)
	if assessmentRef == "" {
		return ""
	}
	if len(assessmentRef) > 25 && assessmentRef[len(assessmentRef)-25] == '-' &&
		liveAgentReconciliationDigestSuffixV0(assessmentRef[len(assessmentRef)-24:]) {
		return assessmentRef[:len(assessmentRef)-25]
	}
	return assessmentRef
}

func liveAgentReconciliationDigestSuffixV0(value string) bool {
	if len(value) != 24 {
		return false
	}
	for _, ch := range value {
		if (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') {
			continue
		}
		return false
	}
	return true
}
