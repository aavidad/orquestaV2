package capacidadapp

import (
	"testing"

	"orquesta/db"
)

type fakeStore struct{}

func (fakeStore) ListPoolsSummary(activo *bool) ([]*db.PoolCapacidadResumen, error) {
	return []*db.PoolCapacidadResumen{{
		Pool:                &db.PoolCapacidad{ID: 1, Slug: "codex", CapacidadTotal: 4, CapacidadReservada: 1},
		SesionesActivas:     2,
		CapacidadDisponible: 1,
	}}, nil
}

func (fakeStore) GetPool(slug string) (*db.PoolCapacidad, error) {
	return &db.PoolCapacidad{ID: 1, Slug: slug, CapacidadTotal: 4, CapacidadReservada: 1}, nil
}

func (fakeStore) ListPoolModels(slug string) ([]*db.PoolModelo, error) {
	return []*db.PoolModelo{{PoolID: 1, ModelSlug: "gpt-5.4"}}, nil
}

func (fakeStore) SavePool(pool *db.PoolCapacidad) (int64, error) { return 1, nil }
func (fakeStore) SavePoolModel(poolSlug string, modelo *db.PoolModelo) (int64, error) {
	return 2, nil
}
func (fakeStore) SeedInitialModels() error { return nil }
func (fakeStore) ListModelPolicies(scopeTipo, scopeRef string, activa *bool) ([]*db.PoliticaModelo, error) {
	return []*db.PoliticaModelo{{ScopeTipo: "global", PerfilTarea: "*"}}, nil
}
func (fakeStore) SaveModelPolicy(policy *db.PoliticaModelo) (int64, error) { return 3, nil }
func (fakeStore) SeedInitialModelPolicies() error                          { return nil }
func (fakeStore) ResolveModelPolicy(input db.ResolverPoliticaInput) (*db.ResolucionModelo, error) {
	return &db.ResolucionModelo{PerfilTarea: "implementacion", PoolSlug: "codex", ModelSlug: "gpt-5.4"}, nil
}

func TestGetPoolDetail(t *testing.T) {
	service := NewService(fakeStore{})
	detail, err := service.GetPoolDetail("codex")
	if err != nil {
		t.Fatalf("GetPoolDetail: %v", err)
	}
	if detail.Pool.Slug != "codex" || detail.SesionesActivas != 2 || detail.CapacidadDisponible != 1 {
		t.Fatalf("detalle inesperado: %+v", detail)
	}
	if len(detail.Modelos) != 1 || detail.Modelos[0].ModelSlug != "gpt-5.4" {
		t.Fatalf("modelos inesperados: %+v", detail.Modelos)
	}
}
