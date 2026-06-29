package orquestapersistence

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"
)

const (
	PersistenceTransaccionEstadoConfirmadaV0 = "confirmada"
	PersistenceTransaccionEstadoRevertidaV0  = "revertida"
	PersistenceTransaccionEstadoFallidaV0    = "fallida"
	PersistenceIsolationDefaultV0            = "default"

	ErrTransaccionRequeridaV0   = "transaccion_requerida"
	ErrTransaccionNoActivaV0    = "transaccion_no_activa"
	ErrTransaccionYaCerradaV0   = "transaccion_ya_cerrada"
	ErrAislamientoNoSoportadoV0 = "aislamiento_no_soportado"
	ErrCommitFallidoV0          = "commit_fallido"
	ErrDuplicadoIncompatibleV0  = "duplicado_idempotente_incompatible"
)

type PersistenceUnitOfWorkOptionsV0 struct {
	ReadOnly            bool   `json:"read_only,omitempty"`
	Isolation           string `json:"isolation,omitempty"`
	ConnectorProfileRef string `json:"connector_profile_ref,omitempty"`
}

type TransaccionPersistenceV0 struct {
	ID                  string `json:"id"`
	ConnectorProfileRef string `json:"connector_profile_ref,omitempty"`
	ReadOnly            bool   `json:"read_only,omitempty"`
	Isolation           string `json:"isolation,omitempty"`
	Estado              string `json:"estado"`
	StartedAt           string `json:"started_at"`
}

type ProyectoPlanBorradorPersistidoV0 struct {
	ProyectoID      string                 `json:"proyecto_id"`
	IdempotencyKey  string                 `json:"idempotency_key"`
	VersionContrato string                 `json:"version_contrato"`
	Payload         ProyectoPlanBorradorV0 `json:"payload"`
	CreatedAt       string                 `json:"created_at"`
	UpdatedAt       string                 `json:"updated_at"`
}

type InMemoryPersistenceUnitOfWorkV0 struct {
	mu         sync.Mutex
	now        func() time.Time
	nextTx     int
	nextRecord int
	records    map[string]uowProyectoBorradorRecordV0
	txs        map[string]*uowTransactionStateV0
}

type uowProyectoBorradorRecordV0 struct {
	payload   []byte
	response  GuardarProyectoBorradorResponseV0
	persisted ProyectoPlanBorradorPersistidoV0
}

type uowTransactionStateV0 struct {
	tx      TransaccionPersistenceV0
	pending map[string]uowProyectoBorradorRecordV0
}

func NewInMemoryPersistenceUnitOfWorkV0(now func() time.Time) *InMemoryPersistenceUnitOfWorkV0 {
	if now == nil {
		now = time.Now
	}
	return &InMemoryPersistenceUnitOfWorkV0{
		now:     now,
		records: make(map[string]uowProyectoBorradorRecordV0),
		txs:     make(map[string]*uowTransactionStateV0),
	}
}

func (uow *InMemoryPersistenceUnitOfWorkV0) Begin(
	ctx context.Context,
	options PersistenceUnitOfWorkOptionsV0,
) (TransaccionPersistenceV0, []PersistenceRepositoryValidationIssueV0) {
	if err := ctx.Err(); err != nil {
		return TransaccionPersistenceV0{}, []PersistenceRepositoryValidationIssueV0{
			issueV0(ErrPersistenciaNoDisponibleV0, "contexto", err.Error()),
		}
	}
	isolation := trimV0(options.Isolation)
	if isolation == "" {
		isolation = PersistenceIsolationDefaultV0
	}
	if isolation != PersistenceIsolationDefaultV0 {
		return TransaccionPersistenceV0{}, []PersistenceRepositoryValidationIssueV0{
			issueV0(ErrAislamientoNoSoportadoV0, "isolation", "aislamiento no soportado por adaptador de contrato en memoria"),
		}
	}
	uow.mu.Lock()
	defer uow.mu.Unlock()

	uow.nextTx++
	tx := TransaccionPersistenceV0{
		ID:                  "mem-tx-" + strconv.FormatInt(int64(uow.nextTx), 10),
		ConnectorProfileRef: trimV0(options.ConnectorProfileRef),
		ReadOnly:            options.ReadOnly,
		Isolation:           isolation,
		Estado:              PersistenceTransaccionEstadoActivaV0,
		StartedAt:           uow.now().UTC().Format(time.RFC3339Nano),
	}
	uow.txs[tx.ID] = &uowTransactionStateV0{
		tx:      tx,
		pending: make(map[string]uowProyectoBorradorRecordV0),
	}
	return tx, nil
}

