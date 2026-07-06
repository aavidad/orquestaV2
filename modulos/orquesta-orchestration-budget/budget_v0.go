package orquestaorchestrationbudget

const (
	RunEventReadPageLimitV0  = 250
	RunEventStorePageLimitV0 = 1000
	RunEventReadMaxV0        = 10000
	RunEventStoreMaxEventsV0 = 20000

	StateFileEventPayloadMaxBytesV0        = 256 * 1024
	SchedulerTickSnapshotBudgetBytesV0     = 64 * 1024
	SchedulerTickPayloadMaxBytesV0         = 256 * 1024
	RunEventStoreDefaultPageLimitV0        = 100
	RunEventStoreDefaultAppendEventLimitV0 = 256
)
