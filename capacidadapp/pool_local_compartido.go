package capacidadapp

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"orquesta/db"
)

type PerfilPoolLocal struct {
	PerfilTarea       string `json:"perfil_tarea"`
	ModelSlug         string `json:"model_slug"`
	ReasoningEffort   string `json:"reasoning_effort"`
	PrioridadPolitica int    `json:"prioridad_politica"`
}

type TelemetriaPoolLocal struct {
	PoolSlug               string     `json:"pool_slug"`
	SlotsMaximos           int        `json:"slots_maximos"`
	SlotsActivos           int        `json:"slots_activos"`
	SesionesLogicasActivas int        `json:"sesiones_logicas_activas"`
	SesionesReady          int        `json:"sesiones_ready"`
	SesionesWorking        int        `json:"sesiones_working"`
	SesionesFailed         int        `json:"sesiones_failed"`
	ActualizadoEn          *time.Time `json:"actualizado_en,omitempty"`
}

type PoolLocalCompartido struct {
	PoolSlug               string               `json:"pool_slug"`
	Proveedor              string               `json:"proveedor"`
	Runtime                string               `json:"runtime"`
	ModeloPreferente       string               `json:"modelo_preferente"`
	SlotsMaximos           int                  `json:"slots_maximos"`
	ConectorCanonico       string               `json:"conector_canonico"`
	ConectorCompatibilidad string               `json:"conector_compatibilidad"`
	ExperimentalCompat     bool                 `json:"experimental_compat"`
	Perfiles               []PerfilPoolLocal    `json:"perfiles"`
	Telemetria             *TelemetriaPoolLocal `json:"telemetria,omitempty"`
	Pool                   *db.PoolCapacidad    `json:"pool,omitempty"`
	Modelos                []*db.PoolModelo     `json:"modelos,omitempty"`
	PoliticasAplicadas     []*db.PoliticaModelo `json:"politicas_aplicadas,omitempty"`
}

type EntradaAsegurarPoolLocalCompartido struct {
	PoolSlug               string
	Proveedor              string
	Runtime                string
	ModeloPreferente       string
	SlotsMaximos           int
	ConectorCanonico       string
	ConectorCompatibilidad string
	ExperimentalCompat     bool
	Perfiles               []PerfilPoolLocal
}

func (s *Service) DescribirPoolLocalCompartido(slug string) (*PoolLocalCompartido, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, fmt.Errorf("pool obligatorio")
	}
	detail, err := s.GetPoolDetail(slug)
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.Pool == nil {
		return nil, fmt.Errorf("pool %q no disponible", slug)
	}
	meta, err := parsePoolMetadata(detail.Pool.MetadataJSON)
	if err != nil {
		return nil, fmt.Errorf("metadata_json invalido en pool %q: %w", slug, err)
	}
	politicas, err := s.collectProfilePolicies(slug)
	if err != nil {
		return nil, err
	}
	perfiles := perfilesDesdePoliticas(politicas)
	modeloPreferente := strings.TrimSpace(metadataString(meta, "modelo_preferente"))
	if modeloPreferente == "" && len(detail.Modelos) > 0 && detail.Modelos[0] != nil {
		modeloPreferente = strings.TrimSpace(detail.Modelos[0].ModelSlug)
	}
	var telemetria *TelemetriaPoolLocal
	if s.poolLocalProvider != nil {
		telemetria, err = s.poolLocalProvider.DescribirPoolLocalCompartido(slug)
		if err != nil {
			return nil, err
		}
		if telemetria != nil && telemetria.SlotsMaximos <= 0 {
			telemetria.SlotsMaximos = metadataInt(meta, "slots_maximos", detail.Pool.CapacidadTotal)
		}
	}
	return &PoolLocalCompartido{
		PoolSlug:               strings.TrimSpace(detail.Pool.Slug),
		Proveedor:              strings.TrimSpace(detail.Pool.Proveedor),
		Runtime:                strings.TrimSpace(detail.Pool.Runtime),
		ModeloPreferente:       modeloPreferente,
		SlotsMaximos:           metadataInt(meta, "slots_maximos", detail.Pool.CapacidadTotal),
		ConectorCanonico:       metadataStringDefault(meta, "conector_canonico", "ollama_pool_local"),
		ConectorCompatibilidad: metadataStringDefault(meta, "conector_compatibilidad", "ollama-cli"),
		ExperimentalCompat:     metadataBoolDefault(meta, "experimental_compat", true),
		Perfiles:               perfiles,
		Telemetria:             telemetria,
		Pool:                   detail.Pool,
		Modelos:                detail.Modelos,
		PoliticasAplicadas:     politicas,
	}, nil
}

