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
	"time"

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
		if got := req.Header.Get("X-Orquesta-Event-Type"); got != "runtime_panic" {
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
		Tipo:       "runtime_failure",
		ID:         7,
		Agente:     "Codex1",
		Texto:      "Runtime failure detectado | agente=Codex1 | clasificacion=runtime_panic | panic nil pointer",
		ProyectoID: 42,
		Payload: map[string]any{
			"classification": "runtime_panic",
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
	if got, _ := gotEvent["event_type"].(string); got != "runtime_panic" {
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
	if got, _ := payload["classification"].(string); got != "runtime_panic" {
		t.Fatalf("payload.classification inesperado: %+v", payload)
	}
}

func TestOpenClawGatewayNotificadorIgnoraEventoBajoValor(t *testing.T) {
	t.Parallel()

	var calls int
	n := &OpenClawGatewayNotificador{
		URL: "https://openclaw.local/gateway",
		Client: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			return &http.Response{
				StatusCode: http.StatusAccepted,
				Body:       io.NopCloser(strings.NewReader("{}")),
				Header:     make(http.Header),
			}, nil
		}),
	}
	if err := n.EnviarEvento(db.EventoNotificacion{
		Tipo:   "runtime_auto_guidance",
		Agente: "Codex1",
		Texto:  "guia automatica",
	}); err != nil {
		t.Fatalf("EnviarEvento low value: %v", err)
	}
	if calls != 0 {
		t.Fatalf("runtime_auto_guidance no deberia salir hacia OpenClaw, calls=%d", calls)
	}
}

func TestCurateOpenClawEventSoloDejaEventosOperador(t *testing.T) {
	t.Parallel()

	cases := []struct {
		nombre   string
		evento   db.EventoNotificacion
		wantType string
		wantText string
		allow    bool
	}{
		{
			nombre:   "bloqueo_hook",
			evento:   db.EventoNotificacion{Tipo: "hook:project_blocked", ID: 41, Agente: "Codex1", Texto: "falta credencial"},
			wantType: "task_blocked",
			wantText: "Tarea #41 bloqueada | agente=Codex1 | falta credencial",
			allow:    true,
		},
		{
			nombre:   "cierre_tarea",
			evento:   db.EventoNotificacion{Tipo: "hook:task_finish", ID: 55, Agente: "Codex2", Texto: "Cerrar review"},
			wantType: "task_completed",
			wantText: "Tarea #55 completada | agente=Codex2 | Cerrar review",
			allow:    true,
		},
		{
			nombre:   "review_ready",
			evento:   db.EventoNotificacion{Tipo: "runtime_review", Texto: "Frente listo", Payload: map[string]any{"stage": "ready_for_review"}},
			wantType: "review_ready",
			wantText: "Frente listo",
			allow:    true,
		},
		{
			nombre:   "handoff",
			evento:   db.EventoNotificacion{Tipo: "runtime_handoff", Texto: "Handoff Codex0 -> Codex1"},
			wantType: "handoff_completed",
			wantText: "Handoff Codex0 -> Codex1",
			allow:    true,
		},
		{
			nombre:   "progreso",
			evento:   db.EventoNotificacion{Tipo: "hook:project_unblocked", Agente: "Codex3", Texto: "review cerrada"},
			wantType: "progress_summary",
			wantText: "Bloqueo resuelto | agente=Codex3 | review cerrada",
			allow:    true,
		},
		{
			nombre:   "ruido",
			evento:   db.EventoNotificacion{Tipo: "runtime_auto_guidance", Texto: "seguir con refactor"},
			wantType: "",
			wantText: "",
			allow:    false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.nombre, func(t *testing.T) {
			t.Parallel()
			curated, eventType, ok := curateOpenClawEvent(tc.evento)
			if ok != tc.allow {
				t.Fatalf("allow inesperado: got=%v want=%v", ok, tc.allow)
			}
			if !tc.allow {
				return
			}
			if eventType != tc.wantType {
				t.Fatalf("event type inesperado: got=%q want=%q", eventType, tc.wantType)
			}
			if curated.Texto != tc.wantText {
				t.Fatalf("texto inesperado: got=%q want=%q", curated.Texto, tc.wantText)
			}
		})
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

func TestOpenClawGatewayNotificadorPersisteFalloYRetry(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "orquesta-notificaciones-retry.db")

	prevDB := os.Getenv("ORQUESTA_DB")
	prevForceLocal := os.Getenv("ORQUESTA_FORCE_LOCAL_DB")
	prevDisableServer := os.Getenv("ORQUESTA_DISABLE_SERVER_CLIENT")
	t.Cleanup(func() {
		db.Close()
		db.DB = nil
		if prevDB == "" {
			_ = os.Unsetenv("ORQUESTA_DB")
		} else {
			_ = os.Setenv("ORQUESTA_DB", prevDB)
		}
		if prevForceLocal == "" {
			_ = os.Unsetenv("ORQUESTA_FORCE_LOCAL_DB")
		} else {
			_ = os.Setenv("ORQUESTA_FORCE_LOCAL_DB", prevForceLocal)
		}
		if prevDisableServer == "" {
			_ = os.Unsetenv("ORQUESTA_DISABLE_SERVER_CLIENT")
		} else {
			_ = os.Setenv("ORQUESTA_DISABLE_SERVER_CLIENT", prevDisableServer)
		}
	})

	_ = os.Setenv("ORQUESTA_DB", dbPath)
	_ = os.Setenv("ORQUESTA_FORCE_LOCAL_DB", "1")
	_ = os.Setenv("ORQUESTA_DISABLE_SERVER_CLIENT", "1")
	if err := db.Open(); err != nil {
		t.Fatalf("open db: %v", err)
	}

	n := &OpenClawGatewayNotificador{
		URL: "https://openclaw.local/gateway",
		Client: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Body:       io.NopCloser(strings.NewReader("gateway down")),
				Header:     make(http.Header),
			}, nil
		}),
	}
	err := n.EnviarEvento(db.EventoNotificacion{Tipo: "bloqueo", ID: 1, Agente: "Codex1", Texto: "falta credencial"})
	if err == nil {
		t.Fatalf("deberia fallar el envio inicial")
	}

	items, err := db.ListarEntregasNotificacion(db.FiltroEntregasNotificacion{Canal: "openclaw_gateway", Limit: 5})
	if err != nil {
		t.Fatalf("listar entregas: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("entregas inesperadas: %+v", items)
	}
	if items[0].Estado != db.EntregaNotificacionFallida || items[0].NextRetryAt == nil {
		t.Fatalf("entrega sin retry esperado: %+v", items[0])
	}
}

