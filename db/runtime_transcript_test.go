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
