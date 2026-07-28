package application

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/identity"
)

// TestAppAnalysisManifestContract proves only the neutral standalone
// foundation for WIZ-10/EXT-00. It does not prove an adapter, public surface,
// Orchestrator wiring, Goal mutation or AC-V28 accreditation.
func TestAppAnalysisManifestContract(t *testing.T) {
	if AppAnalysisManifestSchema != "orquesta.app-analysis.manifest.v1" ||
		AppAnalysisEnvelopeSchema != "orquesta.app-analysis.envelope.v1" {
		t.Fatal("app analysis schema drift")
	}
	if AppAnalysisAttachmentBehavior != "behavior" ||
		AppAnalysisAttachmentCodebase != "codebase" ||
		AppAnalysisPurposeCleanRoomReimplementation != "clean_room_reimplementation" ||
		AppAnalysisPurposeRefactorExisting != "refactor_existing" ||
		AppAnalysisClassificationShareableRedacted != "shareable_redacted" {
		t.Fatal("app analysis vocabulary drift")
	}

	fixture := newAppAnalysisTestFixture(t, AppAnalysisPurposeCleanRoomReimplementation)
	behavior := fixture.envelope.Manifest.Attachments[0]
	codebase := appAnalysisDescriptor(
		t,
		"text/plain",
		[]byte("provider-neutral codebase findings"),
	)
	codebase.Kind = AppAnalysisAttachmentCodebase
	left := cloneAppAnalysisManifest(fixture.envelope.Manifest)
	left.Attachments = []AppAnalysisAttachment{codebase, behavior}
	right := cloneAppAnalysisManifest(left)
	right.Attachments = []AppAnalysisAttachment{behavior, codebase}

	leftDigest, leftErr := AppAnalysisManifestDigest(left)
	rightDigest, rightErr := AppAnalysisManifestDigest(right)
	if leftErr != nil || rightErr != nil || leftDigest != rightDigest {
		t.Fatalf("canonical digest mismatch: %q/%q errors=%v/%v",
			leftDigest, rightDigest, leftErr, rightErr)
	}
	analysis := appAnalysisBuildForTest(t, fixture, left)
	if analysis.Ref() != AppAnalysisRef("app-analysis:"+analysis.Digest()) ||
		analysis.Review().SubjectDigest != leftDigest {
		t.Fatalf("content/review address mismatch: analysis=%+v", analysis)
	}
	manifest := analysis.Manifest()
	if len(manifest.Attachments) != 2 ||
		manifest.Attachments[0].Kind != AppAnalysisAttachmentBehavior ||
		manifest.Attachments[0].MediaType != "application/json; charset=UTF-8" ||
		manifest.Attachments[1].Kind != AppAnalysisAttachmentCodebase {
		t.Fatalf("manifest not canonical: %+v", manifest.Attachments)
	}
	manifest.Attachments[0].MediaType = "application/x-mutated"
	if analysis.Manifest().Attachments[0].MediaType == "application/x-mutated" {
		t.Fatal("manifest getter exposed mutable storage")
	}
}

func TestAppAnalysisManifestDigestCoversTopLevelCausalityAndEveryDescriptorField(t *testing.T) {
	fixture := newAppAnalysisTestFixture(t, AppAnalysisPurposeRefactorExisting)
	base := fixture.envelope.Manifest
	baseDigest, err := AppAnalysisManifestDigest(base)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		mutate func(*AppAnalysisManifest)
	}{
		{"subject", func(m *AppAnalysisManifest) { m.SubjectRef = "application:other" }},
		{"provider", func(m *AppAnalysisManifest) { m.ProviderRef = "provider:other" }},
		{"producer", func(m *AppAnalysisManifest) { m.ProducerRef = "producer:other" }},
		{"version", func(m *AppAnalysisManifest) { m.ProducerVersion = "2026.08" }},
		{"intake", func(m *AppAnalysisManifest) { m.IntakeRef = "intake:other" }},
		{"revision", func(m *AppAnalysisManifest) { m.ExpectedRevision++ }},
		{"purpose", func(m *AppAnalysisManifest) {
			m.Purpose = AppAnalysisPurposeCleanRoomReimplementation
			m.RepositoryRef = identity.RepositoryRef{}
			m.TreeOID = ""
		}},
		{"repository", func(m *AppAnalysisManifest) {
			m.RepositoryRef = appAnalysisRepositoryRef(t, "repository:other")
		}},
		{"tree", func(m *AppAnalysisManifest) { m.TreeOID = appAnalysisTreeOID('b') }},
		{"review receipt", func(m *AppAnalysisManifest) {
			m.ReviewReceiptRef = "analysis-review-receipt:other"
		}},
		{"artifact ref digest size", func(m *AppAnalysisManifest) {
			m.Attachments[0] = appAnalysisDescriptor(
				t,
				"Application/JSON; Charset=UTF-8",
				[]byte("other"),
			)
		}},
		{"media type", func(m *AppAnalysisManifest) {
			m.Attachments[0].MediaType = "text/plain"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			changed := cloneAppAnalysisManifest(base)
			test.mutate(&changed)
			digest, digestErr := AppAnalysisManifestDigest(changed)
			if digestErr != nil || digest == baseDigest {
				t.Fatalf("digest=%q base=%q err=%v", digest, baseDigest, digestErr)
			}
		})
	}
}

