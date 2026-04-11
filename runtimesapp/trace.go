/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package runtimesapp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"orquesta/db"
)

const (
	defaultRuntimeTraceMaxBytes = 8192
	maxRuntimeTraceMaxBytes     = 262144
)

var ErrRuntimeTraceHandleNoEncontrado = errors.New("runtime handle no encontrado")

type RuntimeTraceRequest struct {
	HandleID int64
	Agente   string
	Proyecto string
	MaxBytes int
}

type RuntimeRawTrace struct {
	HandleID      int64  `json:"handle_id"`
	Agente        string `json:"agente"`
	Proyecto      string `json:"proyecto,omitempty"`
	TraceDir      string `json:"trace_dir,omitempty"`
	TraceManifest string `json:"trace_manifest,omitempty"`
	LogPath       string `json:"log_path,omitempty"`
	WorkingDir    string `json:"working_dir,omitempty"`
	Driver        string `json:"driver,omitempty"`
	Available     bool   `json:"available"`
	Reason        string `json:"reason,omitempty"`
	TotalBytes    int64  `json:"total_bytes"`
	BytesRead     int    `json:"bytes_read"`
	Truncated     bool   `json:"truncated"`
	RawTail       string `json:"raw_tail,omitempty"`
}

func (s *Service) ReadRuntimeTrace(req RuntimeTraceRequest) (*RuntimeRawTrace, error) {
	maxBytes := req.MaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultRuntimeTraceMaxBytes
	}
	if maxBytes > maxRuntimeTraceMaxBytes {
		maxBytes = maxRuntimeTraceMaxBytes
	}

	handle, proyectoSlug, err := s.resolveRuntimeTraceHandle(req)
	if err != nil {
		return nil, err
	}
	if handle == nil {
		return nil, ErrRuntimeTraceHandleNoEncontrado
	}

	meta := runtimeTraceMetadata(handle.MetadataJSON)
	trace := &RuntimeRawTrace{
		HandleID:      handle.ID,
		Agente:        strings.TrimSpace(handle.Agente),
		Proyecto:      strings.TrimSpace(proyectoSlug),
		TraceDir:      runtimeTraceString(meta, "trace_dir"),
		TraceManifest: runtimeTraceString(meta, "trace_manifest"),
		LogPath:       runtimeTraceString(meta, "log_path"),
		WorkingDir:    runtimeTraceString(meta, "working_dir"),
		Driver:        runtimeTraceString(meta, "driver"),
	}

	if trace.LogPath == "" {
		trace.Reason = "el handle no expone log_path"
		return trace, nil
	}

	rawTail, totalBytes, truncated, err := readRuntimeTraceTail(trace.LogPath, maxBytes)
	if err != nil {
		if os.IsNotExist(err) {
			trace.Reason = "el fichero de traza ya no existe en disco"
			return trace, nil
		}
		return nil, err
	}

	trace.Available = true
	trace.TotalBytes = totalBytes
	trace.BytesRead = len(rawTail)
	trace.Truncated = truncated
	trace.RawTail = rawTail
	if trace.BytesRead == 0 {
		trace.Reason = "la traza existe pero todavía no contiene salida"
	}
	return trace, nil
}

func (s *Service) resolveRuntimeTraceHandle(req RuntimeTraceRequest) (*db.RuntimeHandle, string, error) {
	if req.HandleID > 0 {
		handle, err := s.store.GetRuntimeHandle(req.HandleID)
		if err != nil {
			return nil, "", err
		}
		return handle, strings.TrimSpace(req.Proyecto), nil
	}

	agente := strings.TrimSpace(req.Agente)
	if agente == "" {
		return nil, "", fmt.Errorf("debes indicar handle_id o agente")
	}

	if proyecto := strings.TrimSpace(req.Proyecto); proyecto != "" {
		item, err := s.store.GetProject(proyecto)
		if err != nil {
			return nil, "", err
		}
		if item == nil {
			return nil, "", fmt.Errorf("proyecto no encontrado: %s", proyecto)
		}
		handle, err := s.store.GetOperationalRuntimeHandleForProject(agente, &item.ID)
		return handle, strings.TrimSpace(item.Slug), err
	}

	handle, err := s.store.GetOperationalRuntimeHandle(agente)
	return handle, "", err
}

func runtimeTraceMetadata(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}
	}
	out := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]any{}
	}
	return out
}

func runtimeTraceString(raw map[string]any, key string) string {
	if raw == nil {
		return ""
	}
	value, ok := raw[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func readRuntimeTraceTail(path string, maxBytes int) (string, int64, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, false, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", 0, false, err
	}
	totalBytes := info.Size()
	if totalBytes <= 0 {
		return "", totalBytes, false, nil
	}

	start := int64(0)
	truncated := false
	if totalBytes > int64(maxBytes) {
		start = totalBytes - int64(maxBytes)
		truncated = true
	}

	buf := make([]byte, totalBytes-start)
	n, err := file.ReadAt(buf, start)
	if err != nil && err != io.EOF {
		return "", totalBytes, truncated, err
	}
	buf = bytes.ToValidUTF8(buf[:n], []byte("?"))
	return string(buf), totalBytes, truncated, nil
}
