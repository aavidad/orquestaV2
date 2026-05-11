package orquestaoutboxdispatch

import "strings"

const (
	ackClosureIssueUnknownItemV0 = "ack_unknown_item"
	ackClosureIssueFailedV0      = "ack_failed"
)

type OutboxDispatchAckObservationStatusV0 string

const (
	OutboxDispatchAckObservationSuccessV0 OutboxDispatchAckObservationStatusV0 = "success"
	OutboxDispatchAckObservationFailedV0  OutboxDispatchAckObservationStatusV0 = "failed"
)

type OutboxDispatchAckClosureStatusV0 string

const (
	OutboxDispatchAckClosurePendingV0 OutboxDispatchAckClosureStatusV0 = "pending"
	OutboxDispatchAckClosureAckedV0   OutboxDispatchAckClosureStatusV0 = "acked"
	OutboxDispatchAckClosureFailedV0  OutboxDispatchAckClosureStatusV0 = "ack_failed"
)

type OutboxDispatchAckObservationV0 struct {
	MessageID    string
	RunID        string
	TargetPort   string
	Status       OutboxDispatchAckObservationStatusV0
	DispatchRef  string
	EvidenceRefs []string
	Issues       []DispatchIssueV0
}

type OutboxDispatchAckClosureInputV0 struct {
	Intents []DispatchIntentV0
	Acks    []OutboxDispatchAckObservationV0
}

type OutboxDispatchAckClosureItemV0 struct {
	MessageID    string
	RunID        string
	TargetPort   string
	Status       OutboxDispatchAckClosureStatusV0
	DispatchRef  string
	EvidenceRefs []string
	Issues       []DispatchIssueV0
}

type OutboxDispatchAckClosureResultV0 struct {
	Items   []OutboxDispatchAckClosureItemV0
	Pending []DispatchIntentV0
	Acked   []OutboxDispatchAckClosureItemV0
	Failed  []OutboxDispatchAckClosureItemV0
	Issues  []DispatchIssueV0
}

func PlanOutboxDispatchAckClosureV0(
	input OutboxDispatchAckClosureInputV0,
) OutboxDispatchAckClosureResultV0 {
	intents := normalizeClosureIntentsV0(input.Intents)
	states := make(map[ackClosureKeyV0]OutboxDispatchAckClosureItemV0, len(intents))
	order := make([]ackClosureKeyV0, 0, len(intents))
	for _, intent := range intents {
		key := ackClosureKeyFromIntentV0(intent)
		if _, exists := states[key]; exists {
			continue
		}
		order = append(order, key)
		states[key] = OutboxDispatchAckClosureItemV0{
			MessageID:  intent.MessageID,
			RunID:      intent.RunID,
			TargetPort: intent.TargetPort,
			Status:     OutboxDispatchAckClosurePendingV0,
		}
	}

	result := OutboxDispatchAckClosureResultV0{}
	for _, ack := range normalizeAckObservationsV0(input.Acks) {
		key := ackClosureKeyFromAckV0(ack)
		current, exists := states[key]
		if !exists {
			result.Issues = append(result.Issues, issueV0(
				ackClosureIssueUnknownItemV0,
				"message_id",
				ack.MessageID+" sin intent correlacionado",
			))
			continue
		}
		switch ack.Status {
		case OutboxDispatchAckObservationSuccessV0:
			if current.Status != OutboxDispatchAckClosureAckedV0 {
				current.Status = OutboxDispatchAckClosureAckedV0
				current.DispatchRef = ack.DispatchRef
				current.EvidenceRefs = compactStringsV0(ack.EvidenceRefs)
				current.Issues = nil
			}
		case OutboxDispatchAckObservationFailedV0:
			if current.Status != OutboxDispatchAckClosureAckedV0 {
				current.Status = OutboxDispatchAckClosureFailedV0
				current.EvidenceRefs = compactStringsV0(ack.EvidenceRefs)
				current.Issues = failedAckClosureIssuesV0(ack)
			}
		default:
			result.Issues = append(result.Issues, issueV0(
				issueInvalidRequestV0,
				"ack_status",
				"ack_status invalido",
			))
		}
		states[key] = current
	}

	for _, key := range order {
		item := states[key]
		result.Items = append(result.Items, item)
		switch item.Status {
		case OutboxDispatchAckClosureAckedV0:
			result.Acked = append(result.Acked, item)
		case OutboxDispatchAckClosureFailedV0:
			result.Failed = append(result.Failed, item)
		default:
			result.Pending = append(result.Pending, intentsByClosureKeyV0(intents, key))
		}
	}
	return result
}

