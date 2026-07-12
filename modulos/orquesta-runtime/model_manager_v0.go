package orquestaruntime

import "context"

type RuntimeModelManagerPortV0 interface {
	ListRuntimeModelsV0(context.Context, RuntimeModelListRequestV0) (RuntimeModelListResultV0, error)
	PullRuntimeModelV0(context.Context, RuntimeModelActionRequestV0) (RuntimeModelActionResultV0, error)
	ServeRuntimeModelV0(context.Context, RuntimeModelActionRequestV0) (RuntimeModelActionResultV0, error)
	StopRuntimeModelV0(context.Context, RuntimeModelActionRequestV0) (RuntimeModelActionResultV0, error)
	RuntimeModelStatusV0(context.Context, RuntimeModelListRequestV0) (RuntimeModelListResultV0, error)
}

type RuntimeModelListRequestV0 struct {
	ProviderRef string   `json:"provider_ref,omitempty"`
	EndpointRef string   `json:"endpoint_ref,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type RuntimeModelActionRequestV0 struct {
	OperationRef string   `json:"operation_ref"`
	ProviderRef  string   `json:"provider_ref,omitempty"`
	EndpointRef  string   `json:"endpoint_ref,omitempty"`
	Model        string   `json:"model"`
	KeepAlive    string   `json:"keep_alive,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Evidence     []string `json:"evidence_refs,omitempty"`
}

type RuntimeModelListResultV0 struct {
	ProviderRef string                 `json:"provider_ref,omitempty"`
	EndpointRef string                 `json:"endpoint_ref,omitempty"`
	BaseURL     string                 `json:"base_url,omitempty"`
	Models      []RuntimeModelInfoV0   `json:"models,omitempty"`
	Evidence    []RuntimeModelEvidence `json:"evidence,omitempty"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
}

type RuntimeModelActionResultV0 struct {
	ProviderRef string                 `json:"provider_ref,omitempty"`
	EndpointRef string                 `json:"endpoint_ref,omitempty"`
	BaseURL     string                 `json:"base_url,omitempty"`
	Model       string                 `json:"model,omitempty"`
	Accepted    bool                   `json:"accepted"`
	Status      string                 `json:"status,omitempty"`
	Evidence    []RuntimeModelEvidence `json:"evidence,omitempty"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
}

type RuntimeModelInfoV0 struct {
	Name       string            `json:"name"`
	Provider   string            `json:"provider,omitempty"`
	Family     string            `json:"family,omitempty"`
	Parameter  string            `json:"parameter_size,omitempty"`
	Quant      string            `json:"quantization,omitempty"`
	SizeBytes  int64             `json:"size_bytes,omitempty"`
	SizeVRAM   int64             `json:"size_vram_bytes,omitempty"`
	Digest     string            `json:"digest,omitempty"`
	ModifiedAt string            `json:"modified_at,omitempty"`
	ExpiresAt  string            `json:"expires_at,omitempty"`
	Status     string            `json:"status,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type RuntimeModelEvidence struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref"`
}