func (uow *InMemoryPersistenceUnitOfWorkV0) GuardarProyectoBorrador(
	ctx context.Context,
	tx TransaccionPersistenceV0,
	request GuardarProyectoBorradorRequestV0,
) (GuardarProyectoBorradorResponseV0, []PersistenceRepositoryValidationIssueV0) {
	if err := ctx.Err(); err != nil {
		return GuardarProyectoBorradorResponseV0{}, []PersistenceRepositoryValidationIssueV0{
			issueV0(ErrPersistenciaNoDisponibleV0, "contexto", err.Error()),
		}
	}
	if trimV0(tx.ID) == "" {
		return GuardarProyectoBorradorResponseV0{}, []PersistenceRepositoryValidationIssueV0{
			issueV0(ErrTransaccionRequeridaV0, "transaccion", "transaccion activa requerida"),
		}
	}
	if issues := ValidateGuardarProyectoBorradorRequestV0(request); len(issues) > 0 {
		return GuardarProyectoBorradorResponseV0{}, issues
	}
	payload, err := json.Marshal(request.Payload)
	if err != nil {
		return GuardarProyectoBorradorResponseV0{}, []PersistenceRepositoryValidationIssueV0{
			issueV0(ErrPayloadInvalidoV0, "payload", err.Error()),
		}
	}

	uow.mu.Lock()
	defer uow.mu.Unlock()

	state, issues := uow.activeTransactionStateV0(tx)
	if len(issues) > 0 {
		return GuardarProyectoBorradorResponseV0{}, issues
	}
	if state.tx.ReadOnly {
		return GuardarProyectoBorradorResponseV0{}, []PersistenceRepositoryValidationIssueV0{
			issueV0(ErrTransaccionNoActivaV0, "transaccion.read_only", "transaccion read_only no acepta escrituras"),
		}
	}

	key := trimV0(request.IdempotencyKey)
	if record, exists := state.pending[key]; exists {
		if !bytes.Equal(record.payload, payload) {
			return GuardarProyectoBorradorResponseV0{}, []PersistenceRepositoryValidationIssueV0{
				issueV0(ErrDuplicadoIncompatibleV0, "idempotency_key", "idempotency_key pendiente con payload distinto"),
			}
		}
		return record.response, nil
	}
	if record, exists := uow.records[key]; exists {
		if !bytes.Equal(record.payload, payload) {
			return GuardarProyectoBorradorResponseV0{}, []PersistenceRepositoryValidationIssueV0{
				issueV0(ErrDuplicadoIncompatibleV0, "idempotency_key", "idempotency_key persistida con payload distinto"),
			}
		}
		return record.response, nil
	}

	uow.nextRecord++
	instant := uow.now().UTC().Format(time.RFC3339Nano)
	response := GuardarProyectoBorradorResponseV0{
		Status:         PersistenceEstadoPersistidoV0,
		ProyectoID:     "mem-uow-proyecto-" + strconv.FormatInt(int64(uow.nextRecord), 10),
		IdempotencyKey: key,
		PayloadVersion: ProyectoPlanBorradorPayloadVersionV0,
		CreatedAt:      instant,
		UpdatedAt:      instant,
	}
	state.pending[key] = uowProyectoBorradorRecordV0{
		payload:  append([]byte(nil), payload...),
		response: response,
		persisted: ProyectoPlanBorradorPersistidoV0{
			ProyectoID:      response.ProyectoID,
			IdempotencyKey:  key,
			VersionContrato: ProyectoPlanBorradorPayloadVersionV0,
			Payload:         request.Payload,
			CreatedAt:       response.CreatedAt,
			UpdatedAt:       response.UpdatedAt,
		},
	}
	return response, nil
}

