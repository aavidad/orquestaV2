package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"orquesta/db"
)

func TestAPIHandlerRuntimeMailboxRespetaLimit(t *testing.T) {
	prevFn := apiListRuntimeMailboxFn
	t.Cleanup(func() { apiListRuntimeMailboxFn = prevFn })

	var gotFilter db.FiltroRuntimeMailbox
	apiListRuntimeMailboxFn = func(filter db.FiltroRuntimeMailbox) ([]*db.RuntimeMailboxMessage, error) {
		gotFilter = filter
		return []*db.RuntimeMailboxMessage{}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/runtime-mailbox?to_agente=Codex1&limit=7", nil)
	rec := httptest.NewRecorder()
	apiHandlerRuntimeMailbox(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	if gotFilter.Limit != 7 {
		t.Fatalf("limit inesperado: got=%d want=7", gotFilter.Limit)
	}
}

func TestCargarRuntimeMailboxRecuperacionLocalRespetaLimit(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	for i := 1; i <= 3; i++ {
		payload, _ := json.Marshal(map[string]any{"i": i})
		if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
			FromAgente:  "server",
			ToAgente:    "Codex1",
			ProyectoID:  &proyectoID,
			Kind:        "autonomia",
			PayloadJSON: string(payload),
		}); err != nil {
			t.Fatalf("crear mailbox %d: %v", i, err)
		}
	}

	query := url.Values{
		"to_agente": []string{"Codex1"},
		"proyecto":  []string{"orquestador"},
		"limit":     []string{"1"},
	}
	items, err := cargarRuntimeMailboxRecuperacionLocal(query)
	if err != nil {
		t.Fatalf("cargarRuntimeMailboxRecuperacionLocal: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("cantidad inesperada: got=%d want=1", len(items))
	}
	if items[0].ID <= 0 {
		t.Fatalf("mailbox inesperada: %+v", items[0])
	}
}
