package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/runtimeagente"
)

type runtimeTranscriptScanStub struct {
	values []any
}

func (s runtimeTranscriptScanStub) Scan(dest ...any) error {
	if len(dest) != len(s.values) {
		return fmt.Errorf("scan stub: dest=%d values=%d", len(dest), len(s.values))
	}
	for i, target := range dest {
		value := s.values[i]
		switch d := target.(type) {
		case *int64:
			switch v := value.(type) {
			case int64:
				*d = v
			case nil:
				*d = 0
			default:
				return fmt.Errorf("scan stub int64[%d]: %T", i, value)
			}
		case *string:
			switch v := value.(type) {
			case string:
				*d = v
			case nil:
				*d = ""
			default:
				return fmt.Errorf("scan stub string[%d]: %T", i, value)
			}
		case *time.Time:
			v, ok := value.(time.Time)
			if !ok {
				return fmt.Errorf("scan stub time[%d]: %T", i, value)
			}
			*d = v
		case *sql.NullInt64:
			switch v := value.(type) {
			case int64:
				*d = sql.NullInt64{Int64: v, Valid: true}
			case nil:
				*d = sql.NullInt64{}
			default:
				return fmt.Errorf("scan stub nullint64[%d]: %T", i, value)
			}
		case *sql.NullString:
			switch v := value.(type) {
			case string:
				*d = sql.NullString{String: v, Valid: true}
			case nil:
				*d = sql.NullString{}
			default:
				return fmt.Errorf("scan stub nullstring[%d]: %T", i, value)
			}
		case *sql.NullTime:
			switch v := value.(type) {
			case time.Time:
				*d = sql.NullTime{Time: v, Valid: true}
			case nil:
				*d = sql.NullTime{}
			default:
				return fmt.Errorf("scan stub nulltime[%d]: %T", i, value)
			}
		default:
			return fmt.Errorf("scan stub dest[%d] unsupported: %T", i, target)
		}
	}
	return nil
}

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

func TestRegistrarRuntimeTranscriptPersistePunteroDurableDeProgresoSemanticoEnHandle(t *testing.T) {
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

	entryID, err := RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
		RuntimeID:      runtime.ID,
		HandleID:       &handle.ID,
		Agente:         "Codex1",
		ProyectoID:     &proyectoID,
		Stream:         "pty_out",
		Text:           "he terminado el parche y voy a ejecutar tests",
		Classification: "progress_update",
	})
	if err != nil {
		t.Fatalf("registrar transcript: %v", err)
	}
	fresh, err := GetRuntimeHandle(handle.ID)
	if err != nil || fresh == nil {
		t.Fatalf("fresh handle: %+v err=%v", fresh, err)
	}
	meta := mapFromJSON(fresh.MetadataJSON)
	if got := int64FromMap(meta, "semantic_progress_transcript_id"); got != entryID {
		t.Fatalf("semantic_progress_transcript_id=%d, want %d meta=%+v", got, entryID, meta)
	}
	if got := stringFromMap(meta, "semantic_progress_classification", ""); got != "progress_update" {
		t.Fatalf("semantic_progress_classification=%q meta=%+v", got, meta)
	}
	if got := stringFromMap(meta, "semantic_progress_at", ""); strings.TrimSpace(got) == "" {
		t.Fatalf("semantic_progress_at vacio: %+v", meta)
	}
}

func TestListarRuntimeTranscriptFiltraDesde(t *testing.T) {
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
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}

	oldID, err := RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
		RuntimeID:  runtime.ID,
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		Stream:     "pty_out",
		Text:       "viejo",
	})
	if err != nil {
		t.Fatalf("insert old transcript: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_transcript SET created_at = ? WHERE id = ?`, time.Now().UTC().Add(-2*time.Hour), oldID); err != nil {
		t.Fatalf("retrofechar transcript: %v", err)
	}
	if _, err := RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
		RuntimeID:  runtime.ID,
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		Stream:     "pty_out",
		Text:       "reciente",
	}); err != nil {
		t.Fatalf("insert recent transcript: %v", err)
	}

	agente := "Codex1"
	desde := time.Now().UTC().Add(-30 * time.Minute)
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Desde: &desde, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 1 || items[0] == nil || !strings.Contains(items[0].Text, "reciente") {
		t.Fatalf("transcript filtrado inesperado: %+v", items)
	}
}

