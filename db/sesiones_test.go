package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIniciarSesionDevuelveIDPersistido(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		Close()
	}()

	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}

	if err := RegistrarAgente("codex-sesion", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}

	id, err := IniciarSesion("codex-sesion")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	if id == 0 {
		t.Fatalf("id de sesion no valido: %d", id)
	}

	var persistedID int64
	if err := DB.QueryRow(`
		SELECT id
		FROM sesiones
		WHERE agente = ? AND activa = 1
		ORDER BY id DESC
		LIMIT 1`, "codex-sesion").Scan(&persistedID); err != nil {
		t.Fatalf("select sesion activa: %v", err)
	}
	if persistedID != id {
		t.Fatalf("id devuelto %d distinto del persistido %d", id, persistedID)
	}
}

func TestRegistrarCodexUsaNombreCanonicoYRespetaExistentes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		Close()
	}()

	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente Codex1: %v", err)
	}
	if err := RegistrarAgente("codex2", "programador"); err != nil {
		t.Fatalf("RegistrarAgente codex2: %v", err)
	}

	nombre, err := RegistrarCodex()
	if err != nil {
		t.Fatalf("RegistrarCodex: %v", err)
	}
	if nombre != "Codex3" {
		t.Fatalf("nombre inesperado: %s", nombre)
	}
}

func TestRegistrarAgenteAutoUsaPrefijoCanonicoSegunProveedor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		Close()
	}()

	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := RegistrarAgente("Claude1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente Claude1: %v", err)
	}
	if err := RegistrarAgente("gemini2", "programador"); err != nil {
		t.Fatalf("RegistrarAgente gemini2: %v", err)
	}

	nombreClaude, err := RegistrarAgenteAuto("anthropic", "documentador")
	if err != nil {
		t.Fatalf("RegistrarAgenteAuto anthropic: %v", err)
	}
	if nombreClaude != "Claude2" {
		t.Fatalf("nombre Claude inesperado: %s", nombreClaude)
	}

	nombreGemini, err := RegistrarAgenteAuto("google", "programador")
	if err != nil {
		t.Fatalf("RegistrarAgenteAuto google: %v", err)
	}
	if nombreGemini != "Gemini3" {
		t.Fatalf("nombre Gemini inesperado: %s", nombreGemini)
	}
}

func TestEliminarAgenteBloqueaTareasActivas(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		Close()
	}()

	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := RegistrarAgente("Codex7", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}

	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Coordinar runtime order",
		Descripcion: "no borrar agente con trabajo vivo",
		Modulo:      "controlplane",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex7"); err != nil {
		t.Fatalf("TomarTarea: %v", err)
	}

	err = EliminarAgente("Codex7")
	if err == nil {
		t.Fatalf("esperaba bloqueo al eliminar agente con tareas activas")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "tarea") {
		t.Fatalf("error inesperado: %v", err)
	}
	if _, err := GetAgente("Codex7"); err != nil {
		t.Fatalf("el agente no deberia haberse borrado: %v", err)
	}
}

