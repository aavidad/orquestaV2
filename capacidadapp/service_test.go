package capacidadapp

import (
	"testing"

	"orquesta/db"
)

type fakeStore struct{}

type fakePoolLocalProvider struct {
	telemetria *TelemetriaPoolLocal
}

func (f fakePoolLocalProvider) DescribirPoolLocalCompartido(slug string) (*TelemetriaPoolLocal, error) {
	if f.telemetria == nil || f.telemetria.PoolSlug != slug {
		return nil, nil
	}
	copia := *f.telemetria
	return &copia, nil
}

type fakeStorePoolLocal struct {
	pool            *db.PoolCapacidad
	modelos         []*db.PoolModelo
	politicas       []*db.PoliticaModelo
	savePoolCalls   []*db.PoolCapacidad
	saveModelCalls  []string
	savePolicyCalls []*db.PoliticaModelo
}

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

func (f *fakeStorePoolLocal) ListPoolsSummary(activo *bool) ([]*db.PoolCapacidadResumen, error) {
	if f.pool == nil {
		return nil, nil
	}
	return []*db.PoolCapacidadResumen{{
		Pool:                f.pool,
		SesionesActivas:     0,
		CapacidadDisponible: f.pool.CapacidadTotal - f.pool.CapacidadReservada,
	}}, nil
}

func (f *fakeStorePoolLocal) GetPool(slug string) (*db.PoolCapacidad, error) {
	if f.pool == nil || f.pool.Slug != slug {
		return nil, nil
	}
	return f.pool, nil
}

func (f *fakeStorePoolLocal) ListPoolModels(slug string) ([]*db.PoolModelo, error) {
	if f.pool == nil || f.pool.Slug != slug {
		return nil, nil
	}
	return f.modelos, nil
}

func (f *fakeStorePoolLocal) SavePool(pool *db.PoolCapacidad) (int64, error) {
	copia := *pool
	f.savePoolCalls = append(f.savePoolCalls, &copia)
	f.pool = &copia
	f.pool.ID = 7
	return 7, nil
}

func (f *fakeStorePoolLocal) SavePoolModel(poolSlug string, modelo *db.PoolModelo) (int64, error) {
	f.saveModelCalls = append(f.saveModelCalls, poolSlug)
	copia := *modelo
	f.modelos = []*db.PoolModelo{&copia}
	return 8, nil
}

func (f *fakeStorePoolLocal) SeedInitialModels() error { return nil }

func (f *fakeStorePoolLocal) ListModelPolicies(scopeTipo, scopeRef string, activa *bool) ([]*db.PoliticaModelo, error) {
	var out []*db.PoliticaModelo
	for _, item := range f.politicas {
		if item == nil {
			continue
		}
		if item.ScopeTipo == scopeTipo && item.ScopeRef == scopeRef {
			out = append(out, item)
		}
	}
	return out, nil
}

func (f *fakeStorePoolLocal) SaveModelPolicy(policy *db.PoliticaModelo) (int64, error) {
	copia := *policy
	f.savePolicyCalls = append(f.savePolicyCalls, &copia)
	f.politicas = append(f.politicas, &copia)
	return int64(len(f.savePolicyCalls)), nil
}

func (f *fakeStorePoolLocal) SeedInitialModelPolicies() error { return nil }

