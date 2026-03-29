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

	"orquesta/coordinacion"
)

func TestRutaProyectoEfectivaPriorizaCWDActivoSobreRutaHistorica(t *testing.T) {
	tmp := prepararDBTemporal(t)
	rutaHistorica := filepath.Join(tmp, "historico", "orquestador")
	rutaActual := filepath.Join(tmp, "actual", "orquesta")
	if err := os.MkdirAll(filepath.Join(rutaActual, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir ruta actual: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaActual, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: rutaHistorica,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(rutaActual, "cmd"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	proyecto, err := GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	efectivo := ProyectoConRutaEfectiva(proyecto, "")
	if efectivo.RutaAbs != rutaActual {
		t.Fatalf("ruta efectiva inesperada: got=%s want=%s", efectivo.RutaAbs, rutaActual)
	}

	contexto, _ := BuildProjectContextSummary("Codex1", proyecto)
	proyectoCtx, ok := contexto["proyecto"].(map[string]any)
	if !ok {
		t.Fatalf("project_context invalido: %+v", contexto)
	}
	if got, _ := proyectoCtx["ruta"].(string); got != rutaActual {
		t.Fatalf("ruta en context summary inesperada: got=%s want=%s", got, rutaActual)
	}
}

func TestGetYListarProyectosConRutaEfectivaPriorizanRutaActual(t *testing.T) {
	tmp := prepararDBTemporal(t)
	rutaHistorica := filepath.Join(tmp, "historico", "orquestador")
	rutaActual := filepath.Join(tmp, "actual", "orquesta")
	if err := os.MkdirAll(filepath.Join(rutaActual, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir ruta actual: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaActual, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: rutaHistorica,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(rutaActual, "cmd"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	proyecto, err := GetProyectoConRutaEfectiva("orquestador", "")
	if err != nil {
		t.Fatalf("get proyecto efectivo: %v", err)
	}
	if proyecto.RutaAbs != rutaActual {
		t.Fatalf("ruta efectiva inesperada en get: got=%s want=%s", proyecto.RutaAbs, rutaActual)
	}

	proyectos, err := ListarProyectosConRutaEfectiva(FiltroProyectos{}, "")
	if err != nil {
		t.Fatalf("listar proyectos efectivos: %v", err)
	}
	if len(proyectos) != 1 || proyectos[0] == nil {
		t.Fatalf("listado inesperado: %+v", proyectos)
	}
	if proyectos[0].RutaAbs != rutaActual {
		t.Fatalf("ruta efectiva inesperada en listar: got=%s want=%s", proyectos[0].RutaAbs, rutaActual)
	}
}

func TestRutaProyectoEfectivaNoSobrescribeRaizSiLaSesionYaEstaEnWorktreeActiva(t *testing.T) {
	tmp := prepararDBTemporal(t)
	rutaBase := filepath.Join(tmp, "orquesta")
	rutaWorktree := filepath.Join(rutaBase, ".orquesta-worktrees", "orq-codex1")
	if err := os.MkdirAll(filepath.Join(rutaWorktree, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir ruta worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaBase, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod base: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaWorktree, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod worktree: %v", err)
	}

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: rutaBase,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO worktrees (proyecto_id, agente, nombre, ruta_abs, branch, base_ref, estado, motivo)
		VALUES (?,?,?,?,?,?,?,?)`,
		proyectoID, "Codex1", "orq-codex1", rutaWorktree, "orq/orquestador/codex1", "HEAD", "activa", "test",
	); err != nil {
		t.Fatalf("crear worktree activa: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(rutaWorktree, "cmd"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	if got := RutaProyectoEfectiva(proyectoID, rutaBase, ""); got != rutaBase {
		t.Fatalf("la worktree valida no deberia reescribir la raiz del proyecto: got=%s want=%s", got, rutaBase)
	}
}

func TestListarWorktreesOcultaActivasEnRutaHistoricaSiElProyectoYaSeMovio(t *testing.T) {
	tmp := prepararDBTemporal(t)
	rutaHistorica := filepath.Join(tmp, "historico", "orquestador")
	rutaActual := filepath.Join(tmp, "actual", "orquesta")
	if err := os.MkdirAll(filepath.Join(rutaActual, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir ruta actual: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaActual, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: rutaHistorica,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(rutaActual, "cmd"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO worktrees (proyecto_id, agente, nombre, ruta_abs, branch, base_ref, estado, motivo)
		VALUES (?,?,?,?,?,?,?,?)`,
		proyectoID, "Codex1", "orq-codex1", filepath.Join(rutaHistorica, ".orquesta-worktrees", "orq-codex1"), "orq/orquestador/codex1", "HEAD", "activa", "test",
	); err != nil {
		t.Fatalf("crear worktree historica: %v", err)
	}

	estado := coordinacion.WorktreeActive
	worktrees, err := ListarWorktreesCoord(coordinacion.WorktreeFilter{
		ProjectID: &proyectoID,
		State:     &estado,
	})
	if err != nil {
		t.Fatalf("listar worktrees: %v", err)
	}
	if len(worktrees) != 0 {
		t.Fatalf("la worktree historica deberia ocultarse al listar activas: %+v", worktrees)
	}
}
