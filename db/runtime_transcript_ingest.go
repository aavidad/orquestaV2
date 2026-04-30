package db

import (
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"orquesta/runtimepolicy"
)

func ingestarRuntimeTranscriptHandle(handle *RuntimeHandle) (int, error) {
	if handle == nil {
		return 0, nil
	}
	now := time.Now().UTC()
	meta := mapFromJSON(handle.MetadataJSON)
	logPath := runtimeTranscriptResolveLogPath(handle, meta)
	if logPath == "" {
		runtimeTranscriptHotIdleClear(handle)
		return 0, nil
	}
	info, err := os.Stat(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			meta["log_path"] = ""
			logPath = runtimeTranscriptResolveLogPath(handle, meta)
			if logPath == "" {
				runtimeTranscriptHotIdleRemember(handle, now)
				return 0, nil
			}
			info, err = os.Stat(logPath)
			if err != nil {
				if os.IsNotExist(err) {
					runtimeTranscriptHotIdleRemember(handle, now)
					return 0, nil
				}
				return 0, err
			}
		} else {
			return 0, err
		}
	}
	offset := int64FromMap(meta, "transcript_log_offset")
	pending := stringFromMap(meta, "transcript_log_pending", "")
	if offset < 0 || offset > info.Size() {
		offset = 0
		pending = ""
	}
	readMax := configIntOrDefault("runtime_transcript_ingest_max_bytes", 16384)
	if readMax <= 0 {
		readMax = 16384
	}
	remaining := info.Size() - offset
	if remaining <= 0 {
		if pending == "" {
			runtimeTranscriptHotIdleRemember(handle, now)
		} else {
			runtimeTranscriptHotIdleClear(handle)
		}
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
		runtimeTranscriptHotIdleRemember(handle, now)
		return 0, nil
	}
	runtime, err := resolverRuntimeTranscript(handle, nil)
	if err != nil || runtime == nil {
		return 0, err
	}

	chunk := string(buf[:n])
	startOffset := offset - int64(len(pending))
	offset += int64(n)
	pending, lines := extraerLineasTranscriptConOffsets(pending+chunk, startOffset)
	if len(pending) > 4096 {
		lines = append(lines, transcriptRawLine{
			ByteOffset: startOffset + int64(len(pending+chunk)-len(pending)),
			Raw:        pending,
		})
		pending = ""
	}
	pending = runtimepolicy.CompactPendingTranscript(pending)

	total := 0
	sawRuntimeOutput := false
	for _, line := range lines {
		texto := limpiarLineaTranscript(line.Raw)
		if texto == "" {
			continue
		}
		stream := "pty_out"
		if esLineaSistemaTranscript(texto) {
			stream = "system"
		}
		normalized := normalizarTextoTranscript(texto)
		classification := ""
		if stream != "system" {
			classification = clasificarTextoTranscript(normalized)
		}
		if descartarRuidoTranscript(stream, texto, normalized, classification) {
			continue
		}
		byteOffset := line.ByteOffset
		id, err := RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
			RuntimeID:      runtime.ID,
			HandleID:       &handle.ID,
			Agente:         strings.TrimSpace(handle.Agente),
			ProyectoID:     runtime.ProyectoID,
			Stream:         stream,
			ByteOffset:     &byteOffset,
			Text:           texto,
			NormalizedText: normalized,
			Classification: classification,
		})
		if err != nil {
			return total, err
		}
		total++
		if stream == "pty_out" {
			sawRuntimeOutput = true
		}
		if err := registrarEventoDerivadoTranscript(id, handle, runtime, texto, classification); err != nil {
			return total, err
		}
	}
	if sawRuntimeOutput {
		if err := AckBootstrapRuntimeLeaseByEvidence(handle, runtime, "runtime_transcript"); err != nil {
			return total, err
		}
	}
	if total > 0 || pending != "" {
		runtimeTranscriptHotIdleClear(handle)
	} else {
		runtimeTranscriptHotIdleRemember(handle, now)
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
	runtimeHandleHotReset()
	return total, nil
}

func registrarEventoDerivadoTranscript(transcriptID int64, handle *RuntimeHandle, runtime *RuntimeInstance, texto, classification string) error {
	if handle == nil || runtime == nil {
		return nil
	}
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
		r, err := GetRuntime(*handle.RuntimeID)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return r, err
	}
	if handle.SesionID != nil && *handle.SesionID > 0 {
		r, err := GetRuntimeBySesionID(*handle.SesionID)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return r, err
	}
	return nil, nil
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
