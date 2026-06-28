package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

type opesBridgeObserveGoalResponseV0 struct {
	Estado                string                 `json:"estado"`
	RunRef                string                 `json:"run_ref"`
	RunStatus             string                 `json:"run_status"`
	DirectorExecutionMode string                 `json:"director_execution_mode"`
	GoalRef               string                 `json:"goal_ref"`
	ExternalGoalRef       string                 `json:"external_goal_ref"`
	GoalStatus            string                 `json:"goal_status"`
	ClosureStatus         string                 `json:"closure_status"`
	ClosureAccepted       bool                   `json:"closure_accepted"`
	ClosureNeedsRework    bool                   `json:"closure_needs_rework"`
	ArtifactRefs          []string               `json:"artifact_refs"`
	DomainReceiptRefs     []string               `json:"domain_receipt_refs"`
	EvidenceRefs          []string               `json:"evidence_refs"`
	Errores               []map[string]string    `json:"errores_publicos"`
	Raw                   map[string]interface{} `json:"-"`
}

func observeOPESExternalWorkGoalV0(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	runRef string,
	correlationID string,
) (opesExternalWorkRunSupervisionV0, error) {
	runRef = strings.TrimSpace(runRef)
	if runRef == "" {
		return opesExternalWorkRunSupervisionV0{}, fmt.Errorf("run_ref_required")
	}
	body, err := json.Marshal(map[string]any{
		"request_id":     "req-opes-bridge-observe-goal-" + opesBridgeCompactRunPartV0(runRef),
		"correlation_id": strings.TrimSpace(correlationID),
		"run_ref":        runRef,
		"requested_by":   "orquesta-opes-bridge",
	})
	if err != nil {
		return opesExternalWorkRunSupervisionV0{}, fmt.Errorf("request_marshal_error")
	}
	target, err := commandRESTEndpointURLV0(baseURL, orquestamcp.MCPObserveAppDirectorGoalHTTPPathV0)
	if err != nil {
		return opesExternalWorkRunSupervisionV0{}, fmt.Errorf("request_build_error")
	}
	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		target,
		bytes.NewReader(body),
	)
	if err != nil {
		return opesExternalWorkRunSupervisionV0{}, fmt.Errorf("request_build_error")
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := client.Do(httpRequest)
	if err != nil {
		return opesExternalWorkRunSupervisionV0{}, errors.New(commandEffectHTTPErrorCodeV0(ctx, err))
	}
	defer response.Body.Close()
	responseBody, err := readCommandHTTPResponseBodyV0(response, "observe_goal")
	if err != nil {
		return opesExternalWorkRunSupervisionV0{}, err
	}
	var decoded opesBridgeObserveGoalResponseV0
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return opesExternalWorkRunSupervisionV0{}, fmt.Errorf("response_decode_error")
	}
	if strings.TrimSpace(decoded.Estado) == "error" {
		return opesExternalWorkRunSupervisionV0{}, fmt.Errorf("observe_goal_error")
	}
	return opesBridgeSupervisionFromObserveGoalV0(decoded), nil
}

func opesBridgeSupervisionFromObserveGoalV0(
	decoded opesBridgeObserveGoalResponseV0,
) opesExternalWorkRunSupervisionV0 {
	status := strings.TrimSpace(decoded.GoalStatus)
	runStatus := strings.TrimSpace(decoded.RunStatus)
	closureStatus := strings.TrimSpace(decoded.ClosureStatus)
	artifactRefs := compactStringsV0(decoded.ArtifactRefs)
	domainReceiptRefs := compactStringsV0(decoded.DomainReceiptRefs)
	evidenceRefs := compactStringsV0(decoded.EvidenceRefs)
	evidenceRef := firstNonEmptyEnvlessV0(
		firstNonEmptyEnvlessV0(evidenceRefs...),
		firstNonEmptyEnvlessV0(domainReceiptRefs...),
		firstNonEmptyEnvlessV0(artifactRefs...),
	)
	switch {
	case decoded.ClosureAccepted || closureStatus == "accepted" || runStatus == "closed":
		return opesExternalWorkRunSupervisionV0{
			Status:            "closed",
			StopReason:        firstNonEmptyEnvlessV0(closureStatus, "goal_first_accepted"),
			EvidenceRef:       evidenceRef,
			ArtifactRefs:      artifactRefs,
			DomainReceiptRefs: domainReceiptRefs,
			EvidenceRefs:      evidenceRefs,
		}
	case decoded.ClosureNeedsRework || closureStatus == "blocked" || status == "blocked" || status == "invalid" || runStatus == "blocked":
		return opesExternalWorkRunSupervisionV0{
			Status:            "blocked",
			StopReason:        firstNonEmptyEnvlessV0(closureStatus, status, "goal_first_blocked"),
			EvidenceRef:       evidenceRef,
			ArtifactRefs:      artifactRefs,
			DomainReceiptRefs: domainReceiptRefs,
			EvidenceRefs:      evidenceRefs,
		}
	case status == "complete":
		return opesExternalWorkRunSupervisionV0{
			Status:            "completed",
			StopReason:        firstNonEmptyEnvlessV0(closureStatus, "goal_first_complete_pending_closure"),
			EvidenceRef:       evidenceRef,
			ArtifactRefs:      artifactRefs,
			DomainReceiptRefs: domainReceiptRefs,
			EvidenceRefs:      evidenceRefs,
		}
	case status == "running":
		return opesExternalWorkRunSupervisionV0{
			Status:            "running",
			StopReason:        firstNonEmptyEnvlessV0(runStatus, "goal_first_running"),
			EvidenceRef:       evidenceRef,
			ArtifactRefs:      artifactRefs,
			DomainReceiptRefs: domainReceiptRefs,
			EvidenceRefs:      evidenceRefs,
		}
	default:
		return opesExternalWorkRunSupervisionV0{
			Status:            firstNonEmptyEnvlessV0(status, runStatus, "goal_first_observe_pending"),
			StopReason:        firstNonEmptyEnvlessV0(closureStatus, "goal_first_observe_pending"),
			EvidenceRef:       evidenceRef,
			ArtifactRefs:      artifactRefs,
			DomainReceiptRefs: domainReceiptRefs,
			EvidenceRefs:      evidenceRefs,
		}
	}
}
