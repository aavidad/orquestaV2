package orquestadirector

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestarails "orquesta/modulos/orquesta-rails"
)

var forbiddenOutboxDispatchCycleTermsV0 = []string{
	"sqlite", "postgres", "mysql", "mongo", "database", "db", "dsn", "sql",
	"provider", "proveedor", "home", "oauth", "transcript", "prompt",
	"secret", "secreto", "token", "password", "credential", "credencial",
}

func normalizeOutboxDispatchCycleInputV0(input OutboxDispatchCycleInputV0) OutboxDispatchCycleInputV0 {
	input.RunID = strings.TrimSpace(input.RunID)
	input.TargetPort = strings.TrimSpace(input.TargetPort)
	input.DispatchedAt = strings.TrimSpace(input.DispatchedAt)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)
	input.Messages = cloneOutboxDispatchCycleMessagesV0(input.Messages)
	return input
}

func validateOutboxDispatchCycleInputV0(ctx context.Context, input OutboxDispatchCycleInputV0) error {
	if ctx == nil {
		return outboxDispatchCycleValidationErrorV0(input, "context requerido", "context")
	}
	if isNilOutboxDispatchCyclePortV0(input.Ledger) {
		return outboxDispatchCycleValidationErrorV0(input, "ledger requerido", "ledger")
	}
	if isNilOutboxDispatchCyclePortV0(input.Dispatcher) {
		return outboxDispatchCycleValidationErrorV0(input, "dispatcher requerido", "dispatcher")
	}
	for _, required := range []struct {
		field string
		value string
	}{
		{field: "run_id", value: input.RunID},
		{field: "target_port", value: input.TargetPort},
		{field: "dispatched_at", value: input.DispatchedAt},
	} {
		if required.value == "" {
			return outboxDispatchCycleValidationErrorV0(input, "campo requerido", required.field)
		}
	}
	if !outboxDispatchCycleTargetSupportedV0(input.TargetPort) {
		return outboxDispatchCycleValidationErrorV0(input, "target_port no soportado", "target_port")
	}
	if _, err := time.Parse(time.RFC3339Nano, input.DispatchedAt); err != nil {
		return outboxDispatchCycleValidationErrorV0(input, "timestamp invalido", "dispatched_at")
	}
	return validateOutboxDispatchCycleMessagesV0(input)
}

func validateOutboxDispatchCycleMessagesV0(input OutboxDispatchCycleInputV0) error {
	for index, message := range input.Messages {
		if strings.TrimSpace(message.RunID) != input.RunID {
			return outboxDispatchCycleValidationErrorV0(input, "run_id no coincide", indexedCycleFieldV0(index, "run_id"))
		}
		if strings.TrimSpace(message.TargetPort) != input.TargetPort {
			return outboxDispatchCycleValidationErrorV0(input, "target_port no coincide", indexedCycleFieldV0(index, "target_port"))
		}
	}
	return nil
}

func outboxDispatchAckFromAttemptV0(
	input OutboxDispatchCycleInputV0,
	message orquestacoreworkflow.OutboxMessageV0,
	receipt OutboxDispatchReceiptV0,
	dispatchErr error,
) (OutboxDispatchAckV0, *OutboxDispatchCycleIssueV0) {
	dispatchRef := strings.TrimSpace(receipt.DispatchRef)
	if dispatchRef == "" {
		issue := outboxDispatchCycleIssueV0(ErrOutboxDispatchRefRequeridoV0, "dispatch_ref", "dispatch_ref requerido")
		return OutboxDispatchAckV0{}, &issue
	}
	ack := OutboxDispatchAckV0{
		MessageID:    strings.TrimSpace(message.MessageID),
		RunID:        strings.TrimSpace(message.RunID),
		TargetPort:   strings.TrimSpace(message.TargetPort),
		Status:       OutboxDispatchStatusDispatchedV0,
		DispatchRef:  dispatchRef,
		DispatchedAt: input.DispatchedAt,
		EvidenceRefs: compactOutboxDispatchCycleStringsV0(receipt.EvidenceRefs),
	}
	if dispatchErr != nil {
		ack.Status = OutboxDispatchStatusFailedV0
		ack.ErrorCode = compactOutboxDispatchErrorCodeV0(dispatchErr)
	}
	return ack, nil
}

