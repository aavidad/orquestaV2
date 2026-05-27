package orquestadomainworkhttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const (
	DefaultDomainWorkHTTPCreateJobPathV0       = "/api/domain-work/jobs"
	DefaultDomainWorkHTTPSubmitArtifactPathV0  = "/api/domain-work/artifacts"
	defaultDomainWorkHTTPTimeoutV0             = 30 * time.Second
	ErrDomainWorkHTTPBaseURLRequiredV0         = "domain_work_http_base_url_required"
	ErrDomainWorkHTTPBaseURLInvalidV0          = "domain_work_http_base_url_invalid"
	ErrDomainWorkHTTPEgressPolicyRequiredV0    = "domain_work_http_egress_policy_required"
	ErrDomainWorkHTTPEgressDestinationDeniedV0 = "domain_work_http_egress_destination_denied"
	ErrDomainWorkHTTPRequestBuildFailedV0      = "domain_work_http_request_build_failed"
	ErrDomainWorkHTTPRequestFailedV0           = "domain_work_http_request_failed"
	ErrDomainWorkHTTPTimeoutV0                 = "domain_work_http_timeout"
	ErrDomainWorkHTTPCancelledV0               = "domain_work_http_cancelled"
	ErrDomainWorkHTTPStatusFailedV0            = "domain_work_http_status_failed"
	ErrDomainWorkHTTPResponseDecodeFailedV0    = "domain_work_http_response_decode_failed"
	ErrDomainWorkHTTPResponseBodyTooLargeV0    = "domain_work_http_response_body_too_large"
	ErrDomainWorkHTTPResponseContentTypeV0     = "domain_work_http_response_content_type"
	ErrDomainWorkHTTPResponseTrailingDataV0    = "domain_work_http_response_trailing_data"
	ErrDomainWorkHTTPResponseJobMissingV0      = "domain_work_http_response_job_missing"
	ErrDomainWorkHTTPResponseReceiptMissingV0  = "domain_work_http_response_receipt_missing"
	ErrDomainWorkHTTPCreatePathInvalidV0       = "domain_work_http_create_path_invalid"
	ErrDomainWorkHTTPSubmitPathInvalidV0       = "domain_work_http_submit_path_invalid"
	ErrDomainWorkHTTPRedirectDeniedV0          = "domain_work_http_redirect_denied"
	ErrDomainWorkHTTPRetryBlockedV0            = "non_idempotent_mutation_retry_blocked"
	ErrDomainWorkHTTPRetryBudgetExhaustedV0    = "retry_budget_exhausted"
)

var _ orquestadomainwork.DomainWorkJobCreatorPortV0 = ClientV0{}
var _ orquestadomainwork.DomainWorkArtifactSubmitterPortV0 = ClientV0{}

type ConfigV0 struct {
	BaseURL            string
	CreateJobPath      string
	SubmitArtifactPath string
	EgressPolicy       EgressPolicyV0
	HTTPClient         *http.Client
	Timeout            time.Duration
	RetryPolicy        RetryPolicyV0
}

type ClientV0 struct {
	baseURL            string
	createJobPath      string
	submitArtifactPath string
	httpClient         *http.Client
	timeout            time.Duration
	destination        DestinationV0
	retryPolicy        RetryPolicyV0
}

type ErrorV0 struct {
	Code string
}

func (err ErrorV0) Error() string {
	return strings.TrimSpace(err.Code)
}

func NewClientV0(config ConfigV0) (ClientV0, error) {
	destination, err := NormalizeDestinationV0(config.BaseURL, effectiveEgressPolicyV0(config.EgressPolicy))
	if err != nil {
		return ClientV0{}, err
	}
	createPath, err := normalizePathV0(config.CreateJobPath, DefaultDomainWorkHTTPCreateJobPathV0, ErrDomainWorkHTTPCreatePathInvalidV0)
	if err != nil {
		return ClientV0{}, err
	}
	submitPath, err := normalizePathV0(config.SubmitArtifactPath, DefaultDomainWorkHTTPSubmitArtifactPathV0, ErrDomainWorkHTTPSubmitPathInvalidV0)
	if err != nil {
		return ClientV0{}, err
	}
	httpClient := config.HTTPClient
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = defaultDomainWorkHTTPTimeoutV0
	}
	if httpClient == nil {
		httpClient = newDomainWorkHTTPClientV0(timeout)
	}
	httpClient = domainWorkHTTPClientWithRedirectPolicyV0(httpClient, timeout, destination, createPath, submitPath)
	return ClientV0{
		baseURL:            destination.BaseURL,
		createJobPath:      createPath,
		submitArtifactPath: submitPath,
		httpClient:         httpClient,
		timeout:            timeout,
		destination:        destination,
		retryPolicy:        normalizeRetryPolicyV0(config.RetryPolicy),
	}, nil
}

