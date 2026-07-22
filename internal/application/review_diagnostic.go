package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const reviewDiagnosticMediaType = "application/vnd.orquesta.review-diagnostic+json"

type reviewDiagnosticEvidence struct {
	SchemaVersion int    `json:"schema_version"`
	ErrorCode     string `json:"error_code"`
	PayloadSHA256 string `json:"payload_sha256"`
	PayloadBytes  int64  `json:"payload_bytes"`
}

// publishReviewDiagnostic preserves useful provenance without persisting raw
// reviewer output, which may contain secrets or provider-private context.
func (orchestrator *Orchestrator) publishReviewDiagnostic(ctx context.Context, record GoalRecord, item goal.WorkItem,
	execution ExecutionRecord, content []byte, code string, at time.Time,
) (*ArtifactRecord, error) {
	digest := sha256.Sum256(content)
	payload, err := json.Marshal(reviewDiagnosticEvidence{
		SchemaVersion: 1, ErrorCode: stableFailureCode(code),
		PayloadSHA256: hex.EncodeToString(digest[:]), PayloadBytes: int64(len(content)),
	})
	if err != nil {
		return nil, err
	}
	stored, err := orchestrator.publishTestArtifact(ctx, ports.PutArtifactRequest{
		MediaType: reviewDiagnosticMediaType, Content: payload,
	})
	if err != nil {
		return nil, err
	}
	return &ArtifactRecord{
		OccurrenceRef: "artifact-occurrence:review-diagnostic:" + execution.Ref.String(),
		Kind:          ArtifactKindReviewDiagnostic, Stored: stored,
		GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
		ExecutionAttempt: execution.AttemptNo, PlanGeneration: execution.PlanGeneration,
		WorkItemGeneration: item.Revision(), AppSpecGeneration: execution.AppSpecGeneration,
		SpecHash: execution.SpecHash, CreatedAt: at.UTC(),
	}, nil
}
