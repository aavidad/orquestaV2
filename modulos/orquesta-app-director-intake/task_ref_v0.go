package orquestaappdirectorintake

import orquestafactory "orquesta/modulos/orquesta-factory"

func appDirectorTaskRefPrefixV0(spec orquestafactory.AppSpecV0) string {
	return safeDirectorIntakeRefPartV0(firstDirectorIntakeValueV0(spec.SpecID, spec.App.Slug))
}
