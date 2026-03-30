/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/coordinacion"
	"orquesta/runtimeagente"
)

func TestSanitizeResumeContextForProjectDescartaContextoAjeno(t *testing.T) {
	tmp := t.TempDir()
	proyecto := &Proyecto{
		Slug:    "orquesta",
		RutaAbs: filepath.Join(tmp, "orquesta"),
	}
	foreignRuta := filepath.Join(tmp, "orquestador")
	resume := runtimeagente.ResumeContext{
		ExternalSessionID:  "sess-foreign",
		ResumePayloadJSON:  fmt.Sprintf(`{"project_context":{"proyecto":{"slug":"orquestador","ruta":"%s"}},"adopted_context":{"cwd":"%s"},"governance_catalog":{"proyecto_id":1},"persist":"ok"}`, foreignRuta, foreignRuta),
		ResumenContinuidad: fmt.Sprintf("Retoma el trabajo del agente Codex4. en el proyecto orquestador. Contexto adoptado por Orquesta sobre orquestador (%s).", foreignRuta),
		Branch:             "feature/orquestador",
		CWD:                foreignRuta,
	}

	got := SanitizeResumeContextForProject(resume, proyecto)

	if got.ExternalSessionID != "" || got.Branch != "" || got.CWD != "" {
		t.Fatalf("el resume ajeno deberia resetear datos nativos: %+v", got)
	}
	if got.ResumenContinuidad != "" {
		t.Fatalf("resumen continuidad ajeno no saneado: %q", got.ResumenContinuidad)
	}
	if strings.Contains(got.ResumePayloadJSON, "orquestador") || strings.Contains(got.ResumePayloadJSON, `"project_context"`) || strings.Contains(got.ResumePayloadJSON, `"adopted_context"`) {
		t.Fatalf("payload ajeno no saneado: %s", got.ResumePayloadJSON)
	}
	if !strings.Contains(got.ResumePayloadJSON, `"persist":"ok"`) {
		t.Fatalf("el payload deberia conservar claves neutras: %s", got.ResumePayloadJSON)
	}
}

