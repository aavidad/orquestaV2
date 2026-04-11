package capacidadapp

import (
	"testing"

	"orquesta/db"
)

type mockPhaseProvider struct {
	fase string
}

func (m *mockPhaseProvider) GetActivePhase(proyecto string) (string, error) {
	return m.fase, nil
}

type repositorioHexagonalFalso struct {
	ultimaEntrada db.ResolverPoliticaInput
	resolucion    *db.ResolucionModelo
}

func (r *repositorioHexagonalFalso) ListPoolsSummary(activo *bool) ([]*db.PoolCapacidadResumen, error) {
	return nil, nil
}

func (r *repositorioHexagonalFalso) GetPool(slug string) (*db.PoolCapacidad, error) {
	return nil, nil
}

func (r *repositorioHexagonalFalso) ListPoolModels(slug string) ([]*db.PoolModelo, error) {
	return nil, nil
}

func (r *repositorioHexagonalFalso) SavePool(pool *db.PoolCapacidad) (int64, error) {
	return 0, nil
}

func (r *repositorioHexagonalFalso) SavePoolModel(poolSlug string, modelo *db.PoolModelo) (int64, error) {
	return 0, nil
}

func (r *repositorioHexagonalFalso) SeedInitialModels() error {
	return nil
}

func (r *repositorioHexagonalFalso) ListModelPolicies(scopeTipo, scopeRef string, activa *bool) ([]*db.PoliticaModelo, error) {
	return nil, nil
}

func (r *repositorioHexagonalFalso) SaveModelPolicy(policy *db.PoliticaModelo) (int64, error) {
	return 0, nil
}

func (r *repositorioHexagonalFalso) SeedInitialModelPolicies() error {
	return nil
}

func (r *repositorioHexagonalFalso) ResolveModelPolicy(input db.ResolverPoliticaInput) (*db.ResolucionModelo, error) {
	r.ultimaEntrada = input
	return r.resolucion, nil
}

func TestHexagonalPhaseResolution(t *testing.T) {
	repo := &repositorioHexagonalFalso{
		resolucion: &db.ResolucionModelo{ModelSlug: "modelo-auditoria"},
	}
	service := NewService(repo)
	mock := &mockPhaseProvider{fase: "auditoria"}
	service.SetPhaseProvider(mock)

	input := db.ResolverPoliticaInput{
		ProyectoSlug: "multi-app",
		PerfilTarea:  "coding",
	}

	resolucion, err := service.ResolveModelPolicy(input)
	if err != nil {
		t.Fatalf("resolver política: %v", err)
	}

	if repo.ultimaEntrada.Fase != "auditoria" {
		t.Fatalf("la fase activa no se propagó al repositorio: %+v", repo.ultimaEntrada)
	}
	if resolucion == nil || resolucion.ModelSlug != "modelo-auditoria" {
		t.Fatalf("resolución inesperada: %+v", resolucion)
	}
}
