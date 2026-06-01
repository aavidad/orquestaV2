package orquestaexternalworkrun

import (
	"context"
	"reflect"
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
)

func requestExternalWorkAppChangeOnceV0(
	ctx context.Context,
	request StartExternalWorkRunRequestV0,
	ports orquestaappchange.AppChangePortsV0,
) (orquestaappchange.AppChangeResultV0, error) {
	if source, ok := ports.Store.(orquestaappchange.AppChangeRecordSourcePortV0); ok {
		result, found, err := existingExternalWorkAppChangeResultV0(ctx, request, source)
		if err != nil || found {
			return result, err
		}
	}
	return orquestaappchange.RequestAppChangeV0(ctx, request.AppChangeRequest, ports)
}

func existingExternalWorkAppChangeResultV0(
	ctx context.Context,
	request StartExternalWorkRunRequestV0,
	source orquestaappchange.AppChangeRecordSourcePortV0,
) (orquestaappchange.AppChangeResultV0, bool, error) {
	records, err := source.ListAppChangeRecordsV0(
		ctx,
		orquestaappchange.AppChangeRecordFilterV0{RunRef: request.RunRef},
	)
	if err != nil {
		return orquestaappchange.AppChangeResultV0{}, false, err
	}
	for _, record := range records {
		if strings.TrimSpace(record.Request.ChangeRef) != request.AppChangeRequest.ChangeRef {
			continue
		}
		if !reflect.DeepEqual(
			orquestaappchange.PrepareAppChangeRequestV0(record.Request),
			request.AppChangeRequest,
		) {
			return orquestaappchange.AppChangeResultV0{
				SchemaVersion: orquestaappchange.AppChangeResultSchemaV0,
				Status:        orquestaappchange.AppChangeStatusInvalidV0,
				RequestID:     request.RequestID,
				CorrelationID: request.CorrelationID,
				RunRef:        request.RunRef,
				AppRef:        request.AppChangeRequest.AppRef,
				ChangeRef:     request.AppChangeRequest.ChangeRef,
				Issues: []orquestaappchange.AppChangeIssueV0{{
					Code:  ErrExternalWorkRunExistingChangeConflictV0,
					Field: "change_ref",
				}},
			}, true, nil
		}
		return orquestaappchange.AppChangeResultV0{
			SchemaVersion:       orquestaappchange.AppChangeResultSchemaV0,
			Status:              orquestaappchange.AppChangeStatusAcceptedV0,
			RequestID:           request.AppChangeRequest.RequestID,
			CorrelationID:       request.AppChangeRequest.CorrelationID,
			RunRef:              request.RunRef,
			AppRef:              request.AppChangeRequest.AppRef,
			ChangeRef:           request.AppChangeRequest.ChangeRef,
			DirectorQuestionRef: "question-ref-app-change-" + request.AppChangeRequest.ChangeRef,
			EvidenceRefs:        []string{"evidence-ref-external-work-app-change-existing"},
		}, true, nil
	}
	return orquestaappchange.AppChangeResultV0{}, false, nil
}
