//go:build !linux

package filesource

import (
	"context"

	"orquesta/internal/credentials"
)

// WithSecret fails closed where openat2 and Linux file identity are unavailable.
func (source *Source) WithSecret(ctx context.Context, callback func(credentials.Secret) error) error {
	if source == nil || nilContext(ctx) || callback == nil {
		return credentials.NewError(credentials.ErrorInvalidRequest, "material_source")
	}
	if err := ctx.Err(); err != nil {
		return credentials.WrapError(credentials.ErrorConsumerFailed, "context", err)
	}
	return credentials.NewError(credentials.ErrorUnsafeFile, "material_source_platform")
}
