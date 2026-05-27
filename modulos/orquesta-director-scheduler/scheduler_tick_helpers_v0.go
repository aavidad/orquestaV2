package orquestadirectorscheduler

import (
	"sort"
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
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
		if orquestarails.TextContainsOperationalRawDetailForFieldV0(
			"director_scheduler_tick",
			"operational_text",
			value,
		) {
			return true
		}
	}
	return false
}
