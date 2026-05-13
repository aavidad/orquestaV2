package orquestaappchangedirectorsource

import (
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
)

func appChangeIsVisualExternalWorkV0(
	request orquestaappchange.AppChangeRequestV0,
) bool {
	return appChangeHasExternalWorkV0(request) &&
		strings.TrimSpace(request.ExternalWork.WorkKind) == "generate_visual_asset"
}

func appChangeVisualTaskSummaryV0() string {
	return "Generar visual pedagogico externo con formato, titulo, caption, alt_text, body y trazabilidad editorial."
}

func appChangeVisualWorkCriteriaV0(
	request orquestaappchange.AppChangeRequestV0,
) []string {
	scope := appChangeExternalWorkTitleScopeV0(request.ExternalWork)
	return []string{
		"Tratar el paquete visual " + scope + " como entrada de dominio suficiente.",
		"Devolver visual_asset con title, caption, alt_text, format y body.",
		"Preferir SVG autocontenido cuando format=svg; sin scripts, eventos JavaScript, foreignObject ni URLs remotas.",
		"No incluir pacientes, menores, personas identificables, datos personales, fotos decorativas ni marcas innecesarias.",
		"Usar etiquetas legibles en A4 y estilo sobrio de temario profesional.",
		"No entregar placeholders; si falta contenido visual suficiente, declararlo como bloqueo de dominio.",
		"Incluir source_refs si el visual deriva de normativa, datos historicos o contenido tecnico verificable.",
		"Si falta un campo requerido por el job visual, declararlo como bloqueo de dominio y no inventarlo.",
	}
}

func appChangeVisualRequiredTestsV0() []string {
	return []string{
		"validar contrato visual OPES",
		"validar formato visual solicitado",
		"validar SVG seguro si format=svg",
		"validar titulo, caption y alt_text",
		"validar ausencia de placeholders",
	}
}
