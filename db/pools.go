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
	"strings"
	"time"
)

type PoolCapacidad struct {
	ID                  int64
	Slug                string
	Proveedor           string
	Runtime             string
	Plan                string
	EsDePago            bool
	CapacidadTotal      int64
	CapacidadReservada  int64
	PermiteHijos        bool
	PermiteModelosMulti bool
	PermiteSobrecoste   bool
	PoliticaHandoff     string
	FuenteTelemetria    string
	MetadataJSON        string
	Activo              bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type PoolModelo struct {
	ID                 int64
	PoolID             int64
	ModelSlug          string
	Activo             bool
	Prioridad          int64
	CosteRelativo      float64
	LimiteConocidoJSON string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type SeleccionModelo struct {
	Agente      string
	Perfil      string
	Pool        *PoolCapacidad
	Modelo      *PoolModelo
	Presupuesto *PresupuestoSesion
	Evaluacion  *EvaluacionPresupuesto
	Criterio    string
}

func ListarPoolsCapacidad(incluirInactivos bool) ([]*PoolCapacidad, error) {
	q := `
		SELECT id, slug, proveedor, runtime, plan, es_de_pago, capacidad_total, capacidad_reservada,
		       permite_hijos, permite_modelos_multi, permite_sobrecoste, politica_handoff,
		       fuente_telemetria, metadata_json, activo, created_at, updated_at
		FROM pools_capacidad`
	if !incluirInactivos {
		q += ` WHERE activo = 1`
	}
	q += ` ORDER BY slug`

	rows, err := DB.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*PoolCapacidad
	for rows.Next() {
		pool, err := escanearPoolCapacidad(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, pool)
	}
	return list, rows.Err()
}

func GetPoolCapacidad(id int64) (*PoolCapacidad, error) {
	row := DB.QueryRow(`
		SELECT id, slug, proveedor, runtime, plan, es_de_pago, capacidad_total, capacidad_reservada,
		       permite_hijos, permite_modelos_multi, permite_sobrecoste, politica_handoff,
		       fuente_telemetria, metadata_json, activo, created_at, updated_at
		FROM pools_capacidad
		WHERE id = ?`, id)
	return escanearPoolCapacidad(row)
}

func CrearPoolCapacidad(pool *PoolCapacidad) (int64, error) {
	if pool == nil {
		return 0, fmt.Errorf("pool nulo")
	}
	if pool.Slug == "" || pool.Proveedor == "" || pool.Runtime == "" {
		return 0, fmt.Errorf("slug, proveedor y runtime son obligatorios")
	}
	if pool.CapacidadTotal <= 0 {
		pool.CapacidadTotal = 1
	}
	if pool.PoliticaHandoff == "" {
		pool.PoliticaHandoff = "preventivo"
	}
	if pool.FuenteTelemetria == "" {
		pool.FuenteTelemetria = "manual"
	}
	if pool.MetadataJSON == "" {
		pool.MetadataJSON = "{}"
	}
	if !pool.Activo {
		pool.Activo = true
	}
	res, err := DB.Exec(`
		INSERT INTO pools_capacidad (
			slug, proveedor, runtime, plan, es_de_pago, capacidad_total, capacidad_reservada,
			permite_hijos, permite_modelos_multi, permite_sobrecoste, politica_handoff,
			fuente_telemetria, metadata_json, activo
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		pool.Slug, pool.Proveedor, pool.Runtime, pool.Plan, pool.EsDePago, pool.CapacidadTotal, pool.CapacidadReservada,
		pool.PermiteHijos, pool.PermiteModelosMulti, pool.PermiteSobrecoste, pool.PoliticaHandoff,
		pool.FuenteTelemetria, pool.MetadataJSON, pool.Activo,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	Audit("orquesta", "crear_pool_capacidad", "pool_capacidad", id, pool.Slug)
	return id, nil
}

func ActualizarPoolCapacidad(pool *PoolCapacidad) error {
	if pool == nil {
		return fmt.Errorf("pool nulo")
	}
	if pool.ID <= 0 {
		return fmt.Errorf("id de pool obligatorio")
	}
	if pool.Slug == "" || pool.Proveedor == "" || pool.Runtime == "" {
		return fmt.Errorf("slug, proveedor y runtime son obligatorios")
	}
	if pool.CapacidadTotal <= 0 {
		return fmt.Errorf("capacidad_total debe ser > 0")
	}
	res, err := DB.Exec(`
		UPDATE pools_capacidad
		SET slug=?, proveedor=?, runtime=?, plan=?, es_de_pago=?, capacidad_total=?, capacidad_reservada=?,
		    permite_hijos=?, permite_modelos_multi=?, permite_sobrecoste=?, politica_handoff=?,
		    fuente_telemetria=?, metadata_json=?, activo=?
		WHERE id=?`,
		pool.Slug, pool.Proveedor, pool.Runtime, pool.Plan, pool.EsDePago, pool.CapacidadTotal, pool.CapacidadReservada,
		pool.PermiteHijos, pool.PermiteModelosMulti, pool.PermiteSobrecoste, pool.PoliticaHandoff,
		pool.FuenteTelemetria, pool.MetadataJSON, pool.Activo, pool.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("pool #%d no encontrado", pool.ID)
	}
	Audit("orquesta", "actualizar_pool_capacidad", "pool_capacidad", pool.ID, pool.Slug)
	return nil
}

func SetPoolCapacidadActivo(id int64, activo bool) error {
	res, err := DB.Exec(`UPDATE pools_capacidad SET activo=? WHERE id=?`, activo, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("pool #%d no encontrado", id)
	}
	accion := "desactivar_pool_capacidad"
	if activo {
		accion = "activar_pool_capacidad"
	}
	Audit("orquesta", accion, "pool_capacidad", id, "")
	return nil
}

func ListarPoolModelos(poolID int64, incluirInactivos bool) ([]*PoolModelo, error) {
	q := `
		SELECT id, pool_id, model_slug, activo, prioridad, coste_relativo, limite_conocido_json, created_at, updated_at
		FROM pool_modelos
		WHERE pool_id = ?`
	args := []any{poolID}
	if !incluirInactivos {
		q += ` AND activo = 1`
	}
	q += ` ORDER BY prioridad, coste_relativo, model_slug`

	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*PoolModelo
	for rows.Next() {
		modelo, err := escanearPoolModelo(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, modelo)
	}
	return list, rows.Err()
}

func GetPoolModelo(id int64) (*PoolModelo, error) {
	row := DB.QueryRow(`
		SELECT id, pool_id, model_slug, activo, prioridad, coste_relativo, limite_conocido_json, created_at, updated_at
		FROM pool_modelos
		WHERE id = ?`, id)
	return escanearPoolModelo(row)
}

func GetPoolModeloPorSlug(poolID int64, modelSlug string) (*PoolModelo, error) {
	row := DB.QueryRow(`
		SELECT id, pool_id, model_slug, activo, prioridad, coste_relativo, limite_conocido_json, created_at, updated_at
		FROM pool_modelos
		WHERE pool_id = ? AND model_slug = ?`, poolID, modelSlug)
	return escanearPoolModelo(row)
}

func RegistrarPoolModelo(modelo *PoolModelo) (int64, error) {
	if modelo == nil {
		return 0, fmt.Errorf("modelo nulo")
	}
	if modelo.PoolID <= 0 || modelo.ModelSlug == "" {
		return 0, fmt.Errorf("pool_id y model_slug son obligatorios")
	}
	if _, err := GetPoolCapacidad(modelo.PoolID); err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("pool #%d no encontrado", modelo.PoolID)
		}
		return 0, err
	}
	if modelo.Prioridad <= 0 {
		modelo.Prioridad = 100
	}
	if modelo.CosteRelativo <= 0 {
		modelo.CosteRelativo = 1.0
	}
	if modelo.LimiteConocidoJSON == "" {
		modelo.LimiteConocidoJSON = "{}"
	}
	if !modelo.Activo {
		modelo.Activo = true
	}
	res, err := DB.Exec(`
		INSERT INTO pool_modelos (pool_id, model_slug, activo, prioridad, coste_relativo, limite_conocido_json)
		VALUES (?,?,?,?,?,?)`,
		modelo.PoolID, modelo.ModelSlug, modelo.Activo, modelo.Prioridad, modelo.CosteRelativo, modelo.LimiteConocidoJSON,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	Audit("orquesta", "crear_pool_modelo", "pool_modelo", id, modelo.ModelSlug)
	return id, nil
}

func ActualizarPoolModelo(modelo *PoolModelo) error {
	if modelo == nil {
		return fmt.Errorf("modelo nulo")
	}
	if modelo.ID <= 0 {
		return fmt.Errorf("id de modelo obligatorio")
	}
	if modelo.PoolID <= 0 || modelo.ModelSlug == "" {
		return fmt.Errorf("pool_id y model_slug son obligatorios")
	}
	if modelo.Prioridad <= 0 {
		return fmt.Errorf("prioridad debe ser > 0")
	}
	if modelo.CosteRelativo <= 0 {
		return fmt.Errorf("coste_relativo debe ser > 0")
	}
	if modelo.LimiteConocidoJSON == "" {
		modelo.LimiteConocidoJSON = "{}"
	}
	res, err := DB.Exec(`
		UPDATE pool_modelos
		SET pool_id=?, model_slug=?, activo=?, prioridad=?, coste_relativo=?, limite_conocido_json=?
		WHERE id=?`,
		modelo.PoolID, modelo.ModelSlug, modelo.Activo, modelo.Prioridad, modelo.CosteRelativo, modelo.LimiteConocidoJSON, modelo.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("modelo #%d no encontrado", modelo.ID)
	}
	Audit("orquesta", "actualizar_pool_modelo", "pool_modelo", modelo.ID, modelo.ModelSlug)
	return nil
}

func SetPoolModeloActivo(id int64, activo bool) error {
	res, err := DB.Exec(`UPDATE pool_modelos SET activo=? WHERE id=?`, activo, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("modelo #%d no encontrado", id)
	}
	accion := "desactivar_pool_modelo"
	if activo {
		accion = "activar_pool_modelo"
	}
	Audit("orquesta", accion, "pool_modelo", id, "")
	return nil
}

func SeleccionarModeloParaAgente(agente string, poolID *int64, perfil string) (*SeleccionModelo, error) {
	if strings.TrimSpace(agente) == "" {
		return nil, fmt.Errorf("agente obligatorio")
	}
	if strings.TrimSpace(perfil) == "" {
		perfil = defaultModelProfile()
	}

	var presupuesto *PresupuestoSesion
	var evaluacion *EvaluacionPresupuesto
	if p, _, err := UltimoPresupuestoAgente(agente); err == nil {
		presupuesto = p
		evaluacion, _ = EvaluarPresupuestoSesion(p)
	} else if err != sql.ErrNoRows {
		return nil, err
	}

	pool, err := resolverPoolSeleccion(poolID, presupuesto)
	if err != nil {
		return nil, err
	}
	modelos, err := ListarPoolModelos(pool.ID, false)
	if err != nil {
		return nil, err
	}
	if len(modelos) == 0 {
		return nil, fmt.Errorf("el pool %s no tiene modelos activos", pool.Slug)
	}

	seleccion := &SeleccionModelo{
		Agente:      agente,
		Perfil:      perfil,
		Pool:        pool,
		Presupuesto: presupuesto,
		Evaluacion:  evaluacion,
	}

	if presupuesto != nil && presupuesto.ModelSlug != "" && evaluacion != nil && !evaluacion.DebeHandoff {
		for _, modelo := range modelos {
			if modelo.ModelSlug == presupuesto.ModelSlug {
				seleccion.Modelo = modelo
				seleccion.Criterio = "continuidad con el modelo ya observado en presupuesto"
				return seleccion, nil
			}
		}
	}

	if evaluacion != nil && evaluacion.DebeHandoff {
		seleccion.Modelo = elegirModeloMasBarato(modelos)
		seleccion.Criterio = "modo ahorro por presupuesto en handoff preventivo"
		return seleccion, nil
	}

	switch strings.ToLower(strings.TrimSpace(perfil)) {
	case "economico", "ahorro", "barato", "revision":
		seleccion.Modelo = elegirModeloMasBarato(modelos)
		seleccion.Criterio = "perfil económico/revisión"
	default:
		seleccion.Modelo = modelos[0]
		seleccion.Criterio = "prioridad por defecto del pool"
	}
	return seleccion, nil
}

func defaultModelProfile() string {
	v, err := ConfigGet("model_policy_default_profile")
	if err != nil || strings.TrimSpace(v) == "" {
		return "implementacion"
	}
	return strings.TrimSpace(v)
}

func resolverPoolSeleccion(explicito *int64, presupuesto *PresupuestoSesion) (*PoolCapacidad, error) {
	if explicito != nil {
		return GetPoolCapacidad(*explicito)
	}
	if presupuesto != nil && presupuesto.PoolID != nil {
		return GetPoolCapacidad(*presupuesto.PoolID)
	}
	pools, err := ListarPoolsCapacidad(false)
	if err != nil {
		return nil, err
	}
	if len(pools) == 0 {
		return nil, fmt.Errorf("no hay pools activos")
	}
	if len(pools) > 1 {
		return nil, fmt.Errorf("hay varios pools activos; especifica --pool o registra presupuesto con pool_id")
	}
	return pools[0], nil
}

func elegirModeloMasBarato(modelos []*PoolModelo) *PoolModelo {
	if len(modelos) == 0 {
		return nil
	}
	elegido := modelos[0]
	for _, modelo := range modelos[1:] {
		if modelo.CosteRelativo < elegido.CosteRelativo {
			elegido = modelo
			continue
		}
		if modelo.CosteRelativo == elegido.CosteRelativo && modelo.Prioridad < elegido.Prioridad {
			elegido = modelo
		}
	}
	return elegido
}

func escanearPoolCapacidad(s scanner) (*PoolCapacidad, error) {
	pool := &PoolCapacidad{}
	if err := s.Scan(
		&pool.ID, &pool.Slug, &pool.Proveedor, &pool.Runtime, &pool.Plan, &pool.EsDePago,
		&pool.CapacidadTotal, &pool.CapacidadReservada, &pool.PermiteHijos, &pool.PermiteModelosMulti,
		&pool.PermiteSobrecoste, &pool.PoliticaHandoff, &pool.FuenteTelemetria, &pool.MetadataJSON,
		&pool.Activo, &pool.CreatedAt, &pool.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return pool, nil
}

func escanearPoolModelo(s scanner) (*PoolModelo, error) {
	modelo := &PoolModelo{}
	if err := s.Scan(
		&modelo.ID, &modelo.PoolID, &modelo.ModelSlug, &modelo.Activo, &modelo.Prioridad,
		&modelo.CosteRelativo, &modelo.LimiteConocidoJSON, &modelo.CreatedAt, &modelo.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return modelo, nil
}
