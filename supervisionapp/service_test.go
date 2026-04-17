package supervisionapp

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

type fakeStore struct {
	projects        map[string]*db.Proyecto
	autonomyByID    map[int64]*Policy
	cyclesByProject map[int64][]*Cycle
	lastEnabled     *bool
	lastCycleFilter CycleFilter
	lastUpsert      *Policy
	lastCycle       *Cycle
	lastSupervised  struct {
		projectID int64
		when      time.Time
	}
	lastReviewed struct {
		projectID int64
		when      time.Time
	}
	nextCycleID int64
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		projects:        map[string]*db.Proyecto{},
		autonomyByID:    map[int64]*Policy{},
		cyclesByProject: map[int64][]*Cycle{},
		nextCycleID:     100,
	}
}

func (f *fakeStore) GetProject(ref string) (*db.Proyecto, error) {
	proyecto, ok := f.projects[ref]
	if !ok {
		return nil, fmt.Errorf("proyecto no encontrado: %s", ref)
	}
	return proyecto, nil
}

func (f *fakeStore) GetProjectAutonomy(proyectoID int64) (*Policy, error) {
	item, ok := f.autonomyByID[proyectoID]
	if !ok {
		return nil, nil
	}
	copia := *item
	return &copia, nil
}

func (f *fakeStore) ListProjectAutonomy(enabled *bool) ([]*Policy, error) {
	f.lastEnabled = enabled
	out := make([]*Policy, 0, len(f.autonomyByID))
	for _, item := range f.autonomyByID {
		if enabled != nil && item.Enabled != *enabled {
			continue
		}
		copia := *item
		out = append(out, &copia)
	}
	return out, nil
}

func (f *fakeStore) UpsertProjectAutonomy(item *Policy) (int64, error) {
	copia := *item
	f.lastUpsert = &copia
	f.autonomyByID[item.ProyectoID] = &copia
	return item.ProyectoID, nil
}

func (f *fakeStore) RegisterAutonomyCycle(item *Cycle) (int64, error) {
	copia := *item
	copia.ID = f.nextCycleID
	f.nextCycleID++
	f.lastCycle = &copia
	f.cyclesByProject[item.ProyectoID] = append([]*Cycle{&copia}, f.cyclesByProject[item.ProyectoID]...)
	return copia.ID, nil
}

