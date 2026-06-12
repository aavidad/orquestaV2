package orquestaappcodexstack

import (
	"encoding/json"
	"strings"
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
)

func TestEgressSanitizerConfigV0CentralizaRuntimePorProveedor(t *testing.T) {
	config := ConfigV0{
		Codex: CodexRuntimeConfigV0{
			CommandPath:     "/bin/codex",
			RuntimeWorkDir:  "runtime/codex",
			Model:           "model-codex",
			ReasoningEffort: "medium",
		},
		Gemini: GeminiRuntimeConfigV0{
			Enabled:        true,
			CommandPath:    "/bin/gemini",
			RuntimeWorkDir: "runtime/gemini",
			Model:          "model-gemini",
		},
		Claude: ClaudeRuntimeConfigV0{
			Enabled:        true,
			CommandPath:    "/bin/claude",
			RuntimeWorkDir: "runtime/claude",
			Model:          "model-claude",
		},
		EgressSanitizer: EgressSanitizerConfigV0{
			Enabled: true,
			Sidecar: PrivacyFilterSidecarConfigV0{
				Enabled:                 true,
				CommandConfigured:       true,
				LocalEndpointConfigured: true,
			},
		},
	}

	providers := CanonicalRuntimeProviderConfigsV0(config)
	if len(providers) != 4 {
		t.Fatalf("expected 4 canonical providers, got %+v", providers)
	}
	requireRuntimeProviderForTestV0(t, providers, RuntimeProviderCodexV0)
	requireRuntimeProviderForTestV0(t, providers, RuntimeProviderGeminiV0)
	requireRuntimeProviderForTestV0(t, providers, RuntimeProviderClaudeV0)
	requireRuntimeProviderForTestV0(t, providers, EgressSanitizerProviderOpenAIPrivacyLocalV0)
	raw, err := json.Marshal(providers)
	if err != nil {
		t.Fatalf("marshal providers: %v", err)
	}
	for _, forbidden := range []string{"/bin/codex", "runtime/codex", "/bin/gemini", "runtime/claude"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("canonical provider config leaked path %q: %s", forbidden, string(raw))
		}
	}
	if strings.Contains(string(raw), "model-ref-openai") {
		t.Fatalf("sidecar projection should not expose provider/model detail: %s", string(raw))
	}
}

func TestEgressSanitizerConfigV0NormalizaPrivacyFilterSidecarLocalOptIn(t *testing.T) {
	sidecar := fakePrivacyFilterSidecarForTestV0{}
	config := NormalizeEgressSanitizerConfigV0(EgressSanitizerConfigV0{
		Enabled: true,
		Sidecar: PrivacyFilterSidecarConfigV0{
			Enabled:                 true,
			CommandConfigured:       true,
			LocalEndpointConfigured: true,
			Port:                    sidecar,
		},
	})

	if config.ProviderRef != EgressSanitizerProviderOpenAIPrivacyLocalV0 {
		t.Fatalf("expected privacy filter provider, got %+v", config)
	}
	if config.Sidecar.SidecarRef == "" ||
		config.Sidecar.AdapterRef == "" ||
		config.Sidecar.TransportRef == "" ||
		config.Sidecar.EvidenceRef == "" {
		t.Fatalf("sidecar refs should be normalized: %+v", config.Sidecar)
	}
	if config.Sidecar.Port != sidecar {
		t.Fatalf("sidecar port should be preserved")
	}
	raw, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if strings.Contains(string(raw), "secret") || strings.Contains(strings.ToLower(string(raw)), "/home/") {
		t.Fatalf("normalized sidecar config leaked forbidden detail: %s", string(raw))
	}
}

