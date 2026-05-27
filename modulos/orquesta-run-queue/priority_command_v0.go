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
	command.RequestedBy = strings.TrimSpace(command.RequestedBy)
	command.Reason = strings.TrimSpace(command.Reason)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.EvidenceRefs = compactRunQueueStringsV0(command.EvidenceRefs)
	command.WorksetClaims = cloneRunQueueWorksetClaimsV0(command.WorksetClaims)
	return command
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
