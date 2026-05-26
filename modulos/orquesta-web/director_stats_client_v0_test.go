package orquestaweb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestRESTConsultarDirectorStatsClientV0EnviaPOSTJSONYProyectaPanel(t *testing.T) {
	var received WebDirectorStatsQueryV0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		if r.URL.Path != WebDirectorStatsInboundEndpointV0 {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if got := r.Header.Get(WebDirectorStatsCorrelationHeaderV0); got != "corr-web-stats-001" {
			t.Fatalf("correlation=%q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("content-type=%q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(directorStatsResultForWebTestV0())
	})

	client := NewRESTConsultarDirectorStatsClientV0(webHTTPClientTestBaseURLV0, time.Second)
	client.HTTPClient = newWebHTTPClientForHandlerV0(handler)
	panel, err := client.ConsultarDirectorStats(context.Background(), WebDirectorStatsQueryV0{
		RequestID:     "request-ref-web-stats-001",
		CorrelationID: "corr-web-stats-001",
		Locale:        "es",
		RunRef:        " run-ref-web-stats-001 ",
		OccurredAt:    "2026-05-10T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("ConsultarDirectorStats: %v", err)
	}
	if received.Locale != "es-ES" ||
		received.RunRef != "run-ref-web-stats-001" ||
		!received.IncludeAgentProgress ||
		!received.IncludeProcessRefs ||
		received.IncludeAgentUsage {
		t.Fatalf("request=%+v", received)
	}
	if panel.RunRef != "run-ref-web-stats-001" ||
		panel.Resumen.TasksTotal != 2 ||
		len(panel.Agentes) != 1 {
		t.Fatalf("panel=%+v", panel)
	}
}

func TestRESTConsultarDirectorStatsClientV0RespetaUsoOptIn(t *testing.T) {
	var received WebDirectorStatsQueryV0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(directorStatsResultForWebTestV0())
	})

	client := NewRESTConsultarDirectorStatsClientV0(webHTTPClientTestBaseURLV0, time.Second)
	client.HTTPClient = newWebHTTPClientForHandlerV0(handler)
	_, err := client.ConsultarDirectorStats(context.Background(), WebDirectorStatsQueryV0{
		RunRef:            "run-ref-web-stats-usage-001",
		IncludeAgentUsage: true,
	})
	if err != nil {
		t.Fatalf("ConsultarDirectorStats: %v", err)
	}
	if !received.IncludeAgentUsage || !received.IncludeAgentProgress || !received.IncludeProcessRefs {
		t.Fatalf("request=%+v", received)
	}
}

func TestRESTConsultarDirectorStatsClientV0CreaRequestIDYCorrelacion(t *testing.T) {
	var received WebDirectorStatsQueryV0
	var correlation string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlation = r.Header.Get(WebDirectorStatsCorrelationHeaderV0)
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(directorStatsResultForWebTestV0())
	})

	client := NewRESTConsultarDirectorStatsClientV0(webHTTPClientTestBaseURLV0, time.Second)
	client.HTTPClient = newWebHTTPClientForHandlerV0(handler)
	if _, err := client.ConsultarDirectorStats(context.Background(), WebDirectorStatsQueryV0{
		RunRef: "run-ref-web-stats-001",
	}); err != nil {
		t.Fatalf("ConsultarDirectorStats: %v", err)
	}
	if received.RequestID == "" ||
		received.CorrelationID != received.RequestID ||
		correlation != received.RequestID {
		t.Fatalf("request_id/correlation no propagados: request=%+v header=%q", received, correlation)
	}
}

func TestRESTConsultarDirectorStatsClientV0AceptaContextNil(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(directorStatsResultForWebTestV0())
	})

	client := NewRESTConsultarDirectorStatsClientV0(webHTTPClientTestBaseURLV0, time.Second)
	client.HTTPClient = newWebHTTPClientForHandlerV0(handler)
	panel, err := client.ConsultarDirectorStats(nil, WebDirectorStatsQueryV0{
		RunRef: "run-ref-web-stats-context-nil",
	})
	if err != nil {
		t.Fatalf("ConsultarDirectorStats: %v", err)
	}
	if panel.Estado != WebDirectorStatsInboundEstadoOKV0 {
		t.Fatalf("panel=%+v", panel)
	}
}

func TestRESTConsultarDirectorStatsClientV0ErroresPublicosDelInboundNoSonTransporte(t *testing.T) {
	currentHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(WebDirectorStatsInboundResultV0{
			Estado: WebDirectorStatsInboundEstadoErrorV0,
			RunRef: "run-ref-missing",
			Errores: []WebDirectorStatsPublicIssueV0{{
				Code:  "run_no_disponible",
				Field: "run_ref",
			}},
		})
	})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		currentHandler(w, r)
	})

	client := NewRESTConsultarDirectorStatsClientV0(webHTTPClientTestBaseURLV0, time.Second)
	client.HTTPClient = newWebHTTPClientForHandlerV0(handler)
	panel, err := client.ConsultarDirectorStats(context.Background(), WebDirectorStatsQueryV0{
		Locale: "en",
		RunRef: "run-ref-missing",
	})
	if err != nil {
		t.Fatalf("error publico inbound no debe ser transporte: %v", err)
	}
	if panel.Estado != WebDirectorStatsInboundEstadoErrorV0 ||
		panel.Locale != "en-US" ||
		len(panel.ErroresPublicos) != 1 {
		t.Fatalf("panel=%+v", panel)
	}

	currentHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(WebDirectorStatsInboundResultV0{
			Estado: WebDirectorStatsInboundEstadoOKV0,
			RunRef: "run-ref-missing",
		})
	})
	_, err = client.ConsultarDirectorStats(context.Background(), WebDirectorStatsQueryV0{
		RunRef: "run-ref-missing",
	})
	if !IsWebDirectorStatsClientErrorCodeV0(err, WebDirectorStatsErrTransporteV0) {
		t.Fatalf("400 sin error publico debe ser transporte: %v", err)
	}
}

func TestRESTConsultarDirectorStatsClientV0Status500EInvalidoSonErroresPublicosCliente(t *testing.T) {
	currentHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "stack privado", http.StatusInternalServerError)
	})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		currentHandler(w, r)
	})

	client := NewRESTConsultarDirectorStatsClientV0(webHTTPClientTestBaseURLV0, time.Second)
	client.HTTPClient = newWebHTTPClientForHandlerV0(handler)
	_, err := client.ConsultarDirectorStats(context.Background(), WebDirectorStatsQueryV0{RunRef: "run-ref"})
	var clientErr WebDirectorStatsClientErrorV0
	if !errors.As(err, &clientErr) ||
		clientErr.Code != WebDirectorStatsErrTransporteV0 ||
		clientErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf("error=%+v", err)
	}

	currentHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{`))
	})
	_, err = client.ConsultarDirectorStats(context.Background(), WebDirectorStatsQueryV0{RunRef: "run-ref"})
	if !IsWebDirectorStatsClientErrorCodeV0(err, WebDirectorStatsErrRespuestaInvalidaV0) {
		t.Fatalf("invalid error=%v", err)
	}
}
