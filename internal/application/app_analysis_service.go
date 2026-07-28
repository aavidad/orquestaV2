package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"strconv"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/intake"
)

type AppAnalysisOperation string

const (
	AppAnalysisOperationAttach AppAnalysisOperation = "attach"
	AppAnalysisOperationGet    AppAnalysisOperation = "get"
)

type AppAnalysisAttachmentReceipt struct {
	Ref                     string
	RequestRef              string
	RequestFingerprint      string
	ActorRef                goal.ActorRef
	ProjectRef              goal.ProjectRef
	AnalysisRef             AppAnalysisRef
	AnalysisDigest          string
	ManifestDigest          string
	SubjectManifestDigest   string
	AuthorizationReceiptRef string
	ReviewReceiptRef        string
	VerifiedReviewDigest    string
	IntakeReceiptRef        string
	IntakeSnapshotDigest    string
	VerifiedIntakeDigest    string
}

type AppAnalysisAttachmentRecord struct {
	ActorRef   goal.ActorRef
	ProjectRef goal.ProjectRef
	Analysis   AppAnalysis
	Receipt    AppAnalysisAttachmentReceipt
}

type AppAnalysisReplayRequest struct {
	RequestRef              string
	RequestFingerprint      string
	ActorRef                goal.ActorRef
	ProjectRef              goal.ProjectRef
	SubjectManifestDigest   string
	AuthorizationReceiptRef string
	ReviewReceiptRef        string
	IntakeRef               intake.Ref
	ExpectedRevision        intake.Revision
}

type AppAnalysisAttachState struct {
	RequestRef           string
	RequestFingerprint   string
	ActorRef             goal.ActorRef
	ProjectRef           goal.ProjectRef
	AuthorizationReceipt identity.AuthorizationReceipt
	Analysis             AppAnalysis
	Receipt              AppAnalysisAttachmentReceipt
}

type AppAnalysisAttachmentDisposition string

const (
	AppAnalysisAttachmentCreated      AppAnalysisAttachmentDisposition = "created"
	AppAnalysisAttachmentReplayed     AppAnalysisAttachmentDisposition = "replayed"
	AppAnalysisAttachmentDeduplicated AppAnalysisAttachmentDisposition = "deduplicated"
)

type AppAnalysisAttachmentStore interface {
	ReplayAppAnalysisAttachment(
		context.Context,
		AppAnalysisReplayRequest,
	) (AppAnalysisAttachmentRecord, bool, error)
	AttachAppAnalysisAttachment(
		context.Context,
		AppAnalysisAttachState,
	) (AppAnalysisAttachmentRecord, AppAnalysisAttachmentDisposition, error)
	GetAppAnalysisAttachment(
		context.Context,
		goal.ActorRef,
		goal.ProjectRef,
		AppAnalysisRef,
	) (AppAnalysisAttachmentRecord, error)
}

type AppAnalysisIntakeVerificationRequest struct {
	ActorRef         goal.ActorRef
	ProjectRef       goal.ProjectRef
	IntakeRef        intake.Ref
	ExpectedRevision intake.Revision
}

// AppAnalysisIntakeSnapshotVerifier resolves one exact project-owned intake
// revision and its immutable snapshot receipt.
type AppAnalysisIntakeSnapshotVerifier interface {
	VerifyAppAnalysisIntakeSnapshot(
		context.Context,
		AppAnalysisIntakeVerificationRequest,
	) (AppAnalysisVerifiedIntake, error)
}

type AppAnalysisArtifactResolutionRequest struct {
	ActorRef        goal.ActorRef
	ProjectRef      goal.ProjectRef
	SubjectRef      string
	ProviderRef     string
	ProducerRef     string
	ProducerVersion string
	ManifestDigest  string
	Descriptor      AppAnalysisAttachment
}

type AppAnalysisArtifactProvenanceReceipt struct {
	ReceiptRef       string
	ActorRef         goal.ActorRef
	ProjectRef       goal.ProjectRef
	SubjectRef       string
	ProviderRef      string
	ProducerRef      string
	ProducerVersion  string
	ManifestDigest   string
	ArtifactRef      goal.ArtifactRef
	DescriptorDigest string
}

