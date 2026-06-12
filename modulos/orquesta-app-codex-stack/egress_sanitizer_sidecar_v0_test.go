package orquestaappcodexstack

import (
	"encoding/json"
	"strings"
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
)

func TestEgressSanitizerContextSanitizerV0UsesPrivacyFilterSidecarOptIn(t *testing.T) {
	sidecar := recordingPrivacyFilterSidecarForTestV0{
		result: PrivacyFilterSidecarResultV0{
			Content:          "public sanitized-ref-sidecar-001",
			Sanitized:        true,
			Categories:       []string{"credential_value"},
			ReplacementCount: 1,
		},
	}
	sanitizer := EgressSanitizerContextSanitizerV0{
		Config: NormalizeEgressSanitizerConfigV0(EgressSanitizerConfigV0{
			Enabled:      true,
			SanitizerRef: "sanitizer-ref-egress-sidecar-test",
			Sidecar: PrivacyFilterSidecarConfigV0{
				Enabled: true,
			},
		}),
		Sidecar: &sidecar,
	}

	result := sanitizer.SanitizeContextEntryV0(privacyFilterSanitizationRequestForTestV0(
		`token="sk-secret-sidecar" public=true`,
	))

	if sidecar.calls != 1 {
		t.Fatalf("sidecar calls=%d", sidecar.calls)
	}
	if strings.Contains(sidecar.lastRequest.Payload, "sk-secret-sidecar") ||
		strings.Contains(sidecar.lastRequest.Payload, `token="`) {
		t.Fatalf("sidecar received raw secret payload: %+v", sidecar.lastRequest)
	}
	if result.Status != orquestacontext.ContextSanitizationStatusSanitizedV0 ||
		result.Content != "public sanitized-ref-sidecar-001" {
		t.Fatalf("unexpected sanitizer result: %+v", result)
	}
	if !stringInSetV0(result.Evidence.Categories, "local_privacy_filter_sidecar") ||
		!stringInSetV0(result.Evidence.Categories, "openai_privacy_filter_local") ||
		!stringInSetV0(result.Evidence.Categories, "credential_value") {
		t.Fatalf("sidecar evidence categories missing: %+v", result.Evidence)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if strings.Contains(string(raw), "sk-secret-sidecar") {
		t.Fatalf("sanitizer result leaked secret: %s", string(raw))
	}
}

func TestEgressSanitizerContextSanitizerV0FallbackDeterministicWhenSidecarDisabled(t *testing.T) {
	sanitizer := NewEgressSanitizerContextSanitizerV0(EgressSanitizerConfigV0{
		Enabled:      true,
		SanitizerRef: "sanitizer-ref-egress-fallback-test",
		Sidecar: PrivacyFilterSidecarConfigV0{
			Enabled: false,
		},
	})

	result := sanitizer.SanitizeContextEntryV0(privacyFilterSanitizationRequestForTestV0(
		`api_key="alpha beta" public=true`,
	))

	if result.Status != orquestacontext.ContextSanitizationStatusSanitizedV0 ||
		result.Evidence.ReplacementCount == 0 {
		t.Fatalf("expected deterministic fallback sanitization, got %+v", result)
	}
	if stringInSetV0(result.Evidence.Categories, "local_privacy_filter_sidecar") {
		t.Fatalf("disabled sidecar should not be reported as used: %+v", result.Evidence)
	}
	for _, forbidden := range []string{"api_key", "alpha beta"} {
		if strings.Contains(strings.ToLower(result.Content), forbidden) {
			t.Fatalf("deterministic fallback leaked %q: %s", forbidden, result.Content)
		}
	}
}

func TestEgressSanitizerContextSanitizerV0SidecarReviewDoesNotPersistPayload(t *testing.T) {
	sidecar := recordingPrivacyFilterSidecarForTestV0{
		result: PrivacyFilterSidecarResultV0{
			ReviewRequired: true,
			Categories:     []string{"non_public_context"},
		},
	}
	sanitizer := EgressSanitizerContextSanitizerV0{
		Config: NormalizeEgressSanitizerConfigV0(EgressSanitizerConfigV0{
			Enabled: true,
			Sidecar: PrivacyFilterSidecarConfigV0{
				Enabled: true,
			},
		}),
		Sidecar: &sidecar,
	}

	result := sanitizer.SanitizeContextEntryV0(privacyFilterSanitizationRequestForTestV0(
		"private_material should not persist",
	))

	if result.Status != orquestacontext.ContextSanitizationStatusReviewRequiredV0 ||
		result.Content != "" ||
		!result.Evidence.ReviewRequired {
		t.Fatalf("expected review-required without payload, got %+v", result)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if strings.Contains(strings.ToLower(string(raw)), "private_material") {
		t.Fatalf("review evidence leaked payload: %s", string(raw))
	}
}

type recordingPrivacyFilterSidecarForTestV0 struct {
	calls       int
	lastRequest PrivacyFilterSidecarRequestV0
	result      PrivacyFilterSidecarResultV0
}

func (sidecar *recordingPrivacyFilterSidecarForTestV0) FilterEgressPayloadV0(
	request PrivacyFilterSidecarRequestV0,
) PrivacyFilterSidecarResultV0 {
	sidecar.calls++
	sidecar.lastRequest = request
	return sidecar.result
}

func privacyFilterSanitizationRequestForTestV0(content string) orquestacontext.ContextSanitizationRequestV0 {
	return orquestacontext.ContextSanitizationRequestV0{
		BundleRef:    "bundle-ref-privacy-filter-test",
		WorkOrderRef: "task-ref-privacy-filter-test",
		TargetModule: "orquesta-app-codex-stack",
		EntryRef:     "entry-ref-privacy-filter-test",
		SourceRef:    "source-ref-privacy-filter-test",
		Content:      content,
		Bytes:        len(content),
	}
}
