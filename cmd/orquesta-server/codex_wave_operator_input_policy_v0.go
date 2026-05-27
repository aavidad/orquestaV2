package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const codexWaveOperatorInputMaxBytesV0 int64 = 256 * 1024

type codexWaveOperatorInputReceiptV0 struct {
	Kind           string `json:"kind"`
	Origin         string `json:"origin"`
	SourceCategory string `json:"source_category"`
	SourceRef      string `json:"source_ref,omitempty"`
	ContentRef     string `json:"content_ref,omitempty"`
	Bytes          int64  `json:"bytes,omitempty"`
}

type codexWaveOperatorInputErrorV0 struct {
	code string
}

func (err codexWaveOperatorInputErrorV0) Error() string {
	return err.code
}

func codexWaveOperatorTextFromFileV0(kind string, rawPath string) (string, codexWaveOperatorInputReceiptV0, error) {
	path := strings.TrimSpace(rawPath)
	if err := codexWaveValidateOperatorInputPathV0(path); err != nil {
		return "", codexWaveOperatorInputReceiptV0{}, codexWaveOperatorInputErrorV0{code: codexWaveOperatorInputErrorCodeV0(kind, "invalid")}
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", codexWaveOperatorInputReceiptV0{}, codexWaveOperatorInputErrorV0{code: codexWaveOperatorInputErrorCodeV0(kind, "unreadable")}
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", codexWaveOperatorInputReceiptV0{}, codexWaveOperatorInputErrorV0{code: codexWaveOperatorInputErrorCodeV0(kind, "invalid")}
	}
	if info.Size() > codexWaveOperatorInputMaxBytesV0 {
		return "", codexWaveOperatorInputReceiptV0{}, codexWaveOperatorInputErrorV0{code: codexWaveOperatorInputErrorCodeV0(kind, "too_large")}
	}
	file, err := os.Open(path)
	if err != nil {
		return "", codexWaveOperatorInputReceiptV0{}, codexWaveOperatorInputErrorV0{code: codexWaveOperatorInputErrorCodeV0(kind, "unreadable")}
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, codexWaveOperatorInputMaxBytesV0+1))
	if err != nil {
		return "", codexWaveOperatorInputReceiptV0{}, codexWaveOperatorInputErrorV0{code: codexWaveOperatorInputErrorCodeV0(kind, "unreadable")}
	}
	if int64(len(data)) > codexWaveOperatorInputMaxBytesV0 {
		return "", codexWaveOperatorInputReceiptV0{}, codexWaveOperatorInputErrorV0{code: codexWaveOperatorInputErrorCodeV0(kind, "too_large")}
	}
	if !utf8.Valid(data) || strings.ContainsRune(string(data), '\x00') {
		return "", codexWaveOperatorInputReceiptV0{}, codexWaveOperatorInputErrorV0{code: codexWaveOperatorInputErrorCodeV0(kind, "invalid")}
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return "", codexWaveOperatorInputReceiptV0{}, codexWaveOperatorInputErrorV0{code: codexWaveOperatorInputErrorCodeV0(kind, "empty")}
	}
	return text, codexWaveOperatorInputReceiptFromFileV0(kind, path, data), nil
}

func codexWaveOperatorInputReceiptFromFileV0(kind string, path string, data []byte) codexWaveOperatorInputReceiptV0 {
	return codexWaveOperatorInputReceiptV0{
		Kind:           kind,
		Origin:         "operator_local_file",
		SourceCategory: codexWaveOperatorInputSourceCategoryV0(path),
		SourceRef:      codexWaveOperatorInputOpaqueRefV0("operator-file", path),
		ContentRef:     codexWaveOperatorInputContentRefV0(data),
		Bytes:          int64(len(data)),
	}
}