func TestListarRuntimeTranscriptReclasificaHistoricoVacioAlLeer(t *testing.T) {
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

	id, err := insertReturningID(`
		INSERT INTO runtime_transcript (
			runtime_id, handle_id, agente, proyecto_id, stream, byte_offset,
			text, normalized_text, classification, handling_note, handled_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		runtime.ID, handle.ID, "Codex1", proyectoID, "pty_out", int64(0),
		"• Ran git diff -- cmd/controlplane_support.go", normalizarTextoTranscript("• Ran git diff -- cmd/controlplane_support.go"), "", "", nil,
	)
	if err != nil {
		t.Fatalf("insert transcript historico: %v", err)
	}
	if id <= 0 {
		t.Fatalf("id inesperado: %d", id)
	}

	agente := "Codex1"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 1 || items[0] == nil {
		t.Fatalf("transcript inesperado: %+v", items)
	}
	if items[0].Classification != "tool_execution" {
		t.Fatalf("deberia reclasificar al leer, got=%q", items[0].Classification)
	}
}

func TestListarRuntimeTranscriptCanonicalizaAliasCodex(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	for _, nombre := range []string{"Codex83", "codex83"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", nombre, err)
		}
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "transcript-canon",
		Nombre:  "Transcript Canon",
		RutaAbs: filepath.Join(t.TempDir(), "transcript-canon"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex83",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "transcript-canon"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("IniciarSesionContexto: %v", err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("GetRuntimeBySesionID: runtime=%+v err=%v", runtime, err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("GetRuntimeHandleBySesionID: handle=%+v err=%v", handle, err)
	}
	if err := RegistrarRuntimeTranscriptInput(handle, runtime, "hola mundo"); err != nil {
		t.Fatalf("RegistrarRuntimeTranscriptInput: %v", err)
	}

	agente := "codex83"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, ProyectoID: &proyectoID, Limit: 10})
	if err != nil {
		t.Fatalf("ListarRuntimeTranscript: %v", err)
	}
	if len(items) != 1 || items[0] == nil || items[0].Agente != "Codex83" {
		t.Fatalf("transcript inesperado: %+v", items)
	}
}

func TestIngestarRuntimeTranscriptHandleNoClasificaLineaSistemaScript(t *testing.T) {
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

	logPath := filepath.Join(t.TempDir(), "system.log")
	logData := strings.Join([]string{
		`Script started on 2026-03-31 13:18:00+02:00 [COMMAND="'/tmp/codex-perfiles/bin/codex-perfil' 'Codex1' 'exec'"]`,
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
		t.Fatalf("esperaba 1 linea de sistema ingerida, got=%d", n)
	}

	agente := "Codex1"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("transcript inesperado: %+v", items)
	}
	if items[0].Stream != "system" || items[0].Classification != "" {
		t.Fatalf("la linea de sistema no deberia clasificarse como fallo: %+v", items[0])
	}
	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_events WHERE runtime_id=?`, runtime.ID).Scan(&count); err != nil {
		t.Fatalf("count runtime_events: %v", err)
	}
	if count != 0 {
		t.Fatalf("no deberia generar runtime_events para el banner de script, got=%d", count)
	}
}

func TestIngestarRuntimeTranscriptActivosFiltraHandlesSinLogPath(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente 1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente 2: %v", err)
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

	sesion1, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador1"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion 1: %v", err)
	}
	handle1, err := GetRuntimeHandleBySesionID(sesion1.ID)
	if err != nil || handle1 == nil {
		t.Fatalf("handle 1: %+v err=%v", handle1, err)
	}

	sesion2, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex2",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador2"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion 2: %v", err)
	}
	handle2, err := GetRuntimeHandleBySesionID(sesion2.ID)
	if err != nil || handle2 == nil {
		t.Fatalf("handle 2: %+v err=%v", handle2, err)
	}

	logPath := filepath.Join(t.TempDir(), "codex1.log")
	if err := os.WriteFile(logPath, []byte("Necesito tu aprobación\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{"log_path": logPath})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle1.ID); err != nil {
		t.Fatalf("update handle1 metadata: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json='{}' WHERE id=?`, handle2.ID); err != nil {
		t.Fatalf("update handle2 metadata: %v", err)
	}

	n, err := IngestarRuntimeTranscriptActivos()
	if err != nil {
		t.Fatalf("ingestar transcript activos: %v", err)
	}
	if n != 1 {
		t.Fatalf("solo deberia ingerir el handle con log_path, got=%d", n)
	}
}

func TestIngestarRuntimeTranscriptActivosDifiereHandleFallidoEstable(t *testing.T) {
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
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}

	logPath := filepath.Join(t.TempDir(), "fallido.log")
	if err := os.WriteFile(logPath, []byte("thread 'main' panicked at src/ui.rs:1:1\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{"log_path": logPath})
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='fallido', metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := IngestarRuntimeTranscriptActivos()
	if err != nil {
		t.Fatalf("primer ingestado transcript: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 línea ingerida en el handle fallido, got=%d", n)
	}
	n, err = IngestarRuntimeTranscriptActivos()
	if err != nil {
		t.Fatalf("segundo ingestado transcript: %v", err)
	}
	if n != 0 {
		t.Fatalf("no debería reingestar el handle fallido estable, got=%d", n)
	}

	blockedDir := filepath.Join(t.TempDir(), "blocked")
	if err := os.Mkdir(blockedDir, 0o700); err != nil {
		t.Fatalf("mkdir blocked: %v", err)
	}
	if err := os.Chmod(blockedDir, 0o000); err != nil {
		t.Fatalf("chmod blocked: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(blockedDir, 0o700)
	})

	blockedMetaJSON, _ := json.Marshal(map[string]any{
		"log_path": filepath.Join(blockedDir, "denied.log"),
	})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(blockedMetaJSON), handle.ID); err != nil {
		t.Fatalf("update blocked handle metadata: %v", err)
	}

	n, err = IngestarRuntimeTranscriptActivos()
	if err != nil {
		t.Fatalf("el cooldown en memoria debería evitar re-sondear el handle fallido estable: %v", err)
	}
	if n != 0 {
		t.Fatalf("el handle fallido estable no debería aportar trabajo mientras sigue en cooldown, got=%d", n)
	}

	resetRuntimeTranscriptHotIdleState()
	if _, err := IngestarRuntimeTranscriptActivos(); err == nil {
		t.Fatalf("sin cooldown en memoria el log_path inaccesible debería volver a aflorar error")
	}
}

func TestIngestarRuntimeTranscriptActivosRespetaBatchBudget(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	prevBudget := runtimeTranscriptBatchBudgetOverride
	prevIngest := ingestRuntimeTranscriptHandleFn
	t.Cleanup(func() {
		runtimeTranscriptBatchBudgetOverride = prevBudget
		ingestRuntimeTranscriptHandleFn = prevIngest
	})

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente 1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente 2: %v", err)
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
	for _, agente := range []string{"Codex1", "Codex2"} {
		sesion, err := IniciarSesionContexto(SesionInicio{
			Agente:      agente,
			ProyectoID:  &proyectoID,
			CWD:         filepath.Join(t.TempDir(), strings.ToLower(agente)),
			Herramienta: "codex-cli",
			Branch:      "main",
		})
		if err != nil {
			t.Fatalf("iniciar sesion %s: %v", agente, err)
		}
		handle, err := GetRuntimeHandleBySesionID(sesion.ID)
		if err != nil || handle == nil {
			t.Fatalf("handle %s: %+v err=%v", agente, handle, err)
		}
		metaJSON, _ := json.Marshal(map[string]any{"log_path": filepath.Join(t.TempDir(), agente+".log")})
		if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
			t.Fatalf("update handle metadata %s: %v", agente, err)
		}
	}

	runtimeTranscriptBatchBudgetOverride = 5 * time.Millisecond
	calls := []int64{}
	ingestRuntimeTranscriptHandleFn = func(handle *RuntimeHandle) (int, error) {
		calls = append(calls, handle.ID)
		time.Sleep(8 * time.Millisecond)
		return 1, nil
	}

	n, err := IngestarRuntimeTranscriptActivos()
	if err != nil {
		t.Fatalf("ingestar transcript activos: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia cortar tras el primer handle al agotar budget, got=%d", n)
	}
	if len(calls) != 1 {
		t.Fatalf("deberia ingerir solo un handle antes de diferir, calls=%v", calls)
	}
}

