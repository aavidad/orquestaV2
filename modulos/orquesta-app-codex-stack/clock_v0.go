package orquestaappcodexstack

import (
	"time"

	orquestafactoryhttp "orquesta/modulos/orquesta-factory-http"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func stackNowV0(clock orquestafactoryhttp.AppSpecHTTPClockV0) time.Time {
	if clock == nil {
		return orquestaruntime.NowUTCV0(nil)
	}
	return orquestaruntime.NowUTCV0(orquestaruntime.ClockFuncV0(clock))
}