func codexWaveOperatorInputReceiptFromTextV0(kind string, origin string, text string) codexWaveOperatorInputReceiptV0 {
	data := []byte(strings.TrimSpace(text))
	return codexWaveOperatorInputReceiptV0{
		Kind:           kind,
		Origin:         origin,
		SourceCategory: "operator_text",
		SourceRef:      codexWaveOperatorInputOpaqueRefV0(origin, string(data)),
		ContentRef:     codexWaveOperatorInputContentRefV0(data),
		Bytes:          int64(len(data)),
	}
}

func codexWaveValidateOperatorInputPathV0(path string) error {
	if path == "" {
		return errors.New("empty_path")
	}
	clean := filepath.Clean(path)
	if strings.Contains(clean, "\x00") || strings.Contains(clean, ".."+string(filepath.Separator)) {
		return errors.New("unsafe_path")
	}
	if codexWaveLooksLikeOpaqueRefV0(clean) || codexWaveOperatorInputIsControlPathV0(clean) {
		return errors.New("unsupported_source")
	}
	return nil
}

func codexWaveLooksLikeOpaqueRefV0(value string) bool {
	if strings.HasPrefix(value, "ref:") || strings.HasPrefix(value, "ref://") {
		return true
	}
	return !strings.ContainsAny(value, `/\`) && strings.Contains(value, "-ref-")
}

func codexWaveOperatorInputIsControlPathV0(path string) bool {
	parts := strings.Split(filepath.ToSlash(filepath.Clean(path)), "/")
	for _, part := range parts {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case ".orquesta-runtime", ".orquesta-codex-runtime", ".codex", ".ssh", ".gnupg":
			return true
		}
	}
	base := strings.ToLower(filepath.Base(path))
	if strings.HasSuffix(base, ".log") || strings.Contains(base, "transcript") {
		return true
	}
	switch base {
	case "agent_ack.json", "director_decisions.json", "agent_packet.json",
		"agent_prompt.txt", "codex_stdout.log", "codex_stderr.log",
		"codex_last_message.txt", "orquesta_shutdown_request.json",
		"agent_shutdown_checkpoint_ack.json", "auth.json":
		return true
	default:
		return false
	}
}

func codexWaveOperatorInputSourceCategoryV0(path string) string {
	if filepath.IsAbs(path) {
		return "operator_absolute_file"
	}
	return "operator_relative_file"
}

func codexWaveOperatorInputErrorCodeV0(kind string, reason string) string {
	switch kind {
	case "prompt":
		if reason == "too_large" {
			return "prompt_file_too_large"
		}
		if reason == "empty" {
			return "prompt_empty"
		}
		return "prompt_file_invalid"
	case "objective":
		if reason == "too_large" {
			return "objective_file_too_large"
		}
		return "objective_file_invalid"
	case "domain_context":
		if reason == "too_large" {
			return "domain_context_file_too_large"
		}
		if reason == "empty" {
			return "domain_context_file_empty"
		}
		return "domain_context_file_unreadable"
	default:
		return "operator_input_invalid"
	}
}

func codexWaveOperatorInputOpaqueRefV0(kind string, value string) string {
	sum := sha256.Sum256([]byte(filepath.Clean(strings.TrimSpace(value))))
	return safeCodexWavePurgeRefPartV0(kind) + "-ref-" + hex.EncodeToString(sum[:])[:16]
}

func codexWaveOperatorInputContentRefV0(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])[:16]
}

func codexWaveOperatorInputReceiptsCopyV0(items []codexWaveOperatorInputReceiptV0) []codexWaveOperatorInputReceiptV0 {
	if len(items) == 0 {
		return nil
	}
	out := make([]codexWaveOperatorInputReceiptV0, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	return out
}

func codexWaveOperatorInputHasKindV0(items []codexWaveOperatorInputReceiptV0, kind string) (codexWaveOperatorInputReceiptV0, bool) {
	for _, item := range items {
		if item.Kind == kind {
			return item, true
		}
	}
	return codexWaveOperatorInputReceiptV0{}, false
}