type AppAnalysisArtifactResolution struct {
	Content          []byte
	Provenance       AppAnalysisArtifactProvenanceReceipt
	ProvenanceDigest string
}

type AppAnalysisArtifactResolver interface {
	ResolveAppAnalysisArtifact(
		context.Context,
		AppAnalysisArtifactResolutionRequest,
	) (AppAnalysisArtifactResolution, error)
}

type AppAnalysisReviewVerificationRequest struct {
	ActorRef         goal.ActorRef
	ProjectRef       goal.ProjectRef
	SubjectRef       string
	ProviderRef      string
	ProducerRef      string
	ProducerVersion  string
	ReviewReceiptRef string
	SubjectDigest    string
}

type AppAnalysisReviewVerifier interface {
	VerifyAppAnalysisReview(
		context.Context,
		AppAnalysisReviewVerificationRequest,
	) (AppAnalysisVerifiedReview, error)
}

type AppAnalysisAttachmentResult struct {
	Record      AppAnalysisAttachmentRecord
	Disposition AppAnalysisAttachmentDisposition
}

type GetAppAnalysisAttachmentRequest struct {
	RequestRef           string
	ActorRef             goal.ActorRef
	ProjectRef           goal.ProjectRef
	AnalysisRef          AppAnalysisRef
	AuthorizationReceipt identity.AuthorizationReceipt
}

type AppAnalysisService struct {
	intakes            AppAnalysisIntakeSnapshotVerifier
	artifacts          AppAnalysisArtifactResolver
	reviews            AppAnalysisReviewVerifier
	attachments        AppAnalysisAttachmentStore
	maxAttachmentBytes int64
}

func NewAppAnalysisService(
	intakes AppAnalysisIntakeSnapshotVerifier,
	artifacts AppAnalysisArtifactResolver,
	reviews AppAnalysisReviewVerifier,
	attachments AppAnalysisAttachmentStore,
	maxAttachmentBytes int64,
) (*AppAnalysisService, error) {
	switch {
	case intakes == nil:
		return nil, errors.New("application.app_analysis_intake_verifier_required")
	case artifacts == nil:
		return nil, errors.New("application.app_analysis_artifact_resolver_required")
	case reviews == nil:
		return nil, errors.New("application.app_analysis_review_verifier_required")
	case attachments == nil:
		return nil, errors.New("application.app_analysis_attachment_store_required")
	case maxAttachmentBytes <= 0:
		return nil, errors.New("application.app_analysis_max_attachment_bytes_invalid")
	default:
		return &AppAnalysisService{
			intakes: intakes, artifacts: artifacts, reviews: reviews,
			attachments: attachments, maxAttachmentBytes: maxAttachmentBytes,
		}, nil
	}
}

