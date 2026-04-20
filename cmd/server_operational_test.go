package cmd

import (
	"strings"
	"testing"
)

func TestFormatServerOperationalSummaryIncluyeDispatch(t *testing.T) {
	summary := formatServerOperationalSummary(&serverOperationalInfo{
		ActiveAgents:      2,
		RegisteredAgents:  4,
		DispatchPending:   1,
		DispatchNotified:  2,
		DispatchFailed:    3,
		DispatchConfirmed: 4,
	})
	if !strings.Contains(summary, "dispatch p:1 n:2 f:3 c:4") {
		t.Fatalf("summary sin dispatch confirmado: %s", summary)
	}
}
