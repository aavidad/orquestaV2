package cmd

import (
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func TestNormalizeOpenClawDeliveryPropagaContextoOperativo(t *testing.T) {
	t.Parallel()

	retryAt := time.Date(2026, time.April, 29, 10, 30, 0, 0, time.UTC)
	ev := normalizeOpenClawDelivery(&db.EntregaNotificacion{
		ID:          8,
		TipoEvento:  "runtime_failure",
		Estado:      db.EntregaNotificacionFallida,
		Intentos:    3,
		UltimoError: "502 bad gateway desde openclaw",
		NextRetryAt: &retryAt,
		UpdatedAt:   retryAt.Add(-time.Minute),
		Evento: db.EventoNotificacion{
			Tipo:       "runtime_failure",
			Agente:     "Codex7",
			Texto:      "panic nil pointer en runtime",
			ProyectoID: 42,
			Payload: map[string]any{
				"project_slug": "orquestador",
			},
		},
	})

	if ev.NormalizedEvent != "notification.failed" {
		t.Fatalf("normalized event inesperado: %+v", ev)
	}
	if ev.Agent != "Codex7" {
		t.Fatalf("agent inesperado: %+v", ev)
	}
	if ev.Project != "orquestador" {
		t.Fatalf("project inesperado: %+v", ev)
	}
	for _, token := range []string{
		"panic nil pointer en runtime",
		"error=502 bad gateway desde openclaw",
		"intentos=3",
		"retry=2026-04-29T10:30:00Z",
	} {
		if !strings.Contains(ev.Message, token) {
			t.Fatalf("message sin %q: %q", token, ev.Message)
		}
	}
}

func TestNormalizeOpenClawDeliveryUsaFallbacksCuandoFaltaContexto(t *testing.T) {
	t.Parallel()

	ev := normalizeOpenClawDelivery(&db.EntregaNotificacion{
		ID:         9,
		TipoEvento: "runtime_review",
		Estado:     db.EntregaNotificacionPendiente,
		Evento: db.EventoNotificacion{
			Tipo:       "runtime_review",
			ProyectoID: 77,
		},
	})

	if ev.Project != "#77" {
		t.Fatalf("project fallback inesperado: %+v", ev)
	}
	if ev.Message != "evento=runtime_review" {
		t.Fatalf("message fallback inesperado: %q", ev.Message)
	}
	if ev.SuggestedAction != "vigilar_entrega" {
		t.Fatalf("suggested action inesperada: %+v", ev)
	}
}

func TestOpenClawDeliveryProjectToleraPayloadNil(t *testing.T) {
	t.Parallel()

	if got := openClawDeliveryProject(db.EventoNotificacion{ProyectoID: 12}); got != "#12" {
		t.Fatalf("project fallback inesperado con payload nil: %q", got)
	}
}
