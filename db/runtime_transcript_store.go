package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func RegistrarRuntimeTranscript(entry *RuntimeTranscriptEntry) (int64, error) {
	if entry == nil {
		return 0, fmt.Errorf("runtime transcript nulo")
	}
	if entry.RuntimeID <= 0 {
		return 0, fmt.Errorf("runtime_id es obligatorio")
	}
	entry.Stream = strings.TrimSpace(entry.Stream)
	if entry.Stream == "" {
		entry.Stream = "system"
	}
	if strings.TrimSpace(entry.Text) == "" {
		return 0, fmt.Errorf("text es obligatorio")
	}
	if strings.TrimSpace(entry.NormalizedText) == "" {
		entry.NormalizedText = normalizarTextoTranscript(entry.Text)
	}
	if strings.TrimSpace(entry.Classification) == "" && runtimeTranscriptShouldAutoClassifyStream(entry.Stream) {
		entry.Classification = clasificarTextoTranscript(entry.NormalizedText)
	}

	id, err := insertReturningID(`
		INSERT INTO runtime_transcript (
			runtime_id, handle_id, agente, proyecto_id, stream, byte_offset,
			text, normalized_text, classification, handling_note, handled_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		entry.RuntimeID, entry.HandleID, strings.TrimSpace(entry.Agente), entry.ProyectoID,
		entry.Stream, entry.ByteOffset, entry.Text, entry.NormalizedText,
		entry.Classification, strings.TrimSpace(entry.HandlingNote), entry.HandledAt,
	)
	if err != nil {
		return 0, err
	}
	entry.ID = id
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	if err := persistRuntimeHandleSemanticProgress(entry); err != nil {
		return 0, err
	}
	return id, nil
}

func persistRuntimeHandleSemanticProgress(entry *RuntimeTranscriptEntry) error {
	if entry == nil || entry.HandleID == nil || *entry.HandleID <= 0 {
		return nil
	}
	if !runtimeTranscriptClassificationCarriesSemanticProgress(entry.Classification) {
		return nil
	}
	handle, err := GetRuntimeHandle(*entry.HandleID)
	if err != nil || handle == nil {
		return err
	}
	meta := mapFromJSON(handle.MetadataJSON)
	currentAt := strings.TrimSpace(stringFromMap(meta, "semantic_progress_at", ""))
	currentID := int64FromMap(meta, "semantic_progress_transcript_id")
	if currentAt != "" {
		if ts, err := time.Parse(time.RFC3339Nano, currentAt); err == nil {
			switch {
			case ts.After(entry.CreatedAt.UTC()):
				return nil
			case ts.Equal(entry.CreatedAt.UTC()) && currentID >= entry.ID:
				return nil
			}
		}
	}
	meta["semantic_progress_transcript_id"] = entry.ID
	meta["semantic_progress_at"] = entry.CreatedAt.UTC().Format(time.RFC3339Nano)
	meta["semantic_progress_classification"] = strings.TrimSpace(entry.Classification)
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return ActualizarMetadataRuntimeHandle(handle.ID, string(metaJSON))
}

func ListarRuntimeTranscript(filter FiltroRuntimeTranscript) ([]*RuntimeTranscriptEntry, error) {
	q := `
		SELECT t.id, t.runtime_id, t.handle_id, t.agente, t.proyecto_id,
		       COALESCE(p.slug, ''), t.stream, t.byte_offset, t.text,
		       t.normalized_text, t.classification, t.handling_note,
		       t.created_at, t.handled_at
		FROM runtime_transcript t
		LEFT JOIN proyectos p ON p.id = t.proyecto_id
		WHERE 1=1`
	args := []any{}
	if filter.Agente != nil {
		agenteCanonico, err := CanonicalizeAgentName(*filter.Agente)
		if err != nil {
			return nil, err
		}
		q += ` AND t.agente = ?`
		args = append(args, agenteCanonico)
	}
	if filter.ProyectoID != nil {
		q += ` AND t.proyecto_id = ?`
		args = append(args, *filter.ProyectoID)
	}
	if filter.RuntimeID != nil {
		q += ` AND t.runtime_id = ?`
		args = append(args, *filter.RuntimeID)
	}
	if filter.HandleID != nil {
		q += ` AND t.handle_id = ?`
		args = append(args, *filter.HandleID)
	}
	if filter.Stream != nil {
		q += ` AND t.stream = ?`
		args = append(args, strings.TrimSpace(*filter.Stream))
	}
	if filter.Classification != nil {
		q += ` AND t.classification = ?`
		args = append(args, strings.TrimSpace(*filter.Classification))
	}
	if filter.Query != nil {
		raw := strings.TrimSpace(*filter.Query)
		if raw != "" {
			normalized := normalizarTextoTranscript(raw)
			if normalized == "" {
				normalized = strings.ToLower(raw)
			}
			tokens := strings.Fields(normalized)
			if len(tokens) == 0 {
				tokens = []string{normalized}
			}
			for _, token := range tokens {
				if strings.TrimSpace(token) == "" {
					continue
				}
				q += ` AND (t.normalized_text LIKE ? OR t.text LIKE ? OR t.classification LIKE ?)`
				args = append(args, "%"+token+"%", "%"+token+"%", "%"+token+"%")
			}
		}
	}
	if filter.Desde != nil {
		q += ` AND t.created_at >= ?`
		args = append(args, filter.Desde.UTC())
	}
	if filter.SoloSenalesPend {
		q += ` AND t.classification <> '' AND t.handled_at IS NULL`
		q += ` AND lower(t.classification) NOT IN ('tool_execution', 'tool_exploration', 'bootstrap_guidance', 'progress_update', 'tool_result_ok', 'patch_or_code_evidence', 'ui_noise')`
	}
	q += ` ORDER BY t.id DESC`
	if filter.Limit <= 0 {
		filter.Limit = 100
	}
	q += ` LIMIT ?`
	args = append(args, filter.Limit)

	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*RuntimeTranscriptEntry
	for rows.Next() {
		item, err := scanRuntimeTranscript(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func scanRuntimeTranscript(scanner interface{ Scan(dest ...any) error }) (*RuntimeTranscriptEntry, error) {
	var (
		item         RuntimeTranscriptEntry
		handleID     sql.NullInt64
		proyectoID   sql.NullInt64
		byteOffset   sql.NullInt64
		handledAt    sql.NullTime
		proyectoSlug sql.NullString
	)
	if err := scanner.Scan(
		&item.ID,
		&item.RuntimeID,
		&handleID,
		&item.Agente,
		&proyectoID,
		&proyectoSlug,
		&item.Stream,
		&byteOffset,
		&item.Text,
		&item.NormalizedText,
		&item.Classification,
		&item.HandlingNote,
		&item.CreatedAt,
		&handledAt,
	); err != nil {
		return nil, err
	}
	if handleID.Valid {
		value := handleID.Int64
		item.HandleID = &value
	}
	if proyectoID.Valid {
		value := proyectoID.Int64
		item.ProyectoID = &value
	}
	if proyectoSlug.Valid {
		item.ProyectoSlug = strings.TrimSpace(proyectoSlug.String)
	}
	if byteOffset.Valid {
		value := byteOffset.Int64
		item.ByteOffset = &value
	}
	if handledAt.Valid {
		value := handledAt.Time
		item.HandledAt = &value
	}
	if strings.TrimSpace(item.Classification) == "" && runtimeTranscriptShouldAutoClassifyStream(item.Stream) {
		if strings.TrimSpace(item.NormalizedText) == "" {
			item.NormalizedText = normalizarTextoTranscript(item.Text)
		}
		item.Classification = clasificarTextoTranscript(item.NormalizedText)
	}
	return &item, nil
}

func runtimeTranscriptShouldAutoClassifyStream(stream string) bool {
	switch strings.ToLower(strings.TrimSpace(stream)) {
	case "pty_out", "assistant", "stdout", "stderr":
		return true
	default:
		return false
	}
}
