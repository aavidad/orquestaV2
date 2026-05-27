package main

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

func codexWaveStopAttemptRefV0(
	summary codexWaveLaunchSummaryV0,
	agent codexWaveAgentSummaryV0,
	config codexWaveControlConfigV0,
	now time.Time,
) string {
	parts := []string{
		"codex-wave-stop-attempt-v0",
		strings.TrimSpace(summary.WaveRef),
		strings.TrimSpace(agent.AgentRef),
		strings.TrimSpace(config.Reason),
		now.UTC().Format(time.RFC3339Nano),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return "shutdown-attempt-ref-" +
		safeCodexWavePurgeRefPartV0(summary.WaveRef) + "-" +
		safeCodexWavePurgeRefPartV0(agent.AgentRef) + "-" +
		hex.EncodeToString(sum[:])[:16]
}
