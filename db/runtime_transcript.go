package db

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

type RuntimeTranscriptEntry struct {
	ID             int64      `json:"id"`
	RuntimeID      int64      `json:"runtime_id"`
	HandleID       *int64     `json:"handle_id,omitempty"`
	Agente         string     `json:"agente"`
	ProyectoID     *int64     `json:"proyecto_id,omitempty"`
	ProyectoSlug   string     `json:"proyecto_slug,omitempty"`
	Stream         string     `json:"stream"`
	ByteOffset     *int64     `json:"byte_offset,omitempty"`
	Text           string     `json:"text"`
	NormalizedText string     `json:"normalized_text"`
	Classification string     `json:"classification"`
	HandlingNote   string     `json:"handling_note"`
	CreatedAt      time.Time  `json:"created_at"`
	HandledAt      *time.Time `json:"handled_at,omitempty"`
}

type transcriptRawLine struct {
	ByteOffset int64
	Raw        string
}

type FiltroRuntimeTranscript struct {
	Agente          *string
	ProyectoID      *int64
	RuntimeID       *int64
	HandleID        *int64
	Stream          *string
	Classification  *string
	Query           *string
	Desde           *time.Time
	SoloSenalesPend bool
	Limit           int
}

var ansiTranscriptRegexp = regexp.MustCompile(`\x1b\[[0-9;?<]*[ -/]*[@-~]`)
var oscTranscriptRegexp = regexp.MustCompile(`\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)`)

var runtimeTranscriptBatchBudgetOverride time.Duration
var ingestRuntimeTranscriptHandleFn = ingestarRuntimeTranscriptHandle