func TestEgressSanitizerConfigV0InyectaPrivacyFilterSidecarLocalOptIn(t *testing.T) {
	config := configWithCanonicalEgressSanitizerV0(ConfigV0{
		EgressSanitizer: EgressSanitizerConfigV0{
			Enabled:      true,
			SanitizerRef: "sanitizer-ref-egress-test",
			Sidecar: PrivacyFilterSidecarConfigV0{
				Enabled:     true,
				SidecarRef:  "sidecar-ref-privacy-filter-test",
				EvidenceRef: "evidence-ref-privacy-filter-test",
				Port:        fakePrivacyFilterSidecarForTestV0{},
			},
		},
	})

	if config.Codex.ContextSanitizer == nil {
		t.Fatalf("context sanitizer should be injected from egress config")
	}
	result := config.Codex.ContextSanitizer.SanitizeContextEntryV0(orquestacontext.ContextSanitizationRequestV0{
		BundleRef:    "bundle-ref-egress-test",
		WorkOrderRef: "task-ref-egress-test",
		TargetModule: "orquesta-app-codex-stack",
		EntryRef:     "entry-ref-egress-test",
		SourceRef:    "source-ref-egress-test",
		Content:      "token=secret-value\npublic=true",
		Bytes:        30,
	})
	if result.Status != orquestacontext.ContextSanitizationStatusSanitizedV0 {
		t.Fatalf("expected sanitized result, got %+v", result)
	}
	if !stringInSetV0(result.Evidence.Categories, "openai_privacy_filter_local") ||
		!stringInSetV0(result.Evidence.Categories, "local_privacy_filter_sidecar") {
		t.Fatalf("privacy filter category missing: %+v", result.Evidence)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if strings.Contains(string(raw), "secret-value") || strings.Contains(strings.ToLower(string(raw)), "token=") {
		t.Fatalf("sanitization evidence leaked secret: %s", string(raw))
	}
}

func TestLocalSensitiveDataSanitizerV0NoBloqueaPorProviderModelRuntime(t *testing.T) {
	sanitizer := LocalSensitiveDataSanitizerV0{SanitizerRef: "sanitizer-ref-egress-test"}
	result := sanitizer.SanitizeContextEntryV0(orquestacontext.ContextSanitizationRequestV0{
		BundleRef:    "bundle-ref-egress-soft-rail",
		WorkOrderRef: "task-ref-egress-soft-rail",
		TargetModule: "orquesta-app-codex-stack",
		EntryRef:     "entry-ref-egress-soft-rail",
		SourceRef:    "source-ref-egress-soft-rail",
		Content:      "provider model runtime capacity son diagnostico publico",
		Bytes:        55,
	})

	if result.Status != orquestacontext.ContextSanitizationStatusCleanV0 ||
		result.Evidence.ReviewRequired {
		t.Fatalf("soft rail words should not block, got %+v", result)
	}
}

func TestLocalSensitiveDataSanitizerV0NoDeclaraSidecarUsadoSinPuerto(t *testing.T) {
	sanitizer := LocalSensitiveDataSanitizerV0{
		SanitizerRef: "sanitizer-ref-egress-test",
		Egress: EgressSanitizerConfigV0{
			Enabled: true,
			Sidecar: PrivacyFilterSidecarConfigV0{
				Enabled: true,
			},
		},
	}
	result := sanitizer.SanitizeContextEntryV0(orquestacontext.ContextSanitizationRequestV0{
		BundleRef:    "bundle-ref-egress-sidecar-configured",
		WorkOrderRef: "task-ref-egress-sidecar-configured",
		TargetModule: "orquesta-app-codex-stack",
		EntryRef:     "entry-ref-egress-sidecar-configured",
		SourceRef:    "source-ref-egress-sidecar-configured",
		Content:      "public=true",
		Bytes:        11,
	})

	if stringInSetV0(result.Evidence.Categories, "local_privacy_filter_sidecar") ||
		!stringInSetV0(result.Evidence.Categories, "local_privacy_filter_sidecar_configured") {
		t.Fatalf("sidecar categories=%+v", result.Evidence.Categories)
	}
}

func TestEgressSanitizerConfigV0RespetaSanitizerExplicito(t *testing.T) {
	explicit := fakeEgressSanitizerForTestV0{}
	config := configWithCanonicalEgressSanitizerV0(ConfigV0{
		Codex: CodexRuntimeConfigV0{ContextSanitizer: explicit},
		EgressSanitizer: EgressSanitizerConfigV0{
			Enabled: true,
		},
	})
	if config.Codex.ContextSanitizer != explicit {
		t.Fatalf("explicit sanitizer should win")
	}
}

type fakeEgressSanitizerForTestV0 struct{}

type fakePrivacyFilterSidecarForTestV0 struct{}

func (fakeEgressSanitizerForTestV0) SanitizeContextEntryV0(
	orquestacontext.ContextSanitizationRequestV0,
) orquestacontext.ContextSanitizationResultV0 {
	return orquestacontext.ContextSanitizationResultV0{}
}

func (fakePrivacyFilterSidecarForTestV0) FilterEgressPayloadV0(
	PrivacyFilterSidecarRequestV0,
) PrivacyFilterSidecarResultV0 {
	return PrivacyFilterSidecarResultV0{}
}

func requireRuntimeProviderForTestV0(
	t *testing.T,
	providers []RuntimeProviderConfigV0,
	kind RuntimeProviderKindV0,
) {
	t.Helper()
	for _, provider := range providers {
		if provider.Kind == kind {
			return
		}
	}
	t.Fatalf("provider %s missing in %+v", kind, providers)
}
