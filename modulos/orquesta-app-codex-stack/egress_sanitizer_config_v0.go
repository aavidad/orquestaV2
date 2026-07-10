package orquestaappcodexstack

import "strings"

type RuntimeProviderKindV0 string

const (
	RuntimeProviderCodexV0  RuntimeProviderKindV0 = "codex"
	RuntimeProviderGeminiV0 RuntimeProviderKindV0 = "gemini"
	RuntimeProviderClaudeV0 RuntimeProviderKindV0 = "claude"

	EgressSanitizerProviderLocalSensitiveDataV0 RuntimeProviderKindV0 = "local_sensitive_data"
	EgressSanitizerProviderOpenAIPrivacyLocalV0 RuntimeProviderKindV0 = "openai_privacy_filter_local"
)

type RuntimeProviderConfigV0 struct {
	ProviderRef              string                `json:"provider_ref"`
	Kind                     RuntimeProviderKindV0 `json:"kind"`
	Enabled                  bool                  `json:"enabled"`
	CommandConfigured        bool                  `json:"command_configured,omitempty"`
	LocalEndpointConfigured  bool                  `json:"local_endpoint_configured,omitempty"`
	ProjectWorkDirConfigured bool                  `json:"project_work_dir_configured,omitempty"`
	RuntimeWorkDirConfigured bool                  `json:"runtime_work_dir_configured,omitempty"`
	HomeDirConfigured        bool                  `json:"home_dir_configured,omitempty"`
	CodeHomeDirConfigured    bool                  `json:"code_home_dir_configured,omitempty"`
	PathEnvConfigured        bool                  `json:"path_env_configured,omitempty"`
	ModelRef                 string                `json:"model_ref,omitempty"`
	ReasoningEffort          string                `json:"reasoning_effort,omitempty"`
	Sandbox                  string                `json:"sandbox,omitempty"`
	ApprovalPolicy           string                `json:"approval_policy,omitempty"`
	ExtraArgsCount           int                   `json:"extra_args_count,omitempty"`
	PromptHintsCount         int                   `json:"prompt_hints_count,omitempty"`
}

type EgressSanitizerConfigV0 struct {
	Enabled      bool                         `json:"enabled"`
	SanitizerRef string                       `json:"sanitizer_ref,omitempty"`
	ProviderRef  RuntimeProviderKindV0        `json:"provider_ref,omitempty"`
	LocalModel   PrivacyFilterModelConfigV0   `json:"local_model,omitempty"`
	Sidecar      PrivacyFilterSidecarConfigV0 `json:"sidecar,omitempty"`
}

type PrivacyFilterModelConfigV0 struct {
	Enabled     bool   `json:"enabled"`
	ModelRef    string `json:"model_ref,omitempty"`
	RuntimeRef  string `json:"runtime_ref,omitempty"`
	EvidenceRef string `json:"evidence_ref,omitempty"`
}

type PrivacyFilterSidecarConfigV0 struct {
	Enabled                 bool                       `json:"enabled"`
	SidecarRef              string                     `json:"sidecar_ref,omitempty"`
	AdapterRef              string                     `json:"adapter_ref,omitempty"`
	TransportRef            string                     `json:"transport_ref,omitempty"`
	EvidenceRef             string                     `json:"evidence_ref,omitempty"`
	CommandConfigured       bool                       `json:"command_configured,omitempty"`
	LocalEndpointConfigured bool                       `json:"local_endpoint_configured,omitempty"`
	Port                    PrivacyFilterSidecarPortV0 `json:"-"`
}

type PrivacyFilterSidecarPortV0 interface {
	FilterEgressPayloadV0(PrivacyFilterSidecarRequestV0) PrivacyFilterSidecarResultV0
}

type PrivacyFilterSidecarRequestV0 struct {
	RequestRef string `json:"request_ref,omitempty"`
	EntryRef   string `json:"entry_ref,omitempty"`
	SourceRef  string `json:"source_ref,omitempty"`
	Payload    string `json:"payload,omitempty"`
}

type PrivacyFilterSidecarResultV0 struct {
	Content          string   `json:"content,omitempty"`
	Sanitized        bool     `json:"sanitized,omitempty"`
	ReviewRequired   bool     `json:"review_required,omitempty"`
	EvidenceRefs     []string `json:"evidence_refs,omitempty"`
	Categories       []string `json:"categories,omitempty"`
	ReplacementCount int      `json:"replacement_count,omitempty"`
}

