/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"orquesta/capacidadapp"
	"orquesta/db"
)

func TestCapacidadUsaAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/pools", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(apiPoolsResponse{
				Pools: []*db.PoolCapacidadResumen{
					{
						Pool: &db.PoolCapacidad{
							Slug:           "codex",
							Proveedor:      "OpenAI",
							Runtime:        "codex",
							Plan:           "default",
							CapacidadTotal: 4,
						},
						SesionesActivas:     1,
						CapacidadDisponible: 3,
					},
				},
			})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(apiPoolSaveResponse{ID: 7})
		default:
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/pools/codex", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiPoolResponse{
			Detalle: &capacidadapp.PoolDetail{
				Pool: &db.PoolCapacidad{
					Slug:               "codex",
					Proveedor:          "OpenAI",
					Runtime:            "codex",
					Plan:               "default",
					CapacidadTotal:     4,
					CapacidadReservada: 1,
					PoliticaHandoff:    "preventivo",
					FuenteTelemetria:   "manual",
					Activo:             true,
				},
				SesionesActivas:     1,
				CapacidadDisponible: 3,
				Modelos: []*db.PoolModelo{
					{ModelSlug: "gpt-5.4", Prioridad: 10, CosteRelativo: 1.5, Activo: true},
				},
			},
		})
	})
	mux.HandleFunc("/api/pools/codex/modelos", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(apiPoolModelosResponse{
				Modelos: []*db.PoolModelo{
					{ModelSlug: "gpt-5.4", Prioridad: 10, CosteRelativo: 1.5, Activo: true},
				},
			})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(apiPoolSaveResponse{ID: 11})
		default:
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/pools/seed-inicial", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/pools/modelos/seed-inicial", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/politicas-modelo", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(apiPoliticasModeloResponse{
				Politicas: []*db.PoliticaModelo{
					{ScopeTipo: "perfil", ScopeRef: "programador", PerfilTarea: "programador", PoolSlug: "codex", ModelSlug: "gpt-5.4", Prioridad: 10, ReasoningEffort: "high", Activa: true},
				},
			})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(apiPoliticaModeloSaveResponse{ID: 13})
		default:
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/politicas-modelo/seed-inicial", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/modelo/resolver", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiResolucionModeloResponse{
			Resolucion: &db.ResolucionModelo{
				PerfilTarea:     "programador",
				PoolSlug:        "codex",
				ModelSlug:       "gpt-5.4",
				ReasoningEffort: "high",
				FuentePool:      "politica",
				FuenteModelo:    "politica",
				FuenteReasoning: "politica",
			},
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	outListar := capturarStdout(t, func() {
		if err := poolListarCmd.RunE(poolListarCmd, nil); err != nil {
			t.Fatalf("pool listar via api: %v", err)
		}
	})
	if !strings.Contains(outListar, "codex") {
		t.Fatalf("salida pool listar sin codex:\n%s", outListar)
	}

	outVer := capturarStdout(t, func() {
		if err := poolVerCmd.RunE(poolVerCmd, []string{"codex"}); err != nil {
			t.Fatalf("pool ver via api: %v", err)
		}
	})
	for _, token := range []string{"Pool codex", "gpt-5.4", "OpenAI"} {
		if !strings.Contains(outVer, token) {
			t.Fatalf("salida pool ver sin %q:\n%s", token, outVer)
		}
	}

	if err := poolGuardarCmd.Flags().Set("proveedor", "OpenAI"); err != nil {
		t.Fatalf("set proveedor: %v", err)
	}
	if err := poolGuardarCmd.Flags().Set("runtime", "codex"); err != nil {
		t.Fatalf("set runtime: %v", err)
	}
	outGuardar := capturarStdout(t, func() {
		if err := poolGuardarCmd.RunE(poolGuardarCmd, []string{"codex"}); err != nil {
			t.Fatalf("pool guardar via api: %v", err)
		}
	})
	if !strings.Contains(outGuardar, "(id: 7)") {
		t.Fatalf("salida pool guardar sin id remoto:\n%s", outGuardar)
	}

	outPoliticas := capturarStdout(t, func() {
		if err := politicaModeloListarCmd.RunE(politicaModeloListarCmd, nil); err != nil {
			t.Fatalf("politica modelo listar via api: %v", err)
		}
	})
	if !strings.Contains(outPoliticas, "programador") {
		t.Fatalf("salida politica listar sin datos remotos:\n%s", outPoliticas)
	}

	if err := politicaModeloGuardarCmd.Flags().Set("scope-tipo", "perfil"); err != nil {
		t.Fatalf("set scope-tipo: %v", err)
	}
	if err := politicaModeloGuardarCmd.Flags().Set("scope-ref", "programador"); err != nil {
		t.Fatalf("set scope-ref: %v", err)
	}
	if err := politicaModeloGuardarCmd.Flags().Set("perfil", "programador"); err != nil {
		t.Fatalf("set perfil: %v", err)
	}
	if err := politicaModeloGuardarCmd.Flags().Set("pool", "codex"); err != nil {
		t.Fatalf("set pool: %v", err)
	}
	if err := politicaModeloGuardarCmd.Flags().Set("modelo", "gpt-5.4"); err != nil {
		t.Fatalf("set modelo: %v", err)
	}
	if err := politicaModeloGuardarCmd.Flags().Set("reasoning", "high"); err != nil {
		t.Fatalf("set reasoning: %v", err)
	}
	outPoliticaGuardar := capturarStdout(t, func() {
		if err := politicaModeloGuardarCmd.RunE(politicaModeloGuardarCmd, nil); err != nil {
			t.Fatalf("politica modelo guardar via api: %v", err)
		}
	})
	if !strings.Contains(outPoliticaGuardar, "(id: 13)") {
		t.Fatalf("salida politica guardar sin id remoto:\n%s", outPoliticaGuardar)
	}

	if err := modeloResolverCmd.Flags().Set("perfil", "programador"); err != nil {
		t.Fatalf("set perfil resolver: %v", err)
	}
	outResolver := capturarStdout(t, func() {
		if err := modeloResolverCmd.RunE(modeloResolverCmd, nil); err != nil {
			t.Fatalf("modelo resolver via api: %v", err)
		}
	})
	for _, token := range []string{"Pool:        codex", "Modelo:      gpt-5.4", "Reasoning:   high"} {
		if !strings.Contains(outResolver, token) {
			t.Fatalf("salida resolver sin %q:\n%s", token, outResolver)
		}
	}
}

