package main

import (
	"strings"

	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func terminalStartupQueueCandidateV0(
	candidate orquestarunqueue.RunSchedulingCandidateV0,
	command orquestaserver.StartupCheckCommandV0,
	status string,
) orquestarunqueue.RunSchedulingCandidateV0 {
	status = strings.TrimSpace(status)
	if status == "" {
		status = orquestarunqueue.RunStatusStoppedV0
	}
	candidate.Status = status
	if !command.OccurredAt.IsZero() {
		candidate.UpdatedAt = command.OccurredAt
	}
	candidate.EvidenceRefs = appendStartupEvidenceRefV0(
		candidate.EvidenceRefs,
		startupQueueTerminalEvidenceRefV0(status),
	)
	return candidate
}

func stoppedStartupQueueCandidateV0(
	candidate orquestarunqueue.RunSchedulingCandidateV0,
	command orquestaserver.StartupCheckCommandV0,
) orquestarunqueue.RunSchedulingCandidateV0 {
	return terminalStartupQueueCandidateV0(candidate, command, orquestarunqueue.RunStatusStoppedV0)
}

func startupQueueTerminalEvidenceRefV0(status string) string {
	switch strings.TrimSpace(status) {
	case orquestarunqueue.RunStatusClosedV0:
		return "evidence-ref-orquesta-startup-queue-closed"
	case orquestarunqueue.RunStatusDeliveredV0:
		return "evidence-ref-orquesta-startup-queue-delivered"
	case orquestarunqueue.RunStatusCanceledV0:
		return "evidence-ref-orquesta-startup-queue-canceled"
	default:
		return "evidence-ref-orquesta-startup-queue-stopped"
	}
}

func appendStartupEvidenceRefV0(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return append([]string(nil), values...)
	}
	out := make([]string, 0, len(values)+1)
	seen := map[string]struct{}{}
	for _, existing := range values {
		existing = strings.TrimSpace(existing)
		if existing == "" {
			continue
		}
		if _, ok := seen[existing]; ok {
			continue
		}
		seen[existing] = struct{}{}
		out = append(out, existing)
	}
	if _, ok := seen[value]; !ok {
		out = append(out, value)
	}
	return out
}

func firstNonEmptyStartupStringV0(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func startupSafeRefPartV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, " ", "-")
	if value == "" {
		return "unknown"
	}
	return value
}
