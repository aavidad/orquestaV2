package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/intake"
)

func TestAppAnalysisServiceResolvesProjectArtifactWithoutCASMediaTypeAndReturnsDefensiveCopies(t *testing.T) {
	ctx := context.Background()
	fixture := newAppAnalysisTestFixture(t, AppAnalysisPurposeCleanRoomReimplementation)
	result, err := fixture.service.AttachAppAnalysis(ctx, fixture.envelope)
	if err != nil || result.Disposition != AppAnalysisAttachmentCreated {
		t.Fatalf("AttachAppAnalysis() result=%+v err=%v", result, err)
	}
	if fixture.intakes.verifyCount() != 1 ||
		fixture.artifacts.resolveCount() != 1 || fixture.reviews.verifyCount() != 1 {
		t.Fatalf("port calls intake=%d resolver=%d reviews=%d, want 1/1/1",
			fixture.intakes.verifyCount(), fixture.artifacts.resolveCount(),
			fixture.reviews.verifyCount())
	}
	if result.Record.Analysis.Ref() == "" ||
		result.Record.Receipt.AnalysisRef != result.Record.Analysis.Ref() ||
		result.Record.Analysis.Review().ReviewerRef != fixture.reviewerRef {
		t.Fatalf("invalid authenticated result: %+v", result.Record)
	}
	manifest := result.Record.Analysis.Manifest()
	if manifest.Attachments[0].ProvenanceReceiptRef == "" ||
		!validAppAnalysisDigest(manifest.Attachments[0].ProvenanceReceiptDigest) ||
		result.Record.Receipt.IntakeReceiptRef == "" ||
		!validAppAnalysisDigest(result.Record.Receipt.IntakeSnapshotDigest) ||
		result.Record.Receipt.VerifiedReviewDigest !=
			result.Record.Analysis.VerifiedReviewDigest() {
		t.Fatalf("durable causal evidence missing: %+v", result.Record.Receipt)
	}

	wantMediaType := "application/json; charset=UTF-8"
	fixture.envelope.Manifest.Attachments[0].MediaType = "application/x-mutated"
	firstCopy := result.Record.Analysis.Manifest()
	firstCopy.Attachments[0].MediaType = "application/x-mutated-output"
	secondCopy := result.Record.Analysis.Manifest()
	if secondCopy.Attachments[0].MediaType != wantMediaType {
		t.Fatalf("manifest alias or media normalization lost: %q", secondCopy.Attachments[0].MediaType)
	}

	getRequestRef := "request:app-analysis-get"
	record, err := fixture.service.GetAppAnalysisAttachment(
		ctx,
		GetAppAnalysisAttachmentRequest{
			RequestRef: getRequestRef, ActorRef: fixture.actor,
			ProjectRef: fixture.project, AnalysisRef: result.Record.Analysis.Ref(),
			AuthorizationReceipt: appAnalysisAuthorization(
				t, getRequestRef, AppAnalysisOperationGet, fixture.principal,
				fixture.project, identity.PermissionArtifactsRead,
				string(result.Record.Analysis.Ref()),
			),
		},
	)
	if err != nil || record.Analysis.Ref() != result.Record.Analysis.Ref() {
		t.Fatalf("GetAppAnalysisAttachment() record=%+v err=%v", record, err)
	}
	got := record.Analysis.Manifest()
	got.Attachments[0].Digest = appAnalysisTestDigest("mutated")
	againRequestRef := getRequestRef + "-again"
	again, err := fixture.service.GetAppAnalysisAttachment(
		ctx,
		GetAppAnalysisAttachmentRequest{
			RequestRef: againRequestRef, ActorRef: fixture.actor,
			ProjectRef: fixture.project, AnalysisRef: result.Record.Analysis.Ref(),
			AuthorizationReceipt: appAnalysisAuthorization(
				t, againRequestRef, AppAnalysisOperationGet, fixture.principal,
				fixture.project, identity.PermissionArtifactsRead,
				string(result.Record.Analysis.Ref()),
			),
		},
	)
	if err != nil || again.Analysis.Manifest().Attachments[0].Digest == got.Attachments[0].Digest {
		t.Fatalf("stored manifest alias leaked: record=%+v err=%v", again, err)
	}
}

func TestAppAnalysisServiceExactReplayPrecedesReviewAndArtifactResolution(t *testing.T) {
	fixture := newAppAnalysisTestFixture(t, AppAnalysisPurposeCleanRoomReimplementation)
	first, err := fixture.service.AttachAppAnalysis(context.Background(), fixture.envelope)
	if err != nil {
		t.Fatalf("first attach: %v", err)
	}
	fixture.artifacts.reset()
	fixture.reviews.reset()
	fixture.intakes.reset()
	fixture.artifacts.err = errors.New("artifact resolver must not be read")
	fixture.reviews.err = errors.New("review verifier must not be read")

	replayed, err := fixture.service.AttachAppAnalysis(context.Background(), fixture.envelope)
	if err != nil || replayed.Disposition != AppAnalysisAttachmentReplayed ||
		replayed.Record.Receipt != first.Record.Receipt {
		t.Fatalf("replay result=%+v err=%v", replayed, err)
	}
	if fixture.intakes.verifyCount() != 0 ||
		fixture.artifacts.resolveCount() != 0 || fixture.reviews.verifyCount() != 0 {
		t.Fatalf("exact replay crossed external ports: intake=%d artifacts=%d reviews=%d",
			fixture.intakes.verifyCount(), fixture.artifacts.resolveCount(),
			fixture.reviews.verifyCount())
	}
}

func TestAppAnalysisServiceSameRequestDivergenceConflictsBeforeExternalPorts(t *testing.T) {
	fixture := newAppAnalysisTestFixture(t, AppAnalysisPurposeCleanRoomReimplementation)
	if _, err := fixture.service.AttachAppAnalysis(context.Background(), fixture.envelope); err != nil {
		t.Fatalf("first attach: %v", err)
	}
	changed := fixture.envelope
	changed.Manifest = cloneAppAnalysisManifest(fixture.envelope.Manifest)
	changed.Manifest.Attachments[0] = appAnalysisDescriptor(
		t,
		"application/json",
		[]byte(`{"different":"analysis"}`),
	)
	fixture.artifacts.reset()
	fixture.reviews.reset()
	fixture.intakes.reset()

	_, err := fixture.service.AttachAppAnalysis(context.Background(), changed)
	if !IsStateError(err, StateConflict) {
		t.Fatalf("changed payload error=%v, want state.conflict", err)
	}
	if fixture.intakes.verifyCount() != 0 ||
		fixture.artifacts.resolveCount() != 0 || fixture.reviews.verifyCount() != 0 {
		t.Fatalf("divergence crossed external ports: intake=%d artifacts=%d reviews=%d",
			fixture.intakes.verifyCount(), fixture.artifacts.resolveCount(),
			fixture.reviews.verifyCount())
	}
}

