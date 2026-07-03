package orquestaserver

import (
	"sort"
	"strings"
)

const idleSelfImprovementSkippedIdenticalCounterV0 = "skipped_identical"

type idleSelfImprovementPublicationV0 struct {
	Status      string
	Reason      string
	RequestRefs []string
}

func newIdleSelfImprovementPublicationV0(
	status string,
	reason string,
	requestRefs []string,
) idleSelfImprovementPublicationV0 {
	refs := compactConfigStringsV0(requestRefs)
	sort.Strings(refs)
	return idleSelfImprovementPublicationV0{
		Status:      compactServerOperationalTokenV0(status),
		Reason:      compactServerOperationalTokenV0(reason),
		RequestRefs: refs,
	}
}

func (publication idleSelfImprovementPublicationV0) activeV0() bool {
	switch strings.TrimSpace(publication.Status) {
	case "blocked", "scheduled":
		return len(publication.RequestRefs) > 0
	default:
		return false
	}
}

func (tracker *StatusTrackerV0) RegisterIdleSelfImprovementPublicationV0(
	publication idleSelfImprovementPublicationV0,
) (bool, int) {
	publication = newIdleSelfImprovementPublicationV0(
		publication.Status,
		publication.Reason,
		publication.RequestRefs,
	)
	if !publication.activeV0() {
		return false, 0
	}
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if tracker.idleSelfImprovementPublicationRecorded &&
		idleSelfImprovementPublicationEqualV0(
			tracker.lastIdleSelfImprovementPublication,
			publication,
		) {
		tracker.idleSelfImprovementSkippedIdentical++
		return true, tracker.idleSelfImprovementSkippedIdentical
	}
	skipped := tracker.idleSelfImprovementSkippedIdentical
	tracker.lastIdleSelfImprovementPublication = publication
	tracker.idleSelfImprovementPublicationRecorded = true
	tracker.idleSelfImprovementSkippedIdentical = 0
	return false, skipped
}

func (tracker *StatusTrackerV0) resetIdleSelfImprovementPublicationLockedV0() {
	tracker.lastIdleSelfImprovementPublication = idleSelfImprovementPublicationV0{}
	tracker.idleSelfImprovementPublicationRecorded = false
	tracker.idleSelfImprovementSkippedIdentical = 0
}

func (tracker *StatusTrackerV0) ConsumeIdleSelfImprovementSkippedIdenticalV0() int {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	skipped := tracker.idleSelfImprovementSkippedIdentical
	tracker.idleSelfImprovementSkippedIdentical = 0
	return skipped
}

func idleSelfImprovementPublicationEqualV0(
	left idleSelfImprovementPublicationV0,
	right idleSelfImprovementPublicationV0,
) bool {
	if left.Status != right.Status || left.Reason != right.Reason {
		return false
	}
	if len(left.RequestRefs) != len(right.RequestRefs) {
		return false
	}
	for index := range left.RequestRefs {
		if left.RequestRefs[index] != right.RequestRefs[index] {
			return false
		}
	}
	return true
}

func idleSelfImprovementAuditPayloadWithSkippedIdenticalV0(
	payload map[string]interface{},
	skipped int,
) map[string]interface{} {
	if payload == nil {
		payload = map[string]interface{}{}
	}
	if skipped > 0 {
		payload[idleSelfImprovementSkippedIdenticalCounterV0] = skipped
	}
	return payload
}
