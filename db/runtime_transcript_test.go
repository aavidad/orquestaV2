package db

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIngestarRuntimeTranscriptHandleClasificaYGeneraEventos(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}

	logPath := filepath.Join(t.TempDir(), "codex.log")
	logData := strings.Join([]string{
		"Script started on 2026-03-27",
		"¿me dejas hacer este cambio?",
		"Quedo a la espera de tu confirmación",
		"",
	}, "\n")
	if err := os.WriteFile(logPath, []byte(logData), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{"log_path": logPath})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := IngestarRuntimeTranscriptHandle(handle.ID)
	if err != nil {
		t.Fatalf("ingestar transcript: %v", err)
	}
	if n != 3 {
		t.Fatalf("esperaba 3 líneas ingeridas, got=%d", n)
	}
	agente := "Codex1"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("transcript inesperado: %+v", items)
	}
	var approval, waiting bool
	for _, item := range items {
		if item == nil {
			continue
		}
		switch item.Classification {
		case "approval_request":
			approval = true
		case "waiting_human":
			waiting = true
		}
	}
	if !approval || !waiting {
		t.Fatalf("clasificación inesperada: %+v", items)
	}
	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_events WHERE runtime_id=? AND kind IN ('approval_request','waiting_human')`, runtime.ID).Scan(&count); err != nil {
		t.Fatalf("count runtime_events: %v", err)
	}
	if count != 2 {
		t.Fatalf("runtime_events derivados inesperados: %d", count)
	}

	n, err = IngestarRuntimeTranscriptHandle(handle.ID)
	if err != nil {
		t.Fatalf("reingestar transcript: %v", err)
	}
	if n != 0 {
		t.Fatalf("no debería reingestar líneas ya procesadas, got=%d", n)
	}
}

func TestRegistrarRuntimeTranscriptInputPersisteLinea(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, _ := GetRuntimeBySesionID(sesion.ID)
	handle, _ := GetRuntimeHandleBySesionID(sesion.ID)
	if err := RegistrarRuntimeTranscriptInput(handle, runtime, "continúa automáticamente"); err != nil {
		t.Fatalf("registrar transcript input: %v", err)
	}
	agente := "Codex1"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 1 || items[0].Stream != "stdin" || items[0].Text != "continúa automáticamente" {
		t.Fatalf("entrada transcript inesperada: %+v", items)
	}
}

func TestListarRuntimeTranscriptFiltraPorTextoLibre(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "demo",
		Nombre:  "Demo",
		RutaAbs: filepath.Join(t.TempDir(), "demo"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex2",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "demo"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, _ := GetRuntimeBySesionID(sesion.ID)

	for _, item := range []*RuntimeTranscriptEntry{
		{
			RuntimeID:      runtime.ID,
			Agente:         "Codex2",
			ProyectoID:     &proyectoID,
			Stream:         "pty_out",
			Text:           "He preparado el refactor del router",
			NormalizedText: "he preparado el refactor del router",
			Classification: "ready_for_review",
		},
		{
			RuntimeID:      runtime.ID,
			Agente:         "Codex2",
			ProyectoID:     &proyectoID,
			Stream:         "pty_out",
			Text:           "Sigo con los tests de integración",
			NormalizedText: "sigo con los tests de integracion",
		},
	} {
		if _, err := RegistrarRuntimeTranscript(item); err != nil {
			t.Fatalf("registrar transcript: %v", err)
		}
	}

	agente := "Codex2"
	query := "refactor router"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{
		Agente: &agente,
		Query:  &query,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("listar transcript filtrado: %v", err)
	}
	if len(items) != 1 || !strings.Contains(strings.ToLower(items[0].Text), "refactor") {
		t.Fatalf("resultado filtrado inesperado: %+v", items)
	}
}

func TestIngestarRuntimeTranscriptHandlePersisteByteOffset(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "trace-demo",
		Nombre:  "Trace Demo",
		RutaAbs: filepath.Join(t.TempDir(), "trace-demo"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex3",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "trace-demo"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}

	logPath := filepath.Join(t.TempDir(), "offsets.log")
	if err := os.WriteFile(logPath, []byte("uno\ndos\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{"log_path": logPath})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := IngestarRuntimeTranscriptHandle(handle.ID)
	if err != nil {
		t.Fatalf("ingestar transcript: %v", err)
	}
	if n != 2 {
		t.Fatalf("esperaba 2 líneas ingeridas, got=%d", n)
	}

	agente := "Codex3"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("transcript inesperado: %+v", items)
	}
	offsets := map[string]int64{}
	for _, item := range items {
		if item == nil || item.ByteOffset == nil {
			t.Fatalf("byte_offset ausente: %+v", items)
		}
		offsets[item.Text] = *item.ByteOffset
	}
	if offsets["uno"] != 0 {
		t.Fatalf("offset de 'uno' inesperado: %d", offsets["uno"])
	}
	if offsets["dos"] != 4 {
		t.Fatalf("offset de 'dos' inesperado: %d", offsets["dos"])
	}
}

func TestClasificarTextoTranscriptReconoceReviewYReplan(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{raw: "Está listo para revisión final", want: "ready_for_review"},
		{raw: "ready for review after the last fix", want: "ready_for_review"},
		{raw: "Review aprobada, LGTM", want: "review_approved"},
		{raw: "Changes requested after review", want: "review_changes_requested"},
		{raw: "Review blocked pending credentials", want: "review_blocked"},
		{raw: "¿Qué hago ahora? no tengo claro el siguiente paso", want: "needs_replan"},
		{raw: "what should i do next after this task?", want: "needs_replan"},
		{raw: "thread 'main' panicked at src/ui.rs:1:1", want: "runtime_panic"},
		{raw: "connection reset by peer", want: "runtime_crash"},
		{raw: "can i continue refactoring without waiting?", want: "approval_request"},
		{raw: "can i continue refactoring without waiting", want: ""},
	}
	for _, tc := range cases {
		if got := clasificarTextoTranscript(normalizarTextoTranscript(tc.raw)); got != tc.want {
			t.Fatalf("clasificacion inesperada para %q: got=%q want=%q", tc.raw, got, tc.want)
		}
	}
}

func TestIngestarRuntimeTranscriptHandleFiltraRuidoYClasificaPanic(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, _ := GetRuntimeBySesionID(sesion.ID)
	handle, _ := GetRuntimeHandleBySesionID(sesion.ID)

	logPath := filepath.Join(t.TempDir(), "panic.log")
	logData := strings.Join([]string{
		"│",
		"⠋",
		"thread 'main' panicked at src/ui.rs:1:1",
		"",
	}, "\n")
	if err := os.WriteFile(logPath, []byte(logData), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{"log_path": logPath})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := IngestarRuntimeTranscriptHandle(handle.ID)
	if err != nil {
		t.Fatalf("ingestar transcript: %v", err)
	}
	if n != 1 {
		t.Fatalf("debería ingerir solo la línea semántica, got=%d", n)
	}

	agente := "Codex1"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 1 || items[0].Classification != "runtime_panic" {
		t.Fatalf("clasificación inesperada: %+v", items)
	}
	var level string
	if err := DB.QueryRow(`SELECT level FROM runtime_events WHERE runtime_id=? AND kind='runtime_panic' ORDER BY id DESC LIMIT 1`, runtime.ID).Scan(&level); err != nil {
		t.Fatalf("runtime_event panic: %v", err)
	}
	if level != "critical" {
		t.Fatalf("level inesperado para runtime_panic: %s", level)
	}
}