func TestCapacidadExigeServidorSalvoRecuperacionLocal(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()

	resetCommandFlags(poolListarCmd)
	if err := poolListarCmd.RunE(poolListarCmd, nil); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("pool listar deberia exigir servidor, err=%v", err)
	}

	resetCommandFlags(poolVerCmd)
	if err := poolVerCmd.RunE(poolVerCmd, []string{"codex"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("pool ver deberia exigir servidor, err=%v", err)
	}

	resetCommandFlags(poolGuardarCmd)
	if err := poolGuardarCmd.RunE(poolGuardarCmd, []string{"codex"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("pool guardar deberia exigir servidor, err=%v", err)
	}

	resetCommandFlags(poolModeloListarCmd)
	if err := poolModeloListarCmd.RunE(poolModeloListarCmd, []string{"codex"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("pool modelo listar deberia exigir servidor, err=%v", err)
	}

	resetCommandFlags(poolModeloGuardarCmd)
	if err := poolModeloGuardarCmd.RunE(poolModeloGuardarCmd, []string{"codex", "gpt-5.4"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("pool modelo guardar deberia exigir servidor, err=%v", err)
	}

	resetCommandFlags(poolModeloSeedCmd)
	if err := poolModeloSeedCmd.RunE(poolModeloSeedCmd, nil); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("pool modelo seed deberia exigir servidor, err=%v", err)
	}

	resetCommandFlags(poolSeedCmd)
	if err := poolSeedCmd.RunE(poolSeedCmd, nil); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("pool seed deberia exigir servidor, err=%v", err)
	}

	resetCommandFlags(politicaModeloListarCmd)
	if err := politicaModeloListarCmd.RunE(politicaModeloListarCmd, nil); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("politica modelo listar deberia exigir servidor, err=%v", err)
	}

	resetCommandFlags(politicaModeloGuardarCmd)
	if err := politicaModeloGuardarCmd.RunE(politicaModeloGuardarCmd, nil); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("politica modelo guardar deberia exigir servidor, err=%v", err)
	}

	resetCommandFlags(politicaModeloSeedCmd)
	if err := politicaModeloSeedCmd.RunE(politicaModeloSeedCmd, nil); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("politica modelo seed deberia exigir servidor, err=%v", err)
	}

	resetCommandFlags(modeloResolverCmd)
	if err := modeloResolverCmd.RunE(modeloResolverCmd, nil); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("modelo resolver deberia exigir servidor, err=%v", err)
	}
}
