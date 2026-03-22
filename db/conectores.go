package db

import (
	"database/sql"
	"fmt"
	"time"
)

type Conector struct {
	ID           int64
	Slug         string
	Nombre       string
	Transporte   string
	Comando      string
	ArgsJSON     string
	EnvJSON      string
	MetadataJSON string
	Activo       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func UpsertConector(c *Conector) (int64, error) {
	if c == nil {
		return 0, fmt.Errorf("conector nil")
	}
	if c.Slug == "" {
		return 0, fmt.Errorf("slug obligatorio")
	}
	if c.Nombre == "" {
		return 0, fmt.Errorf("nombre obligatorio")
	}
	if c.Transporte == "" {
		c.Transporte = "cli"
	}

	var existenteID int64
	err := DB.QueryRow(`SELECT id FROM conectores WHERE slug = ?`, c.Slug).Scan(&existenteID)
	switch err {
	case nil:
		_, err = DB.Exec(`
			UPDATE conectores
			SET nombre=?, transporte=?, comando=?, args_json=?, env_json=?, metadata_json=?, activo=?
			WHERE id=?`,
			c.Nombre, c.Transporte, c.Comando, c.ArgsJSON, c.EnvJSON, c.MetadataJSON, c.Activo, existenteID,
		)
		if err != nil {
			return 0, err
		}
		return existenteID, nil
	case sql.ErrNoRows:
		res, err := DB.Exec(`
			INSERT INTO conectores (slug, nombre, transporte, comando, args_json, env_json, metadata_json, activo)
			VALUES (?,?,?,?,?,?,?,?)`,
			c.Slug, c.Nombre, c.Transporte, c.Comando, c.ArgsJSON, c.EnvJSON, c.MetadataJSON, c.Activo,
		)
		if err != nil {
			return 0, err
		}
		id, _ := res.LastInsertId()
		return id, nil
	default:
		return 0, err
	}
}

func GetConector(ref string) (*Conector, error) {
	if ref == "" {
		return nil, fmt.Errorf("referencia de conector vacía")
	}
	row := DB.QueryRow(`
		SELECT id, slug, nombre, transporte, comando, args_json, env_json, metadata_json, activo, created_at, updated_at
		FROM conectores
		WHERE slug = ? OR CAST(id AS TEXT) = ?`, ref, ref)
	return escanearConector(row)
}

func ListarConectores() ([]*Conector, error) {
	rows, err := DB.Query(`
		SELECT id, slug, nombre, transporte, comando, args_json, env_json, metadata_json, activo, created_at, updated_at
		FROM conectores
		ORDER BY activo DESC, slug`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Conector
	for rows.Next() {
		c, err := escanearConector(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func escanearConector(s scanner) (*Conector, error) {
	var c Conector
	err := s.Scan(&c.ID, &c.Slug, &c.Nombre, &c.Transporte, &c.Comando, &c.ArgsJSON, &c.EnvJSON, &c.MetadataJSON, &c.Activo, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
