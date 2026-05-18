package orquestacionnucleoapp

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

const RequiredTestEvidenceSchemaVersionV0 = "required_test_evidence.v0"

type RequiredTestEvidenceStatusV0 string

const (
	RequiredTestEvidenceStatusPassedV0 RequiredTestEvidenceStatusV0 = "passed"
	RequiredTestEvidenceStatusFailedV0 RequiredTestEvidenceStatusV0 = "failed"
)

type RequiredTestEvidenceV0 struct {
	SchemaVersion     string                       `json:"schema_version"`
	EvidenceRef       string                       `json:"evidence_ref"`
	RunRef            string                       `json:"run_ref"`
	TaskRef           string                       `json:"task_ref"`
	TestCommand       string                       `json:"test_command"`
	Status            RequiredTestEvidenceStatusV0 `json:"status"`
	DeliveryRef       string                       `json:"delivery_ref"`
	ReviewRequestID   string                       `json:"review_request_id"`
	ReviewResultRef   string                       `json:"review_result_ref"`
	AcceptedReviewRef string                       `json:"accepted_review_ref"`
	OccurredAt        string                       `json:"occurred_at"`
	EvidenceRefs      []string                     `json:"evidence_refs,omitempty"`
}

type RequiredTestEvidenceReaderPortV0 interface {
	LoadRequiredTestEvidenceV0(ctx context.Context, runRef string, evidenceRefs []string) ([]RequiredTestEvidenceV0, error)
}

type RequiredTestEvidenceWriterPortV0 interface {
	SaveRequiredTestEvidenceV0(ctx context.Context, evidence RequiredTestEvidenceV0) error
}

type RequiredTestEvidenceStorePortV0 interface {
	RequiredTestEvidenceReaderPortV0
	RequiredTestEvidenceWriterPortV0
}

type InMemoryRequiredTestEvidenceStoreV0 struct {
	mu       sync.Mutex
	evidence map[string]map[string]RequiredTestEvidenceV0
}

var _ RequiredTestEvidenceStorePortV0 = (*InMemoryRequiredTestEvidenceStoreV0)(nil)

func NewInMemoryRequiredTestEvidenceStoreV0(
	items ...RequiredTestEvidenceV0,
) *InMemoryRequiredTestEvidenceStoreV0 {
	store := &InMemoryRequiredTestEvidenceStoreV0{
		evidence: map[string]map[string]RequiredTestEvidenceV0{},
	}
	for _, item := range items {
		_ = store.saveRequiredTestEvidenceV0(item)
	}
	return store
}

func (store *InMemoryRequiredTestEvidenceStoreV0) SaveRequiredTestEvidenceV0(
	ctx context.Context,
	evidence RequiredTestEvidenceV0,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.saveRequiredTestEvidenceV0(evidence)
}

func (store *InMemoryRequiredTestEvidenceStoreV0) LoadRequiredTestEvidenceV0(
	ctx context.Context,
	runRef string,
	evidenceRefs []string,
) ([]RequiredTestEvidenceV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	runRef = strings.TrimSpace(runRef)
	refs := compactStringsV0(evidenceRefs)
	if runRef == "" {
		return nil, errorV0(ErrNucleoOrquestacionInvalidoV0, "run_ref", "run_ref requerido")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	out := make([]RequiredTestEvidenceV0, 0, len(refs))
	byRun := store.evidence[runRef]
	for _, ref := range refs {
		item, ok := byRun[ref]
		if !ok {
			return nil, errorV0(
				ErrNucleoOrquestacionStoreV0,
				"required_test_evidence",
				fmt.Sprintf("evidencia de test no encontrada: %s", ref),
			)
		}
		out = append(out, item)
	}
	return out, nil
}

func (store *InMemoryRequiredTestEvidenceStoreV0) saveRequiredTestEvidenceV0(
	evidence RequiredTestEvidenceV0,
) error {
	normalized, err := NewRequiredTestEvidenceV0(evidence)
	if err != nil {
		return err
	}
	if store.evidence[normalized.RunRef] == nil {
		store.evidence[normalized.RunRef] = map[string]RequiredTestEvidenceV0{}
	}
	if existing, ok := store.evidence[normalized.RunRef][normalized.EvidenceRef]; ok {
		if !reflect.DeepEqual(existing, normalized) {
			return errorV0(ErrNucleoOrquestacionStoreV0, "required_test_evidence", "evidencia de test existente con contrato distinto")
		}
		return nil
	}
	store.evidence[normalized.RunRef][normalized.EvidenceRef] = normalized
	return nil
}

func NewRequiredTestEvidenceV0(
	evidence RequiredTestEvidenceV0,
) (RequiredTestEvidenceV0, error) {
	normalized := NormalizeRequiredTestEvidenceV0(evidence)
	if err := ValidateRequiredTestEvidenceV0(normalized); err != nil {
		return RequiredTestEvidenceV0{}, err
	}
	return normalized, nil
}

func NormalizeRequiredTestEvidenceV0(
	evidence RequiredTestEvidenceV0,
) RequiredTestEvidenceV0 {
	return RequiredTestEvidenceV0{
		SchemaVersion:     strings.TrimSpace(evidence.SchemaVersion),
		EvidenceRef:       strings.TrimSpace(evidence.EvidenceRef),
		RunRef:            strings.TrimSpace(evidence.RunRef),
		TaskRef:           strings.TrimSpace(evidence.TaskRef),
		TestCommand:       strings.TrimSpace(evidence.TestCommand),
		Status:            RequiredTestEvidenceStatusV0(strings.TrimSpace(string(evidence.Status))),
		DeliveryRef:       strings.TrimSpace(evidence.DeliveryRef),
		ReviewRequestID:   strings.TrimSpace(evidence.ReviewRequestID),
		ReviewResultRef:   strings.TrimSpace(evidence.ReviewResultRef),
		AcceptedReviewRef: strings.TrimSpace(evidence.AcceptedReviewRef),
		OccurredAt:        strings.TrimSpace(evidence.OccurredAt),
		EvidenceRefs:      compactStringsV0(evidence.EvidenceRefs),
	}
}

func ValidateRequiredTestEvidenceV0(evidence RequiredTestEvidenceV0) error {
	if evidence.SchemaVersion != RequiredTestEvidenceSchemaVersionV0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "schema_version", "schema_version invalida")
	}
	for field, value := range map[string]string{
		"evidence_ref":        evidence.EvidenceRef,
		"run_ref":             evidence.RunRef,
		"task_ref":            evidence.TaskRef,
		"test_command":        evidence.TestCommand,
		"delivery_ref":        evidence.DeliveryRef,
		"review_request_id":   evidence.ReviewRequestID,
		"review_result_ref":   evidence.ReviewResultRef,
		"accepted_review_ref": evidence.AcceptedReviewRef,
		"occurred_at":         evidence.OccurredAt,
	} {
		if strings.TrimSpace(value) == "" {
			return errorV0(ErrNucleoOrquestacionInvalidoV0, field, field+" requerido")
		}
	}
	if evidence.Status != RequiredTestEvidenceStatusPassedV0 &&
		evidence.Status != RequiredTestEvidenceStatusFailedV0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "status", "status invalido")
	}
	if len(evidence.EvidenceRefs) == 0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "evidence_refs", "evidence_refs requerido")
	}
	return nil
}