func (f *fakeStore) ListAutonomyCycles(filter CycleFilter) ([]*Cycle, error) {
	f.lastCycleFilter = filter
	if filter.ProyectoID == nil {
		return nil, nil
	}
	source := f.cyclesByProject[*filter.ProyectoID]
	limit := filter.Limit
	if limit <= 0 || limit > len(source) {
		limit = len(source)
	}
	out := make([]*Cycle, 0, limit)
	for _, item := range source {
		if filter.Kind != nil && item.Kind != *filter.Kind {
			continue
		}
		copia := *item
		out = append(out, &copia)
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

func (f *fakeStore) MarkProjectAutonomySupervised(proyectoID int64, when time.Time) error {
	f.lastSupervised.projectID = proyectoID
	f.lastSupervised.when = when
	return nil
}

func (f *fakeStore) MarkProjectAutonomyReviewed(proyectoID int64, when time.Time) error {
	f.lastReviewed.projectID = proyectoID
	f.lastReviewed.when = when
	return nil
}

func TestGetProjectPolicyResuelveProyectoPorRef(t *testing.T) {
	store := newFakeStore()
	store.projects["demo"] = &db.Proyecto{ID: 7, Nombre: "demo"}
	store.autonomyByID[7] = &Policy{ProyectoID: 7, Enabled: true}

	svc := NewService(store)
	got, err := svc.GetProjectPolicy("  demo  ")
	if err != nil {
		t.Fatalf("GetProjectPolicy error: %v", err)
	}
	if got == nil || got.ProyectoID != 7 {
		t.Fatalf("GetProjectPolicy = %#v, want proyecto_id 7", got)
	}
}

func TestListEnabledPoliciesPideSoloHabilitadas(t *testing.T) {
	store := newFakeStore()
	store.autonomyByID[1] = &Policy{ProyectoID: 1, Enabled: true}
	store.autonomyByID[2] = &Policy{ProyectoID: 2, Enabled: false}

	svc := NewService(store)
	got, err := svc.ListEnabledPolicies()
	if err != nil {
		t.Fatalf("ListEnabledPolicies error: %v", err)
	}
	if store.lastEnabled == nil || !*store.lastEnabled {
		t.Fatalf("ListEnabledPolicies should request enabled=true, got %#v", store.lastEnabled)
	}
	if len(got) != 1 || got[0].ProyectoID != 1 {
		t.Fatalf("ListEnabledPolicies = %#v, want only enabled project", got)
	}
}

func TestPersistPolicyStateActualizaYValida(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)
	policy := &Policy{ProyectoID: 7, Enabled: true, EstadoAutonomia: AutonomiaProyectoActiva}

	if err := svc.PersistPolicyState(policy, AutonomiaProyectoCerrando); err != nil {
		t.Fatalf("PersistPolicyState error: %v", err)
	}
	if store.lastUpsert == nil {
		t.Fatal("PersistPolicyState debería persistir la política")
	}
	if store.lastUpsert.EstadoAutonomia != AutonomiaProyectoCerrando {
		t.Fatalf("EstadoAutonomia persistido = %q, want %q", store.lastUpsert.EstadoAutonomia, AutonomiaProyectoCerrando)
	}
	if policy.EstadoAutonomia != AutonomiaProyectoCerrando {
		t.Fatalf("EstadoAutonomia en memoria = %q, want %q", policy.EstadoAutonomia, AutonomiaProyectoCerrando)
	}

	if err := svc.PersistPolicyState(policy, "activa"); err == nil {
		t.Fatal("PersistPolicyState debería rechazar estados inválidos")
	}
}

func TestUpsertProjectPolicyNormalizaYRefresca(t *testing.T) {
	store := newFakeStore()
	store.projects["demo"] = &db.Proyecto{ID: 9, Nombre: "demo"}

	svc := NewService(store)
	got, err := svc.UpsertProjectPolicy("demo", PolicyInput{
		Enabled:              true,
		ObjetivoGeneral:      "  cerrar la app  ",
		DefinitionOfDoneJSON: "  {\"done\":true}  ",
		MaxWorkers:           3,
		ReserveReviewer:      true,
		ReserveSupervisor:    true,
		ReviewRequired:       true,
		AutoCreateTasks:      true,
		AutoCloseProject:     true,
		EstadoAutonomia:      AutonomiaProyectoActiva,
	})
	if err != nil {
		t.Fatalf("UpsertProjectPolicy error: %v", err)
	}
	if store.lastUpsert == nil {
		t.Fatalf("UpsertProjectPolicy should call store.UpsertProjectAutonomy")
	}
	if store.lastUpsert.ObjetivoGeneral != "cerrar la app" {
		t.Fatalf("ObjetivoGeneral = %q, want trimmed", store.lastUpsert.ObjetivoGeneral)
	}
	if store.lastUpsert.DefinitionOfDoneJSON != "{\"done\":true}" {
		t.Fatalf("DefinitionOfDoneJSON = %q, want trimmed", store.lastUpsert.DefinitionOfDoneJSON)
	}
	if got == nil || got.ProyectoID != 9 || !got.Enabled {
		t.Fatalf("UpsertProjectPolicy = %#v, want refreshed stored policy", got)
	}
	if store.lastUpsert.EstadoAutonomia != AutonomiaProyectoActiva {
		t.Fatalf("EstadoAutonomia = %q, want %q", store.lastUpsert.EstadoAutonomia, AutonomiaProyectoActiva)
	}
}

func TestUpsertProjectPolicyRechazaEstadoAutonomiaInvalido(t *testing.T) {
	store := newFakeStore()
	store.projects["demo"] = &db.Proyecto{ID: 9, Nombre: "demo"}

	svc := NewService(store)
	_, err := svc.UpsertProjectPolicy("demo", PolicyInput{
		Enabled:          true,
		EstadoAutonomia:  "activa",
		ObjetivoGeneral:  "cerrar la app",
		AutoCreateTasks:  true,
		AutoCloseProject: true,
	})
	if err == nil {
		t.Fatal("UpsertProjectPolicy debería rechazar estado_autonomia inválido")
	}
	if !strings.Contains(err.Error(), "estado_autonomia inválido") {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestRegisterCycleNormalizaYDevuelveCicloPersistido(t *testing.T) {
	store := newFakeStore()
	store.projects["demo"] = &db.Proyecto{ID: 11, Nombre: "demo"}

	svc := NewService(store)
	got, err := svc.RegisterCycle("demo", CycleInput{
		Kind:         "  supervision  ",
		Agente:       "  Codex3  ",
		InputJSON:    "  {\"step\":1}  ",
		DecisionJSON: "  {\"action\":\"continue\"}  ",
		Resultado:    "  ok  ",
	})
	if err != nil {
		t.Fatalf("RegisterCycle error: %v", err)
	}
	if store.lastCycle == nil {
		t.Fatalf("RegisterCycle should call store.RegisterAutonomyCycle")
	}
	if store.lastCycle.Kind != "supervision" || store.lastCycle.Agente != "Codex3" {
		t.Fatalf("RegisterCycle normalized cycle = %#v", store.lastCycle)
	}
	if got == nil || got.ID == 0 || got.Kind != "supervision" {
		t.Fatalf("RegisterCycle = %#v, want persisted cycle with id", got)
	}
	if store.lastCycleFilter.ProyectoID == nil || *store.lastCycleFilter.ProyectoID != 11 || store.lastCycleFilter.Limit != 1 {
		t.Fatalf("RegisterCycle filter = %#v, want project 11 and limit 1", store.lastCycleFilter)
	}
}

func TestRegisterCycleExigeKind(t *testing.T) {
	store := newFakeStore()
	store.projects["demo"] = &db.Proyecto{ID: 11, Nombre: "demo"}

	svc := NewService(store)
	if _, err := svc.RegisterCycle("demo", CycleInput{}); err == nil {
		t.Fatal("RegisterCycle should fail when kind is empty")
	}
}

func TestListCyclesYMarcasResuelvenProyecto(t *testing.T) {
	store := newFakeStore()
	store.projects["demo"] = &db.Proyecto{ID: 13, Nombre: "demo"}
	store.cyclesByProject[13] = []*Cycle{
		{ID: 2, ProyectoID: 13, Kind: "review"},
		{ID: 1, ProyectoID: 13, Kind: "supervision"},
	}
	svc := NewService(store)

	kind := "review"
	ciclos, err := svc.ListCycles("demo", &kind, 5)
	if err != nil {
		t.Fatalf("ListCycles error: %v", err)
	}
	if len(ciclos) != 1 || ciclos[0].Kind != "review" {
		t.Fatalf("ListCycles = %#v, want filtered review cycle", ciclos)
	}

	when := time.Now().UTC().Truncate(time.Second)
	if err := svc.MarkSupervised("demo", when); err != nil {
		t.Fatalf("MarkSupervised error: %v", err)
	}
	if err := svc.MarkReviewed("demo", when); err != nil {
		t.Fatalf("MarkReviewed error: %v", err)
	}
	if store.lastSupervised.projectID != 13 || !store.lastSupervised.when.Equal(when) {
		t.Fatalf("MarkSupervised = %#v, want project 13 at %s", store.lastSupervised, when)
	}
	if store.lastReviewed.projectID != 13 || !store.lastReviewed.when.Equal(when) {
		t.Fatalf("MarkReviewed = %#v, want project 13 at %s", store.lastReviewed, when)
	}
}