func NormalizeEgressSanitizerConfigV0(config EgressSanitizerConfigV0) EgressSanitizerConfigV0 {
	config.SanitizerRef = firstCodexStackStringV0(config.SanitizerRef, "sanitizer-ref-local-sensitive-data-v0")
	if config.ProviderRef == "" {
		config.ProviderRef = EgressSanitizerProviderLocalSensitiveDataV0
	}
	if config.LocalModel.Enabled {
		config.ProviderRef = EgressSanitizerProviderOpenAIPrivacyLocalV0
		config.LocalModel.ModelRef = firstCodexStackStringV0(
			config.LocalModel.ModelRef,
			"model-ref-openai-privacy-filter-local",
		)
		config.LocalModel.RuntimeRef = firstCodexStackStringV0(
			config.LocalModel.RuntimeRef,
			"runtime-ref-local-sanitizer",
		)
	}
	if config.Sidecar.Enabled {
		config.ProviderRef = EgressSanitizerProviderOpenAIPrivacyLocalV0
		config.Sidecar.SidecarRef = firstCodexStackStringV0(
			config.Sidecar.SidecarRef,
			"sidecar-ref-openai-privacy-filter-local",
		)
		config.Sidecar.AdapterRef = firstCodexStackStringV0(
			config.Sidecar.AdapterRef,
			"adapter-ref-openai-privacy-filter-local",
		)
		config.Sidecar.TransportRef = firstCodexStackStringV0(
			config.Sidecar.TransportRef,
			"transport-ref-local-sidecar",
		)
		config.Sidecar.EvidenceRef = firstCodexStackStringV0(
			config.Sidecar.EvidenceRef,
			"evidence-ref-openai-privacy-filter-local",
		)
	}
	return config
}

func CanonicalRuntimeProviderConfigsV0(config ConfigV0) []RuntimeProviderConfigV0 {
	providers := []RuntimeProviderConfigV0{canonicalCodexRuntimeProviderConfigV0(config.Codex)}
	if config.Gemini.Enabled {
		providers = append(providers, canonicalGeminiRuntimeProviderConfigV0(config.Gemini))
	}
	if config.Claude.Enabled {
		providers = append(providers, canonicalClaudeRuntimeProviderConfigV0(config.Claude))
	}
	if config.EgressSanitizer.Enabled {
		providers = append(providers, canonicalEgressSanitizerProviderConfigV0(config.EgressSanitizer))
	}
	return providers
}

func configWithCanonicalEgressSanitizerV0(config ConfigV0) ConfigV0 {
	config.EgressSanitizer = NormalizeEgressSanitizerConfigV0(config.EgressSanitizer)
	if !config.EgressSanitizer.Enabled || config.Codex.ContextSanitizer != nil {
		return config
	}
	config.Codex.ContextSanitizer = NewEgressSanitizerContextSanitizerV0(config.EgressSanitizer)
	return config
}

func (sanitizer LocalSensitiveDataSanitizerV0) egressCategoriesV0(
	categories map[string]bool,
) map[string]bool {
	config := NormalizeEgressSanitizerConfigV0(sanitizer.Egress)
	if !config.Enabled {
		return categories
	}
	if categories == nil {
		categories = map[string]bool{}
	}
	categories["egress_sanitizer"] = true
	if config.LocalModel.Enabled {
		categories["local_privacy_filter_model"] = true
	}
	if config.Sidecar.Enabled {
		if config.Sidecar.Port != nil {
			categories["openai_privacy_filter_local"] = true
			categories["local_privacy_filter_sidecar"] = true
		} else {
			categories["local_privacy_filter_sidecar_configured"] = true
		}
	}
	return categories
}