func (service *AppAnalysisService) AttachAppAnalysis(
	ctx context.Context,
	envelope AppAnalysisEnvelope,
) (AppAnalysisAttachmentResult, error) {
	if service == nil || service.intakes == nil || service.artifacts == nil ||
		service.reviews == nil || service.attachments == nil {
		return AppAnalysisAttachmentResult{}, errors.New("application.unavailable")
	}
	if envelope.Schema != AppAnalysisEnvelopeSchema {
		return AppAnalysisAttachmentResult{}, invalidAppAnalysis("envelope.schema", nil)
	}
	if !validIntakeRequestRef(envelope.RequestRef) {
		return AppAnalysisAttachmentResult{}, invalidAppAnalysis("envelope.request_ref", nil)
	}
	if err := validateIntakeScope(envelope.ActorRef, envelope.ProjectRef); err != nil {
		return AppAnalysisAttachmentResult{}, err
	}
	canonical, err := canonicalAppAnalysisIngressManifest(envelope.Manifest)
	if err != nil {
		return AppAnalysisAttachmentResult{}, err
	}
	if err := validateAppAnalysisAttachmentSizes(canonical, service.maxAttachmentBytes); err != nil {
		return AppAnalysisAttachmentResult{}, err
	}
	authorizationRequestRef, err := AppAnalysisAuthorizationRequestRef(
		AppAnalysisOperationAttach,
		envelope.RequestRef,
	)
	if err != nil {
		return AppAnalysisAttachmentResult{}, err
	}
	if err := validateAppAnalysisAuthorization(
		envelope.AuthorizationReceipt,
		envelope.ActorRef,
		envelope.ProjectRef,
		identity.PermissionGoalsCreate,
		envelope.ProjectRef.String(),
		authorizationRequestRef,
	); err != nil {
		return AppAnalysisAttachmentResult{}, err
	}

	subjectManifestDigest, err := AppAnalysisManifestDigest(canonical)
	if err != nil {
		return AppAnalysisAttachmentResult{}, err
	}
	fingerprint := appAnalysisAttachmentFingerprint(
		envelope.ActorRef,
		envelope.ProjectRef,
		subjectManifestDigest,
		envelope.AuthorizationReceipt.Ref(),
	)
	replay := AppAnalysisReplayRequest{
		RequestRef: envelope.RequestRef, RequestFingerprint: fingerprint,
		ActorRef: envelope.ActorRef, ProjectRef: envelope.ProjectRef,
		SubjectManifestDigest:   subjectManifestDigest,
		AuthorizationReceiptRef: envelope.AuthorizationReceipt.Ref(),
		ReviewReceiptRef:        canonical.ReviewReceiptRef,
		IntakeRef:               canonical.IntakeRef, ExpectedRevision: canonical.ExpectedRevision,
	}
	if record, found, replayErr := service.replay(ctx, replay); replayErr != nil {
		return AppAnalysisAttachmentResult{}, replayErr
	} else if found {
		return AppAnalysisAttachmentResult{
			Record: record, Disposition: AppAnalysisAttachmentReplayed,
		}, nil
	}

	intakeEvidence, err := service.intakes.VerifyAppAnalysisIntakeSnapshot(
		ctx,
		AppAnalysisIntakeVerificationRequest{
			ActorRef: envelope.ActorRef, ProjectRef: envelope.ProjectRef,
			IntakeRef:        canonical.IntakeRef,
			ExpectedRevision: canonical.ExpectedRevision,
		},
	)
	if err != nil {
		return AppAnalysisAttachmentResult{}, err
	}
	if intakeEvidence.ActorRef != envelope.ActorRef ||
		intakeEvidence.ProjectRef != envelope.ProjectRef {
		return AppAnalysisAttachmentResult{}, ErrForbidden
	}
	if err := validateAppAnalysisVerifiedIntake(canonical, intakeEvidence); err != nil {
		return AppAnalysisAttachmentResult{}, err
	}

	verifiedReview, err := service.reviews.VerifyAppAnalysisReview(
		ctx,
		AppAnalysisReviewVerificationRequest{
			ActorRef: envelope.ActorRef, ProjectRef: envelope.ProjectRef,
			SubjectRef: canonical.SubjectRef, ProviderRef: canonical.ProviderRef,
			ProducerRef: canonical.ProducerRef, ProducerVersion: canonical.ProducerVersion,
			ReviewReceiptRef: canonical.ReviewReceiptRef,
			SubjectDigest:    subjectManifestDigest,
		},
	)
	if err != nil {
		return AppAnalysisAttachmentResult{}, err
	}
	if verifiedReview.ActorRef != envelope.ActorRef ||
		verifiedReview.ProjectRef != envelope.ProjectRef {
		return AppAnalysisAttachmentResult{}, ErrForbidden
	}
	if err := validateAppAnalysisVerifiedReview(
		canonical,
		subjectManifestDigest,
		verifiedReview,
	); err != nil {
		return AppAnalysisAttachmentResult{}, err
	}

	persistedManifest := cloneAppAnalysisManifest(canonical)
	provenance := make(
		[]AppAnalysisArtifactProvenanceReceipt,
		0,
		len(canonical.Attachments),
	)
	for index, attachment := range canonical.Attachments {
		resolutionRequest := AppAnalysisArtifactResolutionRequest{
			ActorRef: envelope.ActorRef, ProjectRef: envelope.ProjectRef,
			SubjectRef: canonical.SubjectRef, ProviderRef: canonical.ProviderRef,
			ProducerRef: canonical.ProducerRef, ProducerVersion: canonical.ProducerVersion,
			ManifestDigest: subjectManifestDigest, Descriptor: attachment,
		}
		resolution, resolveErr := service.artifacts.ResolveAppAnalysisArtifact(
			ctx,
			resolutionRequest,
		)
		if resolveErr != nil {
			return AppAnalysisAttachmentResult{}, resolveErr
		}
		provenanceDigest, validationErr := validateAppAnalysisArtifactResolution(
			resolutionRequest,
			resolution,
		)
		if validationErr != nil {
			return AppAnalysisAttachmentResult{}, invalidAppAnalysis(
				"manifest.attachments["+strconv.Itoa(index)+"]",
				validationErr,
			)
		}
		persistedManifest.Attachments[index].ProvenanceReceiptRef =
			resolution.Provenance.ReceiptRef
		persistedManifest.Attachments[index].ProvenanceReceiptDigest =
			provenanceDigest
		provenance = append(provenance, resolution.Provenance)
	}
	analysis, err := buildAppAnalysis(
		persistedManifest,
		intakeEvidence,
		provenance,
		verifiedReview,
	)
	if err != nil {
		return AppAnalysisAttachmentResult{}, err
	}

	receipt, err := buildAppAnalysisAttachmentReceipt(
		envelope.RequestRef,
		fingerprint,
		envelope.ActorRef,
		envelope.ProjectRef,
		subjectManifestDigest,
		analysis,
		envelope.AuthorizationReceipt.Ref(),
	)
	if err != nil {
		return AppAnalysisAttachmentResult{}, err
	}
	persisted, disposition, err := service.attachments.AttachAppAnalysisAttachment(
		ctx,
		AppAnalysisAttachState{
			RequestRef: envelope.RequestRef, RequestFingerprint: fingerprint,
			ActorRef: envelope.ActorRef, ProjectRef: envelope.ProjectRef,
			AuthorizationReceipt: envelope.AuthorizationReceipt,
			Analysis:             analysis, Receipt: receipt,
		},
	)
	if err != nil {
		if IsStateError(err, StateConflict) {
			if recovered, found, replayErr := service.replay(ctx, replay); replayErr != nil {
				return AppAnalysisAttachmentResult{}, replayErr
			} else if found {
				return AppAnalysisAttachmentResult{
					Record: recovered, Disposition: AppAnalysisAttachmentReplayed,
				}, nil
			}
		}
		return AppAnalysisAttachmentResult{}, err
	}
	if disposition != AppAnalysisAttachmentCreated &&
		disposition != AppAnalysisAttachmentReplayed &&
		disposition != AppAnalysisAttachmentDeduplicated {
		return AppAnalysisAttachmentResult{}, &StateError{Code: StateConflict}
	}
	if err := validateAppAnalysisMutationRecord(replay, analysis, persisted); err != nil {
		return AppAnalysisAttachmentResult{}, err
	}
	cloned, err := cloneAppAnalysisAttachmentRecord(persisted)
	if err != nil {
		return AppAnalysisAttachmentResult{}, err
	}
	return AppAnalysisAttachmentResult{Record: cloned, Disposition: disposition}, nil
}

