package orquestafactory

import "time"

const AppSpecSchemaV0 = "app_spec.v0"

func SolicitarNuevaAppV0(req AppSpecRequestV0, now time.Time) (AppSpecV0, []ValidationIssue) {
	if issues := ValidateAppSpecRequestV0(req); len(issues) > 0 {
		return AppSpecV0{}, issues
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return assembleAppSpecV0(req, now), nil
}