type ackClosureKeyV0 struct {
	messageID  string
	runID      string
	targetPort string
}

func ackClosureKeyFromIntentV0(intent DispatchIntentV0) ackClosureKeyV0 {
	return ackClosureKeyV0{
		messageID:  intent.MessageID,
		runID:      intent.RunID,
		targetPort: intent.TargetPort,
	}
}

func ackClosureKeyFromAckV0(ack OutboxDispatchAckObservationV0) ackClosureKeyV0 {
	return ackClosureKeyV0{
		messageID:  ack.MessageID,
		runID:      ack.RunID,
		targetPort: ack.TargetPort,
	}
}

func normalizeClosureIntentsV0(intents []DispatchIntentV0) []DispatchIntentV0 {
	if len(intents) == 0 {
		return nil
	}
	normalized := make([]DispatchIntentV0, 0, len(intents))
	for _, intent := range intents {
		normalized = append(normalized, DispatchIntentV0{
			MessageID:      strings.TrimSpace(intent.MessageID),
			RunID:          strings.TrimSpace(intent.RunID),
			TargetPort:     strings.TrimSpace(intent.TargetPort),
			MessageType:    strings.TrimSpace(intent.MessageType),
			IdempotencyKey: strings.TrimSpace(intent.IdempotencyKey),
			CorrelationID:  strings.TrimSpace(intent.CorrelationID),
			PayloadVersion: strings.TrimSpace(intent.PayloadVersion),
			Payload:        cloneBytesV0(intent.Payload),
		})
	}
	return normalized
}

func normalizeAckObservationsV0(
	acks []OutboxDispatchAckObservationV0,
) []OutboxDispatchAckObservationV0 {
	if len(acks) == 0 {
		return nil
	}
	normalized := make([]OutboxDispatchAckObservationV0, 0, len(acks))
	for _, ack := range acks {
		normalized = append(normalized, OutboxDispatchAckObservationV0{
			MessageID:    strings.TrimSpace(ack.MessageID),
			RunID:        strings.TrimSpace(ack.RunID),
			TargetPort:   strings.TrimSpace(ack.TargetPort),
			Status:       ack.Status,
			DispatchRef:  strings.TrimSpace(ack.DispatchRef),
			EvidenceRefs: compactStringsV0(ack.EvidenceRefs),
			Issues:       normalizeDispatchIssuesV0(ack.Issues),
		})
	}
	return normalized
}

func failedAckClosureIssuesV0(
	ack OutboxDispatchAckObservationV0,
) []DispatchIssueV0 {
	if len(ack.Issues) > 0 {
		return normalizeDispatchIssuesV0(ack.Issues)
	}
	return []DispatchIssueV0{
		issueV0(ackClosureIssueFailedV0, "message_id", ack.MessageID+" ACK failed"),
	}
}

func normalizeDispatchIssuesV0(issues []DispatchIssueV0) []DispatchIssueV0 {
	if len(issues) == 0 {
		return nil
	}
	normalized := make([]DispatchIssueV0, 0, len(issues))
	for _, issue := range issues {
		issue.Code = strings.TrimSpace(issue.Code)
		issue.Field = strings.TrimSpace(issue.Field)
		issue.Message = strings.TrimSpace(issue.Message)
		if issue.Code == "" {
			continue
		}
		normalized = append(normalized, issue)
	}
	return normalized
}

func intentsByClosureKeyV0(intents []DispatchIntentV0, key ackClosureKeyV0) DispatchIntentV0 {
	for _, intent := range intents {
		if ackClosureKeyFromIntentV0(intent) == key {
			return intent
		}
	}
	return DispatchIntentV0{}
}