func (service *AppAnalysisService) GetAppAnalysisAttachment(
	ctx context.Context,
	request GetAppAnalysisAttachmentRequest,
) (AppAnalysisAttachmentRecord, error) {
	if service == nil || service.attachments == nil {
		return AppAnalysisAttachmentRecord{}, errors.New("application.unavailable")
	}
	if !validIntakeRequestRef(request.RequestRef) {
		return AppAnalysisAttachmentRecord{}, invalidAppAnalysis("request_ref", nil)
	}
	if err := validateIntakeScope(request.ActorRef, request.ProjectRef); err != nil {
		return AppAnalysisAttachmentRecord{}, err
	}
	if !validAppAnalysisRef(request.AnalysisRef) {
		return AppAnalysisAttachmentRecord{}, invalidAppAnalysis("analysis_ref", nil)
	}
	authorizationRequestRef, err := AppAnalysisAuthorizationRequestRef(
		AppAnalysisOperationGet,
		request.RequestRef,
	)
	if err != nil {
		return AppAnalysisAttachmentRecord{}, err
	}
	if err := validateAppAnalysisAuthorization(
		request.AuthorizationReceipt,
		request.ActorRef,
		request.ProjectRef,
		identity.PermissionArtifactsRead,
		string(request.AnalysisRef),
		authorizationRequestRef,
	); err != nil {
		return AppAnalysisAttachmentRecord{}, err
	}
	record, err := service.attachments.GetAppAnalysisAttachment(
		ctx,
		request.ActorRef,
		request.ProjectRef,
		request.AnalysisRef,
	)
	if err != nil {
		return AppAnalysisAttachmentRecord{}, err
	}
	if err := validateStoredAppAnalysisAttachmentRecord(
		request.ActorRef,
		request.ProjectRef,
		request.AnalysisRef,
		record,
	); err != nil {
		return AppAnalysisAttachmentRecord{}, err
	}
	return cloneAppAnalysisAttachmentRecord(record)
}