func TestAppAnalysisServiceDifferentRequestSameContentIsDeduplicated(t *testing.T) {
	fixture := newAppAnalysisTestFixture(t, AppAnalysisPurposeCleanRoomReimplementation)
	first, err := fixture.service.AttachAppAnalysis(context.Background(), fixture.envelope)
	if err != nil || first.Disposition != AppAnalysisAttachmentCreated {
		t.Fatalf("first attach result=%+v err=%v", first, err)
	}

	secondEnvelope := fixture.envelope
	secondEnvelope.RequestRef = "request:app-analysis-second"
	secondEnvelope.AuthorizationReceipt = appAnalysisAuthorization(
		t, secondEnvelope.RequestRef, AppAnalysisOperationAttach,
		fixture.principal, fixture.project, identity.PermissionGoalsCreate,
		fixture.project.String(),
	)
	second, err := fixture.service.AttachAppAnalysis(context.Background(), secondEnvelope)
	if err != nil || second.Disposition != AppAnalysisAttachmentDeduplicated {
		t.Fatalf("second attach result=%+v err=%v", second, err)
	}
	if second.Record.Analysis.Ref() != first.Record.Analysis.Ref() {
		t.Fatalf("analysis not deduplicated: %s != %s",
			second.Record.Analysis.Ref(), first.Record.Analysis.Ref())
	}
	if second.Record.Receipt.Ref == first.Record.Receipt.Ref ||
		second.Record.Receipt.RequestRef == first.Record.Receipt.RequestRef {
		t.Fatalf("request receipt deduplicated: first=%+v second=%+v",
			first.Record.Receipt, second.Record.Receipt)
	}
	canonical := fixture.store.canonical(
		fixture.actor,
		fixture.project,
		first.Record.Analysis.Ref(),
	)
	if canonical.Receipt.Ref != first.Record.Receipt.Ref {
		t.Fatalf("dedup replaced canonical record: %+v", canonical.Receipt)
	}
}

func TestAppAnalysisServiceValidatesTopLevelCausalityDescriptorsAndClassification(t *testing.T) {
	tests := []struct {
		name   string
		max    int64
		mutate func(*appAnalysisTestFixture)
	}{
		{"envelope schema", 1024, func(f *appAnalysisTestFixture) {
			f.envelope.Schema = "orquesta.app-analysis.envelope.v2"
		}},
		{"manifest schema", 1024, func(f *appAnalysisTestFixture) {
			f.envelope.Manifest.Schema = "orquesta.app-analysis.manifest.v2"
		}},
		{"subject", 1024, func(f *appAnalysisTestFixture) {
			f.envelope.Manifest.SubjectRef = ""
		}},
		{"provider", 1024, func(f *appAnalysisTestFixture) {
			f.envelope.Manifest.ProviderRef = ""
		}},
		{"producer", 1024, func(f *appAnalysisTestFixture) {
			f.envelope.Manifest.ProducerRef = ""
		}},
		{"producer version", 1024, func(f *appAnalysisTestFixture) {
			f.envelope.Manifest.ProducerVersion = ""
		}},
		{"intake", 1024, func(f *appAnalysisTestFixture) {
			f.envelope.Manifest.IntakeRef = "other:state"
		}},
		{"revision", 1024, func(f *appAnalysisTestFixture) {
			f.envelope.Manifest.ExpectedRevision = 0
		}},
		{"purpose", 1024, func(f *appAnalysisTestFixture) {
			f.envelope.Manifest.Purpose = "migration"
		}},
		{"review receipt", 1024, func(f *appAnalysisTestFixture) {
			f.envelope.Manifest.ReviewReceiptRef = ""
		}},
		{"kind", 1024, func(f *appAnalysisTestFixture) {
			f.envelope.Manifest.Attachments[0].Kind = "binary"
		}},
		{"digest", 1024, func(f *appAnalysisTestFixture) {
			f.envelope.Manifest.Attachments[0].Digest = "sha256:not-canonical"
		}},
		{"declared size", 1024, func(f *appAnalysisTestFixture) {
			f.envelope.Manifest.Attachments[0].Size = 0
		}},
		{"size limit", 4, func(*appAnalysisTestFixture) {}},
		{"media type", 1024, func(f *appAnalysisTestFixture) {
			f.envelope.Manifest.Attachments[0].MediaType = "not-a-media-type"
		}},
		{"classification", 1024, func(f *appAnalysisTestFixture) {
			f.envelope.Manifest.Attachments[0].Classification = "private"
		}},
		{"clean room repository", 1024, func(f *appAnalysisTestFixture) {
			f.envelope.Manifest.RepositoryRef =
				appAnalysisRepositoryRef(t, "repository:forbidden")
		}},
		{"clean room tree", 1024, func(f *appAnalysisTestFixture) {
			f.envelope.Manifest.TreeOID = appAnalysisTreeOID('a')
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newAppAnalysisTestFixture(
				t,
				AppAnalysisPurposeCleanRoomReimplementation,
			)
			fixture.service, _ = NewAppAnalysisService(
				fixture.intakes,
				fixture.artifacts,
				fixture.reviews,
				fixture.store,
				test.max,
			)
			test.mutate(fixture)
			_, err := fixture.service.AttachAppAnalysis(
				context.Background(),
				fixture.envelope,
			)
			if !errors.Is(err, ErrAppAnalysisInvalid) {
				t.Fatalf("error=%v, want application.app_analysis_invalid", err)
			}
			if fixture.store.attachCount() != 0 {
				t.Fatal("invalid analysis reached attachment store")
			}
		})
	}
}

