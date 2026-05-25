package orquestaapprunner

const (
	AppRunnerRouteModePreviewCompatV0       = "preview_compatibilidad"
	AppRunnerPreferredEntrypointDirectorV0  = "orquesta.apps.arrancar_director.v0"
	AppRunnerLegacyEntrypointPrepareV0      = "orquesta.apps.preparar_orquestacion.v0"
	AppRunnerLegacyEntrypointExecuteV0      = "orquesta.apps.ejecutar_orquestacion.v0"
	AppRunnerDirectorV2RequiredFieldV0      = "director_v2_required"
	AppRunnerDirectorV2RequiredReasonV0     = "objetivo_requiere_director_v2_plan_state_waits_review_tests_cierre_o_recursion"
	AppRunnerPreviewCompatibilityReasonV0   = "app_runner_devuelve_app_plan_sin_plan_state_operativo"
	AppRunnerPreviewCompatibilityEvidenceV0 = "evidence-ref-app-runner-routing-preview-compat-v0"
)

type AppRunnerRoutePolicyV0 struct {
	Mode                string `json:"mode"`
	PreferredEntrypoint string `json:"preferred_entrypoint"`
	LegacyEntrypoint    string `json:"legacy_entrypoint,omitempty"`
	PublicReason        string `json:"public_reason"`
}

func AppRunnerPreviewRoutePolicyV0(legacyEntrypoint string) AppRunnerRoutePolicyV0 {
	return AppRunnerRoutePolicyV0{
		Mode:                AppRunnerRouteModePreviewCompatV0,
		PreferredEntrypoint: AppRunnerPreferredEntrypointDirectorV0,
		LegacyEntrypoint:    legacyEntrypoint,
		PublicReason:        AppRunnerPreviewCompatibilityReasonV0,
	}
}

func validateDirectorV2RequirementForAppRunnerV0(
	request RunPreparedAppOrchestrationRequestV0,
) error {
	if !request.RequireDirectorV2 {
		return nil
	}
	return AppRunnerIssueV0{Field: AppRunnerDirectorV2RequiredFieldV0}
}
