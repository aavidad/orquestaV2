package orquestaappchange

import "context"

func RequestAppChangeV0(
	ctx context.Context,
	request AppChangeRequestV0,
	ports AppChangePortsV0,
) (AppChangeResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request = normalizeAppChangeRequestV0(request)
	if issues := validateAppChangeRequestV0(request); len(issues) > 0 {
		return invalidAppChangeResultV0(request, issues), nil
	}
	record := AppChangeRecordV0{
		Request:     request,
		RequestedBy: AppChangeDefaultRequestedByV0,
	}
	return requestAppChangeRecordV0(ctx, record, ports)
}

func requestAppChangeRecordV0(
	ctx context.Context,
	record AppChangeRecordV0,
	ports AppChangePortsV0,
) (AppChangeResultV0, error) {
	record = normalizeAppChangeRecordV0(record)
	request := record.Request
	if issues := validateAppChangePortsV0(ports); len(issues) > 0 {
		return invalidAppChangeResultV0(request, issues), nil
	}
	if err := ports.Store.SaveAppChangeRequestV0(ctx, record); err != nil {
		return AppChangeResultV0{}, err
	}
	notification, err := ports.DirectorNotifier.NotifyAppChangeRequestedV0(ctx, record)
	if err != nil {
		return AppChangeResultV0{}, err
	}
	return acceptedAppChangeResultV0(request, notification), nil
}

func invalidAppChangeResultV0(
	request AppChangeRequestV0,
	issues []AppChangeIssueV0,
) AppChangeResultV0 {
	return AppChangeResultV0{
		SchemaVersion: AppChangeResultSchemaV0,
		Status:        AppChangeStatusInvalidV0,
		RequestID:     request.RequestID,
		CorrelationID: request.CorrelationID,
		RunRef:        request.RunRef,
		AppRef:        request.AppRef,
		ChangeRef:     request.ChangeRef,
		Issues:        append([]AppChangeIssueV0(nil), issues...),
	}
}

func acceptedAppChangeResultV0(
	request AppChangeRequestV0,
	notification AppChangeDirectorNotificationV0,
) AppChangeResultV0 {
	questionRef := notification.DirectorQuestionRef
	if questionRef == "" {
		questionRef = AppChangeDefaultDirectorQuestionV0 + "-" + request.ChangeRef
	}
	return AppChangeResultV0{
		SchemaVersion:       AppChangeResultSchemaV0,
		Status:              AppChangeStatusAcceptedV0,
		RequestID:           request.RequestID,
		CorrelationID:       request.CorrelationID,
		RunRef:              request.RunRef,
		AppRef:              request.AppRef,
		ChangeRef:           request.ChangeRef,
		DirectorQuestionRef: questionRef,
		EvidenceRefs:        compactAppChangeStringsV0(notification.EvidenceRefs),
	}
}