func TestListarRuntimeHandlesParaPresupuestoFiltraSoloOperativosConMetadata(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente 1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente 2: %v", err)
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

	sesion1, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador1"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion 1: %v", err)
	}
	handle1, err := GetRuntimeHandleBySesionID(sesion1.ID)
	if err != nil || handle1 == nil {
		t.Fatalf("handle 1: %+v err=%v", handle1, err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json='{\"working_dir\":\"/tmp/codex1\"}' WHERE id=?`, handle1.ID); err != nil {
		t.Fatalf("update handle1 metadata: %v", err)
	}

	sesion2, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex2",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador2"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion 2: %v", err)
	}
	handle2, err := GetRuntimeHandleBySesionID(sesion2.ID)
	if err != nil || handle2 == nil {
		t.Fatalf("handle 2: %+v err=%v", handle2, err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json='' WHERE id=?`, handle2.ID); err != nil {
		t.Fatalf("update handle2 metadata: %v", err)
	}

	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='cerrado' WHERE id=?`, handle2.ID); err != nil {
		t.Fatalf("cerrar handle2: %v", err)
	}

	handles, err := ListarRuntimeHandlesParaPresupuesto()
	if err != nil {
		t.Fatalf("listar handles para presupuesto: %v", err)
	}
	if len(handles) != 1 || handles[0].ID != handle1.ID {
		t.Fatalf("handles para presupuesto inesperados: %+v", handles)
	}
}

func TestListarRuntimeHandlesParaPresupuestoPrefiereCanonicoTMUXSobreLegacy(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("CodexBudget", "programador"); err != nil {
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

	pid := int64(os.Getpid())
	runtimeLegacyID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexBudget",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime legacy: %v", err)
	}
	legacyMetaJSON, _ := json.Marshal(map[string]any{
		"driver":           "process_pty_cli",
		"working_dir":      filepath.Join(t.TempDir(), "legacy"),
		"rendered_command": "codex-perfil CodexBudget --model gpt-5.4",
	})
	if _, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"CodexBudget", proyectoID, runtimeLegacyID, "cli", "process", strconv.Itoa(os.Getpid()), string(legacyMetaJSON), time.Now().UTC()); err != nil {
		t.Fatalf("insert handle legacy: %v", err)
	}

	runDir := filepath.Join(t.TempDir(), "worker")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir worker: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"CodexBudget","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codexbudget-1","tmux_pane_id":"%17","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+now+`","alive":true,"child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now+`","started_at":"`+now+`","child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	runtimeTMUXID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexBudget",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime tmux: %v", err)
	}
	tmuxMetaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-codexbudget-1",
		"tmux_pane_id":          "%17",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"working_dir":           filepath.Join(runDir, "cwd"),
	})
	tmuxHandleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"CodexBudget", proyectoID, runtimeTMUXID, "tmux", "process", "orq-codexbudget-1/%17", string(tmuxMetaJSON), time.Now().UTC())

	handles, err := ListarRuntimeHandlesParaPresupuesto()
	if err != nil {
		t.Fatalf("listar handles para presupuesto: %v", err)
	}
	if len(handles) != 1 || handles[0].ID != tmuxHandleID {
		t.Fatalf("deberia devolver solo el handle tmux canonico: %+v", handles)
	}
}

func TestIngestarRuntimeTranscriptHandleNoClasificaRuidoOSCSpinner(t *testing.T) {
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

	logPath := filepath.Join(t.TempDir(), "spinner.log")
	logData := strings.Join([]string{
		"\x1b]0;⠋ orquesta-codex1\x1b\\\x1b]0;⠙ orquesta-codex1\x1b\\\x1b]0;⠹ orquesta-codex1\x1b\\",
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
	if n != 0 {
		t.Fatalf("el ruido OSC del spinner deberia descartarse por completo, got=%d", n)
	}

	agente := "Codex1"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("transcript inesperado: %+v", items)
	}
	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_events WHERE runtime_id=?`, runtime.ID).Scan(&count); err != nil {
		t.Fatalf("count runtime_events: %v", err)
	}
	if count != 0 {
		t.Fatalf("no deberia generar runtime_events para ruido OSC del spinner, got=%d", count)
	}
}

