package orquestaruntime

import "context"

func LaunchExternalAgentProcessV0(
	ctx context.Context,
	spec ExternalAgentLaunchSpecV0,
	resolver ExternalAgentProcessCommandResolverV0,
	runtime ExternalAgentProcessRuntimePortV0,
) ExternalAgentProcessLaunchResultV0 {
	if ctx == nil {
		ctx = context.Background()
	}
	if issues := externalAgentProcessSpecIssuesV0(spec); len(issues) > 0 {
		return externalAgentProcessBlockedV0(issues)
	}
	if resolver == nil {
		return externalAgentProcessBlockedV0([]ExternalAgentConnectorErrorV0{
			externalAgentProcessErrorV0(ExternalAgentResolverUnavailableV0, "resolver", spec.CorrelationID, false),
		})
	}
	if runtime == nil {
		return externalAgentProcessBlockedV0([]ExternalAgentConnectorErrorV0{
			externalAgentProcessErrorV0(ExternalAgentRuntimeUnavailableV0, "runtime", spec.CorrelationID, false),
		})
	}

	req, issues := resolver.ResolveExternalAgentProcessCommandV0(ctx, spec)
	if len(issues) > 0 {
		return externalAgentProcessBlockedV0(
			externalAgentProcessNormalizeIssuesV0(spec.CorrelationID, issues),
		)
	}
	if err := validateProcessRuntimeLaunchRequestV0(req); err != nil {
		return externalAgentProcessBlockedV0([]ExternalAgentConnectorErrorV0{
			externalAgentProcessIssueFromRuntimeErrorV0(
				spec.CorrelationID,
				ExternalAgentCommandResolutionInvalidaV0,
				err,
			),
		})
	}

	snapshot, err := runtime.LaunchV0(ctx, req)
	if err != nil {
		return externalAgentProcessBlockedV0([]ExternalAgentConnectorErrorV0{
			externalAgentProcessIssueFromRuntimeErrorV0(
				spec.CorrelationID,
				ExternalAgentRuntimeLaunchFailedV0,
				err,
			),
		})
	}
	if issues := validateProcessRuntimeSnapshotV0(snapshot); len(issues) > 0 {
		return externalAgentProcessBlockedV0(
			externalAgentProcessSnapshotIssuesV0(spec.CorrelationID, issues),
		)
	}
	return ExternalAgentProcessLaunchResultV0{
		Status:   ExternalAgentProcessLaunchStartedV0,
		Snapshot: snapshot,
	}
}

func externalAgentProcessSpecIssuesV0(
	spec ExternalAgentLaunchSpecV0,
) []ExternalAgentConnectorErrorV0 {
	issues := append([]ExternalAgentConnectorErrorV0{}, spec.Issues...)
	issues = append(issues, ValidateExternalAgentLaunchSpecV0(spec)...)
	return externalAgentProcessNormalizeIssuesV0(spec.CorrelationID, issues)
}

func externalAgentProcessBlockedV0(
	issues []ExternalAgentConnectorErrorV0,
) ExternalAgentProcessLaunchResultV0 {
	return ExternalAgentProcessLaunchResultV0{
		Status: ExternalAgentProcessLaunchBlockedV0,
		Issues: issues,
	}
}

func externalAgentProcessErrorV0(
	code ExternalAgentConnectorErrorCodeV0,
	field string,
	correlationID string,
	retryable bool,
) ExternalAgentConnectorErrorV0 {
	return ExternalAgentConnectorErrorV0{
		Code:          code,
		MessageKey:    "orquesta.runtime.external_agent_process." + string(code),
		Field:         field,
		Retryable:     retryable,
		CorrelationID: correlationID,
	}
}

func externalAgentProcessIssueFromRuntimeErrorV0(
	correlationID string,
	code ExternalAgentConnectorErrorCodeV0,
	err error,
) ExternalAgentConnectorErrorV0 {
	issue := externalAgentProcessErrorV0(code, "runtime", correlationID, false)
	if runtimeErr, ok := err.(ProcessRuntimeErrorV0); ok {
		issue.Field = runtimeErr.Field
		issue.Retryable = code == ExternalAgentRuntimeLaunchFailedV0 && runtimeErr.Retryable
		issue.Evidence = append([]string{string(runtimeErr.Code)}, runtimeErr.Evidence...)
		return issue
	}
	issue.Evidence = []string{"runtime_error"}
	return issue
}

func externalAgentProcessNormalizeIssuesV0(
	correlationID string,
	issues []ExternalAgentConnectorErrorV0,
) []ExternalAgentConnectorErrorV0 {
	if len(issues) == 0 {
		return nil
	}
	normalized := make([]ExternalAgentConnectorErrorV0, 0, len(issues))
	for _, issue := range issues {
		if issue.CorrelationID == "" {
			issue.CorrelationID = correlationID
		}
		normalized = append(normalized, issue)
	}
	return normalized
}

func externalAgentProcessSnapshotIssuesV0(
	correlationID string,
	issues []ProcessRuntimeErrorV0,
) []ExternalAgentConnectorErrorV0 {
	normalized := make([]ExternalAgentConnectorErrorV0, 0, len(issues))
	for _, issue := range issues {
		normalized = append(normalized, externalAgentProcessIssueFromRuntimeErrorV0(
			correlationID,
			ExternalAgentRuntimeSnapshotInvalidaV0,
			issue,
		))
	}
	return normalized
}