func TestAppAnalysisInputCannotSpoofReviewOrDuplicateCausalityPerAttachment(t *testing.T) {
	attachment := reflect.TypeOf(AppAnalysisAttachment{})
	for _, forbidden := range []string{
		"Purpose", "RepositoryRef", "TreeOID", "Review", "Reviewer",
		"Policy", "Provider", "Producer", "Subject", "Intake", "Revision",
	} {
		if _, found := attachment.FieldByName(forbidden); found {
			t.Fatalf("attachment duplicates/spoofs %s", forbidden)
		}
	}
	envelope := reflect.TypeOf(AppAnalysisEnvelope{})
	for _, forbidden := range []string{
		"ReviewerRef", "ReviewDecision", "ReviewPolicy", "ReviewSubjectDigest",
	} {
		if _, found := envelope.FieldByName(forbidden); found {
			t.Fatalf("envelope can spoof %s", forbidden)
		}
	}
	manifest := reflect.TypeOf(AppAnalysisManifest{})
	if _, found := manifest.FieldByName("ReviewReceiptRef"); !found {
		t.Fatal("manifest lacks opaque review receipt ref")
	}
}

func TestAppAnalysisSnapshotRestoreRoundTripAndTamper(t *testing.T) {
	fixture := newAppAnalysisTestFixture(t, AppAnalysisPurposeRefactorExisting)
	result, err := fixture.service.AttachAppAnalysis(
		t.Context(),
		fixture.envelope,
	)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := result.Record.Analysis.Snapshot()
	restored, err := RestoreAppAnalysis(snapshot)
	if err != nil || !reflect.DeepEqual(restored.Snapshot(), snapshot) {
		t.Fatalf("roundtrip restored=%+v err=%v", restored.Snapshot(), err)
	}
	snapshot.Manifest.Attachments[0].MediaType = "text/x-mutated"
	snapshot.Provenance[0].ReceiptRef = "artifact-provenance:mutated-copy"
	if reflect.DeepEqual(snapshot, result.Record.Analysis.Snapshot()) {
		t.Fatal("snapshot aliases immutable analysis")
	}

	tests := []struct {
		name   string
		mutate func(*AppAnalysisSnapshot)
	}{
		{"ref", func(s *AppAnalysisSnapshot) {
			s.Ref = AppAnalysisRef("app-analysis:" + appAnalysisTestDigest("other"))
		}},
		{"manifest provenance", func(s *AppAnalysisSnapshot) {
			s.Manifest.Attachments[0].ProvenanceReceiptDigest = appAnalysisTestDigest("other")
		}},
		{"provenance receipt", func(s *AppAnalysisSnapshot) {
			s.Provenance[0].ReceiptRef = "artifact-provenance:other"
		}},
		{"intake digest", func(s *AppAnalysisSnapshot) {
			s.Intake.SnapshotDigest = appAnalysisTestDigest("other")
		}},
		{"review policy", func(s *AppAnalysisSnapshot) {
			s.Review.PolicyRef = "review-policy:other"
		}},
		{"verified review digest", func(s *AppAnalysisSnapshot) {
			s.VerifiedReviewDigest = appAnalysisTestDigest("other")
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tampered := result.Record.Analysis.Snapshot()
			test.mutate(&tampered)
			if _, restoreErr := RestoreAppAnalysis(tampered); !errors.Is(
				restoreErr,
				ErrAppAnalysisInvalid,
			) {
				t.Fatalf("restore error=%v", restoreErr)
			}
		})
	}
}

