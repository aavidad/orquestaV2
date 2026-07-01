package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

func runGuardianShutdownServerV0(ctx context.Context, config guardianConfigV0) guardianResultV0 {
	result := baseGuardianResultV0(config)
	result.Phase = "shutdown"
	if strings.TrimSpace(config.ServerAddr) == "" {
		result.Status = guardianStatusCandidateFailedV0
		result.Message = "server_addr_required"
		return writeGuardianManifestV0(config, result)
	}
	if err := ensureGuardianDirsV0(config); err != nil {
		result.Status = guardianStatusCandidateFailedV0
		result.Message = err.Error()
		return writeGuardianManifestV0(config, result)
	}
	lease, err := acquireGuardianPromotionLeaseV0(config, "shutdown")
	result.Lease = &lease
	if err != nil {
		result.Status = guardianStatusLeaseBusyV0
		result.Phase = "lease"
		result.Message = err.Error()
		result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-guardian-promotion-lease-busy")
		return writeGuardianManifestV0(config, result)
	}
	result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-guardian-promotion-lease-acquired")
	defer releaseGuardianPromotionLeaseV0(config, result.Lease)
	shutdown, err := waitGuardianServerShutdownReadyV0(ctx, config)
	result.Shutdown = &shutdown
	if err != nil {
		result.Status = guardianStatusCandidateFailedV0
		result.Message = err.Error()
		return writeGuardianManifestV0(config, result)
	}
	if config.ServerPID > 0 {
		if err := verifyGuardianPromotionLeaseV0(config, lease, "shutdown_signal"); err != nil {
			result.Status = guardianStatusLeaseLostV0
			result.Message = err.Error()
			result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-guardian-promotion-lease-lost")
			return writeGuardianManifestV0(config, result)
		}
		if err := signalGuardianProcessV0(config.ServerPID); err != nil {
			result.Shutdown.SignalStatus = "signal_blocked"
			result.Shutdown.SignalBlockReason = "guardian_shutdown_escalation_blocked"
			result.Status = guardianStatusCandidateFailedV0
			result.Message = err.Error()
			return writeGuardianManifestV0(config, result)
		}
		result.Shutdown.SignalStatus = "signal_sent"
	}
	result.Status = guardianStatusShutdownReadyV0
	result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-guardian-shutdown-ready")
	if config.ShutdownNow {
		result.Message = "server forced shutdown requested"
	} else if result.Shutdown != nil && !result.Shutdown.ShutdownReady {
		result.Message = "server forced shutdown requested after cooperative timeout"
	} else {
		result.Message = "server shutdown ready"
	}
	return writeGuardianManifestV0(config, result)
}

func waitGuardianServerShutdownReadyV0(
	ctx context.Context,
	config guardianConfigV0,
) (guardianShutdownResultV0, error) {
	if config.ShutdownNow {
		return requestGuardianServerShutdownV0(config)
	}
	timeout := config.ShutdownTimeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	deadline := time.Now().Add(timeout)
	var last guardianShutdownResultV0
	var lastErr error
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return last, err
		}
		shutdown, err := requestGuardianServerShutdownV0(config)
		if err == nil {
			last = shutdown
			if shutdown.ShutdownReady {
				return shutdown, nil
			}
		} else {
			lastErr = err
		}
		time.Sleep(time.Second)
	}
	if lastErr != nil {
		return last, lastErr
	}
	last.EscalationStatus = "cooperative_timeout"
	if config.ForceAfterTimeout {
		evidenceRef := strings.TrimSpace(config.ShutdownEscalationEvidenceRef)
		if evidenceRef == "" {
			last.EscalationStatus = "signal_blocked"
			last.SignalStatus = "signal_blocked"
			last.SignalBlockReason = "guardian_shutdown_escalation_blocked"
			return last, fmt.Errorf("guardian_shutdown_escalation_blocked")
		}
		forced := config
		forced.ShutdownForced = true
		forced.ShutdownNow = true
		shutdown, err := requestGuardianServerShutdownV0(forced)
		if err != nil {
			return last, err
		}
		shutdown.EscalationStatus = "forced_requested"
		shutdown.EscalationEvidenceRef = evidenceRef
		if shutdown.ShutdownReady || config.ServerPID > 0 {
			if shutdown.ShutdownReady {
				shutdown.EscalationStatus = "forced_ready"
			}
			return shutdown, nil
		}
		return shutdown, fmt.Errorf(
			"shutdown_forced_but_not_ready status=%s runs=%d/%d agents_in_flight=%d",
			shutdown.Status,
			shutdown.RunsStopped,
			shutdown.RunsRequested,
			shutdown.AgentsInFlight,
		)
	}
	return last, fmt.Errorf(
		"shutdown_not_ready status=%s runs=%d/%d agents_in_flight=%d checkpoints=%d checkpoint_agents=%d",
		last.Status,
		last.RunsStopped,
		last.RunsRequested,
		last.AgentsInFlight,
		last.CheckpointsPending,
		last.CheckpointAgentsPending,
	)
}

func requestGuardianServerShutdownV0(config guardianConfigV0) (guardianShutdownResultV0, error) {
	now := time.Now().UTC()
	payload := map[string]any{
		"forced":            config.ShutdownForced,
		"requested_by":      "orquesta-director",
		"reason":            guardianShutdownReasonV0(config),
		"idempotency_key":   "idem-orquesta-guardian-shutdown",
		"queue_limit":       config.ShutdownQueueLimit,
		"max_ticks":         8,
		"max_runs_per_tick": 4,
		"occurred_at":       now.Format(time.RFC3339),
		"evidence_refs":     []string{"evidence-ref-guardian-controlled-shutdown"},
	}
	if !config.ShutdownForced {
		payload["checkpoint_deadline_at"] = config.OccurredAt.Add(config.ShutdownTimeout).Format(time.RFC3339)
	}
	if ref := strings.TrimSpace(config.ShutdownEscalationEvidenceRef); config.ShutdownForced && ref != "" {
		payload["evidence_refs"] = []string{"evidence-ref-guardian-controlled-shutdown", ref}
	}
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(payload); err != nil {
		return guardianShutdownResultV0{}, err
	}
	client := newGuardianShutdownHTTPClientV0()
	response, err := client.Post(
		"http://"+strings.TrimSpace(config.ServerAddr)+"/api/v0/server/shutdown",
		"application/json",
		body,
	)
	if err != nil {
		return guardianShutdownResultV0{}, err
	}
	defer response.Body.Close()
	var result guardianShutdownResultV0
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return guardianShutdownResultV0{}, err
	}
	if response.StatusCode != http.StatusOK {
		return result, fmt.Errorf("shutdown_http_%d_%s", response.StatusCode, result.Status)
	}
	return result, nil
}

var newGuardianShutdownHTTPClientV0 = func() http.Client {
	return http.Client{Timeout: 30 * time.Second}
}

func guardianShutdownReasonV0(config guardianConfigV0) string {
	if config.ShutdownNow {
		return "director forced shutdown now via guardian"
	}
	return "director controlled shutdown via guardian"
}

func signalGuardianProcessV0(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Signal(os.Interrupt)
}
