package orquestadomainworkhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const (
	DefaultDomainWorkHTTPCreateJobPathV0      = "/api/domain-work/jobs"
	DefaultDomainWorkHTTPSubmitArtifactPathV0 = "/api/domain-work/artifacts"
	defaultDomainWorkHTTPTimeoutV0            = 30 * time.Second
	ErrDomainWorkHTTPBaseURLRequiredV0        = "domain_work_http_base_url_required"
	ErrDomainWorkHTTPBaseURLInvalidV0         = "domain_work_http_base_url_invalid"
	ErrDomainWorkHTTPRequestBuildFailedV0     = "domain_work_http_request_build_failed"
	ErrDomainWorkHTTPRequestFailedV0          = "domain_work_http_request_failed"
	ErrDomainWorkHTTPStatusFailedV0           = "domain_work_http_status_failed"
	ErrDomainWorkHTTPResponseDecodeFailedV0   = "domain_work_http_response_decode_failed"
	ErrDomainWorkHTTPResponseJobMissingV0     = "domain_work_http_response_job_missing"
	ErrDomainWorkHTTPResponseReceiptMissingV0 = "domain_work_http_response_receipt_missing"
	ErrDomainWorkHTTPCreatePathInvalidV0      = "domain_work_http_create_path_invalid"
	ErrDomainWorkHTTPSubmitPathInvalidV0      = "domain_work_http_submit_path_invalid"
)

var _ orquestadomainwork.DomainWorkJobCreatorPortV0 = ClientV0{}
var _ orquestadomainwork.DomainWorkArtifactSubmitterPortV0 = ClientV0{}

type ConfigV0 struct {
	BaseURL            string
	CreateJobPath      string
	SubmitArtifactPath string
	HTTPClient         *http.Client
	Timeout            time.Duration
}

type ClientV0 struct {
	baseURL            string
	createJobPath      string
	submitArtifactPath string
	httpClient         *http.Client
}

type ErrorV0 struct {
	Code string
}

func (err ErrorV0) Error() string {
	return strings.TrimSpace(err.Code)
}

func NewClientV0(config ConfigV0) (ClientV0, error) {
	baseURL, err := normalizeBaseURLV0(config.BaseURL)
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
	if httpClient == nil {
		timeout := config.Timeout
		if timeout <= 0 {
			timeout = defaultDomainWorkHTTPTimeoutV0
		}
		httpClient = &http.Client{Timeout: timeout}
	}
	return ClientV0{
		baseURL:            baseURL,
		createJobPath:      createPath,
		submitArtifactPath: submitPath,
		httpClient:         httpClient,
	}, nil
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
	if ctx == nil {
		ctx = context.Background()
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return ErrorV0{Code: ErrDomainWorkHTTPRequestBuildFailedV0}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return ErrorV0{Code: ErrDomainWorkHTTPRequestBuildFailedV0}
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.httpClient.Do(request)
	if err != nil {
		return ErrorV0{Code: ErrDomainWorkHTTPRequestFailedV0}
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return ErrorV0{Code: fmt.Sprintf("%s_%d", ErrDomainWorkHTTPStatusFailedV0, response.StatusCode)}
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return ErrorV0{Code: ErrDomainWorkHTTPResponseDecodeFailedV0}
	}
	return nil
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

func normalizeBaseURLV0(raw string) (string, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return "", ErrorV0{Code: ErrDomainWorkHTTPBaseURLRequiredV0}
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", ErrorV0{Code: ErrDomainWorkHTTPBaseURLInvalidV0}
	}
	return raw, nil
}

func normalizePathV0(raw string, fallback string, errCode string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = fallback
	}
	if !strings.HasPrefix(raw, "/") || strings.ContainsAny(raw, " \t\r\n") {
		return "", ErrorV0{Code: errCode}
	}
	return raw, nil
}
