package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestProcesarAgentesDegradadosAutonomiaBatchAutoasignaPremiumPausadoSinSesion(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoPremiumID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto premium: %v", err)
	}
	if err := db.UpsertProyectoOperacion(&db.ProyectoOperacion{
		ProyectoID:       proyectoPremiumID,
		EstadoOperativo:  db.ProyectoOperativoActivo,
		Motivo:           "microrefactor_loop",
		ObjetivoPct:      100,
		MinAgentes:       1,
		MaxAgentes:       3,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion premium: %v", err)
	}
	proyectoAPIID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "api",
		Nombre:  "API",
		RutaAbs: filepath.Join(tmp, "api"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto api: %v", err)
	}
	if err := db.ActivarAsignacion("Gemini1", proyectoPremiumID, "reactivacion_automatica_trabajo_activo"); err != nil {
		t.Fatalf("activar asignacion premium: %v", err)
	}
	if err := db.PausarAsignacion("Gemini1", proyectoPremiumID, "sin_trabajo_reactivacion_automatica"); err != nil {
		t.Fatalf("pausar asignacion premium: %v", err)
	}
	if err := db.ActivarAsignacion("Gemini1", proyectoAPIID, "reactivacion_automatica_trabajo_activo"); err != nil {
		t.Fatalf("activar asignacion api: %v", err)
	}
	if err := db.PausarAsignacion("Gemini1", proyectoAPIID, "sin_trabajo_espera_automatica"); err != nil {
		t.Fatalf("pausar asignacion api: %v", err)
	}
	if _, err := db.DB.Exec(`INSERT INTO sesiones (agente, proyecto_id, activa, inicio, herramienta, estado) VALUES (?, ?, 0, CURRENT_TIMESTAMP, 'gemini-cli', 'cerrada')`, "Gemini1", proyectoAPIID); err != nil {
		t.Fatalf("insertar sesion vieja: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Runtime mailbox/session_resume",
		Descripcion: "Write-set exclusivo: cmd/controlplane_support.go\nTests minimos: go test ./cmd -run 'TestProcesarRuntimeMailboxSessionResumeBatch.*' -count=1",
		ProyectoID:  &proyectoPremiumID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       "autonomia:microrefactor_loop",
	})
	if err != nil {
		t.Fatalf("crear tarea premium libre: %v", err)
	}

	n, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesarAgentesDegradadosAutonomiaBatch: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia procesar reactivacion/autoasignacion premium, got=%d", n)
	}

	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea == nil || tarea.Agente == nil || strings.TrimSpace(*tarea.Agente) != "Gemini1" || tarea.Estado != db.TareaEnProgreso {
		t.Fatalf("la tarea premium libre deberia quedar en progreso en Gemini1: %+v", tarea)
	}

	agente := "Gemini1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoPremiumID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("deberia encolar start premium: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"motivo":"premium_idle_autoassigned"`) {
		t.Fatalf("start sin motivo esperado: %s", orders[0].PayloadJSON)
	}
}

