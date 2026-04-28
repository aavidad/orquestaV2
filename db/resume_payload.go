package db

import (
	"encoding/json"
	"strings"
)

const claveResumePayloadPerfilEjecucion = "perfil_ejecucion"

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

// ResumePayloadPerfilEjecucion extrae el perfil de ejecución persistido dentro
// del envelope de resume sin asumir ninguna forma legacy fuera de ese bloque.
func ResumePayloadPerfilEjecucion(prev string) (string, string, string) {
	envelope := ParseResumePayloadEnvelope(prev)
	raw := envelope[claveResumePayloadPerfilEjecucion]
	perfil, ok := raw.(map[string]any)
	if !ok {
		return "", "", ""
	}
	return strings.TrimSpace(stringFromAny(perfil["perfil_tarea"])),
		strings.TrimSpace(stringFromAny(perfil["modelo"])),
		strings.TrimSpace(stringFromAny(perfil["razonamiento"]))
}

func ResumePayloadPerfilOperativo(prev string) string {
	envelope := ParseResumePayloadEnvelope(prev)
	raw := envelope[claveResumePayloadPerfilEjecucion]
	perfil, ok := raw.(map[string]any)
	if !ok {
		return ""
	}
	return strings.TrimSpace(stringFromAny(perfil["perfil_operativo"]))
}

// MergeResumePayloadPerfilEjecucion persiste el perfil de ejecución actual en
// el resume payload para que reanudaciones y rearmes automáticos no vuelvan a
// caer a defaults del conector si ya existía una decisión previa.
func MergeResumePayloadPerfilEjecucion(prev, perfilTarea, modelo, razonamiento string) string {
	perfilTarea = strings.TrimSpace(perfilTarea)
	modelo = strings.TrimSpace(modelo)
	razonamiento = strings.TrimSpace(razonamiento)
	actualPerfil, actualModelo, actualRazonamiento := ResumePayloadPerfilEjecucion(prev)
	if perfilTarea == "" {
		perfilTarea = actualPerfil
	}
	if modelo == "" {
		modelo = actualModelo
	}
	if razonamiento == "" {
		razonamiento = actualRazonamiento
	}
	if perfilTarea == "" && modelo == "" && razonamiento == "" {
		return strings.TrimSpace(prev)
	}
	perfilPayload := map[string]any{}
	if raw, ok := ParseResumePayloadEnvelope(prev)[claveResumePayloadPerfilEjecucion].(map[string]any); ok {
		for key, value := range raw {
			if strings.TrimSpace(key) == "" {
				continue
			}
			perfilPayload[key] = value
		}
	}
	perfilPayload["perfil_tarea"] = perfilTarea
	perfilPayload["modelo"] = modelo
	perfilPayload["razonamiento"] = razonamiento
	return MergeResumePayloadEnvelope(prev, map[string]any{
		claveResumePayloadPerfilEjecucion: perfilPayload,
	})
}

func MergeResumePayloadPerfilOperativo(prev, perfilOperativo string) string {
	perfilOperativo = strings.TrimSpace(perfilOperativo)
	if perfilOperativo == "" {
		return strings.TrimSpace(prev)
	}
	envelope := ParseResumePayloadEnvelope(prev)
	perfilPayload, _ := envelope[claveResumePayloadPerfilEjecucion].(map[string]any)
	out := map[string]any{}
	for key, value := range perfilPayload {
		if strings.TrimSpace(key) == "" {
			continue
		}
		out[key] = value
	}
	out["perfil_operativo"] = perfilOperativo
	return MergeResumePayloadEnvelope(prev, map[string]any{
		claveResumePayloadPerfilEjecucion: out,
	})
}
