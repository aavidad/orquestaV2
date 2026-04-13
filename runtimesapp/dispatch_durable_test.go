package runtimesapp

import (
	"strings"
	"testing"
)

func TestFusionarResultadoDispatchDurableEscribeContratoCanonico(t *testing.T) {
	got := fusionarResultadoDispatchDurable(`{"delivery_state":"queued"}`, EntradaResultadoDispatchDurable{
		EstadoDispatch: DispatchEntregada,
		EstadoEntrega:  "delivered",
		ReceiptSource:  "git_worktree",
		UltimaRazon:    "entrega valida registrada por la app",
	})

	for _, token := range []string{
		`"dispatch_state":"delivered"`,
		`"delivery_state":"delivered"`,
		`"receipt_source":"git_worktree"`,
		`"last_reason":"entrega valida registrada por la app"`,
	} {
		if !strings.Contains(got, token) {
			t.Fatalf("resultado sin token %q: %s", token, got)
		}
	}
}

func TestFusionarResultadoDispatchDurableIgnoraEstadoInvalido(t *testing.T) {
	got := fusionarResultadoDispatchDurable(`{"dispatch_state":"pending"}`, EntradaResultadoDispatchDurable{
		EstadoDispatch: EstadoDispatchDurable("raro"),
		EstadoEntrega:  "queued",
	})

	if !strings.Contains(got, `"dispatch_state":"pending"`) {
		t.Fatalf("no debio pisar dispatch_state con valor invalido: %s", got)
	}
	if !strings.Contains(got, `"delivery_state":"queued"`) {
		t.Fatalf("debio conservar delivery_state nuevo: %s", got)
	}
}
