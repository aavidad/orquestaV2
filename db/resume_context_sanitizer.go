/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"orquesta/runtimeagente"
)

type resumeProjectIdentity struct {
	slug string
	ruta string
}

var resumePayloadLaunchOnlyKeys = []string{
	"modo",
	"native_resume",
	"continuity_prompt",
	"bootstrap_prompt",
	"launch_prompt_embedded",
	"launch_prompt_mode",
	"launch_prompt_delay_ms",
}

func SanitizeResumeContextForProject(resume runtimeagente.ResumeContext, proyecto *Proyecto) runtimeagente.ResumeContext {
	originalPayload := strings.TrimSpace(resume.ResumePayloadJSON)
	foreignPayload, foreignProject := resumePayloadForeignProject(originalPayload, proyecto)
	resume.ResumePayloadJSON = SanitizeResumePayloadForProject(originalPayload, proyecto)
	if proyecto == nil {
		return resume
	}
	if foreignPayload {
		resume.ResumenContinuidad = ""
		resume.Branch = ""
		resume.ExternalSessionID = ""
		if shouldResetResumeCWD(resume.CWD, foreignProject.ruta, proyecto.RutaAbs) {
			resume.CWD = ""
		}
	}
	resume.ResumenContinuidad = SanitizeContinuitySummaryForProject(resume.ResumenContinuidad, proyecto)
	resume.ExternalSessionID = strings.TrimSpace(resume.ExternalSessionID)
	resume.ResumePayloadJSON = strings.TrimSpace(resume.ResumePayloadJSON)
	resume.ResumenContinuidad = strings.TrimSpace(resume.ResumenContinuidad)
	resume.Branch = strings.TrimSpace(resume.Branch)
	resume.CWD = strings.TrimSpace(resume.CWD)
	return resume
}

func SanitizeResumePayloadForProject(prev string, proyecto *Proyecto) string {
	envelope := ParseResumePayloadEnvelope(prev)
	for _, key := range resumePayloadLaunchOnlyKeys {
		delete(envelope, key)
	}
	if len(envelope) == 0 {
		return ""
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		return ""
	}
	prev = string(data)
	if proyecto == nil {
		return prev
	}
	envelope = ParseResumePayloadEnvelope(prev)
	foreign, _ := envelopeReferencesForeignProject(envelope, proyecto)
	if !foreign {
		return prev
	}
	for _, key := range []string{
		"project_context",
		"governance_catalog",
		"adopted_context",
		"runtime_order",
		"mailbox",
		"checkpoint",
	} {
		delete(envelope, key)
	}
	if len(envelope) == 0 {
		return ""
	}
	data, err = json.Marshal(envelope)
	if err != nil {
		return ""
	}
	return string(data)
}

func SanitizeContinuitySummaryForProject(summary string, proyecto *Proyecto) string {
	summary = strings.TrimSpace(summary)
	if summary == "" || proyecto == nil {
		return summary
	}
	if continuityMentionsForeignProject(summary, proyecto) {
		return ""
	}
	return summary
}

func unirPartesUnicasResume(items []string) string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return strings.Join(out, ". ")
}

func limpiarResumenBootstrapPrevio(prev string) string {
	partes := splitContinuityClauses(prev)
	filtradas := make([]string, 0, len(partes))
	for _, item := range partes {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.HasPrefix(item, "Mailbox: ") && strings.Contains(item, "mensaje(s) inyectados") {
			continue
		}
		if strings.HasPrefix(item, "Checkpoint: ") || strings.HasPrefix(item, "Checkpoint #") {
			continue
		}
		filtradas = append(filtradas, item)
	}
	return unirPartesUnicasResume(filtradas)
}

func resumenCheckpointBootstrap(checkpoint *RuntimeCheckpoint, prev string) string {
	if checkpoint == nil {
		return ""
	}
	label := checkpointLabel(checkpoint)
	resumen := strings.TrimSpace(checkpoint.Resumen)
	if resumen == "" {
		return label
	}
	prev = strings.TrimSpace(prev)
	if prev != "" && (resumen == prev || strings.Contains(resumen, prev) || strings.Contains(prev, resumen)) {
		return label
	}
	if resumeCheckpointShouldCompact(resumen) {
		return label
	}
	if label == "" {
		return "Checkpoint: " + resumen
	}
	return label + ": " + resumen
}

func resumenCheckpointPayload(checkpoint *RuntimeCheckpoint) string {
	if checkpoint == nil {
		return ""
	}
	resumen := strings.TrimSpace(checkpoint.Resumen)
	if resumen == "" {
		return checkpointLabel(checkpoint)
	}
	if resumeCheckpointShouldCompact(resumen) {
		return checkpointLabel(checkpoint)
	}
	return resumen
}

