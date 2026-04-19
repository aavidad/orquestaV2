package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type TipoEntidadMemoria string

const (
	EntidadMemoriaNegocio TipoEntidadMemoria = "negocio"
	EntidadMemoriaAPI     TipoEntidadMemoria = "api"
	EntidadMemoriaDB      TipoEntidadMemoria = "db"
	EntidadMemoriaInfra   TipoEntidadMemoria = "infra"
	EntidadMemoriaRegla   TipoEntidadMemoria = "regla"
)

type EntidadMemoria struct {
	ID                 int64     `json:"id"`
	Nombre             string    `json:"nombre"`
	Tipo               string    `json:"tipo"`
	ValorJSON          string    `json:"valor_json"`
	MetadataJSON       string    `json:"metadata_json"`
	UltimaVerificacion time.Time `json:"ultima_verificacion"`
	VerificadoPor      string    `json:"verificado_por"`
	ProyectoID         *int64    `json:"proyecto_id,omitempty"`
}

type FiltroEntidadesMemoria struct {
	ProyectoID *int64
	Tipo       *string
}

func UpsertEntidadMemoria(entidad *EntidadMemoria) (int64, error) {
	if entidad == nil {
		return 0, sql.ErrNoRows
	}
	nombre := strings.TrimSpace(entidad.Nombre)
	tipo := strings.TrimSpace(entidad.Tipo)
	if nombre == "" || !tipoEntidadMemoriaValido(tipo) {
		return 0, fmt.Errorf("nombre y tipo de entidad son obligatorios")
	}
	if strings.TrimSpace(entidad.ValorJSON) == "" {
		entidad.ValorJSON = "{}"
	}
	if strings.TrimSpace(entidad.MetadataJSON) == "" {
		entidad.MetadataJSON = "{}"
	}
	entidad.Nombre = nombre
	entidad.Tipo = tipo
	entidad.VerificadoPor = strings.TrimSpace(entidad.VerificadoPor)
	if entidad.UltimaVerificacion.IsZero() {
		entidad.UltimaVerificacion = time.Now().UTC()
	}

	existente, err := GetEntidadMemoria(nombre, entidad.ProyectoID)
	if err != nil {
		return 0, err
	}
	if existente == nil {
		id, err := insertReturningID(`
			INSERT INTO entidades_memoria (
				nombre, tipo, valor_json, metadata_json, ultima_verificacion, verificado_por, proyecto_id
			) VALUES (?,?,?,?,?,?,?)`,
			entidad.Nombre, entidad.Tipo, entidad.ValorJSON, entidad.MetadataJSON,
			entidad.UltimaVerificacion, entidad.VerificadoPor, entidad.ProyectoID,
		)
		if err != nil {
			return 0, err
		}
		return id, nil
	}

	_, err = DB.Exec(`
		UPDATE entidades_memoria
		SET tipo = ?,
		    valor_json = ?,
		    metadata_json = ?,
		    ultima_verificacion = ?,
		    verificado_por = ?
		WHERE id = ?`,
		entidad.Tipo, entidad.ValorJSON, entidad.MetadataJSON,
		entidad.UltimaVerificacion, entidad.VerificadoPor, existente.ID,
	)
	if err != nil {
		return 0, err
	}
	return existente.ID, nil
}

func GetEntidadMemoria(nombre string, proyectoID *int64) (*EntidadMemoria, error) {
	q := entidadMemoriaSelectBase() + ` WHERE nombre = ?`
	args := []any{strings.TrimSpace(nombre)}
	if proyectoID != nil {
		q += ` AND proyecto_id = ?`
		args = append(args, *proyectoID)
	} else {
		q += ` AND proyecto_id IS NULL`
	}
	q += ` ORDER BY id DESC LIMIT 1`

	entidad, err := scanEntidadMemoria(DB.QueryRow(q, args...))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return entidad, err
}

func ListarEntidadesMemoria(filter FiltroEntidadesMemoria) ([]*EntidadMemoria, error) {
	q := entidadMemoriaSelectBase() + ` WHERE 1=1`
	args := make([]any, 0, 2)
	if filter.ProyectoID != nil {
		q += ` AND proyecto_id = ?`
		args = append(args, *filter.ProyectoID)
	}
	if filter.Tipo != nil {
		q += ` AND tipo = ?`
		args = append(args, strings.TrimSpace(*filter.Tipo))
	}
	q += ` ORDER BY nombre ASC, id DESC`

	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*EntidadMemoria
	for rows.Next() {
		entidad, err := scanEntidadMemoria(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, entidad)
	}
	return out, rows.Err()
}

func entidadMemoriaSelectBase() string {
	return `
		SELECT id, nombre, tipo, valor_json, metadata_json, ultima_verificacion, verificado_por, proyecto_id
		FROM entidades_memoria`
}

func scanEntidadMemoria(s scanner) (*EntidadMemoria, error) {
	var entidad EntidadMemoria
	var proyectoID sql.NullInt64
	if err := s.Scan(
		&entidad.ID,
		&entidad.Nombre,
		&entidad.Tipo,
		&entidad.ValorJSON,
		&entidad.MetadataJSON,
		&entidad.UltimaVerificacion,
		&entidad.VerificadoPor,
		&proyectoID,
	); err != nil {
		return nil, err
	}
	if proyectoID.Valid {
		entidad.ProyectoID = &proyectoID.Int64
	}
	return &entidad, nil
}

func tipoEntidadMemoriaValido(tipo string) bool {
	switch strings.TrimSpace(tipo) {
	case string(EntidadMemoriaNegocio), string(EntidadMemoriaAPI), string(EntidadMemoriaDB), string(EntidadMemoriaInfra), string(EntidadMemoriaRegla):
		return true
	default:
		return false
	}
}