func AppAnalysisAuthorizationRequestRef(
	operation AppAnalysisOperation,
	requestRef string,
) (string, error) {
	if !validIntakeRequestRef(requestRef) {
		return "", invalidAppAnalysis("request_ref", nil)
	}
	switch operation {
	case AppAnalysisOperationAttach, AppAnalysisOperationGet:
		return "authorization-request:app-analysis-" +
			string(operation) + ":" + requestRef, nil
	default:
		return "", invalidAppAnalysis("operation", nil)
	}
}

func (service *AppAnalysisService) replay(
	ctx context.Context,
	request AppAnalysisReplayRequest,
) (AppAnalysisAttachmentRecord, bool, error) {
	record, found, err := service.attachments.ReplayAppAnalysisAttachment(ctx, request)
	if err != nil || !found {
		return AppAnalysisAttachmentRecord{}, false, err
	}
	if err := validateAppAnalysisReplayRecord(request, record); err != nil {
		return AppAnalysisAttachmentRecord{}, false, err
	}
	cloned, err := cloneAppAnalysisAttachmentRecord(record)
	if err != nil {
		return AppAnalysisAttachmentRecord{}, false, err
	}
	return cloned, true, nil
}

func validateAppAnalysisAuthorization(
	receipt identity.AuthorizationReceipt,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	permission identity.Permission,
	resourceRef string,
	authorizationRequestRef string,
) error {
	decision := receipt.Decision()
	request := decision.Request()
	principal := request.Principal()
	if receipt.Ref() == "" || receipt.RecordedAt().IsZero() ||
		decision.Outcome() != identity.AuthorizationAllowed ||
		!identity.RoleAllows(decision.Role(), permission) ||
		request.Permission() != permission ||
		request.ProjectRef() != projectRef ||
		request.ResourceRef() != resourceRef ||
		request.RequestRef() != authorizationRequestRef ||
		principal.ActorRef != actorRef ||
		receipt.RecordedAt().Before(decision.DecidedAt()) {
		return ErrForbidden
	}
	return nil
}

func validateAppAnalysisAttachmentSizes(
	manifest AppAnalysisManifest,
	maxAttachmentBytes int64,
) error {
	var total int64
	for index, attachment := range manifest.Attachments {
		if attachment.Size > maxAttachmentBytes ||
			total > maxAttachmentBytes-attachment.Size {
			return invalidAppAnalysis(
				"manifest.attachments["+strconv.Itoa(index)+"].size",
				nil,
			)
		}
		total += attachment.Size
	}
	return nil
}