func TestPrepararStartRuntimeOrderSaneaContinuidadDeProyectoAjeno(t *testing.T) {
	prepararDBTemporal(t)
	rutaActual := filepath.Join(t.TempDir(), "orquesta-saneado")
	foreignRuta := filepath.Join(t.TempDir(), "orquestador")
	if err := os.MkdirAll(rutaActual, 0o755); err != nil {
		t.Fatalf("mkdir ruta actual: %v", err)
	}
	if err := os.MkdirAll(foreignRuta, 0o755); err != nil {
		t.Fatalf("mkdir ruta ajena: %v", err)
	}

	if err := RegistrarAgente("Codex4", "programador"); err != nil {
		t.Fatalf("registrar Codex4: %v", err)
	}
	const slugActual = "orquesta-saneado"
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    slugActual,
		Nombre:  "Orquesta",
		RutaAbs: rutaActual,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertConector(&Conector{
		Slug:       "cat-cli",
		Nombre:     "Cat CLI",
		Transporte: "cli",
		Comando:    "cat",
		Activo:     true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex4",
		ProyectoID:         &proyectoID,
		CWD:                foreignRuta,
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-foreign",
		ResumenContinuidad: fmt.Sprintf("Retoma el trabajo del agente Codex4. en el proyecto orquestador. Contexto adoptado por Orquesta sobre orquestador (%s).", foreignRuta),
		ResumePayloadJSON:  fmt.Sprintf(`{"project_context":{"proyecto":{"slug":"orquestador","ruta":"%s"}},"adopted_context":{"cwd":"%s"},"governance_catalog":{"proyecto_id":1},"persist":"ok"}`, foreignRuta, foreignRuta),
		Branch:             "feature/orquestador",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	_, proyecto, _, _, resume, _, plan, err := prepararStartRuntimeOrder("Codex4", slugActual, 0, "cat-cli", "", "", "")
	if err != nil {
		t.Fatalf("prepararStartRuntimeOrder: %v", err)
	}
	if proyecto == nil {
		t.Fatalf("proyecto nil")
	}
	if resume.ExternalSessionID != "" || resume.Branch != "" {
		t.Fatalf("resume nativo ajeno no saneado: %+v", resume)
	}
	if resume.CWD == foreignRuta || resume.CWD == "" {
		t.Fatalf("cwd no deberia seguir apuntando al proyecto ajeno: cwd=%s ruta_proyecto=%s", resume.CWD, proyecto.RutaAbs)
	}
	if strings.Contains(resume.ResumePayloadJSON, "orquestador") {
		t.Fatalf("payload sigue filtrando proyecto ajeno: %s", resume.ResumePayloadJSON)
	}
	if !strings.Contains(resume.ResumePayloadJSON, `"project_context"`) || !strings.Contains(resume.ResumePayloadJSON, `"slug":"`+slugActual+`"`) {
		t.Fatalf("payload sin contexto del proyecto actual: %s", resume.ResumePayloadJSON)
	}
	if !strings.Contains(resume.ResumePayloadJSON, `"persist":"ok"`) {
		t.Fatalf("payload sin claves neutras heredadas: %s", resume.ResumePayloadJSON)
	}
	if strings.Contains(resume.ResumenContinuidad, "orquestador") {
		t.Fatalf("resumen continuidad sigue contaminado: %s", resume.ResumenContinuidad)
	}
	if plan == nil {
		t.Fatalf("plan nil")
	}
	if plan.WorkingDir == foreignRuta || plan.WorkingDir == "" {
		t.Fatalf("working dir inesperado: wd=%s ruta_proyecto=%s", plan.WorkingDir, proyecto.RutaAbs)
	}
	if strings.Contains(plan.ContinuityPrompt, "orquestador") {
		t.Fatalf("continuity prompt contaminado: %s", plan.ContinuityPrompt)
	}
}

func TestPrepararStartRuntimeOrderAplicaXHighPorDefectoEnImplementacion(t *testing.T) {
	prepararDBTemporal(t)
	rutaActual := filepath.Join(t.TempDir(), "orquesta-runtime")
	if err := os.MkdirAll(rutaActual, 0o755); err != nil {
		t.Fatalf("mkdir ruta actual: %v", err)
	}

	if err := RegistrarAgente("Codex5", "programador"); err != nil {
		t.Fatalf("registrar Codex5: %v", err)
	}
	const slugActual = "orquesta-runtime"
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    slugActual,
		Nombre:  "Orquesta Runtime",
		RutaAbs: rutaActual,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertConector(&Conector{
		Slug:         "codex-cli",
		Nombre:       "Codex CLI",
		Transporte:   "cli",
		Comando:      "codex",
		MetadataJSON: `{"model_flag":"--model","reasoning_flag":"--reasoning-effort"}`,
		Activo:       true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	if _, err := CrearTarea(&Tarea{
		Titulo:      "Implementar runtime xhigh",
		Descripcion: "validar razonamiento por defecto",
		ProyectoID:  &proyectoID,
		Modulo:      "runtime",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	}); err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	_, _, _, _, _, _, plan, err := prepararStartRuntimeOrder("Codex5", slugActual, 0, "codex-cli", "", "", "")
	if err != nil {
		t.Fatalf("prepararStartRuntimeOrder: %v", err)
	}
	if plan == nil {
		t.Fatalf("plan nil")
	}
	if plan.PerfilTarea != "implementacion" {
		t.Fatalf("perfil inesperado: %+v", plan)
	}
	if plan.Modelo != "gpt-5.4" {
		t.Fatalf("modelo inesperado: %+v", plan)
	}
	if plan.Razonamiento != "xhigh" {
		t.Fatalf("reasoning inesperado: %+v", plan)
	}
}

func TestPrepararStartRuntimeOrderRecuperaWorktreeActivaSiResumeCWDInvalido(t *testing.T) {
	prepararDBTemporal(t)
	rutaBase := filepath.Join(t.TempDir(), "orquesta")
	rutaWorktree := filepath.Join(rutaBase, ".orquesta-worktrees", "orquesta-codex1")
	rutaInvalida := filepath.Join(t.TempDir(), "orquesta-validate-launch")
	if err := os.MkdirAll(rutaWorktree, 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}
	if err := os.MkdirAll(rutaInvalida, 0o755); err != nil {
		t.Fatalf("mkdir ruta invalida: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaBase, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod base: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaWorktree, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod worktree: %v", err)
	}

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
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
	if _, err := UpsertConector(&Conector{
		Slug:       "cat-cli",
		Nombre:     "Cat CLI",
		Transporte: "cli",
		Comando:    "cat",
		Activo:     true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	if _, err := (CoordinationWorktreeSQLRepository{}).Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		Agent:     "Codex1",
		Name:      "orquesta-codex1",
		Path:      rutaWorktree,
		Branch:    "orq-orquesta-codex1",
		BaseRef:   "HEAD",
		State:     coordinacion.WorktreeActive,
		Reason:    "test",
	}); err != nil {
		t.Fatalf("crear worktree activa: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                rutaInvalida,
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-invalid-cwd",
		ResumenContinuidad: "continuidad valida del mismo proyecto",
		ResumePayloadJSON:  `{"persist":"ok"}`,
		Branch:             "master",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	_, proyecto, _, _, resume, _, plan, err := prepararStartRuntimeOrder("Codex1", "orquestador", 0, "cat-cli", "", "", "")
	if err != nil {
		t.Fatalf("prepararStartRuntimeOrder: %v", err)
	}
	if proyecto == nil {
		t.Fatalf("proyecto nil")
	}
	if resume.CWD != rutaWorktree {
		t.Fatalf("resume cwd deberia recuperar la worktree activa: got=%s want=%s", resume.CWD, rutaWorktree)
	}
	if plan == nil {
		t.Fatalf("plan nil")
	}
	if plan.WorkingDir != rutaWorktree {
		t.Fatalf("working dir deberia apuntar a la worktree activa: got=%s want=%s", plan.WorkingDir, rutaWorktree)
	}
}

func TestConstruirResumenBootstrapDBCompactaRuidoOperativoRepetido(t *testing.T) {
	prev := "Catálogo efectivo abc123 (3 reglas, 1 skills, 1 workflows). Mailbox: 4 mensaje(s) inyectados"
	checkpoint := &RuntimeCheckpoint{
		ID:             9,
		CheckpointKind: "stop",
		Resumen:        prev,
	}
	got := construirResumenBootstrapDB(prev, nil, []*RuntimeMailboxMessage{
		{ID: 1},
		{ID: 2},
	}, checkpoint)
	if strings.Contains(got, "Mailbox: 4 mensaje(s) inyectados") {
		t.Fatalf("el resumen no deberia conservar mailbox historico: %s", got)
	}
	if !strings.Contains(got, "Mailbox: 2 mensaje(s) inyectados") {
		t.Fatalf("falta mailbox actual: %s", got)
	}
	if !strings.Contains(got, "Checkpoint #9 (stop)") {
		t.Fatalf("falta label compacta de checkpoint: %s", got)
	}
	if strings.Count(got, "Catálogo efectivo abc123") != 1 {
		t.Fatalf("el resumen deberia compactar duplicados: %s", got)
	}
}

func TestConstruirResumePayloadBootstrapDBCompactaResumenDeCheckpointRuidoso(t *testing.T) {
	checkpoint := &RuntimeCheckpoint{
		ID:             11,
		CheckpointKind: "stop",
		Resumen:        "Mailbox: 8 mensaje(s) inyectados. Catálogo efectivo abc123 (3 reglas, 1 skills, 1 workflows).",
		CWD:            "/tmp/orquesta",
	}
	got := construirResumePayloadBootstrapDB(`{"persist":"ok"}`, nil, nil, checkpoint)
	if !strings.Contains(got, `"persist":"ok"`) {
		t.Fatalf("deberia conservar el envelope previo: %s", got)
	}
	if !strings.Contains(got, `"resumen":"Checkpoint #11 (stop)"`) {
		t.Fatalf("el payload deberia compactar el resumen del checkpoint: %s", got)
	}
	if strings.Contains(got, "Mailbox: 8 mensaje(s) inyectados") {
		t.Fatalf("el payload no deberia arrastrar el resumen ruidoso del checkpoint: %s", got)
	}
}

func TestSanitizeResumePayloadForProjectEliminaMetadatosEphemerosDeArranque(t *testing.T) {
	proyecto := &Proyecto{
		Slug:    "orquestador",
		RutaAbs: "/tmp/orquestador",
	}
	got := SanitizeResumePayloadForProject(`{
		"modo":"resume",
		"native_resume":false,
		"bootstrap_prompt":"Bootstrap de Orquesta para Codex2.",
		"continuity_prompt":"Retoma el trabajo del agente Codex2. Resume payload JSON: {...}",
		"launch_prompt_embedded":true,
		"launch_prompt_mode":"embedded",
		"launch_prompt_delay_ms":250,
		"project_context":{"proyecto":{"slug":"orquestador","ruta":"/tmp/orquestador"}},
		"governance_catalog":{"hash":"abc123"},
		"checkpoint":{"id":9,"kind":"stop"},
		"persist":"ok"
	}`, proyecto)
	for _, token := range []string{
		`"modo"`,
		`"native_resume"`,
		`"bootstrap_prompt"`,
		`"continuity_prompt"`,
		`"launch_prompt_embedded"`,
		`"launch_prompt_mode"`,
		`"launch_prompt_delay_ms"`,
	} {
		if strings.Contains(got, token) {
			t.Fatalf("el payload no deberia conservar %s: %s", token, got)
		}
	}
	for _, token := range []string{
		`"project_context"`,
		`"governance_catalog"`,
		`"checkpoint"`,
		`"persist":"ok"`,
	} {
		if !strings.Contains(got, token) {
			t.Fatalf("falta informacion util %s: %s", token, got)
		}
	}
}
