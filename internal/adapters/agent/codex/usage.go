package codex

import "orquesta/internal/governance"

// Codex CLI currently exposes no trustworthy token, money, active-time, slot,
// or artifact-storage accounting. Unknown is explicit; zero is never invented.
func unknownCodexUsage() governance.ResourceUsage {
	return governance.ResourceUsage{Quality: governance.UsageQualityUnknown}
}
