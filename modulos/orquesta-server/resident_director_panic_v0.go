package orquestaserver

import (
	"context"
	"fmt"
)

func (runtime *RuntimeV0) recoverResidentDirectorTickPanicV0(ctx context.Context) {
	recovered := recover()
	if recovered == nil {
		return
	}
	result := ResidentDirectorResultV0{Status: "panic"}
	message := fmt.Sprintf("panic:%v", recovered)
	runtime.auditEventV0(ctx, "resident_director_tick_panic", "error", message, map[string]interface{}{
		"result_summary": residentDirectorResultAuditSummaryV0(result),
	})
	runtime.persistStateTransitionV0(
		ctx,
		runtime.tracker.MarkResidentDirectorErrorV0(result, message, runtime.clock.Now()),
		"resident_director_panic",
	)
}
