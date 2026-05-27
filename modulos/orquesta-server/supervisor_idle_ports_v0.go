package orquestaserver

import "context"

const (
	idleSelfImprovementTriggerIdleV0         = "idle_self_improvement"
	idleSelfImprovementTriggerCapacityFreeV0 = "capacity_free"
)

type IdleSelfImprovementRunFreshnessPortV0 interface {
	RetryableIdleSelfImprovementRunRefsV0(
		context.Context,
		IdleSelfImprovementRunFreshnessRequestV0,
	) (IdleSelfImprovementRunFreshnessResultV0, error)
}

type IdleSelfImprovementRunFreshnessRequestV0 struct {
	KnownRunRefs     []string
	KnownRequestRefs []string
}

type IdleSelfImprovementRunFreshnessResultV0 struct {
	RetryableRunRefs     []string
	RetryableRequestRefs []string
	EvidenceRefs         []string
}
