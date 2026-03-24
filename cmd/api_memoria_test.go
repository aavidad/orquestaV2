package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"orquesta/db"
)

func TestAPIMemoriaEndpoints(t *testing.T) {
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
	if _, err := db.UpsertEntidadMemoria(&db.EntidadMemoria{
		Nombre:        "Core_API",
		Tipo:          string(db.EntidadMemoriaAPI),
		ValorJSON:     `{"version":"v2"}`,
		MetadataJSON:  `{"fuente":"manual"}`,
		VerificadoPor: "Codex1",
		ProyectoID:    &proyectoID,
	}); err != nil {
		t.Fatalf("upsert entidad memoria: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	assertKey := func(method, path string, body []byte, key string, wantCode int) {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, bytes.NewReader(body))
		if len(body) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}
		mux.ServeHTTP(rec, req)
		if rec.Code != wantCode {
			t.Fatalf("status inesperado %s %s: %d body=%s", method, path, rec.Code, rec.Body.String())
		}
		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode json %s %s: %v", method, path, err)
		}
		if _, ok := payload[key]; !ok {
			t.Fatalf("respuesta %s %s sin clave %q: %s", method, path, key, rec.Body.String())
		}
	}

	assertKey(http.MethodGet, "/api/memoria?proyecto=orquestador", nil, "entidades", http.StatusOK)
	assertKey(http.MethodGet, "/api/memoria/Core_API?proyecto=orquestador", nil, "entidad", http.StatusOK)
	assertKey(http.MethodPost, "/api/memoria", []byte(`{"nombre":"DB_Cluster","tipo":"infra","valor":"{\"readonly\":true}","metadata":"{}","verificado_por":"Codex2","proyecto":"orquestador"}`), "entidad", http.StatusCreated)
}
