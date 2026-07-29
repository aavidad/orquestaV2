package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

type behaviorEvidenceHTTPClock struct {
	now time.Time
}

func (clock behaviorEvidenceHTTPClock) Now() time.Time { return clock.now }

type behaviorEvidenceHTTPStore struct {
	mu       sync.Mutex
	receipts map[string]application.BehaviorEvidenceReceipt
}

func (store *behaviorEvidenceHTTPStore) AcceptBehaviorEvidenceManifest(
	_ context.Context,
	manifest application.BehaviorEvidenceManifest,
	acceptedAt time.Time,
) (application.BehaviorEvidenceReceipt, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if receipt, found := store.receipts[manifest.ManifestDigest]; found {
		return receipt, nil
	}
	receipt := application.BehaviorEvidenceReceipt{
		Schema:         application.BehaviorEvidenceReceiptSchema,
		ReceiptRef:     "receipt-" + manifest.ManifestDigest,
		ManifestDigest: manifest.ManifestDigest,
		AcceptedAt:     acceptedAt.UTC().Format(time.RFC3339Nano),
	}
	store.receipts[manifest.ManifestDigest] = receipt
	return receipt, nil
}

type behaviorEvidenceIdentity struct {
	mu        sync.Mutex
	calls     int
	principal identity.Principal
	err       error
}

func (provider *behaviorEvidenceIdentity) Principal(context.Context) (identity.Principal, error) {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	provider.calls++
	return provider.principal, provider.err
}

func TestBehaviorEvidenceHTTPAcceptsExactManifestAndReplaysExactReceipt(t *testing.T) {
	handler, identityProvider := behaviorEvidenceHTTPHandler(t)
	manifest := behaviorEvidenceHTTPManifest(t)
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	first := postBehaviorEvidence(t, handler, body)
	second := postBehaviorEvidence(t, handler, body)
	if first.Code != http.StatusOK || second.Code != http.StatusOK ||
		!bytes.Equal(first.Body.Bytes(), second.Body.Bytes()) {
		t.Fatalf(
			"first=%d %s second=%d %s",
			first.Code,
			first.Body.String(),
			second.Code,
			second.Body.String(),
		)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(first.Body.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"schema", "receipt_ref", "manifest_digest", "accepted_at",
	} {
		if len(fields[required]) == 0 {
			t.Fatalf("receipt lacks %s: %v", required, fields)
		}
	}
	if len(fields) != 4 {
		t.Fatalf("receipt exposes extra fields: %v", fields)
	}
	if string(fields["schema"]) !=
		`"`+application.BehaviorEvidenceReceiptSchema+`"` {
		t.Fatalf("schema=%s", fields["schema"])
	}
	identityProvider.mu.Lock()
	defer identityProvider.mu.Unlock()
	if identityProvider.calls != 2 {
		t.Fatalf("identity calls=%d", identityProvider.calls)
	}
}

func TestBehaviorEvidenceHTTPRejectsUnknownPrivateFieldsBeforeAuthentication(t *testing.T) {
	handler, identityProvider := behaviorEvidenceHTTPHandler(t)
	manifest := behaviorEvidenceHTTPManifest(t)
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	body = append(bytes.TrimSuffix(body, []byte("}")),
		[]byte(`,"sqlite_path":"C:\\private\\evidence.db","secret":"must-not-leak"}`)...)
	response := postBehaviorEvidence(t, handler, body)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "private") ||
		strings.Contains(response.Body.String(), "must-not-leak") {
		t.Fatalf("failure leaked request: %s", response.Body.String())
	}
	identityProvider.mu.Lock()
	defer identityProvider.mu.Unlock()
	if identityProvider.calls != 0 {
		t.Fatalf("invalid body reached identity: calls=%d", identityProvider.calls)
	}
}

func TestBehaviorEvidenceHTTPRejectsDigestOrApprovalDriftBeforeAuthentication(t *testing.T) {
	for _, mutate := range []func(*application.BehaviorEvidenceManifest){
		func(manifest *application.BehaviorEvidenceManifest) {
			manifest.ManifestDigest = strings.Repeat("c", 64)
		},
		func(manifest *application.BehaviorEvidenceManifest) {
			manifest.Classification = "restricted"
		},
		func(manifest *application.BehaviorEvidenceManifest) {
			manifest.Review.Status = "pending"
		},
		func(manifest *application.BehaviorEvidenceManifest) {
			manifest.Artifacts[0].ReviewStatus = "rejected"
		},
	} {
		handler, identityProvider := behaviorEvidenceHTTPHandler(t)
		manifest := behaviorEvidenceHTTPManifest(t)
		mutate(&manifest)
		body, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		response := postBehaviorEvidence(t, handler, body)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("code=%d body=%s", response.Code, response.Body.String())
		}
		identityProvider.mu.Lock()
		calls := identityProvider.calls
		identityProvider.mu.Unlock()
		if calls != 0 {
			t.Fatalf("invalid manifest reached identity: calls=%d", calls)
		}
	}
}

