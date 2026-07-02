package orquestaappcodexstack

import (
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

func enrichGoalFirstDomainWorkSubmissionLifecycleV0(
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0,
	result orquestagoal.GoalWorkResultV0,
) orquestadomainwork.DomainWorkArtifactSubmissionV0 {
	fields := submission.PayloadFields
	fields = appendDomainWorkFieldIfMissingV0(fields, "director_execution_mode", "goal_first")
	fields = appendDomainWorkFieldIfMissingV0(fields, "goal_first_status", result.Status)
	fields = appendDomainWorkFieldIfMissingV0(fields, "goal_ref", result.GoalRef)
	fields = appendDomainWorkFieldIfMissingV0(fields, "external_goal_ref", result.ExternalGoalRef)
	fields = appendDomainWorkValuesFieldIfMissingV0(fields, "orquesta_goal_result_refs", goalFirstDomainWorkResultRefsV0(result))
	fields = appendDomainWorkValuesFieldIfMissingV0(fields, "goal_first_checkpoint_refs", goalFirstDomainWorkCheckpointRefsV0(result))
	fields = appendDomainWorkValuesFieldIfMissingV0(fields, "goal_first_heartbeat_refs", goalFirstDomainWorkHeartbeatRefsV0(result))
	submission.PayloadFields = fields
	return orquestadomainwork.NormalizeDomainWorkArtifactSubmissionV0(submission)
}

func appendDomainWorkValuesFieldIfMissingV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
	values []string,
) []orquestadomainwork.DomainWorkFieldV0 {
	name = strings.TrimSpace(name)
	values = compactStringsV0(values)
	if name == "" || len(values) == 0 || domainWorkFieldHasNameV0(fields, name) {
		return fields
	}
	return append(fields, orquestadomainwork.DomainWorkFieldV0{Name: name, Values: values})
}

func goalFirstDomainWorkResultRefsV0(result orquestagoal.GoalWorkResultV0) []string {
	refs := []string{result.GoalRef, result.ExternalGoalRef}
	refs = append(refs, result.ArtifactRefs...)
	refs = append(refs, result.DomainReceiptRefs...)
	refs = append(refs, result.EvidenceRefs...)
	for _, artifact := range result.MaterializedArtifacts {
		refs = append(refs, artifact.ArtifactRef)
		refs = append(refs, artifact.EvidenceRefs...)
	}
	refs = append(refs, result.Checklist.EvidenceRefs...)
	for _, test := range result.RequiredTestResults {
		refs = append(refs, test.EvidenceRefs...)
	}
	return compactStringsV0(refs)
}

func goalFirstDomainWorkCheckpointRefsV0(result orquestagoal.GoalWorkResultV0) []string {
	return goalFirstDomainWorkRefsContainingAnyV0(
		goalFirstDomainWorkResultRefsV0(result),
		"checkpoint",
		"goal-result",
		"goal_result",
		"orquesta_goal_result",
	)
}

func goalFirstDomainWorkHeartbeatRefsV0(result orquestagoal.GoalWorkResultV0) []string {
	return goalFirstDomainWorkRefsContainingAnyV0(
		goalFirstDomainWorkResultRefsV0(result),
		"heartbeat",
		"progress",
	)
}

func goalFirstDomainWorkRefsContainingAnyV0(values []string, needles ...string) []string {
	var out []string
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" {
			continue
		}
		for _, needle := range needles {
			if needle = strings.ToLower(strings.TrimSpace(needle)); needle != "" && strings.Contains(normalized, needle) {
				out = append(out, value)
				break
			}
		}
	}
	return compactStringsV0(out)
}
