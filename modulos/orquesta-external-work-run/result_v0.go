package orquestaexternalworkrun

import orquestaappchange "orquesta/modulos/orquesta-app-change"

func invalidStartExternalWorkRunResultV0(
	request StartExternalWorkRunRequestV0,
	issues []ExternalWorkRunIssueV0,
) StartExternalWorkRunResultV0 {
	return StartExternalWorkRunResultV0{
		SchemaVersion: StartExternalWorkRunResultSchemaV0,
		Status:        ExternalWorkRunStatusInvalidV0,
		RequestID:     request.RequestID,
		CorrelationID: request.CorrelationID,
		RunRef:        request.RunRef,
		ProjectRef:    request.ProjectRef,
		AppRef:        request.AppChangeRequest.AppRef,
		ChangeRef:     request.AppChangeRequest.ChangeRef,
		Issues:        append([]ExternalWorkRunIssueV0(nil), issues...),
	}
}

func acceptedStartExternalWorkRunResultV0(
	request StartExternalWorkRunRequestV0,
	change orquestaappchange.AppChangeResultV0,
	evidenceRefs []string,
) StartExternalWorkRunResultV0 {
	return StartExternalWorkRunResultV0{
		SchemaVersion:       StartExternalWorkRunResultSchemaV0,
		Status:              ExternalWorkRunStatusAcceptedV0,
		RequestID:           request.RequestID,
		CorrelationID:       request.CorrelationID,
		RunRef:              request.RunRef,
		ProjectRef:          request.ProjectRef,
		AppRef:              change.AppRef,
		ChangeRef:           change.ChangeRef,
		DirectorQuestionRef: change.DirectorQuestionRef,
		EvidenceRefs: compactExternalWorkRunStringsV0(
			append(evidenceRefs, change.EvidenceRefs...),
		),
	}
}

func appChangeResultIssuesV0(
	change orquestaappchange.AppChangeResultV0,
) []ExternalWorkRunIssueV0 {
	issues := make([]ExternalWorkRunIssueV0, 0, len(change.Issues)+1)
	if len(change.Issues) == 0 {
		issues = append(issues, externalWorkRunIssueV0(ErrExternalWorkRunAppChangeResultInvalidV0, "app_change_request"))
	}
	for _, issue := range change.Issues {
		issues = append(issues, externalWorkRunIssueV0(
			ErrExternalWorkRunAppChangeInvalidV0+":"+issue.Code,
			"app_change_request."+issue.Field,
		))
	}
	return issues
}
