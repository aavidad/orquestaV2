package orquestaappchange

import "context"

const (
	AppChangeRequestSchemaV0 = "app_change_request.v0"
	AppChangeResultSchemaV0  = "app_change_result.v0"

	AppChangeStatusAcceptedV0 = "accepted"
	AppChangeStatusInvalidV0  = "invalid"

	ErrAppChangeRunRefRequiredV0        = "app_change_run_ref_required"
	ErrAppChangeRunRefInvalidV0         = "app_change_run_ref_invalid"
	ErrAppChangeChangeRefRequiredV0     = "app_change_change_ref_required"
	ErrAppChangeChangeRefInvalidV0      = "app_change_change_ref_invalid"
	ErrAppChangeIntentRequiredV0        = "app_change_user_intent_required"
	ErrAppChangeWriteSetInvalidV0       = "app_change_write_set_invalid"
	ErrAppChangePortUnavailableV0       = "app_change_port_unavailable"
	ErrAppChangeCurrentStateRefV0       = "app_change_current_state_ref_invalid"
	ErrAppChangeMetadataRefV0           = "app_change_metadata_ref_invalid"
	ErrAppChangeExternalWorkRefV0       = "app_change_external_work_ref_invalid"
	ErrAppChangeIntentEventSchemaV0     = "app_change_intent_event_schema_invalid"
	ErrAppChangeIntentEventRefV0        = "app_change_intent_event_ref_required"
	ErrAppChangeIntentEventRefInvalidV0 = "app_change_intent_event_ref_invalid"
	AppChangeDefaultRequestedByV0       = "orquesta-app-change"
	AppChangeDefaultDirectorQuestionV0  = "question-ref-app-change"
)

type AppChangeRequestV0 struct {
	SchemaVersion      string                   `json:"schema_version"`
	RequestID          string                   `json:"request_id,omitempty"`
	CorrelationID      string                   `json:"correlation_id,omitempty"`
	RunRef             string                   `json:"run_ref"`
	AppRef             string                   `json:"app_ref,omitempty"`
	ChangeRef          string                   `json:"change_ref"`
	ActorRef           string                   `json:"actor_ref,omitempty"`
	Locale             string                   `json:"locale,omitempty"`
	UserIntent         string                   `json:"user_intent"`
	TargetArea         string                   `json:"target_area,omitempty"`
	CurrentStateRefs   []string                 `json:"current_state_refs,omitempty"`
	Scope              []string                 `json:"scope,omitempty"`
	AcceptanceCriteria []string                 `json:"acceptance_criteria,omitempty"`
	Constraints        []string                 `json:"constraints,omitempty"`
	AllowedWriteSet    []string                 `json:"allowed_write_set,omitempty"`
	MetadataRefs       []string                 `json:"metadata_refs,omitempty"`
	ExternalWork       *AppChangeExternalWorkV0 `json:"external_work,omitempty"`
}

type AppChangeExternalWorkV0 struct {
	ProjectRef    string   `json:"project_ref,omitempty"`
	InterfaceRefs []string `json:"interface_refs,omitempty"`
	WorkKind      string   `json:"work_kind,omitempty"`
	WorkRefs      []string `json:"work_refs,omitempty"`
}

type AppChangeRecordV0 struct {
	Request     AppChangeRequestV0 `json:"request"`
	ReceivedAt  string             `json:"received_at,omitempty"`
	RequestedBy string             `json:"requested_by,omitempty"`
}

type AppChangeRecordFilterV0 struct {
	RunRef string `json:"run_ref,omitempty"`
}

type AppChangeDirectorNotificationV0 struct {
	DirectorQuestionRef string   `json:"director_question_ref,omitempty"`
	EvidenceRefs        []string `json:"evidence_refs,omitempty"`
}

type AppChangeResultV0 struct {
	SchemaVersion       string             `json:"schema_version"`
	Status              string             `json:"status"`
	RequestID           string             `json:"request_id,omitempty"`
	CorrelationID       string             `json:"correlation_id,omitempty"`
	RunRef              string             `json:"run_ref,omitempty"`
	AppRef              string             `json:"app_ref,omitempty"`
	ChangeRef           string             `json:"change_ref,omitempty"`
	DirectorQuestionRef string             `json:"director_question_ref,omitempty"`
	EvidenceRefs        []string           `json:"evidence_refs,omitempty"`
	Issues              []AppChangeIssueV0 `json:"issues,omitempty"`
}

type AppChangeIssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

type AppChangePortsV0 struct {
	Store            AppChangeStorePortV0
	DirectorNotifier AppChangeDirectorNotifierPortV0
}

type AppChangeStorePortV0 interface {
	SaveAppChangeRequestV0(context.Context, AppChangeRecordV0) error
}

type AppChangeRecordSourcePortV0 interface {
	ListAppChangeRecordsV0(context.Context, AppChangeRecordFilterV0) ([]AppChangeRecordV0, error)
}

type AppChangeRecordStorePortV0 interface {
	AppChangeStorePortV0
	AppChangeRecordSourcePortV0
}

type AppChangeDirectorNotifierPortV0 interface {
	NotifyAppChangeRequestedV0(context.Context, AppChangeRecordV0) (AppChangeDirectorNotificationV0, error)
}
