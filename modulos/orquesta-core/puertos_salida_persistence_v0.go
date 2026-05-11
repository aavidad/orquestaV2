package orquestacore

import (
	"context"
	"strings"
)

const (
	PersistenceRepositoryContractVersionV0                  = "PersistenceRepositoryV0"
	PersistenceRepositoryOperationGuardarProyectoBorradorV0 = "guardar_proyecto_borrador"
	ProyectoPlanBorradorPayloadVersionV0                    = "ProyectoPlanBorradorV0"
	PersistenceEstadoPersistidoV0                           = "persistido"
)

type PersistenceRepositoryPortV0 interface {
	GuardarProyectoBorradorV0(context.Context, GuardarProyectoBorradorPersistenceRequestV0) (GuardarProyectoBorradorPersistenceAcceptedV0, error)
}

type GuardarProyectoBorradorPersistenceRequestV0 struct {
	SchemaVersion  string                 `json:"schema_version"`
	Contrato       string                 `json:"contrato"`
	Operacion      string                 `json:"operacion"`
	RequestID      string                 `json:"request_id,omitempty"`
	CorrelationID  string                 `json:"correlation_id,omitempty"`
	IdempotencyKey string                 `json:"idempotency_key"`
	PayloadVersion string                 `json:"payload_version"`
	Payload        ProyectoPlanBorradorV0 `json:"payload"`
}

type GuardarProyectoBorradorPersistenceAcceptedV0 struct {
	Status         string `json:"status"`
	ProyectoID     string `json:"proyecto_id"`
	IdempotencyKey string `json:"idempotency_key"`
	PayloadVersion string `json:"payload_version"`
	CreatedAt      string `json:"created_at,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`
}

type PersistenceRepositoryErrorV0 struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	Field         string `json:"field,omitempty"`
	Retryable     bool   `json:"retryable"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

func (err PersistenceRepositoryErrorV0) Error() string {
	return err.Code
}

func NewGuardarProyectoBorradorPersistenceRequestV0(cmd RegistrarProyectoDesdeAppSpecCommandV0, accepted RegistroProyectoAceptadoV0) GuardarProyectoBorradorPersistenceRequestV0 {
	plan := accepted.ProyectoPlanBorrador
	return GuardarProyectoBorradorPersistenceRequestV0{
		SchemaVersion:  PersistenceRepositoryContractVersionV0,
		Contrato:       PersistenceRepositoryContractVersionV0,
		Operacion:      PersistenceRepositoryOperationGuardarProyectoBorradorV0,
		RequestID:      firstNonEmptyV0(cmd.RequestID, plan.RequestID, cmd.AppSpec.RequestID),
		CorrelationID:  strings.TrimSpace(cmd.CorrelationID),
		IdempotencyKey: strings.TrimSpace(cmd.IdempotencyKey),
		PayloadVersion: ProyectoPlanBorradorPayloadVersionV0,
		Payload:        plan,
	}
}
