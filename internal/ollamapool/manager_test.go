package ollamapool

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGestorLanzarEnviarYDetener(t *testing.T) {
	var descargaSolicitada bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/chat":
		case "/api/generate":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode payload descarga: %v", err)
			}
			if got := payload["model"]; got != "gemma4:26b" {
				t.Fatalf("modelo descarga inesperado: %+v", got)
			}
			if got := payload["keep_alive"]; got != float64(0) && got != 0 {
				t.Fatalf("keep_alive descarga inesperado: %+v", got)
			}
			descargaSolicitada = true
			_ = json.NewEncoder(w).Encode(map[string]any{"done": true})
			return
		default:
			t.Fatalf("ruta inesperada: %s", r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if got := payload["model"]; got != "gemma4:26b" {
			t.Fatalf("modelo inesperado: %+v", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]any{
				"role":    "assistant",
				"content": "PATCH: listo",
			},
		})
	}))
	defer srv.Close()

	gestor := NuevoGestor(srv.URL, nil)
	sesion, err := gestor.Lanzar(context.Background(), EntradaLanzamiento{
		Agente:       "Gemma1",
		Proyecto:     "orquestador",
		PoolSlug:     "ollama-gemma4",
		SlotsMaximos: 1,
		Modelo:       "gemma4:26b",
		PerfilTarea:  "implementacion",
		Sistema:      "MICROTAREA CERRADA",
	})
	if err != nil {
		t.Fatalf("lanzar: %v", err)
	}
	if sesion.Estado != "ready" {
		t.Fatalf("estado inicial inesperado: %+v", sesion)
	}
	resultado, err := gestor.Enviar(context.Background(), sesion.HandleRef, "Implementa la funcion")
	if err != nil {
		t.Fatalf("enviar: %v", err)
	}
	if !strings.Contains(resultado.Respuesta, "PATCH") {
		t.Fatalf("respuesta inesperada: %+v", resultado)
	}
	estado, err := gestor.Estado(sesion.HandleRef)
	if err != nil {
		t.Fatalf("estado: %v", err)
	}
	if estado.Estado != "ready" || len(estado.Mensajes) != 3 {
		t.Fatalf("estado final inesperado: %+v", estado)
	}
	if err := gestor.Detener(sesion.HandleRef); err != nil {
		t.Fatalf("detener: %v", err)
	}
	estado, err = gestor.Estado(sesion.HandleRef)
	if err != nil {
		t.Fatalf("estado tras detener: %v", err)
	}
	if estado.Estado != "stopped" {
		t.Fatalf("estado tras detener inesperado: %+v", estado)
	}
	if !descargaSolicitada {
		t.Fatal("deberia haber solicitado descarga del modelo")
	}
}

func TestGestorNoDescargaModeloSiQuedaOtraSesionActiva(t *testing.T) {
	var descargaSolicitada bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/chat":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": map[string]any{
					"role":    "assistant",
					"content": "PATCH: listo",
				},
			})
		case "/api/generate":
			descargaSolicitada = true
			_ = json.NewEncoder(w).Encode(map[string]any{"done": true})
		default:
			t.Fatalf("ruta inesperada: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	gestor := NuevoGestor(srv.URL, nil)
	s1, err := gestor.Lanzar(context.Background(), EntradaLanzamiento{
		Agente:       "Gemma1",
		Proyecto:     "orquestador",
		PoolSlug:     "ollama-gemma4",
		SlotsMaximos: 2,
		Modelo:       "gemma4:26b",
	})
	if err != nil {
		t.Fatalf("launch 1: %v", err)
	}
	if _, err := gestor.Lanzar(context.Background(), EntradaLanzamiento{
		Agente:       "Gemma2",
		Proyecto:     "orquestador",
		PoolSlug:     "ollama-gemma4",
		SlotsMaximos: 2,
		Modelo:       "gemma4:26b",
	}); err != nil {
		t.Fatalf("launch 2: %v", err)
	}
	if err := gestor.Detener(s1.HandleRef); err != nil {
		t.Fatalf("detener: %v", err)
	}
	if descargaSolicitada {
		t.Fatal("no deberia descargar modelo si queda otra sesion activa")
	}
}

