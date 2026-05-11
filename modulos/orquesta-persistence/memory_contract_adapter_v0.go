package orquestapersistence

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"
)

// InMemoryPersistenceRepositoryContractAdapterV0 is a pure in-memory adapter for contract tests.
// It is not a production persistence adapter and does not model real persistence transactions.
type InMemoryPersistenceRepositoryContractAdapterV0 struct {
	mu      sync.Mutex
	now     func() time.Time
	nextID  int
	records map[string]inMemoryProyectoBorradorRecordV0
}

type inMemoryProyectoBorradorRecordV0 struct {
	payload  []byte
	response GuardarProyectoBorradorResponseV0
}

func NewInMemoryPersistenceRepositoryContractAdapterV0(now func() time.Time) *InMemoryPersistenceRepositoryContractAdapterV0 {
	if now == nil {
		now = time.Now
	}
	return &InMemoryPersistenceRepositoryContractAdapterV0{
		now:     now,
		records: make(map[string]inMemoryProyectoBorradorRecordV0),
	}
}

func (adapter *InMemoryPersistenceRepositoryContractAdapterV0) GuardarProyectoBorrador(
	ctx context.Context,
	request GuardarProyectoBorradorRequestV0,
) (GuardarProyectoBorradorResponseV0, []PersistenceRepositoryValidationIssueV0) {
	if err := ctx.Err(); err != nil {
		return GuardarProyectoBorradorResponseV0{}, []PersistenceRepositoryValidationIssueV0{
			issueV0(ErrPersistenciaNoDisponibleV0, "contexto", err.Error()),
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

	key := trimV0(request.IdempotencyKey)
	adapter.mu.Lock()
	defer adapter.mu.Unlock()

	if record, exists := adapter.records[key]; exists {
		if !bytes.Equal(record.payload, payload) {
			return GuardarProyectoBorradorResponseV0{}, []PersistenceRepositoryValidationIssueV0{
				issueV0(ErrConflictoIdempotenciaV0, "idempotency_key", "idempotency_key ya existe con payload distinto"),
			}
		}
		return record.response, nil
	}

	adapter.nextID++
	instant := adapter.now().UTC().Format(time.RFC3339Nano)
	response := GuardarProyectoBorradorResponseV0{
		Status:         PersistenceEstadoPersistidoV0,
		ProyectoID:     "mem-proyecto-" + strconv.FormatInt(int64(adapter.nextID), 10),
		IdempotencyKey: key,
		PayloadVersion: ProyectoPlanBorradorPayloadVersionV0,
		CreatedAt:      instant,
		UpdatedAt:      instant,
	}
	adapter.records[key] = inMemoryProyectoBorradorRecordV0{
		payload:  append([]byte(nil), payload...),
		response: response,
	}
	return response, nil
}
