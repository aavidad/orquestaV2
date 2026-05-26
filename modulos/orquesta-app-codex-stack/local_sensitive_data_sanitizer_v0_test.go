package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappchangedirectorsource "orquesta/modulos/orquesta-app-change-director-source"
	orquestacontext "orquesta/modulos/orquesta-context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestLocalSensitiveDataSanitizerV0SaneaContextoAntesDelPacket(t *testing.T) {
	runRef := "run-ref-local-sanitizer-001"
	changeRef := "opes-job-job-ref-local-sanitizer-001"
	taskRef := orquestaappchangedirectorsource.AppChangeTaskRefV0(changeRef)
	resolver := codexLaunchSpecResolverForLocalSanitizerTestV0(
		t,
		runRef,
		changeRef,
		taskRef,
		[]orquestadomainwork.DomainWorkFieldV0{{
			Name: "access_token",
			Value: `access_token="sk-secret-local" path=/home/alberto/app ` +
				`auth="Bearer live-token-001" prompt="prompt completo"`,
		}},
	)

	resolution, err := resolver.ResolveExternalAgentLaunchSpecV0(
		context.Background(),
		agentLaunchInboundForExternalContextPolicyTestV0(runRef, taskRef, "agent-ref-local-sanitizer-001"),
	)
	if err != nil {
		t.Fatalf("ResolveExternalAgentLaunchSpecV0: %v", err)
	}
	packet := resolution.Spec.AgentPacket
	if len(packet.Context.SanitizationEvidence) == 0 {
		t.Fatalf("sanitization evidence missing: %+v", packet.Context)
	}
	if !stringInSetV0(packet.Policies, "context_sanitization_evidence_present") {
		t.Fatalf("sanitization policy missing: %+v", packet.Policies)
	}
	raw, err := json.Marshal(resolution.Spec)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	assertNoLocalSensitiveDataForTestV0(t, string(raw))
}

func TestLocalSensitiveDataSanitizerV0DudaYActivaRevisionDirector(t *testing.T) {
	runRef := "run-ref-local-sanitizer-review-001"
	changeRef := "opes-job-job-ref-local-sanitizer-review-001"
	taskRef := orquestaappchangedirectorsource.AppChangeTaskRefV0(changeRef)
	resolver := codexLaunchSpecResolverForLocalSanitizerTestV0(
		t,
		runRef,
		changeRef,
		taskRef,
		[]orquestadomainwork.DomainWorkFieldV0{{
			Name:  "private_material",
			Value: "-----BEGIN PRIVATE KEY----- abc",
		}},
	)

	resolution, err := resolver.ResolveExternalAgentLaunchSpecV0(
		context.Background(),
		agentLaunchInboundForExternalContextPolicyTestV0(
			runRef,
			taskRef,
			"agent-ref-local-sanitizer-review-001",
		),
	)
	if err != nil {
		t.Fatalf("ResolveExternalAgentLaunchSpecV0: %v", err)
	}
	packet := resolution.Spec.AgentPacket
	if !stringInSetV0(packet.Task.RequiredTests, externalContextRefOnlyRequiredTestV0) ||
		!stringInSetV0(packet.Task.DoneCriteria, externalContextRefOnlyDoneCriteriaV0) {
		t.Fatalf("sanitization guard missing: %+v", packet.Task)
	}
	if !stringInSetV0(packet.Policies, "required_ref_only_context_guard") {
		t.Fatalf("ref-only context policy missing: %+v", packet.Policies)
	}
	if !contextBundleHasRequiredTruncatedEntryV0(packet.Context) {
		t.Fatalf("context should be minimized by ref: %+v", packet.Context.Entries)
	}
	raw, err := json.Marshal(packet)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(strings.ToLower(string(raw)), "private key") {
		t.Fatalf("private material leaked: %s", string(raw))
	}
}

func TestLocalSensitiveDataSanitizerV0TranscriptCompletoRequiereRevision(t *testing.T) {
	sanitizer := LocalSensitiveDataSanitizerV0{SanitizerRef: "sanitizer-ref-local-test"}
	result := sanitizer.SanitizeContextEntryV0(orquestacontext.ContextSanitizationRequestV0{
		BundleRef:    "bundle-ref-local-sanitizer-review-001",
		WorkOrderRef: "task-ref-local-sanitizer-review-001",
		TargetModule: "orquesta-app-codex-stack",
		EntryRef:     "entry-ref-local-sanitizer-review-001",
		SourceRef:    "source-ref-local-sanitizer-review-001",
		Content:      `{"name":"transcript","value":"transcript completo token=sk-secret-local"}`,
		Bytes:        72,
	})

	if result.Status != orquestacontext.ContextSanitizationStatusReviewRequiredV0 ||
		result.Content != "" ||
		!result.Evidence.ReviewRequired {
		t.Fatalf("expected review-required sanitizer result, got %+v", result)
	}
	assertNoLocalSensitiveDataForTestV0(t, result.Content)
}