func TestRetirarAgentePausaAsignacionesYLiberaTrabajo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		Close()
	}()

	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := RegistrarAgente("CodexRetiro", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "demo-retiro",
		Nombre:  "Demo Retiro",
		RutaAbs: filepath.Join(dir, "demo-retiro"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if err := ActivarAsignacion("CodexRetiro", proyectoID, "frente_activo"); err != nil {
		t.Fatalf("ActivarAsignacion: %v", err)
	}
	if _, err := IniciarSesion("CodexRetiro"); err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}

	tareaAsignada, err := CrearTarea(&Tarea{
		Titulo:      "Tarea asignada",
		Descripcion: "debe volver al pool",
		ProyectoID:  &proyectoID,
		Modulo:      "backend",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea asignada: %v", err)
	}
	if err := TomarTarea(tareaAsignada, "CodexRetiro"); err != nil {
		t.Fatalf("TomarTarea asignada: %v", err)
	}

	tareaViva, err := CrearTarea(&Tarea{
		Titulo:      "Tarea viva",
		Descripcion: "debe volver al pool conservando nota",
		ProyectoID:  &proyectoID,
		Modulo:      "frontend",
		Prioridad:   PrioridadMedia,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea viva: %v", err)
	}
	if err := TomarTarea(tareaViva, "CodexRetiro"); err != nil {
		t.Fatalf("TomarTarea viva: %v", err)
	}
	if err := IniciarTarea(tareaViva, "CodexRetiro"); err != nil {
		t.Fatalf("IniciarTarea viva: %v", err)
	}

	if err := RetirarAgente("CodexRetiro"); err != nil {
		t.Fatalf("RetirarAgente: %v", err)
	}

	agente, err := GetAgente("CodexRetiro")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.Habilitado || agente.Activo {
		t.Fatalf("estado de agente inesperado: %+v", agente)
	}

	agenteNombre := "CodexRetiro"
	asignaciones, err := ListarAsignaciones(FiltroAsignaciones{Agente: &agenteNombre})
	if err != nil {
		t.Fatalf("ListarAsignaciones: %v", err)
	}
	if len(asignaciones) == 0 || asignaciones[0].Estado != AsignacionPausada {
		t.Fatalf("asignacion no pausada: %+v", asignaciones)
	}
	if !strings.Contains(asignaciones[0].Nota, "agente_retirado") {
		t.Fatalf("nota de asignacion inesperada: %+v", asignaciones[0])
	}

	for _, id := range []int64{tareaAsignada, tareaViva} {
		tarea, err := GetTarea(id)
		if err != nil {
			t.Fatalf("GetTarea %d: %v", id, err)
		}
		if tarea.Estado != TareaLibre {
			t.Fatalf("tarea %d no liberada: %+v", id, tarea)
		}
		if tarea.Agente != nil {
			t.Fatalf("tarea %d mantiene agente: %+v", id, tarea)
		}
		if !strings.Contains(strings.ToLower(tarea.Notas), "agente retirado automáticamente") {
			t.Fatalf("tarea %d sin anotacion de retiro: %+v", id, tarea)
		}
	}

	var activas int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM sesiones WHERE agente=? AND activa=1`, "CodexRetiro").Scan(&activas); err != nil {
		t.Fatalf("count sesiones activas: %v", err)
	}
	if activas != 0 {
		t.Fatalf("esperaba 0 sesiones activas, got=%d", activas)
	}
}

func TestListarAgentesAlineaEstadoVisibleConSesiones(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		Close()
	}()

	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	for _, agente := range []string{"CodexVisible1", "CodexVisible2"} {
		if err := RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", agente, err)
		}
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='pensando' WHERE nombre='CodexVisible1'`); err != nil {
		t.Fatalf("marcar estado visible1: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET activo=1, estado_sesion='disponible' WHERE nombre='CodexVisible2'`); err != nil {
		t.Fatalf("marcar CodexVisible2 activo: %v", err)
	}
	if _, err := DB.Exec(`INSERT INTO sesiones (agente, activa, estado, herramienta, host) VALUES (?,?,?,?,?)`,
		"CodexVisible1", 1, "activa", "codex", "localhost",
	); err != nil {
		t.Fatalf("insert sesion activa visible: %v", err)
	}

	agentes, err := ListarAgentes()
	if err != nil {
		t.Fatalf("ListarAgentes: %v", err)
	}
	estado := map[string]*Agente{}
	for _, agente := range agentes {
		estado[agente.Nombre] = agente
	}

	if !estado["CodexVisible1"].Activo || estado["CodexVisible1"].EstadoSesion != "pensando" {
		t.Fatalf("CodexVisible1 visible inesperado: %+v", estado["CodexVisible1"])
	}
	if estado["CodexVisible2"].Activo || estado["CodexVisible2"].EstadoSesion != "" {
		t.Fatalf("CodexVisible2 visible inesperado: %+v", estado["CodexVisible2"])
	}

	visible1, err := GetAgente("CodexVisible1")
	if err != nil {
		t.Fatalf("GetAgente visible1: %v", err)
	}
	if !visible1.Activo || visible1.EstadoSesion != "pensando" {
		t.Fatalf("GetAgente visible1 inesperado: %+v", visible1)
	}

	visible2, err := GetAgente("CodexVisible2")
	if err != nil {
		t.Fatalf("GetAgente visible2: %v", err)
	}
	if visible2.Activo || visible2.EstadoSesion != "" {
		t.Fatalf("GetAgente visible2 inesperado: %+v", visible2)
	}
}
