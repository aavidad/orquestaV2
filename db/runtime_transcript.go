package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
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

type FiltroRuntimeTranscript struct {
	Agente          *string
	ProyectoID      *int64
	RuntimeID       *int64
	HandleID        *int64
	Stream          *string
	Classification  *string
	SoloSenalesPend bool
	Limit           int
}

var ansiTranscriptRegexp = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

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
	if strings.TrimSpace(entry.Classification) == "" && entry.Stream == "pty_out" {
		entry.Classification = clasificarTextoTranscript(entry.NormalizedText)
	}

	res, err := DB.Exec(`
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
	id, _ := res.LastInsertId()
	entry.ID = id
	return id, nil
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
		q += ` AND t.agente = ?`
		args = append(args, strings.TrimSpace(*filter.Agente))
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
	if filter.SoloSenalesPend {
		q += ` AND t.classification <> '' AND t.handled_at IS NULL`
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
	handles, err := ListarRuntimeHandles(nil)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, handle := range handles {
		if handle == nil {
			continue
		}
		switch strings.TrimSpace(handle.Estado) {
		case "activo", "pausado", "fallido":
		default:
			continue
		}
		n, err := ingestarRuntimeTranscriptHandle(handle)
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
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
	return &item, nil
}

func ingestarRuntimeTranscriptHandle(handle *RuntimeHandle) (int, error) {
	if handle == nil {
		return 0, nil
	}
	meta := mapFromJSON(handle.MetadataJSON)
	logPath := strings.TrimSpace(stringFromMap(meta, "log_path", ""))
	if logPath == "" {
		return 0, nil
	}
	info, err := os.Stat(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	offset := int64FromMap(meta, "transcript_log_offset")
	pending := stringFromMap(meta, "transcript_log_pending", "")
	if offset < 0 || offset > info.Size() {
		offset = 0
		pending = ""
	}
	readMax := configIntOrDefault("runtime_transcript_ingest_max_bytes", 65536)
	if readMax <= 0 {
		readMax = 65536
	}
	remaining := info.Size() - offset
	if remaining <= 0 {
		return 0, nil
	}
	if remaining > int64(readMax) {
		remaining = int64(readMax)
	}
	file, err := os.Open(logPath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	buf := make([]byte, remaining)
	n, err := file.ReadAt(buf, offset)
	if err != nil && err != io.EOF {
		return 0, err
	}
	if n == 0 {
		return 0, nil
	}
	runtime, err := resolverRuntimeTranscript(handle, nil)
	if err != nil || runtime == nil {
		return 0, err
	}

	chunk := string(buf[:n])
	offset += int64(n)
	pending, lines := extraerLineasTranscript(pending + chunk)
	if len(pending) > 4096 {
		lines = append(lines, pending)
		pending = ""
	}

	total := 0
	for _, line := range lines {
		texto := limpiarLineaTranscript(line)
		if texto == "" {
			continue
		}
		stream := "pty_out"
		if esLineaSistemaTranscript(texto) {
			stream = "system"
		}
		normalized := normalizarTextoTranscript(texto)
		classification := clasificarTextoTranscript(normalized)
		if descartarRuidoTranscript(stream, texto, normalized, classification) {
			continue
		}
		var byteOffset *int64
		id, err := RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
			RuntimeID:      runtime.ID,
			HandleID:       &handle.ID,
			Agente:         strings.TrimSpace(handle.Agente),
			ProyectoID:     runtime.ProyectoID,
			Stream:         stream,
			ByteOffset:     byteOffset,
			Text:           texto,
			NormalizedText: normalized,
			Classification: classification,
		})
		if err != nil {
			return total, err
		}
		total++
		if err := registrarEventoDerivadoTranscript(id, handle, runtime, texto); err != nil {
			return total, err
		}
	}

	meta["transcript_log_offset"] = offset
	if pending != "" {
		meta["transcript_log_pending"] = pending
	} else {
		delete(meta, "transcript_log_pending")
	}
	meta["transcript_last_ingested_at"] = time.Now().UTC().Format(time.RFC3339)
	metaJSON, _ := json.Marshal(meta)
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json = ? WHERE id = ?`, string(metaJSON), handle.ID); err != nil {
		return total, err
	}
	return total, nil
}

func registrarEventoDerivadoTranscript(transcriptID int64, handle *RuntimeHandle, runtime *RuntimeInstance, texto string) error {
	if handle == nil || runtime == nil {
		return nil
	}
	classification := clasificarTextoTranscript(normalizarTextoTranscript(texto))
	if classification == "" {
		return nil
	}
	level := "warn"
	if classification == "runtime_panic" || classification == "runtime_crash" {
		level = "critical"
	}
	payload, _ := json.Marshal(map[string]any{
		"transcript_id":  transcriptID,
		"handle_id":      handle.ID,
		"agente":         strings.TrimSpace(handle.Agente),
		"classification": classification,
		"text":           texto,
	})
	_, err := RegistrarRuntimeEvent(&RuntimeEvent{
		RuntimeID:   runtime.ID,
		Kind:        classification,
		Level:       level,
		Message:     texto,
		PayloadJSON: string(payload),
	})
	return err
}

func resolverRuntimeTranscript(handle *RuntimeHandle, runtime *RuntimeInstance) (*RuntimeInstance, error) {
	if runtime != nil && runtime.ID > 0 {
		return runtime, nil
	}
	if handle == nil {
		return nil, nil
	}
	if handle.RuntimeID != nil && *handle.RuntimeID > 0 {
		return GetRuntime(*handle.RuntimeID)
	}
	if handle.SesionID != nil && *handle.SesionID > 0 {
		return GetRuntimeBySesionID(*handle.SesionID)
	}
	return nil, nil
}