func TestIngestarRuntimeTranscriptHandleDescartaFragmentosPTYConControlChars(t *testing.T) {
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

	logPath := filepath.Join(t.TempDir(), "pty-fragments.log")
	logData := strings.Join([]string{
		"\x07\x08\t\tr",
		"\x07\x08\t\t\"",
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
	if n != 0 {
		t.Fatalf("los fragmentos PTY con control chars deberian descartarse, got=%d", n)
	}

	agente := "Codex1"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("transcript inesperado: %+v", items)
	}
	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_events WHERE runtime_id=?`, runtime.ID).Scan(&count); err != nil {
		t.Fatalf("count runtime_events: %v", err)
	}
	if count != 0 {
		t.Fatalf("no deberia generar runtime_events para fragmentos PTY, got=%d", count)
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
		{raw: "Retoma el trabajo actual desde Orquesta. Write-set preferente: cmd/controlplane_support.go", want: "bootstrap_guidance"},
		{raw: "diff --git a/cmd/api.go b/cmd/api.go", want: "patch_or_code_evidence"},
		{raw: "• Ran go test ./cmd -run 'TestX$' -count=1", want: "tool_execution"},
		{raw: "└ Search resolverBootstrapRuntimeLeaseConFiltro in controlplane_entities.go", want: "tool_exploration"},
		{raw: "• Explored", want: "tool_exploration"},
		{raw: "PASS\nok\tgithub.com/example/project/cmd\t0.223s", want: "tool_result_ok"},
		{raw: "nothing to commit, working tree clean", want: "tool_result_ok"},
		{raw: "Already up to date.", want: "tool_result_ok"},
		{raw: "FAIL\tgithub.com/example/project/cmd [build failed]", want: "tool_result_error"},
		{raw: "undefined: runtimeBootstrapLeaseBaseline", want: "tool_result_error"},
		{raw: "flag provided but not defined: -z", want: "tool_result_error"},
		{raw: "context deadline exceeded", want: "tool_result_error"},
		{raw: "He actualizado el handler y corregido el test roto", want: "progress_update"},
		{raw: "He dejado alineado el control plane y sigo con el siguiente corte", want: "progress_update"},
		{raw: "• hecho: ajustado selector en db/controlplane_entities.go para que, sin", want: "progress_update"},
		{raw: "Use /skills to list available skills", want: "ui_noise"},
		{raw: "codex@box:~/repo$", want: "ui_noise"},
		{raw: "$", want: "ui_noise"},
		{raw: "│ … +1 lines", want: "ui_noise"},
		{raw: "• Summarize recent commits  gpt-5.4 medium · ~/Trabajo/orquesta", want: "ui_noise"},
		{raw: "controlplane_support_test.go", want: "tool_exploration"},
		{raw: "external_session_handle|session_resume in", want: "tool_exploration"},
		{raw: "thread 'main' panicked at src/ui.rs:1:1", want: "runtime_panic"},
		{raw: "connection reset by peer", want: "runtime_crash"},
		{raw: "can i continue refactoring without waiting?", want: "approval_request"},
		{raw: "¿Qué comando tengo que ejecutar para lanzar los tests?", want: "cli_query"},
		{raw: "curl: (28) Operation timed out after 20002 milliseconds with 0 bytes from http://localhost:16543/api/status", want: "server_url_error"},
		{raw: "missing credentials for the deploy token", want: "credentials_request"},
		{raw: "can i continue refactoring without waiting", want: ""},
	}
	for _, tc := range cases {
		if got := clasificarTextoTranscript(normalizarTextoTranscript(tc.raw)); got != tc.want {
			t.Fatalf("clasificacion inesperada para %q: got=%q want=%q", tc.raw, got, tc.want)
		}
	}
}

func TestScanRuntimeTranscriptClasificaStreamsSemanticos(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	cases := []struct {
		stream string
		text   string
		want   string
	}{
		{stream: "assistant", text: "He actualizado el handler y corregido el test roto", want: "progress_update"},
		{stream: "stdout", text: "PASS\nok\tgithub.com/example/project/cmd\t0.223s", want: "tool_result_ok"},
		{stream: "stderr", text: "FAIL\tgithub.com/example/project/cmd [build failed]", want: "tool_result_error"},
	}
	for _, tc := range cases {
		createdAt := time.Now().UTC()
		row := runtimeTranscriptScanStub{values: []any{
			int64(1),
			int64(2),
			int64(3),
			"CodexStreams",
			int64(4),
			"streams-demo",
			tc.stream,
			nil,
			tc.text,
			"",
			"",
			"",
			createdAt,
			nil,
		}}
		entry, err := scanRuntimeTranscript(row)
		if err != nil {
			t.Fatalf("scan transcript %s: %v", tc.stream, err)
		}
		if entry.Classification != tc.want {
			t.Fatalf("clasificacion inesperada para %s: got=%q want=%q", tc.stream, entry.Classification, tc.want)
		}
	}
}

func TestListarRuntimeTranscriptSoloSenalesPendIgnoraProgresoUtil(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("CodexSignals", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "signal-demo",
		Nombre:  "Signal Demo",
		RutaAbs: filepath.Join(t.TempDir(), "signal-demo"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "CodexSignals",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "signal-demo"),
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
	now := time.Now().UTC()
	for _, item := range []*RuntimeTranscriptEntry{
		{
			RuntimeID:      runtime.ID,
			HandleID:       &handle.ID,
			ProyectoID:     &proyectoID,
			Agente:         "CodexSignals",
			Stream:         "pty_out",
			Text:           "• Ran go test ./cmd -run 'TestX$' -count=1",
			NormalizedText: "ran go test ./cmd -run 'testx$' -count=1",
			Classification: "tool_execution",
			CreatedAt:      now.Add(-3 * time.Minute),
		},
		{
			RuntimeID:      runtime.ID,
			HandleID:       &handle.ID,
			ProyectoID:     &proyectoID,
			Agente:         "CodexSignals",
			Stream:         "pty_out",
			Text:           "└ Search resolverBootstrapRuntimeLeaseConFiltro in controlplane_entities.go",
			NormalizedText: "search resolverbootstrapruntimeleaseconfiltro in controlplane_entities.go",
			Classification: "tool_exploration",
			CreatedAt:      now.Add(-2 * time.Minute),
		},
		{
			RuntimeID:      runtime.ID,
			HandleID:       &handle.ID,
			ProyectoID:     &proyectoID,
			Agente:         "CodexSignals",
			Stream:         "pty_out",
			Text:           "Retoma el trabajo actual desde Orquesta. Write-set preferente: cmd/controlplane_support.go",
			NormalizedText: "retoma el trabajo actual desde orquesta write-set preferente cmd/controlplane_support.go",
			Classification: "bootstrap_guidance",
			CreatedAt:      now.Add(-1 * time.Minute),
		},
		{
			RuntimeID:      runtime.ID,
			HandleID:       &handle.ID,
			ProyectoID:     &proyectoID,
			Agente:         "CodexSignals",
			Stream:         "pty_out",
			Text:           "He actualizado el handler y corregido el test roto",
			NormalizedText: "he actualizado el handler y corregido el test roto",
			Classification: "progress_update",
			CreatedAt:      now.Add(-30 * time.Second),
		},
		{
			RuntimeID:      runtime.ID,
			HandleID:       &handle.ID,
			ProyectoID:     &proyectoID,
			Agente:         "CodexSignals",
			Stream:         "pty_out",
			Text:           "No puedo continuar con el frente actual",
			NormalizedText: "no puedo continuar con el frente actual",
			Classification: "blocked",
			CreatedAt:      now,
		},
	} {
		if _, err := RegistrarRuntimeTranscript(item); err != nil {
			t.Fatalf("registrar transcript: %v", err)
		}
	}

	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{
		SoloSenalesPend: true,
		Limit:           10,
	})
	if err != nil {
		t.Fatalf("listar signals pendientes: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("signals pendientes inesperadas: %+v", items)
	}
	if got := strings.TrimSpace(items[0].Classification); got != "blocked" {
		t.Fatalf("deberia conservar solo blocked, got=%q", got)
	}
}

func TestRuntimeTranscriptLooksLikeTMUXInteractiveActivityReconoceClasificacionUtil(t *testing.T) {
	if !runtimeTranscriptLooksLikeTMUXInteractiveActivity(&RuntimeTranscriptEntry{
		Stream:         "pty_out",
		Text:           "• Ran go test ./cmd -run 'TestX$' -count=1",
		NormalizedText: "ran go test ./cmd -run 'testx$' -count=1",
		Classification: "tool_execution",
	}) {
		t.Fatal("tool_execution deberia contar como actividad interactiva útil")
	}
	if !runtimeTranscriptLooksLikeTMUXInteractiveActivity(&RuntimeTranscriptEntry{
		Stream:         "pty_out",
		Text:           "└ Search resolverBootstrapRuntimeLeaseConFiltro in controlplane_entities.go",
		NormalizedText: "search resolverbootstrapruntimeleaseconfiltro in controlplane_entities.go",
		Classification: "tool_exploration",
	}) {
		t.Fatal("tool_exploration deberia contar como actividad interactiva útil")
	}
	if runtimeTranscriptLooksLikeTMUXInteractiveActivity(&RuntimeTranscriptEntry{
		Stream:         "pty_out",
		Text:           "Retoma el trabajo actual desde Orquesta",
		NormalizedText: "retoma el trabajo actual desde orquesta",
		Classification: "bootstrap_guidance",
	}) {
		t.Fatal("bootstrap_guidance no deberia contar como actividad interactiva útil")
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

func TestIngestarRuntimeTranscriptHandleRecortaPreambuloScriptEnRuntimePanic(t *testing.T) {
	prepararDBTemporal(t)

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

	logPath := filepath.Join(t.TempDir(), "script-panic.log")
	logData := strings.Join([]string{
		`Script started on 2026-03-31 16:48:56+02:00 [COMMAND="'/tmp/codex-perfiles/bin/codex-perfil' 'Codex1' '-C' '/tmp/orquesta' 'Bootstrap enorme ...'"] The application panicked (crashed).`,
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
		t.Fatalf("debería ingerir solo una línea útil, got=%d", n)
	}

	agente := "Codex1"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 1 || items[0].Classification != "runtime_panic" {
		t.Fatalf("clasificación inesperada: %+v", items)
	}
	if strings.Contains(items[0].Text, "Script started on ") {
		t.Fatalf("el preámbulo de script no debería persistirse en el panic: %+v", items[0])
	}
	if !strings.Contains(items[0].Text, "The application panicked (crashed).") {
		t.Fatalf("debería conservar el texto útil del panic: %+v", items[0])
	}
	var message string
	if err := DB.QueryRow(`SELECT message FROM runtime_events WHERE runtime_id=? AND kind='runtime_panic' ORDER BY id DESC LIMIT 1`, runtime.ID).Scan(&message); err != nil {
		t.Fatalf("runtime_event panic: %v", err)
	}
	if strings.Contains(message, "Script started on ") {
		t.Fatalf("el evento runtime_panic no debería arrastrar el comando de script: %s", message)
	}
}

func TestIngestarRuntimeTranscriptHandleLimpiaCSIPrivadoEnRuntimePanic(t *testing.T) {
	prepararDBTemporal(t)

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

	logPath := filepath.Join(t.TempDir(), "csi-panic.log")
	logData := strings.Join([]string{
		".\x1b[<1uThe application panicked (crashed).",
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
		t.Fatalf("debería ingerir una línea útil, got=%d", n)
	}

	agente := "Codex1"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 1 || items[0].Classification != "runtime_panic" {
		t.Fatalf("clasificación inesperada: %+v", items)
	}
	if strings.Contains(items[0].Text, "\x1b[<1u") {
		t.Fatalf("la secuencia CSI privada no debería persistirse: %+v", items[0])
	}
	if strings.Contains(items[0].Text, "[<1u") {
		t.Fatalf("la secuencia CSI privada no debería quedar visible: %+v", items[0])
	}
	var message string
	if err := DB.QueryRow(`SELECT message FROM runtime_events WHERE runtime_id=? AND kind='runtime_panic' ORDER BY id DESC LIMIT 1`, runtime.ID).Scan(&message); err != nil {
		t.Fatalf("runtime_event panic: %v", err)
	}
	if strings.Contains(message, "[<1u") {
		t.Fatalf("el runtime_event no debería arrastrar el CSI privado: %s", message)
	}
}

func TestSanitizarTextoObservabilidadRuntimeLimpiaPrefijoPuntualAntesDelPanic(t *testing.T) {
	got := SanitizarTextoObservabilidadRuntime(".\x1b[<1uThe application panicked (crashed).")
	if got != "The application panicked (crashed)." {
		t.Fatalf("texto saneado inesperado: %q", got)
	}
}

func TestSanitizarTextoObservabilidadRuntimeLimpiaPrefijosHistoricosCortos(t *testing.T) {
	cases := []string{
		"sThe application panicked (crashed).",
		"[>7uThe application panicked (crashed).",
	}
	for _, raw := range cases {
		got := SanitizarTextoObservabilidadRuntime(raw)
		if got != "The application panicked (crashed)." {
			t.Fatalf("texto saneado inesperado para %q: %q", raw, got)
		}
	}
}

func TestIngestarRuntimeTranscriptHandleCompactaPendingTranscript(t *testing.T) {
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
	handle, _ := GetRuntimeHandleBySesionID(sesion.ID)

	logPath := filepath.Join(t.TempDir(), "pending.log")
	logData := "\x1b]0;spinner\x1b\\\x07\x08\t\tpartial pending"
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
	if n != 0 {
		t.Fatalf("no deberia ingerir líneas completas, got=%d", n)
	}

	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("recargar handle: %+v err=%v", handle, err)
	}
	meta := map[string]any{}
	if err := json.Unmarshal([]byte(handle.MetadataJSON), &meta); err != nil {
		t.Fatalf("metadata invalida: %v", err)
	}
	got, _ := meta["transcript_log_pending"].(string)
	if got != "partial pending" {
		t.Fatalf("pending compactado inesperado: %q", got)
	}
}

func TestIngestarRuntimeTranscriptHandleDescartaBannerCodex(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex2", "programador"); err != nil {
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
		Agente:      "Codex2",
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

	logPath := filepath.Join(t.TempDir(), "codex-banner.log")
	logData := strings.Join([]string{
		"Perfil activo: Codex2",
		"CODEX_HOME: /tmp/codex",
		"Credenciales: /tmp/auth.json",
		"Consejo: si es el primer arranque, ejecuta codex-perfil Codex2 login",
		"https://github.com/openai/codex/releases/latest",
		"https://chatgpt.com/codex/settings/usage",
		"(https://chatgpt.com/explore/pro),",
		"  (http://localhost:8080).",
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
	if n != 0 {
		t.Fatalf("el banner de Codex deberia descartarse, got=%d", n)
	}

	agente := "Codex2"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("transcript inesperado: %+v", items)
	}
	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_events WHERE runtime_id=?`, runtime.ID).Scan(&count); err != nil {
		t.Fatalf("count runtime_events: %v", err)
	}
	if count != 0 {
		t.Fatalf("no deberia generar runtime_events para el banner de Codex, got=%d", count)
	}
}

