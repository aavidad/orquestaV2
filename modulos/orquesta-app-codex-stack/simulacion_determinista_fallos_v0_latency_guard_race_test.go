//go:build race

package orquestaappcodexstack

import "time"

func simulacionDeterministaFallosElapsedLimitV0() time.Duration {
	// The race detector adds instrumentation overhead; simulation assertions remain active.
	return 0
}
