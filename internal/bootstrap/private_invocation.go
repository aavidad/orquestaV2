package bootstrap

import "orquesta/internal/adapters/agent/codex"

// DispatchPrivateInvocation runs a descriptor-authenticated subprocess mode
// composed into the production binary. Public CLI arguments are not handled.
func DispatchPrivateInvocation(arguments []string) (exitCode int, handled bool) {
	if !isPrivateInvocation(arguments) {
		return 0, false
	}
	return codex.RunLocalSupervisor(), true
}

func isPrivateInvocation(arguments []string) bool {
	return codex.IsLocalSupervisorInvocation(arguments)
}