func TestAppAnalysisInputContractHasNoExecutionLocationOrGlobalCASSurface(t *testing.T) {
	for _, contract := range []reflect.Type{
		reflect.TypeOf(AppAnalysisEnvelope{}),
		reflect.TypeOf(AppAnalysisManifest{}),
		reflect.TypeOf(AppAnalysisAttachment{}),
	} {
		for index := 0; index < contract.NumField(); index++ {
			name := strings.ToLower(contract.Field(index).Name)
			for _, forbidden := range []string{
				"path", "url", "argv", "command", "tool", "test", "sqlite", "etw",
			} {
				if strings.Contains(name, forbidden) {
					t.Fatalf("%s exposes forbidden field %s", contract, contract.Field(index).Name)
				}
			}
		}
	}
	service := reflect.TypeOf(AppAnalysisService{})
	if service.NumField() != 5 ||
		service.Field(0).Type != reflect.TypeOf((*AppAnalysisIntakeSnapshotVerifier)(nil)).Elem() ||
		service.Field(1).Type != reflect.TypeOf((*AppAnalysisArtifactResolver)(nil)).Elem() ||
		service.Field(2).Type != reflect.TypeOf((*AppAnalysisReviewVerifier)(nil)).Elem() ||
		service.Field(3).Type != reflect.TypeOf((*AppAnalysisAttachmentStore)(nil)).Elem() ||
		service.Field(4).Type.Kind() != reflect.Int64 {
		t.Fatalf("service crossed standalone ports: %+v", service)
	}
	if reflect.TypeOf(AppAnalysisManifest{}.RepositoryRef) !=
		reflect.TypeOf(identity.RepositoryRef{}) {
		t.Fatal("repository_ref lost opaque type")
	}
}

func appAnalysisVerifiedReview(
	t *testing.T,
	fixture *appAnalysisTestFixture,
	manifest AppAnalysisManifest,
) AppAnalysisVerifiedReview {
	t.Helper()
	canonical, err := canonicalAppAnalysisManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := AppAnalysisManifestDigest(canonical)
	if err != nil {
		t.Fatal(err)
	}
	return AppAnalysisVerifiedReview{
		ReceiptRef: canonical.ReviewReceiptRef,
		ActorRef:   fixture.actor, ProjectRef: fixture.project,
		SubjectRef: canonical.SubjectRef, ProviderRef: canonical.ProviderRef,
		ProducerRef: canonical.ProducerRef, ProducerVersion: canonical.ProducerVersion,
		ReviewerRef: fixture.reviewerRef, Decision: AppAnalysisReviewAccepted,
		PolicyRef: "review-policy:shareable-analysis", SubjectDigest: digest,
	}
}

func appAnalysisBuildForTest(
	t *testing.T,
	fixture *appAnalysisTestFixture,
	manifest AppAnalysisManifest,
) AppAnalysis {
	t.Helper()
	canonical, err := canonicalAppAnalysisIngressManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	subjectDigest, err := AppAnalysisManifestDigest(canonical)
	if err != nil {
		t.Fatal(err)
	}
	intakeEvidence := AppAnalysisVerifiedIntake{
		ReceiptRef: "intake-snapshot-receipt:" + string(canonical.IntakeRef),
		ActorRef:   fixture.actor, ProjectRef: fixture.project,
		IntakeRef: canonical.IntakeRef, Revision: canonical.ExpectedRevision,
		SnapshotDigest: appAnalysisTestDigest("snapshot:" + string(canonical.IntakeRef)),
	}
	persisted := cloneAppAnalysisManifest(canonical)
	provenance := make([]AppAnalysisArtifactProvenanceReceipt, len(canonical.Attachments))
	for index, attachment := range canonical.Attachments {
		descriptorDigest, digestErr := AppAnalysisAttachmentDescriptorDigest(attachment)
		if digestErr != nil {
			t.Fatal(digestErr)
		}
		provenance[index] = AppAnalysisArtifactProvenanceReceipt{
			ReceiptRef: "artifact-provenance:" + descriptorDigest,
			ActorRef:   fixture.actor, ProjectRef: fixture.project,
			SubjectRef: canonical.SubjectRef, ProviderRef: canonical.ProviderRef,
			ProducerRef: canonical.ProducerRef, ProducerVersion: canonical.ProducerVersion,
			ManifestDigest: subjectDigest, ArtifactRef: attachment.ArtifactRef,
			DescriptorDigest: descriptorDigest,
		}
		provenanceDigest, digestErr :=
			AppAnalysisArtifactProvenanceReceiptDigest(provenance[index])
		if digestErr != nil {
			t.Fatal(digestErr)
		}
		persisted.Attachments[index].ProvenanceReceiptRef =
			provenance[index].ReceiptRef
		persisted.Attachments[index].ProvenanceReceiptDigest = provenanceDigest
	}
	analysis, err := buildAppAnalysis(
		persisted,
		intakeEvidence,
		provenance,
		appAnalysisVerifiedReview(t, fixture, canonical),
	)
	if err != nil {
		t.Fatal(err)
	}
	return analysis
}
