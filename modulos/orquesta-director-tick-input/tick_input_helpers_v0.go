package orquestadirectortickinput

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func compactTickInputRefsV0(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func projectionPrefixRefsV0(refs []string, separator string) []string {
	if len(refs) == 0 {
		return nil
	}
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		prefix, _, ok := strings.Cut(strings.TrimSpace(ref), separator)
		if ok {
			out = append(out, boundedProjectionRefV0(prefix))
			continue
		}
		out = append(out, boundedProjectionRefV0(ref))
	}
	return compactTickInputRefsV0(out)
}

func boundedProjectionRefV0(ref string) string {
	ref = strings.TrimSpace(ref)
	if len(ref) <= maxTickInputStringV0 {
		return ref
	}
	sum := sha256.Sum256([]byte(ref))
	return "projection-ref-" + hex.EncodeToString(sum[:])[:32]
}

func cloneLeaseCandidatesV0(
	candidates []orquestadirectorscheduler.SchedulableLeaseActionCandidateV0,
) []orquestadirectorscheduler.SchedulableLeaseActionCandidateV0 {
	if len(candidates) == 0 {
		return nil
	}
	cloned := make([]orquestadirectorscheduler.SchedulableLeaseActionCandidateV0, len(candidates))
	copy(cloned, candidates)
	return cloned
}

func cloneProgressCandidatesV0(
	candidates []orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0,
) []orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0 {
	if len(candidates) == 0 {
		return nil
	}
	cloned := make([]orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0, len(candidates))
	copy(cloned, candidates)
	return cloned
}

func clonePhaseArtifactCandidatesV0(
	candidates []orquestadirectorscheduler.SchedulablePhaseArtifactCandidateV0,
) []orquestadirectorscheduler.SchedulablePhaseArtifactCandidateV0 {
	if len(candidates) == 0 {
		return nil
	}
	cloned := make([]orquestadirectorscheduler.SchedulablePhaseArtifactCandidateV0, len(candidates))
	copy(cloned, candidates)
	return cloned
}

func cloneDeliveryCandidatesV0(
	candidates []orquestadirectorscheduler.SchedulableDeliveryCandidateV0,
) []orquestadirectorscheduler.SchedulableDeliveryCandidateV0 {
	if len(candidates) == 0 {
		return nil
	}
	cloned := make([]orquestadirectorscheduler.SchedulableDeliveryCandidateV0, len(candidates))
	copy(cloned, candidates)
	return cloned
}

func cloneReviewGateCandidatesV0(
	candidates []orquestadirectorscheduler.SchedulableReviewGateCandidateV0,
) []orquestadirectorscheduler.SchedulableReviewGateCandidateV0 {
	if len(candidates) == 0 {
		return nil
	}
	cloned := make([]orquestadirectorscheduler.SchedulableReviewGateCandidateV0, len(candidates))
	copy(cloned, candidates)
	return cloned
}

func cloneReplanCandidatesV0(
	candidates []orquestadirectorscheduler.SchedulableReplanFollowupCandidateV0,
) []orquestadirectorscheduler.SchedulableReplanFollowupCandidateV0 {
	if len(candidates) == 0 {
		return nil
	}
	cloned := make([]orquestadirectorscheduler.SchedulableReplanFollowupCandidateV0, len(candidates))
	copy(cloned, candidates)
	return cloned
}

func cloneWorkCandidatesV0(
	candidates []orquestadirectorscheduler.SchedulableWorkCandidateV0,
) []orquestadirectorscheduler.SchedulableWorkCandidateV0 {
	if len(candidates) == 0 {
		return nil
	}
	cloned := make([]orquestadirectorscheduler.SchedulableWorkCandidateV0, len(candidates))
	copy(cloned, candidates)
	return cloned
}

func cloneWorkClaimsV0(
	claims []orquestacoreconcurrency.WorksetClaimV0,
) []orquestacoreconcurrency.WorksetClaimV0 {
	if len(claims) == 0 {
		return nil
	}
	cloned := make([]orquestacoreconcurrency.WorksetClaimV0, len(claims))
	copy(cloned, claims)
	return cloned
}