func PurgarRuntimeTranscriptRuidoHistorico() (int, error) {
	if DB == nil {
		return 0, nil
	}
	minutes := configIntOrDefault("runtime_transcript_noise_hygiene_min_age_minutes", 15)
	if minutes <= 0 {
		minutes = 15
	}
	limit := configIntOrDefault("runtime_transcript_noise_hygiene_batch_size", 100)
	if limit <= 0 {
		limit = 100
	}
	scanLimit := configIntOrDefault("runtime_transcript_noise_hygiene_scan_limit", limit*10)
	if scanLimit < limit {
		scanLimit = limit
	}
	cutoff := time.Now().UTC().Add(-time.Duration(minutes) * time.Minute)
	rows, err := DB.Query(`
		SELECT id, runtime_id, handle_id, agente, proyecto_id, '', stream, byte_offset, text,
		       normalized_text, classification, handling_note, created_at, handled_at
		FROM runtime_transcript
		WHERE stream = 'pty_out'
		  AND TRIM(COALESCE(classification, '')) = ''
		  AND created_at <= ?
		ORDER BY id ASC
		LIMIT ?`, cutoff, scanLimit)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	ids := make([]int64, 0, limit)
	for rows.Next() {
		item, err := scanRuntimeTranscript(rows)
		if err != nil {
			return 0, err
		}
		if item == nil {
			continue
		}
		if !descartarRuidoTranscript(item.Stream, item.Text, item.NormalizedText, item.Classification) {
			continue
		}
		if len(ids) < limit {
			ids = append(ids, item.ID)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	res, err := DB.Exec(`DELETE FROM runtime_transcript WHERE id IN (`+runtimeSQLPlaceholders(len(ids))+`)`, args...)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func PurgarRuntimeTranscriptRuidoHistoricoCompleto() (int, error) {
	if DB == nil {
		return 0, nil
	}
	minutes := configIntOrDefault("runtime_transcript_noise_hygiene_min_age_minutes", 15)
	if minutes <= 0 {
		minutes = 15
	}
	pageSize := configIntOrDefault("runtime_transcript_noise_hygiene_scan_limit", 1000)
	if pageSize <= 0 {
		pageSize = 1000
	}
	deleteLimit := configIntOrDefault("runtime_transcript_noise_hygiene_full_scan_delete_limit", 5000)
	if deleteLimit <= 0 {
		deleteLimit = 5000
	}
	cutoff := time.Now().UTC().Add(-time.Duration(minutes) * time.Minute)
	ids := make([]int64, 0, min(deleteLimit, pageSize))
	lastID := int64(0)

	for len(ids) < deleteLimit {
		rows, err := DB.Query(`
			SELECT id, runtime_id, handle_id, agente, proyecto_id, '', stream, byte_offset, text,
			       normalized_text, classification, handling_note, created_at, handled_at
			FROM runtime_transcript
			WHERE stream = 'pty_out'
			  AND TRIM(COALESCE(classification, '')) = ''
			  AND created_at <= ?
			  AND id > ?
			ORDER BY id ASC
			LIMIT ?`, cutoff, lastID, pageSize)
		if err != nil {
			return 0, err
		}

		seen := 0
		for rows.Next() {
			item, err := scanRuntimeTranscript(rows)
			if err != nil {
				rows.Close()
				return 0, err
			}
			if item == nil {
				continue
			}
			seen++
			lastID = item.ID
			if !descartarRuidoTranscript(item.Stream, item.Text, item.NormalizedText, item.Classification) {
				continue
			}
			if len(ids) < deleteLimit {
				ids = append(ids, item.ID)
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return 0, err
		}
		rows.Close()
		if seen < pageSize {
			break
		}
	}
	if len(ids) == 0 {
		return 0, nil
	}
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	res, err := DB.Exec(`DELETE FROM runtime_transcript WHERE id IN (`+runtimeSQLPlaceholders(len(ids))+`)`, args...)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func MarcarRuntimeTranscriptManejado(id int64, note string) error {
	if id <= 0 {
		return fmt.Errorf("id inválido")
	}
	_, err := DB.Exec(`
		UPDATE runtime_transcript
		SET handled_at = CURRENT_TIMESTAMP,
		    handling_note = ?
		WHERE id = ?`,
		strings.TrimSpace(note), id,
	)
	return err
}

func RegistrarRuntimeTranscriptInput(handle *RuntimeHandle, runtime *RuntimeInstance, texto string) error {
	texto = limpiarLineaTranscript(texto)
	if texto == "" {
		return nil
	}
	runtime, err := resolverRuntimeTranscript(handle, runtime)
	if err != nil || runtime == nil {
		return err
	}
	var handleID *int64
	var proyectoID *int64
	agente := strings.TrimSpace(runtime.Agente)
	if handle != nil {
		handleID = &handle.ID
		if agente == "" {
			agente = strings.TrimSpace(handle.Agente)
		}
		proyectoID = handle.ProyectoID
	}
	if proyectoID == nil {
		proyectoID = runtime.ProyectoID
	}
	_, err = RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
		RuntimeID:  runtime.ID,
		HandleID:   handleID,
		Agente:     agente,
		ProyectoID: proyectoID,
		Stream:     "stdin",
		Text:       texto,
	})
	return err
}

func RegistrarRuntimeTranscriptSistema(handle *RuntimeHandle, runtime *RuntimeInstance, texto string) error {
	texto = limpiarLineaTranscript(texto)
	if texto == "" {
		return nil
	}
	runtime, err := resolverRuntimeTranscript(handle, runtime)
	if err != nil || runtime == nil {
		return err
	}
	var handleID *int64
	var proyectoID *int64
	agente := strings.TrimSpace(runtime.Agente)
	if handle != nil {
		handleID = &handle.ID
		if agente == "" {
			agente = strings.TrimSpace(handle.Agente)
		}
		proyectoID = handle.ProyectoID
	}
	if proyectoID == nil {
		proyectoID = runtime.ProyectoID
	}
	_, err = RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
		RuntimeID:  runtime.ID,
		HandleID:   handleID,
		Agente:     agente,
		ProyectoID: proyectoID,
		Stream:     "system",
		Text:       texto,
	})
	return err
}

func IngestarRuntimeTranscriptHandle(handleID int64) (int, error) {
	handle, err := GetRuntimeHandle(handleID)
	if err != nil || handle == nil {
		return 0, err
	}
	return ingestarRuntimeTranscriptHandle(handle)
}

func IngestarRuntimeTranscriptActivos() (int, error) {
	handles, err := ListarRuntimeHandlesParaTranscript()
	if err != nil {
		return 0, err
	}
	start := time.Now().UTC()
	budget := runtimeTranscriptBatchBudget()
	total := 0
	now := time.Now().UTC()
	for _, handle := range handles {
		if handle == nil {
			continue
		}
		if runtimeTranscriptHotIdleSkip(handle, now) {
			continue
		}
		n, err := ingestRuntimeTranscriptHandleFn(handle)
		if err != nil {
			return total, err
		}
		total += n
		if budget > 0 && time.Since(start) >= budget {
			Audit("orquesta", "runtime_transcript_batch_deferred", "runtime_handle", 0,
				fmt.Sprintf("budget=%s total=%d last_handle=%d", budget, total, handle.ID))
			return total, nil
		}
	}
	return total, nil
}

func runtimeTranscriptBatchBudget() time.Duration {
	if runtimeTranscriptBatchBudgetOverride > 0 {
		return runtimeTranscriptBatchBudgetOverride
	}
	ms := configIntOrDefault("runtime_transcript_batch_budget_ms", 300)
	if ms <= 0 {
		ms = 300
	}
	return time.Duration(ms) * time.Millisecond
}
