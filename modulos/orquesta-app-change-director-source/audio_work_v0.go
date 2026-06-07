package orquestaappchangedirectorsource

import orquestaappchange "orquesta/modulos/orquesta-app-change"

func appChangeIsAudioExternalWorkV0(
	request orquestaappchange.AppChangeRequestV0,
) bool {
	return appChangeExternalWorkKindHasClassV0(
		request,
		appChangeExternalWorkKindAudioV0,
	)
}

func appChangeAudioTaskSummaryV0() string {
	return "Generar audio accesible externo desde tema ensamblado o paquete final, con manifest y trazabilidad editorial."
}

func appChangeAudioWorkCriteriaV0(
	request orquestaappchange.AppChangeRequestV0,
) []string {
	scope := appChangeExternalWorkTitleScopeV0(request.ExternalWork)
	return []string{
		"Tratar el paquete de audio " + scope + " como entrada de dominio suficiente.",
		"Devolver audio_asset con manifest de idioma, formato, duracion y refs/checksums de artefactos.",
		"Derivar el audio desde assembled_topic o refs opacas del paquete final aprobado.",
		"Mantener el texto narrado trazable a secciones del tema sin inventar contenido nuevo.",
		"Si aparecen rutas locales, proveedor, GPU, modelo o procesos internos, registrarlos como nota de saneamiento antes del payload publico.",
		"Si falta el tema ensamblado o audio_profile_ref requerido, conservar manifest parcial y dejar nota de rework de dominio.",
	}
}

func appChangeAudioRequiredTestsV0(
	request orquestaappchange.AppChangeRequestV0,
) []string {
	scope := appChangeExternalWorkTitleScopeV0(request.ExternalWork)
	return []string{
		"validar contrato audio " + scope,
		"validar artifact_type=audio_asset",
		"validar manifest de audio accesible",
		"validar trazabilidad al tema ensamblado",
		"validar notas de saneamiento de proveedor, GPU, modelo, rutas o procesos internos",
	}
}