func TestLocalSensitiveDataSanitizerV0SaneaValoresSensiblesConEspacios(t *testing.T) {
	sanitizer := LocalSensitiveDataSanitizerV0{SanitizerRef: "sanitizer-ref-local-test"}
	result := sanitizer.SanitizeContextEntryV0(orquestacontext.ContextSanitizationRequestV0{
		BundleRef:    "bundle-ref-local-sanitizer-spaces-001",
		WorkOrderRef: "task-ref-local-sanitizer-spaces-001",
		TargetModule: "orquesta-app-codex-stack",
		EntryRef:     "entry-ref-local-sanitizer-spaces-001",
		SourceRef:    "source-ref-local-sanitizer-spaces-001",
		Content:      `{"api_key":"alpha beta,gamma","next":"public"}`,
		Bytes:        48,
	})

	if result.Status != orquestacontext.ContextSanitizationStatusSanitizedV0 ||
		result.Evidence.ReplacementCount == 0 ||
		result.Evidence.ReviewRequired {
		t.Fatalf("expected sanitized result, got %+v", result)
	}
	for _, forbidden := range []string{"api_key", "alpha beta", "gamma"} {
		if strings.Contains(strings.ToLower(result.Content), forbidden) {
			t.Fatalf("sensitive fragment %q leaked: %s", forbidden, result.Content)
		}
	}
	if !strings.Contains(result.Content, `"next":"public"`) {
		t.Fatalf("public sibling field lost: %s", result.Content)
	}
}

func TestLocalSensitiveDataSanitizerV0SaneaAsignacionBareHastaSeparadorSeguro(t *testing.T) {
	sanitizer := LocalSensitiveDataSanitizerV0{SanitizerRef: "sanitizer-ref-local-test"}
	result := sanitizer.SanitizeContextEntryV0(orquestacontext.ContextSanitizationRequestV0{
		BundleRef:    "bundle-ref-local-sanitizer-bare-001",
		WorkOrderRef: "task-ref-local-sanitizer-bare-001",
		TargetModule: "orquesta-app-codex-stack",
		EntryRef:     "entry-ref-local-sanitizer-bare-001",
		SourceRef:    "source-ref-local-sanitizer-bare-001",
		Content:      "token=alpha beta gamma,\npublic=true",
		Bytes:        35,
	})

	if result.Status != orquestacontext.ContextSanitizationStatusSanitizedV0 {
		t.Fatalf("expected sanitized result, got %+v", result)
	}
	for _, forbidden := range []string{"token=", "alpha beta", "gamma"} {
		if strings.Contains(strings.ToLower(result.Content), forbidden) {
			t.Fatalf("sensitive fragment %q leaked: %s", forbidden, result.Content)
		}
	}
	if !strings.Contains(result.Content, "public=true") {
		t.Fatalf("public trailing field lost: %s", result.Content)
	}
}

func TestLocalSensitiveDataSanitizerV0NoFiltraRefsSensiblesEnEvidencia(t *testing.T) {
	sanitizer := LocalSensitiveDataSanitizerV0{SanitizerRef: "sanitizer-ref-local-test"}
	result := sanitizer.SanitizeContextEntryV0(orquestacontext.ContextSanitizationRequestV0{
		BundleRef:    "bundle-ref-local-sanitizer-ref-001",
		WorkOrderRef: "task-ref-local-sanitizer-ref-001",
		TargetModule: "orquesta-app-codex-stack",
		EntryRef:     "entry-ref-access_token=sk-secret-local",
		SourceRef:    "/home/alberto/private/token=sk-secret-local",
		Content:      `{"name":"public","value":"ok"}`,
		Bytes:        30,
	})

	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, forbidden := range []string{"/home/alberto", "access_token", "sk-secret-local", "token="} {
		if strings.Contains(strings.ToLower(string(raw)), forbidden) {
			t.Fatalf("sensitive ref fragment %q leaked: %s", forbidden, string(raw))
		}
	}
}

func codexLaunchSpecResolverForLocalSanitizerTestV0(
	t *testing.T,
	runRef string,
	changeRef string,
	taskRef string,
	fields []orquestadomainwork.DomainWorkFieldV0,
) CodexLaunchSpecResolverV0 {
	t.Helper()
	task := orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion: orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:        taskRef,
		RunID:         runRef,
		PhaseID:       orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:         "Ejecutar trabajo externo con contexto saneado",
		Summary:       "Usar contexto externo sin datos sensibles locales.",
		WriteSet:      []string{"external/opes/draft_content_block"},
		AcceptanceCriteria: []string{
			"conservar evidencia de saneamiento",
		},
	}
	return CodexLaunchSpecResolverV0{
		Config: CodexRuntimeConfigV0{
			RuntimeWorkDir:   t.TempDir(),
			ProjectWorkDir:   t.TempDir(),
			ContextSanitizer: LocalSensitiveDataSanitizerV0{SanitizerRef: "sanitizer-ref-local-test"},
		},
		TaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
		AppChangeStore: orquestaappchange.NewInMemoryAppChangeStoreV0(orquestaappchange.AppChangeRecordV0{
			Request: orquestaappchange.AppChangeRequestV0{
				RunRef:    runRef,
				AppRef:    "opes",
				ChangeRef: changeRef,
				ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
					ProjectRef:  "opes",
					JobRef:      "job-ref-local-sanitizer-001",
					WorkKind:    "draft_content_block",
					InputFields: fields,
				},
			},
		}),
	}
}

func assertNoLocalSensitiveDataForTestV0(t *testing.T, value string) {
	t.Helper()
	lower := strings.ToLower(value)
	for _, forbidden := range []string{
		"access_token", "sk-secret-local", "/home/alberto", "bearer ", "prompt=",
	} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("sensitive data %q leaked: %s", forbidden, value)
		}
	}
}
