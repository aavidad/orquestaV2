package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"orquesta/db"
)

func TestConstruirInstruccionRefreshRuntime(t *testing.T) {
	t.Run("gobernanza", func(t *testing.T) {
		msg := &db.RuntimeMailboxMessage{
			ID:         7,
			FromAgente: "server",
			ToAgente:   "Codex1",
			Kind:       db.MailboxKindGovernanceRefresh,
			PayloadJSON: `{
				"tipo_agente":"programador",
				"motivo":"regla_actualizada",
				"hash":"abc123",
				"reglas":4,
				"skills":2,
				"workflows":1
			}`,
		}
		texto, ok := construirInstruccionRefreshRuntime(msg)
		if !ok {
			t.Fatalf("deberia reconocer governance_refresh")
		}
		for _, token := range []string{"gobernanza efectiva", "programador", "regla_actualizada", "abc123", "reglas=4"} {
			if !strings.Contains(texto, token) {
				t.Fatalf("texto sin %q: %s", token, texto)
			}
		}
	})

	t.Run("gobernanza_con_catalogo_efectivo", func(t *testing.T) {
		msg := &db.RuntimeMailboxMessage{
			ID:         9,
			FromAgente: "server",
			ToAgente:   "Codex1",
			Kind:       db.MailboxKindGovernanceRefresh,
			PayloadJSON: `{
				"tipo_agente":"programador",
				"motivo":"override_proyecto",
				"hash":"ctx123",
				"reglas":2,
				"skills":1,
				"workflows":1,
				"scope_tipo":"proyecto",
				"scope_ref":"orquestador"
			}`,
		}
		catalogo := &db.GovernanceCatalog{
			TipoAgente:       "programador",
			ResolucionActual: "rol+proyecto+agente",
			Reglas:           []*db.Regla{{Titulo: "server-first"}},
			Skills:           []*db.Skill{{Nombre: "docker-build"}},
			Workflows:        []*db.Workflow{{Nombre: "inicio-sesion"}},
		}
		texto, ok := construirInstruccionRefreshRuntimeConCatalogo(msg, catalogo)
		if !ok {
			t.Fatalf("deberia reconocer governance_refresh")
		}
		for _, token := range []string{"rol+proyecto+agente", "server-first", "docker-build", "inicio-sesion", "Ámbito del cambio: proyecto orquestador."} {
			if !strings.Contains(texto, token) {
				t.Fatalf("texto sin %q: %s", token, texto)
			}
		}
	})

	t.Run("skills", func(t *testing.T) {
		msg := &db.RuntimeMailboxMessage{
			ID:         8,
			FromAgente: "server",
			ToAgente:   "Codex1",
			Kind:       db.MailboxKindSkillsRefresh,
			PayloadJSON: `{
				"nombre":"docker-build",
				"motivo":"skill_actualizada",
				"origen":"catalogo"
			}`,
		}
		texto, ok := construirInstruccionRefreshRuntime(msg)
		if !ok {
			t.Fatalf("deberia reconocer skills_refresh")
		}
		for _, token := range []string{"catálogo de skills", "docker-build", "skill_actualizada", "catalogo"} {
			if !strings.Contains(texto, token) {
				t.Fatalf("texto sin %q: %s", token, texto)
			}
		}
	})

	t.Run("ignora_otros_mensajes", func(t *testing.T) {
		msg := &db.RuntimeMailboxMessage{Kind: "handoff"}
		if texto, ok := construirInstruccionRefreshRuntime(msg); ok || texto != "" {
			t.Fatalf("no deberia convertir kinds ajenos: %q", texto)
		}
	})
}

