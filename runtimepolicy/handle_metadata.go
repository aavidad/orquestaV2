package runtimepolicy

import (
	"fmt"
	"path/filepath"
	"strings"
)

func CompactRuntimeHandleMetadata(meta map[string]any) {
	if meta == nil {
		return
	}
	for _, key := range []string{"bootstrap_prompt", "continuity_prompt"} {
		summarizePromptMetadata(meta, key, mapValueString(meta, key, ""))
	}
	summarizePromptMetadata(meta, "resumen_continuidad", mapValueString(meta, "resumen_continuidad", ""))
	for _, key := range []string{"rendered_command", "wrapped_command"} {
		compactCommandMetadata(meta, key)
	}
}

func mapValueString(m map[string]any, key, fallback string) string {
	if m == nil {
		return fallback
	}
	value, ok := m[key]
	if !ok || value == nil {
		return fallback
	}
	v, ok := value.(string)
	if !ok {
		return fallback
	}
	return strings.TrimSpace(v)
}

func summarizePromptMetadata(meta map[string]any, key, raw string) {
	if meta == nil {
		return
	}
	raw = strings.TrimSpace(raw)
	delete(meta, key)
	delete(meta, key+"_summary")
	delete(meta, key+"_len")
	if raw == "" {
		return
	}
	meta[key+"_summary"] = summarizePromptHandle(raw)
	meta[key+"_len"] = len([]rune(raw))
}

func summarizePromptHandle(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	first := raw
	if idx := strings.IndexByte(first, '\n'); idx >= 0 {
		first = first[:idx]
	}
	first = strings.Join(strings.Fields(strings.TrimSpace(first)), " ")
	runes := []rune(first)
	if len(runes) > 160 {
		first = strings.TrimSpace(string(runes[:160])) + "..."
	}
	return first
}

func compactCommandMetadata(meta map[string]any, key string) {
	if meta == nil {
		return
	}
	raw := strings.TrimSpace(mapValueString(meta, key, ""))
	if raw == "" {
		delete(meta, key+"_len")
		return
	}
	delete(meta, key+"_len")
	compactado := compactRuntimeHandleCommand(raw)
	if compactado == raw {
		return
	}
	meta[key] = compactado
	meta[key+"_len"] = len([]rune(raw))
}

func compactRuntimeHandleCommand(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	tokens, err := splitShellQuotedCommandRuntimeHandle(raw)
	if err != nil || len(tokens) == 0 {
		return raw
	}
	base := filepath.Base(strings.TrimSpace(tokens[0]))
	switch base {
	case "codex-perfil":
		if len(tokens) >= 2 {
			return joinShellQuotedTokens(tokens[:2])
		}
		return joinShellQuotedTokens(tokens[:1])
	case "codex":
		return joinShellQuotedTokens(tokens[:1])
	case "script":
		keep := minInt(len(tokens), 4)
		out := append([]string{}, tokens[:keep]...)
		if len(tokens) > keep {
			last := strings.TrimSpace(tokens[len(tokens)-1])
			if last != "" && last != tokens[keep-1] {
				out = append(out, "<omitted>", last)
			}
		}
		return joinShellQuotedTokens(out)
	default:
		if len(raw) <= 240 {
			return raw
		}
		return strings.TrimSpace(string([]rune(raw)[:240])) + "..."
	}
}

func joinShellQuotedTokens(tokens []string) string {
	if len(tokens) == 0 {
		return ""
	}
	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		out = append(out, shellQuoteRuntimeHandle(token))
	}
	return strings.Join(out, " ")
}

func shellQuoteRuntimeHandle(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(raw, "'", `'\''`) + "'"
}

func splitShellQuotedCommandRuntimeHandle(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var out []string
	var token strings.Builder
	inSingle := false
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		switch {
		case inSingle && ch == '\'':
			inSingle = false
		case !inSingle && ch == '\'':
			inSingle = true
		case !inSingle && (ch == ' ' || ch == '\t' || ch == '\n'):
			if token.Len() > 0 {
				out = append(out, token.String())
				token.Reset()
			}
		case ch == '\\' && i+1 < len(raw):
			i++
			token.WriteByte(raw[i])
		default:
			token.WriteByte(ch)
		}
	}
	if inSingle {
		return nil, fmt.Errorf("runtime_handle command con comillas sin cerrar")
	}
	if token.Len() > 0 {
		out = append(out, token.String())
	}
	return out, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
