package orquestaobservability

import "sync"

const (
	WebHTMLRenderObservationSchemaVersionV0 = "web_html_render_observation.v0"

	WebHTMLRenderStageRenderV0 = "render"
	WebHTMLRenderStageWriteV0  = "write"

	WebHTMLRenderFailedReasonV0        = "web_html_render_failed"
	WebHTMLResponseWriteFailedReasonV0 = "web_response_write_failed"
)

type WebHTMLRenderObservationV0 struct {
	SchemaVersion string `json:"schema_version"`
	RouteRef      string `json:"route_ref"`
	ReasonCode    string `json:"reason_code"`
	Stage         string `json:"stage"`
	StatusCode    int    `json:"status_code"`
	Locale        string `json:"locale,omitempty"`
	CounterKey    string `json:"counter_key"`
	Count         int    `json:"count"`
}

type WebHTMLRenderObserverV0 interface {
	ObserveWebHTMLRenderV0(WebHTMLRenderObservationV0)
}

func NewWebHTMLRenderObservationV0(
	routeRef string,
	reasonCode string,
	stage string,
	statusCode int,
	locale string,
) WebHTMLRenderObservationV0 {
	observation := WebHTMLRenderObservationV0{
		SchemaVersion: WebHTMLRenderObservationSchemaVersionV0,
		RouteRef:      routeRef,
		ReasonCode:    reasonCode,
		Stage:         stage,
		StatusCode:    statusCode,
		Locale:        locale,
		Count:         1,
	}
	observation.CounterKey = webHTMLRenderCounterKeyV0(observation)
	return observation
}

func ValidateWebHTMLRenderObservationV0(observation WebHTMLRenderObservationV0) error {
	var issues []OperationalStatusValidationIssueV0
	add := addOperationalStatusIssueFuncV0(&issues)
	if observation.SchemaVersion != WebHTMLRenderObservationSchemaVersionV0 {
		add(ErrOperationalStatusQueryInvalidaV0, "schema_version")
	}
	validateRequiredOperationalRefV0(observation.RouteRef, "route_ref", add)
	validateOperationalTokenTextV0(observation.ReasonCode, "reason_code", add)
	if observation.Stage != WebHTMLRenderStageRenderV0 && observation.Stage != WebHTMLRenderStageWriteV0 {
		add(ErrOperationalStatusQueryInvalidaV0, "stage")
	}
	if observation.StatusCode < 100 || observation.StatusCode > 599 {
		add(ErrOperationalStatusQueryInvalidaV0, "status_code")
	}
	if observation.Locale != "" && !operationalLocalePatternV0.MatchString(observation.Locale) {
		add(ErrOperationalStatusQueryInvalidaV0, "locale")
	}
	validateOperationalTokenTextV0(observation.CounterKey, "counter_key", add)
	if observation.Count <= 0 {
		add(ErrOperationalStatusQueryInvalidaV0, "count")
	}
	if len(issues) > 0 {
		return OperationalStatusValidationErrorV0{Issues: issues}
	}
	return nil
}

type InMemoryWebHTMLRenderObserverV0 struct {
	mu           sync.Mutex
	observations []WebHTMLRenderObservationV0
	counters     map[string]int
}

func NewInMemoryWebHTMLRenderObserverV0() *InMemoryWebHTMLRenderObserverV0 {
	return &InMemoryWebHTMLRenderObserverV0{counters: map[string]int{}}
}

func (observer *InMemoryWebHTMLRenderObserverV0) ObserveWebHTMLRenderV0(observation WebHTMLRenderObservationV0) {
	if observer == nil || ValidateWebHTMLRenderObservationV0(observation) != nil {
		return
	}
	observer.mu.Lock()
	defer observer.mu.Unlock()
	observer.observations = append(observer.observations, observation)
	observer.counters[observation.CounterKey] += observation.Count
}

func (observer *InMemoryWebHTMLRenderObserverV0) SnapshotV0() ([]WebHTMLRenderObservationV0, map[string]int) {
	if observer == nil {
		return nil, nil
	}
	observer.mu.Lock()
	defer observer.mu.Unlock()
	observations := append([]WebHTMLRenderObservationV0(nil), observer.observations...)
	counters := make(map[string]int, len(observer.counters))
	for key, value := range observer.counters {
		counters[key] = value
	}
	return observations, counters
}

func webHTMLRenderCounterKeyV0(observation WebHTMLRenderObservationV0) string {
	if observation.ReasonCode == "" {
		return "web_html_render_unknown_total"
	}
	return observation.ReasonCode + "_total"
}
