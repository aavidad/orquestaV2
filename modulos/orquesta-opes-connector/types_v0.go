package orquestaopesconnector

import (
	"encoding/json"
	"net/http"
	"time"
)

const (
	DefaultOPESCreateJobPathV0 = "/api/jobs"

	ErrOPESBaseURLRequiredV0      = "opes_base_url_required"
	ErrOPESHTTPRequestFailedV0    = "opes_http_request_failed"
	ErrOPESHTTPStatusV0           = "opes_http_status"
	ErrOPESHTTPTimeoutV0          = "opes_http_timeout"
	ErrOPESHTTPCancelledV0        = "opes_http_cancelled"
	ErrOPESResponseInvalidV0      = "opes_response_invalid"
	ErrOPESResponseBodyTooLargeV0 = "opes_response_body_too_large"
	ErrOPESResponseContentTypeV0  = "opes_response_content_type"
	ErrOPESResponseTrailingDataV0 = "opes_response_trailing_data"
	ErrOPESJobRefMissingV0        = "opes_job_ref_missing"
	ErrOPESDomainWorkInvalidV0    = "opes_domain_work_invalid"
	ErrOPESArtifactJobRequiredV0  = "opes_artifact_job_required"
	ErrOPESPathInvalidV0          = "opes_path_invalid"
	ErrOPESHTTPRedirectDeniedV0   = "opes_http_redirect_denied"
	ErrOPESRetryBlockedV0         = "non_idempotent_mutation_retry_blocked"
	ErrOPESRetryBudgetExhaustedV0 = "retry_budget_exhausted"
)

type ExternalJobQueryV0 struct {
	ExecutionMode string
	Status        string
	JobType       string
	JobRef        string
	Limit         int
}

type ExternalJobV0 struct {
	ID             string            `json:"id"`
	Type           string            `json:"type"`
	Status         string            `json:"status"`
	ExecutionMode  string            `json:"execution_mode"`
	PayloadJSON    string            `json:"payload_json"`
	CorrelationID  string            `json:"correlation_id,omitempty"`
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
	RequestedBy    string            `json:"requested_by,omitempty"`
	Attempts       int               `json:"attempts,omitempty"`
	MaxAttempts    int               `json:"max_attempts,omitempty"`
	LastError      string            `json:"last_error,omitempty"`
	LockedBy       string            `json:"locked_by,omitempty"`
	ExternalRefs   map[string]string `json:"external_refs,omitempty"`
	CreatedAt      string            `json:"created_at,omitempty"`
}

type TopicBlockV0 struct {
	ID               string          `json:"ID"`
	StableID         string          `json:"StableID"`
	CanonicalTopicID string          `json:"CanonicalTopicID"`
	ChapterID        string          `json:"ChapterID"`
	Type             string          `json:"Type"`
	Status           string          `json:"Status"`
	Title            string          `json:"Title"`
	Markdown         string          `json:"Markdown"`
	LanguageCode     string          `json:"LanguageCode"`
	SourceRefs       []string        `json:"SourceRefs"`
	Citations        json.RawMessage `json:"Citations,omitempty"`
	VersionNumber    int             `json:"VersionNumber,omitempty"`
	CreatedAt        string          `json:"CreatedAt,omitempty"`
}

type RESTClientConfigV0 struct {
	BaseURL            string
	HTTPClient         *http.Client
	Timeout            time.Duration
	DefaultMaxAttempts int
	RetryPolicy        RetryPolicyV0
}

type RESTClientV0 struct {
	baseURL            string
	httpClient         *http.Client
	timeout            time.Duration
	defaultMaxAttempts int
	retryPolicy        RetryPolicyV0
}

func NewRESTClientV0(config RESTClientConfigV0) RESTClientV0 {
	client := config.HTTPClient
	if client == nil {
		client = newOPESHTTPClientV0(opesRESTTimeoutV0(config.Timeout))
	}
	client = opesHTTPClientWithRedirectPolicyV0(client, opesRESTTimeoutV0(config.Timeout), config.BaseURL)
	maxAttempts := config.DefaultMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	return RESTClientV0{
		baseURL:            trimTrailingSlashV0(config.BaseURL),
		httpClient:         client,
		timeout:            opesRESTTimeoutV0(config.Timeout),
		defaultMaxAttempts: maxAttempts,
		retryPolicy:        normalizeRetryPolicyV0(config.RetryPolicy),
	}
}

type connectorErrorV0 struct {
	code string
}

func (err connectorErrorV0) Error() string {
	return err.code
}