func (client ClientV0) DestinationV0() DestinationV0 {
	return client.destination
}

func (client ClientV0) CreateDomainWorkJobV0(
	ctx context.Context,
	request orquestadomainwork.DomainWorkJobRequestV0,
) (orquestadomainwork.DomainWorkJobV0, error) {
	request = orquestadomainwork.NormalizeDomainWorkJobRequestV0(request)
	if issues := orquestadomainwork.ValidateDomainWorkJobRequestV0(request); len(issues) > 0 {
		return orquestadomainwork.DomainWorkJobV0{
			SchemaVersion:  orquestadomainwork.DomainWorkJobSchemaV0,
			Status:         orquestadomainwork.DomainWorkStatusInvalidV0,
			DomainRef:      request.DomainRef,
			WorkKind:       request.WorkKind,
			CorrelationID:  request.CorrelationID,
			IdempotencyKey: request.IdempotencyKey,
			ExternalRefs:   request.ExternalRefs,
			EvidenceRefs:   request.EvidenceRefs,
			Issues:         issues,
		}, nil
	}
	var envelope domainWorkHTTPJobEnvelopeV0
	if err := client.postJSONV0(ctx, client.createJobPath, request, &envelope); err != nil {
		return orquestadomainwork.DomainWorkJobV0{}, err
	}
	job, ok := envelope.jobV0()
	if !ok {
		return orquestadomainwork.DomainWorkJobV0{}, ErrorV0{Code: ErrDomainWorkHTTPResponseJobMissingV0}
	}
	return job, nil
}

func (client ClientV0) SubmitDomainWorkArtifactV0(
	ctx context.Context,
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0,
) (orquestadomainwork.DomainWorkArtifactReceiptV0, error) {
	submission = orquestadomainwork.NormalizeDomainWorkArtifactSubmissionV0(submission)
	if issues := orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(submission); len(issues) > 0 {
		return orquestadomainwork.DomainWorkArtifactReceiptV0{
			SchemaVersion:  orquestadomainwork.DomainWorkArtifactReceiptSchemaV0,
			Status:         orquestadomainwork.DomainWorkStatusInvalidV0,
			JobRef:         submission.JobRef,
			ArtifactRef:    submission.ArtifactRef,
			CorrelationID:  submission.CorrelationID,
			IdempotencyKey: submission.IdempotencyKey,
			ExternalRefs:   submission.ExternalRefs,
			EvidenceRefs:   submission.EvidenceRefs,
			Issues:         issues,
		}, nil
	}
	var envelope domainWorkHTTPReceiptEnvelopeV0
	if err := client.postJSONV0(ctx, client.submitArtifactPath, submission, &envelope); err != nil {
		return orquestadomainwork.DomainWorkArtifactReceiptV0{}, err
	}
	receipt, ok := envelope.receiptV0()
	if !ok {
		return orquestadomainwork.DomainWorkArtifactReceiptV0{}, ErrorV0{Code: ErrDomainWorkHTTPResponseReceiptMissingV0}
	}
	return receipt, nil
}

func (client ClientV0) postJSONV0(
	ctx context.Context,
	path string,
	payload any,
	target any,
) error {
	ctx, cancel := client.effectContextV0(ctx)
	defer cancel()
	data, err := json.Marshal(payload)
	if err != nil {
		return ErrorV0{Code: ErrDomainWorkHTTPRequestBuildFailedV0}
	}
	return client.postJSONEncodedV0(ctx, path, data, retryKeyFromPayloadV0(payload), target)
}

func (client ClientV0) effectContextV0(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, client.timeout)
}

func domainWorkHTTPRequestErrorCodeV0(ctx context.Context, err error) string {
	if code, ok := domainWorkHTTPRedirectErrorCodeV0(err); ok {
		return code
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded), errors.Is(ctx.Err(), context.DeadlineExceeded):
		return ErrDomainWorkHTTPTimeoutV0
	case errors.Is(err, context.Canceled), errors.Is(ctx.Err(), context.Canceled):
		return ErrDomainWorkHTTPCancelledV0
	default:
		return ErrDomainWorkHTTPRequestFailedV0
	}
}

