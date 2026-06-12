package main

import (
	"context"
	"strings"
	"time"
)

const defaultExternalBridgeShutdownTimeoutV0 = 5 * time.Second

func externalBridgeResultCountersV0(result any) map[string]int {
	switch value := result.(type) {
	case opesDrainSummaryV0:
		return map[string]int{
			"seen":              value.Seen,
			"submitted":         value.Submitted,
			"already_submitted": value.AlreadySubmitted,
			"claimed":           value.Claimed,
			"skipped":           value.Skipped,
			"errors":            len(value.Errors),
		}
	case opesRegistryFinalPkgSummaryV0:
		return map[string]int{
			"seen":                  value.Seen,
			"submitted":             value.Submitted,
			"skipped":               value.Skipped,
			"reconciled":            value.Reconciled,
			"reconcile_skipped":     value.ReconcileSkipped,
			"errors":                len(value.Errors),
			"completed_nonterminal": value.CompletedNonTerminal,
		}
	default:
		return nil
	}
}

func externalBridgeResultEvidenceRefsV0(result any) []string {
	switch value := result.(type) {
	case opesRegistryFinalPkgSummaryV0:
		refs := make([]string, 0, len(value.CompletedNonTerminalRefs))
		for _, drift := range value.CompletedNonTerminalRefs {
			refs = append(refs, drift.RunRef)
		}
		return compactExternalBridgeStringsV0(refs)
	default:
		return nil
	}
}

func compactExternalBridgeErrorCodeV0(err error) string {
	if err == nil {
		return ""
	}
	value := strings.TrimSpace(err.Error())
	if value == "" {
		return "external_bridge_tick_error"
	}
	value = strings.ToLower(value)
	for _, separator := range []string{":", "\n", "\r", "/", "\\", " "} {
		if index := strings.Index(value, separator); index >= 0 {
			value = value[:index]
		}
	}
	value = compactExternalBridgeTokenV0(value)
	if value == "" {
		return "external_bridge_tick_error"
	}
	return value
}

func firstExternalBridgeValueV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func compactExternalBridgeStringsV0(values []string) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func compactExternalBridgeTokenV0(value string) string {
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func externalBridgeLoopShutdownTimeoutV0(config externalBridgeLoopConfigV0) time.Duration {
	if config.EffectTimeout > 0 {
		return config.EffectTimeout
	}
	return defaultExternalBridgeShutdownTimeoutV0
}

func externalBridgeLoopContextV0(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
