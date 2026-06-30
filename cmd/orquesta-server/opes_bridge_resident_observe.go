package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func observeOPESBridgeResidentDispatchUntilV0(
	ctx context.Context,
	client *http.Client,
	config opesDrainConfigV0,
	runRef string,
) (opesExternalWorkRunSupervisionV0, error) {
	interval := config.ResidentDispatchPollInterval
	if interval <= 0 {
		interval = defaultOPESBridgeResidentDispatchIntervalV0
	}
	timeout := time.NewTimer(config.ResidentDispatchWait)
	defer timeout.Stop()
	var last opesExternalWorkRunSupervisionV0
	var lastErr error
	for {
		supervision, err := observeOPESExternalWorkRunStatsV0(
			ctx,
			client,
			config.OrquestaBaseURL,
			runRef,
			config.CorrelationID,
		)
		if err == nil {
			last = supervision
			if opesBridgeResidentDispatchObservedV0(supervision) {
				return supervision, nil
			}
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return opesExternalWorkRunSupervisionV0{}, lastErr
			}
			return opesExternalWorkRunSupervisionV0{}, ctx.Err()
		case <-timeout.C:
			if strings.TrimSpace(last.Status) == "" {
				last.Status = "resident_director_pending"
			}
			if strings.TrimSpace(last.Status) == "resident_director_pending" ||
				strings.TrimSpace(last.StopReason) == "" {
				last.StopReason = "resident_dispatch_wait_timeout"
			}
			return last, nil
		case <-time.After(interval):
		}
	}
}

func opesBridgeResidentDispatchObservedV0(
	supervision opesExternalWorkRunSupervisionV0,
) bool {
	switch strings.TrimSpace(supervision.Status) {
	case "started", "running", "blocked", "closed", "completed", "done", "retry_pending", "needs_reconcile":
		return true
	default:
		return false
	}
}

func observeOPESExternalWorkRunStatsV0(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	runRef string,
	correlationID string,
) (opesExternalWorkRunSupervisionV0, error) {
	decoded, err := loadOPESExternalWorkRunStatsResponseV0(ctx, client, baseURL, runRef, correlationID)
	if err != nil {
		return opesExternalWorkRunSupervisionV0{}, err
	}
	return opesBridgeSupervisionFromDirectorStatsV0(decoded), nil
}

func loadOPESExternalWorkRunStatsResponseV0(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	runRef string,
	correlationID string,
) (opesBridgeDirectorStatsResponseV0, error) {
	runRef = strings.TrimSpace(runRef)
	if runRef == "" {
		return opesBridgeDirectorStatsResponseV0{}, fmt.Errorf("run_ref_required")
	}
	body, err := json.Marshal(map[string]any{
		"request_id":             "req-opes-bridge-resident-observe-" + opesBridgeCompactRunPartV0(runRef),
		"correlation_id":         strings.TrimSpace(correlationID),
		"run_ref":                runRef,
		"include_process_refs":   true,
		"include_agent_usage":    false,
		"include_agent_progress": false,
	})
	if err != nil {
		return opesBridgeDirectorStatsResponseV0{}, fmt.Errorf("request_marshal_error")
	}
	target, err := commandRESTEndpointURLV0(baseURL, "/api/v0/director/stats")
	if err != nil {
		return opesBridgeDirectorStatsResponseV0{}, fmt.Errorf("request_build_error")
	}
	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		target,
		bytes.NewReader(body),
	)
	if err != nil {
		return opesBridgeDirectorStatsResponseV0{}, fmt.Errorf("request_build_error")
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := client.Do(httpRequest)
	if err != nil {
		return opesBridgeDirectorStatsResponseV0{}, errors.New(commandEffectHTTPErrorCodeV0(ctx, err))
	}
	defer response.Body.Close()
	responseBody, err := readCommandHTTPResponseBodyV0(response, "director_stats")
	if err != nil {
		return opesBridgeDirectorStatsResponseV0{}, err
	}
	var decoded opesBridgeDirectorStatsResponseV0
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return opesBridgeDirectorStatsResponseV0{}, fmt.Errorf("response_decode_error")
	}
	if strings.TrimSpace(decoded.Estado) == "error" {
		return opesBridgeDirectorStatsResponseV0{}, fmt.Errorf("director_stats_error")
	}
	return decoded, nil
}