type domainWorkHTTPJobEnvelopeV0 struct {
	SchemaVersion  string                                       `json:"schema_version,omitempty"`
	Status         string                                       `json:"status,omitempty"`
	JobRef         string                                       `json:"job_ref,omitempty"`
	DomainRef      string                                       `json:"domain_ref,omitempty"`
	WorkKind       string                                       `json:"work_kind,omitempty"`
	CorrelationID  string                                       `json:"correlation_id,omitempty"`
	IdempotencyKey string                                       `json:"idempotency_key,omitempty"`
	ExternalRefs   []orquestadomainwork.DomainWorkExternalRefV0 `json:"external_refs,omitempty"`
	EvidenceRefs   []string                                     `json:"evidence_refs,omitempty"`
	Issues         []orquestadomainwork.DomainWorkIssueV0       `json:"issues,omitempty"`
	Job            *orquestadomainwork.DomainWorkJobV0          `json:"job,omitempty"`
}

func (envelope domainWorkHTTPJobEnvelopeV0) jobV0() (orquestadomainwork.DomainWorkJobV0, bool) {
	if envelope.Job != nil {
		return *envelope.Job, strings.TrimSpace(envelope.Job.JobRef) != ""
	}
	job := orquestadomainwork.DomainWorkJobV0{
		SchemaVersion:  envelope.SchemaVersion,
		Status:         envelope.Status,
		JobRef:         envelope.JobRef,
		DomainRef:      envelope.DomainRef,
		WorkKind:       envelope.WorkKind,
		CorrelationID:  envelope.CorrelationID,
		IdempotencyKey: envelope.IdempotencyKey,
		ExternalRefs:   envelope.ExternalRefs,
		EvidenceRefs:   envelope.EvidenceRefs,
		Issues:         envelope.Issues,
	}
	if job.SchemaVersion == "" {
		job.SchemaVersion = orquestadomainwork.DomainWorkJobSchemaV0
	}
	return job, strings.TrimSpace(job.JobRef) != ""
}

type domainWorkHTTPReceiptEnvelopeV0 struct {
	SchemaVersion  string                                          `json:"schema_version,omitempty"`
	Status         string                                          `json:"status,omitempty"`
	JobRef         string                                          `json:"job_ref,omitempty"`
	ArtifactRef    string                                          `json:"artifact_ref,omitempty"`
	ReceiptRef     string                                          `json:"receipt_ref,omitempty"`
	CorrelationID  string                                          `json:"correlation_id,omitempty"`
	IdempotencyKey string                                          `json:"idempotency_key,omitempty"`
	ExternalRefs   []orquestadomainwork.DomainWorkExternalRefV0    `json:"external_refs,omitempty"`
	EvidenceRefs   []string                                        `json:"evidence_refs,omitempty"`
	Issues         []orquestadomainwork.DomainWorkIssueV0          `json:"issues,omitempty"`
	Receipt        *orquestadomainwork.DomainWorkArtifactReceiptV0 `json:"receipt,omitempty"`
}

func (envelope domainWorkHTTPReceiptEnvelopeV0) receiptV0() (orquestadomainwork.DomainWorkArtifactReceiptV0, bool) {
	if envelope.Receipt != nil {
		return *envelope.Receipt, strings.TrimSpace(envelope.Receipt.ReceiptRef) != ""
	}
	receipt := orquestadomainwork.DomainWorkArtifactReceiptV0{
		SchemaVersion:  envelope.SchemaVersion,
		Status:         envelope.Status,
		JobRef:         envelope.JobRef,
		ArtifactRef:    envelope.ArtifactRef,
		ReceiptRef:     envelope.ReceiptRef,
		CorrelationID:  envelope.CorrelationID,
		IdempotencyKey: envelope.IdempotencyKey,
		ExternalRefs:   envelope.ExternalRefs,
		EvidenceRefs:   envelope.EvidenceRefs,
		Issues:         envelope.Issues,
	}
	if receipt.SchemaVersion == "" {
		receipt.SchemaVersion = orquestadomainwork.DomainWorkArtifactReceiptSchemaV0
	}
	return receipt, strings.TrimSpace(receipt.ReceiptRef) != ""
}

func normalizePathV0(raw string, fallback string, errCode string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = fallback
	}
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") ||
		strings.ContainsAny(raw, " \t\r\n?#\\") ||
		strings.Contains(raw, "://") {
		return "", ErrorV0{Code: errCode}
	}
	if parsed, err := url.Parse(raw); err != nil || parsed.IsAbs() || parsed.Host != "" {
		return "", ErrorV0{Code: errCode}
	}
	return raw, nil
}
