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

type DecisionProyecto struct {
	ID                      int64
	Proyecto                string
	Titulo                  string
	SolucionElegida         string
	Motivo                  string
	AlternativasDescartadas string
	Impacto                 string
	PropuestaCodigo         string
	TareaID                 *int64
	RegistradoPor           string
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

type DocumentoExterno struct {
	ID            int64
	Proyecto      string
	TipoDocumento string
	Ruta          string
	Resumen       string
	Estado        string
	RegistradoPor string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func RegistrarDecisionProyecto(d *DecisionProyecto) (int64, error) {
	if d == nil {
		return 0, fmt.Errorf("decision nula")
	}
	if strings.TrimSpace(d.Proyecto) == "" || strings.TrimSpace(d.Titulo) == "" {
		return 0, fmt.Errorf("proyecto y titulo son obligatorios")
	}
	if d.Impacto == "" {
		d.Impacto = "medio"
	}
	if d.PropuestaCodigo != "" {
		if _, err := GetPropuesta(d.PropuestaCodigo); err != nil {
			if err == sql.ErrNoRows {
				return 0, fmt.Errorf("propuesta %s no encontrada", d.PropuestaCodigo)
			}
			return 0, err
		}
	}
	if d.TareaID != nil {
		if _, err := GetTarea(*d.TareaID); err != nil {
			if err == sql.ErrNoRows {
				return 0, fmt.Errorf("tarea #%d no encontrada", *d.TareaID)
			}
			return 0, err
		}
	}
	res, err := DB.Exec(`
		INSERT INTO decisiones_proyecto (
			proyecto, titulo, solucion_elegida, motivo, alternativas_descartadas,
			impacto, propuesta_codigo, tarea_id, registrado_por
		) VALUES (?,?,?,?,?,?,?,?,?)`,
		d.Proyecto, d.Titulo, d.SolucionElegida, d.Motivo, d.AlternativasDescartadas,
		d.Impacto, d.PropuestaCodigo, nullableInt64(d.TareaID), d.RegistradoPor,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	Audit("orquesta", "registrar_decision_proyecto", "decision_proyecto", id, d.Titulo)
	return id, nil
}

func GetDecisionProyecto(id int64) (*DecisionProyecto, error) {
	row := DB.QueryRow(`
		SELECT id, proyecto, titulo, solucion_elegida, motivo, alternativas_descartadas,
		       impacto, propuesta_codigo, tarea_id, registrado_por, created_at, updated_at
		FROM decisiones_proyecto
		WHERE id = ?`, id)
	return escanearDecisionProyecto(row)
}

func ListarDecisionesProyecto(proyecto string) ([]*DecisionProyecto, error) {
	q := `
		SELECT id, proyecto, titulo, solucion_elegida, motivo, alternativas_descartadas,
		       impacto, propuesta_codigo, tarea_id, registrado_por, created_at, updated_at
		FROM decisiones_proyecto`
	args := []any{}
	if strings.TrimSpace(proyecto) != "" {
		q += ` WHERE proyecto = ?`
		args = append(args, strings.TrimSpace(proyecto))
	}
	q += ` ORDER BY proyecto, id DESC`

	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*DecisionProyecto
	for rows.Next() {
		d, err := escanearDecisionProyecto(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

func ActualizarDecisionProyecto(d *DecisionProyecto) error {
	if d == nil {
		return fmt.Errorf("decision nula")
	}
	if d.ID <= 0 {
		return fmt.Errorf("id de decision obligatorio")
	}
	if strings.TrimSpace(d.Proyecto) == "" || strings.TrimSpace(d.Titulo) == "" {
		return fmt.Errorf("proyecto y titulo son obligatorios")
	}
	if d.PropuestaCodigo != "" {
		if _, err := GetPropuesta(d.PropuestaCodigo); err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("propuesta %s no encontrada", d.PropuestaCodigo)
			}
			return err
		}
	}
	if d.TareaID != nil {
		if _, err := GetTarea(*d.TareaID); err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("tarea #%d no encontrada", *d.TareaID)
			}
			return err
		}
	}
	res, err := DB.Exec(`
		UPDATE decisiones_proyecto
		SET proyecto=?, titulo=?, solucion_elegida=?, motivo=?, alternativas_descartadas=?,
		    impacto=?, propuesta_codigo=?, tarea_id=?, registrado_por=?
		WHERE id=?`,
		d.Proyecto, d.Titulo, d.SolucionElegida, d.Motivo, d.AlternativasDescartadas,
		d.Impacto, d.PropuestaCodigo, nullableInt64(d.TareaID), d.RegistradoPor, d.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("decision #%d no encontrada", d.ID)
	}
	Audit("orquesta", "actualizar_decision_proyecto", "decision_proyecto", d.ID, d.Titulo)
	return nil
}

func RegistrarDocumentoExterno(doc *DocumentoExterno) (int64, error) {
	if doc == nil {
		return 0, fmt.Errorf("documento externo nulo")
	}
	if strings.TrimSpace(doc.Proyecto) == "" || strings.TrimSpace(doc.Ruta) == "" {
		return 0, fmt.Errorf("proyecto y ruta son obligatorios")
	}
	if doc.TipoDocumento == "" {
		doc.TipoDocumento = "referencia"
	}
	if doc.Estado == "" {
		doc.Estado = "vigente"
	}
	res, err := DB.Exec(`
		INSERT INTO documentacion_externa (proyecto, tipo_documento, ruta, resumen, estado, registrado_por)
		VALUES (?,?,?,?,?,?)`,
		doc.Proyecto, doc.TipoDocumento, doc.Ruta, doc.Resumen, doc.Estado, doc.RegistradoPor,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	Audit("orquesta", "registrar_documentacion_externa", "documentacion_externa", id, doc.Ruta)
	return id, nil
}

func GetDocumentoExterno(id int64) (*DocumentoExterno, error) {
	row := DB.QueryRow(`
		SELECT id, proyecto, tipo_documento, ruta, resumen, estado, registrado_por, created_at, updated_at
		FROM documentacion_externa
		WHERE id = ?`, id)
	return escanearDocumentoExterno(row)
}

func ListarDocumentosExternos(proyecto string) ([]*DocumentoExterno, error) {
	q := `
		SELECT id, proyecto, tipo_documento, ruta, resumen, estado, registrado_por, created_at, updated_at
		FROM documentacion_externa`
	args := []any{}
	if strings.TrimSpace(proyecto) != "" {
		q += ` WHERE proyecto = ?`
		args = append(args, strings.TrimSpace(proyecto))
	}
	q += ` ORDER BY proyecto, id DESC`

	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*DocumentoExterno
	for rows.Next() {
		doc, err := escanearDocumentoExterno(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, doc)
	}
	return list, rows.Err()
}

func ActualizarDocumentoExterno(doc *DocumentoExterno) error {
	if doc == nil {
		return fmt.Errorf("documento externo nulo")
	}
	if doc.ID <= 0 {
		return fmt.Errorf("id de documento obligatorio")
	}
	if strings.TrimSpace(doc.Proyecto) == "" || strings.TrimSpace(doc.Ruta) == "" {
		return fmt.Errorf("proyecto y ruta son obligatorios")
	}
	res, err := DB.Exec(`
		UPDATE documentacion_externa
		SET proyecto=?, tipo_documento=?, ruta=?, resumen=?, estado=?, registrado_por=?
		WHERE id=?`,
		doc.Proyecto, doc.TipoDocumento, doc.Ruta, doc.Resumen, doc.Estado, doc.RegistradoPor, doc.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("documento externo #%d no encontrado", doc.ID)
	}
	Audit("orquesta", "actualizar_documentacion_externa", "documentacion_externa", doc.ID, doc.Ruta)
	return nil
}

func escanearDecisionProyecto(s scanner) (*DecisionProyecto, error) {
	d := &DecisionProyecto{}
	var tareaID sql.NullInt64
	if err := s.Scan(
		&d.ID, &d.Proyecto, &d.Titulo, &d.SolucionElegida, &d.Motivo, &d.AlternativasDescartadas,
		&d.Impacto, &d.PropuestaCodigo, &tareaID, &d.RegistradoPor, &d.CreatedAt, &d.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if tareaID.Valid {
		d.TareaID = &tareaID.Int64
	}
	return d, nil
}

func escanearDocumentoExterno(s scanner) (*DocumentoExterno, error) {
	doc := &DocumentoExterno{}
	if err := s.Scan(
		&doc.ID, &doc.Proyecto, &doc.TipoDocumento, &doc.Ruta, &doc.Resumen,
		&doc.Estado, &doc.RegistradoPor, &doc.CreatedAt, &doc.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return doc, nil
}
