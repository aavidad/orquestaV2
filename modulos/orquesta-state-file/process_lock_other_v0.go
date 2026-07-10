//go:build !linux

package orquestastatefile

import "context"

func withProcessFileLockV0(ctx context.Context, _ string, fn func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return fn()
}