func validateAppAnalysisArtifactResolution(
	request AppAnalysisArtifactResolutionRequest,
	resolution AppAnalysisArtifactResolution,
) (string, error) {
	descriptorDigest, err := AppAnalysisAttachmentDescriptorDigest(request.Descriptor)
	if err != nil {
		return "", err
	}
	contentDigest := sha256.Sum256(resolution.Content)
	contentDigestText := hex.EncodeToString(contentDigest[:])
	receipt := resolution.Provenance
	provenanceDigest, err := AppAnalysisArtifactProvenanceReceiptDigest(receipt)
	if err != nil ||
		receipt.ActorRef != request.ActorRef ||
		receipt.ProjectRef != request.ProjectRef ||
		receipt.SubjectRef != request.SubjectRef ||
		receipt.ProviderRef != request.ProviderRef ||
		receipt.ProducerRef != request.ProducerRef ||
		receipt.ProducerVersion != request.ProducerVersion ||
		receipt.ManifestDigest != request.ManifestDigest ||
		receipt.ArtifactRef != request.Descriptor.ArtifactRef ||
		receipt.DescriptorDigest != descriptorDigest ||
		resolution.ProvenanceDigest != provenanceDigest ||
		request.Descriptor.ArtifactRef.String() != "artifact:sha256:"+contentDigestText ||
		request.Descriptor.Digest != contentDigestText ||
		request.Descriptor.Size != int64(len(resolution.Content)) {
		return "", invalidAppAnalysis("artifact.resolution", err)
	}
	return provenanceDigest, nil
}

func appAnalysisAttachmentFingerprint(
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	subjectManifestDigest string,
	authorizationReceiptRef string,
) string {
	return fingerprintFields(
		"orquesta.app-analysis.attach.v1",
		actorRef.String(),
		projectRef.String(),
		subjectManifestDigest,
		authorizationReceiptRef,
	)
}

func buildAppAnalysisAttachmentReceipt(
	requestRef string,
	requestFingerprint string,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	subjectManifestDigest string,
	analysis AppAnalysis,
	authorizationReceiptRef string,
) (AppAnalysisAttachmentReceipt, error) {
	intakeEvidence := analysis.Intake()
	if !validIntakeRequestRef(requestRef) ||
		!validAppAnalysisDigest(requestFingerprint) ||
		actorRef.String() == "" ||
		projectRef.String() == "" ||
		!validAppAnalysisDigest(subjectManifestDigest) ||
		!validAppAnalysisRef(analysis.Ref()) ||
		!validAppAnalysisDigest(analysis.Digest()) ||
		!validAppAnalysisOpaque(authorizationReceiptRef, 512) ||
		!validAppAnalysisOpaque(analysis.Manifest().ReviewReceiptRef, 512) ||
		!validAppAnalysisDigest(analysis.VerifiedIntakeDigest()) ||
		!validAppAnalysisDigest(analysis.VerifiedReviewDigest()) ||
		!validAppAnalysisOpaque(intakeEvidence.ReceiptRef, 512) ||
		!validAppAnalysisDigest(intakeEvidence.SnapshotDigest) {
		return AppAnalysisAttachmentReceipt{}, invalidAppAnalysis("receipt", nil)
	}
	reviewReceiptRef := analysis.Manifest().ReviewReceiptRef
	refDigest := fingerprintFields(
		"orquesta.app-analysis.attachment-receipt.v1",
		requestRef,
		requestFingerprint,
		actorRef.String(),
		projectRef.String(),
		string(analysis.Ref()),
		analysis.Digest(),
		subjectManifestDigest,
		authorizationReceiptRef,
		reviewReceiptRef,
		analysis.VerifiedReviewDigest(),
		intakeEvidence.ReceiptRef,
		intakeEvidence.SnapshotDigest,
		analysis.VerifiedIntakeDigest(),
	)
	return AppAnalysisAttachmentReceipt{
		Ref:        "app-analysis-attachment-receipt:" + refDigest,
		RequestRef: requestRef, RequestFingerprint: requestFingerprint,
		ActorRef: actorRef, ProjectRef: projectRef,
		AnalysisRef: analysis.Ref(), AnalysisDigest: analysis.Digest(),
		ManifestDigest:          analysis.Digest(),
		SubjectManifestDigest:   subjectManifestDigest,
		AuthorizationReceiptRef: authorizationReceiptRef,
		ReviewReceiptRef:        reviewReceiptRef,
		VerifiedReviewDigest:    analysis.VerifiedReviewDigest(),
		IntakeReceiptRef:        intakeEvidence.ReceiptRef,
		IntakeSnapshotDigest:    intakeEvidence.SnapshotDigest,
		VerifiedIntakeDigest:    analysis.VerifiedIntakeDigest(),
	}, nil
}

