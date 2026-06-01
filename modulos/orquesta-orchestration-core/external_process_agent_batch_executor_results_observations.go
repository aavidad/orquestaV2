package orquestacionnucleoapp

import (
	"strings"

	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

func handledFailedAgentBatchObservationV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
	evidenceRefs []string,
) orquestaoutboxdispatch.OutboxDispatchAckObservationV0 {
	return orquestaoutboxdispatch.OutboxDispatchAckObservationV0{
		MessageID:   intent.MessageID,
		RunID:       intent.RunID,
		TargetPort:  intent.TargetPort,
		Status:      orquestaoutboxdispatch.OutboxDispatchAckObservationSuccessV0,
		DispatchRef: "dispatch-ref-agent-failed-" + strings.TrimSpace(intent.MessageID),
		EvidenceRefs: compactStringsV0(append(
			append([]string(nil), evidenceRefs...),
			"evidence-ref-external-process-batch-failed-agent-recorded",
		)),
	}
}

func successBatchObservationV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
	launch AgentLaunchResultV0,
) orquestaoutboxdispatch.OutboxDispatchAckObservationV0 {
	return orquestaoutboxdispatch.OutboxDispatchAckObservationV0{
		MessageID:    intent.MessageID,
		RunID:        intent.RunID,
		TargetPort:   intent.TargetPort,
		Status:       orquestaoutboxdispatch.OutboxDispatchAckObservationSuccessV0,
		DispatchRef:  launch.LaunchRef,
		EvidenceRefs: launch.EvidenceRefs,
	}
}

func failedBatchObservationV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
	evidenceRef string,
) orquestaoutboxdispatch.OutboxDispatchAckObservationV0 {
	return failedBatchObservationWithIssuesV0(intent, evidenceRef, nil)
}

func failedBatchObservationFromErrorV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
	evidenceRef string,
	code string,
	field string,
	err error,
) orquestaoutboxdispatch.OutboxDispatchAckObservationV0 {
	message := ""
	if err != nil {
		message = strings.TrimSpace(err.Error())
	}
	return failedBatchObservationWithIssuesV0(
		intent,
		evidenceRef,
		[]orquestaoutboxdispatch.DispatchIssueV0{{
			Code:    strings.TrimSpace(code),
			Field:   strings.TrimSpace(field),
			Message: message,
		}},
	)
}

func failedBatchObservationWithIssuesV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
	evidenceRef string,
	issues []orquestaoutboxdispatch.DispatchIssueV0,
) orquestaoutboxdispatch.OutboxDispatchAckObservationV0 {
	return orquestaoutboxdispatch.OutboxDispatchAckObservationV0{
		MessageID:    intent.MessageID,
		RunID:        intent.RunID,
		TargetPort:   intent.TargetPort,
		Status:       orquestaoutboxdispatch.OutboxDispatchAckObservationFailedV0,
		EvidenceRefs: compactStringsV0([]string{evidenceRef}),
		Issues:       issues,
	}
}
