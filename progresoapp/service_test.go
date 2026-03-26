package progresoapp

import (
	"testing"

	"orquesta/db"
)

type stubStore struct {
	summaryProject string
	summary        *db.ResumenProgresoProyecto
	phasesProject  string
	phases         []*db.FaseProyecto
	registerPhase  *db.FaseProyecto
	registerID     int64
	getPhaseID     int64
	phase          *db.FaseProyecto
	updatePhase    *db.FaseProyecto
	taskProgress   *db.AvanceTarea
}

func (s *stubStore) CalculateProjectSummary(proyecto string) (*db.ResumenProgresoProyecto, error) {
	s.summaryProject = proyecto
	return s.summary, nil
}

func (s *stubStore) ListProjectPhases(proyecto string) ([]*db.FaseProyecto, error) {
	s.phasesProject = proyecto
	return s.phases, nil
}

func (s *stubStore) RegisterProjectPhase(fase *db.FaseProyecto) (int64, error) {
	s.registerPhase = fase
	return s.registerID, nil
}

func (s *stubStore) GetProjectPhase(id int64) (*db.FaseProyecto, error) {
	s.getPhaseID = id
	return s.phase, nil
}

func (s *stubStore) UpdateProjectPhase(fase *db.FaseProyecto) error {
	s.updatePhase = fase
	return nil
}

func (s *stubStore) RegisterTaskProgress(avance *db.AvanceTarea) error {
	s.taskProgress = avance
	return nil
}

func TestServiceDelegatesSummaryAndPhases(t *testing.T) {
	store := &stubStore{
		summary: &db.ResumenProgresoProyecto{Proyecto: "orquestador"},
		phases:  []*db.FaseProyecto{{ID: 1, Proyecto: "orquestador", Nombre: "Analisis"}},
	}
	service := NewService(store)

	if _, err := service.GetSummary(" orquestador "); err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if store.summaryProject != "orquestador" {
		t.Fatalf("summaryProject=%q", store.summaryProject)
	}

	if _, err := service.ListPhases(" orquestador "); err != nil {
		t.Fatalf("ListPhases: %v", err)
	}
	if store.phasesProject != "orquestador" {
		t.Fatalf("phasesProject=%q", store.phasesProject)
	}
}

func TestRegisterAndUpdatePhase(t *testing.T) {
	proyecto := "orquestador"
	nombre := "QA"
	descripcion := "pruebas"
	estado := "activa"
	peso := 1.5
	orden := int64(30)
	store := &stubStore{
		registerID: 12,
		phase:      &db.FaseProyecto{ID: 12, Proyecto: proyecto, Nombre: nombre, Descripcion: descripcion, Orden: orden, Peso: peso, Estado: estado},
	}
	service := NewService(store)

	id, fase, err := service.RegisterPhase(RegisterPhaseInput{
		Proyecto:    " orquestador ",
		Nombre:      " QA ",
		Descripcion: " pruebas ",
		Orden:       orden,
		Peso:        peso,
		Estado:      " activa ",
	})
	if err != nil {
		t.Fatalf("RegisterPhase: %v", err)
	}
	if id != 12 || fase == nil || fase.ID != 12 {
		t.Fatalf("resultado inesperado id=%d fase=%+v", id, fase)
	}
	if store.registerPhase == nil || store.registerPhase.Proyecto != proyecto || store.registerPhase.Nombre != nombre || store.registerPhase.Estado != estado {
		t.Fatalf("registerPhase=%+v", store.registerPhase)
	}

	nombreNuevo := "Implementacion"
	if _, err := service.UpdatePhase(UpdatePhaseInput{ID: 12, Nombre: &nombreNuevo}); err != nil {
		t.Fatalf("UpdatePhase: %v", err)
	}
	if store.getPhaseID != 12 {
		t.Fatalf("getPhaseID=%d", store.getPhaseID)
	}
	if store.updatePhase == nil || store.updatePhase.Nombre != nombreNuevo {
		t.Fatalf("updatePhase=%+v", store.updatePhase)
	}
}

func TestRegisterTaskProgress(t *testing.T) {
	faseID := int64(7)
	store := &stubStore{}
	service := NewService(store)

	if err := service.RegisterTaskProgress(RegisterTaskProgressInput{
		TareaID:        42,
		Proyecto:       " orquestador ",
		FaseID:         &faseID,
		ProgresoPct:    55,
		ActualizadoPor: " Codex1 ",
	}); err != nil {
		t.Fatalf("RegisterTaskProgress: %v", err)
	}
	if store.taskProgress == nil || store.taskProgress.TareaID != 42 || store.taskProgress.Proyecto != "orquestador" || store.taskProgress.ActualizadoPor != "Codex1" {
		t.Fatalf("taskProgress=%+v", store.taskProgress)
	}
}