func TestRetryDueGatewayDeliveriesReenviaPendientes(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "orquesta-notificaciones-outbox.db")

	prevDB := os.Getenv("ORQUESTA_DB")
	prevForceLocal := os.Getenv("ORQUESTA_FORCE_LOCAL_DB")
	prevDisableServer := os.Getenv("ORQUESTA_DISABLE_SERVER_CLIENT")
	t.Cleanup(func() {
		db.Close()
		db.DB = nil
		if prevDB == "" {
			_ = os.Unsetenv("ORQUESTA_DB")
		} else {
			_ = os.Setenv("ORQUESTA_DB", prevDB)
		}
		if prevForceLocal == "" {
			_ = os.Unsetenv("ORQUESTA_FORCE_LOCAL_DB")
		} else {
			_ = os.Setenv("ORQUESTA_FORCE_LOCAL_DB", prevForceLocal)
		}
		if prevDisableServer == "" {
			_ = os.Unsetenv("ORQUESTA_DISABLE_SERVER_CLIENT")
		} else {
			_ = os.Setenv("ORQUESTA_DISABLE_SERVER_CLIENT", prevDisableServer)
		}
	})

	_ = os.Setenv("ORQUESTA_DB", dbPath)
	_ = os.Setenv("ORQUESTA_FORCE_LOCAL_DB", "1")
	_ = os.Setenv("ORQUESTA_DISABLE_SERVER_CLIENT", "1")
	if err := db.Open(); err != nil {
		t.Fatalf("open db: %v", err)
	}

	id, err := db.CrearEntregaNotificacion("openclaw_gateway", "https://openclaw.local/gateway", db.EventoNotificacion{Tipo: "bloqueo", ID: 1, Agente: "Codex1", Texto: "falta credencial"})
	if err != nil {
		t.Fatalf("crear entrega: %v", err)
	}
	if err := db.MarcarEntregaNotificacionFallida(id, "boom", time.Now().UTC().Add(-time.Minute)); err != nil {
		t.Fatalf("marcar fallida: %v", err)
	}

	calls := 0
	notifier := &OpenClawGatewayNotificador{
		URL: "https://openclaw.local/gateway",
		Client: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			return &http.Response{
				StatusCode: http.StatusAccepted,
				Body:       io.NopCloser(strings.NewReader("{}")),
				Header:     make(http.Header),
			}, nil
		}),
	}
	n := &fanoutNotificador{items: []Notificador{notifier}}
	retried, err := RetryDueGatewayDeliveries(n, 5)
	if err != nil {
		t.Fatalf("retry due deliveries: %v", err)
	}
	if retried != 1 || calls != 1 {
		t.Fatalf("reintentos inesperados: retried=%d calls=%d", retried, calls)
	}
	item, err := db.GetEntregaNotificacion(id)
	if err != nil {
		t.Fatalf("get entrega: %v", err)
	}
	if item == nil || item.Estado != db.EntregaNotificacionEntregada || item.DeliveredAt == nil {
		t.Fatalf("entrega no consolidada como entregada: %+v", item)
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