func (s *Service) AsegurarPoolLocalCompartido(entrada EntradaAsegurarPoolLocalCompartido) (*PoolLocalCompartido, error) {
	normalizada, err := normalizarEntradaPoolLocalCompartido(entrada)
	if err != nil {
		return nil, err
	}
	meta := map[string]any{
		"slots_maximos":           normalizada.SlotsMaximos,
		"conector_canonico":       normalizada.ConectorCanonico,
		"conector_compatibilidad": normalizada.ConectorCompatibilidad,
		"experimental_compat":     normalizada.ExperimentalCompat,
		"modelo_preferente":       normalizada.ModeloPreferente,
		"perfiles_tarea":          perfilesComoStrings(normalizada.Perfiles),
	}
	rawMeta, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("serializando metadata del pool local: %w", err)
	}
	pool := &db.PoolCapacidad{
		Slug:                normalizada.PoolSlug,
		Proveedor:           normalizada.Proveedor,
		Runtime:             normalizada.Runtime,
		Plan:                "local",
		EsDePago:            false,
		CapacidadTotal:      normalizada.SlotsMaximos,
		CapacidadReservada:  0,
		PermiteHijos:        false,
		PermiteModelosMulti: true,
		PermiteSobrecoste:   false,
		PoliticaHandoff:     "preventivo",
		FuenteTelemetria:    "manual",
		MetadataJSON:        string(rawMeta),
		Activo:              true,
	}
	if _, err := s.SavePool(pool); err != nil {
		return nil, err
	}
	if _, err := s.SavePoolModel(normalizada.PoolSlug, &db.PoolModelo{
		ModelSlug:          normalizada.ModeloPreferente,
		Activo:             true,
		Prioridad:          10,
		CosteRelativo:      1.0,
		LimiteConocidoJSON: "{}",
	}); err != nil {
		return nil, err
	}
	for _, perfil := range normalizada.Perfiles {
		if _, err := s.SaveModelPolicy(&db.PoliticaModelo{
			ScopeTipo:       "perfil",
			ScopeRef:        perfil.PerfilTarea,
			PerfilTarea:     perfil.PerfilTarea,
			PoolSlug:        normalizada.PoolSlug,
			ModelSlug:       perfil.ModelSlug,
			ReasoningEffort: perfil.ReasoningEffort,
			Prioridad:       perfil.PrioridadPolitica,
			Activa:          true,
			MetadataJSON:    `{"fuente":"pool_local_compartido"}`,
		}); err != nil {
			return nil, err
		}
	}
	return s.DescribirPoolLocalCompartido(normalizada.PoolSlug)
}

