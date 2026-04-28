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
	"strings"
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

func TestRutaProyectoEfectivaOmiteSesionMasRecienteStaleSiOtraSigueSana(t *testing.T) {
	tmp := prepararDBTemporal(t)
	rutaHistorica := filepath.Join(tmp, "historico", "orquestador")
	rutaActual := filepath.Join(tmp, "actual", "orquesta")
	rutaStale := filepath.Join(tmp, "tmp-stale", "orquestador")
	if err := os.MkdirAll(filepath.Join(rutaActual, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir ruta actual: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaActual, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod actual: %v", err)
	}
	if err := os.MkdirAll(rutaStale, 0o755); err != nil {
		t.Fatalf("mkdir ruta stale: %v", err)
	}
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex4", "programador"); err != nil {
		t.Fatalf("registrar Codex4: %v", err)
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
		t.Fatalf("iniciar sesion sana: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex4",
		ProyectoID:  &proyectoID,
		CWD:         rutaStale,
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion stale: %v", err)
	}

	proyecto, err := GetProyectoConRutaEfectiva("orquestador", "")
	if err != nil {
		t.Fatalf("get proyecto efectivo: %v", err)
	}
	if proyecto.RutaAbs != rutaActual {
		t.Fatalf("ruta efectiva inesperada: got=%s want=%s", proyecto.RutaAbs, rutaActual)
	}
}

func TestRutaProyectoEfectivaIgnoraCWDNoRepoAjenoAlProyecto(t *testing.T) {
	tmp := prepararDBTemporal(t)
	rutaBase := filepath.Join(tmp, "orquesta")
	rutaInvalida := filepath.Join(tmp, "orquesta-validate-launch")
	if err := os.MkdirAll(rutaBase, 0o755); err != nil {
		t.Fatalf("mkdir ruta base: %v", err)
	}
	if err := os.MkdirAll(rutaInvalida, 0o755); err != nil {
		t.Fatalf("mkdir ruta invalida: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaBase, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
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
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         rutaInvalida,
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	proyecto, err := GetProyectoConRutaEfectiva("orquestador", "")
	if err != nil {
		t.Fatalf("get proyecto efectivo: %v", err)
	}
	if proyecto.RutaAbs != rutaBase {
		t.Fatalf("ruta efectiva no deberia contaminarse con cwd invalida: got=%s want=%s", proyecto.RutaAbs, rutaBase)
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

func TestRutaTrabajoPreferidaAgenteProyectoPrepareLitePriorizaSesionDelAgente(t *testing.T) {
	tmp := prepararDBTemporal(t)
	rutaBase := filepath.Join(tmp, "orquesta")
	rutaAjena := filepath.Join(tmp, "otro-repo")
	if err := os.MkdirAll(filepath.Join(rutaBase, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir ruta base: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(rutaAjena, "pkg"), 0o755); err != nil {
		t.Fatalf("mkdir ruta ajena: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaBase, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod base: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaAjena, "go.mod"), []byte("module otrorepo\n"), 0o644); err != nil {
		t.Fatalf("write go.mod ajena: %v", err)
	}

	if err := RegistrarAgente("QwenCoder1", "programador"); err != nil {
		t.Fatalf("registrar qwen: %v", err)
	}
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar codex: %v", err)
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
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:      "QwenCoder1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(rutaBase, "cmd"),
		Herramienta: "ollama-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion qwen: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(rutaAjena, "pkg"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion codex: %v", err)
	}

	proyecto, err := GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	got := RutaTrabajoPreferidaAgenteProyectoPrepareLite("QwenCoder1", proyecto, "")
	if got != rutaBase {
		t.Fatalf("ruta prepare-lite inesperada: got=%s want=%s", got, rutaBase)
	}
}

func TestRutaTrabajoPreferidaAgenteProyectoPrepareLiteCanonicalizaAliasCodex(t *testing.T) {
	tmp := prepararDBTemporal(t)
	rutaBase := filepath.Join(tmp, "orquesta")
	rutaWorktree := filepath.Join(rutaBase, ".orquesta-worktrees", "orquestador-codex95")
	if err := os.MkdirAll(filepath.Join(rutaWorktree, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaBase, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod base: %v", err)
	}
	for _, nombre := range []string{"Codex95", "codex95"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("registrar %s: %v", nombre, err)
		}
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
		proyectoID, "Codex95", "orquestador-codex95", ".orquesta-worktrees/orquestador-codex95", "orq-orquestador-codex95", "HEAD", "activa", "legacy",
	); err != nil {
		t.Fatalf("insert worktree: %v", err)
	}
	proyecto, err := GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	got := RutaTrabajoPreferidaAgenteProyectoPrepareLite("codex95", proyecto, filepath.Join(rutaWorktree, "cmd"))
	if got != filepath.Join(rutaWorktree, "cmd") {
		t.Fatalf("ruta prepare-lite canonica inesperada: got=%s want=%s", got, filepath.Join(rutaWorktree, "cmd"))
	}
}

func TestProyectoPrepareLiteConRutaEfectivaSinAgenteNoContaminaRaizPorSesionAjena(t *testing.T) {
	tmp := prepararDBTemporal(t)
	rutaBase := filepath.Join(tmp, "orquesta")
	rutaAjena := filepath.Join(tmp, "otro-repo")
	if err := os.MkdirAll(filepath.Join(rutaBase, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir ruta base: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(rutaAjena, "pkg"), 0o755); err != nil {
		t.Fatalf("mkdir ruta ajena: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaBase, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod base: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaAjena, "go.mod"), []byte("module otrorepo\n"), 0o644); err != nil {
		t.Fatalf("write go.mod ajena: %v", err)
	}

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar codex: %v", err)
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
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(rutaAjena, "pkg"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion codex: %v", err)
	}

	proyecto, err := GetProyectoPrepareLite("orquestador")
	if err != nil {
		t.Fatalf("get proyecto prepare-lite: %v", err)
	}
	efectivo := ProyectoPrepareLiteConRutaEfectiva(proyecto, "")
	if efectivo.RutaAbs != rutaBase {
		t.Fatalf("ruta prepare-lite sin agente inesperada: got=%s want=%s", efectivo.RutaAbs, rutaBase)
	}
}

func TestProyectoPrepareLiteConRutaEfectivaRecuperaSesionSiLaRutaGuardadaEstaRota(t *testing.T) {
	tmp := prepararDBTemporal(t)
	rutaHistorica := filepath.Join(tmp, "historica-rota", "orquestador")
	rutaActual := filepath.Join(tmp, "actual", "orquesta")
	if err := os.MkdirAll(filepath.Join(rutaActual, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir ruta actual: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaActual, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod actual: %v", err)
	}

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar codex: %v", err)
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
		t.Fatalf("iniciar sesion codex: %v", err)
	}

	proyecto, err := GetProyectoPrepareLite("orquestador")
	if err != nil {
		t.Fatalf("get proyecto prepare-lite: %v", err)
	}
	efectivo := ProyectoPrepareLiteConRutaEfectiva(proyecto, "")
	if efectivo.RutaAbs != rutaActual {
		t.Fatalf("ruta prepare-lite con fallback roto inesperada: got=%s want=%s", efectivo.RutaAbs, rutaActual)
	}
}

func TestProyectoPrepareLiteConRutaEfectivaConBackendSinReadOnlyUsaSesionPrincipal(t *testing.T) {
	tmp := prepararDBTemporal(t)
	rutaStale := filepath.Join(tmp, "historica-valida", "orquestador")
	rutaActual := filepath.Join(tmp, "actual", "orquesta")
	if err := os.MkdirAll(rutaStale, 0o755); err != nil {
		t.Fatalf("mkdir ruta stale: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaStale, "go.mod"), []byte("module stale\n"), 0o644); err != nil {
		t.Fatalf("write go.mod stale: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(rutaActual, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir ruta actual: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaActual, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod actual: %v", err)
	}

	if err := RegistrarAgente("QwenCoder1", "programador"); err != nil {
		t.Fatalf("registrar qwen: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: rutaStale,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:      "QwenCoder1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(rutaActual, "cmd"),
		Herramienta: "ollama-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion qwen: %v", err)
	}

	t.Setenv("ORQUESTA_DB_DRIVER", "postgres")
	t.Setenv("ORQUESTA_DB_DSN", "")

	proyecto, err := GetProyectoPrepareLite("orquestador")
	if err != nil {
		t.Fatalf("get proyecto prepare-lite: %v", err)
	}
	efectivo := ProyectoPrepareLiteConRutaEfectiva(proyecto, "QwenCoder1")
	if efectivo.RutaAbs != rutaActual {
		t.Fatalf("ruta prepare-lite sin read-only backend inesperada: got=%s want=%s", efectivo.RutaAbs, rutaActual)
	}
}

func TestRutaTrabajoPreferidaAgenteProyectoPrefiereWorktreeActivaSobreRaizRepo(t *testing.T) {
	tmp := prepararDBTemporal(t)
	rutaBase := filepath.Join(tmp, "orquesta")
	rutaWorktree := filepath.Join(rutaBase, ".orquesta-worktrees", "orq-gemini1")
	if err := os.MkdirAll(rutaWorktree, 0o755); err != nil {
		t.Fatalf("mkdir ruta worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaBase, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod base: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaWorktree, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod worktree: %v", err)
	}
	if err := RegistrarAgente("Gemini1", "programador"); err != nil {
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
		proyectoID, "Gemini1", "orq-gemini1", rutaWorktree, "orq/orquestador/gemini1", "HEAD", "activa", "test",
	); err != nil {
		t.Fatalf("crear worktree activa: %v", err)
	}
	if got := RutaTrabajoPreferidaAgenteProyecto("Gemini1", &Proyecto{ID: proyectoID, RutaAbs: rutaBase, Slug: "orquestador"}, rutaBase); got != rutaWorktree {
		t.Fatalf("deberia preferir la worktree activa sobre la raiz del repo: got=%s want=%s", got, rutaWorktree)
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

func TestBuildProjectContextSummaryOcultaWorktreeActivaHistoricaSiElProyectoYaSeMovio(t *testing.T) {
	tmp := prepararDBTemporal(t)
	rutaHistorica := filepath.Join(tmp, "historico", "orquestador")
	rutaActual := filepath.Join(tmp, "actual", "orquesta")
	rutaWorktreeHistorica := filepath.Join(rutaHistorica, ".orquesta-worktrees", "orq-codex1")
	if err := os.MkdirAll(filepath.Join(rutaActual, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir ruta actual: %v", err)
	}
	if err := os.MkdirAll(rutaWorktreeHistorica, 0o755); err != nil {
		t.Fatalf("mkdir worktree historica: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaActual, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod actual: %v", err)
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
		proyectoID, "Codex1", "orq-codex1", rutaWorktreeHistorica, "feature/wt-codex1", "HEAD", "activa", "test",
	); err != nil {
		t.Fatalf("crear worktree historica: %v", err)
	}

	proyecto, err := GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	contexto, resumen := BuildProjectContextSummary("Codex1", proyecto)
	if _, ok := contexto["worktree_activa"]; ok {
		t.Fatalf("la worktree historica no deberia entrar en project_context: %+v", contexto)
	}
	if strings.Contains(resumen, "Worktree activa en") {
		t.Fatalf("el resumen no deberia anunciar una worktree historica: %s", resumen)
	}
}

func TestRutaTrabajoPreferidaAgenteProyectoNoReutilizaSubdirectorioViejoSinWorktreeActiva(t *testing.T) {
	tmp := prepararDBTemporal(t)
	rutaBase := filepath.Join(tmp, "repo", "orquestador")
	rutaVieja := filepath.Join(rutaBase, "claw-code-dev-rust", "rust", "crates", "api")
	if err := os.MkdirAll(rutaVieja, 0o755); err != nil {
		t.Fatalf("mkdir ruta vieja: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaBase, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod base: %v", err)
	}
	if err := RegistrarAgente("Claude1", "programador"); err != nil {
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
	got := RutaTrabajoPreferidaAgenteProyecto("Claude1", &Proyecto{ID: proyectoID, RutaAbs: rutaBase, Slug: "orquestador"}, rutaVieja)
	if got != rutaBase {
		t.Fatalf("sin worktree activa deberia volver a la raiz del proyecto: got=%s want=%s", got, rutaBase)
	}
}

func TestRutaTrabajoPreferidaAgenteProyectoConservaWorktreeSesionBajoRaizWorktrees(t *testing.T) {
	tmp := prepararDBTemporal(t)
	rutaBase := filepath.Join(tmp, "repo", "orquestador")
	rutaSesion := filepath.Join(rutaBase, ".orquesta-worktrees", "orq-ollama1")
	if err := os.MkdirAll(rutaSesion, 0o755); err != nil {
		t.Fatalf("mkdir ruta sesion: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaBase, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod base: %v", err)
	}
	if err := RegistrarAgente("Ollama1", "programador"); err != nil {
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
	got := RutaTrabajoPreferidaAgenteProyecto("Ollama1", &Proyecto{ID: proyectoID, RutaAbs: rutaBase, Slug: "orquestador"}, rutaSesion)
	if got != rutaSesion {
		t.Fatalf("la worktree de sesion bajo .orquesta-worktrees debe conservarse: got=%s want=%s", got, rutaSesion)
	}
}
