package firecrackerlauncher

import (
	"context"
	"os"
)

type RunResult struct {
	CapturedOutputBytes uint64
	AssetDigest         string
}

// Runner is the physical-launch boundary. Implementations must obey Run's
// context, terminate and reap every child within the configured cleanup
// timeout, and make Close return only after no owned process remains.
// unavailableRunner deliberately does not accredit that physical contract.
type Runner interface {
	Run(context.Context, LaunchRequest, *os.File, *os.File) (RunResult, error)
	Close() error
}
