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
	limit := configIntOrDefault("project_memory_vote_history_limit", 50)
	if limit <= 0 {
		limit = 50
	}
	rows, err := DB.Query(`
		SELECT id, codigo, titulo, tipo, estado, propuesto_por, created_at, cerrada_at
		FROM propuestas
		WHERE proyecto_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT ?`, proyectoID, limit)
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

	if len(list) == 0 {
		return list, nil
	}

	placeholders := make([]string, 0, len(list))
	args := make([]any, 0, len(list))
	indexByProposalID := make(map[int64]*HistorialVotacionProyecto, len(list))
	for _, item := range list {
		if item == nil {
			continue
		}
		placeholders = append(placeholders, "?")
		args = append(args, item.PropuestaID)
		indexByProposalID[item.PropuestaID] = item
	}

	voteRows, err := DB.Query(`
		SELECT propuesta_id, agente, posicion, comentario
		FROM votos
		WHERE propuesta_id IN (`+strings.Join(placeholders, ",")+`)
		ORDER BY propuesta_id, id`, args...)
	if err != nil {
		return nil, err
	}
	defer voteRows.Close()

	for voteRows.Next() {
		var (
			propuestaID int64
			voto        Voto
		)
		if err := voteRows.Scan(&propuestaID, &voto.Agente, &voto.Posicion, &voto.Comentario); err != nil {
			return nil, err
		}
		item := indexByProposalID[propuestaID]
		if item == nil {
			continue
		}
		switch voto.Posicion {
		case VotoAcuerdo:
			item.Acuerdo++
		case VotoDesacuerdo:
			item.Desacuerdo++
		case VotoAbstencion:
			item.Abstencion++
		case VotoPendiente:
			item.Pendiente++
		default:
			item.Pendiente++
		}
		vCopy := voto
		item.Votos = append(item.Votos, &vCopy)
	}
	if err := voteRows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func GuardarDecisionProyecto(d *DecisionProyecto) (int64, error) {
	if d == nil {
		return 0, fmt.Errorf("decision obligatoria")
	}
	if legacy, err := decisionProjectLegacySchema(); err != nil {
		return 0, err
	} else if legacy {
		return guardarDecisionProyectoLegacy(d)
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
	if legacy, err := decisionProjectLegacySchema(); err != nil {
		return nil, err
	} else if legacy {
		return listarDecisionesProyectoLegacy(proyectoID)
	}
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

func decisionProjectLegacySchema() (bool, error) {
	hasProjectID, err := ColumnExists("decisiones_proyecto", "proyecto_id")
	if err != nil {
		return false, err
	}
	return !hasProjectID, nil
}

func guardarDecisionProyectoLegacy(d *DecisionProyecto) (int64, error) {
	d.Categoria = strings.TrimSpace(d.Categoria)
	d.Titulo = strings.TrimSpace(d.Titulo)
	d.Solucion = strings.TrimSpace(d.Solucion)
	d.Motivo = strings.TrimSpace(d.Motivo)
	d.Alternativas = strings.TrimSpace(d.Alternativas)
	d.Impacto = strings.TrimSpace(d.Impacto)
	if d.MetadataJSON == "" {
		d.MetadataJSON = "{}"
	}
	if d.ProyectoID == 0 || d.Titulo == "" || d.Solucion == "" {
		return 0, fmt.Errorf("proyecto, titulo y solucion son obligatorios")
	}

	proyectoSlug, err := proyectoSlugDesdeID(d.ProyectoID)
	if err != nil {
		return 0, err
	}
	propuestaCodigo, err := propuestaCodigoDesdeID(d.PropuestaID)
	if err != nil {
		return 0, err
	}

	var existenteID int64
	err = DB.QueryRow(`SELECT id FROM decisiones_proyecto WHERE proyecto = ? AND titulo = ?`, proyectoSlug, d.Titulo).Scan(&existenteID)
	switch err {
	case nil:
		_, err = DB.Exec(`
			UPDATE decisiones_proyecto
			   SET solucion_elegida = ?, motivo = ?, alternativas_descartadas = ?, impacto = ?,
			       propuesta_codigo = ?, tarea_id = ?, registrado_por = ?
			 WHERE id = ?`,
			d.Solucion,
			d.Motivo,
			d.Alternativas,
			legacyDecisionImpact(d.Impacto),
			propuestaCodigo,
			d.TareaID,
			"orquesta",
			existenteID,
		)
		if err != nil {
			return 0, err
		}
		return existenteID, nil
	case sql.ErrNoRows:
		res, err := DB.Exec(`
			INSERT INTO decisiones_proyecto (
				proyecto, titulo, solucion_elegida, motivo, alternativas_descartadas, impacto,
				propuesta_codigo, tarea_id, registrado_por
			) VALUES (?,?,?,?,?,?,?,?,?)`,
			proyectoSlug,
			d.Titulo,
			d.Solucion,
			d.Motivo,
			d.Alternativas,
			legacyDecisionImpact(d.Impacto),
			propuestaCodigo,
			d.TareaID,
			"orquesta",
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

func listarDecisionesProyectoLegacy(proyectoID int64) ([]*DecisionProyecto, error) {
	proyectoSlug, err := proyectoSlugDesdeID(proyectoID)
	if err != nil {
		return nil, err
	}
	rows, err := DB.Query(`
		SELECT id, proyecto, titulo, solucion_elegida, motivo, alternativas_descartadas,
		       impacto, propuesta_codigo, tarea_id, registrado_por, created_at, updated_at
		  FROM decisiones_proyecto
		 WHERE proyecto = ?
		 ORDER BY titulo`, proyectoSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type legacyDecisionRow struct {
		item         *DecisionProyecto
		propuestaCod string
	}

	var rowsScanned []legacyDecisionRow
	for rows.Next() {
		var (
			item           DecisionProyecto
			proyectoLegacy string
			propuestaCod   string
			registradoPor  string
			tareaID        sql.NullInt64
		)
		if err := rows.Scan(
			&item.ID,
			&proyectoLegacy,
			&item.Titulo,
			&item.Solucion,
			&item.Motivo,
			&item.Alternativas,
			&item.Impacto,
			&propuestaCod,
			&tareaID,
			&registradoPor,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.ProyectoID = proyectoID
		item.Categoria = "general"
		item.Estado = "vigente"
		item.MetadataJSON = "{}"
		if tareaID.Valid {
			item.TareaID = &tareaID.Int64
		}
		rowsScanned = append(rowsScanned, legacyDecisionRow{
			item:         &item,
			propuestaCod: strings.TrimSpace(propuestaCod),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	list := make([]*DecisionProyecto, 0, len(rowsScanned))
	for _, row := range rowsScanned {
		if row.item == nil {
			continue
		}
		if row.propuestaCod != "" {
			if propuesta, err := GetPropuesta(row.propuestaCod); err == nil && propuesta != nil {
				row.item.PropuestaID = &propuesta.ID
			}
		}
		list = append(list, row.item)
	}
	return list, nil
}

func proyectoSlugDesdeID(proyectoID int64) (string, error) {
	proyecto, err := GetProyecto(fmt.Sprintf("%d", proyectoID))
	if err != nil {
		return "", err
	}
	if proyecto == nil || strings.TrimSpace(proyecto.Slug) == "" {
		return "", fmt.Errorf("proyecto %d no encontrado", proyectoID)
	}
	return strings.TrimSpace(proyecto.Slug), nil
}

func propuestaCodigoDesdeID(propuestaID *int64) (string, error) {
	if propuestaID == nil || *propuestaID <= 0 {
		return "", nil
	}
	propuesta, err := GetPropuesta(fmt.Sprintf("%d", *propuestaID))
	if err != nil {
		return "", err
	}
	if propuesta == nil {
		return "", nil
	}
	return strings.TrimSpace(propuesta.Codigo), nil
}

func legacyDecisionImpact(impacto string) string {
	impacto = strings.ToLower(strings.TrimSpace(impacto))
	switch {
	case strings.Contains(impacto, "alto"):
		return "alto"
	case strings.Contains(impacto, "bajo"):
		return "bajo"
	default:
		return "medio"
	}
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