func normalizarEntradaPoolLocalCompartido(entrada EntradaAsegurarPoolLocalCompartido) (EntradaAsegurarPoolLocalCompartido, error) {
	entrada.PoolSlug = strings.TrimSpace(entrada.PoolSlug)
	entrada.Proveedor = strings.TrimSpace(entrada.Proveedor)
	entrada.Runtime = strings.TrimSpace(entrada.Runtime)
	entrada.ModeloPreferente = strings.TrimSpace(entrada.ModeloPreferente)
	entrada.ConectorCanonico = strings.TrimSpace(entrada.ConectorCanonico)
	entrada.ConectorCompatibilidad = strings.TrimSpace(entrada.ConectorCompatibilidad)
	if entrada.PoolSlug == "" {
		return entrada, fmt.Errorf("pool_slug obligatorio")
	}
	if entrada.Proveedor == "" {
		entrada.Proveedor = "Ollama"
	}
	if entrada.Runtime == "" {
		entrada.Runtime = "ollama"
	}
	if entrada.ModeloPreferente == "" {
		return entrada, fmt.Errorf("modelo_preferente obligatorio")
	}
	if entrada.SlotsMaximos <= 0 {
		entrada.SlotsMaximos = 1
	}
	if entrada.ConectorCanonico == "" {
		entrada.ConectorCanonico = "ollama_pool_local"
	}
	if entrada.ConectorCompatibilidad == "" {
		entrada.ConectorCompatibilidad = "ollama-cli"
	}
	if len(entrada.Perfiles) == 0 {
		entrada.Perfiles = []PerfilPoolLocal{
			{PerfilTarea: "implementacion", ModelSlug: entrada.ModeloPreferente, ReasoningEffort: "high", PrioridadPolitica: 10},
			{PerfilTarea: "revision", ModelSlug: entrada.ModeloPreferente, ReasoningEffort: "high", PrioridadPolitica: 10},
			{PerfilTarea: "analisis", ModelSlug: entrada.ModeloPreferente, ReasoningEffort: "high", PrioridadPolitica: 10},
		}
	}
	for i := range entrada.Perfiles {
		entrada.Perfiles[i].PerfilTarea = strings.TrimSpace(entrada.Perfiles[i].PerfilTarea)
		entrada.Perfiles[i].ModelSlug = strings.TrimSpace(entrada.Perfiles[i].ModelSlug)
		entrada.Perfiles[i].ReasoningEffort = strings.TrimSpace(strings.ToLower(entrada.Perfiles[i].ReasoningEffort))
		if entrada.Perfiles[i].PerfilTarea == "" {
			return entrada, fmt.Errorf("perfil_tarea obligatorio en perfiles")
		}
		if entrada.Perfiles[i].ModelSlug == "" {
			entrada.Perfiles[i].ModelSlug = entrada.ModeloPreferente
		}
		if entrada.Perfiles[i].ReasoningEffort == "" {
			entrada.Perfiles[i].ReasoningEffort = "high"
		}
		if entrada.Perfiles[i].PrioridadPolitica <= 0 {
			entrada.Perfiles[i].PrioridadPolitica = 10
		}
	}
	sort.SliceStable(entrada.Perfiles, func(i, j int) bool {
		return entrada.Perfiles[i].PerfilTarea < entrada.Perfiles[j].PerfilTarea
	})
	return entrada, nil
}

func (s *Service) collectProfilePolicies(poolSlug string) ([]*db.PoliticaModelo, error) {
	var result []*db.PoliticaModelo
	for _, perfil := range []string{"implementacion", "revision", "analisis"} {
		items, err := s.store.ListModelPolicies("perfil", perfil, boolPtr(true))
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if item == nil || strings.TrimSpace(item.PoolSlug) != poolSlug {
				continue
			}
			result = append(result, item)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].PerfilTarea == result[j].PerfilTarea {
			return result[i].Prioridad < result[j].Prioridad
		}
		return result[i].PerfilTarea < result[j].PerfilTarea
	})
	return result, nil
}

func perfilesDesdePoliticas(items []*db.PoliticaModelo) []PerfilPoolLocal {
	out := make([]PerfilPoolLocal, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, PerfilPoolLocal{
			PerfilTarea:       strings.TrimSpace(item.PerfilTarea),
			ModelSlug:         strings.TrimSpace(item.ModelSlug),
			ReasoningEffort:   strings.TrimSpace(item.ReasoningEffort),
			PrioridadPolitica: item.Prioridad,
		})
	}
	return out
}

func perfilesComoStrings(items []PerfilPoolLocal) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if perfil := strings.TrimSpace(item.PerfilTarea); perfil != "" {
			out = append(out, perfil)
		}
	}
	return out
}

func parsePoolMetadata(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}, nil
	}
	var meta map[string]any
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return nil, err
	}
	if meta == nil {
		meta = map[string]any{}
	}
	return meta, nil
}

func metadataString(meta map[string]any, key string) string {
	if meta == nil {
		return ""
	}
	value, _ := meta[key].(string)
	return strings.TrimSpace(value)
}

func metadataStringDefault(meta map[string]any, key, fallback string) string {
	if value := metadataString(meta, key); value != "" {
		return value
	}
	return fallback
}

func metadataBoolDefault(meta map[string]any, key string, fallback bool) bool {
	if meta == nil {
		return fallback
	}
	value, ok := meta[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case bool:
		return typed
	default:
		return fallback
	}
}

func metadataInt(meta map[string]any, key string, fallback int) int {
	if meta == nil {
		return fallback
	}
	value, ok := meta[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case float64:
		if int(typed) > 0 {
			return int(typed)
		}
	case int:
		if typed > 0 {
			return typed
		}
	}
	return fallback
}

func boolPtr(v bool) *bool {
	return &v
}
