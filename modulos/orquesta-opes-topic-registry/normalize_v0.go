package orquestaopestopicregistry

import "strings"

func NormalizeTopicRegistryUpdateRequestV0(
	request TopicRegistryUpdateRequestV0,
) TopicRegistryUpdateRequestV0 {
	request.ToolPath = strings.TrimSpace(request.ToolPath)
	request.Action = strings.ToLower(strings.TrimSpace(request.Action))
	if request.Action == "" {
		request.Action = TopicRegistryActionUpdateV0
	}
	request.CourseID = strings.TrimSpace(request.CourseID)
	request.TopicID = strings.TrimSpace(request.TopicID)
	request.AgentID = strings.TrimSpace(request.AgentID)
	if request.AgentID == "" {
		request.AgentID = DefaultTopicRegistryAgentIDV0
	}
	request.Status = strings.TrimSpace(request.Status)
	request.Summary = strings.TrimSpace(request.Summary)
	request.Done = strings.TrimSpace(request.Done)
	request.Pending = strings.TrimSpace(request.Pending)
	request.EvidenceRefs = compactStringsV0(request.EvidenceRefs)
	return request
}

func ValidateTopicRegistryUpdateRequestV0(
	request TopicRegistryUpdateRequestV0,
) []TopicRegistryIssueV0 {
	request = NormalizeTopicRegistryUpdateRequestV0(request)
	var issues []TopicRegistryIssueV0
	if request.ToolPath == "" {
		issues = append(issues, issueV0(ErrTopicRegistryToolPathRequiredV0, "tool_path"))
	}
	if request.CourseID == "" {
		issues = append(issues, issueV0(ErrTopicRegistryCourseIDRequiredV0, "course_id"))
	}
	if request.TopicID == "" {
		issues = append(issues, issueV0(ErrTopicRegistryTopicIDRequiredV0, "topic_id"))
	}
	if request.Action != TopicRegistryActionUpdateV0 &&
		request.Action != TopicRegistryActionReleaseV0 {
		issues = append(issues, issueV0(ErrTopicRegistryActionInvalidV0, "action"))
	}
	return issues
}

func issueV0(code string, field string) TopicRegistryIssueV0 {
	return TopicRegistryIssueV0{Code: strings.TrimSpace(code), Field: strings.TrimSpace(field)}
}
