package orquestastatefile

import "context"

func contextOrBackgroundV0(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
