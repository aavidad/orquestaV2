package orquestaopesbridge

import (
	"sort"
	"strings"
	"unicode"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func appendOpaqueExecutionRefFieldsV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	externalRefs map[string]string,
) ([]orquestadomainwork.DomainWorkFieldV0, bool) {
	refs := opaqueExecutionRefsFromExternalRefsV0(externalRefs)
	for _, name := range opaqueExecutionRefFieldNamesV0() {
		value := strings.TrimSpace(refs[name])
		if value == "" {
			value = fieldValueV0(fields, name)
		}
		if value == "" {
			continue
		}
		if !isOpaqueExecutionRefV0(value) {
			return fields, false
		}
		fields = appendFieldIfMissingV0(fields, name, value)
	}
	return fields, true
}

func opaqueExecutionRefsFromExternalRefsV0(refs map[string]string) map[string]string {
	if len(refs) == 0 {
		return map[string]string{}
	}
	keys := make([]string, 0, len(refs))
	for key := range refs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := map[string]string{}
	for _, key := range keys {
		normalized := strings.TrimSpace(key)
		switch normalized {
		case "worktree_ref", "branch_ref":
			out[normalized] = strings.TrimSpace(refs[key])
		}
	}
	return out
}

func opaqueExecutionRefFieldNamesV0() []string {
	return []string{"worktree_ref", "branch_ref"}
}

func isOpaqueExecutionRefV0(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && !strings.ContainsAny(value, " /\\\t\n\r")
}

func expectedArtifactTypeV0(jobType string) string {
	return orquestadomainwork.ExpectedDomainWorkArtifactTypeForWorkKindV0(jobType)
}

func contextProfileForJobTypeV0(jobType string) string {
	switch strings.TrimSpace(jobType) {
	case "summarize_topic",
		"expand_topic_from_summary",
		"plan_documento",
		"plan_tema",
		"plan_temario",
		"draft_content_block",
		"review_legal",
		"review_pedagogical",
		"review_quality",
		"validate_topic",
		"assemble_topic",
		"generate_audio_asset",
		"generate_topic_audio":
		return "large"
	default:
		return "standard"
	}
}

func userIntentForJobV0(jobType string) string {
	return "Resolver job OPES " + strings.TrimSpace(jobType) +
		" y devolver artefacto " + expectedArtifactTypeV0(jobType) +
		" por el contrato publico OPES."
}

func acceptanceCriteriaForJobV0(jobType string) []string {
	criteria := []string{
		"devolver artifact_type=" + expectedArtifactTypeV0(jobType),
		"payload_json valido y trazable",
		"sin placeholders",
		"sin leer internals de OPES",
		"entrega en fichero unico bajo allowed_write_set",
	}
	if strings.TrimSpace(jobType) == "expand_topic_from_summary" {
		criteria = append(criteria, expansionAcceptanceCriteriaV0()...)
	}
	if isDocumentPlanJobTypeV0(jobType) {
		criteria = append(criteria, documentPlanAcceptanceCriteriaV0()...)
	}
	switch strings.TrimSpace(jobType) {
	case "generate_audio_asset", "generate_topic_audio":
		criteria = append(criteria, topicAudioAcceptanceCriteriaV0()...)
	}
	return criteria
}

func topicAudioAcceptanceCriteriaV0() []string {
	return []string{
		"derivar el audio desde el tema ensamblado aprobado o refs de paquete final",
		"devolver manifest de audio con idioma, formatos, duracion aproximada y checksum o refs de artefactos",
		"mantener texto narrado trazable a secciones del tema sin inventar contenido nuevo",
		"no incluir rutas locales, proveedor, GPU, modelo ni procesos internos en el payload publico",
	}
}

func constraintsForJobV0() []string {
	return []string{
		"no inventar contenido",
		"no leer DB ni ficheros internos de OPES",
		"usar solo el paquete de dominio recibido",
		"si falta contexto obligatorio declarar bloqueo",
	}
}

func opesJobWriteSetV0(workKind string, safeJob string) string {
	return "external/opes/" + strings.TrimSpace(workKind) + "/" + strings.TrimSpace(safeJob)
}

func compactOPESBridgeRefV0(value string) string {
	value = strings.TrimSpace(value)
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '_', r == '.', r == '-':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "opes"
	}
	return out
}

func compactStringsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	if out == nil {
		return []string{}
	}
	return out
}

func firstNonEmptyV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