func checkpointLabel(checkpoint *RuntimeCheckpoint) string {
	if checkpoint == nil {
		return ""
	}
	label := "Checkpoint"
	if checkpoint.ID > 0 {
		label += fmt.Sprintf(" #%d", checkpoint.ID)
	}
	if kind := strings.TrimSpace(checkpoint.CheckpointKind); kind != "" {
		label += " (" + kind + ")"
	}
	return label
}

func resumeCheckpointShouldCompact(resumen string) bool {
	resumen = strings.TrimSpace(resumen)
	if resumen == "" {
		return false
	}
	return strings.Contains(resumen, "Mailbox: ") ||
		strings.Contains(resumen, "Catálogo efectivo ") ||
		strings.Contains(resumen, "tarea(s) activas")
}

func splitContinuityClauses(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ". ")
	out := make([]string, 0, len(parts))
	for _, item := range parts {
		item = strings.TrimSpace(strings.TrimSuffix(item, "."))
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func resumePayloadForeignProject(prev string, proyecto *Proyecto) (bool, resumeProjectIdentity) {
	if proyecto == nil {
		return false, resumeProjectIdentity{}
	}
	envelope := ParseResumePayloadEnvelope(prev)
	return envelopeReferencesForeignProject(envelope, proyecto)
}

func envelopeReferencesForeignProject(envelope map[string]any, proyecto *Proyecto) (bool, resumeProjectIdentity) {
	if proyecto == nil || len(envelope) == 0 {
		return false, resumeProjectIdentity{}
	}
	contexto, ok := envelope["project_context"].(map[string]any)
	if !ok {
		return false, resumeProjectIdentity{}
	}
	identity := projectIdentityFromEnvelope(contexto)
	if identity.slug == "" && identity.ruta == "" {
		return false, identity
	}
	return !projectIdentityMatches(identity, proyecto), identity
}

func projectIdentityFromEnvelope(contexto map[string]any) resumeProjectIdentity {
	identity := resumeProjectIdentity{
		slug: strings.TrimSpace(stringFromAny(contexto["slug"])),
		ruta: firstNonEmptyResume(
			stringFromAny(contexto["ruta_abs"]),
			stringFromAny(contexto["ruta"]),
		),
	}
	if nested, ok := contexto["proyecto"].(map[string]any); ok {
		if identity.slug == "" {
			identity.slug = strings.TrimSpace(stringFromAny(nested["slug"]))
		}
		if identity.ruta == "" {
			identity.ruta = firstNonEmptyResume(
				stringFromAny(nested["ruta_abs"]),
				stringFromAny(nested["ruta"]),
			)
		}
	}
	identity.ruta = normalizarRutaProyecto(identity.ruta)
	return identity
}

func projectIdentityMatches(identity resumeProjectIdentity, proyecto *Proyecto) bool {
	if proyecto == nil {
		return true
	}
	if slug := strings.TrimSpace(identity.slug); slug != "" && !strings.EqualFold(slug, strings.TrimSpace(proyecto.Slug)) {
		return false
	}
	if ruta := normalizarRutaProyecto(identity.ruta); ruta != "" && ruta != normalizarRutaProyecto(proyecto.RutaAbs) {
		return false
	}
	return true
}

func shouldResetResumeCWD(cwd, foreignRuta, currentRuta string) bool {
	cwd = normalizarRutaProyecto(cwd)
	foreignRuta = normalizarRutaProyecto(foreignRuta)
	currentRuta = normalizarRutaProyecto(currentRuta)
	if cwd == "" || foreignRuta == "" || foreignRuta == currentRuta {
		return false
	}
	return cwd == foreignRuta
}

func continuityMentionsForeignProject(summary string, proyecto *Proyecto) bool {
	summary = strings.TrimSpace(summary)
	if summary == "" || proyecto == nil {
		return false
	}
	currentSlug := strings.ToLower(strings.TrimSpace(proyecto.Slug))
	for _, marker := range []string{
		"en el proyecto ",
		"contexto adoptado por orquesta sobre ",
		"proyecto: ",
	} {
		if slug := firstProjectSlugAfterMarker(summary, marker); slug != "" && slug != currentSlug {
			return true
		}
	}
	return false
}

func firstProjectSlugAfterMarker(summary, marker string) string {
	lowerSummary := strings.ToLower(summary)
	marker = strings.ToLower(strings.TrimSpace(marker))
	if marker == "" {
		return ""
	}
	pos := strings.Index(lowerSummary, marker)
	if pos < 0 {
		return ""
	}
	start := pos + len(marker)
	for start < len(lowerSummary) {
		r := rune(lowerSummary[start])
		if unicode.IsSpace(r) || r == ':' {
			start++
			continue
		}
		break
	}
	if start >= len(lowerSummary) {
		return ""
	}
	var b strings.Builder
	for _, r := range lowerSummary[start:] {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			b.WriteRune(r)
			continue
		}
		break
	}
	return strings.TrimSpace(b.String())
}

func stringFromAny(v any) string {
	if v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func firstNonEmptyResume(items ...string) string {
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			return item
		}
	}
	return ""
}
