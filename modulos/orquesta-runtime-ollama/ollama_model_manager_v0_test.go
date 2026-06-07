package orquestaruntimeollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestOllamaModelManagerV0ListaYEstado(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			writeJSONV0(t, w, map[string]any{
				"models": []map[string]any{
					{
						"name":        "llama3.2:latest",
						"modified_at": "2026-06-07T10:00:00Z",
						"size":        123,
						"digest":      "sha256:abc",
						"details": map[string]any{
							"family":             "llama",
							"parameter_size":     "3B",
							"quantization_level": "Q4_K_M",
						},
					},
				},
			})
		case "/api/ps":
			writeJSONV0(t, w, map[string]any{
				"models": []map[string]any{
					{
						"name":       "llama3.2:latest",
						"size_vram":  456,
						"expires_at": "2026-06-07T12:00:00Z",
					},
				},
			})
		default:
			t.Fatalf("ruta inesperada: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	manager := NewOllamaModelManagerV0(server.URL, server.Client())

	list, err := manager.ListRuntimeModelsV0(context.Background(), orquestaruntime.RuntimeModelListRequestV0{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list.Models) != 1 ||
		list.Models[0].Name != "llama3.2:latest" ||
		list.Models[0].Status != "available" ||
		list.Models[0].Family != "llama" ||
		list.Models[0].Parameter != "3B" ||
		list.Models[0].Quant != "Q4_K_M" {
		t.Fatalf("list=%+v", list)
	}

	status, err := manager.RuntimeModelStatusV0(context.Background(), orquestaruntime.RuntimeModelListRequestV0{})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if len(status.Models) != 1 || status.Models[0].Status != "running" || status.Models[0].SizeVRAM != 456 {
		t.Fatalf("status=%+v", status)
	}
}

func TestOllamaModelManagerV0Acciones(t *testing.T) {
	seen := make([]string, 0, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer token-real" {
			t.Fatalf("authorization=%q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		seen = append(seen, r.URL.Path+":"+body["model"].(string)+":"+stringValueV0(body["keep_alive"]))
		writeJSONV0(t, w, map[string]any{"status": "ok"})
	}))
	defer server.Close()

	manager := NewOllamaModelManagerV0(server.URL, server.Client())
	manager.BearerToken = "token-real"

	pull, err := manager.PullRuntimeModelV0(context.Background(), orquestaruntime.RuntimeModelActionRequestV0{
		Model: "qwen2.5:7b",
	})
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	serve, err := manager.ServeRuntimeModelV0(context.Background(), orquestaruntime.RuntimeModelActionRequestV0{
		Model: "qwen2.5:7b",
	})
	if err != nil {
		t.Fatalf("serve: %v", err)
	}
	stop, err := manager.StopRuntimeModelV0(context.Background(), orquestaruntime.RuntimeModelActionRequestV0{
		Model: "qwen2.5:7b",
	})
	if err != nil {
		t.Fatalf("stop: %v", err)
	}

	if !pull.Accepted || !serve.Accepted || !stop.Accepted ||
		strings.Contains(pull.BaseURL+serve.BaseURL+stop.BaseURL, "token-real") {
		t.Fatalf("pull=%+v serve=%+v stop=%+v", pull, serve, stop)
	}
	want := []string{
		"/api/pull:qwen2.5:7b:",
		"/api/generate:qwen2.5:7b:-1",
		"/api/generate:qwen2.5:7b:0",
	}
	if strings.Join(seen, "|") != strings.Join(want, "|") {
		t.Fatalf("seen=%v want=%v", seen, want)
	}
}

func TestJoinOllamaURLV0NormalizaBase(t *testing.T) {
	got, err := joinOllamaURLV0("http://127.0.0.1:11434/", "/api/tags")
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if got != "http://127.0.0.1:11434/api/tags" {
		t.Fatalf("got=%s", got)
	}
}

func writeJSONV0(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode: %v", err)
	}
}

func stringValueV0(value any) string {
	if value == nil {
		return ""
	}
	return value.(string)
}
