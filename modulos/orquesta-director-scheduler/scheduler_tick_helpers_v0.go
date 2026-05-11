package orquestadirectorscheduler

import (
	"sort"
	"strings"
)

func compactSchedulerStringsV0(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var compact []string
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		compact = append(compact, value)
	}
	sort.Strings(compact)
	return compact
}

func schedulerStringSetV0(values []string) map[string]bool {
	set := map[string]bool{}
	for _, value := range values {
		set[value] = true
	}
	return set
}

func compactSchedulerWaitingReasonsV0(reasons []SchedulerWaitingReasonV0) []SchedulerWaitingReasonV0 {
	if len(reasons) == 0 {
		return nil
	}
	seen := map[SchedulerWaitingReasonV0]bool{}
	for _, reason := range reasons {
		if reason != "" {
			seen[reason] = true
		}
	}
	ordered := []SchedulerWaitingReasonV0{
		SchedulerWaitingOutboxPendingV0,
		SchedulerWaitingCapacityPendingV0,
		SchedulerWaitingAgentLifecyclePendingV0,
		SchedulerWaitingAgentDeliveryPendingV0,
		SchedulerWaitingCandidateMissingV0,
		SchedulerWaitingDirectorQuestionPendingV0,
		SchedulerWaitingQualityGateFollowupV0,
	}
	compact := make([]SchedulerWaitingReasonV0, 0, len(seen))
	for _, reason := range ordered {
		if seen[reason] {
			compact = append(compact, reason)
			delete(seen, reason)
		}
	}
	var unknown []SchedulerWaitingReasonV0
	for reason := range seen {
		unknown = append(unknown, reason)
	}
	sort.Slice(unknown, func(left int, right int) bool {
		return unknown[left] < unknown[right]
	})
	return append(compact, unknown...)
}

func schedulerContainsForbiddenDetailsV0(values []string) bool {
	for _, value := range values {
		lower := strings.ToLower(value)
		for _, forbidden := range forbiddenSchedulerFragmentsV0 {
			if containsSchedulerTokenV0(lower, forbidden) {
				return true
			}
		}
	}
	return false
}

func containsSchedulerTokenV0(value string, fragment string) bool {
	fragment = strings.ToLower(strings.TrimSpace(fragment))
	if fragment == "" {
		return false
	}
	start := 0
	for {
		index := strings.Index(value[start:], fragment)
		if index < 0 {
			return false
		}
		absolute := start + index
		if schedulerTokenBoundaryV0(value, absolute, absolute+len(fragment)) {
			return true
		}
		start = absolute + len(fragment)
	}
}

func schedulerTokenBoundaryV0(value string, start int, end int) bool {
	before := start == 0 || !schedulerAsciiLetterOrDigitV0(value[start-1])
	after := end >= len(value) || !schedulerAsciiLetterOrDigitV0(value[end])
	return before && after
}

func schedulerAsciiLetterOrDigitV0(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
}