func TestGestorMarcaFalloSiOllamaFalla(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer srv.Close()

	gestor := NuevoGestor(srv.URL, nil)
	sesion, err := gestor.Lanzar(context.Background(), EntradaLanzamiento{
		Agente:       "Qwen1",
		PoolSlug:     "ollama-qwen",
		SlotsMaximos: 1,
		Modelo:       "qwen3.5:9b",
	})
	if err != nil {
		t.Fatalf("lanzar: %v", err)
	}
	if _, err := gestor.Enviar(context.Background(), sesion.HandleRef, "hola"); err == nil {
		t.Fatal("esperaba error de ollama")
	}
	estado, err := gestor.Estado(sesion.HandleRef)
	if err != nil {
		t.Fatalf("estado: %v", err)
	}
	if estado.Estado != "failed" || estado.ErrorUltimo == "" {
		t.Fatalf("estado fallido inesperado: %+v", estado)
	}
}

func TestGestorUsaThinkingComoFallbackSoloParaPatchOBloqueo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]any{
				"role":     "assistant",
				"content":  "",
				"thinking": "PATCH_UNIFICADO:\n--- a/a.go\n+++ b/a.go\n@@ -1 +1 @@\n-package a\n+package b",
			},
		})
	}))
	defer srv.Close()

	gestor := NuevoGestor(srv.URL, nil)
	sesion, err := gestor.Lanzar(context.Background(), EntradaLanzamiento{
		Agente:       "Gemma1",
		Proyecto:     "orquestador",
		PoolSlug:     "ollama-gemma4",
		SlotsMaximos: 1,
		Modelo:       "gemma4:26b",
	})
	if err != nil {
		t.Fatalf("lanzar: %v", err)
	}
	resultado, err := gestor.Enviar(context.Background(), sesion.HandleRef, "devuelve patch")
	if err != nil {
		t.Fatalf("enviar: %v", err)
	}
	if !strings.Contains(resultado.Respuesta, "PATCH_UNIFICADO") {
		t.Fatalf("deberia haber usado thinking como fallback: %+v", resultado)
	}
}

func TestResolverContenidoRespuestaOllamaIgnoraThinkingSinSalidaValida(t *testing.T) {
	if got := resolverContenidoRespuestaOllama("", "razonamiento interno sin patch"); got != "" {
		t.Fatalf("no deberia filtrar thinking generico: %q", got)
	}
	if got := resolverContenidoRespuestaOllama("PATCH_UNIFICADO: ok", "PATCH_UNIFICADO: thinking"); got != "PATCH_UNIFICADO: ok" {
		t.Fatalf("deberia priorizar content: %q", got)
	}
}

func TestGestorRespetaSlotsPorPool(t *testing.T) {
	gestor := NuevoGestor("http://127.0.0.1:11434", &http.Client{})
	_, err := gestor.Lanzar(context.Background(), EntradaLanzamiento{
		Agente:       "Gemma1",
		PoolSlug:     "ollama-gemma4",
		SlotsMaximos: 1,
		Modelo:       "gemma4:26b",
	})
	if err != nil {
		t.Fatalf("primer launch: %v", err)
	}
	if _, err := gestor.Lanzar(context.Background(), EntradaLanzamiento{
		Agente:       "Gemma2",
		PoolSlug:     "ollama-gemma4",
		SlotsMaximos: 1,
		Modelo:       "gemma4:26b",
	}); err == nil {
		t.Fatal("esperaba error de slots agotados")
	}
}

