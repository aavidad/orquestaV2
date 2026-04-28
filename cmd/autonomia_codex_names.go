package cmd

import "strings"

func canonicalAutonomyCodexName(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || !perteneceAFamiliaCodexAutobootstrap(raw) {
		return raw
	}
	if len(raw) <= len("codex") {
		return "Codex"
	}
	return "Codex" + strings.TrimSpace(raw[len("codex"):])
}

func canonicalAutonomyCodexNames(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	out := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, item := range raw {
		item = canonicalAutonomyCodexName(item)
		if item == "" {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(item))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}
