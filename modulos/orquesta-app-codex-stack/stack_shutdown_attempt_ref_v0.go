package orquestaappcodexstack

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"time"

	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

func stackShutdownAttemptRefV0(
	command orquestaservershutdown.PrepareAgentShutdownCommandV0,
) string {
	parts := []string{
		"codex-shutdown-attempt-v0",
		strings.TrimSpace(command.RunRef),
		strings.TrimSpace(command.AppRef),
		strings.TrimSpace(command.CorrelationID),
		strings.TrimSpace(command.RequestedBy),
		strings.TrimSpace(command.Reason),
		stackShutdownAttemptOccurredAtV0(command.OccurredAt),
	}
	refs := append([]string(nil), compactStringsV0(command.EvidenceRefs)...)
	sort.Strings(refs)
	parts = append(parts, refs...)
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return "shutdown-attempt-ref-" +
		safeStackShutdownRefPartV0(command.RunRef) + "-" +
		hex.EncodeToString(sum[:])[:16]
}

func stackShutdownAttemptOccurredAtV0(occurredAt time.Time) string {
	if occurredAt.IsZero() {
		return "occurred-at-zero"
	}
	return occurredAt.UTC().Format(time.RFC3339Nano)
}
