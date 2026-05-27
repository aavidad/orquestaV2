package orquestaserver

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	StartupCheckStatusReadyV0    = "startup_ready"
	StartupCheckStatusBlockedV0  = "startup_blocked"
	StartupCheckStatusCheckingV0 = "startup_checking"
)

type StartupCheckCommandV0 struct {
	ProjectWorkDir string
	RuntimeWorkDir string
	StateDir       string
	CorrelationID  string
	OccurredAt     time.Time
}

type StartupCheckResultV0 struct {
	Status          string                   `json:"status"`
	Ready           bool                     `json:"ready"`
	Message         string                   `json:"message,omitempty"`
	EvidenceRefs    []string                 `json:"evidence_refs,omitempty"`
	StartupRevision StartupRevisionSummaryV0 `json:"startup_revision,omitempty"`
}

type StartupRevisionSummaryV0 struct {
	RevisionRef     string `json:"revision_ref,omitempty"`
	QueueRemoved    int    `json:"queue_removed,omitempty"`
	QueueKept       int    `json:"queue_kept,omitempty"`
	ControlRemoved  int    `json:"control_removed,omitempty"`
	ControlKept     int    `json:"control_kept,omitempty"`
	RuntimeArchived int    `json:"runtime_archived,omitempty"`
	Artifacts       int    `json:"artifacts,omitempty"`
	RetentionDays   int    `json:"retention_days,omitempty"`
	MaxBytes        int64  `json:"max_bytes,omitempty"`
}

type StartupNotReadyErrorV0 struct {
	Status  string
	Message string
}

func (err StartupNotReadyErrorV0) Error() string {
	status := strings.TrimSpace(err.Status)
	if status == "" {
		status = StartupCheckStatusBlockedV0
	}
	message := strings.TrimSpace(err.Message)
	if message == "" {
		return "orquesta_server: " + status
	}
	return fmt.Sprintf("orquesta_server: %s: %s", status, message)
}

func (runtime *RuntimeV0) prepareStartupV0(ctx context.Context) error {
	if runtime.startupCheck == nil {
		result := StartupCheckResultV0{
			Status:  StartupCheckStatusReadyV0,
			Ready:   true,
			Message: "orquesta_startup_ready",
		}
		runtime.auditEventV0(ctx, "startup_check_skipped", "ready", "", map[string]interface{}{
			"result": result,
		})
		runtime.persistStateTransitionV0(ctx, runtime.tracker.MarkStartupReadyV0(
			result,
			runtime.clock.Now(),
		), "startup_ready")
		return nil
	}
	now := runtime.clock.Now()
	command := StartupCheckCommandV0{
		ProjectWorkDir: runtime.config.ProjectWorkDir,
		RuntimeWorkDir: runtime.config.RuntimeWorkDir,
		StateDir:       runtime.config.StateDir,
		CorrelationID:  "corr-orquesta-server-startup",
		OccurredAt:     now,
	}
	runtime.auditEventV0(ctx, "startup_check_start", "checking", "", map[string]interface{}{
		"command_summary": startupCheckCommandAuditSummaryV0(command),
	})
	runtime.persistStateTransitionV0(ctx, runtime.tracker.MarkStartupCheckingV0(now), "startup_checking")
	result, err := runtime.startupCheck.PrepareStartupV0(ctx, command)
	if err != nil {
		runtime.auditEventV0(ctx, "startup_check_error", "blocked", err.Error(), map[string]interface{}{
			"command_summary": startupCheckCommandAuditSummaryV0(command),
		})
		runtime.persistStateTransitionV0(ctx, runtime.tracker.MarkStartupBlockedV0(
			StartupCheckResultV0{
				Status:  StartupCheckStatusBlockedV0,
				Ready:   false,
				Message: err.Error(),
			},
			runtime.clock.Now(),
		), "startup_blocked")
		return err
	}
	result = normalizeStartupCheckResultV0(result)
	if !result.Ready {
		runtime.auditEventV0(ctx, "startup_check_blocked", "blocked", result.Message, map[string]interface{}{
			"command_summary": startupCheckCommandAuditSummaryV0(command),
			"result_summary":  startupCheckResultAuditSummaryV0(result),
		})
		runtime.persistStateTransitionV0(ctx, runtime.tracker.MarkStartupBlockedV0(result, runtime.clock.Now()), "startup_blocked")
		return StartupNotReadyErrorV0{Status: result.Status, Message: result.Message}
	}
	runtime.auditEventV0(ctx, "startup_check_ready", "ready", "", map[string]interface{}{
		"command_summary": startupCheckCommandAuditSummaryV0(command),
		"result_summary":  startupCheckResultAuditSummaryV0(result),
	})
	runtime.persistStateTransitionV0(ctx, runtime.tracker.MarkStartupReadyV0(result, runtime.clock.Now()), "startup_ready")
	return nil
}

