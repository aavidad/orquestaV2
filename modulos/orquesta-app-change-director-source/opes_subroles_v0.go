package orquestaappchangedirectorsource

import (
	"strconv"
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const appChangeOPESParentSixSubrolesInterfaceV0 = "opes.padre-tema-6-subroles.v1"

type appChangeOPESSubroleV0 struct {
	Ref          string
	Title        string
	Summary      string
	Criteria     string
	RequiredTest string
	SkillRef     string
}

func appChangeRequiresOPESParentSixSubrolesV0(
	request orquestaappchange.AppChangeRequestV0,
) bool {
	if !appChangeHasExternalWorkV0(request) || request.ExternalWork == nil {
		return false
	}
	if !appChangeExternalWorkLooksOPESV0(request.ExternalWork) {
		return false
	}
	for _, ref := range request.ExternalWork.InterfaceRefs {
		if strings.TrimSpace(ref) == appChangeOPESParentSixSubrolesInterfaceV0 {
			return true
		}
	}
	return appChangeSubrolesRequiredCountV0(request.ExternalWork.InputFields) >= 6
}

func appChangeExternalWorkLooksOPESV0(
	work *orquestaappchange.AppChangeExternalWorkV0,
) bool {
	if work == nil {
		return false
	}
	if strings.TrimSpace(work.ProjectRef) == "opes" ||
		strings.TrimSpace(work.ProjectRef) == "project-ref-opes" {
		return true
	}
	for _, ref := range append(append([]string{}, work.InterfaceRefs...), work.WorkRefs...) {
		normalized := strings.ToLower(strings.TrimSpace(ref))
		if normalized == "opes" || strings.HasPrefix(normalized, "opes-") || strings.HasPrefix(normalized, "opes.") {
			return true
		}
	}
	return false
}

func appChangeSubrolesRequiredCountV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) int {
	for _, field := range fields {
		if strings.TrimSpace(field.Name) != "subroles_required" {
			continue
		}
		if len(field.Values) > 0 {
			return len(field.Values)
		}
		if count, err := strconv.Atoi(strings.TrimSpace(string(field.ValueJSON))); err == nil {
			return count
		}
		if strings.EqualFold(strings.TrimSpace(string(field.ValueJSON)), "true") {
			return 6
		}
		if count, err := strconv.Atoi(strings.TrimSpace(field.Value)); err == nil {
			return count
		}
		if strings.EqualFold(strings.TrimSpace(field.Value), "true") {
			return 6
		}
	}
	return 0
}

func appChangeOPESSubrolesV0() []appChangeOPESSubroleV0 {
	return []appChangeOPESSubroleV0{
		{
			Ref:          "fuentes",
			Title:        "Subagente OPES S1 fuentes",
			Summary:      "Localizar y comprobar bases, programa, fuentes oficiales y vigencia del tema.",
			Criteria:     "Entregar matriz de fuentes oficiales, vigencia, huecos y evidencia compacta del tema.",
			RequiredTest: "validar fuentes oficiales y alcance del tema",
			SkillRef:     "opes-investigacion-bases",
		},
		{
			Ref:          "reutilizacion",
			Title:        "Subagente OPES S2 reutilizacion",
			Summary:      "Inventariar material local reutilizable: comunes, temas previos, tests, visuales, HTML, RAG y audio.",
			Criteria:     "Entregar decision de reutilizacion, adaptacion o rework sin rehacer material valido.",
			RequiredTest: "validar inventario local y reutilizacion antes de crear",
			SkillRef:     "opes-inventario-reutilizacion",
		},
		{
			Ref:          "redaccion",
			Title:        "Subagente OPES S3 redaccion",
			Summary:      "Redactar, ampliar o sanear solo las lagunas textuales detectadas.",
			Criteria:     "Entregar texto o rework textual trazable, en castellano correcto y sin metacomentarios publicables.",
			RequiredTest: "validar redaccion, extension y castellano publicable",
			SkillRef:     "opes-redaccion-temarios",
		},
		{
			Ref:          "visuales",
			Title:        "Subagente OPES S4 visuales",
			Summary:      "Planificar o revisar visuales utiles, anclados al apartado y con alt/leyenda correctos.",
			Criteria:     "Entregar mapa de visuales reutilizados, pendientes o rechazados por falta de funcion didactica.",
			RequiredTest: "validar visuales con anclaje semantico y calidad",
			SkillRef:     "opes-infografias-temario",
		},
		{
			Ref:          "tests-tutor",
			Title:        "Subagente OPES S5 tests-tutor",
			Summary:      "Crear o auditar tests, supuestos, tutor y explicaciones contra el temario propio terminado.",
			Criteria:     "Entregar banco o auditoria con cuatro opciones, explicacion util y evidencia de apartado/ancla.",
			RequiredTest: "validar tests y tutor contra temario propio",
			SkillRef:     "opes-tests-4-respuestas",
		},
		{
			Ref:          "html-rag-audio-qa",
			Title:        "Subagente OPES S6 html-rag-audio-qa",
			Summary:      "Preparar o auditar HTML canon, RAG, audio, manifests, capturas y QA final.",
			Criteria:     "Entregar checklist de HTML/RAG/audio/QA con bloqueos concretos y refs de evidencia.",
			RequiredTest: "validar HTML canon, RAG, audio y QA final",
			SkillRef:     "opes-qa-final-curso",
		},
	}
}

func appChangeSubroleTaskRefV0(
	refs appChangeRefSetV0,
	subroleRef string,
) string {
	return refs.TaskRef + "-subrole-" + strings.TrimSpace(subroleRef)
}

func appChangeOPESCohortRefV0(refs appChangeRefSetV0) string {
	return "cohort-ref-app-change-opes-subroles-" + refs.Suffix
}

func appChangeOPESWaveRefV0(refs appChangeRefSetV0) string {
	return "wave-ref-app-change-opes-subroles-" + refs.Suffix
}

func appChangeSubroleWriteSetV0(
	base []string,
	subroleRef string,
) []string {
	suffix := "subroles/" + strings.TrimSpace(subroleRef)
	return appChangeOPESWriteSetWithSuffixV0(base, suffix)
}

func appChangeOPESParentCoordinationWriteSetV0(base []string) []string {
	return appChangeOPESWriteSetWithSuffixV0(base, "coordinacion")
}

func appChangeOPESParentProductWriteSetV0(base []string) []string {
	out := make([]string, 0, len(base)*2)
	out = append(out, base...)
	out = append(out, appChangeOPESParentCoordinationWriteSetV0(base)...)
	return compactAppChangeSourceRefsV0(out)
}

func appChangeOPESWriteSetWithSuffixV0(base []string, suffix string) []string {
	suffix = strings.Trim(strings.TrimSpace(suffix), "/")
	out := make([]string, 0, len(base))
	for _, entry := range base {
		entry = strings.Trim(strings.TrimSpace(entry), "/")
		if entry == "" || suffix == "" {
			continue
		}
		out = append(out, entry+"/"+suffix)
	}
	return compactAppChangeSourceRefsV0(out)
}
