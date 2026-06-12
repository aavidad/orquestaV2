package orquestarunqueue

import "strings"

func NormalizeRunQueuePriorityCommandV0(
	command RunQueuePriorityCommandV0,
) RunQueuePriorityCommandV0 {
	command.RunRef = strings.TrimSpace(command.RunRef)
	command.QueueRef = strings.TrimSpace(command.QueueRef)
	command.AppRef = strings.TrimSpace(command.AppRef)
	command.Status = strings.TrimSpace(command.Status)
	command.FairnessGroupRef = strings.TrimSpace(command.FairnessGroupRef)
	command.AttemptGroup = NormalizeRunQueueAttemptGroupV0(command.AttemptGroup)
	command.ParentRunRef = strings.TrimSpace(command.ParentRunRef)
	command.SupersedesRunRef = strings.TrimSpace(command.SupersedesRunRef)
	command.RescueReason = strings.TrimSpace(command.RescueReason)
	command.RequestedBy = strings.TrimSpace(command.RequestedBy)
	command.Reason = strings.TrimSpace(command.Reason)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.EvidenceRefs = compactRunQueueStringsV0(command.EvidenceRefs)
	command.WorksetClaims = cloneRunQueueWorksetClaimsV0(command.WorksetClaims)
	return command
}

func NormalizeRunQueueAttemptGroupV0(group RunQueueAttemptGroupV0) RunQueueAttemptGroupV0 {
	group.GroupRef = strings.TrimSpace(group.GroupRef)
	group.ConsumerRef = strings.TrimSpace(group.ConsumerRef)
	group.ObjectiveRef = strings.TrimSpace(group.ObjectiveRef)
	group.WorkItemRef = strings.TrimSpace(group.WorkItemRef)
	group.WriteSetRefs = compactRunQueueStringsV0(group.WriteSetRefs)
	return group
}

func RunQueueAttemptGroupEmptyV0(group RunQueueAttemptGroupV0) bool {
	group = NormalizeRunQueueAttemptGroupV0(group)
	return group.GroupRef == "" &&
		group.ConsumerRef == "" &&
		group.ObjectiveRef == "" &&
		group.WorkItemRef == "" &&
		len(group.WriteSetRefs) == 0
}

func ValidateRunQueuePriorityCommandV0(
	command RunQueuePriorityCommandV0,
) []RunQueueIssueV0 {
	command = NormalizeRunQueuePriorityCommandV0(command)
	var issues []RunQueueIssueV0
	if command.RunRef == "" {
		issues = append(issues, RunQueueIssueV0{Code: "run_ref_requerido", Field: "run_ref"})
	}
	return issues
}

func compactRunQueueStringsV0(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if out == nil {
		return []string{}
	}
	return out
}