func TestIngestarRuntimeTranscriptHandleDescartaRuidoUIYConservaSenal(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex3", "programador"); err != nil {
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
		Agente:      "Codex3",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, _ := GetRuntimeHandleBySesionID(sesion.ID)

	logPath := filepath.Join(t.TempDir(), "codex-ui-noise.log")
	logData := strings.Join([]string{
		"────────────────────────────────────────────────────────────────────────────────◦working(1m 00s • esc to interrupt)›use /skills to list available skills gpt-5.4 xhigh · /home/alberto/trabajo/orquesta",
		"• waited for background terminal",
		"• ran go test ./db -run 'Test(ListarWorktreesCoordCanonizaRelativasYOcultaActivasInexistentes)'",
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
		t.Fatalf("deberia conservar solo la señal útil, got=%d", n)
	}

	agente := "Codex3"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("transcript inesperado: %+v", items)
	}
	if got := items[0].NormalizedText; !strings.Contains(got, "ran go test ./db") {
		t.Fatalf("deberia conservar la linea útil de ejecución, got=%q", got)
	}
}

func TestIngestarRuntimeTranscriptHandleDescartaSpinnerCorruptoYConservaLineaUtil(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("CodexSpinner", "programador"); err != nil {
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
		Agente:      "CodexSpinner",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, _ := GetRuntimeHandleBySesionID(sesion.ID)

	logPath := filepath.Join(t.TempDir(), "codex-spinner-corrupt.log")
	logData := strings.Join([]string{
		"0mWWoorrk•kiinWng3Wogorrkkiin◦ngg•4◦WoorrkkiinWng5Wogorrkkiinngg•6◦WWoorrkkiinWng◦7Wogorrkkiinngg•8",
		"• Necesito un repro corto dentro del mismo paquete para ver covered/observed/blocked. Lo añado temporalmente y luego lo quito.",
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
		t.Fatalf("deberia conservar solo la línea útil tras descartar spinner corrupto, got=%d", n)
	}

	agente := "CodexSpinner"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("transcript inesperado: %+v", items)
	}
	if got := items[0].NormalizedText; !strings.Contains(got, "necesito un repro corto") {
		t.Fatalf("deberia conservar la línea útil, got=%q", got)
	}
}

func TestIngestarRuntimeTranscriptHandleRecuperaLogPathDesdeWorkerSnapshot(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("CodexSnapshot", "programador"); err != nil {
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
		Agente:      "CodexSnapshot",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(t.TempDir(), "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, _ := GetRuntimeHandleBySesionID(sesion.ID)

	logPath := filepath.Join(t.TempDir(), "codex-snapshot.log")
	if err := os.WriteFile(logPath, []byte("he terminado el slice útil\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	statusPath := filepath.Join(t.TempDir(), "worker-status.json")
	statusData, _ := json.Marshal(runtimeagente.WorkerStatus{
		State:     "running",
		Alive:     true,
		LogPath:   logPath,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err := os.WriteFile(statusPath, statusData, 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_status_path": statusPath,
	})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := IngestarRuntimeTranscriptHandle(handle.ID)
	if err != nil {
		t.Fatalf("ingestar transcript: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia ingerir desde log_path rescatado del snapshot, got=%d", n)
	}

	agente := "CodexSnapshot"
	items, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("transcript inesperado: %+v", items)
	}
	if !strings.Contains(items[0].NormalizedText, "he terminado el slice útil") {
		t.Fatalf("deberia conservar la línea ingerida, got=%q", items[0].NormalizedText)
	}
}

func TestPurgarRuntimeTranscriptRuidoHistoricoBorraSoloChromeUI(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("Codex4", "programador"); err != nil {
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
		Agente:      "Codex4",
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
	handleID := handle.ID
	old := time.Now().UTC().Add(-20 * time.Minute)

	noiseID, err := RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
		RuntimeID:      runtime.ID,
		HandleID:       &handleID,
		Agente:         "Codex4",
		ProyectoID:     &proyectoID,
		Stream:         "pty_out",
		Text:           "• waited for background terminal",
		NormalizedText: normalizarTextoTranscript("• waited for background terminal"),
	})
	if err != nil {
		t.Fatalf("registrar ruido: %v", err)
	}
	keepID, err := RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
		RuntimeID:      runtime.ID,
		HandleID:       &handleID,
		Agente:         "Codex4",
		ProyectoID:     &proyectoID,
		Stream:         "pty_out",
		Text:           "• ran go test ./db -run TestFoo",
		NormalizedText: normalizarTextoTranscript("• ran go test ./db -run TestFoo"),
	})
	if err != nil {
		t.Fatalf("registrar señal util: %v", err)
	}
	signalID, err := RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
		RuntimeID:      runtime.ID,
		HandleID:       &handleID,
		Agente:         "Codex4",
		ProyectoID:     &proyectoID,
		Stream:         "pty_out",
		Text:           "¿me dejas seguir con el cambio?",
		NormalizedText: normalizarTextoTranscript("¿me dejas seguir con el cambio?"),
		Classification: "approval_request",
	})
	if err != nil {
		t.Fatalf("registrar signal: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_transcript SET created_at=? WHERE id IN (?,?,?)`, old, noiseID, keepID, signalID); err != nil {
		t.Fatalf("envejecer transcript: %v", err)
	}

	n, err := PurgarRuntimeTranscriptRuidoHistorico()
	if err != nil {
		t.Fatalf("purgar ruido historico: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia borrar solo una entrada de ruido, got=%d", n)
	}

	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_transcript WHERE id=?`, noiseID).Scan(&count); err != nil {
		t.Fatalf("count ruido: %v", err)
	}
	if count != 0 {
		t.Fatalf("la entrada de ruido deberia desaparecer")
	}
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_transcript WHERE id IN (?,?)`, keepID, signalID).Scan(&count); err != nil {
		t.Fatalf("count restantes: %v", err)
	}
	if count != 2 {
		t.Fatalf("deberia conservar señal útil y clasificación, got=%d", count)
	}
}

