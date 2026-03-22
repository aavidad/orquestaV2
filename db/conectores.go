/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"fmt"
	"strings"
	"time"
)

type ConectorForge struct {
	ID           int64
	Slug         string
	Tipo         string
	Owner        string
	OwnerKind    string
	APIBaseURL   string
	TokenEnv     string
	MetadataJSON string
	Activo       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type SubproyectoRemoto struct {
	ID            int64
	Proyecto      string
	ConectorSlug  string
	ForgeTipo     string
	Owner         string
	RepoName      string
	RepoFullName  string
	Visibility    string
	Descripcion   string
	HTMLURL       string
	CloneURL      string
	SSHURL        string
	DefaultBranch string
	Estado        string
	MetadataJSON  string
	RegistradoPor string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func RegistrarConectorForge(c *ConectorForge) (int64, error) {
	if c == nil {
		return 0, fmt.Errorf("conector nulo")
	}
	c.Slug = strings.TrimSpace(c.Slug)
	c.Tipo = strings.TrimSpace(c.Tipo)
	c.Owner = strings.TrimSpace(c.Owner)
	c.OwnerKind = strings.TrimSpace(c.OwnerKind)
	c.APIBaseURL = strings.TrimSpace(c.APIBaseURL)
	c.TokenEnv = strings.TrimSpace(c.TokenEnv)
	c.MetadataJSON = strings.TrimSpace(c.MetadataJSON)
	if c.Slug == "" || c.Tipo == "" || c.Owner == "" {
		return 0, fmt.Errorf("slug, tipo y owner son obligatorios")
	}
	if c.OwnerKind == "" {
		c.OwnerKind = "org"
	}
	if c.MetadataJSON == "" {
		c.MetadataJSON = "{}"
	}
	if c.APIBaseURL == "" && c.Tipo == "github" {
		c.APIBaseURL = "https://api.github.com"
	}

	_, err := DB.Exec(`
		INSERT INTO conectores_forge (slug, tipo, owner, owner_kind, api_base_url, token_env, metadata_json, activo)
		VALUES (?,?,?,?,?,?,?,?)
		ON CONFLICT(slug) DO UPDATE SET
			tipo=excluded.tipo,
			owner=excluded.owner,
			owner_kind=excluded.owner_kind,
			api_base_url=excluded.api_base_url,
			token_env=excluded.token_env,
			metadata_json=excluded.metadata_json,
			activo=excluded.activo`,
		c.Slug, c.Tipo, c.Owner, c.OwnerKind, c.APIBaseURL, c.TokenEnv, c.MetadataJSON, c.Activo,
	)
	if err != nil {
		return 0, err
	}
	var id int64
	if err := DB.QueryRow(`SELECT id FROM conectores_forge WHERE slug = ?`, c.Slug).Scan(&id); err != nil {
		return 0, err
	}
	Audit("orquesta", "registrar_conector_forge", "conector_forge", id, c.Slug)
	return id, nil
}

func GetConectorForge(slug string) (*ConectorForge, error) {
	row := DB.QueryRow(`
		SELECT id, slug, tipo, owner, owner_kind, api_base_url, token_env, metadata_json, activo, created_at, updated_at
		FROM conectores_forge
		WHERE slug = ?`, strings.TrimSpace(slug))
	return escanearConectorForge(row)
}

func ListarConectoresForge(tipo string, incluirInactivos bool) ([]*ConectorForge, error) {
	q := `
		SELECT id, slug, tipo, owner, owner_kind, api_base_url, token_env, metadata_json, activo, created_at, updated_at
		FROM conectores_forge
		WHERE 1=1`
	args := []any{}
	if strings.TrimSpace(tipo) != "" {
		q += ` AND tipo = ?`
		args = append(args, strings.TrimSpace(tipo))
	}
	if !incluirInactivos {
		q += ` AND activo = 1`
	}
	q += ` ORDER BY tipo, slug`

	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*ConectorForge
	for rows.Next() {
		item, err := escanearConectorForge(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func RegistrarSubproyectoRemoto(s *SubproyectoRemoto) (int64, error) {
	if s == nil {
		return 0, fmt.Errorf("subproyecto remoto nulo")
	}
	s.Proyecto = strings.TrimSpace(s.Proyecto)
	s.ConectorSlug = strings.TrimSpace(s.ConectorSlug)
	s.ForgeTipo = strings.TrimSpace(s.ForgeTipo)
	s.Owner = strings.TrimSpace(s.Owner)
	s.RepoName = strings.TrimSpace(s.RepoName)
	s.RepoFullName = strings.TrimSpace(s.RepoFullName)
	s.Visibility = strings.TrimSpace(s.Visibility)
	s.Descripcion = strings.TrimSpace(s.Descripcion)
	s.HTMLURL = strings.TrimSpace(s.HTMLURL)
	s.CloneURL = strings.TrimSpace(s.CloneURL)
	s.SSHURL = strings.TrimSpace(s.SSHURL)
	s.DefaultBranch = strings.TrimSpace(s.DefaultBranch)
	s.Estado = strings.TrimSpace(s.Estado)
	s.MetadataJSON = strings.TrimSpace(s.MetadataJSON)
	s.RegistradoPor = strings.TrimSpace(s.RegistradoPor)
	if s.Proyecto == "" || s.ConectorSlug == "" || s.ForgeTipo == "" || s.Owner == "" || s.RepoName == "" || s.RepoFullName == "" {
		return 0, fmt.Errorf("proyecto, conector, forge, owner, repo_name y repo_full_name son obligatorios")
	}
	if s.Visibility == "" {
		s.Visibility = "private"
	}
	if s.Estado == "" {
		s.Estado = "registrado"
	}
	if s.MetadataJSON == "" {
		s.MetadataJSON = "{}"
	}
	if s.RegistradoPor == "" {
		s.RegistradoPor = "orquesta"
	}

	_, err := DB.Exec(`
		INSERT INTO subproyectos_remotos (
			proyecto, conector_slug, forge_tipo, owner, repo_name, repo_full_name,
			visibility, descripcion, html_url, clone_url, ssh_url, default_branch,
			estado, metadata_json, registrado_por
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(conector_slug, repo_full_name) DO UPDATE SET
			proyecto=excluded.proyecto,
			forge_tipo=excluded.forge_tipo,
			owner=excluded.owner,
			repo_name=excluded.repo_name,
			visibility=excluded.visibility,
			descripcion=excluded.descripcion,
			html_url=excluded.html_url,
			clone_url=excluded.clone_url,
			ssh_url=excluded.ssh_url,
			default_branch=excluded.default_branch,
			estado=excluded.estado,
			metadata_json=excluded.metadata_json,
			registrado_por=excluded.registrado_por`,
		s.Proyecto, s.ConectorSlug, s.ForgeTipo, s.Owner, s.RepoName, s.RepoFullName,
		s.Visibility, s.Descripcion, s.HTMLURL, s.CloneURL, s.SSHURL, s.DefaultBranch,
		s.Estado, s.MetadataJSON, s.RegistradoPor,
	)
	if err != nil {
		return 0, err
	}
	var id int64
	if err := DB.QueryRow(`SELECT id FROM subproyectos_remotos WHERE conector_slug = ? AND repo_full_name = ?`, s.ConectorSlug, s.RepoFullName).Scan(&id); err != nil {
		return 0, err
	}
	Audit(s.RegistradoPor, "registrar_subproyecto_remoto", "subproyecto_remoto", id, s.RepoFullName)
	return id, nil
}

func ListarSubproyectosRemotos(proyecto string) ([]*SubproyectoRemoto, error) {
	q := `
		SELECT id, proyecto, conector_slug, forge_tipo, owner, repo_name, repo_full_name,
		       visibility, descripcion, html_url, clone_url, ssh_url, default_branch,
		       estado, metadata_json, registrado_por, created_at, updated_at
		FROM subproyectos_remotos`
	args := []any{}
	if strings.TrimSpace(proyecto) != "" {
		q += ` WHERE proyecto = ?`
		args = append(args, strings.TrimSpace(proyecto))
	}
	q += ` ORDER BY proyecto, repo_full_name`

	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*SubproyectoRemoto
	for rows.Next() {
		item, err := escanearSubproyectoRemoto(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func escanearConectorForge(s scanner) (*ConectorForge, error) {
	item := &ConectorForge{}
	if err := s.Scan(
		&item.ID, &item.Slug, &item.Tipo, &item.Owner, &item.OwnerKind,
		&item.APIBaseURL, &item.TokenEnv, &item.MetadataJSON, &item.Activo,
		&item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return item, nil
}

func escanearSubproyectoRemoto(s scanner) (*SubproyectoRemoto, error) {
	item := &SubproyectoRemoto{}
	if err := s.Scan(
		&item.ID, &item.Proyecto, &item.ConectorSlug, &item.ForgeTipo, &item.Owner,
		&item.RepoName, &item.RepoFullName, &item.Visibility, &item.Descripcion,
		&item.HTMLURL, &item.CloneURL, &item.SSHURL, &item.DefaultBranch,
		&item.Estado, &item.MetadataJSON, &item.RegistradoPor, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return item, nil
}
