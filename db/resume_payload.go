package db

import (
	"encoding/json"
	"strings"
)

// ParseResumePayloadEnvelope normaliza un resume payload previo como objeto JSON.
// Si el payload ya era un objeto, copia sus claves tal cual. Si era JSON válido
// pero no objeto, lo conserva como resume_previo. Si era texto no JSON, lo
// conserva como resume_previo_raw.
func ParseResumePayloadEnvelope(prev string) map[string]any {
	prev = strings.TrimSpace(prev)
	envelope := map[string]any{}
	if prev == "" {
		return envelope
	}
	var parsed any
	if err := json.Unmarshal([]byte(prev), &parsed); err != nil {
		envelope["resume_previo_raw"] = prev
		return envelope
	}
	if obj, ok := parsed.(map[string]any); ok {
		for key, value := range obj {
			envelope[key] = value
		}
		return envelope
	}
	envelope["resume_previo"] = parsed
	return envelope
}

// MergeResumePayloadEnvelope mezcla claves sobre el envelope existente sin
// reanidar el payload previo. Las claves nuevas sobrescriben las existentes.
func MergeResumePayloadEnvelope(prev string, additions map[string]any) string {
	prev = strings.TrimSpace(prev)
	if len(additions) == 0 {
		return prev
	}
	envelope := ParseResumePayloadEnvelope(prev)
	for key, value := range additions {
		if strings.TrimSpace(key) == "" {
			continue
		}
		envelope[key] = value
	}
	if len(envelope) == 0 {
		return prev
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		return prev
	}
	return string(data)
}