func compactOutboxDispatchErrorCodeV0(err error) string {
	code := ""
	var coded OutboxDispatchCodedErrorV0
	if errors.As(err, &coded) {
		code = coded.OutboxDispatchErrorCodeV0()
	}
	if code == "" {
		code = ErrOutboxDispatchFailedV0
	}
	code = compactOutboxDispatchCycleCodeV0(code)
	if code == "" || outboxDispatchCycleHasForbiddenTermV0(code) {
		return ErrOutboxDispatchFailedV0
	}
	return code
}

func compactOutboxDispatchCycleCodeV0(value string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastUnderscore = false
		default:
			if !lastUnderscore && b.Len() > 0 {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
		if b.Len() >= 64 {
			break
		}
	}
	return strings.Trim(b.String(), "_")
}

func outboxDispatchCycleTargetSupportedV0(targetPort string) bool {
	switch targetPort {
	case orquestacoreworkflow.OutboxTargetPersistenceV0,
		orquestacoreworkflow.OutboxTargetObservabilityV0,
		orquestacoreworkflow.OutboxTargetCapacityV0,
		orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		orquestacoreworkflow.OutboxTargetDeployPlannerV0,
		orquestacoreworkflow.OutboxTargetDirectorV0:
		return true
	default:
		return false
	}
}

func compactOutboxDispatchCycleStringsV0(values []string) []string {
	if values == nil {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if compact := strings.TrimSpace(value); compact != "" {
			result = append(result, compact)
		}
	}
	return result
}

func cloneOutboxDispatchCycleMessagesV0(
	messages []orquestacoreworkflow.OutboxMessageV0,
) []orquestacoreworkflow.OutboxMessageV0 {
	if messages == nil {
		return nil
	}
	cloned := make([]orquestacoreworkflow.OutboxMessageV0, 0, len(messages))
	for _, message := range messages {
		message.Payload = append([]byte(nil), message.Payload...)
		cloned = append(cloned, message)
	}
	return cloned
}

func outboxDispatchCycleIssueV0(code, field, message string) OutboxDispatchCycleIssueV0 {
	return OutboxDispatchCycleIssueV0{
		Code:    strings.TrimSpace(code),
		Field:   strings.TrimSpace(field),
		Message: strings.TrimSpace(message),
	}
}

func outboxDispatchCycleValidationErrorV0(
	input OutboxDispatchCycleInputV0,
	message string,
	field string,
) OutboxDispatchCycleErrorV0 {
	issue := outboxDispatchCycleIssueV0(ErrDirectorOutboxDispatchCycleInvalidoV0, field, message)
	return outboxDispatchCycleErrorV0(input, ErrDirectorOutboxDispatchCycleInvalidoV0, message, field, []OutboxDispatchCycleIssueV0{issue})
}

func outboxDispatchCycleErrorV0(
	input OutboxDispatchCycleInputV0,
	code string,
	message string,
	field string,
	issues []OutboxDispatchCycleIssueV0,
) OutboxDispatchCycleErrorV0 {
	return OutboxDispatchCycleErrorV0{
		Code:          strings.TrimSpace(code),
		Message:       strings.TrimSpace(message),
		Field:         strings.TrimSpace(field),
		Retryable:     false,
		Issues:        append([]OutboxDispatchCycleIssueV0(nil), issues...),
		CorrelationID: input.CorrelationID,
	}
}

func isNilOutboxDispatchCyclePortV0(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}

func indexedCycleFieldV0(index int, field string) string {
	return "messages." + strconv.Itoa(index) + "." + field
}

func outboxDispatchCycleHasForbiddenTermV0(value string) bool {
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return false
	}
	normalized := strings.ToLower(strings.TrimSpace(value))
	for _, term := range forbiddenOutboxDispatchCycleTermsV0 {
		if strings.Contains(normalized, term) {
			return true
		}
	}
	return false
}
