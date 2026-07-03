package orquestaruntimecodexappserver

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"strconv"
	"strings"

	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
)

const codexAppServerThreadTextMaxBytesV0 = orquestaruntimecodexgoal.CodexGoalToolOutputMaxBytesV0
const codexAppServerThreadItemsViewMaxBytesV0 = 4 * 1024
const codexAppServerThreadOutputSanitizedEvidenceRefV0 = "evidence-ref-codex-app-server-thread-output-sanitized"

func sanitizeCodexAppServerRPCDecodedOutV0(out interface{}) {
	switch value := out.(type) {
	case *serverCodexAppServerThreadReadResponseV0:
		thread, _ := sanitizeCodexAppServerThreadReadV0(value.Thread)
		value.Thread = thread
	case *serverCodexAppServerThreadReadV0:
		thread, _ := sanitizeCodexAppServerThreadReadV0(*value)
		*value = thread
	}
}

func sanitizeCodexAppServerThreadReadV0(
	thread serverCodexAppServerThreadReadV0,
) (serverCodexAppServerThreadReadV0, bool) {
	sanitized := false
	for turnIndex := range thread.Turns {
		if text, ok := sanitizeCodexAppServerOperationalTextV0(
			thread.Turns[turnIndex].ItemsView,
			codexAppServerThreadItemsViewMaxBytesV0,
			false,
		); ok {
			thread.Turns[turnIndex].ItemsView = text
			sanitized = true
		}
		for itemIndex := range thread.Turns[turnIndex].Items {
			if text, ok := sanitizeCodexAppServerOperationalTextV0(
				thread.Turns[turnIndex].Items[itemIndex].Text,
				codexAppServerThreadTextMaxBytesV0,
				true,
			); ok {
				thread.Turns[turnIndex].Items[itemIndex].Text = text
				sanitized = true
			}
		}
	}
	return thread, sanitized
}

func sanitizeCodexAppServerOperationalTextV0(
	value string,
	maxBytes int,
	preserveGoalMarker bool,
) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return value, false
	}
	sanitized, changed := redactCodexAppServerDataURIsV0(value)
	if len(sanitized) <= maxBytes {
		return sanitized, changed
	}
	if preserveGoalMarker {
		if marker := codexAppServerGoalResultMarkerWindowV0(sanitized); marker != "" {
			return marker, true
		}
	}
	return "codex_app_server_operational_text_sanitized " +
		codexAppServerGoalResultSanitizedRefV0("thread-output-ref", sanitized), true
}

func redactCodexAppServerDataURIsV0(value string) (string, bool) {
	lower := strings.ToLower(value)
	start := strings.Index(lower, "data:")
	if start < 0 {
		return value, false
	}
	var out strings.Builder
	cursor := 0
	changed := false
	for start >= 0 {
		start += cursor
		headerEndRelative := strings.Index(strings.ToLower(value[start:]), ";base64,")
		if headerEndRelative < 0 {
			break
		}
		headerEnd := start + headerEndRelative + len(";base64,")
		payloadEnd := headerEnd
		for payloadEnd < len(value) && isCodexAppServerBase64DataURICharV0(value[payloadEnd]) {
			payloadEnd++
		}
		if payloadEnd == headerEnd {
			cursor = headerEnd
			next := strings.Index(strings.ToLower(value[cursor:]), "data:")
			start = next
			continue
		}
		out.WriteString(value[cursor:start])
		out.WriteString(codexAppServerDataURIProjectionV0(value[start:headerEnd], value[headerEnd:payloadEnd]))
		cursor = payloadEnd
		changed = true
		next := strings.Index(strings.ToLower(value[cursor:]), "data:")
		start = next
	}
	if !changed {
		return value, false
	}
	out.WriteString(value[cursor:])
	return out.String(), true
}

func isCodexAppServerBase64DataURICharV0(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') ||
		(ch >= 'a' && ch <= 'z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '+' ||
		ch == '/' ||
		ch == '='
}

func codexAppServerDataURIProjectionV0(header string, encoded string) string {
	normalizedHeader := strings.ToLower(strings.TrimSpace(header))
	mime := strings.TrimSpace(strings.TrimPrefix(strings.Split(strings.TrimSuffix(normalizedHeader, ";base64,"), ";")[0], "data:"))
	if mime == "" {
		mime = "application/octet-stream"
	}
	sum := sha256.Sum256([]byte(header + encoded))
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	decodedBytes := 0
	width := 0
	height := 0
	if err == nil {
		decodedBytes = len(decoded)
		if cfg, _, cfgErr := image.DecodeConfig(bytes.NewReader(decoded)); cfgErr == nil {
			width = cfg.Width
			height = cfg.Height
		}
	}
	parts := []string{
		"orquesta_multimodal_payload_ref",
		"mime=" + mime,
		"sha256=" + hex.EncodeToString(sum[:])[:12],
		"encoded_bytes=" + strconv.Itoa(len(encoded)),
	}
	if decodedBytes > 0 {
		parts = append(parts, "decoded_bytes="+strconv.Itoa(decodedBytes))
	}
	if width > 0 && height > 0 {
		parts = append(parts, fmt.Sprintf("width=%d height=%d", width, height))
	}
	return "[" + strings.Join(parts, " ") + "]"
}