func TestAppAnalysisPurposeFencesRepositoryAndTreeAtAnalysisLevel(t *testing.T) {
	refactor := newAppAnalysisTestFixture(t, AppAnalysisPurposeRefactorExisting)
	if _, err := refactor.service.AttachAppAnalysis(context.Background(), refactor.envelope); err != nil {
		t.Fatalf("valid refactor rejected: %v", err)
	}

	for _, test := range []struct {
		name   string
		mutate func(*AppAnalysisManifest)
	}{
		{"repository required", func(m *AppAnalysisManifest) {
			m.RepositoryRef = identity.RepositoryRef{}
		}},
		{"tree required", func(m *AppAnalysisManifest) { m.TreeOID = "" }},
		{"tree must be oid", func(m *AppAnalysisManifest) { m.TreeOID = "HEAD" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newAppAnalysisTestFixture(t, AppAnalysisPurposeRefactorExisting)
			test.mutate(&fixture.envelope.Manifest)
			if _, err := fixture.service.AttachAppAnalysis(
				context.Background(),
				fixture.envelope,
			); !errors.Is(err, ErrAppAnalysisInvalid) {
				t.Fatalf("invalid refactor error=%v", err)
			}
		})
	}
}

func TestAppAnalysisReviewReceiptBindsCompleteCanonicalManifest(t *testing.T) {
	tests := []struct {
		name         string
		basePurpose  AppAnalysisPurpose
		mutate       func(*appAnalysisTestFixture)
		wantVerifier bool
	}{
		{
			name: "purpose", basePurpose: AppAnalysisPurposeRefactorExisting,
			mutate: func(f *appAnalysisTestFixture) {
				f.envelope.Manifest.Purpose = AppAnalysisPurposeCleanRoomReimplementation
				f.envelope.Manifest.RepositoryRef = identity.RepositoryRef{}
				f.envelope.Manifest.TreeOID = ""
			},
			wantVerifier: true,
		},
		{
			name: "classification", basePurpose: AppAnalysisPurposeCleanRoomReimplementation,
			mutate: func(f *appAnalysisTestFixture) {
				f.envelope.Manifest.Attachments[0].Classification = "private"
			},
			wantVerifier: false,
		},
		{
			name: "repository", basePurpose: AppAnalysisPurposeRefactorExisting,
			mutate: func(f *appAnalysisTestFixture) {
				f.envelope.Manifest.RepositoryRef =
					appAnalysisRepositoryRef(t, "repository:substituted")
			},
			wantVerifier: true,
		},
		{
			name: "tree", basePurpose: AppAnalysisPurposeRefactorExisting,
			mutate: func(f *appAnalysisTestFixture) {
				f.envelope.Manifest.TreeOID = appAnalysisTreeOID('b')
			},
			wantVerifier: true,
		},
		{
			name: "artifact", basePurpose: AppAnalysisPurposeCleanRoomReimplementation,
			mutate: func(f *appAnalysisTestFixture) {
				f.envelope.Manifest.Attachments[0] = appAnalysisDescriptor(
					t,
					"application/json",
					[]byte(`{"substituted":true}`),
				)
			},
			wantVerifier: true,
		},
		{
			name: "canonical media type", basePurpose: AppAnalysisPurposeCleanRoomReimplementation,
			mutate: func(f *appAnalysisTestFixture) {
				f.envelope.Manifest.Attachments[0].MediaType = "text/plain"
			},
			wantVerifier: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newAppAnalysisTestFixture(t, test.basePurpose)
			test.mutate(fixture)
			_, err := fixture.service.AttachAppAnalysis(
				context.Background(),
				fixture.envelope,
			)
			if test.wantVerifier {
				if !errors.Is(err, ErrForbidden) || fixture.reviews.verifyCount() != 1 {
					t.Fatalf("substitution error=%v verifier_calls=%d, want forbidden/1",
						err, fixture.reviews.verifyCount())
				}
			} else if !errors.Is(err, ErrAppAnalysisInvalid) ||
				fixture.reviews.verifyCount() != 0 {
				t.Fatalf("classification substitution error=%v verifier_calls=%d",
					err, fixture.reviews.verifyCount())
			}
			if fixture.artifacts.resolveCount() != 0 || fixture.store.attachCount() != 0 {
				t.Fatalf("substitution crossed later ports: resolve=%d attach=%d",
					fixture.artifacts.resolveCount(), fixture.store.attachCount())
			}
		})
	}
}

func TestAppAnalysisServiceAcceptsOnlyAuthenticatedAcceptedReviewDecision(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*AppAnalysisVerifiedReview)
	}{
		{"receipt", func(r *AppAnalysisVerifiedReview) {
			r.ReceiptRef = "analysis-review-receipt:other"
		}},
		{"actor", func(r *AppAnalysisVerifiedReview) {
			r.ActorRef = goal.ActorRef{}
		}},
		{"project", func(r *AppAnalysisVerifiedReview) {
			r.ProjectRef = goal.ProjectRef{}
		}},
		{"subject", func(r *AppAnalysisVerifiedReview) {
			r.SubjectRef = "application:other"
		}},
		{"provider", func(r *AppAnalysisVerifiedReview) {
			r.ProviderRef = "provider:other"
		}},
		{"producer", func(r *AppAnalysisVerifiedReview) {
			r.ProducerRef = "producer:other"
		}},
		{"producer version", func(r *AppAnalysisVerifiedReview) {
			r.ProducerVersion = "other"
		}},
		{"reviewer", func(r *AppAnalysisVerifiedReview) {
			r.ReviewerRef = identity.PrincipalRef{}
		}},
		{"decision", func(r *AppAnalysisVerifiedReview) {
			r.Decision = "pending"
		}},
		{"policy", func(r *AppAnalysisVerifiedReview) {
			r.PolicyRef = ""
		}},
		{"subject digest", func(r *AppAnalysisVerifiedReview) {
			r.SubjectDigest = appAnalysisTestDigest("other")
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newAppAnalysisTestFixture(
				t,
				AppAnalysisPurposeCleanRoomReimplementation,
			)
			fixture.reviews.mutate = test.mutate
			_, err := fixture.service.AttachAppAnalysis(
				context.Background(),
				fixture.envelope,
			)
			if test.name == "actor" || test.name == "project" {
				if !errors.Is(err, ErrForbidden) {
					t.Fatalf("review substitution error=%v, want forbidden", err)
				}
			} else if !errors.Is(err, ErrAppAnalysisInvalid) {
				t.Fatalf("review substitution error=%v, want invalid", err)
			}
			if fixture.artifacts.resolveCount() != 0 ||
				fixture.store.attachCount() != 0 {
				t.Fatalf("invalid review crossed later ports: resolve=%d attach=%d",
					fixture.artifacts.resolveCount(), fixture.store.attachCount())
			}
		})
	}
}

