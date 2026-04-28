package cmd

import "time"

const openClawOperationalInfoTimeout = 150 * time.Millisecond

func buildOpenClawOperationalInfo() serverOperationalInfo {
	if controlPlaneOperationalInfoFetcher == nil {
		return degradedServerOperationalInfo()
	}
	info, err := runAPITimeboxed(openClawOperationalInfoTimeout, controlPlaneOperationalInfoFetcher, errStatusFetchTimeout)
	if err != nil {
		return degradedServerOperationalInfo()
	}
	return normalizeServerOperationalInfo(info)
}

func buildOpenClawOperationalSummary(info serverOperationalInfo) string {
	info = normalizeServerOperationalInfo(info)
	return formatServerOperationalSummary(&info)
}
