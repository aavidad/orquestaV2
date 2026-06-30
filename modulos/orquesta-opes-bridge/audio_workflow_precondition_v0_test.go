package orquestaopesbridge

import (
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestEvaluateOPESAudioWorkflowPreconditionsV0BloqueaPrepareStalePorHashVigente(t *testing.T) {
	evaluation := EvaluateOPESAudioWorkflowPreconditionsV0(orquestadomainwork.DomainWorkJobRequestV0{
		WorkKind: "generate_audio_asset",
		InputFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "text_public_status", Value: "pass"},
			{Name: "audio_manifest_ref", Value: "audio-manifest-ref-001"},
			{Name: "source_content_ref", Value: "assembled-topic-ref-001"},
			{Name: "current_text_hash_ref", Value: "hash-ref-html-final-v2"},
			{Name: "prepared_text_hash_ref", Value: "hash-ref-html-final-v1"},
			{Name: "audio_regeneration_mode", Value: "selective_by_sidecar"},
		},
	})

	if evaluation.Ready ||
		evaluation.CurrentPhase != "prepare" ||
		evaluation.OperationalReason != OPESAudioWorkflowPrepareStaleRequiredReasonV0 ||
		evaluation.RecommendedRetryPhase != "prepare" ||
		!containsAudioWorkflowTestStringV0(evaluation.NextActions, "retry_from_phase=prepare") ||
		evaluation.AudioCounters["audio_manifest_refs"] != 1 ||
		evaluation.AudioCounters["source_content_refs"] != 1 ||
		evaluation.AudioCounters["text_hash_refs"] != 2 {
		t.Fatalf("evaluation=%+v", evaluation)
	}
}

func TestEvaluateOPESAudioWorkflowPreconditionsV0BloqueaPrepareStalePorStatus(t *testing.T) {
	evaluation := EvaluateOPESAudioWorkflowPreconditionsV0(orquestadomainwork.DomainWorkJobRequestV0{
		WorkKind: "generate_audio_asset",
		InputFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "text_public_status", Value: "pass"},
			{Name: "audio_manifest_ref", Value: "audio-manifest-ref-001"},
			{Name: "source_content_ref", Value: "assembled-topic-ref-001"},
			{Name: "audio_prepare_status", Value: "html_changed"},
			{Name: "audio_regeneration_mode", Value: "selective_by_sidecar"},
		},
	})

	if evaluation.Ready ||
		evaluation.OperationalReason != OPESAudioWorkflowPrepareStaleRequiredReasonV0 ||
		!containsAudioWorkflowTestStringV0(evaluation.NextActions, "invalidate_audio_sidecars") {
		t.Fatalf("evaluation=%+v", evaluation)
	}
}

func TestEvaluateOPESAudioWorkflowPreconditionsV0BloqueaMojibakeAntesDeTTS(t *testing.T) {
	evaluation := EvaluateOPESAudioWorkflowPreconditionsV0(orquestadomainwork.DomainWorkJobRequestV0{
		WorkKind: "generate_audio_asset",
		InputFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "text_public_status", Value: "pass"},
			{Name: "audio_manifest_ref", Value: "audio-manifest-ref-001"},
			{Name: "source_content_ref", Value: "assembled-topic-ref-001"},
			{Name: "audio_regeneration_mode", Value: "selective_by_sidecar"},
			{Name: "public_markdown", Value: "La AdministraciÃ³n local garantiza derechos."},
			{Name: "html_final", ValueJSON: []byte(`{"body":"<p>Texto Â corrupto</p>"}`)},
		},
	})

	if evaluation.Ready ||
		evaluation.CurrentPhase != "text_qa" ||
		evaluation.OperationalReason != OPESAudioWorkflowTextEncodingCorruptReasonV0 ||
		evaluation.RecommendedRetryPhase != "text_qa" ||
		!containsAudioWorkflowTestStringV0(evaluation.MissingPreconditions, "text_encoding_clean") ||
		!containsAudioWorkflowTestStringV0(evaluation.NextActions, "repair_public_text_encoding") ||
		!containsAudioWorkflowTestStringV0(evaluation.NextActions, "regenerate_audio_prepare_artifacts") {
		t.Fatalf("evaluation=%+v", evaluation)
	}
}

func containsAudioWorkflowTestStringV0(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
