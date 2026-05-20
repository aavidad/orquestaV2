package orquestaopesbridge

import (
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const defaultExpansionTargetWordsMinV0 = "16000"

func appendExpansionDocumentContractFieldsV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	jobType string,
) []orquestadomainwork.DomainWorkFieldV0 {
	if strings.TrimSpace(jobType) != "expand_topic_from_summary" {
		return fields
	}
	if !fieldHasNameV0(fields, "required_document_variants") {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{
			Name:   "required_document_variants",
			Values: expansionDocumentVariantsV0(),
		})
	}
	if !fieldHasNameV0(fields, "target_words_min") {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{
			Name:  "target_words_min",
			Value: defaultExpansionTargetWordsMinV0,
		})
	}
	if !fieldHasNameV0(fields, "minimum_quality_gates") {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{
			Name:   "minimum_quality_gates",
			Values: expansionQualityGatesV0(),
		})
	}
	return fields
}

func expansionDocumentVariantsV0() []string {
	return []string{
		"tema_grande",
		"tema_mediano",
		"resumen",
		"esquema_repaso",
		"plan_visuales",
	}
}

func expansionQualityGatesV0() []string {
	return []string{
		"tema_grande_min_words=16000",
		"aplicar opes_global_editorial_policy_2026_05_18",
		"si level=A1/A1_A2, tema_grande debe cumplir minimo 20.250 palabras salvo target superior declarado",
		"sin secciones Pendiente de ampliacion",
		"sin placeholders ni TODO",
		"cada capitulo debe aportar desarrollo propio y trazable",
	}
}

func expansionAcceptanceCriteriaV0() []string {
	return []string{
		"paquete apto para tema_grande con capitulos y bloques trazables",
		"tema_grande debe alcanzar target_words_min si esta declarado",
		"aplicar opes_global_editorial_policy_2026_05_18: modo tutor completo, no infantilizar, notas de test separadas y visuales utiles",
		"no dejar secciones Pendiente de ampliacion ni material borrador sin resolver",
		"conservar base para tema_mediano sin perder autores normativa ni procedimientos",
		"incluir resumen/memoria de repaso derivado del tema desarrollado",
		"incluir esquema de examen y plan de visuales cuando aporten valor",
	}
}
