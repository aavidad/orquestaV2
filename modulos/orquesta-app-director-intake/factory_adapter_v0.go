package orquestaappdirectorintake

import orquestafactory "orquesta/modulos/orquesta-factory"

func AppDirectorInputSpecFromFactoryV0(spec orquestafactory.AppSpecV0) AppDirectorInputSpecV0 {
	return AppDirectorInputSpecV0{
		SchemaVersion: AppDirectorInputSpecSchemaVersionV0,
		SpecID:        spec.SpecID,
		CreatedAt:     spec.CreatedAt,
		App: AppDirectorInputAppV0{
			Nombre:      spec.App.Nombre,
			Objetivo:    spec.App.Objetivo,
			Descripcion: spec.App.Descripcion,
			TipoApp:     spec.App.TipoApp,
			Slug:        spec.App.Slug,
		},
		Platforms: append([]string(nil), spec.Platforms...),
		Data: AppDirectorInputDataV0{
			Needs:               append([]string(nil), spec.Data.Needs...),
			PersistenceRequired: spec.Data.PersistenceRequired,
		},
		I18N: AppDirectorInputI18NV0{
			Enabled: spec.I18N.Enabled,
		},
		Quality: AppDirectorInputQualityV0{
			Tests: spec.Quality.Tests,
		},
		AgentPreferences: AppDirectorInputAgentPreferencesV0{
			Autonomy: spec.AgentPreferences.Autonomy,
		},
		RequestKind:   orquestafactory.NormalizeRequestKindV0(spec.RequestKind),
		ExecutionMode: orquestafactory.NormalizeExecutionModeV0(spec.ExecutionMode),
		Validation: AppDirectorInputValidationV0{
			Estado: spec.Validation.Estado,
		},
	}
}
