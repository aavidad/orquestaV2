package db

import (
	"testing"
	"time"
)

func TestGuardarYListarPoliticasModelo(t *testing.T) {
	withTempDBPools(t, func() {
		if _, err := GuardarPool(&PoolCapacidad{
			Slug:                "codex",
			Proveedor:           "OpenAI",
			Runtime:             "codex",
			Plan:                "default",
			EsDePago:            true,
			CapacidadTotal:      4,
			CapacidadReservada:  0,
			PermiteHijos:        true,
			PermiteModelosMulti: true,
			PermiteSobrecoste:   false,
			PoliticaHandoff:     "preventivo",
			FuenteTelemetria:    "manual",
			MetadataJSON:        "{}",
			Activo:              true,
		}); err != nil {
			t.Fatalf("GuardarPool: %v", err)
		}

		id, err := GuardarPoliticaModelo(&PoliticaModelo{
			ScopeTipo:       "proyecto",
			ScopeRef:        "orquestador",
			PerfilTarea:     "implementacion",
			PoolSlug:        "codex",
			ReasoningEffort: "high",
			Prioridad:       20,
			Activa:          true,
		})
		if err != nil {
			t.Fatalf("GuardarPoliticaModelo: %v", err)
		}
		if id == 0 {
			t.Fatalf("id inesperado: %d", id)
		}

		items, err := ListarPoliticasModelo("proyecto", "orquestador", boolPtr(true))
		if err != nil {
			t.Fatalf("ListarPoliticasModelo: %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("numero de politicas inesperado: %+v", items)
		}
		if items[0].PoolSlug != "codex" || items[0].ReasoningEffort != "high" {
			t.Fatalf("politica inesperada: %+v", items[0])
		}
	})
}

func TestGuardarPoliticaModeloAceptaScopeAgente(t *testing.T) {
	withTempDBPools(t, func() {
		if _, err := GuardarPool(&PoolCapacidad{
			Slug:                "google",
			Proveedor:           "Google",
			Runtime:             "gemini",
			Plan:                "default",
			EsDePago:            true,
			CapacidadTotal:      1,
			CapacidadReservada:  0,
			PermiteHijos:        true,
			PermiteModelosMulti: true,
			PermiteSobrecoste:   false,
			PoliticaHandoff:     "preventivo",
			FuenteTelemetria:    "manual",
			MetadataJSON:        "{}",
			Activo:              true,
		}); err != nil {
			t.Fatalf("GuardarPool google: %v", err)
		}
		id, err := GuardarPoliticaModelo(&PoliticaModelo{
			ScopeTipo:       "agente",
			ScopeRef:        "Gemini1",
			PerfilTarea:     "implementacion",
			PoolSlug:        "google",
			ModelSlug:       "gemini-2.5-flash-lite",
			ReasoningEffort: "medium",
			Prioridad:       5,
			Activa:          true,
		})
		if err != nil {
			t.Fatalf("GuardarPoliticaModelo agente: %v", err)
		}
		if id == 0 {
			t.Fatalf("id inesperado: %d", id)
		}
		items, err := ListarPoliticasModelo("agente", "Gemini1", boolPtr(true))
		if err != nil {
			t.Fatalf("ListarPoliticasModelo agente: %v", err)
		}
		if len(items) != 1 || items[0].ModelSlug != "gemini-2.5-flash-lite" {
			t.Fatalf("politica agente inesperada: %+v", items)
		}
	})
}

func TestResolverPoliticaModeloConOverridesPorProyectoYTarea(t *testing.T) {
	withTempDBPools(t, func() {
		insertPoolsYModelosTest(t)

		if _, err := GuardarPoliticaModelo(&PoliticaModelo{
			ScopeTipo:       "perfil",
			ScopeRef:        "implementacion",
			PerfilTarea:     "implementacion",
			PoolSlug:        "codex",
			ReasoningEffort: "high",
			Prioridad:       10,
			Activa:          true,
		}); err != nil {
			t.Fatalf("politica perfil: %v", err)
		}
		if _, err := GuardarPoliticaModelo(&PoliticaModelo{
			ScopeTipo:   "proyecto",
			ScopeRef:    "orquestador",
			PerfilTarea: "implementacion",
			ModelSlug:   "gpt-5.4",
			Prioridad:   20,
			Activa:      true,
		}); err != nil {
			t.Fatalf("politica proyecto: %v", err)
		}
		if _, err := GuardarPoliticaModelo(&PoliticaModelo{
			ScopeTipo:       "tarea",
			ScopeRef:        "154",
			PerfilTarea:     "implementacion",
			ModelSlug:       "gpt-5.4-mini",
			ReasoningEffort: "medium",
			Prioridad:       5,
			Activa:          true,
		}); err != nil {
			t.Fatalf("politica tarea: %v", err)
		}

		tareaID := int64(154)
		res, err := ResolverPoliticaModelo(ResolverPoliticaInput{
			TareaID:      &tareaID,
			ProyectoSlug: "orquestador",
			PerfilTarea:  "implementacion",
		})
		if err != nil {
			t.Fatalf("ResolverPoliticaModelo: %v", err)
		}
		if res.PoolSlug != "codex" {
			t.Fatalf("pool inesperado: %+v", res)
		}
		if res.ModelSlug != "gpt-5.4-mini" {
			t.Fatalf("modelo inesperado: %+v", res)
		}
		if res.ReasoningEffort != "medium" {
			t.Fatalf("reasoning inesperado: %+v", res)
		}
		if len(res.PoliticasAplicadas) != 3 {
			t.Fatalf("politicas aplicadas inesperadas: %+v", res.PoliticasAplicadas)
		}
	})
}

func TestResolverPoliticaModeloEconomicaPorPerfil(t *testing.T) {
	withTempDBPools(t, func() {
		insertPoolsYModelosTest(t)

		if err := SeedPoliticasModeloIniciales(); err != nil {
			t.Fatalf("SeedPoliticasModeloIniciales: %v", err)
		}

		res, err := ResolverPoliticaModelo(ResolverPoliticaInput{
			PerfilTarea: "script",
		})
		if err != nil {
			t.Fatalf("ResolverPoliticaModelo: %v", err)
		}
		if res.ModelSlug != "android-shell" {
			t.Fatalf("modelo economico inesperado: %+v", res)
		}
		if res.ReasoningEffort != "medium" {
			t.Fatalf("reasoning inesperado: %+v", res)
		}
	})
}

func TestSeedPoliticasModeloInicialesUsaReasoningProfesionalPorPerfil(t *testing.T) {
	withTempDBPools(t, func() {
		insertPoolsYModelosTest(t)

		if err := SeedPoliticasModeloIniciales(); err != nil {
			t.Fatalf("SeedPoliticasModeloIniciales: %v", err)
		}

		casos := []struct {
			perfil      string
			razonamiento string
		}{
			{perfil: "orquestacion", razonamiento: "high"},
			{perfil: "analisis", razonamiento: "high"},
			{perfil: "implementacion", razonamiento: "high"},
			{perfil: "revision", razonamiento: "high"},
			{perfil: "script", razonamiento: "medium"},
			{perfil: "handoff", razonamiento: "medium"},
		}

		for _, tc := range casos {
			res, err := ResolverPoliticaModelo(ResolverPoliticaInput{PerfilTarea: tc.perfil})
			if err != nil {
				t.Fatalf("ResolverPoliticaModelo(%s): %v", tc.perfil, err)
			}
			if res.ReasoningEffort != tc.razonamiento {
				t.Fatalf("reasoning inesperado para %s: got=%s want=%s", tc.perfil, res.ReasoningEffort, tc.razonamiento)
			}
		}
	})
}

func TestEnsureCapacidadModeloBaseCodexFuerzaHighEnImplementacion(t *testing.T) {
	withTempDBPools(t, func() {
		if err := SeedPoolsIniciales(); err != nil {
			t.Fatalf("SeedPoolsIniciales: %v", err)
		}
		if err := SeedModelosIniciales(); err != nil {
			t.Fatalf("SeedModelosIniciales: %v", err)
		}
		if err := SeedPoliticasModeloIniciales(); err != nil {
			t.Fatalf("SeedPoliticasModeloIniciales: %v", err)
		}
		if _, err := DB.Exec(`
			UPDATE politicas_modelo
			SET reasoning_effort = 'high'
			WHERE scope_tipo = 'perfil'
			  AND scope_ref = 'implementacion'
			  AND perfil_tarea = 'implementacion'`); err != nil {
			t.Fatalf("downgrade politica implementacion: %v", err)
		}

		if err := EnsureCapacidadModeloBaseCodex(); err != nil {
			t.Fatalf("EnsureCapacidadModeloBaseCodex: %v", err)
		}

		res, err := ResolverPoliticaModelo(ResolverPoliticaInput{PerfilTarea: "implementacion"})
		if err != nil {
			t.Fatalf("ResolverPoliticaModelo: %v", err)
		}
		if res.ModelSlug != "gpt-5.4" {
			t.Fatalf("modelo inesperado: %+v", res)
		}
		if res.ReasoningEffort != "high" {
			t.Fatalf("reasoning inesperado: %+v", res)
		}
	})
}

func TestEnsureCapacidadModeloBaseCodexMemoizaPorStorageTarget(t *testing.T) {
	withTempDBPools(t, func() {
		ensureCapacidadModeloBaseCodexMu.Lock()
		prevDone := ensureCapacidadModeloBaseCodexDone
		ensureCapacidadModeloBaseCodexDone = map[string]bool{}
		ensureCapacidadModeloBaseCodexMu.Unlock()

		prevPools := ensureCapacidadModeloBaseSeedPoolsFn
		prevModels := ensureCapacidadModeloBaseSeedModelsFn
		prevPolicies := ensureCapacidadModeloBaseSeedPoliciesFn
		prevEnsure := ensureCapacidadModeloBaseEnsureImplementationFn
		t.Cleanup(func() {
			ensureCapacidadModeloBaseSeedPoolsFn = prevPools
			ensureCapacidadModeloBaseSeedModelsFn = prevModels
			ensureCapacidadModeloBaseSeedPoliciesFn = prevPolicies
			ensureCapacidadModeloBaseEnsureImplementationFn = prevEnsure
			ensureCapacidadModeloBaseCodexMu.Lock()
			ensureCapacidadModeloBaseCodexDone = prevDone
			ensureCapacidadModeloBaseCodexMu.Unlock()
		})

		poolsCalls := 0
		modelsCalls := 0
		policiesCalls := 0
		ensureCalls := 0
		ensureCapacidadModeloBaseSeedPoolsFn = func() error {
			poolsCalls++
			return nil
		}
		ensureCapacidadModeloBaseSeedModelsFn = func() error {
			modelsCalls++
			return nil
		}
		ensureCapacidadModeloBaseSeedPoliciesFn = func() error {
			policiesCalls++
			return nil
		}
		ensureCapacidadModeloBaseEnsureImplementationFn = func() error {
			ensureCalls++
			return nil
		}

		if err := EnsureCapacidadModeloBaseCodex(); err != nil {
			t.Fatalf("primer EnsureCapacidadModeloBaseCodex: %v", err)
		}
		if err := EnsureCapacidadModeloBaseCodex(); err != nil {
			t.Fatalf("segundo EnsureCapacidadModeloBaseCodex: %v", err)
		}
		if poolsCalls != 1 || modelsCalls != 1 || policiesCalls != 1 || ensureCalls != 1 {
			t.Fatalf("memoizacion inesperada pools=%d models=%d policies=%d ensure=%d", poolsCalls, modelsCalls, policiesCalls, ensureCalls)
		}
	})
}

func TestResolverPerfilEjecucionLanzamientoMemoizaResultado(t *testing.T) {
	prevEnsure := resolverPerfilEnsureBaseFn
	prevFase := resolverPerfilObtenerFaseFn
	prevResolver := resolverPerfilResolverPoliticaFn
	prevTTL := resolverPerfilEjecucionCacheTTL
	t.Cleanup(func() {
		resolverPerfilEnsureBaseFn = prevEnsure
		resolverPerfilObtenerFaseFn = prevFase
		resolverPerfilResolverPoliticaFn = prevResolver
		resolverPerfilEjecucionCacheTTL = prevTTL
		resetResolverPerfilEjecucionLanzamientoCache()
	})

	resetResolverPerfilEjecucionLanzamientoCache()
	resolverPerfilEjecucionCacheTTL = time.Minute

	ensureCalls := 0
	faseCalls := 0
	resolverCalls := 0
	resolverPerfilEnsureBaseFn = func() error {
		ensureCalls++
		return nil
	}
	resolverPerfilObtenerFaseFn = func(proyecto string) (string, error) {
		faseCalls++
		return "implementacion", nil
	}
	resolverPerfilResolverPoliticaFn = func(input ResolverPoliticaInput) (*ResolucionModelo, error) {
		resolverCalls++
		return &ResolucionModelo{
			PerfilTarea:     "implementacion",
			ModelSlug:       "gpt-5.4",
			ReasoningEffort: "high",
		}, nil
	}

	agente := "Codex2"
	for i := 0; i < 2; i++ {
		perfil, modelo, razonamiento, err := ResolverPerfilEjecucionLanzamiento(&agente, "orquestador", "", "", "")
		if err != nil {
			t.Fatalf("ResolverPerfilEjecucionLanzamiento: %v", err)
		}
		if perfil != "implementacion" || modelo != "gpt-5.4" || razonamiento != "high" {
			t.Fatalf("resultado inesperado perfil=%s modelo=%s razonamiento=%s", perfil, modelo, razonamiento)
		}
	}

	if ensureCalls != 1 {
		t.Fatalf("ensureCalls=%d want=1", ensureCalls)
	}
	if resolverCalls != 1 {
		t.Fatalf("resolverCalls=%d want=1", resolverCalls)
	}
	if faseCalls != 2 {
		t.Fatalf("faseCalls=%d want=2", faseCalls)
	}
}

func insertPoolsYModelosTest(t *testing.T) {
	t.Helper()
	for _, pool := range []PoolCapacidad{
		{
			Slug:                "codex",
			Proveedor:           "OpenAI",
			Runtime:             "codex",
			Plan:                "default",
			EsDePago:            true,
			CapacidadTotal:      4,
			CapacidadReservada:  0,
			PermiteHijos:        true,
			PermiteModelosMulti: true,
			PermiteSobrecoste:   false,
			PoliticaHandoff:     "preventivo",
			FuenteTelemetria:    "manual",
			MetadataJSON:        "{}",
			Activo:              true,
		},
		{
			Slug:                "android",
			Proveedor:           "Android",
			Runtime:             "android",
			Plan:                "default",
			EsDePago:            false,
			CapacidadTotal:      1,
			CapacidadReservada:  0,
			PermiteHijos:        false,
			PermiteModelosMulti: false,
			PermiteSobrecoste:   false,
			PoliticaHandoff:     "preventivo",
			FuenteTelemetria:    "manual",
			MetadataJSON:        "{}",
			Activo:              true,
		},
	} {
		pool := pool
		if _, err := GuardarPool(&pool); err != nil {
			t.Fatalf("GuardarPool %s: %v", pool.Slug, err)
		}
	}
	for _, item := range []struct {
		poolSlug string
		model    PoolModelo
	}{
		{poolSlug: "codex", model: PoolModelo{ModelSlug: "gpt-5.4", Activo: true, Prioridad: 10, CosteRelativo: 1.5, LimiteConocidoJSON: "{}"}},
		{poolSlug: "codex", model: PoolModelo{ModelSlug: "gpt-5.4-mini", Activo: true, Prioridad: 20, CosteRelativo: 0.5, LimiteConocidoJSON: "{}"}},
		{poolSlug: "android", model: PoolModelo{ModelSlug: "android-shell", Activo: true, Prioridad: 30, CosteRelativo: 0.1, LimiteConocidoJSON: "{}"}},
	} {
		model := item.model
		if _, err := GuardarPoolModelo(item.poolSlug, &model); err != nil {
			t.Fatalf("GuardarPoolModelo %s/%s: %v", item.poolSlug, model.ModelSlug, err)
		}
	}
}