func TestAppAnalysisArtifactResolverDeniesCrossProjectEvenWithValidAuthorityAndReview(t *testing.T) {
	fixture := newAppAnalysisTestFixture(t, AppAnalysisPurposeCleanRoomReimplementation)
	otherProject, err := goal.NewProjectRef("project:other")
	if err != nil {
		t.Fatal(err)
	}
	fixture.envelope.ProjectRef = otherProject
	fixture.envelope.AuthorizationReceipt = appAnalysisAuthorization(
		t, fixture.envelope.RequestRef, AppAnalysisOperationAttach,
		fixture.principal, otherProject, identity.PermissionGoalsCreate,
		otherProject.String(),
	)
	fixture.reviews.allow(t, fixture.actor, otherProject, fixture.envelope.Manifest)
	fixture.intakes.allow(
		fixture.actor,
		otherProject,
		fixture.envelope.Manifest.IntakeRef,
		fixture.envelope.Manifest.ExpectedRevision,
	)

	_, err = fixture.service.AttachAppAnalysis(context.Background(), fixture.envelope)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("cross-project artifact error=%v, want forbidden", err)
	}
	if fixture.reviews.verifyCount() != 1 || fixture.artifacts.resolveCount() != 1 ||
		fixture.store.attachCount() != 0 {
		t.Fatalf("cross-project flow reviews=%d resolve=%d attach=%d",
			fixture.reviews.verifyCount(), fixture.artifacts.resolveCount(),
			fixture.store.attachCount())
	}
}

func TestAppAnalysisServiceRejectsArtifactContentAndProvenanceSubstitution(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*AppAnalysisArtifactResolution)
	}{
		{"content", func(r *AppAnalysisArtifactResolution) {
			r.Content = []byte("tampered")
		}},
		{"receipt ref", func(r *AppAnalysisArtifactResolution) {
			r.Provenance.ReceiptRef = ""
		}},
		{"actor", func(r *AppAnalysisArtifactResolution) {
			r.Provenance.ActorRef = goal.ActorRef{}
		}},
		{"project", func(r *AppAnalysisArtifactResolution) {
			r.Provenance.ProjectRef = goal.ProjectRef{}
		}},
		{"subject", func(r *AppAnalysisArtifactResolution) {
			r.Provenance.SubjectRef = "app:other"
		}},
		{"provider", func(r *AppAnalysisArtifactResolution) {
			r.Provenance.ProviderRef = "provider:other"
		}},
		{"artifact", func(r *AppAnalysisArtifactResolution) {
			ref, _ := goal.NewArtifactRef("artifact:sha256:" + appAnalysisTestDigest("other"))
			r.Provenance.ArtifactRef = ref
		}},
		{"descriptor", func(r *AppAnalysisArtifactResolution) {
			r.Provenance.DescriptorDigest = appAnalysisTestDigest("other")
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newAppAnalysisTestFixture(
				t,
				AppAnalysisPurposeCleanRoomReimplementation,
			)
			fixture.artifacts.mutate = test.mutate
			_, err := fixture.service.AttachAppAnalysis(
				context.Background(),
				fixture.envelope,
			)
			if !errors.Is(err, ErrAppAnalysisInvalid) ||
				fixture.store.attachCount() != 0 {
				t.Fatalf("substitution error=%v attach=%d",
					err, fixture.store.attachCount())
			}
		})
	}
}

func TestAppAnalysisServiceRejectsCrossScopeAuthorityBeforeAnyPort(t *testing.T) {
	fixture := newAppAnalysisTestFixture(t, AppAnalysisPurposeCleanRoomReimplementation)
	otherProject, err := goal.NewProjectRef("project:other")
	if err != nil {
		t.Fatal(err)
	}
	fixture.envelope.AuthorizationReceipt = appAnalysisAuthorization(
		t, fixture.envelope.RequestRef, AppAnalysisOperationAttach,
		fixture.principal, otherProject, identity.PermissionGoalsCreate,
		otherProject.String(),
	)
	_, err = fixture.service.AttachAppAnalysis(context.Background(), fixture.envelope)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("cross-scope authority error=%v, want forbidden", err)
	}
	if fixture.store.replayCount() != 0 || fixture.artifacts.resolveCount() != 0 ||
		fixture.reviews.verifyCount() != 0 {
		t.Fatalf("denied request crossed ports: replay=%d resolve=%d review=%d",
			fixture.store.replayCount(), fixture.artifacts.resolveCount(),
			fixture.reviews.verifyCount())
	}
}

func TestAppAnalysisServiceFencesIntakeBeforeReviewArtifactAndPersistence(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*appAnalysisTestFixture)
		want  error
	}{
		{"missing", func(f *appAnalysisTestFixture) {
			f.intakes.err = &StateError{Code: StateNotFound}
		}, nil},
		{"cross project", func(f *appAnalysisTestFixture) {
			f.intakes.mutate = func(e *AppAnalysisVerifiedIntake) {
				e.ProjectRef, _ = goal.NewProjectRef("project:other")
			}
		}, ErrForbidden},
		{"stale", func(f *appAnalysisTestFixture) {
			f.intakes.mutate = func(e *AppAnalysisVerifiedIntake) { e.Revision++ }
		}, ErrAppAnalysisInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newAppAnalysisTestFixture(
				t,
				AppAnalysisPurposeCleanRoomReimplementation,
			)
			test.setup(fixture)
			_, err := fixture.service.AttachAppAnalysis(
				context.Background(),
				fixture.envelope,
			)
			if test.want != nil && !errors.Is(err, test.want) {
				t.Fatalf("error=%v, want %v", err, test.want)
			}
			if err == nil || fixture.intakes.verifyCount() != 1 ||
				fixture.reviews.verifyCount() != 0 ||
				fixture.artifacts.resolveCount() != 0 ||
				fixture.store.attachCount() != 0 {
				t.Fatalf("err=%v intake=%d review=%d artifact=%d attach=%d",
					err, fixture.intakes.verifyCount(), fixture.reviews.verifyCount(),
					fixture.artifacts.resolveCount(), fixture.store.attachCount())
			}
		})
	}
}

