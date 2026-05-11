package orquestaobservability

import (
	"errors"
	"strings"
)

func (err OrquestaEventValidationErrorV0) Error() string {
	if len(err.Issues) == 0 {
		return ErrOrquestaEventInvalidoV0
	}
	return err.Issues[0].Code
}

func DecodeOrquestaEventV0(data []byte) (OrquestaEventV0, error) {
	var event OrquestaEventV0
	if err := decodeStrictJSONV0(data, &event); err != nil {
		return OrquestaEventV0{}, validationErrorV0(ErrOrquestaEventInvalidoV0, "")
	}
	if err := ValidateOrquestaEventV0(event); err != nil {
		return OrquestaEventV0{}, err
	}
	return event, nil
}

func DecodePublishOrquestaEventRequestV0(data []byte) (PublishOrquestaEventRequestV0, error) {
	var request PublishOrquestaEventRequestV0
	if err := decodeStrictJSONV0(data, &request); err != nil {
		return PublishOrquestaEventRequestV0{}, validationErrorV0(ErrOrquestaEventInvalidoV0, "")
	}
	if err := ValidatePublishOrquestaEventRequestV0(request); err != nil {
		return PublishOrquestaEventRequestV0{}, err
	}
	return request, nil
}

func ValidatePublishOrquestaEventRequestV0(request PublishOrquestaEventRequestV0) error {
	var issues []OrquestaEventValidationIssueV0
	add := addIssueFuncV0(&issues)

	correlationID := strings.TrimSpace(request.CorrelationID)
	if correlationID == "" {
		add(ErrCorrelationIDRequeridoV0, "correlation_id")
	} else if !isOpaqueIDV0(correlationID) {
		add(ErrOrquestaEventInvalidoV0, "correlation_id")
	}
	if strings.TrimSpace(request.IdempotencyKey) == "" {
		add(ErrIdempotencyKeyRequeridaV0, "idempotency_key")
	} else if !isOpaqueIDV0(request.IdempotencyKey) {
		add(ErrOrquestaEventInvalidoV0, "idempotency_key")
	}
	validateOptionalOpaqueIDV0(request.RequestID, "request_id", add)

	validateOrquestaEventV0(request.Event, "event", add)
	if correlationID != "" && strings.TrimSpace(request.Event.Correlation.CorrelationID) != "" && correlationID != strings.TrimSpace(request.Event.Correlation.CorrelationID) {
		add(ErrOrquestaEventInvalidoV0, "event.correlation.correlation_id")
	}

	if len(issues) > 0 {
		return OrquestaEventValidationErrorV0{Issues: issues}
	}
	return nil
}

func ValidateOrquestaEventV0(event OrquestaEventV0) error {
	var issues []OrquestaEventValidationIssueV0
	validateOrquestaEventV0(event, "", addIssueFuncV0(&issues))
	if len(issues) > 0 {
		return OrquestaEventValidationErrorV0{Issues: issues}
	}
	return nil
}

func AcceptPublishOrquestaEventRequestV0(request PublishOrquestaEventRequestV0) (PublishOrquestaEventAcceptedV0, error) {
	if err := ValidatePublishOrquestaEventRequestV0(request); err != nil {
		return PublishOrquestaEventAcceptedV0{}, err
	}
	return PublishOrquestaEventAcceptedV0{
		Accepted:      true,
		EventID:       strings.TrimSpace(request.Event.EventID),
		CorrelationID: strings.TrimSpace(request.CorrelationID),
		Stored:        false,
	}, nil
}

func HasOrquestaEventIssueV0(err error, code string) bool {
	var validationErr OrquestaEventValidationErrorV0
	if !errors.As(err, &validationErr) {
		return false
	}
	for _, issue := range validationErr.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
