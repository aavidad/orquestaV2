package orquestaappchange

const AppChangeIntentEventSchemaV0 = "app_change_intent_event.v0"

type AppChangeIntentEventV0 struct {
	SchemaVersion      string                   `json:"schema_version"`
	EventID            string                   `json:"event_id,omitempty"`
	Source             string                   `json:"source,omitempty"`
	OccurredAt         string                   `json:"occurred_at,omitempty"`
	RequestID          string                   `json:"request_id,omitempty"`
	CorrelationID      string                   `json:"correlation_id,omitempty"`
	RunRef             string                   `json:"run_ref"`
	AppRef             string                   `json:"app_ref,omitempty"`
	ChangeRef          string                   `json:"change_ref,omitempty"`
	ActorRef           string                   `json:"actor_ref,omitempty"`
	Locale             string                   `json:"locale,omitempty"`
	UserIntent         string                   `json:"user_intent"`
	TargetArea         string                   `json:"target_area,omitempty"`
	CurrentStateRefs   []string                 `json:"current_state_refs,omitempty"`
	Scope              []string                 `json:"scope,omitempty"`
	AcceptanceCriteria []string                 `json:"acceptance_criteria,omitempty"`
	Constraints        []string                 `json:"constraints,omitempty"`
	AllowedWriteSet    []string                 `json:"allowed_write_set,omitempty"`
	RequiredTests      []string                 `json:"required_tests,omitempty"`
	MetadataRefs       []string                 `json:"metadata_refs,omitempty"`
	ExternalWork       *AppChangeExternalWorkV0 `json:"external_work,omitempty"`
}
