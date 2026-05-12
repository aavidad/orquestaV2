package orquestastatefile

import orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"

func invalidErrorV0(field string, message string) error {
	return orquestacionnucleoapp.ErrorV0{
		Code:    orquestacionnucleoapp.ErrNucleoOrquestacionInvalidoV0,
		Field:   field,
		Message: message,
	}
}

func storeErrorV0(field string, message string) error {
	return orquestacionnucleoapp.ErrorV0{
		Code:    orquestacionnucleoapp.ErrNucleoOrquestacionStoreV0,
		Field:   field,
		Message: message,
	}
}
