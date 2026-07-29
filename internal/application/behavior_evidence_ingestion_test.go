package application

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

const behaviorEvidenceFixtureDigest = "312620700a661a354ceecf2403da3bd55f9c98f1585d894b06e0f9423b9266a1"

type behaviorEvidenceFixedClock struct {
	now time.Time
}

func (clock behaviorEvidenceFixedClock) Now() time.Time { return clock.now }

type behaviorEvidenceTestStore struct {
	mu       sync.Mutex
	calls    int
	manifest BehaviorEvidenceManifest
	receipt  BehaviorEvidenceReceipt
	err      error
}

func (store *behaviorEvidenceTestStore) AcceptBehaviorEvidenceManifest(
	_ context.Context,
	manifest BehaviorEvidenceManifest,
	acceptedAt time.Time,
) (BehaviorEvidenceReceipt, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.calls++
	store.manifest = cloneBehaviorEvidenceManifest(manifest)
	if store.err != nil {
		return BehaviorEvidenceReceipt{}, store.err
	}
	if store.receipt.Schema == "" {
		store.receipt = BehaviorEvidenceReceipt{
			Schema:         BehaviorEvidenceReceiptSchema,
			ReceiptRef:     "receipt-" + manifest.ManifestDigest,
			ManifestDigest: manifest.ManifestDigest,
			AcceptedAt:     acceptedAt.UTC().Format(time.RFC3339Nano),
		}
	}
	return store.receipt, nil
}

func TestBehaviorEvidenceCanonicalDigestMatchesConnectorContract(t *testing.T) {
	manifest := behaviorEvidenceFixture(t)
	manifest.ManifestDigest = ""
	digest, err := BehaviorEvidenceManifestDigest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if digest != behaviorEvidenceFixtureDigest {
		t.Fatalf("digest=%s want=%s", digest, behaviorEvidenceFixtureDigest)
	}
}

func TestBehaviorEvidenceIngestionReturnsOpaqueReceiptAndDefensiveManifest(t *testing.T) {
	acceptedAt := time.Date(2026, 7, 29, 17, 2, 3, 456000000, time.UTC)
	store := &behaviorEvidenceTestStore{}
	service, err := NewBehaviorEvidenceIngestionService(
		store,
		behaviorEvidenceFixedClock{now: acceptedAt},
	)
	if err != nil {
		t.Fatal(err)
	}
	manifest := behaviorEvidenceFixture(t)
	receipt, err := service.IngestBehaviorEvidenceManifest(t.Context(), manifest)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Schema != BehaviorEvidenceReceiptSchema ||
		receipt.ManifestDigest != manifest.ManifestDigest ||
		receipt.AcceptedAt != "2026-07-29T17:02:03.456Z" ||
		!validBehaviorEvidenceOpaqueRef(receipt.ReceiptRef) {
		t.Fatalf("receipt=%+v", receipt)
	}
	manifest.Artifacts[0].ArtifactRef = "artifact-mutated"
	if store.manifest.Artifacts[0].ArtifactRef != "artifact-001" {
		t.Fatal("store received aliased manifest")
	}
}

func TestBehaviorEvidenceManifestRejectsContractAndSecurityDrift(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*BehaviorEvidenceManifest)
	}{
		{"schema", func(manifest *BehaviorEvidenceManifest) {
			manifest.Schema = "application.behavior-evidence.manifest.v2"
		}},
		{"version", func(manifest *BehaviorEvidenceManifest) {
			manifest.ContractVersion = "2.0.0"
		}},
		{"analysis ref", func(manifest *BehaviorEvidenceManifest) {
			manifest.AnalysisRef = `C:\private\analysis`
		}},
		{"purpose", func(manifest *BehaviorEvidenceManifest) {
			manifest.Purpose = "execute"
		}},
		{"created at", func(manifest *BehaviorEvidenceManifest) {
			manifest.CreatedAt = "yesterday"
		}},
		{"classification", func(manifest *BehaviorEvidenceManifest) {
			manifest.Classification = "restricted"
		}},
		{"review", func(manifest *BehaviorEvidenceManifest) {
			manifest.Review.Status = "pending"
		}},
		{"review scope", func(manifest *BehaviorEvidenceManifest) {
			manifest.Review.Scope = "private"
		}},
		{"review digest", func(manifest *BehaviorEvidenceManifest) {
			manifest.Review.ReviewedDigestSHA256 = strings.Repeat("A", 64)
		}},
		{"artifact ref", func(manifest *BehaviorEvidenceManifest) {
			manifest.Artifacts[0].ArtifactRef = "../artifact"
		}},
		{"artifact kind", func(manifest *BehaviorEvidenceManifest) {
			manifest.Artifacts[0].Kind = ""
		}},
		{"artifact media", func(manifest *BehaviorEvidenceManifest) {
			manifest.Artifacts[0].MediaType = "not a media type"
		}},
		{"artifact size", func(manifest *BehaviorEvidenceManifest) {
			manifest.Artifacts[0].SizeBytes = BehaviorEvidenceMaxArtifactBytes + 1
		}},
		{"artifact digest", func(manifest *BehaviorEvidenceManifest) {
			manifest.Artifacts[0].SHA256 = "sha256:invalid"
		}},
		{"artifact classification", func(manifest *BehaviorEvidenceManifest) {
			manifest.Artifacts[0].Classification = "restricted"
		}},
		{"artifact review", func(manifest *BehaviorEvidenceManifest) {
			manifest.Artifacts[0].ReviewStatus = "rejected"
		}},
		{"duplicate artifact", func(manifest *BehaviorEvidenceManifest) {
			manifest.Artifacts = append(
				manifest.Artifacts,
				manifest.Artifacts[0],
			)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := behaviorEvidenceFixture(t)
			test.mutate(&manifest)
			if digest, err := BehaviorEvidenceManifestDigest(manifest); err == nil {
				manifest.ManifestDigest = digest
			}
			if !errors.Is(
				ValidateBehaviorEvidenceManifest(manifest),
				ErrBehaviorEvidenceInvalid,
			) {
				t.Fatalf("invalid manifest accepted: %+v", manifest)
			}
		})
	}
}

