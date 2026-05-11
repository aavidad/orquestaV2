package orquestaappdirectorintake

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
)

const (
	AppDirectorIntakeWizardResultSchemaVersionV0 = "app_director_intake_wizard_result.v0"

	AppDirectorIntakeWizardStatusNeedsInputV0 = "needs_input"
	AppDirectorIntakeWizardStatusReadyV0      = "ready"
	AppDirectorIntakeWizardStatusInvalidV0    = "invalid"

	ErrAppDirectorIntakeWizardRequestRefV0        = "app_director_intake_wizard_request_ref_required"
	ErrAppDirectorIntakeWizardOccurredAtV0        = "app_director_intake_wizard_occurred_at_required"
	ErrAppDirectorIntakeWizardOccurredAtInvalidV0 = "app_director_intake_wizard_occurred_at_invalid"
	ErrAppDirectorIntakeWizardFieldUnsupportedV0  = "app_director_intake_wizard_field_unsupported"
)

type AppDirectorIntakeWizardRequestV0 struct {
	RunRef        string                            `json:"run_ref,omitempty"`
	ProjectRef    string                            `json:"project_ref,omitempty"`
	RequestRef    string                            `json:"request_ref,omitempty"`
	Source        string                            `json:"source,omitempty"`
	Locale        string                            `json:"locale,omitempty"`
	OccurredAt    string                            `json:"occurred_at"`
	CorrelationID string                            `json:"correlation_id,omitempty"`
	RequestedBy   string                            `json:"requested_by,omitempty"`
	Draft         orquestafactory.AppSpecRequestV0  `json:"draft,omitempty"`
	Answers       []AppDirectorIntakeWizardAnswerV0 `json:"answers,omitempty"`
}

type AppDirectorIntakeWizardAnswerV0 struct {
	Field  string   `json:"field"`
	Value  string   `json:"value,omitempty"`
	Values []string `json:"values,omitempty"`
}

type AppDirectorIntakeWizardResultV0 struct {
	SchemaVersion string                                   `json:"schema_version"`
	Status        string                                   `json:"status"`
	Draft         orquestafactory.AppSpecRequestV0         `json:"draft"`
	NextQuestion  *orquestacoreworkflow.DirectorQuestionV0 `json:"next_question,omitempty"`
	AppSpec       *orquestafactory.AppSpecV0               `json:"app_spec,omitempty"`
	Prepared      *AppDirectorIntakePreparedV0             `json:"prepared,omitempty"`
	EvidenceRefs  []string                                 `json:"evidence_refs,omitempty"`
	Issues        []AppDirectorIntakeWizardIssueV0         `json:"issues,omitempty"`
}

type AppDirectorIntakeWizardIssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}
