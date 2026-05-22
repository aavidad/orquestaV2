package orquestaappcodexstack

import (
	"time"

	orquestafactoryhttp "orquesta/modulos/orquesta-factory-http"
)

func stackNowV0(clock orquestafactoryhttp.AppSpecHTTPClockV0) time.Time {
	if clock == nil {
		return time.Now().UTC()
	}
	now := clock().UTC()
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now
}