type opesBridgeDirectorStatsResponseV0 struct {
	Estado string `json:"estado"`
	Goal   struct {
		DirectorExecutionMode string         `json:"director_execution_mode"`
		GoalRef               string         `json:"goal_ref"`
		ExternalGoalRef       string         `json:"external_goal_ref"`
		Status                string         `json:"status"`
		ClosureStatus         string         `json:"closure_status"`
		ClosureAccepted       bool           `json:"closure_accepted"`
		ClosureNeedsRework    bool           `json:"closure_needs_rework"`
		CurrentPhase          string         `json:"current_phase"`
		RetryFromPhase        string         `json:"retry_from_phase"`
		OperationalReason     string         `json:"operational_reason"`
		DomainCounters        map[string]int `json:"domain_counters"`
		EvidenceRefs          []string       `json:"evidence_refs"`
	} `json:"goal"`
	ExternalJob struct {
		CurrentPhase      string         `json:"current_phase"`
		RetryFromPhase    string         `json:"retry_from_phase"`
		OperationalReason string         `json:"operational_reason"`
		DomainCounters    map[string]int `json:"domain_counters"`
	} `json:"external_job"`
	Stats struct {
		Status string `json:"status"`
		Counts struct {
			TasksDelivered  int `json:"tasks_delivered"`
			AgentsStarted   int `json:"agents_started"`
			AgentsInFlight  int `json:"agents_in_flight"`
			AgentsFailed    int `json:"agents_failed"`
			AgentsLost      int `json:"agents_lost"`
			AgentsDelivered int `json:"agents_delivered"`
			Deliveries      int `json:"deliveries"`
			Blockers        int `json:"blockers"`
			Closures        int `json:"closures"`
		} `json:"counts"`
		Refs struct {
			DeliveredTasks  []string `json:"delivered_tasks"`
			AgentsStarted   []string `json:"agents_started"`
			AgentsFailed    []string `json:"agents_failed"`
			AgentsLost      []string `json:"agents_lost"`
			AgentsDelivered []string `json:"agents_delivered"`
			Deliveries      []string `json:"deliveries"`
			Blockers        []string `json:"blockers"`
			Closures        []string `json:"closures"`
		} `json:"refs"`
		Closure struct {
			Status      string   `json:"status"`
			Blocked     bool     `json:"blocked"`
			Closed      bool     `json:"closed"`
			BlockedBy   []string `json:"blocked_by"`
			BlockerRefs []string `json:"blocker_refs"`
		} `json:"closure"`
		Agents []struct {
			AgentRequestID string `json:"agent_request_id"`
			Started        bool   `json:"started"`
			InFlight       bool   `json:"in_flight"`
			Failed         bool   `json:"failed"`
			Lost           bool   `json:"lost"`
			Process        *struct {
				ProcessRef   string   `json:"process_ref"`
				EvidenceRefs []string `json:"evidence_refs"`
			} `json:"process"`
		} `json:"agents"`
	} `json:"stats"`
}

func opesBridgeSupervisionFromDirectorStatsV0(
	decoded opesBridgeDirectorStatsResponseV0,
) opesExternalWorkRunSupervisionV0 {
	if metadata, ok := opesBridgeGoalMetadataFromDirectorStatsV0(decoded); ok {
		return opesExternalWorkRunSupervisionV0{
			Status:                opesBridgeGoalStatsSupervisionStatusV0(decoded),
			StopReason:            firstNonEmptyEnvlessV0(metadata.OperationalReason, decoded.Goal.ClosureStatus, decoded.Goal.Status, "goal_first_observe_required"),
			EvidenceRef:           firstNonEmptyEnvlessV0(decoded.Goal.EvidenceRefs...),
			RoutePolicy:           metadata.RoutePolicy,
			DirectorExecutionMode: metadata.DirectorExecutionMode,
			GoalRef:               metadata.GoalRef,
			ExternalGoalRef:       metadata.ExternalGoalRef,
			NextActions:           compactStringsV0(metadata.NextActions),
			CurrentPhase:          metadata.CurrentPhase,
			OperationalReason:     metadata.OperationalReason,
			AudioCounters:         copyStringIntMapV0(metadata.DomainCounters),
		}
	}
	stats := decoded.Stats
	processRef, processEvidence := opesBridgeFirstProcessFromDirectorStatsV0(decoded)
	switch {
	case stats.Closure.Closed || stats.Counts.Closures > 0:
		return opesExternalWorkRunSupervisionV0{
			Status:      "closed",
			StopReason:  firstNonEmptyEnvlessV0(stats.Closure.Status, "run_closed"),
			ProcessRef:  processRef,
			EvidenceRef: firstNonEmptyEnvlessV0(opesBridgePrependStringV0(processEvidence, stats.Refs.Closures)...),
		}
	case stats.Closure.Blocked || stats.Counts.Blockers > 0:
		return opesExternalWorkRunSupervisionV0{
			Status:      "blocked",
			StopReason:  firstNonEmptyEnvlessV0(stats.Closure.BlockedBy...),
			ProcessRef:  processRef,
			EvidenceRef: firstNonEmptyEnvlessV0(opesBridgePrependStringV0(processEvidence, stats.Closure.BlockerRefs)...),
		}
	case stats.Counts.AgentsFailed > 0 || stats.Counts.AgentsLost > 0 ||
		len(stats.Refs.AgentsFailed) > 0 || len(stats.Refs.AgentsLost) > 0 ||
		opesBridgeDirectorStatsHasFailedOrLostAgentV0(decoded):
		failedRefs := append([]string{}, stats.Refs.AgentsFailed...)
		failedRefs = append(failedRefs, stats.Refs.AgentsLost...)
		return opesExternalWorkRunSupervisionV0{
			Status:      "blocked",
			StopReason:  "agent_failed_or_lost",
			ProcessRef:  processRef,
			EvidenceRef: firstNonEmptyEnvlessV0(opesBridgePrependStringV0(processEvidence, failedRefs)...),
		}
	case opesBridgeDirectorStatsHasDeliveryWithoutLiveProcessV0(decoded):
		deliveryRefs := append([]string{}, stats.Refs.Deliveries...)
		deliveryRefs = append(deliveryRefs, stats.Refs.DeliveredTasks...)
		deliveryRefs = append(deliveryRefs, stats.Refs.AgentsDelivered...)
		return opesExternalWorkRunSupervisionV0{
			Status:      "needs_reconcile",
			StopReason:  "stale_lock_no_process",
			ProcessRef:  processRef,
			EvidenceRef: firstNonEmptyEnvlessV0(opesBridgePrependStringV0(processEvidence, deliveryRefs)...),
		}
	case stats.Counts.AgentsStarted > 0 || stats.Counts.AgentsInFlight > 0 ||
		len(stats.Refs.AgentsStarted) > 0 || opesBridgeDirectorStatsHasStartedAgentV0(decoded):
		return opesExternalWorkRunSupervisionV0{
			Status:      "started",
			ProcessRef:  processRef,
			EvidenceRef: firstNonEmptyEnvlessV0(opesBridgePrependStringV0(processEvidence, stats.Refs.AgentsStarted)...),
		}
	default:
		return opesExternalWorkRunSupervisionV0{
			Status:     "resident_director_pending",
			StopReason: firstNonEmptyEnvlessV0(stats.Status, "waiting_resident_dispatch"),
		}
	}
}