func canonicalCodexRuntimeProviderConfigV0(config CodexRuntimeConfigV0) RuntimeProviderConfigV0 {
	return RuntimeProviderConfigV0{
		ProviderRef:              "provider-ref-codex",
		Kind:                     RuntimeProviderCodexV0,
		Enabled:                  true,
		CommandConfigured:        strings.TrimSpace(config.CommandPath) != "",
		ProjectWorkDirConfigured: strings.TrimSpace(config.ProjectWorkDir) != "",
		RuntimeWorkDirConfigured: strings.TrimSpace(config.RuntimeWorkDir) != "",
		HomeDirConfigured:        strings.TrimSpace(config.HomeDir) != "",
		CodeHomeDirConfigured:    strings.TrimSpace(config.CodeHomeDir) != "",
		PathEnvConfigured:        strings.TrimSpace(config.PathEnv) != "",
		ModelRef:                 strings.TrimSpace(config.Model),
		ReasoningEffort:          strings.TrimSpace(config.ReasoningEffort),
		Sandbox:                  strings.TrimSpace(config.Sandbox),
		ApprovalPolicy:           strings.TrimSpace(config.ApprovalPolicy),
		ExtraArgsCount:           len(compactStringsV0(config.ExtraArgs)),
		PromptHintsCount:         len(compactStringsV0(config.PromptHints)),
	}
}

func canonicalGeminiRuntimeProviderConfigV0(config GeminiRuntimeConfigV0) RuntimeProviderConfigV0 {
	return RuntimeProviderConfigV0{
		ProviderRef:              "provider-ref-gemini",
		Kind:                     RuntimeProviderGeminiV0,
		Enabled:                  config.Enabled,
		CommandConfigured:        strings.TrimSpace(config.CommandPath) != "",
		ProjectWorkDirConfigured: strings.TrimSpace(config.ProjectWorkDir) != "",
		RuntimeWorkDirConfigured: strings.TrimSpace(config.RuntimeWorkDir) != "",
		HomeDirConfigured:        strings.TrimSpace(config.HomeDir) != "",
		PathEnvConfigured:        strings.TrimSpace(config.PathEnv) != "",
		ModelRef:                 strings.TrimSpace(config.Model),
		ApprovalPolicy:           strings.TrimSpace(config.ApprovalMode),
		ExtraArgsCount:           len(compactStringsV0(config.ExtraArgs)),
		PromptHintsCount:         len(compactStringsV0(config.PromptHints)),
	}
}

func canonicalClaudeRuntimeProviderConfigV0(config ClaudeRuntimeConfigV0) RuntimeProviderConfigV0 {
	return RuntimeProviderConfigV0{
		ProviderRef:              "provider-ref-claude",
		Kind:                     RuntimeProviderClaudeV0,
		Enabled:                  config.Enabled,
		CommandConfigured:        strings.TrimSpace(config.CommandPath) != "",
		ProjectWorkDirConfigured: strings.TrimSpace(config.ProjectWorkDir) != "",
		RuntimeWorkDirConfigured: strings.TrimSpace(config.RuntimeWorkDir) != "",
		HomeDirConfigured:        strings.TrimSpace(config.HomeDir) != "",
		PathEnvConfigured:        strings.TrimSpace(config.PathEnv) != "",
		ModelRef:                 strings.TrimSpace(config.ModelRouting.Policy.PolicyRef),
		ReasoningEffort:          claudeModelRoutingEffortsV0(config.ModelRouting),
		ApprovalPolicy:           strings.TrimSpace(config.PermissionMode),
		ExtraArgsCount:           len(compactStringsV0(config.ExtraArgs)),
		PromptHintsCount:         len(compactStringsV0(config.PromptHints)),
	}
}

func claudeModelRoutingEffortsV0(config ClaudeModelRoutingConfigV0) string {
	return strings.Join([]string{
		strings.TrimSpace(config.Policy.TrivialEffort),
		strings.TrimSpace(config.Policy.NormalEffort),
		strings.TrimSpace(config.Policy.ComplexEffort),
		strings.TrimSpace(config.Policy.CriticalEffort),
	}, ",")
}

func canonicalEgressSanitizerProviderConfigV0(config EgressSanitizerConfigV0) RuntimeProviderConfigV0 {
	config = NormalizeEgressSanitizerConfigV0(config)
	return RuntimeProviderConfigV0{
		ProviderRef:             "provider-ref-egress-sanitizer",
		Kind:                    config.ProviderRef,
		Enabled:                 config.Enabled,
		CommandConfigured:       config.Sidecar.Enabled && config.Sidecar.CommandConfigured,
		LocalEndpointConfigured: config.Sidecar.Enabled && config.Sidecar.LocalEndpointConfigured,
		ModelRef:                egressSanitizerProviderModelRefV0(config),
	}
}

func egressSanitizerProviderModelRefV0(config EgressSanitizerConfigV0) string {
	if config.Sidecar.Enabled {
		return ""
	}
	return strings.TrimSpace(config.LocalModel.ModelRef)
}
