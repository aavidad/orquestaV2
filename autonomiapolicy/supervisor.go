package autonomiapolicy

import "strings"

func SupervisorRoleScore(role string) int {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "supervisor", "orquestador":
		return 0
	case "admin":
		return 1
	case "programador":
		return 2
	case "revisor", "reviewer":
		return 3
	default:
		return 4
	}
}
