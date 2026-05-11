package orquestafactory

import "strings"

const (
	RequestKindDocumentarAppV0             = "documentar_app"
	RequestKindAnalizarAppV0               = "analizar_app"
	RequestKindBrainstormingArquitecturaV0 = "brainstorming_arquitectura"
	RequestKindPlanificarAppV0             = "planificar_app"
	RequestKindCrearAppCompletaV0          = "crear_app_completa"
	RequestKindProgramarModuloV0           = "programar_modulo"
	RequestKindModificarAppExistenteV0     = "modificar_app_existente"
	RequestKindRevisarCodigoV0             = "revisar_codigo"
	RequestKindPruebasYValidacionV0        = "pruebas_y_validacion"
	RequestKindSeguridadV0                 = "seguridad"
	RequestKindDeployV0                    = "deploy"
	RequestKindOperacionSoporteV0          = "operacion_soporte"
	RequestKindIntegracionExternaV0        = "integracion_externa"
	RequestKindI18NL10NV0                  = "i18n_l10n"
	RequestKindMigracionRefactorV0         = "migracion_refactor"
	RequestKindInvestigacionTecnicaV0      = "investigacion_tecnica"
	DefaultRequestKindV0                   = RequestKindCrearAppCompletaV0
	ExecutionModeNormalV0                  = "normal"
	ExecutionModeDebugV0                   = "debug"
	DefaultExecutionModeV0                 = ExecutionModeNormalV0
)

var supportedRequestKindsV0 = []string{
	RequestKindDocumentarAppV0,
	RequestKindAnalizarAppV0,
	RequestKindBrainstormingArquitecturaV0,
	RequestKindPlanificarAppV0,
	RequestKindCrearAppCompletaV0,
	RequestKindProgramarModuloV0,
	RequestKindModificarAppExistenteV0,
	RequestKindRevisarCodigoV0,
	RequestKindPruebasYValidacionV0,
	RequestKindSeguridadV0,
	RequestKindDeployV0,
	RequestKindOperacionSoporteV0,
	RequestKindIntegracionExternaV0,
	RequestKindI18NL10NV0,
	RequestKindMigracionRefactorV0,
	RequestKindInvestigacionTecnicaV0,
}

var supportedExecutionModesV0 = []string{
	ExecutionModeNormalV0,
	ExecutionModeDebugV0,
}

func SupportedRequestKindsV0() []string {
	return append([]string(nil), supportedRequestKindsV0...)
}

func SupportedExecutionModesV0() []string {
	return append([]string(nil), supportedExecutionModesV0...)
}

func NormalizeRequestKindV0(value string) string {
	return firstNonEmptyV0(strings.ToLower(strings.TrimSpace(value)), DefaultRequestKindV0)
}

func NormalizeExecutionModeV0(value string) string {
	return firstNonEmptyV0(strings.ToLower(strings.TrimSpace(value)), DefaultExecutionModeV0)
}

func RequestKindSupportedV0(value string) bool {
	return containsV0(NormalizeRequestKindV0(value), supportedRequestKindsV0...)
}

func ExecutionModeSupportedV0(value string) bool {
	return containsV0(NormalizeExecutionModeV0(value), supportedExecutionModesV0...)
}
