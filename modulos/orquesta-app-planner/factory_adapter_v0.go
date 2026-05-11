package orquestaappplanner

import (
	"strings"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func AppPlanRequestFromAppSpecV0(
	runRef string,
	spec orquestafactory.AppSpecV0,
) (AppPlanRequestV0, error) {
	if err := validateFactoryAppSpecForPlanV0(spec); err != nil {
		return AppPlanRequestV0{}, err
	}
	api, web := appSpecSurfacesV0(spec)
	return AppPlanRequestV0{
		RunRef:  strings.TrimSpace(runRef),
		AppRef:  firstAppPlanValueV0(spec.App.Slug, spec.SpecID),
		AppName: strings.TrimSpace(spec.App.Nombre),
		AppKind: strings.TrimSpace(spec.App.TipoApp),
		Scale:   appPlanScaleFromSpecV0(spec),
		API:     api,
		Web:     web,
		Locale:  strings.TrimSpace(spec.Locale),
	}, nil
}

func appPlanScaleFromSpecV0(spec orquestafactory.AppSpecV0) string {
	if spec.Data.PersistenceRequired ||
		strings.TrimSpace(spec.Quality.Tests) == "alta" ||
		appPlanDeployNeedsPlanV0(spec.Deploy.Target) {
		return AppPlanScaleLargeV0
	}
	return AppPlanScaleStandardV0
}

func appPlanDeployNeedsPlanV0(target string) bool {
	switch strings.TrimSpace(target) {
	case "", "sin_preferencia", "local":
		return false
	default:
		return true
	}
}

func BuildGoAPIWebMicrotaskPlanFromAppSpecV0(
	runRef string,
	spec orquestafactory.AppSpecV0,
) (AppMicrotaskPlanV0, error) {
	request, err := AppPlanRequestFromAppSpecV0(runRef, spec)
	if err != nil {
		return AppMicrotaskPlanV0{}, err
	}
	return BuildGoAPIWebMicrotaskPlanV0(request)
}

func validateFactoryAppSpecForPlanV0(spec orquestafactory.AppSpecV0) error {
	if strings.TrimSpace(spec.SchemaVersion) != orquestafactory.AppSpecSchemaV0 {
		return AppPlannerIssueV0{Field: "app_spec.schema_version"}
	}
	if strings.TrimSpace(spec.SpecID) == "" {
		return AppPlannerIssueV0{Field: "app_spec.spec_id"}
	}
	if strings.TrimSpace(spec.Validation.Estado) != "valida" {
		return AppPlannerIssueV0{Field: "app_spec.validation.estado"}
	}
	if strings.TrimSpace(spec.App.Nombre) == "" {
		return AppPlannerIssueV0{Field: "app_spec.app.nombre"}
	}
	if api, web := appSpecSurfacesV0(spec); !api && !web {
		return AppPlannerIssueV0{Field: "app_spec.app.tipo_app"}
	}
	return nil
}

func appSpecSurfacesV0(spec orquestafactory.AppSpecV0) (bool, bool) {
	switch strings.TrimSpace(spec.App.TipoApp) {
	case "api":
		return true, false
	case "web":
		return false, true
	case "mixed":
		return true, true
	default:
		return false, false
	}
}

func firstAppPlanValueV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
