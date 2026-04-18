package db

import "strings"

func NormalizeRuntimeWorkerState(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "starting", "booting", "launching", "arrancando":
		return "starting"
	case "working", "running", "active", "busy":
		return "working"
	case "waiting_input", "waiting", "paused", "awaiting_input":
		return "waiting_input"
	case "blocked_quota", "quota", "quota_exceeded", "rate_limited", "rate-limited", "sin_cuota", "blocked_auth", "auth_required":
		return "blocked_quota"
	case "blocked_runtime", "blocked", "failed", "error", "crashed":
		return "blocked_runtime"
	case "stuck", "stale", "hung":
		return "stuck"
	case "stopped", "completed", "cancelled", "canceled", "superseded", "closed", "archived", "finished":
		return "stopped"
	case "ready":
		return "idle"
	default:
		return "idle"
	}
}
