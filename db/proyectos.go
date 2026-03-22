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
	"strconv"
	"strings"
)

type ProyectoRef struct {
	ID     int64
	Slug   string
	Nombre string
}

func GetProyectoRef(selector string) (*ProyectoRef, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return nil, sql.ErrNoRows
	}

	row := DB.QueryRow(`
		SELECT id, slug, nombre
		FROM proyectos
		WHERE slug = ? OR nombre = ?`, selector, selector)
	ref, err := escanearProyectoRef(row)
	if err == nil {
		return ref, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	if id, parseErr := strconv.ParseInt(selector, 10, 64); parseErr == nil {
		row = DB.QueryRow(`SELECT id, slug, nombre FROM proyectos WHERE id = ?`, id)
		return escanearProyectoRef(row)
	}
	return nil, sql.ErrNoRows
}

func EnsureProyectoRef(selector string) (*ProyectoRef, error) {
	ref, err := GetProyectoRef(selector)
	if err == nil {
		return ref, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	slug := strings.TrimSpace(selector)
	if slug == "" {
		return nil, fmt.Errorf("selector de proyecto obligatorio")
	}
	_, err = DB.Exec(`
		INSERT INTO proyectos (slug, nombre, ruta_abs, tipo, activo)
		VALUES (?, ?, '', 'repo', 1)`,
		slug, slug,
	)
	if err != nil {
		return nil, err
	}
	return GetProyectoRef(slug)
}

func escanearProyectoRef(s scanner) (*ProyectoRef, error) {
	ref := &ProyectoRef{}
	if err := s.Scan(&ref.ID, &ref.Slug, &ref.Nombre); err != nil {
		return nil, err
	}
	return ref, nil
}
