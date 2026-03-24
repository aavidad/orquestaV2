package memoriaproyecto

import (
	"testing"
	"time"

	"orquesta/db"
)

type fakeStore struct {
	projectID int64
	project   *db.Proyecto
	decision  *db.DecisionProyecto
	documento *db.DocumentoExterno
}

func (f *fakeStore) ResolveProyectoIDBySlug(slug string) (*int64, error) {
	return &f.projectID, nil
}

func (f *fakeStore) ListProjects(activo *bool) ([]*db.Proyecto, error) {
	return []*db.Proyecto{f.project}, nil
}

func (f *fakeStore) GetProjectBySlug(slug string) (*db.Proyecto, error) {
	return f.project, nil
}

func (f *fakeStore) ListVoteHistoryByProjectID(proyectoID int64) ([]*db.HistorialVotacionProyecto, error) {
	return []*db.HistorialVotacionProyecto{{PropuestaID: proyectoID}}, nil
}

func (f *fakeStore) SaveDecision(d *db.DecisionProyecto) (int64, error) {
	f.decision = d
	return 77, nil
}

func (f *fakeStore) ListDecisionsByProjectID(proyectoID int64) ([]*db.DecisionProyecto, error) {
	return []*db.DecisionProyecto{{ProyectoID: proyectoID, Titulo: "bd"}}, nil
}

func (f *fakeStore) SaveExternalDoc(doc *db.DocumentoExterno) (int64, error) {
	f.documento = doc
	return 88, nil
}

func (f *fakeStore) ListExternalDocsByProjectID(proyectoID int64) ([]*db.DocumentoExterno, error) {
	return []*db.DocumentoExterno{{ProyectoID: proyectoID, Titulo: "ADR"}}, nil
}

func TestCreateDecisionResuelveProyectoYConstruyeEntidad(t *testing.T) {
	t.Parallel()

	store := &fakeStore{projectID: 9}
	svc := NewService(store)

	id, err := svc.CreateDecision(CreateDecisionInput{
		ProyectoSlug: "orquestador",
		Categoria:    "arquitectura",
		Titulo:       "Base de datos",
		Solucion:     "Puerto de almacenamiento",
		Motivo:       "Permitir SQLite y MySQL",
	})
	if err != nil {
		t.Fatalf("CreateDecision: %v", err)
	}
	if id != 77 {
		t.Fatalf("id inesperado: %d", id)
	}
	if store.decision == nil {
		t.Fatalf("decision no capturada")
	}
	if store.decision.ProyectoID != 9 {
		t.Fatalf("proyectoID inesperado: %d", store.decision.ProyectoID)
	}
	if store.decision.Titulo != "Base de datos" {
		t.Fatalf("titulo inesperado: %s", store.decision.Titulo)
	}
}

func TestCreateExternalDocResuelveProyectoYConstruyeEntidad(t *testing.T) {
	t.Parallel()

	store := &fakeStore{projectID: 12}
	svc := NewService(store)

	id, err := svc.CreateExternalDoc(CreateExternalDocInput{
		ProyectoSlug:  "orquestador",
		TipoDocumento: "markdown",
		Titulo:        "Diseno",
		RutaRef:       "/tmp/diseno.md",
		Resumen:       "Resumen corto",
	})
	if err != nil {
		t.Fatalf("CreateExternalDoc: %v", err)
	}
	if id != 88 {
		t.Fatalf("id inesperado: %d", id)
	}
	if store.documento == nil {
		t.Fatalf("documento no capturado")
	}
	if store.documento.ProyectoID != 12 {
		t.Fatalf("proyectoID inesperado: %d", store.documento.ProyectoID)
	}
	if store.documento.RutaRef != "/tmp/diseno.md" {
		t.Fatalf("ruta inesperada: %s", store.documento.RutaRef)
	}
}

func TestOverviewAgrupaBloquesPorProyecto(t *testing.T) {
	t.Parallel()

	now := time.Now()
	store := &fakeStore{
		projectID: 5,
		project: &db.Proyecto{
			ID:        5,
			Slug:      "orquestador",
			Nombre:    "Orquestador",
			RutaAbs:   "/tmp/orquestador",
			Tipo:      "repo",
			Activo:    true,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	svc := NewService(store)

	overview, err := svc.Overview("orquestador")
	if err != nil {
		t.Fatalf("Overview: %v", err)
	}
	if overview.Proyecto == nil || overview.Proyecto.Slug != "orquestador" {
		t.Fatalf("overview sin proyecto")
	}
	if len(overview.Votaciones) != 1 || len(overview.Decisiones) != 1 || len(overview.Documentos) != 1 {
		t.Fatalf("overview incompleto: %+v", overview)
	}
}
