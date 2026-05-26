package orquestaweb

import "context"

func webContextOrBackgroundV0(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
