package orquestacli

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func mustNewSolicitarNuevaAppCliClientV0(t *testing.T, serverURL string, timeout time.Duration) *SolicitarNuevaAppCliClientV0 {
	t.Helper()
	client, err := NewSolicitarNuevaAppCliClientV0(serverURL, timeout)
	if err != nil {
		t.Fatalf("NewSolicitarNuevaAppCliClientV0: %v", err)
	}
	return client
}

func invocationForCliClientV0(serverURL string) CliInvocationContextV0 {
	return CliInvocationContextV0{
		RequestID:     "req-cli-test",
		CorrelationID: "corr-cli-test",
		ServerURL:     serverURL,
		Timeout:       time.Second,
		OutputFormat:  CliOutputFormatJSONV0,
	}
}

func minimalAppSpecRequestForCliV0() orquestafactory.AppSpecRequestV0 {
	return orquestafactory.AppSpecRequestV0{
		SchemaVersion: orquestafactory.AppSpecRequestSchemaV0,
		Locale:        "es",
		Nombre:        "Agenda",
		Objetivo:      "Coordinar ensayos",
		TipoApp:       "web",
	}
}

func writeSolicitarNuevaAppSuccessV0(t *testing.T, w http.ResponseWriter, req orquestafactory.AppSpecRequestV0, aliases bool) {
	t.Helper()
	req.SchemaVersion = orquestafactory.AppSpecRequestSchemaV0
	if req.RequestID == "" {
		req.RequestID = "req-cli-test"
	}
	if req.Source == "" {
		req.Source = SolicitarNuevaAppCliSourceV0
	}
	spec, issues := orquestafactory.SolicitarNuevaAppV0(req, fixedSolicitarNuevaAppCliClockV0())
	if len(issues) > 0 {
		t.Fatalf("fixture spec invalida: %+v", issues)
	}
	backlog, issues := orquestafactory.GenerarBacklogInicialPropuestoV0(spec)
	if len(issues) > 0 {
		t.Fatalf("fixture backlog invalido: %+v", issues)
	}
	w.Header().Set("Content-Type", "application/json")
	if aliases {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"spec":                         spec,
			"backlog_inicial_propuesto":    backlog,
			"campo_extra_ignorado_por_cli": "compat",
		})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"app_spec": spec,
		"backlog":  backlog,
	})
}

func fixedSolicitarNuevaAppCliClockV0() time.Time {
	return time.Date(2026, 5, 4, 10, 30, 0, 0, time.UTC)
}
