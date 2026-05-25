package orquestaweb

import (
	"net/http"
	"strconv"
)

func runQueueQueryFromURLV0(r *http.Request) WebRunQueueQueryV0 {
	return WebRunQueueQueryV0{
		RequestID:      r.URL.Query().Get("request_id"),
		CorrelationID:  r.URL.Query().Get("correlation_id"),
		Locale:         r.URL.Query().Get("locale"),
		Action:         r.URL.Query().Get("action"),
		QueueRef:       r.URL.Query().Get("queue_ref"),
		AppRefs:        queryValuesV0(r, "app_refs"),
		RunRef:         r.URL.Query().Get("run_ref"),
		AppRef:         r.URL.Query().Get("app_ref"),
		Status:         r.URL.Query().Get("status"),
		PriorityScore:  intQueryValueV0(r.URL.Query().Get("priority_score")),
		RequestedBy:    r.URL.Query().Get("requested_by"),
		Reason:         r.URL.Query().Get("reason"),
		IdempotencyKey: r.URL.Query().Get("idempotency_key"),
		Limit:          intQueryValueV0(r.URL.Query().Get("limit")),
		OccurredAt:     r.URL.Query().Get("occurred_at"),
	}
}

func runQueueQueryFromValuesV0(values map[string][]string) WebRunQueueQueryV0 {
	return WebRunQueueQueryV0{
		RequestID:      formValueV0(values, "request_id"),
		CorrelationID:  formValueV0(values, "correlation_id"),
		Locale:         formValueV0(values, "locale"),
		Action:         formValueV0(values, "action"),
		QueueRef:       formValueV0(values, "queue_ref"),
		AppRefs:        formValuesV0(values, "app_refs"),
		RunRef:         formValueV0(values, "run_ref"),
		AppRef:         formValueV0(values, "app_ref"),
		Status:         formValueV0(values, "status"),
		PriorityScore:  intQueryValueV0(formValueV0(values, "priority_score")),
		RequestedBy:    formValueV0(values, "requested_by"),
		Reason:         formValueV0(values, "reason"),
		IdempotencyKey: formValueV0(values, "idempotency_key"),
		Limit:          intQueryValueV0(formValueV0(values, "limit")),
		OccurredAt:     formValueV0(values, "occurred_at"),
	}
}

func queryValuesV0(r *http.Request, key string) []string {
	values := r.URL.Query()
	return formValuesV0(map[string][]string{key: values[key]}, key)
}

func intQueryValueV0(value string) int {
	parsed, err := strconv.Atoi(trimV0(value))
	if err != nil {
		return 0
	}
	return parsed
}
