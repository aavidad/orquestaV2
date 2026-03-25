package controlruntime

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"orquesta/runtimeagente"
)

func TestArrancarPlanRemotoYControlHTTP(t *testing.T) {
	calls := make([]string, 0, 5)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("auth header inesperado: %q", got)
		}
		switch r.URL.Path {
		case "/launch":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"external_session_id": "sess-remote-1",
				"handle_ref":          "remote-handle-1",
				"capabilities": map[string]any{
					"can_pause":      true,
					"can_stop":       true,
					"can_send_input": true,
				},
			})
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer srv.Close()
	t.Setenv("ORQUESTA_REMOTE_TOKEN", "Bearer test-token")

	cfg := map[string]any{
		"endpoint":         srv.URL,
		"auth_header":      "Authorization",
		"auth_token_env":   "ORQUESTA_REMOTE_TOKEN",
		"launch_path":      "/launch",
		"pause_path":       "/pause",
		"continue_path":    "/continue",
		"stop_path":        "/stop",
		"input_path":       "/input",
		"session_id_field": "external_session_id",
		"handle_ref_field": "handle_ref",
	}
	cfgJSON, _ := json.Marshal(cfg)
	arranque, err := ArrancarPlan(SolicitudArranque{
		Agente:   "Codex1",
		Proyecto: "orquestador",
		Plan: &runtimeagente.LaunchPlan{
			Transporte:       "api",
			Modo:             "launch",
			WorkingDir:       "/tmp/orquestador",
			RemoteConfigJSON: string(cfgJSON),
		},
	})
	if err != nil {
		t.Fatalf("arrancar remoto: %v", err)
	}
	if arranque.PID != 0 || arranque.HandleKind != "session" || arranque.ExternalSessionID != "sess-remote-1" {
		t.Fatalf("arranque remoto inesperado: %+v", arranque)
	}

	obj := ObjetivoProceso{
		HandleKind:   arranque.HandleKind,
		HandleRef:    arranque.HandleRef,
		MetadataJSON: arranque.MetadataJSON,
	}
	if aplicado, _, err := EnviarInstruccionProceso(obj, "hola remoto"); err != nil || !aplicado {
		t.Fatalf("send remoto: aplicado=%t err=%v", aplicado, err)
	}
	if aplicado, _, err := PausarProceso(obj); err != nil || !aplicado {
		t.Fatalf("pause remoto: aplicado=%t err=%v", aplicado, err)
	}
	if aplicado, _, err := ContinuarProceso(obj); err != nil || !aplicado {
		t.Fatalf("continue remoto: aplicado=%t err=%v", aplicado, err)
	}
	if aplicado, _, err := DetenerProceso(obj); err != nil || !aplicado {
		t.Fatalf("stop remoto: aplicado=%t err=%v", aplicado, err)
	}

	got := strings.Join(calls, ",")
	for _, expected := range []string{"/launch", "/input", "/pause", "/continue", "/stop"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("faltaba llamada %s en %s", expected, got)
		}
	}
}

func TestArrancarPlanRemotoRespetaTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(80 * time.Millisecond)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	cfgJSON, _ := json.Marshal(map[string]any{
		"endpoint":    srv.URL,
		"launch_path": "/launch",
		"timeout_ms":  10,
	})
	_, err := ArrancarPlan(SolicitudArranque{
		Agente:   "Codex1",
		Proyecto: "orquestador",
		Plan: &runtimeagente.LaunchPlan{
			Transporte:       "api",
			Modo:             "launch",
			RemoteConfigJSON: string(cfgJSON),
		},
	})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "timeout") {
		t.Fatalf("esperaba timeout remoto, got=%v", err)
	}
}

func TestControlRemotoReintentaEnErroresTemporales(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if r.URL.Path != "/pause" {
			t.Fatalf("ruta inesperada: %s", r.URL.Path)
		}
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"busy"}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	cfgJSON, _ := json.Marshal(map[string]any{
		"endpoint":                 srv.URL,
		"pause_path":               "/pause",
		"control_retry_count":      2,
		"control_retry_backoff_ms": 1,
	})
	aplicado, _, err := PausarProceso(ObjetivoProceso{
		HandleKind:   "session",
		HandleRef:    "remote-handle-1",
		MetadataJSON: string(cfgJSON),
	})
	if err != nil || !aplicado {
		t.Fatalf("pause remoto con reintentos: aplicado=%t err=%v", aplicado, err)
	}
	if attempts != 3 {
		t.Fatalf("intentos inesperados: %d", attempts)
	}
}

func TestEnviarInstruccionRemotaNoReintentaParaEvitarDuplicados(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"busy"}`))
	}))
	defer srv.Close()

	cfgJSON, _ := json.Marshal(map[string]any{
		"endpoint":                 srv.URL,
		"input_path":               "/input",
		"control_retry_count":      3,
		"control_retry_backoff_ms": 1,
	})
	aplicado, _, err := EnviarInstruccionProceso(ObjetivoProceso{
		HandleKind:   "session",
		HandleRef:    "remote-handle-1",
		MetadataJSON: string(cfgJSON),
	}, "hola")
	if !aplicado {
		t.Fatalf("send remoto deberia intentar aplicarse")
	}
	if err == nil {
		t.Fatalf("esperaba error remoto")
	}
	if attempts != 1 {
		t.Fatalf("send_instruction no deberia reintentar, got=%d", attempts)
	}
}

func TestArrancarPlanRemotoRespetaCapacidadesDesactivadas(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/launch":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"external_session_id": "sess-remote-2",
				"handle_ref":          "remote-handle-2",
			})
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer srv.Close()

	cfgJSON, _ := json.Marshal(map[string]any{
		"endpoint":       srv.URL,
		"launch_path":    "/launch",
		"pause_path":     "",
		"input_path":     "",
		"can_pause":      false,
		"can_send_input": false,
		"can_stop":       false,
	})
	arranque, err := ArrancarPlan(SolicitudArranque{
		Agente:   "Codex1",
		Proyecto: "orquestador",
		Plan: &runtimeagente.LaunchPlan{
			Transporte:       "api",
			Modo:             "launch",
			RemoteConfigJSON: string(cfgJSON),
		},
	})
	if err != nil {
		t.Fatalf("arrancar remoto: %v", err)
	}
	caps := metadataMap(arranque.CapabilitiesJSON)
	if caps["can_pause"] != false || caps["can_send_input"] != false || caps["can_stop"] != false {
		t.Fatalf("capacidades remotas inesperadas: %+v", caps)
	}
	if caps["can_stop_without_reauth"] != false {
		t.Fatalf("can_stop_without_reauth inesperado: %+v", caps)
	}
}
