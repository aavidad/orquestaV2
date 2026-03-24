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

type PermisoEdicionCatalogo struct {
	ID             int64
	Entidad        string
	Rol            string
	Alcance        string
	PuedeCrear     bool
	PuedeEditar    bool
	PuedeActivar   bool
	PuedeVersionar bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ReglaVersion struct {
	ID          int64
	ReglaID     int64
	VersionNum  int64
	TipoAgente  string
	Categoria   string
	Titulo      string
	Descripcion string
	Activa      bool
	Actor       string
	Accion      string
	CreatedAt   time.Time
}

type SkillVersion struct {
	ID          int64
	SkillID     int64
	VersionNum  int64
	TipoAgente  string
	Nombre      string
	Descripcion string
	CuandoUsar  string
	Activa      bool
	Actor       string
	Accion      string
	CreatedAt   time.Time
}

type WorkflowVersion struct {
	ID          int64
	WorkflowID  int64
	VersionNum  int64
	TipoAgente  string
	Nombre      string
	Descripcion string
	Pasos       string
	Activo      bool
	Actor       string
	Accion      string
	CreatedAt   time.Time
}

func ListarPermisosEdicionCatalogo(entidad string) ([]*PermisoEdicionCatalogo, error) {
	q := `
		SELECT id, entidad, rol, alcance, puede_crear, puede_editar, puede_activar, puede_versionar, created_at, updated_at
		FROM catalogo_edicion_permisos
		WHERE 1=1`
	args := []any{}
	if strings.TrimSpace(entidad) != "" {
		q += ` AND entidad = ?`
		args = append(args, strings.TrimSpace(entidad))
	}
	q += ` ORDER BY entidad, rol`

	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*PermisoEdicionCatalogo
	for rows.Next() {
		p, err := escanearPermisoEdicionCatalogo(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func GuardarPermisoEdicionCatalogo(actor string, permiso *PermisoEdicionCatalogo) error {
	if permiso == nil {
		return fmt.Errorf("permiso nulo")
	}
	permiso.Entidad = strings.TrimSpace(permiso.Entidad)
	permiso.Rol = strings.TrimSpace(permiso.Rol)
	permiso.Alcance = strings.TrimSpace(permiso.Alcance)
	if permiso.Entidad == "" || permiso.Rol == "" || permiso.Alcance == "" {
		return fmt.Errorf("entidad, rol y alcance son obligatorios")
	}
	rolActor, habilitado, err := rolAgente(actor)
	if err != nil {
		return err
	}
	if !habilitado {
		return fmt.Errorf("el actor %s está retirado", actor)
	}
	if rolActor != "admin" {
		return fmt.Errorf("solo admin puede modificar permisos del catálogo")
	}
	if permiso.Alcance != "mismo_rol" && permiso.Alcance != "todos" {
		return fmt.Errorf("alcance inválido: %s", permiso.Alcance)
	}
	_, err = DB.Exec(`
		INSERT INTO catalogo_edicion_permisos
			(entidad, rol, alcance, puede_crear, puede_editar, puede_activar, puede_versionar, updated_at)
		VALUES (?,?,?,?,?,?,?,CURRENT_TIMESTAMP)
		ON CONFLICT(entidad, rol) DO UPDATE SET
			alcance=excluded.alcance,
			puede_crear=excluded.puede_crear,
			puede_editar=excluded.puede_editar,
			puede_activar=excluded.puede_activar,
			puede_versionar=excluded.puede_versionar,
			updated_at=CURRENT_TIMESTAMP`,
		permiso.Entidad,
		permiso.Rol,
		permiso.Alcance,
		func() int {
			if permiso.PuedeCrear {
				return 1
			}
			return 0
		}(),
		func() int {
			if permiso.PuedeEditar {
				return 1
			}
			return 0
		}(),
		func() int {
			if permiso.PuedeActivar {
				return 1
			}
			return 0
		}(),
		func() int {
			if permiso.PuedeVersionar {
				return 1
			}
			return 0
		}(),
	)
	return err
}

func validarPermisoEdicionCatalogo(actor, entidad, targetRol, operacion string) error {
	actor = strings.TrimSpace(actor)
	entidad = strings.TrimSpace(entidad)
	targetRol = strings.TrimSpace(targetRol)
	operacion = strings.TrimSpace(operacion)

	if actor == "" {
		return fmt.Errorf("actor obligatorio para %s %s", operacion, entidad)
	}
	if entidad == "" || targetRol == "" || operacion == "" {
		return fmt.Errorf("entidad, rol objetivo y operacion son obligatorios")
	}

	rolActor, habilitado, err := rolAgente(actor)
	if err != nil {
		return err
	}
	if !habilitado {
		return fmt.Errorf("el actor %s está retirado", actor)
	}

	if rolActor == "admin" {
		return nil
	}

	permiso, err := permisoEdicionCatalogoPorRol(entidad, rolActor)
	if err != nil {
		return err
	}
	if permiso == nil {
		return fmt.Errorf("no hay permisos de edición para %s en %s", rolActor, entidad)
	}

	if permiso.Alcance == "mismo_rol" && targetRol != rolActor {
		return fmt.Errorf("el alcance de %s sobre %s es solo para su propio rol", rolActor, entidad)
	}

	switch operacion {
	case "crear":
		if !permiso.PuedeCrear {
			return fmt.Errorf("el rol %s no puede crear %s", rolActor, entidad)
		}
	case "editar":
		if !permiso.PuedeEditar {
			return fmt.Errorf("el rol %s no puede editar %s", rolActor, entidad)
		}
	case "activar":
		if !permiso.PuedeActivar {
			return fmt.Errorf("el rol %s no puede activar %s", rolActor, entidad)
		}
	case "versionar":
		if !permiso.PuedeVersionar {
			return fmt.Errorf("el rol %s no puede versionar %s", rolActor, entidad)
		}
	default:
		return fmt.Errorf("operacion %s no soportada", operacion)
	}
	return nil
}

func permisoEdicionCatalogoPorRol(entidad, rol string) (*PermisoEdicionCatalogo, error) {
	row := DB.QueryRow(`
		SELECT id, entidad, rol, alcance, puede_crear, puede_editar, puede_activar, puede_versionar, created_at, updated_at
		FROM catalogo_edicion_permisos
		WHERE entidad = ? AND rol = ?`, strings.TrimSpace(entidad), strings.TrimSpace(rol))
	return escanearPermisoEdicionCatalogo(row)
}

func rolAgente(nombre string) (string, bool, error) {
	var rol string
	var habilitado bool
	err := DB.QueryRow(`SELECT rol, habilitado FROM agentes WHERE nombre = ?`, strings.TrimSpace(nombre)).Scan(&rol, &habilitado)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", false, fmt.Errorf("agente '%s' no registrado", nombre)
		}
		return "", false, err
	}
	return rol, habilitado, nil
}

func ListarVersionesRegla(reglaID int64) ([]*ReglaVersion, error) {
	rows, err := DB.Query(`
		SELECT id, regla_id, version_num, tipo_agente, categoria, titulo, descripcion, activa, actor, accion, created_at
		FROM reglas_versiones
		WHERE regla_id = ?
		ORDER BY version_num`, reglaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*ReglaVersion
	for rows.Next() {
		v, err := escanearReglaVersion(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

func ListarVersionesSkill(skillID int64) ([]*SkillVersion, error) {
	rows, err := DB.Query(`
		SELECT id, skill_id, version_num, tipo_agente, nombre, descripcion, cuando_usar, activa, actor, accion, created_at
		FROM skills_versiones
		WHERE skill_id = ?
		ORDER BY version_num`, skillID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*SkillVersion
	for rows.Next() {
		v, err := escanearSkillVersion(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

func ListarVersionesWorkflow(workflowID int64) ([]*WorkflowVersion, error) {
	rows, err := DB.Query(`
		SELECT id, workflow_id, version_num, tipo_agente, nombre, descripcion, pasos, activo, actor, accion, created_at
		FROM workflows_versiones
		WHERE workflow_id = ?
		ORDER BY version_num`, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*WorkflowVersion
	for rows.Next() {
		v, err := escanearWorkflowVersion(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

func registrarVersionReglaTx(tx *sql.Tx, r *Regla, actor, accion string) error {
	var version int64
	if err := tx.QueryRow(`SELECT COALESCE(MAX(version_num),0)+1 FROM reglas_versiones WHERE regla_id = ?`, r.ID).Scan(&version); err != nil {
		return err
	}
	_, err := tx.Exec(`
		INSERT INTO reglas_versiones (regla_id, version_num, tipo_agente, categoria, titulo, descripcion, activa, actor, accion)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		r.ID, version, r.TipoAgente, r.Categoria, r.Titulo, r.Descripcion, r.Activa, actor, accion,
	)
	return err
}

func registrarVersionSkillTx(tx *sql.Tx, s *Skill, actor, accion string) error {
	var version int64
	if err := tx.QueryRow(`SELECT COALESCE(MAX(version_num),0)+1 FROM skills_versiones WHERE skill_id = ?`, s.ID).Scan(&version); err != nil {
		return err
	}
	_, err := tx.Exec(`
		INSERT INTO skills_versiones (skill_id, version_num, tipo_agente, nombre, descripcion, cuando_usar, activa, actor, accion)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		s.ID, version, s.TipoAgente, s.Nombre, s.Descripcion, s.CuandoUsar, s.Activa, actor, accion,
	)
	return err
}

func registrarVersionWorkflowTx(tx *sql.Tx, w *Workflow, actor, accion string) error {
	var version int64
	if err := tx.QueryRow(`SELECT COALESCE(MAX(version_num),0)+1 FROM workflows_versiones WHERE workflow_id = ?`, w.ID).Scan(&version); err != nil {
		return err
	}
	_, err := tx.Exec(`
		INSERT INTO workflows_versiones (workflow_id, version_num, tipo_agente, nombre, descripcion, pasos, activo, actor, accion)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		w.ID, version, w.TipoAgente, w.Nombre, w.Descripcion, w.Pasos, w.Activo, actor, accion,
	)
	return err
}

func escanearPermisoEdicionCatalogo(s scanner) (*PermisoEdicionCatalogo, error) {
	p := &PermisoEdicionCatalogo{}
	var puedeCrear, puedeEditar, puedeActivar, puedeVersionar int
	if err := s.Scan(&p.ID, &p.Entidad, &p.Rol, &p.Alcance, &puedeCrear, &puedeEditar, &puedeActivar, &puedeVersionar, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	p.PuedeCrear = puedeCrear == 1
	p.PuedeEditar = puedeEditar == 1
	p.PuedeActivar = puedeActivar == 1
	p.PuedeVersionar = puedeVersionar == 1
	return p, nil
}

func escanearReglaVersion(s scanner) (*ReglaVersion, error) {
	v := &ReglaVersion{}
	var activa int
	if err := s.Scan(&v.ID, &v.ReglaID, &v.VersionNum, &v.TipoAgente, &v.Categoria, &v.Titulo, &v.Descripcion, &activa, &v.Actor, &v.Accion, &v.CreatedAt); err != nil {
		return nil, err
	}
	v.Activa = activa == 1
	return v, nil
}

func escanearSkillVersion(s scanner) (*SkillVersion, error) {
	v := &SkillVersion{}
	var activa int
	if err := s.Scan(&v.ID, &v.SkillID, &v.VersionNum, &v.TipoAgente, &v.Nombre, &v.Descripcion, &v.CuandoUsar, &activa, &v.Actor, &v.Accion, &v.CreatedAt); err != nil {
		return nil, err
	}
	v.Activa = activa == 1
	return v, nil
}

func escanearWorkflowVersion(s scanner) (*WorkflowVersion, error) {
	v := &WorkflowVersion{}
	var activo int
	if err := s.Scan(&v.ID, &v.WorkflowID, &v.VersionNum, &v.TipoAgente, &v.Nombre, &v.Descripcion, &v.Pasos, &activo, &v.Actor, &v.Accion, &v.CreatedAt); err != nil {
		return nil, err
	}
	v.Activo = activo == 1
	return v, nil
}
