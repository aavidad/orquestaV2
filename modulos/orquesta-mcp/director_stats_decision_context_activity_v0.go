package orquestamcp

import (
	"fmt"
	"strings"

	orquestaobservability "orquesta/modulos/orquesta-observability"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

func mcpDirectorDecisionBlockersV0(
	stats orquestacionnucleoapp.DirectorRunStatsV0,
) []orquestaobservability.DirectorDecisionBlockerV0 {
	out := make([]orquestaobservability.DirectorDecisionBlockerV0, 0, len(stats.Closure.BlockerRefs)+len(stats.Refs.Blockers))
	seen := map[string]bool{}
	for index, ref := range compactStringsMCPV0(stats.Closure.BlockerRefs) {
		cause := mcpDirectorCauseForIndexV0(stats.Closure.BlockedBy, index)
		out = appendDirectorBlockerMCPV0(out, seen, ref, cause, "closure")
	}
	for _, ref := range compactStringsMCPV0(stats.Refs.Blockers) {
		out = appendDirectorBlockerMCPV0(out, seen, ref, "", "run")
	}
	if len(out) == 0 && stats.Closure.Blocked {
		for index, cause := range compactStringsMCPV0(stats.Closure.BlockedBy) {
			ref := fmt.Sprintf("blocker-ref-closure-%03d", index+1)
			out = appendDirectorBlockerMCPV0(out, seen, ref, cause, "closure")
		}
	}
	return out
}

func appendDirectorBlockerMCPV0(
	out []orquestaobservability.DirectorDecisionBlockerV0,
	seen map[string]bool,
	ref string,
	cause string,
	source string,
) []orquestaobservability.DirectorDecisionBlockerV0 {
	ref = strings.TrimSpace(ref)
	if ref == "" || seen[ref] {
		return out
	}
	seen[ref] = true
	return append(out, orquestaobservability.DirectorDecisionBlockerV0{
		BlockerRef: ref,
		Cause:      strings.TrimSpace(cause),
		Source:     strings.TrimSpace(source),
		SummaryKey: "director.decision_context.blocker." + strings.TrimSpace(source),
	})
}

func mcpDirectorDecisionActivityV0(
	context orquestaobservability.DirectorDecisionContextV0,
	observedAt string,
) []orquestaobservability.DirectorDecisionActivityV0 {
	out := make([]orquestaobservability.DirectorDecisionActivityV0, 0, 12)
	out = appendMCPDirectorActivityV0(out, observedAt, orquestaobservability.DirectorDecisionActivityPhaseCurrentV0, context.RunRef, "", "")
	for _, task := range context.Tasks {
		out = appendMCPDirectorActivityV0(out, observedAt, orquestaobservability.DirectorDecisionActivityTaskProgressV0, task.TaskRef, task.AgentRequestID, task.TaskRef)
	}
	for _, agent := range context.Agents {
		out = appendMCPDirectorActivityV0(out, observedAt, orquestaobservability.DirectorDecisionActivityAgentLifecycleV0, agent.AgentRequestID, agent.AgentRequestID, "")
	}
	for _, ref := range context.ReworkReplan.ReworkRequestRefs {
		out = appendMCPDirectorActivityV0(out, observedAt, orquestaobservability.DirectorDecisionActivityReworkRequestV0, ref, "", "")
	}
	for _, ref := range context.ReworkReplan.ReplanDecisionRefs {
		out = appendMCPDirectorActivityV0(out, observedAt, orquestaobservability.DirectorDecisionActivityReplanDecisionV0, ref, "", "")
	}
	for _, blocker := range context.Blockers {
		out = appendMCPDirectorActivityV0(out, observedAt, orquestaobservability.DirectorDecisionActivityClosureBlockedV0, blocker.BlockerRef, "", "")
	}
	if len(out) > 20 {
		return out[:20]
	}
	return out
}

func appendMCPDirectorActivityV0(
	out []orquestaobservability.DirectorDecisionActivityV0,
	observedAt string,
	kind string,
	sourceRef string,
	agentRef string,
	taskRef string,
) []orquestaobservability.DirectorDecisionActivityV0 {
	if strings.TrimSpace(sourceRef) == "" {
		return out
	}
	index := len(out) + 1
	return append(out, orquestaobservability.DirectorDecisionActivityV0{
		ActivityRef:    mcpDirectorActivityRefV0(index, sourceRef),
		Kind:           kind,
		OccurredAt:     observedAt,
		SourceRef:      strings.TrimSpace(sourceRef),
		AgentRequestID: strings.TrimSpace(agentRef),
		TaskRef:        strings.TrimSpace(taskRef),
		SummaryKey:     "director.decision_context.activity." + kind,
	})
}

func mcpDirectorActivityRefV0(index int, sourceRef string) string {
	slug := neutralAppDirectorSlugV0(sourceRef)
	if len(slug) > 40 {
		slug = slug[:40]
	}
	return fmt.Sprintf("activity-ref-director-%03d-%s", index, slug)
}
