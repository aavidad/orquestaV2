/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package notificaciones

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestOpenClawGatewayNotificadorEnviarEvento(t *testing.T) {
	t.Parallel()

	var (
		gotAuth  string
		gotEvent map[string]any
	)
	client := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		gotAuth = req.Header.Get("Authorization")
		if req.Method != http.MethodPost {
			t.Fatalf("method inesperado: %s", req.Method)
		}
		if got := req.Header.Get("X-Orquesta-Event-Type"); got != "runtime_auto_guidance" {
			t.Fatalf("header event type inesperado: %s", got)
		}
		if req.URL.String() != "https://openclaw.local/gateway" {
			t.Fatalf("url inesperada: %s", req.URL.String())
		}
		if err := json.NewDecoder(req.Body).Decode(&gotEvent); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		return &http.Response{
			StatusCode: http.StatusAccepted,
			Body:       io.NopCloser(strings.NewReader("{}")),
			Header:     make(http.Header),
		}, nil
	})

	n := &OpenClawGatewayNotificador{
		URL:      "https://openclaw.local/gateway",
		Token:    "secreto",
		Operator: "alberto",
		Client:   client,
	}
	err := n.EnviarEvento(db.EventoNotificacion{
		Tipo:       "runtime_auto_guidance",
		ID:         7,
		Agente:     "Codex1",
		Texto:      "Orquesta envió guía automática",
		ProyectoID: 42,
		Payload: map[string]any{
			"classification": "approval_request",
			"runtime_id":     9,
		},
	})
	if err != nil {
		t.Fatalf("EnviarEvento: %v", err)
	}
	if gotAuth != "Bearer secreto" {
		t.Fatalf("auth inesperada: %q", gotAuth)
	}
	if got, _ := gotEvent["source"].(string); got != "orquesta" {
		t.Fatalf("source inesperado: %+v", gotEvent)
	}
	if got, _ := gotEvent["event_type"].(string); got != "runtime_auto_guidance" {
		t.Fatalf("event_type inesperado: %+v", gotEvent)
	}
	if got, _ := gotEvent["agent"].(string); got != "Codex1" {
		t.Fatalf("agent inesperado: %+v", gotEvent)
	}
	if got, _ := gotEvent["operator"].(string); got != "alberto" {
		t.Fatalf("operator inesperado: %+v", gotEvent)
	}
	payload, _ := gotEvent["payload"].(map[string]any)
	if payload == nil {
		t.Fatalf("payload inesperado: %+v", gotEvent)
	}
	if got, _ := payload["classification"].(string); got != "approval_request" {
		t.Fatalf("payload.classification inesperado: %+v", payload)
	}
}

func TestFanoutNotificadorEnviarEventoUsaEventAwareYFallback(t *testing.T) {
	t.Parallel()

	aware := &stubAwareNotifier{}
	legacy := &stubLegacyNotifier{}
	n := &fanoutNotificador{items: []Notificador{aware, legacy}}

	err := n.EnviarEvento(db.EventoNotificacion{
		Tipo:   "runtime_auto_guidance",
		Agente: "Codex1",
		Texto:  "guía automática",
	})
	if err != nil {
		t.Fatalf("EnviarEvento fanout: %v", err)
	}
	if len(aware.events) != 1 || aware.events[0].Tipo != "runtime_auto_guidance" {
		t.Fatalf("eventAware inesperado: %+v", aware.events)
	}
	if len(legacy.messages) != 1 || legacy.messages[0] != "OpenClaw Gateway: guía automática" {
		t.Fatalf("legacy fallback inesperado: %+v", legacy.messages)
	}
}

