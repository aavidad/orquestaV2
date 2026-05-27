package main

import "strings"

func idleSelfImprovementFederatedBacklogSourceRunnableV0(source idleSelfImprovementFederatedBacklogSourceV0) bool {
	state := strings.ToLower(strings.TrimSpace(source.State))
	return state == "vigente" || state == "promocionada" || state == "promoted" || state == "active"
}
