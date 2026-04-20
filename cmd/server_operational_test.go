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
		AutonomySupervising: 1,
		AutonomyContinuing:  2,
		AutonomyPending:     3,
		AutonomyConfirmed:   4,
		AutonomyHandoffs:    5,
	})
	if !strings.Contains(summary, "dispatch p:1 n:2 f:3 c:4") {
		t.Fatalf("summary sin dispatch confirmado: %s", summary)
	}
	if !strings.Contains(summary, "autonomia s:1 c:2 p:3 ok:4 h:5") {
		t.Fatalf("summary sin resumen de autonomia: %s", summary)
	}
}