func validateStoredAppAnalysisAttachmentRecord(
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	analysisRef AppAnalysisRef,
	record AppAnalysisAttachmentRecord,
) error {
	analysis, err := RestoreAppAnalysis(record.Analysis.Snapshot())
	if err != nil {
		return &StateError{Code: StateConflict, Cause: err}
	}
	if record.ActorRef != actorRef ||
		record.ProjectRef != projectRef ||
		analysis.Intake().ActorRef != actorRef ||
		analysis.Intake().ProjectRef != projectRef ||
		analysis.Review().ActorRef != actorRef ||
		analysis.Review().ProjectRef != projectRef ||
		record.Analysis.Ref() != analysisRef ||
		record.Analysis.Ref() != analysis.Ref() ||
		record.Analysis.Digest() != analysis.Digest() {
		return &StateError{Code: StateConflict}
	}
	subjectDigest, err := AppAnalysisManifestDigest(
		appAnalysisSubjectManifest(analysis.Manifest()),
	)
	if err != nil {
		return &StateError{Code: StateConflict, Cause: err}
	}
	receipt := record.Receipt
	fingerprint := appAnalysisAttachmentFingerprint(
		actorRef,
		projectRef,
		subjectDigest,
		receipt.AuthorizationReceiptRef,
	)
	if receipt.RequestFingerprint != fingerprint {
		return &StateError{Code: StateConflict}
	}
	expected, receiptErr := buildAppAnalysisAttachmentReceipt(
		receipt.RequestRef,
		receipt.RequestFingerprint,
		actorRef,
		projectRef,
		subjectDigest,
		analysis,
		receipt.AuthorizationReceiptRef,
	)
	if receiptErr != nil || expected != receipt {
		return &StateError{Code: StateConflict, Cause: receiptErr}
	}
	return nil
}

func validateAppAnalysisReplayRecord(
	request AppAnalysisReplayRequest,
	record AppAnalysisAttachmentRecord,
) error {
	if err := validateStoredAppAnalysisAttachmentRecord(
		request.ActorRef,
		request.ProjectRef,
		record.Analysis.Ref(),
		record,
	); err != nil {
		return err
	}
	manifest := record.Analysis.Manifest()
	receipt := record.Receipt
	if receipt.RequestRef != request.RequestRef ||
		receipt.RequestFingerprint != request.RequestFingerprint ||
		receipt.SubjectManifestDigest != request.SubjectManifestDigest ||
		receipt.AuthorizationReceiptRef != request.AuthorizationReceiptRef ||
		receipt.ReviewReceiptRef != request.ReviewReceiptRef ||
		manifest.IntakeRef != request.IntakeRef ||
		manifest.ExpectedRevision != request.ExpectedRevision {
		return &StateError{Code: StateConflict}
	}
	return nil
}

func validateAppAnalysisMutationRecord(
	replay AppAnalysisReplayRequest,
	expected AppAnalysis,
	record AppAnalysisAttachmentRecord,
) error {
	if !reflect.DeepEqual(record.Analysis.Snapshot(), expected.Snapshot()) {
		return &StateError{Code: StateConflict}
	}
	return validateAppAnalysisReplayRecord(replay, record)
}

func cloneAppAnalysisAttachmentRecord(
	record AppAnalysisAttachmentRecord,
) (AppAnalysisAttachmentRecord, error) {
	analysis, err := RestoreAppAnalysis(record.Analysis.Snapshot())
	if err != nil {
		return AppAnalysisAttachmentRecord{}, err
	}
	record.Analysis = analysis
	return record, nil
}
