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
	Status       string   `json:"status"`
	Ready        bool     `json:"ready"`
	Message      string   `json:"message,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
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
		runtime.persistStateV0(ctx, runtime.tracker.MarkStartupReadyV0(
			result,
			runtime.clock.Now(),
		))
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
		"command": command,
	})
	runtime.persistStateV0(ctx, runtime.tracker.MarkStartupCheckingV0(now))
	result, err := runtime.startupCheck.PrepareStartupV0(ctx, command)
	if err != nil {
		runtime.auditEventV0(ctx, "startup_check_error", "blocked", err.Error(), map[string]interface{}{
			"command": command,
		})
		runtime.persistStateV0(ctx, runtime.tracker.MarkStartupBlockedV0(
			StartupCheckResultV0{
				Status:  StartupCheckStatusBlockedV0,
				Ready:   false,
				Message: err.Error(),
			},
			runtime.clock.Now(),
		))
		return err
	}
	result = normalizeStartupCheckResultV0(result)
	if !result.Ready {
		runtime.auditEventV0(ctx, "startup_check_blocked", "blocked", result.Message, map[string]interface{}{
			"command": command,
			"result":  result,
		})
		runtime.persistStateV0(ctx, runtime.tracker.MarkStartupBlockedV0(result, runtime.clock.Now()))
		return StartupNotReadyErrorV0{Status: result.Status, Message: result.Message}
	}
	runtime.auditEventV0(ctx, "startup_check_ready", "ready", "", map[string]interface{}{
		"command": command,
		"result":  result,
	})
	runtime.persistStateV0(ctx, runtime.tracker.MarkStartupReadyV0(result, runtime.clock.Now()))
	return nil
}

func normalizeStartupCheckResultV0(result StartupCheckResultV0) StartupCheckResultV0 {
	result.Status = strings.TrimSpace(result.Status)
	result.Message = strings.TrimSpace(result.Message)
	result.EvidenceRefs = compactServerStringsV0(result.EvidenceRefs)
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
