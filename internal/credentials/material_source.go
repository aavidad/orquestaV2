package credentials

import "context"

// MaterialSource provides ephemeral credential material to one callback. The
// material is never returned by the port and implementations must destroy it
// when the callback finishes, including on failure or cancellation.
type MaterialSource interface {
	WithSecret(context.Context, func(Secret) error) error
}