func TestDescribirConfiguracionReflejaOpenClawYTelegram(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "orquesta-notificaciones.db")

	anteriorDB := os.Getenv("ORQUESTA_DB")
	anteriorDSN, teniaDSN := os.LookupEnv("ORQUESTA_DB_DSN")
	anteriorDriver, teniaDriver := os.LookupEnv("ORQUESTA_DB_DRIVER")
	anteriorBackend, teniaBackend := os.LookupEnv("ORQUESTA_DB_BACKEND")
	anteriorForceLocal, teniaForceLocal := os.LookupEnv("ORQUESTA_FORCE_LOCAL_DB")
	anteriorDisableServer, teniaDisableServer := os.LookupEnv("ORQUESTA_DISABLE_SERVER_CLIENT")
	t.Cleanup(func() {
		db.Close()
		db.DB = nil
		if anteriorDB == "" {
			_ = os.Unsetenv("ORQUESTA_DB")
		} else {
			_ = os.Setenv("ORQUESTA_DB", anteriorDB)
		}
		if teniaDSN {
			_ = os.Setenv("ORQUESTA_DB_DSN", anteriorDSN)
		} else {
			_ = os.Unsetenv("ORQUESTA_DB_DSN")
		}
		if teniaDriver {
			_ = os.Setenv("ORQUESTA_DB_DRIVER", anteriorDriver)
		} else {
			_ = os.Unsetenv("ORQUESTA_DB_DRIVER")
		}
		if teniaBackend {
			_ = os.Setenv("ORQUESTA_DB_BACKEND", anteriorBackend)
		} else {
			_ = os.Unsetenv("ORQUESTA_DB_BACKEND")
		}
		if teniaForceLocal {
			_ = os.Setenv("ORQUESTA_FORCE_LOCAL_DB", anteriorForceLocal)
		} else {
			_ = os.Unsetenv("ORQUESTA_FORCE_LOCAL_DB")
		}
		if teniaDisableServer {
			_ = os.Setenv("ORQUESTA_DISABLE_SERVER_CLIENT", anteriorDisableServer)
		} else {
			_ = os.Unsetenv("ORQUESTA_DISABLE_SERVER_CLIENT")
		}
	})

	_ = os.Setenv("ORQUESTA_DB", dbPath)
	_ = os.Unsetenv("ORQUESTA_DB_DSN")
	_ = os.Unsetenv("ORQUESTA_DB_DRIVER")
	_ = os.Unsetenv("ORQUESTA_DB_BACKEND")
	_ = os.Setenv("ORQUESTA_FORCE_LOCAL_DB", "1")
	_ = os.Setenv("ORQUESTA_DISABLE_SERVER_CLIENT", "1")

	if err := db.Open(); err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		db.DB = nil
	})

	if err := db.ConfigSet("openclaw_gateway_url", "https://openclaw.local/gateway"); err != nil {
		t.Fatalf("config openclaw url: %v", err)
	}
	if err := db.ConfigSet("openclaw_gateway_token", "secreto"); err != nil {
		t.Fatalf("config openclaw token: %v", err)
	}
	if err := db.ConfigSet("openclaw_gateway_operator", "alberto"); err != nil {
		t.Fatalf("config openclaw operator: %v", err)
	}
	if err := db.ConfigSet("telegram_token", "tg-token"); err != nil {
		t.Fatalf("config telegram token: %v", err)
	}
	if err := db.ConfigSet("telegram_chat_id", "12345"); err != nil {
		t.Fatalf("config telegram chat id: %v", err)
	}

	estado := DescribirConfiguracion()
	if len(estado.Canales) != 2 {
		t.Fatalf("canales inesperados: %+v", estado)
	}
	if !estado.Canales[0].Activo || !strings.Contains(estado.Canales[0].Detalle, "openclaw.local/gateway") {
		t.Fatalf("estado openclaw inesperado: %+v", estado.Canales[0])
	}
	if !estado.Canales[1].Activo || !strings.Contains(estado.Canales[1].Detalle, "12345") {
		t.Fatalf("estado telegram inesperado: %+v", estado.Canales[1])
	}
}

type stubAwareNotifier struct {
	events []db.EventoNotificacion
}

func (s *stubAwareNotifier) EnviarMensaje(texto string) error { return nil }
func (s *stubAwareNotifier) EnviarAlertaBloqueo(tareaID int64, agente, motivo string) error {
	return nil
}
func (s *stubAwareNotifier) EnviarPropuestaVotacion(codigo, titulo string) error { return nil }
func (s *stubAwareNotifier) EnviarAvisoFinProyecto(proyectoID int64, nombre string) error {
	return nil
}
func (s *stubAwareNotifier) EnviarEvento(ev db.EventoNotificacion) error {
	s.events = append(s.events, ev)
	return nil
}

type stubLegacyNotifier struct {
	messages []string
}

func (s *stubLegacyNotifier) EnviarMensaje(texto string) error {
	s.messages = append(s.messages, texto)
	return nil
}
func (s *stubLegacyNotifier) EnviarAlertaBloqueo(tareaID int64, agente, motivo string) error {
	return nil
}
func (s *stubLegacyNotifier) EnviarPropuestaVotacion(codigo, titulo string) error { return nil }
func (s *stubLegacyNotifier) EnviarAvisoFinProyecto(proyectoID int64, nombre string) error {
	return nil
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return fn(req)
}