func opesBridgeGoalStatsSupervisionStatusV0(
	decoded opesBridgeDirectorStatsResponseV0,
) string {
	closureStatus := strings.ToLower(strings.TrimSpace(decoded.Goal.ClosureStatus))
	goalStatus := strings.ToLower(strings.TrimSpace(decoded.Goal.Status))
	switch {
	case decoded.Goal.ClosureAccepted || closureStatus == "accepted" || closureStatus == "closed":
		return "closed"
	case decoded.Goal.ClosureNeedsRework ||
		closureStatus == "needs_rework" ||
		closureStatus == "blocked" ||
		goalStatus == "blocked" ||
		goalStatus == "invalid":
		return "blocked"
	case goalStatus == "complete":
		return "completed"
	case goalStatus != "":
		return goalStatus
	default:
		return "running"
	}
}

func opesBridgeDirectorStatsHasDeliveryWithoutLiveProcessV0(
	decoded opesBridgeDirectorStatsResponseV0,
) bool {
	stats := decoded.Stats
	if stats.Counts.AgentsStarted > 0 ||
		stats.Counts.AgentsInFlight > 0 ||
		len(stats.Refs.AgentsStarted) > 0 ||
		opesBridgeDirectorStatsHasStartedAgentV0(decoded) {
		return false
	}
	return stats.Counts.TasksDelivered > 0 ||
		stats.Counts.AgentsDelivered > 0 ||
		stats.Counts.Deliveries > 0 ||
		len(stats.Refs.DeliveredTasks) > 0 ||
		len(stats.Refs.AgentsDelivered) > 0 ||
		len(stats.Refs.Deliveries) > 0
}

func opesBridgePrependStringV0(value string, values []string) []string {
	out := make([]string, 0, len(values)+1)
	out = append(out, value)
	out = append(out, values...)
	return out
}

func opesBridgeDirectorStatsHasFailedOrLostAgentV0(
	decoded opesBridgeDirectorStatsResponseV0,
) bool {
	for _, agent := range decoded.Stats.Agents {
		if agent.Failed || agent.Lost {
			return true
		}
	}
	return false
}

func opesBridgeDirectorStatsHasStartedAgentV0(
	decoded opesBridgeDirectorStatsResponseV0,
) bool {
	for _, agent := range decoded.Stats.Agents {
		if agent.Started || agent.InFlight {
			return true
		}
	}
	return false
}

func opesBridgeFirstProcessFromDirectorStatsV0(
	decoded opesBridgeDirectorStatsResponseV0,
) (string, string) {
	for _, agent := range decoded.Stats.Agents {
		if agent.Process == nil {
			continue
		}
		processRef := strings.TrimSpace(agent.Process.ProcessRef)
		evidenceRef := firstNonEmptyEnvlessV0(agent.Process.EvidenceRefs...)
		if processRef != "" || evidenceRef != "" {
			return processRef, evidenceRef
		}
	}
	return "", ""
}
