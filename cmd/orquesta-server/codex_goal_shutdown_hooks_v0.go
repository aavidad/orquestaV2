package main

import (
	"reflect"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func serverGoalShutdownHooksFromBackendsV0(
	appGoal serverCodexGoalBackendV0,
	idleGoal serverCodexGoalBackendV0,
) []orquestaserver.RuntimeShutdownHookPortV0 {
	hooks := make([]orquestaserver.RuntimeShutdownHookPortV0, 0, 2)
	for _, hook := range []orquestaserver.RuntimeShutdownHookPortV0{
		appGoal.ShutdownHook,
		idleGoal.ShutdownHook,
	} {
		if hook == nil || serverGoalShutdownHookAlreadyRegisteredV0(hooks, hook) {
			continue
		}
		hooks = append(hooks, hook)
	}
	return hooks
}

func serverGoalShutdownHookAlreadyRegisteredV0(
	hooks []orquestaserver.RuntimeShutdownHookPortV0,
	hook orquestaserver.RuntimeShutdownHookPortV0,
) bool {
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