func TestAppAnalysisReplayAndGetFailClosedOnEvidenceTamper(t *testing.T) {
	t.Run("replay receipt", func(t *testing.T) {
		fixture := newAppAnalysisTestFixture(t, AppAnalysisPurposeCleanRoomReimplementation)
		if _, err := fixture.service.AttachAppAnalysis(context.Background(), fixture.envelope); err != nil {
			t.Fatal(err)
		}
		key := appAnalysisRequestKey(
			fixture.actor,
			fixture.project,
			fixture.envelope.RequestRef,
		)
		fixture.store.mu.Lock()
		record := fixture.store.requests[key]
		record.Receipt.IntakeSnapshotDigest = appAnalysisTestDigest("tampered")
		fixture.store.requests[key] = record
		fixture.store.mu.Unlock()
		if _, err := fixture.service.AttachAppAnalysis(
			context.Background(),
			fixture.envelope,
		); !IsStateError(err, StateConflict) {
			t.Fatalf("replay tamper error=%v", err)
		}
	})
	t.Run("get record", func(t *testing.T) {
		fixture := newAppAnalysisTestFixture(t, AppAnalysisPurposeCleanRoomReimplementation)
		result, err := fixture.service.AttachAppAnalysis(context.Background(), fixture.envelope)
		if err != nil {
			t.Fatal(err)
		}
		key := appAnalysisKey(fixture.actor, fixture.project, result.Record.Analysis.Ref())
		fixture.store.mu.Lock()
		record := fixture.store.analyses[key]
		record.Receipt.VerifiedReviewDigest = appAnalysisTestDigest("tampered")
		fixture.store.analyses[key] = record
		fixture.store.mu.Unlock()
		requestRef := "request:app-analysis-tampered-get"
		_, err = fixture.service.GetAppAnalysisAttachment(
			context.Background(),
			GetAppAnalysisAttachmentRequest{
				RequestRef: requestRef, ActorRef: fixture.actor,
				ProjectRef: fixture.project, AnalysisRef: result.Record.Analysis.Ref(),
				AuthorizationReceipt: appAnalysisAuthorization(
					t, requestRef, AppAnalysisOperationGet, fixture.principal,
					fixture.project, identity.PermissionArtifactsRead,
					string(result.Record.Analysis.Ref()),
				),
			},
		)
		if !IsStateError(err, StateConflict) {
			t.Fatalf("get tamper error=%v", err)
		}
	})
}

func TestAppAnalysisServiceConcurrentDedupReportsExplicitDisposition(t *testing.T) {
	const workers = 16
	fixture := newAppAnalysisTestFixture(t, AppAnalysisPurposeCleanRoomReimplementation)
	envelopes := make([]AppAnalysisEnvelope, workers)
	for index := range envelopes {
		envelopes[index] = fixture.envelope
		envelopes[index].RequestRef = "request:app-analysis-race-" + string(rune('a'+index))
		envelopes[index].AuthorizationReceipt = appAnalysisAuthorization(
			t, envelopes[index].RequestRef, AppAnalysisOperationAttach,
			fixture.principal, fixture.project, identity.PermissionGoalsCreate,
			fixture.project.String(),
		)
	}

	results := make(chan AppAnalysisAttachmentResult, workers)
	errs := make(chan error, workers)
	for index := range envelopes {
		envelope := envelopes[index]
		go func() {
			result, err := fixture.service.AttachAppAnalysis(context.Background(), envelope)
			results <- result
			errs <- err
		}()
	}
	dispositions := map[AppAnalysisAttachmentDisposition]int{}
	refs := make(map[AppAnalysisRef]struct{})
	receipts := make(map[string]struct{})
	for range envelopes {
		result, err := <-results, <-errs
		if err != nil {
			t.Fatalf("concurrent attach: %v", err)
		}
		dispositions[result.Disposition]++
		refs[result.Record.Analysis.Ref()] = struct{}{}
		receipts[result.Record.Receipt.Ref] = struct{}{}
	}
	if dispositions[AppAnalysisAttachmentCreated] != 1 ||
		dispositions[AppAnalysisAttachmentDeduplicated] != workers-1 ||
		dispositions[AppAnalysisAttachmentReplayed] != 0 ||
		len(refs) != 1 || len(receipts) != workers {
		t.Fatalf("dispositions=%v refs=%d receipts=%d",
			dispositions, len(refs), len(receipts))
	}
}

type appAnalysisTestFixture struct {
	actor       goal.ActorRef
	project     goal.ProjectRef
	principal   identity.Principal
	reviewerRef identity.PrincipalRef
	intakes     *appAnalysisIntakeVerifier
	artifacts   *appAnalysisArtifactResolver
	reviews     *appAnalysisReviewVerifier
	store       *appAnalysisMemoryStore
	service     *AppAnalysisService
	envelope    AppAnalysisEnvelope
	content     []byte
}

