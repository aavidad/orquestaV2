package orquestapersistence

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryPersistenceRepositoryContractAdapterV0GuardaBorrador(t *testing.T) {
	adapter := NewInMemoryPersistenceRepositoryContractAdapterV0(fixedTimeV0)
	request := validRequestV0(t)

	response, issues := adapter.GuardarProyectoBorrador(context.Background(), request)
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if response.Status != PersistenceEstadoPersistidoV0 {
		t.Fatalf("status=%q", response.Status)
	}
	if response.ProyectoID == "" {
		t.Fatal("proyecto_id vacio")
	}
	if response.IdempotencyKey != request.IdempotencyKey {
		t.Fatalf("idempotency_key=%q", response.IdempotencyKey)
	}
	if response.CreatedAt != "2026-05-04T10:00:00Z" || response.UpdatedAt != response.CreatedAt {
		t.Fatalf("timestamps invalidos: %+v", response)
	}
}

func TestInMemoryPersistenceRepositoryContractAdapterV0EsIdempotenteConPayloadEquivalente(t *testing.T) {
	adapter := NewInMemoryPersistenceRepositoryContractAdapterV0(fixedTimeV0)
	request := validRequestV0(t)

	first, issues := adapter.GuardarProyectoBorrador(context.Background(), request)
	if len(issues) != 0 {
		t.Fatalf("first issues=%+v", issues)
	}
	second, issues := adapter.GuardarProyectoBorrador(context.Background(), request)
	if len(issues) != 0 {
		t.Fatalf("second issues=%+v", issues)
	}
	if first != second {
		t.Fatalf("responses differ: first=%+v second=%+v", first, second)
	}
}

func TestInMemoryPersistenceRepositoryContractAdapterV0DetectaConflictoIdempotencia(t *testing.T) {
	adapter := NewInMemoryPersistenceRepositoryContractAdapterV0(fixedTimeV0)
	request := validRequestV0(t)
	if _, issues := adapter.GuardarProyectoBorrador(context.Background(), request); len(issues) != 0 {
		t.Fatalf("initial issues=%+v", issues)
	}
	request.Payload.Nombre = request.Payload.Nombre + " cambiado"

	_, issues := adapter.GuardarProyectoBorrador(context.Background(), request)
	if !HasPersistenceRepositoryIssueV0(issues, ErrConflictoIdempotenciaV0, "idempotency_key") {
		t.Fatalf("expected idempotency conflict, got %+v", issues)
	}
}

func TestInMemoryPersistenceRepositoryContractAdapterV0RechazaRequestInvalido(t *testing.T) {
	adapter := NewInMemoryPersistenceRepositoryContractAdapterV0(fixedTimeV0)
	request := validRequestV0(t)
	request.IdempotencyKey = ""

	_, issues := adapter.GuardarProyectoBorrador(context.Background(), request)
	if !HasPersistenceRepositoryIssueV0(issues, ErrPayloadInvalidoV0, "idempotency_key") {
		t.Fatalf("expected idempotency_key issue, got %+v", issues)
	}
}

func fixedTimeV0() time.Time {
	return time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
}
