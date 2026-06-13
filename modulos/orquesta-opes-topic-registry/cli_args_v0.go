package orquestaopestopicregistry

func TopicRegistryCLIArgsV0(request TopicRegistryUpdateRequestV0) []string {
	request = NormalizeTopicRegistryUpdateRequestV0(request)
	args := []string{
		request.Action,
		"--course-id", request.CourseID,
		"--topic-id", request.TopicID,
		"--agent-id", request.AgentID,
	}
	if request.Status != "" {
		args = append(args, "--status", request.Status)
	}
	if request.Summary != "" {
		args = append(args, "--summary", request.Summary)
	}
	if request.Done != "" {
		args = append(args, "--done", request.Done)
	}
	if request.Pending != "" {
		args = append(args, "--pending", request.Pending)
	}
	for _, ref := range request.EvidenceRefs {
		args = append(args, "--evidence-ref", ref)
	}
	if request.Force {
		args = append(args, "--force")
	}
	return args
}
