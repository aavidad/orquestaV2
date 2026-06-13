package orquestaopestopicregistry

import "context"

func ApplyTopicRegistryUpdateV0(
	ctx context.Context,
	request TopicRegistryUpdateRequestV0,
	runner TopicRegistryCommandRunnerPortV0,
) (TopicRegistryUpdateResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request = NormalizeTopicRegistryUpdateRequestV0(request)
	result := TopicRegistryUpdateResultV0{
		SchemaVersion: TopicRegistryUpdateResultSchemaV0,
		Status:        TopicRegistryUpdateStatusInvalidV0,
		Action:        request.Action,
		CourseID:      request.CourseID,
		TopicID:       request.TopicID,
		EvidenceRefs:  compactStringsV0(append(request.EvidenceRefs, "evidence-ref-opes-topic-registry")),
	}
	if issues := ValidateTopicRegistryUpdateRequestV0(request); len(issues) > 0 {
		result.Issues = append(result.Issues, issues...)
		return result, nil
	}
	if runner == nil {
		result.Issues = append(result.Issues, issueV0(ErrTopicRegistryRunnerRequiredV0, "runner"))
		return result, nil
	}
	args := TopicRegistryCLIArgsV0(request)
	result.CommandArgs = append([]string(nil), args...)
	commandResult, err := runner.RunTopicRegistryCommandV0(ctx, TopicRegistryCommandInvocationV0{
		ToolPath: request.ToolPath,
		Args:     args,
	})
	if err != nil || commandResult.ExitCode != 0 {
		result.Status = TopicRegistryUpdateStatusFailedV0
		result.Issues = append(result.Issues, issueV0(ErrTopicRegistryCommandFailedV0, "command"))
		return result, err
	}
	result.Status = TopicRegistryUpdateStatusAppliedV0
	return result, nil
}
