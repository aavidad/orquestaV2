package orquestacore

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

func stableBacklogItemIDV0(index int, sourceID string) string {
	sourceID = strings.TrimSpace(sourceID)
	if sourceID != "" {
		return "core-" + strings.ToLower(sourceID)
	}
	return stableIDV0("backlog", strconv.Itoa(index+1))
}

func stableIDV0(prefix, value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return prefix + "_" + hex.EncodeToString(sum[:])[:16]
}

func occurredAtV0(cmd RegistrarProyectoDesdeAppSpecCommandV0) string {
	if value := strings.TrimSpace(cmd.SolicitadoEn); value != "" {
		return value
	}
	if value := strings.TrimSpace(cmd.AppSpec.CreatedAt); value != "" {
		return value
	}
	return time.Time{}.UTC().Format(time.RFC3339)
}

func nonEmptyNotEqualV0(value, want string) bool {
	value = strings.TrimSpace(value)
	return value != "" && value != want
}

func firstNonEmptyV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func compactStringsV0(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		result = append(result, trimmed)
	}
	return result
}