func TestBehaviorEvidenceHTTPRequiresAuthenticatedPrincipal(t *testing.T) {
	store := &behaviorEvidenceHTTPStore{
		receipts: make(map[string]application.BehaviorEvidenceReceipt),
	}
	service, err := application.NewBehaviorEvidenceIngestionService(
		store,
		behaviorEvidenceHTTPClock{now: time.Now()},
	)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewBehaviorEvidenceHandler(BehaviorEvidenceConfig{
		Ingestor: service,
		Identity: &behaviorEvidenceIdentity{err: errors.New("missing")},
	})
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(behaviorEvidenceHTTPManifest(t))
	if err != nil {
		t.Fatal(err)
	}
	response := postBehaviorEvidence(t, handler, body)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d body=%s", response.Code, response.Body.String())
	}
}

func TestBehaviorEvidenceHTTPBoundsBodyAndRoute(t *testing.T) {
	handler, _ := behaviorEvidenceHTTPHandler(t)
	oversized := bytes.Repeat([]byte("x"), int(BehaviorEvidenceMaxRequestBytes)+1)
	response := postBehaviorEvidence(t, handler, oversized)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("oversized code=%d", response.Code)
	}
	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodGet, BehaviorEvidenceManifestPath, nil),
		httptest.NewRequest(http.MethodPost, BehaviorEvidenceManifestPath+"/other", nil),
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("method=%s path=%s code=%d", request.Method, request.URL.Path, response.Code)
		}
	}
}

func behaviorEvidenceHTTPHandler(
	t *testing.T,
) (*BehaviorEvidenceHandler, *behaviorEvidenceIdentity) {
	t.Helper()
	store := &behaviorEvidenceHTTPStore{
		receipts: make(map[string]application.BehaviorEvidenceReceipt),
	}
	service, err := application.NewBehaviorEvidenceIngestionService(
		store,
		behaviorEvidenceHTTPClock{
			now: time.Date(2026, 7, 29, 17, 2, 3, 0, time.UTC),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	principalRef, err := identity.NewPrincipalRef("principal-behavior-evidence")
	if err != nil {
		t.Fatal(err)
	}
	actorRef, err := goal.NewActorRef("actor-behavior-evidence")
	if err != nil {
		t.Fatal(err)
	}
	principal, err := identity.NewPrincipal(
		principalRef,
		actorRef,
		identity.PrincipalKindService,
		"local_token",
	)
	if err != nil {
		t.Fatal(err)
	}
	identityProvider := &behaviorEvidenceIdentity{principal: principal}
	handler, err := NewBehaviorEvidenceHandler(BehaviorEvidenceConfig{
		Ingestor: service,
		Identity: identityProvider,
	})
	if err != nil {
		t.Fatal(err)
	}
	return handler, identityProvider
}

func behaviorEvidenceHTTPManifest(
	t *testing.T,
) application.BehaviorEvidenceManifest {
	t.Helper()
	manifest := application.BehaviorEvidenceManifest{
		Schema:          application.BehaviorEvidenceManifestSchema,
		ContractVersion: application.BehaviorEvidenceContractVersion,
		AnalysisRef:     "analysis-001",
		RequestRef:      "request-001",
		SubjectRef:      "subject-001",
		Purpose:         "clean_room_reimplementation",
		Producer: application.BehaviorEvidenceProducer{
			Name: "connector", Version: "1.2.3",
		},
		CreatedAt:      "2026-07-29T17:00:00Z",
		Classification: "shareable_redacted",
		Review: application.BehaviorEvidenceReview{
			Status: "approved", ReviewerRef: "reviewer-001",
			ReviewedAt:           "2026-07-29T17:01:00Z",
			ReviewedDigestSHA256: strings.Repeat("a", 64),
			Scope:                "shareable_redacted_evidence",
		},
		Artifacts: []application.BehaviorEvidenceArtifact{{
			ArtifactRef: "artifact-001", Kind: "behavior_specification",
			Schema: "application.behavior-specification.v1", MediaType: "application/json",
			SizeBytes: 42, SHA256: strings.Repeat("b", 64),
			Classification: "shareable_redacted", ReviewStatus: "approved",
		}},
	}
	digest, err := application.BehaviorEvidenceManifestDigest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifest.ManifestDigest = digest
	return manifest
}

func postBehaviorEvidence(
	t *testing.T,
	handler http.Handler,
	body []byte,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(
		http.MethodPost,
		BehaviorEvidenceManifestPath,
		bytes.NewReader(body),
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
