package orquestaopesconnector

import (
	"net/http"
	"time"
)

const (
	DefaultOPESCreateJobPathV0 = "/api/jobs"

	ErrOPESBaseURLRequiredV0     = "opes_base_url_required"
	ErrOPESHTTPStatusV0          = "opes_http_status"
	ErrOPESResponseInvalidV0     = "opes_response_invalid"
	ErrOPESJobRefMissingV0       = "opes_job_ref_missing"
	ErrOPESDomainWorkInvalidV0   = "opes_domain_work_invalid"
	ErrOPESArtifactJobRequiredV0 = "opes_artifact_job_required"
)

type ExternalJobQueryV0 struct {
	ExecutionMode string
	Status        string
	JobType       string
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

type RESTClientConfigV0 struct {
	BaseURL            string
	HTTPClient         *http.Client
	DefaultMaxAttempts int
}

type RESTClientV0 struct {
	baseURL            string
	httpClient         *http.Client
	defaultMaxAttempts int
}

func NewRESTClientV0(config RESTClientConfigV0) RESTClientV0 {
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	maxAttempts := config.DefaultMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	return RESTClientV0{
		baseURL:            trimTrailingSlashV0(config.BaseURL),
		httpClient:         client,
		defaultMaxAttempts: maxAttempts,
	}
}

type connectorErrorV0 struct {
	code string
}

func (err connectorErrorV0) Error() string {
	return err.code
}
