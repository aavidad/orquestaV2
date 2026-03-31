/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type PoliticaModelo struct {
	ID              int64
	ScopeTipo       string
	ScopeRef        string
	PerfilTarea     string
	PoolSlug        string
	ModelSlug       string
	ReasoningEffort string
	Prioridad       int
	Activa          bool
	MetadataJSON    string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ResolverPoliticaInput struct {
	TareaID      *int64
	ProyectoSlug string
	Fase         string
	PerfilTarea  string
}

type ResolucionModelo struct {
	PerfilTarea        string            `json:"perfil_tarea"`
	PoolSlug           string            `json:"pool_slug"`
	ModelSlug          string            `json:"model_slug"`
	ReasoningEffort    string            `json:"reasoning_effort"`
	FuentePool         string            `json:"fuente_pool"`
	FuenteModelo       string            `json:"fuente_modelo"`
	FuenteReasoning    string            `json:"fuente_reasoning"`
	PoliticasAplicadas []*PoliticaModelo `json:"politicas_aplicadas,omitempty"`
}

func ResolverPerfilEjecucionLanzamiento(proyectoSlug, perfilTarea, modelo, razonamiento string) (string, string, string, error) {
	perfilTarea = strings.TrimSpace(perfilTarea)
	modelo = strings.TrimSpace(modelo)
	razonamiento = strings.TrimSpace(strings.ToLower(razonamiento))
	if perfilTarea != "" && modelo != "" && razonamiento != "" {
		return perfilTarea, modelo, razonamiento, nil
	}
	resolucion, err := ResolverPoliticaModelo(ResolverPoliticaInput{
		ProyectoSlug: strings.TrimSpace(proyectoSlug),
		PerfilTarea:  perfilTarea,
	})
	if err != nil {
		return "", "", "", err
	}
	if perfilTarea == "" {
		perfilTarea = strings.TrimSpace(resolucion.PerfilTarea)
	}
	if modelo == "" {
		modelo = strings.TrimSpace(resolucion.ModelSlug)
	}
	if razonamiento == "" {
		razonamiento = strings.TrimSpace(strings.ToLower(resolucion.ReasoningEffort))
	}
	return perfilTarea, modelo, razonamiento, nil
}

func EnsureCapacidadModeloBaseCodex() error {
	if err := SeedPoolsIniciales(); err != nil {
		return err
	}
	if err := SeedModelosIniciales(); err != nil {
		return err
	}
	if err := SeedPoliticasModeloIniciales(); err != nil {
		return err
	}
	return asegurarPoliticaImplementacionXHigh()
}

func asegurarPoliticaImplementacionXHigh() error {
	res, err := DB.Exec(`
		UPDATE politicas_modelo
		SET reasoning_effort = 'xhigh'
		WHERE scope_tipo = 'perfil'
		  AND scope_ref = 'implementacion'
		  AND perfil_tarea = 'implementacion'
		  AND activa = 1`)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}
	_, err = GuardarPoliticaModelo(&PoliticaModelo{
		ScopeTipo:       "perfil",
		ScopeRef:        "implementacion",
		PerfilTarea:     "implementacion",
		ReasoningEffort: "xhigh",
		Prioridad:       10,
		Activa:          true,
	})
	return err
}

