package orquestapersistence

import (
	"context"
	"testing"
)

func TestInMemoryPersistenceUnitOfWorkV0CommitHaceVisibleEscritura(t *testing.T) {
	uow := NewInMemoryPersistenceUnitOfWorkV0(fixedTimeV0)
	tx := beginMemoryUOWV0(t, uow)
	request := validRequestV0(t)

	response, issues := uow.GuardarProyectoBorrador(context.Background(), tx, request)
	if len(issues) != 0 {
		t.Fatalf("guardar issues=%+v", issues)
	}
	if _, ok := uow.ObtenerProyectoBorradorPorIdempotencyKey(request.IdempotencyKey); ok {
		t.Fatalf("escritura visible antes de commit")
	}
	if issues := uow.Commit(context.Background(), tx); len(issues) != 0 {
		t.Fatalf("commit issues=%+v", issues)
	}

	persisted, ok := uow.ObtenerProyectoBorradorPorIdempotencyKey(request.IdempotencyKey)
	if !ok {
		t.Fatalf("registro no visible tras commit")
	}
	if persisted.ProyectoID != response.ProyectoID || persisted.Payload.Nombre != request.Payload.Nombre {
		t.Fatalf("persisted=%+v response=%+v request=%+v", persisted, response, request)
	}
}

func TestInMemoryPersistenceUnitOfWorkV0RollbackDescartaEscritura(t *testing.T) {
	uow := NewInMemoryPersistenceUnitOfWorkV0(fixedTimeV0)
	tx := beginMemoryUOWV0(t, uow)
	request := validRequestV0(t)
	if _, issues := uow.GuardarProyectoBorrador(context.Background(), tx, request); len(issues) != 0 {
		t.Fatalf("guardar issues=%+v", issues)
	}

	if issues := uow.Rollback(context.Background(), tx); len(issues) != 0 {
		t.Fatalf("rollback issues=%+v", issues)
	}
	if _, ok := uow.ObtenerProyectoBorradorPorIdempotencyKey(request.IdempotencyKey); ok {
		t.Fatalf("registro visible tras rollback")
	}
}

func TestInMemoryPersistenceUnitOfWorkV0RollbackEsIdempotente(t *testing.T) {
	uow := NewInMemoryPersistenceUnitOfWorkV0(fixedTimeV0)
	tx := beginMemoryUOWV0(t, uow)

	if issues := uow.Rollback(context.Background(), tx); len(issues) != 0 {
		t.Fatalf("first rollback issues=%+v", issues)
	}
	if issues := uow.Rollback(context.Background(), tx); len(issues) != 0 {
		t.Fatalf("second rollback issues=%+v", issues)
	}
}

func TestInMemoryPersistenceUnitOfWorkV0CommitDespuesDeRollbackFalla(t *testing.T) {
	uow := NewInMemoryPersistenceUnitOfWorkV0(fixedTimeV0)
	tx := beginMemoryUOWV0(t, uow)
	if issues := uow.Rollback(context.Background(), tx); len(issues) != 0 {
		t.Fatalf("rollback issues=%+v", issues)
	}

	issues := uow.Commit(context.Background(), tx)
	if !HasPersistenceRepositoryIssueV0(issues, ErrTransaccionYaCerradaV0, "transaccion") {
		t.Fatalf("expected closed transaction issue, got %+v", issues)
	}
}

func TestInMemoryPersistenceUnitOfWorkV0RechazaEscrituraSinTransaccionActiva(t *testing.T) {
	uow := NewInMemoryPersistenceUnitOfWorkV0(fixedTimeV0)

	_, issues := uow.GuardarProyectoBorrador(context.Background(), TransaccionPersistenceV0{}, validRequestV0(t))
	if !HasPersistenceRepositoryIssueV0(issues, ErrTransaccionRequeridaV0, "transaccion") {
		t.Fatalf("expected transaction required, got %+v", issues)
	}
}

func TestInMemoryPersistenceUnitOfWorkV0RechazaAislamientoNoSoportado(t *testing.T) {
	uow := NewInMemoryPersistenceUnitOfWorkV0(fixedTimeV0)

	_, issues := uow.Begin(context.Background(), PersistenceUnitOfWorkOptionsV0{Isolation: "serializable"})
	if !HasPersistenceRepositoryIssueV0(issues, ErrAislamientoNoSoportadoV0, "isolation") {
		t.Fatalf("expected unsupported isolation, got %+v", issues)
	}
}

func beginMemoryUOWV0(t *testing.T, uow *InMemoryPersistenceUnitOfWorkV0) TransaccionPersistenceV0 {
	t.Helper()
	tx, issues := uow.Begin(context.Background(), PersistenceUnitOfWorkOptionsV0{
		ConnectorProfileRef: "connector-profile-test",
	})
	if len(issues) != 0 {
		t.Fatalf("begin issues=%+v", issues)
	}
	if tx.Estado != PersistenceTransaccionEstadoActivaV0 || tx.ID == "" {
		t.Fatalf("tx invalida=%+v", tx)
	}
	return tx
}
