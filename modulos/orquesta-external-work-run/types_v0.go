package orquestaexternalworkrun

import (
	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

const (
	StartExternalWorkRunRequestSchemaV0 = "external_work_run_request.v0"
	StartExternalWorkRunResultSchemaV0  = "external_work_run_result.v0"

	ExternalWorkRunStatusAcceptedV0 = "accepted"
	ExternalWorkRunStatusInvalidV0  = "invalid"

	ExternalWorkRunDefaultQueueRefV0      = "global"
	ExternalWorkRunDefaultPriorityScoreV0 = 50
	ExternalWorkRunDefaultRequestedByV0   = "orquesta-external-work-run"

	ErrExternalWorkRunRunStoreRequiredV0       = "external_work_run_store_required"
	ErrExternalWorkRunEventSinkRequiredV0      = "external_work_event_sink_required"
	ErrExternalWorkRunQueueWriterRequiredV0    = "external_work_queue_writer_required"
	ErrExternalWorkRunAppChangeStoreV0         = "external_work_app_change_store_required"
	ErrExternalWorkRunAppChangeNotifierV0      = "external_work_app_change_notifier_required"
	ErrExternalWorkRunSchemaVersionV0          = "external_work_schema_version_invalid"
	ErrExternalWorkRunOccurredAtRequiredV0     = "external_work_occurred_at_required"
	ErrExternalWorkRunOccurredAtInvalidV0      = "external_work_occurred_at_invalid"
	ErrExternalWorkRunExternalWorkRequiredV0   = "external_work_required"
	ErrExternalWorkRunAppChangeInvalidV0       = "external_work_app_change_invalid"
	ErrExternalWorkRunExistingRunConflictV0    = "external_work_existing_run_conflict"
	ErrExternalWorkRunExistingChangeConflictV0 = "external_work_existing_change_conflict"
	ErrExternalWorkRunRefInvalidV0             = "external_work_ref_invalid"
	ErrExternalWorkRunProjectRefRequiredV0     = "external_work_project_ref_required"
	ErrExternalWorkRunAppSpecRefRequiredV0     = "external_work_app_spec_ref_required"
	ErrExternalWorkRunAppChangeResultInvalidV0 = "external_work_app_change_result_invalid"
	ErrExternalWorkRunMissingContextV0         = "external_work_missing_context"
	ErrExternalWorkRunRequiredInputMissingV0   = "external_work_required_input_missing"
	ErrExternalWorkRunGoalSpecInvalidV0        = "external_work_goal_spec_invalid"
)

type StartExternalWorkRunRequestV0 struct {
	SchemaVersion    string                               `json:"schema_version"`
	RequestID        string                               `json:"request_id,omitempty"`
	CorrelationID    string                               `json:"correlation_id,omitempty"`
	RunRef           string                               `json:"run_ref,omitempty"`
	ProjectRef       string                               `json:"project_ref,omitempty"`
	AppSpecRef       string                               `json:"app_spec_ref,omitempty"`
	QueueRef         string                               `json:"queue_ref,omitempty"`
	PriorityScore    int                                  `json:"priority_score,omitempty"`
	OccurredAt       string                               `json:"occurred_at,omitempty"`
	RequestedBy      string                               `json:"requested_by,omitempty"`
	AppChangeRequest orquestaappchange.AppChangeRequestV0 `json:"app_change_request"`
}

type StartExternalWorkRunConfigV0 struct {
	QueueRef             string `json:"queue_ref,omitempty"`
	DefaultPriorityScore int    `json:"default_priority_score,omitempty"`
	OccurredAt           string `json:"occurred_at,omitempty"`
	RequestedBy          string `json:"requested_by,omitempty"`
}

type StartExternalWorkRunPortsV0 struct {
	RunStore  orquestacionnucleoapp.RunStorePortV0
	EventSink orquestacionnucleoapp.EventSinkPortV0
	RunQueue  orquestarunqueue.RunQueuePriorityWriterPortV0
	AppChange orquestaappchange.AppChangePortsV0
}

type StartExternalWorkRunResultV0 struct {
	SchemaVersion       string                   `json:"schema_version"`
	Status              string                   `json:"status"`
	RequestID           string                   `json:"request_id,omitempty"`
	CorrelationID       string                   `json:"correlation_id,omitempty"`
	RunRef              string                   `json:"run_ref,omitempty"`
	ProjectRef          string                   `json:"project_ref,omitempty"`
	AppRef              string                   `json:"app_ref,omitempty"`
	ChangeRef           string                   `json:"change_ref,omitempty"`
	DirectorQuestionRef string                   `json:"director_question_ref,omitempty"`
	EvidenceRefs        []string                 `json:"evidence_refs,omitempty"`
	Issues              []ExternalWorkRunIssueV0 `json:"issues,omitempty"`
}

type ExternalWorkRunIssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}