func newAppAnalysisTestFixture(
	t *testing.T,
	purpose AppAnalysisPurpose,
) *appAnalysisTestFixture {
	t.Helper()
	actor, err := goal.NewActorRef("actor:app-analysis")
	if err != nil {
		t.Fatal(err)
	}
	project, err := goal.NewProjectRef("project:app-analysis")
	if err != nil {
		t.Fatal(err)
	}
	principalRef, err := identity.NewPrincipalRef("principal:app-analysis")
	if err != nil {
		t.Fatal(err)
	}
	principal, err := identity.NewPrincipal(
		principalRef,
		actor,
		identity.PrincipalKindHuman,
		"test",
	)
	if err != nil {
		t.Fatal(err)
	}
	reviewerRef, err := identity.NewPrincipalRef("principal:analysis-reviewer")
	if err != nil {
		t.Fatal(err)
	}
	content := []byte(`{"surface":"observable behavior"}`)
	manifest := AppAnalysisManifest{
		Schema:           AppAnalysisManifestSchema,
		SubjectRef:       "application:opaque-subject",
		ProviderRef:      "provider:external-analysis",
		ProducerRef:      "producer:analyzer",
		ProducerVersion:  "2026.07",
		IntakeRef:        intake.Ref("intake:app-analysis"),
		ExpectedRevision: 7,
		Purpose:          purpose,
		ReviewReceiptRef: "analysis-review-receipt:accepted",
		Attachments: []AppAnalysisAttachment{
			appAnalysisDescriptor(
				t,
				"Application/JSON; Charset=UTF-8",
				content,
			),
		},
	}
	if purpose == AppAnalysisPurposeRefactorExisting {
		manifest.RepositoryRef = appAnalysisRepositoryRef(
			t,
			"repository:opaque-existing-app",
		)
		manifest.TreeOID = appAnalysisTreeOID('a')
	}
	requestRef := "request:app-analysis-attach"
	envelope := AppAnalysisEnvelope{
		Schema: AppAnalysisEnvelopeSchema, RequestRef: requestRef,
		ActorRef: actor, ProjectRef: project, Manifest: manifest,
		AuthorizationReceipt: appAnalysisAuthorization(
			t, requestRef, AppAnalysisOperationAttach, principal,
			project, identity.PermissionGoalsCreate, project.String(),
		),
	}
	artifacts := newAppAnalysisArtifactResolver()
	intakes := newAppAnalysisIntakeVerifier()
	reviews := newAppAnalysisReviewVerifier(reviewerRef)
	intakes.allow(actor, project, manifest.IntakeRef, manifest.ExpectedRevision)
	reviews.allow(t, actor, project, manifest)
	artifacts.allow(t, actor, project, manifest, manifest.Attachments[0], content)
	store := newAppAnalysisMemoryStore()
	service, err := NewAppAnalysisService(intakes, artifacts, reviews, store, 1024)
	if err != nil {
		t.Fatal(err)
	}
	return &appAnalysisTestFixture{
		actor: actor, project: project, principal: principal,
		reviewerRef: reviewerRef, intakes: intakes,
		artifacts: artifacts, reviews: reviews,
		store: store, service: service, envelope: envelope,
		content: append([]byte(nil), content...),
	}
}

func appAnalysisDescriptor(
	t *testing.T,
	mediaType string,
	content []byte,
) AppAnalysisAttachment {
	t.Helper()
	digest := sha256.Sum256(content)
	digestText := hex.EncodeToString(digest[:])
	ref, err := goal.NewArtifactRef("artifact:sha256:" + digestText)
	if err != nil {
		t.Fatal(err)
	}
	return AppAnalysisAttachment{
		Kind:        AppAnalysisAttachmentBehavior,
		ArtifactRef: ref, Digest: digestText, Size: int64(len(content)),
		MediaType:      mediaType,
		Classification: AppAnalysisClassificationShareableRedacted,
	}
}

