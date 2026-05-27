package orquestaserver

import (
	"strings"
	"time"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func supervisorResultIsIdleForSelfImprovementV0(result orquestarunsupervisor.RunSupervisorResultV0) bool {
	if strings.TrimSpace(result.StopReason) != orquestarunsupervisor.RunSupervisorStopNoExecutionV0 {
		return false
	}
	for _, tick := range result.Ticks {
		if len(tick.Result.Ranked) > 0 || len(tick.Result.Executions) > 0 || len(tick.Result.Skips) > 0 {
			return false
		}
	}
	return true
}

func idleSelfImprovementKnownRunRefsV0(result orquestarunsupervisor.RunSupervisorResultV0) []string {
	var refs []string
	for _, tick := range result.Ticks {
		for _, ranked := range tick.Result.Ranked {
			refs = append(refs, ranked.RunRef)
		}
		for _, execution := range tick.Result.Executions {
			refs = append(refs, execution.RunRef)
		}
		for _, skip := range tick.Result.Skips {
			refs = append(refs, skip.RunRef)
		}
	}
	refs = append(refs, result.ErrorRunRefs...)
	return compactConfigStringsV0(refs)
}

func idleSelfImprovementKnownRequestRefsV0(runRefs []string) []string {
	out := make([]string, 0, len(runRefs))
	for _, runRef := range runRefs {
		out = append(out, idleSelfImprovementRequestRefFromRunRefV0(runRef))
	}
	return compactConfigStringsV0(out)
}

func idleSelfImprovementRequestRefFromRunRefV0(runRef string) string {
	ref := strings.TrimSpace(runRef)
	if ref == "" {
		return ""
	}
	if before, _, ok := strings.Cut(ref, "-retry-"); ok {
		ref = before
	}
	return ref
}

func idleSelfImprovementAttemptBlocksV0(
	idleSince time.Time,
	lastAttempt time.Time,
	now time.Time,
	cooldown time.Duration,
	accepted bool,
) bool {
	if lastAttempt.IsZero() {
		return false
	}
	if accepted && !idleSince.IsZero() && !lastAttempt.Before(idleSince) {
		return true
	}
	if cooldown <= 0 {
		return false
	}
	return now.Sub(lastAttempt) < cooldown
}
