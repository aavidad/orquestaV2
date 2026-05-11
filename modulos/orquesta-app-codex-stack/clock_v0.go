package orquestaappcodexstack

import (
	"time"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func stackNowV0(clock orquestafactory.AppSpecHTTPClockV0) time.Time {
	if clock == nil {
		return time.Now().UTC()
	}
	now := clock().UTC()
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now
}