func appAnalysisAuthorization(
	t *testing.T,
	requestRef string,
	operation AppAnalysisOperation,
	principal identity.Principal,
	project goal.ProjectRef,
	permission identity.Permission,
	resourceRef string,
) identity.AuthorizationReceipt {
	t.Helper()
	authorizationRef, err := AppAnalysisAuthorizationRequestRef(
		operation,
		requestRef,
	)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	request, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: authorizationRef, Principal: principal,
		ProjectRef: project, Permission: permission,
		ResourceRef: resourceRef, RequestedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	decision, err := identity.NewAuthorizationDecision(identity.AuthorizationDecisionInput{
		Request: request, Outcome: identity.AuthorizationAllowed,
		Role: identity.RolePlatformAdmin, MembershipRevision: 0,
		ReasonCode: "access.allowed", DecidedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := identity.NewAuthorizationReceipt(identity.AuthorizationReceiptInput{
		Ref:      "authorization-receipt:" + string(operation) + ":" + requestRef,
		Decision: decision, RecordedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func appAnalysisRepositoryRef(t *testing.T, value string) identity.RepositoryRef {
	t.Helper()
	ref, err := identity.NewRepositoryRef(value)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func appAnalysisTreeOID(digit byte) string {
	return string(makeAppAnalysisBytes(40, digit))
}

func makeAppAnalysisBytes(size int, value byte) []byte {
	result := make([]byte, size)
	for index := range result {
		result[index] = value
	}
	return result
}

func appAnalysisTestDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

type appAnalysisIntakeVerifier struct {
	mu       sync.Mutex
	allowed  map[string]AppAnalysisVerifiedIntake
	verifies int
	err      error
	mutate   func(*AppAnalysisVerifiedIntake)
}

func newAppAnalysisIntakeVerifier() *appAnalysisIntakeVerifier {
	return &appAnalysisIntakeVerifier{
		allowed: make(map[string]AppAnalysisVerifiedIntake),
	}
}

func (verifier *appAnalysisIntakeVerifier) allow(
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	intakeRef intake.Ref,
	revision intake.Revision,
) {
	evidence := AppAnalysisVerifiedIntake{
		ReceiptRef: "intake-snapshot-receipt:" + string(intakeRef),
		ActorRef:   actorRef, ProjectRef: projectRef,
		IntakeRef: intakeRef, Revision: revision,
		SnapshotDigest: appAnalysisTestDigest(
			string(intakeRef) + ":" + strconv.FormatUint(uint64(revision), 10),
		),
	}
	request := AppAnalysisIntakeVerificationRequest{
		ActorRef: actorRef, ProjectRef: projectRef,
		IntakeRef: intakeRef, ExpectedRevision: revision,
	}
	verifier.mu.Lock()
	verifier.allowed[appAnalysisIntakeKey(request)] = evidence
	verifier.mu.Unlock()
}

func (verifier *appAnalysisIntakeVerifier) VerifyAppAnalysisIntakeSnapshot(
	_ context.Context,
	request AppAnalysisIntakeVerificationRequest,
) (AppAnalysisVerifiedIntake, error) {
	verifier.mu.Lock()
	defer verifier.mu.Unlock()
	verifier.verifies++
	if verifier.err != nil {
		return AppAnalysisVerifiedIntake{}, verifier.err
	}
	evidence, found := verifier.allowed[appAnalysisIntakeKey(request)]
	if !found {
		return AppAnalysisVerifiedIntake{}, &StateError{Code: StateNotFound}
	}
	if verifier.mutate != nil {
		verifier.mutate(&evidence)
	}
	return evidence, nil
}

func (verifier *appAnalysisIntakeVerifier) verifyCount() int {
	verifier.mu.Lock()
	defer verifier.mu.Unlock()
	return verifier.verifies
}

func (verifier *appAnalysisIntakeVerifier) reset() {
	verifier.mu.Lock()
	verifier.verifies = 0
	verifier.mu.Unlock()
}

func appAnalysisIntakeKey(request AppAnalysisIntakeVerificationRequest) string {
	return request.ActorRef.String() + "\x00" + request.ProjectRef.String() +
		"\x00" + string(request.IntakeRef) + "\x00" +
		strconv.FormatUint(uint64(request.ExpectedRevision), 10)
}

type appAnalysisArtifactResolver struct {
	mu      sync.Mutex
	allowed map[string]appAnalysisAllowedResolution
	reads   int
	err     error
	mutate  func(*AppAnalysisArtifactResolution)
}

type appAnalysisAllowedResolution struct {
	request AppAnalysisArtifactResolutionRequest
	result  AppAnalysisArtifactResolution
}

func newAppAnalysisArtifactResolver() *appAnalysisArtifactResolver {
	return &appAnalysisArtifactResolver{
		allowed: make(map[string]appAnalysisAllowedResolution),
	}
}

func (resolver *appAnalysisArtifactResolver) allow(
	t *testing.T,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	manifest AppAnalysisManifest,
	descriptor AppAnalysisAttachment,
	content []byte,
) {
	t.Helper()
	canonical, err := canonicalAppAnalysisManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestDigest, err := AppAnalysisManifestDigest(canonical)
	if err != nil {
		t.Fatal(err)
	}
	canonicalDescriptor, err := canonicalAppAnalysisAttachment(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	descriptorDigest, err := AppAnalysisAttachmentDescriptorDigest(canonicalDescriptor)
	if err != nil {
		t.Fatal(err)
	}
	request := AppAnalysisArtifactResolutionRequest{
		ActorRef: actorRef, ProjectRef: projectRef,
		SubjectRef: canonical.SubjectRef, ProviderRef: canonical.ProviderRef,
		ProducerRef: canonical.ProducerRef, ProducerVersion: canonical.ProducerVersion,
		ManifestDigest: manifestDigest, Descriptor: canonicalDescriptor,
	}
	result := AppAnalysisArtifactResolution{
		Content: append([]byte(nil), content...),
		Provenance: AppAnalysisArtifactProvenanceReceipt{
			ReceiptRef: "artifact-provenance:" + descriptorDigest,
			ActorRef:   actorRef, ProjectRef: projectRef,
			SubjectRef: canonical.SubjectRef, ProviderRef: canonical.ProviderRef,
			ProducerRef: canonical.ProducerRef, ProducerVersion: canonical.ProducerVersion,
			ManifestDigest: manifestDigest, ArtifactRef: canonicalDescriptor.ArtifactRef,
			DescriptorDigest: descriptorDigest,
		},
	}
	result.ProvenanceDigest, err =
		AppAnalysisArtifactProvenanceReceiptDigest(result.Provenance)
	if err != nil {
		t.Fatal(err)
	}
	resolver.mu.Lock()
	resolver.allowed[appAnalysisResolutionKey(request)] =
		appAnalysisAllowedResolution{request: request, result: result}
	resolver.mu.Unlock()
}

func (resolver *appAnalysisArtifactResolver) ResolveAppAnalysisArtifact(
	_ context.Context,
	request AppAnalysisArtifactResolutionRequest,
) (AppAnalysisArtifactResolution, error) {
	resolver.mu.Lock()
	defer resolver.mu.Unlock()
	resolver.reads++
	if resolver.err != nil {
		return AppAnalysisArtifactResolution{}, resolver.err
	}
	allowed, found := resolver.allowed[appAnalysisResolutionKey(request)]
	if !found || allowed.request != request {
		return AppAnalysisArtifactResolution{}, ErrForbidden
	}
	result := allowed.result
	result.Content = append([]byte(nil), result.Content...)
	if resolver.mutate != nil {
		resolver.mutate(&result)
	}
	return result, nil
}

func (resolver *appAnalysisArtifactResolver) resolveCount() int {
	resolver.mu.Lock()
	defer resolver.mu.Unlock()
	return resolver.reads
}

func (resolver *appAnalysisArtifactResolver) reset() {
	resolver.mu.Lock()
	resolver.reads = 0
	resolver.mu.Unlock()
}

func appAnalysisResolutionKey(request AppAnalysisArtifactResolutionRequest) string {
	return request.ActorRef.String() + "\x00" + request.ProjectRef.String() + "\x00" +
		request.SubjectRef + "\x00" + request.ProviderRef + "\x00" +
		request.Descriptor.ArtifactRef.String()
}

type appAnalysisReviewVerifier struct {
	mu          sync.Mutex
	reviewerRef identity.PrincipalRef
	allowed     map[string]appAnalysisAllowedReview
	verifies    int
	err         error
	mutate      func(*AppAnalysisVerifiedReview)
}

type appAnalysisAllowedReview struct {
	request AppAnalysisReviewVerificationRequest
	result  AppAnalysisVerifiedReview
}

func newAppAnalysisReviewVerifier(
	reviewerRef identity.PrincipalRef,
) *appAnalysisReviewVerifier {
	return &appAnalysisReviewVerifier{
		reviewerRef: reviewerRef,
		allowed:     make(map[string]appAnalysisAllowedReview),
	}
}

func (verifier *appAnalysisReviewVerifier) allow(
	t *testing.T,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	manifest AppAnalysisManifest,
) {
	t.Helper()
	canonical, err := canonicalAppAnalysisManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := AppAnalysisManifestDigest(canonical)
	if err != nil {
		t.Fatal(err)
	}
	request := AppAnalysisReviewVerificationRequest{
		ActorRef: actorRef, ProjectRef: projectRef,
		SubjectRef: canonical.SubjectRef, ProviderRef: canonical.ProviderRef,
		ProducerRef: canonical.ProducerRef, ProducerVersion: canonical.ProducerVersion,
		ReviewReceiptRef: canonical.ReviewReceiptRef, SubjectDigest: digest,
	}
	result := AppAnalysisVerifiedReview{
		ReceiptRef: canonical.ReviewReceiptRef,
		ActorRef:   actorRef, ProjectRef: projectRef,
		SubjectRef: canonical.SubjectRef, ProviderRef: canonical.ProviderRef,
		ProducerRef: canonical.ProducerRef, ProducerVersion: canonical.ProducerVersion,
		ReviewerRef:   verifier.reviewerRef,
		Decision:      AppAnalysisReviewAccepted,
		PolicyRef:     "review-policy:shareable-analysis",
		SubjectDigest: digest,
	}
	verifier.mu.Lock()
	verifier.allowed[appAnalysisReviewKey(request)] =
		appAnalysisAllowedReview{request: request, result: result}
	verifier.mu.Unlock()
}

func (verifier *appAnalysisReviewVerifier) VerifyAppAnalysisReview(
	_ context.Context,
	request AppAnalysisReviewVerificationRequest,
) (AppAnalysisVerifiedReview, error) {
	verifier.mu.Lock()
	defer verifier.mu.Unlock()
	verifier.verifies++
	if verifier.err != nil {
		return AppAnalysisVerifiedReview{}, verifier.err
	}
	allowed, found := verifier.allowed[appAnalysisReviewKey(request)]
	if !found || allowed.request != request {
		return AppAnalysisVerifiedReview{}, ErrForbidden
	}
	result := allowed.result
	if verifier.mutate != nil {
		verifier.mutate(&result)
	}
	return result, nil
}

func (verifier *appAnalysisReviewVerifier) verifyCount() int {
	verifier.mu.Lock()
	defer verifier.mu.Unlock()
	return verifier.verifies
}

func (verifier *appAnalysisReviewVerifier) reset() {
	verifier.mu.Lock()
	verifier.verifies = 0
	verifier.mu.Unlock()
}

func appAnalysisReviewKey(request AppAnalysisReviewVerificationRequest) string {
	return request.ActorRef.String() + "\x00" + request.ProjectRef.String() +
		"\x00" + request.ReviewReceiptRef
}

type appAnalysisMemoryStore struct {
	mu       sync.Mutex
	requests map[string]AppAnalysisAttachmentRecord
	analyses map[string]AppAnalysisAttachmentRecord
	replays  int
	attaches int
}

func newAppAnalysisMemoryStore() *appAnalysisMemoryStore {
	return &appAnalysisMemoryStore{
		requests: make(map[string]AppAnalysisAttachmentRecord),
		analyses: make(map[string]AppAnalysisAttachmentRecord),
	}
}

func (store *appAnalysisMemoryStore) ReplayAppAnalysisAttachment(
	_ context.Context,
	request AppAnalysisReplayRequest,
) (AppAnalysisAttachmentRecord, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.replays++
	record, found := store.requests[appAnalysisRequestKey(
		request.ActorRef,
		request.ProjectRef,
		request.RequestRef,
	)]
	if !found {
		return AppAnalysisAttachmentRecord{}, false, nil
	}
	if record.Receipt.RequestFingerprint != request.RequestFingerprint ||
		record.Receipt.SubjectManifestDigest != request.SubjectManifestDigest ||
		record.Receipt.AuthorizationReceiptRef != request.AuthorizationReceiptRef ||
		record.Receipt.ReviewReceiptRef != request.ReviewReceiptRef ||
		record.Analysis.Manifest().IntakeRef != request.IntakeRef ||
		record.Analysis.Manifest().ExpectedRevision != request.ExpectedRevision {
		return AppAnalysisAttachmentRecord{}, false, &StateError{Code: StateConflict}
	}
	return record, true, nil
}

func (store *appAnalysisMemoryStore) AttachAppAnalysisAttachment(
	_ context.Context,
	state AppAnalysisAttachState,
) (AppAnalysisAttachmentRecord, AppAnalysisAttachmentDisposition, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.attaches++
	requestKey := appAnalysisRequestKey(
		state.ActorRef,
		state.ProjectRef,
		state.RequestRef,
	)
	if current, found := store.requests[requestKey]; found {
		if current.Receipt.RequestFingerprint != state.RequestFingerprint ||
			current.Receipt != state.Receipt {
			return AppAnalysisAttachmentRecord{}, "", &StateError{Code: StateConflict}
		}
		return current, AppAnalysisAttachmentReplayed, nil
	}
	record := AppAnalysisAttachmentRecord{
		ActorRef: state.ActorRef, ProjectRef: state.ProjectRef,
		Analysis: state.Analysis, Receipt: state.Receipt,
	}
	analysisKey := appAnalysisKey(
		state.ActorRef,
		state.ProjectRef,
		state.Analysis.Ref(),
	)
	_, found := store.analyses[analysisKey]
	disposition := AppAnalysisAttachmentDeduplicated
	if !found {
		store.analyses[analysisKey] = record
		disposition = AppAnalysisAttachmentCreated
	}
	store.requests[requestKey] = record
	return record, disposition, nil
}

func (store *appAnalysisMemoryStore) GetAppAnalysisAttachment(
	_ context.Context,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	analysisRef AppAnalysisRef,
) (AppAnalysisAttachmentRecord, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	record, found := store.analyses[appAnalysisKey(actorRef, projectRef, analysisRef)]
	if !found {
		return AppAnalysisAttachmentRecord{}, &StateError{Code: StateNotFound}
	}
	return record, nil
}

func (store *appAnalysisMemoryStore) canonical(
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	analysisRef AppAnalysisRef,
) AppAnalysisAttachmentRecord {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.analyses[appAnalysisKey(actorRef, projectRef, analysisRef)]
}

func (store *appAnalysisMemoryStore) replayCount() int {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.replays
}

func (store *appAnalysisMemoryStore) attachCount() int {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.attaches
}

func appAnalysisRequestKey(
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	requestRef string,
) string {
	return actorRef.String() + "\x00" + projectRef.String() + "\x00" + requestRef
}

func appAnalysisKey(
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	analysisRef AppAnalysisRef,
) string {
	return actorRef.String() + "\x00" + projectRef.String() + "\x00" +
		string(analysisRef)
}
