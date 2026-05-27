package orquestafactory

import "time"

const AppSpecReceptionTimeRequiredReasonV0 = "reloj_recepcion_utc_requerido"

func ValidateAppSpecReceptionTimeV0(now time.Time) []ValidationIssue {
	if now.IsZero() {
		return []ValidationIssue{
			issue(ErrAppSpecInvalida, "received_at", AppSpecReceptionTimeRequiredReasonV0),
		}
	}
	return nil
}
