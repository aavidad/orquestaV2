package fabricaapp

import (
	"strings"
	"testing"

	"orquesta/db"
)

type fakeMaterializeStore struct {
	nextID      int64
	created     []*db.Tarea
	backlogIDs  []int64
	contractIDs []int64
}

func (f *fakeMaterializeStore) CreateTask(t *db.Tarea) (int64, error) {
	f.nextID++
	cloned := *t
	f.created = append(f.created, &cloned)
	return f.nextID, nil
}

func (f *fakeMaterializeStore) MoveTaskToBacklog(id int64) error {
	f.backlogIDs = append(f.backlogIDs, id)
	return nil
}

func (f *fakeMaterializeStore) DefineContract(id int64, actor string) error {
	f.contractIDs = append(f.contractIDs, id)
	return nil
}

func TestMaterializeResolvesDependenciesAndBacklog(t *testing.T) {
	store := &fakeMaterializeStore{}
	plan := GenerationResult{
		Tasks: []BlueprintTask{
			{Key: "a", Titulo: "Arquitectura", Modulo: "arquitectura", Prioridad: db.PrioridadAlta, ContratoDefinido: true},
			{Key: "b", Titulo: "Frontend", Modulo: "frontend", Prioridad: db.PrioridadAlta, Dependencias: []string{"a"}},
		},
	}

	result, err := Materialize(store, 42, "Codex1", plan)
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if result.Created != 2 || result.Backlog != 1 {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	if len(store.created) != 2 {
		t.Fatalf("creadas inesperadas: %+v", store.created)
	}
	if got := store.created[1].Dependencias; len(got) != 1 || got[0] != 1 {
		t.Fatalf("dependencias no resueltas: %+v", got)
	}
	if len(store.backlogIDs) != 1 || store.backlogIDs[0] != 2 {
		t.Fatalf("backlog inesperado: %+v", store.backlogIDs)
	}
	if len(store.contractIDs) != 1 || store.contractIDs[0] != 1 {
		t.Fatalf("contratos inesperados: %+v", store.contractIDs)
	}
}

func TestMaterializeFailsOnUnknownDependency(t *testing.T) {
	store := &fakeMaterializeStore{}
	_, err := Materialize(store, 42, "Codex1", GenerationResult{
		Tasks: []BlueprintTask{
			{Key: "b", Titulo: "Frontend", Modulo: "frontend", Prioridad: db.PrioridadAlta, Dependencias: []string{"missing"}},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "dependencia blueprint no resuelta") {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestMaterializeValidatesInputs(t *testing.T) {
	if _, err := Materialize(nil, 42, "Codex1", GenerationResult{}); err == nil {
		t.Fatalf("se esperaba error por store nil")
	}
	if _, err := Materialize(&fakeMaterializeStore{}, 0, "Codex1", GenerationResult{}); err == nil {
		t.Fatalf("se esperaba error por projectID invalido")
	}
}