func TestProcesarRefreshRuntimeMailboxEncolaSendInstructionYConsumeMensajes(t *testing.T) {
	var (
		orderPayloads []map[string]any
		deliveredIDs  []string
		consumedIDs   []string
	)
	srv := newTestHTTPServerOrSkip(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case r.URL.Path == "/api/runtime-mailbox" && r.Method == http.MethodGet:
			if got := r.URL.Query().Get("to_agente"); got != "Codex1" {
				t.Fatalf("to_agente inesperado: %s", got)
			}
			if got := r.URL.Query().Get("proyecto"); got != "orquestador" {
				t.Fatalf("proyecto inesperado: %s", got)
			}
			if got := r.URL.Query().Get("estado"); got != "pendiente" {
				t.Fatalf("estado inesperado: %s", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"mailbox": []map[string]any{
					{
						"id":           31,
						"from_agente":  "server",
						"to_agente":    "Codex1",
						"kind":         db.MailboxKindGovernanceRefresh,
						"payload_json": `{"tipo_agente":"programador","motivo":"regla_actualizada","hash":"h1","reglas":3,"skills":2,"workflows":1,"scope_tipo":"proyecto","scope_ref":"orquestador"}`,
						"estado":       "pendiente",
						"created_at":   "2026-03-26T10:00:00Z",
					},
					{
						"id":           32,
						"from_agente":  "server",
						"to_agente":    "Codex1",
						"kind":         db.MailboxKindSkillsRefresh,
						"payload_json": `{"nombre":"docker-build","motivo":"skill_actualizada","origen":"catalogo"}`,
						"estado":       "pendiente",
						"created_at":   "2026-03-26T10:00:01Z",
					},
					{
						"id":           33,
						"from_agente":  "server",
						"to_agente":    "Codex1",
						"kind":         "handoff",
						"payload_json": `{"ok":true}`,
						"estado":       "pendiente",
						"created_at":   "2026-03-26T10:00:02Z",
					},
				},
			})
		case r.URL.Path == "/api/gobernanza/catalogo" && r.Method == http.MethodGet:
			if got := r.URL.Query().Get("tipo_agente"); got != "programador" {
				t.Fatalf("tipo_agente inesperado: %s", got)
			}
			if got := r.URL.Query().Get("proyecto"); got != "orquestador" {
				t.Fatalf("proyecto catalogo inesperado: %s", got)
			}
			if got := r.URL.Query().Get("agente"); got != "Codex1" {
				t.Fatalf("agente catalogo inesperado: %s", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"catalogo": map[string]any{
					"tipo_agente":       "programador",
					"resolucion_actual": "rol+proyecto",
					"hash":              "h1",
					"reglas": []map[string]any{
						{"id": 1, "titulo": "server-first"},
					},
					"skills": []map[string]any{
						{"id": 2, "nombre": "docker-build"},
					},
					"workflows": []map[string]any{
						{"id": 3, "nombre": "inicio-sesion"},
					},
				},
			})
		case r.URL.Path == "/api/runtime-orders" && r.Method == http.MethodPost:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode runtime-order: %v", err)
			}
			payload := map[string]any{}
			if raw, _ := body["payload"].(string); raw != "" {
				if err := json.Unmarshal([]byte(raw), &payload); err != nil {
					t.Fatalf("unmarshal payload: %v", err)
				}
			}
			orderPayloads = append(orderPayloads, payload)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": 90 + len(orderPayloads)})
		case strings.HasPrefix(r.URL.Path, "/api/runtime-mailbox/") && strings.HasSuffix(r.URL.Path, "/entregar") && r.Method == http.MethodPost:
			deliveredIDs = append(deliveredIDs, r.URL.Path)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case strings.HasPrefix(r.URL.Path, "/api/runtime-mailbox/") && strings.HasSuffix(r.URL.Path, "/consumir") && r.Method == http.MethodPost:
			consumedIDs = append(consumedIDs, r.URL.Path)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	processed, err := procesarRefreshRuntimeMailbox("Codex1", "orquestador")
	if err != nil {
		t.Fatalf("procesar refresh runtime mailbox: %v", err)
	}
	if processed != 2 {
		t.Fatalf("processed=%d, want 2", processed)
	}
	if len(orderPayloads) != 2 {
		t.Fatalf("runtime orders=%d, want 2", len(orderPayloads))
	}
	if len(deliveredIDs) != 2 || len(consumedIDs) != 2 {
		t.Fatalf("delivered=%d consumed=%d, want 2 y 2", len(deliveredIDs), len(consumedIDs))
	}
	if got := orderPayloads[0]["refresh_kind"]; got != db.MailboxKindGovernanceRefresh {
		t.Fatalf("refresh_kind governance inesperado: %#v", got)
	}
	if got := orderPayloads[1]["refresh_kind"]; got != db.MailboxKindSkillsRefresh {
		t.Fatalf("refresh_kind skills inesperado: %#v", got)
	}
	if got := orderPayloads[0]["mailbox_id"]; got != float64(31) {
		t.Fatalf("mailbox_id governance inesperado: %#v", got)
	}
	if got := orderPayloads[1]["mailbox_id"]; got != float64(32) {
		t.Fatalf("mailbox_id skills inesperado: %#v", got)
	}
	for _, payload := range orderPayloads {
		if payload["to_agente"] != "Codex1" {
			t.Fatalf("payload sin to_agente correcto: %#v", payload)
		}
		if texto, _ := payload["texto"].(string); strings.TrimSpace(texto) == "" {
			t.Fatalf("payload sin texto: %#v", payload)
		}
	}
	if texto, _ := orderPayloads[0]["texto"].(string); !strings.Contains(texto, "server-first") || !strings.Contains(texto, "rol+proyecto") {
		t.Fatalf("texto governance no enriquecido con catálogo efectivo: %s", texto)
	}
}
