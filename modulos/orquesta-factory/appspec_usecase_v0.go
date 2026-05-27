package orquestafactory

import "time"

const AppSpecSchemaV0 = "app_spec.v0"

func SolicitarNuevaAppV0(req AppSpecRequestV0, now time.Time) (AppSpecV0, []ValidationIssue) {
	if issues := ValidateAppSpecRequestV0(req); len(issues) > 0 {
		return AppSpecV0{}, issues
	}
	if issues := ValidateAppSpecReceptionTimeV0(now); len(issues) > 0 {
		return AppSpecV0{}, issues
	}
	return assembleAppSpecV0(req, now), nil
}