func (f *fakeStorePoolLocal) ResolveModelPolicy(input db.ResolverPoliticaInput) (*db.ResolucionModelo, error) {
	return nil, nil
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

func TestAsegurarPoolLocalCompartidoProvisionaGemmaConPerfilesCanonicos(t *testing.T) {
	store := &fakeStorePoolLocal{}
	service := NewService(store)

	pool, err := service.AsegurarPoolLocalCompartido(EntradaAsegurarPoolLocalCompartido{
		PoolSlug:         "ollama-gemma4",
		ModeloPreferente: "gemma4:26b",
		SlotsMaximos:     1,
	})
	if err != nil {
		t.Fatalf("AsegurarPoolLocalCompartido: %v", err)
	}
	if pool == nil {
		t.Fatalf("pool local nil")
	}
	if pool.PoolSlug != "ollama-gemma4" || pool.ModeloPreferente != "gemma4:26b" || pool.SlotsMaximos != 1 {
		t.Fatalf("pool local inesperado: %+v", pool)
	}
	if pool.ConectorCanonico != "ollama_pool_local" || pool.ConectorCompatibilidad != "ollama-cli" {
		t.Fatalf("conectores inesperados: %+v", pool)
	}
	if len(store.savePoolCalls) != 1 {
		t.Fatalf("se esperaba un save pool, got=%d", len(store.savePoolCalls))
	}
	if got := store.savePoolCalls[0].Runtime; got != "ollama" {
		t.Fatalf("runtime de pool inesperado: %s", got)
	}
	if len(store.savePolicyCalls) != 3 {
		t.Fatalf("se esperaban 3 politicas, got=%d", len(store.savePolicyCalls))
	}
	perfiles := []string{
		store.savePolicyCalls[0].PerfilTarea,
		store.savePolicyCalls[1].PerfilTarea,
		store.savePolicyCalls[2].PerfilTarea,
	}
	want := []string{"analisis", "implementacion", "revision"}
	for i := range want {
		if perfiles[i] != want[i] {
			t.Fatalf("perfiles inesperados: got=%v want=%v", perfiles, want)
		}
	}
}

func TestDescribirPoolLocalCompartidoLeeMetadataYPoliticas(t *testing.T) {
	store := &fakeStorePoolLocal{
		pool: &db.PoolCapacidad{
			ID:             9,
			Slug:           "ollama-gemma4",
			Proveedor:      "Ollama",
			Runtime:        "ollama",
			CapacidadTotal: 1,
			MetadataJSON:   `{"slots_maximos":1,"conector_canonico":"ollama_pool_local","conector_compatibilidad":"ollama-cli","experimental_compat":true,"modelo_preferente":"gemma4:26b"}`,
		},
		modelos: []*db.PoolModelo{
			{PoolID: 9, ModelSlug: "gemma4:26b", Activo: true},
		},
		politicas: []*db.PoliticaModelo{
			{ScopeTipo: "perfil", ScopeRef: "implementacion", PerfilTarea: "implementacion", PoolSlug: "ollama-gemma4", ModelSlug: "gemma4:26b", ReasoningEffort: "high", Prioridad: 10},
			{ScopeTipo: "perfil", ScopeRef: "revision", PerfilTarea: "revision", PoolSlug: "ollama-gemma4", ModelSlug: "gemma4:26b", ReasoningEffort: "high", Prioridad: 10},
			{ScopeTipo: "perfil", ScopeRef: "analisis", PerfilTarea: "analisis", PoolSlug: "ollama-gemma4", ModelSlug: "gemma4:26b", ReasoningEffort: "high", Prioridad: 10},
		},
	}
	service := NewService(store)
	service.SetPoolLocalProvider(fakePoolLocalProvider{telemetria: &TelemetriaPoolLocal{
		PoolSlug:               "ollama-gemma4",
		SlotsMaximos:           1,
		SlotsActivos:           1,
		SesionesLogicasActivas: 1,
		SesionesReady:          1,
	}})

	pool, err := service.DescribirPoolLocalCompartido("ollama-gemma4")
	if err != nil {
		t.Fatalf("DescribirPoolLocalCompartido: %v", err)
	}
	if pool == nil {
		t.Fatalf("pool local nil")
	}
	if pool.SlotsMaximos != 1 || pool.ModeloPreferente != "gemma4:26b" {
		t.Fatalf("metadata inesperada: %+v", pool)
	}
	if len(pool.Perfiles) != 3 {
		t.Fatalf("perfiles inesperados: %+v", pool.Perfiles)
	}
	if pool.Telemetria == nil || pool.Telemetria.SlotsActivos != 1 || pool.Telemetria.SesionesReady != 1 {
		t.Fatalf("telemetria inesperada: %+v", pool.Telemetria)
	}
}
