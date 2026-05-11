package orquestacore

import (
	"context"
	"strings"
)

const GovernanceCatalogContractVersionV0 = "governance_catalog.v0"

type GovernanceCatalogPortV0 interface {
	ConsultarGovernanceCatalogV0(context.Context, GovernanceCatalogRequestV0) (GovernanceCatalogSnapshotV0, error)
}

type GovernanceCatalogRequestV0 struct {
	SchemaVersion string                   `json:"schema_version"`
	RequestID     string                   `json:"request_id,omitempty"`
	CorrelationID string                   `json:"correlation_id,omitempty"`
	Scope         GovernanceCatalogScopeV0 `json:"scope"`
}

type GovernanceCatalogScopeV0 struct {
	Modulo string   `json:"modulo,omitempty"`
	Rol    string   `json:"rol,omitempty"`
	Fase   string   `json:"fase,omitempty"`
	Tags   []string `json:"tags,omitempty"`
}

type GovernanceCatalogSnapshotV0 struct {
	SchemaVersion   string                     `json:"schema_version"`
	CorrelationID   string                     `json:"correlation_id,omitempty"`
	ReadAt          string                     `json:"read_at,omitempty"`
	Effective       []GovernanceCatalogEntryV0 `json:"effective"`
	ProposedCount   int                        `json:"proposed_count"`
	QuarantineCount int                        `json:"quarantine_count"`
}

type GovernanceCatalogEntryV0 struct {
	ID          string   `json:"id"`
	Tipo        string   `json:"tipo"`
	Modulo      string   `json:"modulo,omitempty"`
	Rol         string   `json:"rol,omitempty"`
	Fase        string   `json:"fase,omitempty"`
	DecisionRef string   `json:"decision_ref"`
	Version     string   `json:"version"`
	Summary     string   `json:"summary"`
	Rules       []string `json:"rules"`
}

type GovernanceCatalogErrorV0 struct {
	Code          string   `json:"code"`
	Message       string   `json:"message"`
	Field         string   `json:"field,omitempty"`
	Retryable     bool     `json:"retryable"`
	Evidence      []string `json:"evidence,omitempty"`
	CorrelationID string   `json:"correlation_id,omitempty"`
}

func (err GovernanceCatalogErrorV0) Error() string {
	return err.Code
}

func NewGovernanceCatalogRequestV0(requestID, correlationID string, scope GovernanceCatalogScopeV0) GovernanceCatalogRequestV0 {
	scope.Modulo = strings.TrimSpace(scope.Modulo)
	scope.Rol = strings.TrimSpace(scope.Rol)
	scope.Fase = strings.TrimSpace(scope.Fase)
	scope.Tags = compactStringsV0(scope.Tags)
	return GovernanceCatalogRequestV0{
		SchemaVersion: GovernanceCatalogContractVersionV0,
		RequestID:     strings.TrimSpace(requestID),
		CorrelationID: strings.TrimSpace(correlationID),
		Scope:         scope,
	}
}
