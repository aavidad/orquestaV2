/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"os"
	"path/filepath"
	"testing"
)

func prepararDBTemporal(t *testing.T) string {
	t.Helper()

	anteriorDB := os.Getenv("ORQUESTA_DB")
	anteriorRoot := os.Getenv("ORQUESTA_WORKSPACE_ROOT")
	t.Cleanup(func() {
		Close()
		DB = nil
		if anteriorDB == "" {
			_ = os.Unsetenv("ORQUESTA_DB")
		} else {
			_ = os.Setenv("ORQUESTA_DB", anteriorDB)
		}
		if anteriorRoot == "" {
			_ = os.Unsetenv("ORQUESTA_WORKSPACE_ROOT")
		} else {
			_ = os.Setenv("ORQUESTA_WORKSPACE_ROOT", anteriorRoot)
		}
	})

	Close()
	DB = nil

	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "orquesta-test.db")
	if err := os.Setenv("ORQUESTA_DB", dbPath); err != nil {
		t.Fatalf("setenv ORQUESTA_DB: %v", err)
	}
	if err := os.Setenv("ORQUESTA_WORKSPACE_ROOT", tmp); err != nil {
		t.Fatalf("setenv ORQUESTA_WORKSPACE_ROOT: %v", err)
	}
	if err := Open(); err != nil {
		t.Fatalf("open db temporal: %v", err)
	}
	return tmp
}

func TestSesionReanudableConProyectoYConector(t *testing.T) {
	tmp := prepararDBTemporal(t)

	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "autofirmav2",
		Nombre:  "AutofirmaV2",
		RutaAbs: filepath.Join(tmp, "AutofirmaV2"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	conector, err := GetConector("codex-cli")
	if err != nil {
		t.Fatalf("get conector: %v", err)
	}

	if err := ActivarAsignacion("codex1", proyectoID, "asignacion de prueba"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}

	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:             "codex1",
		ConectorID:         &conector.ID,
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "AutofirmaV2"),
		Herramienta:        conector.Slug,
		ExternalSessionID:  "codex-session-001",
		ResumenContinuidad: "retomar firma pendiente",
		Branch:             "feature/orquestador",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	if sesion.ConectorSlug != "codex-cli" {
		t.Fatalf("conector inesperado: %q", sesion.ConectorSlug)
	}
	if sesion.ProyectoSlug != "autofirmav2" {
		t.Fatalf("proyecto inesperado: %q", sesion.ProyectoSlug)
	}

	branch := "feature/orquestador-2"
	resumen := "reanudar desde el ultimo commit"
	if err := GuardarSesionActiva("codex1", &proyectoID, SesionUpdate{
		Branch:             &branch,
		ResumenContinuidad: &resumen,
		Heartbeat:          true,
	}); err != nil {
		t.Fatalf("guardar sesion: %v", err)
	}

	ultima, err := ObtenerUltimaSesion("codex1", &proyectoID)
	if err != nil {
		t.Fatalf("obtener ultima sesion: %v", err)
	}
	if ultima.Branch != branch {
		t.Fatalf("branch inesperada: %q", ultima.Branch)
	}
	if ultima.ResumenContinuidad != resumen {
		t.Fatalf("resumen inesperado: %q", ultima.ResumenContinuidad)
	}

	asignaciones, err := ListarAsignaciones(FiltroAsignaciones{})
	if err != nil {
		t.Fatalf("listar asignaciones: %v", err)
	}
	if len(asignaciones) != 1 {
		t.Fatalf("asignaciones esperadas=1 obtenidas=%d", len(asignaciones))
	}
	if asignaciones[0].ProyectoSlug != "autofirmav2" {
		t.Fatalf("slug de asignacion inesperado: %q", asignaciones[0].ProyectoSlug)
	}
}

func TestObtenerUltimaSesionConFiltroPorCWD(t *testing.T) {
	tmp := prepararDBTemporal(t)

	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	cwdA := filepath.Join(tmp, "orquestador", ".orquesta-worktrees", "a")
	cwdB := filepath.Join(tmp, "orquestador", ".orquesta-worktrees", "b")

	for _, in := range []SesionInicio{
		{Agente: "codex1", ProyectoID: &proyectoID, CWD: cwdA, Herramienta: "codex", ExternalSessionID: "sess-a", ResumenContinuidad: "resumen-a", Branch: "branch-a"},
		{Agente: "codex1", ProyectoID: &proyectoID, CWD: cwdB, Herramienta: "codex", ExternalSessionID: "sess-b", ResumenContinuidad: "resumen-b", Branch: "branch-b"},
	} {
		if _, err := IniciarSesionContexto(in); err != nil {
			t.Fatalf("iniciar sesion %+v: %v", in, err)
		}
		if err := FinSesion("codex1"); err != nil {
			t.Fatalf("fin sesion: %v", err)
		}
	}

	sesion, err := ObtenerUltimaSesionConFiltro("codex1", &proyectoID, cwdA)
	if err != nil {
		t.Fatalf("obtener ultima sesion filtrada: %v", err)
	}
	if sesion.CWD != cwdA {
		t.Fatalf("cwd inesperado: %q", sesion.CWD)
	}
	if sesion.ExternalSessionID != "sess-a" {
		t.Fatalf("external_session_id inesperado: %q", sesion.ExternalSessionID)
	}
}

func TestListarTareasConProyectoID(t *testing.T) {
	tmp := prepararDBTemporal(t)

	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	id, err := CrearTarea(&Tarea{
		Titulo:      "Validar listado con proyecto",
		Descripcion: "Evita desalineacion entre SELECT y Scan",
		ProyectoID:  &proyectoID,
		Modulo:      "infra",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	tareas, err := ListarTareas(FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	if len(tareas) != 1 {
		t.Fatalf("tareas esperadas=1 obtenidas=%d", len(tareas))
	}
	if tareas[0].ID != id {
		t.Fatalf("id inesperado: %d", tareas[0].ID)
	}
	if tareas[0].ProyectoID == nil || *tareas[0].ProyectoID != proyectoID {
		t.Fatalf("proyecto_id inesperado: %+v", tareas[0].ProyectoID)
	}
}
