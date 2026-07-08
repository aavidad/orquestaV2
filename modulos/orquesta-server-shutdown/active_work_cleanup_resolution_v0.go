package orquestaservershutdown

import "strings"

func activeShutdownWorksCompletedAfterCleanupV0(
	cleanupTargetWorks []ActiveShutdownWorkV0,
	remainingWorks []ActiveShutdownWorkV0,
	cleaned ActiveShutdownWorkCleanupResultV0,
) []ActiveShutdownWorkV0 {
	if cleaned.CleanedWorkCount <= 0 {
		return []ActiveShutdownWorkV0{}
	}
	targets := compactActiveShutdownWorksV0(cleanupTargetWorks)
	remaining := compactActiveShutdownWorksV0(remainingWorks)
	remainingByIdentity := map[string]struct{}{}
	for _, work := range remaining {
		key := activeShutdownWorkIdentityKeyV0(work)
		if key == "" {
			continue
		}
		remainingByIdentity[key] = struct{}{}
	}
	out := make([]ActiveShutdownWorkV0, 0, len(targets))
	for _, work := range targets {
		key := activeShutdownWorkIdentityKeyV0(work)
		if key == "" {
			continue
		}
		if _, stillPresent := remainingByIdentity[key]; stillPresent {
			continue
		}
		out = append(out, work)
	}
	if out == nil {
		return []ActiveShutdownWorkV0{}
	}
	return out
}

func activeShutdownWorkIdentityKeyV0(
	work ActiveShutdownWorkV0,
) string {
	work = normalizeActiveShutdownWorkV0(work)
	if work.Kind == "" && work.RunRef == "" && work.WorkRef == "" && work.ExternalWorkRef == "" {
		return ""
	}
	return strings.Join([]string{work.Kind, work.RunRef, work.WorkRef, work.ExternalWorkRef}, "\x00")
}
