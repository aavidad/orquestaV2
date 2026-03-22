/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"fmt"
	"time"
)

type PoolCapacidad struct {
	ID                  int64
	Slug                string
	Proveedor           string
	Runtime             string
	Plan                string
	EsDePago            bool
	CapacidadTotal      int
	CapacidadReservada  int
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
	Prioridad          int
	CosteRelativo      float64
	LimiteConocidoJSON string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type PoolCapacidadResumen struct {
	Pool                *PoolCapacidad
	SesionesActivas     int
	CapacidadDisponible int
}

func GuardarPool(p *PoolCapacidad) (int64, error) {
	if p == nil {
		return 0, fmt.Errorf("pool obligatorio")
	}
	if p.Slug == "" || p.Proveedor == "" || p.Runtime == "" {
		return 0, fmt.Errorf("slug, proveedor y runtime son obligatorios")
	}
	if p.CapacidadTotal < 0 || p.CapacidadReservada < 0 {
		return 0, fmt.Errorf("las capacidades no pueden ser negativas")
	}

	if _, err := DB.Exec(
		upsertValuesSQL(
			"pools_capacidad",
			[]string{
				"slug", "proveedor", "runtime", "plan", "es_de_pago", "capacidad_total",
				"capacidad_reservada", "permite_hijos", "permite_modelos_multi",
				"permite_sobrecoste", "politica_handoff", "fuente_telemetria",
				"metadata_json", "activo",
			},
			[]string{"slug"},
			[]upsertAssignment{
				{Column: "proveedor"},
				{Column: "runtime"},
				{Column: "plan"},
				{Column: "es_de_pago"},
				{Column: "capacidad_total"},
				{Column: "capacidad_reservada"},
				{Column: "permite_hijos"},
				{Column: "permite_modelos_multi"},
				{Column: "permite_sobrecoste"},
				{Column: "politica_handoff"},
				{Column: "fuente_telemetria"},
				{Column: "metadata_json"},
				{Column: "activo"},
			},
		),
		p.Slug, p.Proveedor, p.Runtime, p.Plan, p.EsDePago, p.CapacidadTotal,
		p.CapacidadReservada, p.PermiteHijos, p.PermiteModelosMulti,
		p.PermiteSobrecoste, p.PoliticaHandoff, p.FuenteTelemetria,
		p.MetadataJSON, p.Activo,
	); err != nil {
		return 0, err
	}
	var id int64
	if err := DB.QueryRow(`SELECT id FROM pools_capacidad WHERE slug = ?`, p.Slug).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func GetPool(slug string) (*PoolCapacidad, error) {
	row := DB.QueryRow(`
		SELECT id, slug, proveedor, runtime, plan, es_de_pago, capacidad_total,
		       capacidad_reservada, permite_hijos, permite_modelos_multi,
		       permite_sobrecoste, politica_handoff, fuente_telemetria,
		       metadata_json, activo, created_at, updated_at
		FROM pools_capacidad WHERE slug = ?`, slug)
	return scanPool(row)
}

func ListarPools(activo *bool) ([]*PoolCapacidad, error) {
	q := `
		SELECT id, slug, proveedor, runtime, plan, es_de_pago, capacidad_total,
		       capacidad_reservada, permite_hijos, permite_modelos_multi,
		       permite_sobrecoste, politica_handoff, fuente_telemetria,
		       metadata_json, activo, created_at, updated_at
		FROM pools_capacidad`
	var args []any
	if activo != nil {
		q += ` WHERE activo = ?`
		args = append(args, *activo)
	}
	q += ` ORDER BY slug`
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*PoolCapacidad
	for rows.Next() {
		p, err := scanPool(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func ListarPoolsResumen(activo *bool) ([]*PoolCapacidadResumen, error) {
	pools, err := ListarPools(activo)
	if err != nil {
		return nil, err
	}
	var list []*PoolCapacidadResumen
	for _, pool := range pools {
		sesionesActivas, err := contarSesionesActivasPool(pool.ID)
		if err != nil {
			return nil, err
		}
		disponible := pool.CapacidadTotal - pool.CapacidadReservada - sesionesActivas
		list = append(list, &PoolCapacidadResumen{
			Pool:                pool,
			SesionesActivas:     sesionesActivas,
			CapacidadDisponible: disponible,
		})
	}
	return list, nil
}

func GuardarPoolModelo(poolSlug string, modelo *PoolModelo) (int64, error) {
	if modelo == nil {
		return 0, fmt.Errorf("modelo obligatorio")
	}
	if poolSlug == "" || modelo.ModelSlug == "" {
		return 0, fmt.Errorf("pool y model_slug son obligatorios")
	}
	pool, err := GetPool(poolSlug)
	if err != nil {
		return 0, err
	}

	if _, err := DB.Exec(
		upsertValuesSQL(
			"pool_modelos",
			[]string{"pool_id", "model_slug", "activo", "prioridad", "coste_relativo", "limite_conocido_json"},
			[]string{"pool_id", "model_slug"},
			[]upsertAssignment{
				{Column: "activo"},
				{Column: "prioridad"},
				{Column: "coste_relativo"},
				{Column: "limite_conocido_json"},
			},
		),
		pool.ID, modelo.ModelSlug, modelo.Activo, modelo.Prioridad,
		modelo.CosteRelativo, modelo.LimiteConocidoJSON,
	); err != nil {
		return 0, err
	}

	var id int64
	if err := DB.QueryRow(
		`SELECT id FROM pool_modelos WHERE pool_id = ? AND model_slug = ?`,
		pool.ID, modelo.ModelSlug,
	).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func ListarModelosPool(poolSlug string) ([]*PoolModelo, error) {
	pool, err := GetPool(poolSlug)
	if err != nil {
		return nil, err
	}
	rows, err := DB.Query(`
		SELECT id, pool_id, model_slug, activo, prioridad, coste_relativo,
		       limite_conocido_json, created_at, updated_at
		FROM pool_modelos
		WHERE pool_id = ?
		ORDER BY prioridad, model_slug`, pool.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*PoolModelo
	for rows.Next() {
		m, err := scanPoolModelo(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func SeedModelosIniciales() error {
	seeds := []struct {
		poolSlug string
		modelo   PoolModelo
	}{
		{
			poolSlug: "codex",
			modelo: PoolModelo{
				ModelSlug:          "gpt-5.4",
				Activo:             true,
				Prioridad:          10,
				CosteRelativo:      1.5,
				LimiteConocidoJSON: "{}",
			},
		},
		{
			poolSlug: "codex",
			modelo: PoolModelo{
				ModelSlug:          "gpt-5.4-mini",
				Activo:             true,
				Prioridad:          20,
				CosteRelativo:      0.5,
				LimiteConocidoJSON: "{}",
			},
		},
		{
			poolSlug: "claude",
			modelo: PoolModelo{
				ModelSlug:          "claude-sonnet",
				Activo:             true,
				Prioridad:          10,
				CosteRelativo:      1.0,
				LimiteConocidoJSON: "{}",
			},
		},
		{
			poolSlug: "android",
			modelo: PoolModelo{
				ModelSlug:          "android-shell",
				Activo:             true,
				Prioridad:          10,
				CosteRelativo:      0.1,
				LimiteConocidoJSON: "{}",
			},
		},
	}
	for _, seed := range seeds {
		modelo := seed.modelo
		if _, err := GuardarPoolModelo(seed.poolSlug, &modelo); err != nil {
			return err
		}
	}
	return nil
}

func contarSesionesActivasPool(poolID int64) (int, error) {
	var n int
	err := DB.QueryRow(`SELECT COUNT(*) FROM sesiones WHERE activa = 1 AND pool_id = ?`, poolID).Scan(&n)
	return n, err
}

func scanPool(s scanner) (*PoolCapacidad, error) {
	p := &PoolCapacidad{}
	err := s.Scan(
		&p.ID, &p.Slug, &p.Proveedor, &p.Runtime, &p.Plan, &p.EsDePago,
		&p.CapacidadTotal, &p.CapacidadReservada, &p.PermiteHijos,
		&p.PermiteModelosMulti, &p.PermiteSobrecoste, &p.PoliticaHandoff,
		&p.FuenteTelemetria, &p.MetadataJSON, &p.Activo, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func scanPoolModelo(s scanner) (*PoolModelo, error) {
	m := &PoolModelo{}
	err := s.Scan(
		&m.ID, &m.PoolID, &m.ModelSlug, &m.Activo, &m.Prioridad,
		&m.CosteRelativo, &m.LimiteConocidoJSON, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return m, nil
}
