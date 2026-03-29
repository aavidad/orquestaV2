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

type ProjectMemoryRepository struct{}

type HistorialVotacionProyecto struct {
	PropuestaID  int64
	Codigo       string
	Titulo       string
	Tipo         string
	Estado       EstadoPropuesta
	PropuestoPor string
	CreatedAt    time.Time
	CerradaAt    *time.Time
	Acuerdo      int
	Desacuerdo   int
	Abstencion   int
	Pendiente    int
	Votos        []*Voto
}

type DecisionProyecto struct {
	ID           int64
	ProyectoID   int64
	Categoria    string
	Titulo       string
	Solucion     string
	Motivo       string
	Alternativas string
	Impacto      string
	Estado       string
	PropuestaID  *int64
	TareaID      *int64
	MetadataJSON string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type DocumentoExterno struct {
	ID            int64
	ProyectoID    int64
	TipoDocumento string
	Titulo        string
	RutaRef       string
	Resumen       string
	Estado        string
	Fuente        string
	PropuestaID   *int64
	TareaID       *int64
	MetadataJSON  string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func ListarHistorialVotacionesProyecto(proyectoID int64) ([]*HistorialVotacionProyecto, error) {
	rows, err := DB.Query(`
		SELECT id, codigo, titulo, tipo, estado, propuesto_por, created_at, cerrada_at
		FROM propuestas
		WHERE proyecto_id = ?
		ORDER BY created_at DESC, id DESC`, proyectoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*HistorialVotacionProyecto
	for rows.Next() {
		var item HistorialVotacionProyecto
		var cerradaAt sql.NullTime
		if err := rows.Scan(
			&item.PropuestaID, &item.Codigo, &item.Titulo, &item.Tipo, &item.Estado,
			&item.PropuestoPor, &item.CreatedAt, &cerradaAt,
		); err != nil {
			return nil, err
		}
		if cerradaAt.Valid {
			item.CerradaAt = &cerradaAt.Time
		}
		list = append(list, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, item := range list {
		item.Acuerdo, item.Desacuerdo, item.Abstencion, item.Pendiente, err = ContarVotos(item.PropuestaID)
		if err != nil {
			return nil, err
		}
		item.Votos, err = VotosDePropuesta(item.PropuestaID)
		if err != nil {
			return nil, err
		}
	}
	return list, nil
}

func GuardarDecisionProyecto(d *DecisionProyecto) (int64, error) {
	if d == nil {
		return 0, fmt.Errorf("decision obligatoria")
	}
	d.Categoria = strings.TrimSpace(d.Categoria)
	d.Titulo = strings.TrimSpace(d.Titulo)
	d.Solucion = strings.TrimSpace(d.Solucion)
	d.Motivo = strings.TrimSpace(d.Motivo)
	d.Alternativas = strings.TrimSpace(d.Alternativas)
	d.Impacto = strings.TrimSpace(d.Impacto)
	d.Estado = strings.TrimSpace(d.Estado)
	if d.Categoria == "" {
		d.Categoria = "general"
	}
	if d.Estado == "" {
		d.Estado = "vigente"
	}
	if d.MetadataJSON == "" {
		d.MetadataJSON = "{}"
	}
	if d.ProyectoID == 0 || d.Titulo == "" || d.Solucion == "" {
		return 0, fmt.Errorf("proyecto, titulo y solucion son obligatorios")
	}
	if !decisionProyectoEstadoValido(d.Estado) {
		return 0, fmt.Errorf("estado de decision invalido: %s", d.Estado)
	}

	id, err := execAndResolveID(
		upsertValuesSQL(
			"decisiones_proyecto",
			[]string{
				"proyecto_id", "categoria", "titulo", "solucion", "motivo", "alternativas", "impacto",
				"estado", "propuesta_id", "tarea_id", "metadata_json",
			},
			[]string{"proyecto_id", "titulo"},
			[]upsertAssignment{
				{Column: "categoria"},
				{Column: "solucion"},
				{Column: "motivo"},
				{Column: "alternativas"},
				{Column: "impacto"},
				{Column: "estado"},
				{Column: "propuesta_id"},
				{Column: "tarea_id"},
				{Column: "metadata_json"},
			},
		),
		[]any{
			d.ProyectoID, d.Categoria, d.Titulo, d.Solucion, d.Motivo, d.Alternativas,
			d.Impacto, d.Estado, d.PropuestaID, d.TareaID, d.MetadataJSON,
		},
		`SELECT id FROM decisiones_proyecto WHERE proyecto_id = ? AND titulo = ?`,
		d.ProyectoID, d.Titulo,
	)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func ListarDecisionesProyecto(proyectoID int64) ([]*DecisionProyecto, error) {
	rows, err := DB.Query(`
		SELECT id, proyecto_id, categoria, titulo, solucion, motivo, alternativas,
		       impacto, estado, propuesta_id, tarea_id, metadata_json, created_at, updated_at
		FROM decisiones_proyecto
		WHERE proyecto_id = ?
		ORDER BY categoria, titulo`, proyectoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*DecisionProyecto
	for rows.Next() {
		item, err := scanDecisionProyecto(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func GuardarDocumentoExterno(doc *DocumentoExterno) (int64, error) {
	if doc == nil {
		return 0, fmt.Errorf("documento obligatorio")
	}
	doc.TipoDocumento = strings.TrimSpace(doc.TipoDocumento)
	doc.Titulo = strings.TrimSpace(doc.Titulo)
	doc.RutaRef = strings.TrimSpace(doc.RutaRef)
	doc.Resumen = strings.TrimSpace(doc.Resumen)
	doc.Estado = strings.TrimSpace(doc.Estado)
	doc.Fuente = strings.TrimSpace(doc.Fuente)
	if doc.TipoDocumento == "" {
		doc.TipoDocumento = "markdown"
	}
	if doc.Estado == "" {
		doc.Estado = "vigente"
	}
	if doc.Fuente == "" {
		doc.Fuente = "manual"
	}
	if doc.MetadataJSON == "" {
		doc.MetadataJSON = "{}"
	}
	if doc.ProyectoID == 0 || doc.Titulo == "" || doc.RutaRef == "" || doc.Resumen == "" {
		return 0, fmt.Errorf("proyecto, titulo, ruta_ref y resumen son obligatorios")
	}
	if !documentoExternoEstadoValido(doc.Estado) {
		return 0, fmt.Errorf("estado de documento invalido: %s", doc.Estado)
	}
	if !documentoExternoFuenteValida(doc.Fuente) {
		return 0, fmt.Errorf("fuente de documento invalida: %s", doc.Fuente)
	}

	id, err := execAndResolveID(
		upsertValuesSQL(
			"documentos_externos",
			[]string{
				"proyecto_id", "tipo_documento", "titulo", "ruta_ref", "resumen", "estado", "fuente",
				"propuesta_id", "tarea_id", "metadata_json",
			},
			[]string{"proyecto_id", "ruta_ref"},
			[]upsertAssignment{
				{Column: "tipo_documento"},
				{Column: "titulo"},
				{Column: "resumen"},
				{Column: "estado"},
				{Column: "fuente"},
				{Column: "propuesta_id"},
				{Column: "tarea_id"},
				{Column: "metadata_json"},
			},
		),
		[]any{
			doc.ProyectoID, doc.TipoDocumento, doc.Titulo, doc.RutaRef, doc.Resumen,
			doc.Estado, doc.Fuente, doc.PropuestaID, doc.TareaID, doc.MetadataJSON,
		},
		`SELECT id FROM documentos_externos WHERE proyecto_id = ? AND ruta_ref = ?`,
		doc.ProyectoID, doc.RutaRef,
	)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func ListarDocumentosExternosProyecto(proyectoID int64) ([]*DocumentoExterno, error) {
	rows, err := DB.Query(`
		SELECT id, proyecto_id, tipo_documento, titulo, ruta_ref, resumen, estado,
		       fuente, propuesta_id, tarea_id, metadata_json, created_at, updated_at
		FROM documentos_externos
		WHERE proyecto_id = ?
		ORDER BY tipo_documento, titulo`, proyectoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*DocumentoExterno
	for rows.Next() {
		item, err := scanDocumentoExterno(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func (ProjectMemoryRepository) ResolveProyectoIDBySlug(slug string) (*int64, error) {
	return ResolveProyectoIDBySlug(slug)
}

func (ProjectMemoryRepository) ListProjects(activo *bool) ([]*Proyecto, error) {
	return ListarProyectosConRutaEfectiva(FiltroProyectos{Activo: activo}, "")
}

func (ProjectMemoryRepository) GetProjectBySlug(slug string) (*Proyecto, error) {
	return GetProyectoConRutaEfectiva(slug, "")
}

func (ProjectMemoryRepository) ListVoteHistoryByProjectID(proyectoID int64) ([]*HistorialVotacionProyecto, error) {
	return ListarHistorialVotacionesProyecto(proyectoID)
}

func (ProjectMemoryRepository) SaveDecision(d *DecisionProyecto) (int64, error) {
	return GuardarDecisionProyecto(d)
}

func (ProjectMemoryRepository) ListDecisionsByProjectID(proyectoID int64) ([]*DecisionProyecto, error) {
	return ListarDecisionesProyecto(proyectoID)
}

func (ProjectMemoryRepository) SaveExternalDoc(doc *DocumentoExterno) (int64, error) {
	return GuardarDocumentoExterno(doc)
}

func (ProjectMemoryRepository) ListExternalDocsByProjectID(proyectoID int64) ([]*DocumentoExterno, error) {
	return ListarDocumentosExternosProyecto(proyectoID)
}

func scanDecisionProyecto(s scanner) (*DecisionProyecto, error) {
	var item DecisionProyecto
	var propuestaID, tareaID sql.NullInt64
	if err := s.Scan(
		&item.ID, &item.ProyectoID, &item.Categoria, &item.Titulo, &item.Solucion,
		&item.Motivo, &item.Alternativas, &item.Impacto, &item.Estado,
		&propuestaID, &tareaID, &item.MetadataJSON, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if propuestaID.Valid {
		item.PropuestaID = &propuestaID.Int64
	}
	if tareaID.Valid {
		item.TareaID = &tareaID.Int64
	}
	return &item, nil
}

func scanDocumentoExterno(s scanner) (*DocumentoExterno, error) {
	var item DocumentoExterno
	var propuestaID, tareaID sql.NullInt64
	if err := s.Scan(
		&item.ID, &item.ProyectoID, &item.TipoDocumento, &item.Titulo, &item.RutaRef,
		&item.Resumen, &item.Estado, &item.Fuente, &propuestaID, &tareaID,
		&item.MetadataJSON, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if propuestaID.Valid {
		item.PropuestaID = &propuestaID.Int64
	}
	if tareaID.Valid {
		item.TareaID = &tareaID.Int64
	}
	return &item, nil
}

func decisionProyectoEstadoValido(estado string) bool {
	switch estado {
	case "vigente", "experimental", "reemplazada", "descartada", "archivada":
		return true
	default:
		return false
	}
}

func documentoExternoEstadoValido(estado string) bool {
	switch estado {
	case "vigente", "borrador", "archivado":
		return true
	default:
		return false
	}
}

func documentoExternoFuenteValida(fuente string) bool {
	switch fuente {
	case "manual", "propuesta", "tarea", "externo":
		return true
	default:
		return false
	}
}
