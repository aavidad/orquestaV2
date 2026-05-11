package orquestaappplanner

import (
	"strings"
	"unicode"
)

func taskRefV0(request AppPlanRequestV0, key string) string {
	return "task-" + safeRefPartV0(request.AppRef) + "-" + safeRefPartV0(key)
}

func claimRefV0(request AppPlanRequestV0, key string) string {
	return "claim-" + safeRefPartV0(request.AppRef) + "-" + safeRefPartV0(key)
}

func agentRefV0(request AppPlanRequestV0, key string) string {
	return "agent-" + safeRefPartV0(request.AppRef) + "-" + safeRefPartV0(key)
}

func deliveryRefV0(request AppPlanRequestV0, key string) string {
	return "ack-" + safeRefPartV0(request.AppRef) + "-" + safeRefPartV0(key)
}

func capacityRefV0(unit AppWorkUnitV0) string {
	return "capacity-" + strings.TrimPrefix(unit.TaskRef, "task-")
}

func commandRefV0(unit AppWorkUnitV0, kind string) string {
	return "cmd-" + safeRefPartV0(kind) + "-" + strings.TrimPrefix(unit.TaskRef, "task-")
}

func idempotencyRefV0(unit AppWorkUnitV0, kind string) string {
	return "idem-" + safeRefPartV0(kind) + "-" + strings.TrimPrefix(unit.TaskRef, "task-")
}

func safeRefPartV0(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "ref"
	}
	return out
}