func (uow *InMemoryPersistenceUnitOfWorkV0) Commit(
	ctx context.Context,
	tx TransaccionPersistenceV0,
) []PersistenceRepositoryValidationIssueV0 {
	if err := ctx.Err(); err != nil {
		return []PersistenceRepositoryValidationIssueV0{
			issueV0(ErrPersistenciaNoDisponibleV0, "contexto", err.Error()),
		}
	}
	uow.mu.Lock()
	defer uow.mu.Unlock()

	state, issues := uow.activeTransactionStateV0(tx)
	if len(issues) > 0 {
		return issues
	}
	for key, pending := range state.pending {
		if record, exists := uow.records[key]; exists && !bytes.Equal(record.payload, pending.payload) {
			state.tx.Estado = PersistenceTransaccionEstadoFallidaV0
			return []PersistenceRepositoryValidationIssueV0{
				issueV0(ErrCommitFallidoV0, "idempotency_key", "conflicto idempotente durante commit"),
			}
		}
	}
	for key, pending := range state.pending {
		uow.records[key] = pending
	}
	state.tx.Estado = PersistenceTransaccionEstadoConfirmadaV0
	return nil
}

func (uow *InMemoryPersistenceUnitOfWorkV0) Rollback(
	ctx context.Context,
	tx TransaccionPersistenceV0,
) []PersistenceRepositoryValidationIssueV0 {
	if err := ctx.Err(); err != nil {
		return []PersistenceRepositoryValidationIssueV0{
			issueV0(ErrPersistenciaNoDisponibleV0, "contexto", err.Error()),
		}
	}
	if trimV0(tx.ID) == "" {
		return []PersistenceRepositoryValidationIssueV0{
			issueV0(ErrTransaccionRequeridaV0, "transaccion", "transaccion activa requerida"),
		}
	}
	uow.mu.Lock()
	defer uow.mu.Unlock()

	state, ok := uow.txs[tx.ID]
	if !ok {
		return []PersistenceRepositoryValidationIssueV0{
			issueV0(ErrTransaccionNoActivaV0, "transaccion", "transaccion desconocida"),
		}
	}
	if state.tx.Estado == PersistenceTransaccionEstadoConfirmadaV0 {
		return []PersistenceRepositoryValidationIssueV0{
			issueV0(ErrTransaccionYaCerradaV0, "transaccion", "transaccion ya confirmada"),
		}
	}
	state.tx.Estado = PersistenceTransaccionEstadoRevertidaV0
	state.pending = make(map[string]uowProyectoBorradorRecordV0)
	return nil
}

func (uow *InMemoryPersistenceUnitOfWorkV0) ObtenerProyectoBorradorPorIdempotencyKey(
	idempotencyKey string,
) (ProyectoPlanBorradorPersistidoV0, bool) {
	uow.mu.Lock()
	defer uow.mu.Unlock()

	record, ok := uow.records[trimV0(idempotencyKey)]
	if !ok {
		return ProyectoPlanBorradorPersistidoV0{}, false
	}
	return record.persisted, true
}

func (uow *InMemoryPersistenceUnitOfWorkV0) activeTransactionStateV0(
	tx TransaccionPersistenceV0,
) (*uowTransactionStateV0, []PersistenceRepositoryValidationIssueV0) {
	state, ok := uow.txs[tx.ID]
	if !ok {
		return nil, []PersistenceRepositoryValidationIssueV0{
			issueV0(ErrTransaccionNoActivaV0, "transaccion", "transaccion desconocida"),
		}
	}
	if state.tx.Estado != PersistenceTransaccionEstadoActivaV0 {
		return nil, []PersistenceRepositoryValidationIssueV0{
			issueV0(ErrTransaccionYaCerradaV0, "transaccion", "transaccion ya cerrada"),
		}
	}
	return state, nil
}
