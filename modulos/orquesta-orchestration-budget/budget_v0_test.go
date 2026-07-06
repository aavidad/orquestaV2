package orquestaorchestrationbudget

import "testing"

func TestOrchestrationBudgetV0CoherenciaEventosYPayloadScheduler(t *testing.T) {
	if RunEventReadPageLimitV0 <= 0 {
		t.Fatalf("RunEventReadPageLimitV0=%d, want positive", RunEventReadPageLimitV0)
	}
	if RunEventStorePageLimitV0 <= 0 {
		t.Fatalf("RunEventStorePageLimitV0=%d, want positive", RunEventStorePageLimitV0)
	}
	if RunEventReadPageLimitV0 > RunEventReadMaxV0 {
		t.Fatalf("read page=%d > read max=%d", RunEventReadPageLimitV0, RunEventReadMaxV0)
	}
	if RunEventStorePageLimitV0 > RunEventReadMaxV0 {
		t.Fatalf("store page=%d > read max=%d", RunEventStorePageLimitV0, RunEventReadMaxV0)
	}
	if RunEventReadMaxV0 > RunEventStoreMaxEventsV0 {
		t.Fatalf("read max=%d > store max=%d", RunEventReadMaxV0, RunEventStoreMaxEventsV0)
	}
	if SchedulerTickSnapshotBudgetBytesV0 > SchedulerTickPayloadMaxBytesV0 {
		t.Fatalf("snapshot budget=%d > scheduler payload=%d", SchedulerTickSnapshotBudgetBytesV0, SchedulerTickPayloadMaxBytesV0)
	}
	if SchedulerTickPayloadMaxBytesV0 != 256*1024 {
		t.Fatalf("scheduler payload=%d, want 256KiB", SchedulerTickPayloadMaxBytesV0)
	}
}
