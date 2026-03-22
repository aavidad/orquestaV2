package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type TipoProyecto string

const (
	ProyectoRaiz  TipoProyecto = "raiz"
	ProyectoGrupo TipoProyecto = "grupo"
	ProyectoRepo  TipoProyecto = "repo"
)

type Proyecto struct {
	ID        int64
	Slug      string
	Nombre    string
	RutaAbs   string
	Tipo      TipoProyecto
	ParentID  *int64
	Activo    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type FiltroProyectos struct {
	Tipo     *TipoProyecto
	ParentID *int64
	Activo   *bool
}

func UpsertProyecto(p *Proyecto) (int64, error) {
	if p == nil {
		return 0, fmt.Errorf("proyecto nil")
	}
	if strings.TrimSpace(p.RutaAbs) == "" {
		return 0, fmt.Errorf("ruta_abs obligatoria")
	}
	abs, err := filepath.Abs(p.RutaAbs)
	if err != nil {
		return 0, err
	}
	p.RutaAbs = filepath.Clean(abs)
	if strings.TrimSpace(p.Nombre) == "" {
		p.Nombre = filepath.Base(p.RutaAbs)
	}
	if strings.TrimSpace(p.Slug) == "" {
		p.Slug = slugProyecto(p.Nombre)
	}
	if strings.TrimSpace(string(p.Tipo)) == "" {
		p.Tipo = ProyectoRepo
	}

	var existenteID int64
	err = DB.QueryRow(`SELECT id FROM proyectos WHERE slug = ? OR ruta_abs = ?`, p.Slug, p.RutaAbs).Scan(&existenteID)
	switch err {
	case nil:
		_, err = DB.Exec(`
			UPDATE proyectos
			SET slug=?, nombre=?, ruta_abs=?, tipo=?, parent_id=?, activo=?
			WHERE id=?`,
			p.Slug, p.Nombre, p.RutaAbs, p.Tipo, p.ParentID, p.Activo, existenteID,
		)
		if err != nil {
			return 0, err
		}
		return existenteID, nil
	case sql.ErrNoRows:
		res, err := DB.Exec(`
			INSERT INTO proyectos (slug, nombre, ruta_abs, tipo, parent_id, activo)
			VALUES (?,?,?,?,?,?)`,
			p.Slug, p.Nombre, p.RutaAbs, p.Tipo, p.ParentID, p.Activo,
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

func GetProyecto(ref string) (*Proyecto, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("referencia de proyecto vacía")
	}

	q := `
		SELECT id, slug, nombre, ruta_abs, tipo, parent_id, activo, created_at, updated_at
		FROM proyectos
		WHERE slug = ? OR ruta_abs = ?`
	args := []any{ref, filepath.Clean(ref)}

	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		q += ` OR id = ?`
		args = append(args, id)
	}

	return escanearProyecto(DB.QueryRow(q, args...))
}

func ListarProyectos(f FiltroProyectos) ([]*Proyecto, error) {
	q := `
		SELECT id, slug, nombre, ruta_abs, tipo, parent_id, activo, created_at, updated_at
		FROM proyectos WHERE 1=1`
	var args []any
	if f.Tipo != nil {
		q += ` AND tipo = ?`
		args = append(args, *f.Tipo)
	}
	if f.ParentID != nil {
		q += ` AND parent_id = ?`
		args = append(args, *f.ParentID)
	}
	if f.Activo != nil {
		q += ` AND activo = ?`
		args = append(args, *f.Activo)
	}
	q += ` ORDER BY ruta_abs`

	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Proyecto
	for rows.Next() {
		p, err := escanearProyecto(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func DescubrirProyectos(root string) ([]*Proyecto, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		var err error
		root, err = WorkspaceRoot()
		if err != nil {
			return nil, err
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	root = filepath.Clean(abs)

	raiz := &Proyecto{
		Slug:    slugProyecto(filepath.Base(root)),
		Nombre:  filepath.Base(root),
		RutaAbs: root,
		Tipo:    ProyectoRaiz,
		Activo:  true,
	}
	id, err := UpsertProyecto(raiz)
	if err != nil {
		return nil, err
	}
	raiz.ID = id

	resultados := []*Proyecto{raiz}
	subdirs, err := listarSubdirectorios(root)
	if err != nil {
		return nil, err
	}
	for _, subdir := range subdirs {
		disc, err := descubrirProyectoRec(subdir, &raiz.ID, false)
		if err != nil {
			return nil, err
		}
		resultados = append(resultados, disc...)
	}

	sort.Slice(resultados, func(i, j int) bool { return resultados[i].RutaAbs < resultados[j].RutaAbs })
	return resultados, nil
}

func WorkspaceRoot() (string, error) {
	if v := strings.TrimSpace(os.Getenv("ORQUESTA_WORKSPACE_ROOT")); v != "" {
		return filepath.Abs(v)
	}
	if v, err := ConfigGet("workspace_root"); err == nil && strings.TrimSpace(v) != "" {
		return filepath.Abs(v)
	}
	return "", fmt.Errorf("workspace_root no configurado; usa 'orquesta config set workspace_root <ruta>' o 'orquesta proyecto descubrir <ruta>'")
}

func descubrirProyectoRec(path string, parentID *int64, esRaiz bool) ([]*Proyecto, error) {
	path = filepath.Clean(path)
	subdirs, err := listarSubdirectorios(path)
	if err != nil {
		return nil, err
	}

	var hijosConProyecto []string
	for _, subdir := range subdirs {
		if contieneProyecto(subdir) {
			hijosConProyecto = append(hijosConProyecto, subdir)
		}
	}

	tieneMarcadores, err := tieneMarcadoresRepo(path)
	if err != nil {
		return nil, err
	}

	var tipo TipoProyecto
	switch {
	case esRaiz:
		tipo = ProyectoRaiz
	case len(hijosConProyecto) > 0:
		tipo = ProyectoGrupo
	case tieneMarcadores:
		tipo = ProyectoRepo
	default:
		return nil, nil
	}

	p := &Proyecto{
		Slug:     slugProyecto(filepath.Base(path)),
		Nombre:   filepath.Base(path),
		RutaAbs:  path,
		Tipo:     tipo,
		ParentID: parentID,
		Activo:   true,
	}
	id, err := UpsertProyecto(p)
	if err != nil {
		return nil, err
	}
	p.ID = id

	resultados := []*Proyecto{p}
	if tipo == ProyectoRepo {
		return resultados, nil
	}

	for _, hijo := range hijosConProyecto {
		disc, err := descubrirProyectoRec(hijo, &id, false)
		if err != nil {
			return nil, err
		}
		resultados = append(resultados, disc...)
	}
	return resultados, nil
}

func listarSubdirectorios(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		nombre := entry.Name()
		if strings.HasPrefix(nombre, ".") {
			continue
		}
		if nombre == "node_modules" || nombre == "vendor" || nombre == "dist" || nombre == "build" || nombre == "tmp" {
			continue
		}
		out = append(out, filepath.Join(path, nombre))
	}
	sort.Strings(out)
	return out, nil
}

func contieneProyecto(path string) bool {
	ok, err := tieneMarcadoresRepo(path)
	if err == nil && ok {
		return true
	}
	subdirs, err := listarSubdirectorios(path)
	if err != nil {
		return false
	}
	for _, subdir := range subdirs {
		ok, err := tieneMarcadoresRepo(subdir)
		if err == nil && ok {
			return true
		}
	}
	return false
}

func tieneMarcadoresRepo(path string) (bool, error) {
	marcadores := []string{
		".git",
		"go.mod",
		"package.json",
		"pyproject.toml",
		"Cargo.toml",
		"Makefile",
	}
	for _, marcador := range marcadores {
		info, err := os.Stat(filepath.Join(path, marcador))
		if err == nil {
			if marcador == ".git" || !info.IsDir() {
				return true, nil
			}
		}
	}
	return false, nil
}

func slugProyecto(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return "proyecto"
	}
	var b strings.Builder
	ultimoGuion := false
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			ultimoGuion = false
			continue
		}
		if !ultimoGuion {
			b.WriteRune('-')
			ultimoGuion = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "proyecto"
	}
	return out
}

func escanearProyecto(s scanner) (*Proyecto, error) {
	var p Proyecto
	var parentID sql.NullInt64
	err := s.Scan(&p.ID, &p.Slug, &p.Nombre, &p.RutaAbs, &p.Tipo, &parentID, &p.Activo, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		p.ParentID = &parentID.Int64
	}
	return &p, nil
}
