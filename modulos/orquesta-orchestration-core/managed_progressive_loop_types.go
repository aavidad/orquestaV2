package orquestacionnucleoapp

import "context"

type ExternalProgressWaiterPortV0 interface {
	WaitExternalProgressV0(
		context.Context,
		ExternalProgressWaitRequestV0,
	) (ExternalProgressWaitResultV0, error)
}

type ExternalProgressWaitRequestV0 struct {
	RunRef        string
	WaitNumber    int
	LastResult    ProgressiveLoopResultV0
	CorrelationID string
	EvidenceRefs  []string
	WaitAgentRefs []string
}

type ExternalProgressWaitResultV0 struct {
	Continue     bool
	EvidenceRefs []string
}

type ManagedProgressiveLoopRequestV0 struct {
	Loop             ProgressiveLoopRequestV0
	ExternalWaiter   ExternalProgressWaiterPortV0
	MaxExternalWaits int
}

type ManagedProgressiveLoopAttemptV0 struct {
	AttemptNumber int
	Result        ProgressiveLoopResultV0
}

type ManagedProgressiveLoopExternalWaitV0 struct {
	WaitNumber   int
	Continue     bool
	EvidenceRefs []string
}

type ManagedProgressiveLoopResultV0 struct {
	Status        ProgressiveLoopStatusV0
	Final         ProgressiveLoopResultV0
	Attempts      []ManagedProgressiveLoopAttemptV0
	ExternalWaits []ManagedProgressiveLoopExternalWaitV0
}
