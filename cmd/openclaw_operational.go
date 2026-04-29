package cmd

import "time"

var openClawOperationalInfoTimeout = 150 * time.Millisecond

func buildOpenClawOperationalInfo() serverOperationalInfo {
	return buildOpenClawOperationalInfoWithStatus(apiStatusResponse{})
}

func buildOpenClawOperationalInfoWithStatus(status apiStatusResponse) serverOperationalInfo {
	fallback := normalizeServerOperationalInfo(buildServerOperationalInfo(status))
	if controlPlaneOperationalInfoFetcher == nil {
		if fallback.Generated == "" {
			return degradedServerOperationalInfo()
		}
		return fallback
	}
	info, err := runAPITimeboxed(openClawOperationalInfoTimeout, controlPlaneOperationalInfoFetcher, errStatusFetchTimeout)
	if err != nil {
		if fallback.Generated == "" {
			return degradedServerOperationalInfo()
		}
		return fallback
	}
	return normalizeServerOperationalInfo(info)
}

func buildOpenClawOperationalSummary(info serverOperationalInfo) string {
	info = normalizeServerOperationalInfo(info)
	return formatServerOperationalSummary(&info)
}