func TestGestorBloqueaModeloActivoDistinto(t *testing.T) {
	gestor := NuevoGestor("http://127.0.0.1:11434", &http.Client{})
	if _, err := gestor.Lanzar(context.Background(), EntradaLanzamiento{
		Agente:       "Gemma1",
		PoolSlug:     "ollama-local",
		SlotsMaximos: 2,
		Modelo:       "gemma4:26b",
	}); err != nil {
		t.Fatalf("primer launch: %v", err)
	}
	if _, err := gestor.Lanzar(context.Background(), EntradaLanzamiento{
		Agente:       "Qwen1",
		PoolSlug:     "ollama-local",
		SlotsMaximos: 2,
		Modelo:       "qwen2.5-coder:7b",
	}); err == nil || !strings.Contains(err.Error(), "modelo ollama activo incompatible") {
		t.Fatalf("deberia bloquear segundo modelo distinto, err=%v", err)
	}
	if _, err := gestor.Lanzar(context.Background(), EntradaLanzamiento{
		Agente:       "Gemma2",
		PoolSlug:     "ollama-local",
		SlotsMaximos: 2,
		Modelo:       "gemma4:26b",
	}); err != nil {
		t.Fatalf("deberia permitir segunda sesion del mismo modelo: %v", err)
	}
}

func TestGestorCompactaContextoYConservaResumenBreve(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]any{
				"role":    "assistant",
				"content": "PATCH: listo",
			},
		})
	}))
	defer srv.Close()

	gestor := NuevoGestor(srv.URL, nil)
	sesion, err := gestor.Lanzar(context.Background(), EntradaLanzamiento{
		Agente:              "Gemma1",
		Proyecto:            "orquestador",
		PoolSlug:            "ollama-gemma4",
		SlotsMaximos:        1,
		Modelo:              "gemma4:26b",
		PerfilTarea:         "implementacion",
		Sistema:             "MICROTAREA CERRADA",
		ResumenContinuidad:  "Firma previa establecida",
		MaxMensajesContexto: 4,
	})
	if err != nil {
		t.Fatalf("lanzar: %v", err)
	}
	for i := 0; i < 4; i++ {
		if _, err := gestor.Enviar(context.Background(), sesion.HandleRef, "mensaje de trabajo numero "+strings.Repeat("x", 40)); err != nil {
			t.Fatalf("enviar %d: %v", i, err)
		}
	}
	estado, err := gestor.Estado(sesion.HandleRef)
	if err != nil {
		t.Fatalf("estado: %v", err)
	}
	if strings.TrimSpace(estado.ResumenContinuidad) == "" {
		t.Fatalf("faltaba resumen continuidad: %+v", estado)
	}
	if len(estado.Mensajes) > 6 {
		t.Fatalf("demasiados mensajes tras compactar: %d %+v", len(estado.Mensajes), estado.Mensajes)
	}
	found := false
	for _, msg := range estado.Mensajes {
		if strings.HasPrefix(strings.TrimSpace(msg.Contenido), "CONTINUIDAD BREVE:") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("faltaba mensaje system de continuidad: %+v", estado.Mensajes)
	}
}

func TestGestorDescribePoolResumeOcupacionYEstados(t *testing.T) {
	gestor := NuevoGestor("http://127.0.0.1:11434", &http.Client{})
	ready, err := gestor.Lanzar(context.Background(), EntradaLanzamiento{
		Agente:       "Gemma1",
		PoolSlug:     "ollama-gemma4",
		SlotsMaximos: 3,
		Modelo:       "gemma4:26b",
	})
	if err != nil {
		t.Fatalf("launch ready: %v", err)
	}
	failed, err := gestor.Lanzar(context.Background(), EntradaLanzamiento{
		Agente:       "Gemma2",
		PoolSlug:     "ollama-gemma4",
		SlotsMaximos: 3,
		Modelo:       "gemma4:26b",
	})
	if err != nil {
		t.Fatalf("launch failed: %v", err)
	}
	gestor.mu.Lock()
	gestor.sesiones[ready.HandleRef].Estado = "ready"
	gestor.sesiones[failed.HandleRef].Estado = "failed"
	gestor.sesiones[failed.HandleRef].ActualizadoEn = time.Now().UTC()
	gestor.mu.Unlock()
	telemetria := gestor.DescribirPool("ollama-gemma4")
	if telemetria == nil {
		t.Fatal("telemetria nil")
	}
	if telemetria.SlotsActivos != 1 || telemetria.SesionesLogicasActivas != 2 || telemetria.SesionesReady != 1 || telemetria.SesionesFailed != 1 {
		t.Fatalf("telemetria inesperada: %+v", telemetria)
	}
}