func TestPurgarRuntimeTranscriptRuidoHistoricoEscaneaMasAllaDelPrimerBatch(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("CodexNoise", "programador"); err != nil {
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
		Agente:      "CodexNoise",
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
	handleID := handle.ID
	old := time.Now().UTC().Add(-20 * time.Minute)

	if _, err := DB.Exec(`INSERT INTO config (clave, valor) VALUES (?, ?)`, "runtime_transcript_noise_hygiene_batch_size", "1"); err != nil {
		t.Fatalf("config batch size: %v", err)
	}
	if _, err := DB.Exec(`INSERT INTO config (clave, valor) VALUES (?, ?)`, "runtime_transcript_noise_hygiene_scan_limit", "20"); err != nil {
		t.Fatalf("config scan limit: %v", err)
	}

	for i := 0; i < 3; i++ {
		id, err := RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
			RuntimeID:      runtime.ID,
			HandleID:       &handleID,
			Agente:         "CodexNoise",
			ProyectoID:     &proyectoID,
			Stream:         "pty_out",
			Text:           fmt.Sprintf("• ran go test ./db -run TestKeep%d", i),
			NormalizedText: normalizarTextoTranscript(fmt.Sprintf("• ran go test ./db -run TestKeep%d", i)),
		})
		if err != nil {
			t.Fatalf("registrar señal util %d: %v", i, err)
		}
		if _, err := DB.Exec(`UPDATE runtime_transcript SET created_at=? WHERE id=?`, old, id); err != nil {
			t.Fatalf("envejecer señal útil %d: %v", i, err)
		}
	}

	noiseID, err := RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
		RuntimeID:      runtime.ID,
		HandleID:       &handleID,
		Agente:         "CodexNoise",
		ProyectoID:     &proyectoID,
		Stream:         "pty_out",
		Text:           "0mWWoorrk•kiinWng3Wogorrkkiin◦ngg•4W◦WoorrkkiinWng5Wogorrk•kiinngg◦6•WWoorrkkiinWng◦7Wogorrkkiinngg•8◦WWoorrk•kiinWng9Wogorrkkiin◦ngg[14;8",
		NormalizedText: normalizarTextoTranscript("0mWWoorrk•kiinWng3Wogorrkkiin◦ngg•4W◦WoorrkkiinWng5Wogorrk•kiinngg◦6•WWoorrkkiinWng◦7Wogorrkkiinngg•8◦WWoorrk•kiinWng9Wogorrkkiin◦ngg[14;8"),
	})
	if err != nil {
		t.Fatalf("registrar ruido spinner: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_transcript SET created_at=? WHERE id=?`, old, noiseID); err != nil {
		t.Fatalf("envejecer ruido: %v", err)
	}

	n, err := PurgarRuntimeTranscriptRuidoHistorico()
	if err != nil {
		t.Fatalf("purgar ruido historico: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia borrar el ruido aunque no este en el primer batch, got=%d", n)
	}

	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_transcript WHERE id=?`, noiseID).Scan(&count); err != nil {
		t.Fatalf("count ruido: %v", err)
	}
	if count != 0 {
		t.Fatalf("la entrada de ruido deberia desaparecer")
	}
}

func TestPurgarRuntimeTranscriptRuidoHistoricoCompletoAlcanzaRuidoFueraDelScanInicial(t *testing.T) {
	abrirDBTemporalRuntimeObservabilidad(t)

	if err := RegistrarAgente("CodexNoiseFull", "programador"); err != nil {
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
		Agente:      "CodexNoiseFull",
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
	handleID := handle.ID
	old := time.Now().UTC().Add(-20 * time.Minute)

	if _, err := DB.Exec(`INSERT INTO config (clave, valor) VALUES (?, ?)`, "runtime_transcript_noise_hygiene_batch_size", "1"); err != nil {
		t.Fatalf("config batch size: %v", err)
	}
	if _, err := DB.Exec(`INSERT INTO config (clave, valor) VALUES (?, ?)`, "runtime_transcript_noise_hygiene_scan_limit", "2"); err != nil {
		t.Fatalf("config scan limit: %v", err)
	}
	if _, err := DB.Exec(`INSERT INTO config (clave, valor) VALUES (?, ?)`, "runtime_transcript_noise_hygiene_full_scan_delete_limit", "10"); err != nil {
		t.Fatalf("config full delete limit: %v", err)
	}

	for i := 0; i < 3; i++ {
		id, err := RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
			RuntimeID:      runtime.ID,
			HandleID:       &handleID,
			Agente:         "CodexNoiseFull",
			ProyectoID:     &proyectoID,
			Stream:         "pty_out",
			Text:           fmt.Sprintf("• ran go test ./db -run TestKeepFull%d", i),
			NormalizedText: normalizarTextoTranscript(fmt.Sprintf("• ran go test ./db -run TestKeepFull%d", i)),
		})
		if err != nil {
			t.Fatalf("registrar señal util %d: %v", i, err)
		}
		if _, err := DB.Exec(`UPDATE runtime_transcript SET created_at=? WHERE id=?`, old, id); err != nil {
			t.Fatalf("envejecer señal útil %d: %v", i, err)
		}
	}

	noiseID, err := RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
		RuntimeID:      runtime.ID,
		HandleID:       &handleID,
		Agente:         "CodexNoiseFull",
		ProyectoID:     &proyectoID,
		Stream:         "pty_out",
		Text:           "0mWWoorrk•kiinWng3Wogorrkkiin◦ngg•4W◦WoorrkkiinWng5Wogorrk•kiinngg◦6•WWoorrkkiinWng◦7Wogorrkkiinngg•8◦WWoorrk•kiinWng9Wogorrkkiin◦ngg[14;8",
		NormalizedText: normalizarTextoTranscript("0mWWoorrk•kiinWng3Wogorrkkiin◦ngg•4W◦WoorrkkiinWng5Wogorrk•kiinngg◦6•WWoorrkkiinWng◦7Wogorrkkiinngg•8◦WWoorrk•kiinWng9Wogorrkkiin◦ngg[14;8"),
	})
	if err != nil {
		t.Fatalf("registrar ruido spinner: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_transcript SET created_at=? WHERE id=?`, old, noiseID); err != nil {
		t.Fatalf("envejecer ruido: %v", err)
	}

	n, err := PurgarRuntimeTranscriptRuidoHistorico()
	if err != nil {
		t.Fatalf("purgar ruido historico acotado: %v", err)
	}
	if n != 0 {
		t.Fatalf("la pasada acotada no deberia alcanzar el ruido fuera del scan inicial, got=%d", n)
	}

	n, err = PurgarRuntimeTranscriptRuidoHistoricoCompleto()
	if err != nil {
		t.Fatalf("purgar ruido historico completo: %v", err)
	}
	if n != 1 {
		t.Fatalf("la pasada completa deberia alcanzar el ruido fuera del scan inicial, got=%d", n)
	}

	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_transcript WHERE id=?`, noiseID).Scan(&count); err != nil {
		t.Fatalf("count ruido: %v", err)
	}
	if count != 0 {
		t.Fatalf("la entrada de ruido deberia desaparecer")
	}
}