func TestBehaviorEvidenceManifestRejectsDigestMismatchBeforeStore(t *testing.T) {
	store := &behaviorEvidenceTestStore{}
	service, err := NewBehaviorEvidenceIngestionService(
		store,
		behaviorEvidenceFixedClock{now: time.Now()},
	)
	if err != nil {
		t.Fatal(err)
	}
	manifest := behaviorEvidenceFixture(t)
	manifest.ManifestDigest = strings.Repeat("c", 64)
	if _, err := service.IngestBehaviorEvidenceManifest(
		t.Context(),
		manifest,
	); !errors.Is(err, ErrBehaviorEvidenceInvalid) {
		t.Fatalf("error=%v", err)
	}
	if store.calls != 0 {
		t.Fatalf("invalid manifest reached store: calls=%d", store.calls)
	}
}

func TestBehaviorEvidenceIngestionRejectsStoreReceiptSubstitution(t *testing.T) {
	store := &behaviorEvidenceTestStore{
		receipt: BehaviorEvidenceReceipt{
			Schema:         BehaviorEvidenceReceiptSchema,
			ReceiptRef:     "receipt-substituted",
			ManifestDigest: strings.Repeat("d", 64),
			AcceptedAt:     "2026-07-29T17:02:03Z",
		},
	}
	service, err := NewBehaviorEvidenceIngestionService(
		store,
		behaviorEvidenceFixedClock{now: time.Now()},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.IngestBehaviorEvidenceManifest(
		t.Context(),
		behaviorEvidenceFixture(t),
	); !errors.Is(err, ErrBehaviorEvidenceConflict) {
		t.Fatalf("error=%v", err)
	}
}

func TestBehaviorEvidencePublicContractHasNoPrivateAnalyzerSurface(t *testing.T) {
	for _, contract := range []reflect.Type{
		reflect.TypeOf(BehaviorEvidenceManifest{}),
		reflect.TypeOf(BehaviorEvidenceArtifact{}),
		reflect.TypeOf(BehaviorEvidenceReview{}),
		reflect.TypeOf(BehaviorEvidenceReceipt{}),
	} {
		for index := 0; index < contract.NumField(); index++ {
			field := strings.ToLower(contract.Field(index).Name)
			for _, forbidden := range []string{
				"path", "sqlite", "etw", "fibratus", "capture", "secret",
				"token", "dsn", "command", "argument", "payload", "content",
			} {
				if strings.Contains(field, forbidden) {
					t.Fatalf("%s exposes forbidden field %s", contract, field)
				}
			}
		}
	}
}

func behaviorEvidenceFixture(t *testing.T) BehaviorEvidenceManifest {
	t.Helper()
	manifest := BehaviorEvidenceManifest{
		Schema:          BehaviorEvidenceManifestSchema,
		ContractVersion: BehaviorEvidenceContractVersion,
		AnalysisRef:     "analysis-001",
		RequestRef:      "request-001",
		SubjectRef:      "subject-001",
		Purpose:         "clean_room_reimplementation",
		Producer: BehaviorEvidenceProducer{
			Name:    "connector",
			Version: "1.2.3",
		},
		CreatedAt:      "2026-07-29T17:00:00Z",
		Classification: "shareable_redacted",
		Review: BehaviorEvidenceReview{
			Status:               "approved",
			ReviewerRef:          "reviewer-001",
			ReviewedAt:           "2026-07-29T17:01:00Z",
			ReviewedDigestSHA256: strings.Repeat("a", 64),
			Scope:                "shareable_redacted_evidence",
		},
		Artifacts: []BehaviorEvidenceArtifact{{
			ArtifactRef:    "artifact-001",
			Kind:           "behavior_specification",
			Schema:         "application.behavior-specification.v1",
			MediaType:      "application/json",
			SizeBytes:      42,
			SHA256:         strings.Repeat("b", 64),
			Classification: "shareable_redacted",
			ReviewStatus:   "approved",
		}},
	}
	digest, err := BehaviorEvidenceManifestDigest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifest.ManifestDigest = digest
	return manifest
}
