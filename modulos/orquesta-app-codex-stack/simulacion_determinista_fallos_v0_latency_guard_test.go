//go:build !race

package orquestaappcodexstack

import "time"

func simulacionDeterministaFallosElapsedLimitV0() time.Duration {
	return 60 * time.Second
}