func extraerLineasTranscript(raw string) (string, []string) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	if raw == "" {
		return "", nil
	}
	parts := strings.Split(raw, "\n")
	if strings.HasSuffix(raw, "\n") {
		return "", parts[:len(parts)-1]
	}
	if len(parts) == 1 {
		return parts[0], nil
	}
	return parts[len(parts)-1], parts[:len(parts)-1]
}

func limpiarLineaTranscript(raw string) string {
	raw = ansiTranscriptRegexp.ReplaceAllString(raw, "")
	raw = strings.ReplaceAll(raw, "\x00", "")
	raw = strings.TrimSpace(raw)
	return raw
}

func normalizarTextoTranscript(raw string) string {
	raw = strings.ToLower(limpiarLineaTranscript(raw))
	if raw == "" {
		return ""
	}
	return strings.Join(strings.Fields(raw), " ")
}

func descartarRuidoTranscript(stream, texto, normalized, classification string) bool {
	if strings.TrimSpace(stream) != "pty_out" {
		return false
	}
	if strings.TrimSpace(classification) != "" {
		return false
	}
	texto = strings.TrimSpace(texto)
	normalized = strings.TrimSpace(normalized)
	if normalized == "" {
		return true
	}
	alnum := contarRunasSemanticas(normalized)
	if alnum == 0 {
		return true
	}
	if alnum < 3 && len(strings.Fields(normalized)) <= 1 {
		return true
	}
	total := contarRunasVisibles(texto)
	if total == 0 {
		return true
	}
	if float64(alnum)/float64(total) < 0.25 {
		return true
	}
	return false
}

func contarRunasSemanticas(raw string) int {
	count := 0
	for _, r := range raw {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			count++
		}
	}
	return count
}

func contarRunasVisibles(raw string) int {
	count := 0
	for _, r := range raw {
		if !unicode.IsSpace(r) {
			count++
		}
	}
	return count
}

func clasificarTextoTranscript(normalized string) string {
	if normalized == "" {
		return ""
	}
	panicPhrases := []string{
		"panic:",
		"fatal error:",
		"the application panicked",
		"thread 'main' panicked",
		"stack backtrace:",
		"segmentation fault",
		"sigsegv",
		"traceback (most recent call last):",
		"uncaught exception",
	}
	for _, phrase := range panicPhrases {
		if strings.Contains(normalized, phrase) {
			return "runtime_panic"
		}
	}
	crashPhrases := []string{
		"aborted",
		"core dumped",
		"broken pipe",
		"unexpected eof",
		"connection reset by peer",
		"process exited",
	}
	for _, phrase := range crashPhrases {
		if strings.Contains(normalized, phrase) {
			return "runtime_crash"
		}
	}
	if strings.Contains(normalized, "goroutine ") && strings.Contains(normalized, "[") {
		return "runtime_panic"
	}
	reviewReadyPhrases := []string{
		"listo para review",
		"lista para review",
		"listo para revisión",
		"lista para revisión",
		"ready for review",
		"ready to review",
		"puedes revisar",
		"ya puede revisarse",
		"he terminado y está listo para review",
		"he terminado y esta listo para review",
	}
	for _, phrase := range reviewReadyPhrases {
		if strings.Contains(normalized, phrase) {
			return "ready_for_review"
		}
	}
	replanPhrases := []string{
		"qué hago ahora",
		"que hago ahora",
		"cuál es el siguiente paso",
		"cual es el siguiente paso",
		"no tengo siguiente paso",
		"no tengo claro el siguiente paso",
		"no veo el siguiente frente",
		"no encuentro el siguiente frente",
		"what should i do next",
		"what next",
	}
	for _, phrase := range replanPhrases {
		if strings.Contains(normalized, phrase) {
			return "needs_replan"
		}
	}
	approvalPhrases := []string{
		"si me dejas",
		"me dejas",
		"si me permites",
		"me permites",
		"quieres que",
		"te parece bien si",
		"puedo hacerlo",
		"puedo seguir",
		"me autorizas",
		"do you want me to",
	}
	for _, phrase := range approvalPhrases {
		if strings.Contains(normalized, phrase) {
			return "approval_request"
		}
	}
	if parecePreguntaAprobacion(normalized) {
		return "approval_request"
	}
	waitingPhrases := []string{
		"espero tu respuesta",
		"quedo a la espera",
		"esperando tu confirmacion",
		"esperando tu confirmación",
		"esperando aprobacion",
		"esperando aprobación",
		"awaiting approval",
		"waiting for approval",
		"waiting for your answer",
	}
	for _, phrase := range waitingPhrases {
		if strings.Contains(normalized, phrase) {
			return "waiting_human"
		}
	}
	blockedPhrases := []string{
		"bloqueado",
		"no puedo continuar sin",
		"no puedo seguir sin",
		"cannot continue without",
		"can't continue without",
		"blocked on",
		"pendiente de credenciales",
		"faltan credenciales",
		"falta acceso",
	}
	for _, phrase := range blockedPhrases {
		if strings.Contains(normalized, phrase) {
			return "blocked"
		}
	}
	return ""
}

func parecePreguntaAprobacion(normalized string) bool {
	if normalized == "" {
		return false
	}
	if !strings.ContainsAny(normalized, "?¿") {
		return false
	}
	questionPhrases := []string{
		"puedo ",
		"can i ",
		"may i ",
		"should i",
		"quieres que",
		"te parece bien si",
	}
	for _, phrase := range questionPhrases {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}
	return false
}

func esLineaSistemaTranscript(texto string) bool {
	return strings.HasPrefix(texto, "Script started on ") || strings.HasPrefix(texto, "Script done on ")
}

func int64FromMap(m map[string]any, key string) int64 {
	if m == nil {
		return 0
	}
	switch v := m[key].(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return 0
		}
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return 0
}