func GuardarPoliticaModelo(p *PoliticaModelo) (int64, error) {
	if p == nil {
		return 0, fmt.Errorf("politica obligatoria")
	}
	p.ScopeTipo = strings.TrimSpace(p.ScopeTipo)
	p.ScopeRef = strings.TrimSpace(p.ScopeRef)
	p.PerfilTarea = strings.TrimSpace(p.PerfilTarea)
	p.PoolSlug = strings.TrimSpace(p.PoolSlug)
	p.ModelSlug = strings.TrimSpace(p.ModelSlug)
	p.ReasoningEffort = strings.TrimSpace(strings.ToLower(p.ReasoningEffort))
	if p.PerfilTarea == "" {
		p.PerfilTarea = "*"
	}
	if p.MetadataJSON == "" {
		p.MetadataJSON = "{}"
	}
	if p.ScopeTipo == "perfil" && p.ScopeRef == "" && p.PerfilTarea != "*" {
		p.ScopeRef = p.PerfilTarea
	}
	if p.ScopeTipo == "global" {
		p.ScopeRef = ""
	}
	if !scopeTipoPoliticaValido(p.ScopeTipo) {
		return 0, fmt.Errorf("scope_tipo invalido: %s", p.ScopeTipo)
	}
	if p.ScopeTipo != "global" && p.ScopeRef == "" {
		return 0, fmt.Errorf("scope_ref obligatorio para scope_tipo %s", p.ScopeTipo)
	}
	if !reasoningEffortValido(p.ReasoningEffort) {
		return 0, fmt.Errorf("reasoning_effort invalido: %s", p.ReasoningEffort)
	}
	if p.Prioridad <= 0 {
		p.Prioridad = 100
	}
	if p.PoolSlug != "" {
		if _, err := GetPool(p.PoolSlug); err != nil {
			return 0, fmt.Errorf("pool inexistente en politica: %w", err)
		}
	}

	if _, err := DB.Exec(`
		INSERT INTO politicas_modelo (
			scope_tipo, scope_ref, perfil_tarea, pool_slug, model_slug,
			reasoning_effort, prioridad, activa, metadata_json
		) VALUES (?,?,?,?,?,?,?,?,?)`,
		p.ScopeTipo, p.ScopeRef, p.PerfilTarea, p.PoolSlug, p.ModelSlug,
		p.ReasoningEffort, p.Prioridad, p.Activa, p.MetadataJSON,
	); err != nil {
		return 0, err
	}

	var id int64
	if err := DB.QueryRow(`
		SELECT id FROM politicas_modelo
		WHERE scope_tipo = ? AND scope_ref = ? AND perfil_tarea = ? AND prioridad = ?
		ORDER BY id DESC LIMIT 1`,
		p.ScopeTipo, p.ScopeRef, p.PerfilTarea, p.Prioridad,
	).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func ListarPoliticasModelo(scopeTipo, scopeRef string, activa *bool) ([]*PoliticaModelo, error) {
	q := `
		SELECT id, scope_tipo, scope_ref, perfil_tarea, pool_slug, model_slug,
		       reasoning_effort, prioridad, activa, metadata_json, created_at, updated_at
		FROM politicas_modelo
		WHERE 1=1`
	args := []any{}
	if scopeTipo = strings.TrimSpace(scopeTipo); scopeTipo != "" {
		q += ` AND scope_tipo = ?`
		args = append(args, scopeTipo)
	}
	if scopeRef = strings.TrimSpace(scopeRef); scopeRef != "" {
		q += ` AND scope_ref = ?`
		args = append(args, scopeRef)
	}
	if activa != nil {
		q += ` AND activa = ?`
		args = append(args, *activa)
	}
	q += ` ORDER BY
		CASE scope_tipo
			WHEN 'global' THEN 1
			WHEN 'perfil' THEN 2
			WHEN 'proyecto' THEN 3
			WHEN 'fase' THEN 4
			WHEN 'tarea' THEN 5
			ELSE 99
		END,
		scope_ref, prioridad, id`
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*PoliticaModelo
	for rows.Next() {
		item, err := scanPoliticaModelo(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func ResolverPoliticaModelo(input ResolverPoliticaInput) (*ResolucionModelo, error) {
	perfil := strings.TrimSpace(input.PerfilTarea)
	if perfil == "" {
		perfil = configOrDefault("model_policy_default_profile", "implementacion")
	}
	resultado := &ResolucionModelo{
		PerfilTarea:     perfil,
		ReasoningEffort: configOrDefault("model_policy_default_reasoning", "high"),
		FuenteReasoning: "config:model_policy_default_reasoning",
	}

	for _, layer := range capasPolitica(input, perfil) {
		policy, err := mejorPoliticaLayer(layer.scopeTipo, layer.scopeRef, perfil)
		if err != nil {
			return nil, err
		}
		if policy == nil {
			continue
		}
		resultado.PoliticasAplicadas = append(resultado.PoliticasAplicadas, policy)
		if policy.PoolSlug != "" {
			resultado.PoolSlug = policy.PoolSlug
			resultado.FuentePool = politicaFuente(policy)
		}
		if policy.ModelSlug != "" {
			resultado.ModelSlug = policy.ModelSlug
			resultado.FuenteModelo = politicaFuente(policy)
		}
		if policy.ReasoningEffort != "" {
			resultado.ReasoningEffort = policy.ReasoningEffort
			resultado.FuenteReasoning = politicaFuente(policy)
		}
	}

	if resultado.PoolSlug == "" && resultado.ModelSlug != "" {
		pool, err := resolverPoolPorModelo(perfil, resultado.ModelSlug)
		if err != nil {
			return nil, err
		}
		resultado.PoolSlug = pool.Slug
		resultado.FuentePool = "pool-por-modelo:" + resultado.ModelSlug
	}

	if resultado.PoolSlug == "" {
		pool, model, err := resolverPoolYModeloPorPerfil(perfil)
		if err != nil {
			return nil, err
		}
		resultado.PoolSlug = pool.Slug
		resultado.FuentePool = "fallback-perfil:" + perfil
		if resultado.ModelSlug == "" {
			resultado.ModelSlug = model.ModelSlug
			resultado.FuenteModelo = "fallback-perfil:" + perfil
		}
	}

	if resultado.ModelSlug == "" {
		model, err := resolverModeloPool(resultado.PoolSlug, perfil)
		if err != nil {
			return nil, err
		}
		resultado.ModelSlug = model.ModelSlug
		resultado.FuenteModelo = "fallback-pool:" + resultado.PoolSlug
	}

	if resultado.FuentePool == "" {
		resultado.FuentePool = "politica"
	}
	if resultado.FuenteModelo == "" {
		resultado.FuenteModelo = "politica"
	}
	return resultado, nil
}

type capaPolitica struct {
	scopeTipo string
	scopeRef  string
}

func capasPolitica(input ResolverPoliticaInput, perfil string) []capaPolitica {
	layers := []capaPolitica{
		{scopeTipo: "global", scopeRef: ""},
		{scopeTipo: "perfil", scopeRef: perfil},
	}
	if strings.TrimSpace(input.ProyectoSlug) != "" {
		layers = append(layers, capaPolitica{scopeTipo: "proyecto", scopeRef: strings.TrimSpace(input.ProyectoSlug)})
	}
	if strings.TrimSpace(input.Fase) != "" {
		layers = append(layers, capaPolitica{scopeTipo: "fase", scopeRef: strings.TrimSpace(input.Fase)})
	}
	if input.TareaID != nil {
		layers = append(layers, capaPolitica{scopeTipo: "tarea", scopeRef: strconv.FormatInt(*input.TareaID, 10)})
	}
	return layers
}

func mejorPoliticaLayer(scopeTipo, scopeRef, perfil string) (*PoliticaModelo, error) {
	row := DB.QueryRow(`
		SELECT id, scope_tipo, scope_ref, perfil_tarea, pool_slug, model_slug,
		       reasoning_effort, prioridad, activa, metadata_json, created_at, updated_at
		FROM politicas_modelo
		WHERE activa = 1
		  AND scope_tipo = ?
		  AND scope_ref = ?
		  AND perfil_tarea IN (?, '*')
		ORDER BY CASE WHEN perfil_tarea = ? THEN 0 ELSE 1 END, prioridad, id DESC
		LIMIT 1`,
		scopeTipo, scopeRef, perfil, perfil,
	)
	item, err := scanPoliticaModelo(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return item, err
}

func resolverPoolPorModelo(perfil, modelSlug string) (*PoolCapacidad, error) {
	resumen, err := ListarPoolsResumen(boolPtr(true))
	if err != nil {
		return nil, err
	}
	var mejor *poolCandidate
	for _, item := range resumen {
		modelos, err := ListarModelosPool(item.Pool.Slug)
		if err != nil {
			return nil, err
		}
		for _, model := range modelos {
			if !model.Activo || model.ModelSlug != modelSlug {
				continue
			}
			c := &poolCandidate{Pool: item.Pool, Model: model, CapacidadDisponible: item.CapacidadDisponible}
			if betterCandidate(c, mejor, perfil) {
				mejor = c
			}
		}
	}
	if mejor == nil {
		return nil, fmt.Errorf("no hay pool activo para el modelo %s", modelSlug)
	}
	return mejor.Pool, nil
}

func resolverPoolYModeloPorPerfil(perfil string) (*PoolCapacidad, *PoolModelo, error) {
	resumen, err := ListarPoolsResumen(boolPtr(true))
	if err != nil {
		return nil, nil, err
	}
	var mejor *poolCandidate
	for _, item := range resumen {
		modelo, err := resolverModeloPool(item.Pool.Slug, perfil)
		if err != nil {
			if strings.Contains(err.Error(), "no hay modelos activos") {
				continue
			}
			return nil, nil, err
		}
		c := &poolCandidate{Pool: item.Pool, Model: modelo, CapacidadDisponible: item.CapacidadDisponible}
		if betterCandidate(c, mejor, perfil) {
			mejor = c
		}
	}
	if mejor == nil {
		return nil, nil, fmt.Errorf("no hay pools activos con modelos configurados")
	}
	return mejor.Pool, mejor.Model, nil
}

func resolverModeloPool(poolSlug, perfil string) (*PoolModelo, error) {
	modelos, err := ListarModelosPool(poolSlug)
	if err != nil {
		return nil, err
	}
	var activos []*PoolModelo
	for _, model := range modelos {
		if model.Activo {
			activos = append(activos, model)
		}
	}
	if len(activos) == 0 {
		return nil, fmt.Errorf("no hay modelos activos para el pool %s", poolSlug)
	}
	sort.SliceStable(activos, func(i, j int) bool {
		a, b := activos[i], activos[j]
		if perfilEconomico(perfil) {
			if a.CosteRelativo != b.CosteRelativo {
				return a.CosteRelativo < b.CosteRelativo
			}
			if a.Prioridad != b.Prioridad {
				return a.Prioridad < b.Prioridad
			}
			return a.ModelSlug < b.ModelSlug
		}
		if a.Prioridad != b.Prioridad {
			return a.Prioridad < b.Prioridad
		}
		if a.CosteRelativo != b.CosteRelativo {
			return a.CosteRelativo < b.CosteRelativo
		}
		return a.ModelSlug < b.ModelSlug
	})
	return activos[0], nil
}

type poolCandidate struct {
	Pool                *PoolCapacidad
	Model               *PoolModelo
	CapacidadDisponible int
}

func betterCandidate(a, b *poolCandidate, perfil string) bool {
	if a == nil {
		return false
	}
	if b == nil {
		return true
	}
	aLibre := a.CapacidadDisponible > 0
	bLibre := b.CapacidadDisponible > 0
	if aLibre != bLibre {
		return aLibre
	}
	if perfilEconomico(perfil) {
		if a.Model.CosteRelativo != b.Model.CosteRelativo {
			return a.Model.CosteRelativo < b.Model.CosteRelativo
		}
		if a.Pool.EsDePago != b.Pool.EsDePago {
			return !a.Pool.EsDePago
		}
		if a.CapacidadDisponible != b.CapacidadDisponible {
			return a.CapacidadDisponible > b.CapacidadDisponible
		}
		if a.Model.Prioridad != b.Model.Prioridad {
			return a.Model.Prioridad < b.Model.Prioridad
		}
		return a.Pool.Slug < b.Pool.Slug
	}
	if a.Model.Prioridad != b.Model.Prioridad {
		return a.Model.Prioridad < b.Model.Prioridad
	}
	if a.CapacidadDisponible != b.CapacidadDisponible {
		return a.CapacidadDisponible > b.CapacidadDisponible
	}
	if a.Model.CosteRelativo != b.Model.CosteRelativo {
		return a.Model.CosteRelativo < b.Model.CosteRelativo
	}
	return a.Pool.Slug < b.Pool.Slug
}

func politicaFuente(p *PoliticaModelo) string {
	return fmt.Sprintf("%s:%s@%s", p.ScopeTipo, p.ScopeRef, p.PerfilTarea)
}

func perfilEconomico(perfil string) bool {
	switch strings.TrimSpace(perfil) {
	case "script", "handoff":
		return true
	default:
		return false
	}
}

func configOrDefault(clave, fallback string) string {
	v, err := ConfigGet(clave)
	if err != nil || strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func scopeTipoPoliticaValido(scope string) bool {
	switch scope {
	case "global", "perfil", "proyecto", "fase", "tarea":
		return true
	default:
		return false
	}
}

func reasoningEffortValido(v string) bool {
	switch v {
	case "", "low", "medium", "high", "xhigh":
		return true
	default:
		return false
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func scanPoliticaModelo(scanner interface{ Scan(...any) error }) (*PoliticaModelo, error) {
	item := &PoliticaModelo{}
	err := scanner.Scan(
		&item.ID,
		&item.ScopeTipo,
		&item.ScopeRef,
		&item.PerfilTarea,
		&item.PoolSlug,
		&item.ModelSlug,
		&item.ReasoningEffort,
		&item.Prioridad,
		&item.Activa,
		&item.MetadataJSON,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func SeedPoliticasModeloIniciales() error {
	iniciales := []PoliticaModelo{
		{ScopeTipo: "perfil", ScopeRef: "orquestacion", PerfilTarea: "orquestacion", ReasoningEffort: "xhigh", Prioridad: 10, Activa: true},
		{ScopeTipo: "perfil", ScopeRef: "analisis", PerfilTarea: "analisis", ReasoningEffort: "high", Prioridad: 10, Activa: true},
		{ScopeTipo: "perfil", ScopeRef: "implementacion", PerfilTarea: "implementacion", ReasoningEffort: "xhigh", Prioridad: 10, Activa: true},
		{ScopeTipo: "perfil", ScopeRef: "script", PerfilTarea: "script", ReasoningEffort: "medium", Prioridad: 10, Activa: true},
		{ScopeTipo: "perfil", ScopeRef: "revision", PerfilTarea: "revision", ReasoningEffort: "high", Prioridad: 10, Activa: true},
		{ScopeTipo: "perfil", ScopeRef: "handoff", PerfilTarea: "handoff", ReasoningEffort: "medium", Prioridad: 10, Activa: true},
	}
	for _, item := range iniciales {
		if err := seedPoliticaModeloPerfil(item); err != nil {
			return err
		}
	}
	return nil
}

func seedPoliticaModeloPerfil(p PoliticaModelo) error {
	var existente int
	err := DB.QueryRow(`
		SELECT COUNT(*) FROM politicas_modelo
		WHERE scope_tipo = ? AND scope_ref = ? AND perfil_tarea = ? AND reasoning_effort = ?`,
		p.ScopeTipo, p.ScopeRef, p.PerfilTarea, p.ReasoningEffort,
	).Scan(&existente)
	if err != nil {
		return err
	}
	if existente > 0 {
		return nil
	}
	_, err = GuardarPoliticaModelo(&p)
	return err
}
