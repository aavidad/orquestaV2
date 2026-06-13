package orquestaopestopicregistry

import (
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TopicRegistryUpdateRequestFromDomainWorkJobV0(
	job orquestadomainwork.DomainWorkJobRequestV0,
	toolPath string,
	agentID string,
) TopicRegistryUpdateRequestV0 {
	return NormalizeTopicRegistryUpdateRequestV0(TopicRegistryUpdateRequestV0{
		ToolPath:     toolPath,
		Action:       firstNonEmptyV0(fieldStringV0(job.InputFields, "registry_action"), TopicRegistryActionUpdateV0),
		CourseID:     fieldStringV0(job.InputFields, "course_id"),
		TopicID:      fieldStringV0(job.InputFields, "topic_id"),
		AgentID:      firstNonEmptyV0(agentID, fieldStringV0(job.InputFields, "agent_id"), DefaultTopicRegistryAgentIDV0),
		Status:       fieldStringV0(job.InputFields, "proposed_status", "status"),
		Summary:      firstNonEmptyV0(fieldStringV0(job.InputFields, "source_summary", "summary"), job.Objective),
		Done:         strings.Join(fieldStringsV0(job.InputFields, "done_refs"), ", "),
		Pending:      strings.Join(fieldStringsV0(job.InputFields, "pending_refs"), ", "),
		EvidenceRefs: compactStringsV0(append(job.EvidenceRefs, fieldStringsV0(job.InputFields, "evidence_refs")...)),
	})
}

func fieldStringV0(fields []orquestadomainwork.DomainWorkFieldV0, names ...string) string {
	for _, name := range names {
		for _, field := range fields {
			if strings.TrimSpace(field.Name) != strings.TrimSpace(name) {
				continue
			}
			if strings.TrimSpace(field.Value) != "" {
				return strings.TrimSpace(field.Value)
			}
			if len(field.Values) > 0 {
				return strings.TrimSpace(field.Values[0])
			}
		}
	}
	return ""
}

func fieldStringsV0(fields []orquestadomainwork.DomainWorkFieldV0, names ...string) []string {
	var out []string
	for _, name := range names {
		for _, field := range fields {
			if strings.TrimSpace(field.Name) != strings.TrimSpace(name) {
				continue
			}
			out = append(out, field.Value)
			out = append(out, field.Values...)
		}
	}
	return compactStringsV0(out)
}
