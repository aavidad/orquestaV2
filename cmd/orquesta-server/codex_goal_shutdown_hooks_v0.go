package main

import (
	"reflect"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

func serverGoalShutdownHooksFromBackendsV0(
	appGoal serverCodexGoalBackendV0,
	idleGoal serverCodexGoalBackendV0,
) []orquestaserver.RuntimeShutdownHookPortV0 {
	hooks := make([]orquestaserver.RuntimeShutdownHookPortV0, 0, 2)
	causalIdentities := map[string]struct{}{}
	for _, hook := range []orquestaserver.RuntimeShutdownHookPortV0{
		appGoal.ShutdownHook,
		idleGoal.ShutdownHook,
	} {
		if hook == nil || serverGoalShutdownHookAlreadyRegisteredV0(hooks, causalIdentities, hook) {
			continue
		}
		hooks = append(hooks, hook)
		if identity := serverGoalShutdownHookCausalIdentityV0(hook); identity != "" {
			causalIdentities[identity] = struct{}{}
		}
	}
	return hooks
}

func serverGoalShutdownHookAlreadyRegisteredV0(
	hooks []orquestaserver.RuntimeShutdownHookPortV0,
	causalIdentities map[string]struct{},
	hook orquestaserver.RuntimeShutdownHookPortV0,
) bool {
	if identity := serverGoalShutdownHookCausalIdentityV0(hook); identity != "" {
		_, exists := causalIdentities[identity]
		return exists
	}
	hookType := reflect.TypeOf(hook)
	if hookType == nil || !hookType.Comparable() {
		return false
	}
	for _, existing := range hooks {
		if reflect.TypeOf(existing) == hookType && existing == hook {
			return true
		}
	}
	return false
}

func serverGoalShutdownHookCausalIdentityV0(hook orquestaserver.RuntimeShutdownHookPortV0) string {
	identity, ok := hook.(orquestaservershutdown.ActiveShutdownWorkIdentityPortV0)
	if !ok {
		return ""
	}
	return strings.TrimSpace(identity.ActiveShutdownWorkIdentityV0())
}