func startupCheckCommandAuditSummaryV0(command StartupCheckCommandV0) map[string]interface{} {
	return map[string]interface{}{
		"correlation_id":       strings.TrimSpace(command.CorrelationID),
		"occurred_at":          formatTimeV0(command.OccurredAt),
		"project_configured":   strings.TrimSpace(command.ProjectWorkDir) != "",
		"runtime_configured":   strings.TrimSpace(command.RuntimeWorkDir) != "",
		"statefile_configured": strings.TrimSpace(command.StateDir) != "",
	}
}

func startupCheckResultAuditSummaryV0(result StartupCheckResultV0) map[string]interface{} {
	evidenceRefs := compactServerStringsV0(result.EvidenceRefs)
	summary := map[string]interface{}{
		"status":              strings.TrimSpace(result.Status),
		"ready":               result.Ready,
		"message":             publicReadinessMessageV0(result.Message),
		"evidence_refs":       append([]string(nil), evidenceRefs...),
		"evidence_refs_count": len(evidenceRefs),
	}
	if revision := normalizeStartupRevisionSummaryV0(result.StartupRevision); revision.RevisionRef != "" {
		summary["startup_revision"] = revision
	}
	return summary
}

func normalizeStartupCheckResultV0(result StartupCheckResultV0) StartupCheckResultV0 {
	result.Status = strings.TrimSpace(result.Status)
	result.Message = strings.TrimSpace(result.Message)
	result.EvidenceRefs = compactServerStringsV0(result.EvidenceRefs)
	result.StartupRevision = normalizeStartupRevisionSummaryV0(result.StartupRevision)
	if result.Ready && result.Status == "" {
		result.Status = StartupCheckStatusReadyV0
	}
	if !result.Ready && result.Status == "" {
		result.Status = StartupCheckStatusBlockedV0
	}
	if result.Ready && result.Message == "" {
		result.Message = "orquesta_startup_ready"
	}
	return result
}

func normalizeStartupRevisionSummaryV0(summary StartupRevisionSummaryV0) StartupRevisionSummaryV0 {
	summary.RevisionRef = strings.TrimSpace(summary.RevisionRef)
	if summary.QueueRemoved < 0 {
		summary.QueueRemoved = 0
	}
	if summary.QueueKept < 0 {
		summary.QueueKept = 0
	}
	if summary.ControlRemoved < 0 {
		summary.ControlRemoved = 0
	}
	if summary.ControlKept < 0 {
		summary.ControlKept = 0
	}
	if summary.RuntimeArchived < 0 {
		summary.RuntimeArchived = 0
	}
	if summary.Artifacts < 0 {
		summary.Artifacts = 0
	}
	if summary.RetentionDays < 0 {
		summary.RetentionDays = 0
	}
	if summary.MaxBytes < 0 {
		summary.MaxBytes = 0
	}
	return summary
}

func compactServerStringsV0(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
