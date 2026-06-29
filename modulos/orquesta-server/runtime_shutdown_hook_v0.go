package orquestaserver

import "context"

type RuntimeShutdownHookPortV0 interface {
	ShutdownV0(context.Context) error
}

func compactRuntimeShutdownHooksV0(hooks []RuntimeShutdownHookPortV0) []RuntimeShutdownHookPortV0 {
	if len(hooks) == 0 {
		return nil
	}
	out := make([]RuntimeShutdownHookPortV0, 0, len(hooks))
	for _, hook := range hooks {
		if hook != nil {
			out = append(out, hook)
		}
	}
	return out
}
