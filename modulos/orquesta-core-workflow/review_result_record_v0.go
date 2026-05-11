package orquestacoreworkflow

import (
	"encoding/json"
	"errors"
	"strings"
)

type RecordReviewResultCommandPayloadV0 = ReviewResultV0
type ReviewResultRecordedPayloadV0 = ReviewResultV0

func NewRecordReviewResultCommandV0(meta OrchestrationCommandMetaV0, payload RecordReviewResultCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandRecordReviewResultV0, NormalizeReviewResultV0(payload))
}

func NewReviewResultRecordedEventV0(meta OrchestrationEventMetaV0, payload ReviewResultRecordedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventReviewResultRecordedV0, NormalizeReviewResultV0(payload))
}

func decodeRecordReviewResultCommandPayloadV0(raw json.RawMessage) (RecordReviewResultCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxReviewResultPayloadBytesV0 {
		return ReviewResultV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload ReviewResultV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ReviewResultV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = NormalizeReviewResultV0(payload)
	if err := validateRecordReviewResultPayloadV0(payload); err != nil {
		return ReviewResultV0{}, reviewResultCommandErrorV0(err)
	}
	return payload, nil
}

func validateReviewResultRecordedPayloadV0(event OrchestrationEventV0) error {
	var payload ReviewResultV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	payload = NormalizeReviewResultV0(payload)
	if err := validateRecordReviewResultPayloadV0(payload); err != nil {
		return reviewResultEventErrorV0(err)
	}
	return nil
}

func handleRecordReviewResultCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeRecordReviewResultCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureRecordReviewResultCommandAllowedV0(current, command, payload); err != nil {
		return emptyCommandResultV0(), err
	}
	matches, conflicts := reviewResultRecordMatchesPayloadV0(current, payload)
	if conflicts {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload.review_result_ref")
	}
	if matches {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventReviewResultRecordedV0, payload.ReviewResultRef, reviewResultRecordedPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	event, err := NewReviewResultRecordedEventV0(commandEventMetaV0(current, command, OrchestrationEventReviewResultRecordedV0), reviewResultRecordedPayloadFromCommandV0(payload))
	return eventCommandResultV0(event, err)
}

func applyReviewResultRecordedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload ReviewResultV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	payload = NormalizeReviewResultV0(payload)
	if err := ensureReviewResultRecordedEventAllowedV0(current, event, payload); err != nil {
		return current, err
	}
	matches, conflicts := reviewResultRecordMatchesPayloadV0(current, payload)
	if conflicts {
		return current, eventErrorV0(ErrSecuenciaInvalidaV0, "payload.review_result_ref")
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.ReviewResultRef); err != nil {
		return current, err
	}

	next := cloneRunForReducerV0(current)
	if !matches {
		next.ReviewResults = appendUniqueCompactRefV0(cloneStringsV0(next.ReviewResults), reviewResultProjectionRefV0(payload))
	}
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.ReviewResultRef)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func reviewResultRecordedPayloadFromCommandV0(payload RecordReviewResultCommandPayloadV0) ReviewResultRecordedPayloadV0 {
	return NormalizeReviewResultV0(payload)
}

func validateRecordReviewResultPayloadV0(payload ReviewResultV0) error {
	if err := ValidateReviewResultV0(payload); err != nil {
		return err
	}
	if reviewResultProjectionFieldUnsafeV0(payload) {
		return reviewResultErrorV0(ErrReviewResultPayloadInvalidoV0, "review_result_ref")
	}
	return nil
}

func reviewResultCommandErrorV0(err error) error {
	var reviewErr ReviewResultErrorV0
	if !errors.As(err, &reviewErr) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if reviewErr.Code == ErrDetalleProhibidoV0 {
		return commandErrorV0(ErrDetalleProhibidoV0, reviewResultPayloadFieldV0(reviewErr.Field))
	}
	return commandErrorV0(ErrPayloadInvalidoV0, reviewResultPayloadFieldV0(reviewErr.Field))
}

func reviewResultEventErrorV0(err error) error {
	var reviewErr ReviewResultErrorV0
	if !errors.As(err, &reviewErr) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if reviewErr.Code == ErrDetalleProhibidoV0 {
		return eventErrorV0(ErrDetalleProhibidoV0, reviewResultPayloadFieldV0(reviewErr.Field))
	}
	return eventErrorV0(ErrPayloadInvalidoV0, reviewResultPayloadFieldV0(reviewErr.Field))
}

func reviewResultPayloadFieldV0(field string) string {
	field = strings.TrimSpace(field)
	if field == "" {
		return "payload"
	}
	return "payload." + field
}
