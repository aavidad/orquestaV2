/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// setupRefineriaFixture crea la infraestructura mínima (proyecto + tarea en progreso)
// y devuelve el ID de la tarea lista para ser enviada a la Refinería.
func setupRefineriaFixture(t *testing.T) (tareaID int64, agente string) {
	t.Helper()
	tmp := t.TempDir()
	_ = tmp // prepararDBTemporal ya usa TempDir internamente
	prepararDBTemporal(t)

	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "test-refineria",
		Nombre:  "Test Refinería",
		RutaAbs: filepath.Join(t.TempDir(), "test-refineria"),
		Tipo:    ProyectoRepo,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}

	agente = "claude"
	tareaID, err = CrearTarea(&Tarea{
		Titulo:    "Tarea de prueba refinería",
		ProyectoID: &proyectoID,
		Prioridad:  PrioridadAlta,
		CreadoPor: agente,
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	if err := TomarTarea(tareaID, agente); err != nil {
		t.Fatalf("TomarTarea: %v", err)
	}
	if err := IniciarTarea(tareaID, agente); err != nil {
		t.Fatalf("IniciarTarea: %v", err)
	}
	return tareaID, agente
}

func TestSolicitarRefineriaOK(t *testing.T) {
	tareaID, agente := setupRefineriaFixture(t)

	s, err := SolicitarRefineria(tareaID, agente, "feature/test", "/tmp", "go test ./...")
	if err != nil {
		t.Fatalf("SolicitarRefineria: %v", err)
	}
	if s.ID == 0 {
		t.Fatal("se esperaba ID > 0")
	}
	if s.TareaID != tareaID {
		t.Fatalf("tarea_id esperado %d, got %d", tareaID, s.TareaID)
	}
	if s.Estado != "pendiente" {
		t.Fatalf("estado esperado 'pendiente', got %s", s.Estado)
	}
	if s.CmdTest != "go test ./..." {
		t.Fatalf("cmd_test inesperado: %s", s.CmdTest)
	}
}

func TestSolicitarRefineriaUsaDefaultCmdTest(t *testing.T) {
	tareaID, agente := setupRefineriaFixture(t)

	s, err := SolicitarRefineria(tareaID, agente, "", "", "")
	if err != nil {
		t.Fatalf("SolicitarRefineria: %v", err)
	}
	if s.CmdTest != "go test ./..." {
		t.Fatalf("cmd_test por defecto esperado, got %s", s.CmdTest)
	}
}

func TestSolicitarRefineriaFallaSiTareaNoEnProgreso(t *testing.T) {
	prepararDBTemporal(t)

	tareaID, err := CrearTarea(&Tarea{
		Titulo:    "Tarea libre",
		Prioridad: PrioridadMedia,
		CreadoPor: "claude",
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}

	_, err = SolicitarRefineria(tareaID, "claude", "", "", "")
	if err == nil {
		t.Fatal("se esperaba error para tarea en estado 'libre'")
	}
}

func TestSolicitarRefineriaFallaTareaInexistente(t *testing.T) {
	prepararDBTemporal(t)

	_, err := SolicitarRefineria(99999, "claude", "", "", "")
	if err == nil {
		t.Fatal("se esperaba error para tarea inexistente")
	}
}

func TestListarRefineriasPendientes(t *testing.T) {
	tareaID, agente := setupRefineriaFixture(t)

	// Sin solicitudes todavía
	ss, err := ListarRefineriasPendientes()
	if err != nil {
		t.Fatalf("ListarRefineriasPendientes: %v", err)
	}
	if len(ss) != 0 {
		t.Fatalf("esperaba 0 solicitudes, got %d", len(ss))
	}

	// Crear una solicitud
	if _, err := SolicitarRefineria(tareaID, agente, "main", "", ""); err != nil {
		t.Fatalf("SolicitarRefineria: %v", err)
	}

	ss, err = ListarRefineriasPendientes()
	if err != nil {
		t.Fatalf("ListarRefineriasPendientes tras crear: %v", err)
	}
	if len(ss) != 1 {
		t.Fatalf("esperaba 1 solicitud, got %d", len(ss))
	}
	if ss[0].TareaID != tareaID {
		t.Fatalf("tarea_id inesperado: %d", ss[0].TareaID)
	}
}

func TestCancelarRefineriaOK(t *testing.T) {
	tareaID, agente := setupRefineriaFixture(t)

	s, err := SolicitarRefineria(tareaID, agente, "", "", "")
	if err != nil {
		t.Fatalf("SolicitarRefineria: %v", err)
	}

	if err := CancelarRefineria(s.ID, agente); err != nil {
		t.Fatalf("CancelarRefineria: %v", err)
	}

	got, err := GetRefineriaSolicitud(s.ID)
	if err != nil {
		t.Fatalf("GetRefineriaSolicitud: %v", err)
	}
	if got.Estado != "cancelada" {
		t.Fatalf("estado esperado 'cancelada', got %s", got.Estado)
	}
}

func TestCancelarRefineriaFallaYaCancelada(t *testing.T) {
	tareaID, agente := setupRefineriaFixture(t)

	s, _ := SolicitarRefineria(tareaID, agente, "", "", "")
	_ = CancelarRefineria(s.ID, agente)

	// Segunda cancelación debe fallar
	if err := CancelarRefineria(s.ID, agente); err == nil {
		t.Fatal("se esperaba error al cancelar una solicitud ya cancelada")
	}
}

func TestListarRefineriaPorTarea(t *testing.T) {
	tareaID, agente := setupRefineriaFixture(t)

	// Dos solicitudes para la misma tarea
	s1, _ := SolicitarRefineria(tareaID, agente, "rama-1", "", "")
	_ = CancelarRefineria(s1.ID, agente) // cancelar la primera para poder crear la segunda
	_, _ = SolicitarRefineria(tareaID, agente, "rama-2", "", "")

	ss, err := ListarRefineriaPorTarea(tareaID)
	if err != nil {
		t.Fatalf("ListarRefineriaPorTarea: %v", err)
	}
	if len(ss) != 2 {
		t.Fatalf("esperaba 2 solicitudes, got %d", len(ss))
	}
}

func TestGetRefineriaSolicitudNoExiste(t *testing.T) {
	prepararDBTemporal(t)

	_, err := GetRefineriaSolicitud(99999)
	if err != sql.ErrNoRows {
		t.Fatalf("esperaba sql.ErrNoRows, got %v", err)
	}
}
