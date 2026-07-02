package orquestaopesbridge

import (
	"context"
	"strings"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestOPESRequiredTestPolicyV0AudioExigeTTSReanudableSinDuplicarMP3Validos(t *testing.T) {
	plan, err := (OPESRequiredTestPolicyV0{}).BuildDomainWorkRequiredTestPlanV0(
		context.Background(),
		orquestadomainwork.DomainWorkJobRequestV0{
			CorrelationID:  "corr-policy-opes-audio",
			IdempotencyKey: "idem-policy-opes-audio",
			RequestedBy:    "orquesta",
			DomainRef:      "opes",
			WorkKind:       "generate_audio_asset",
			WorkRefs:       []string{"job-ref-policy-opes-audio"},
			Objective:      "generar audio OPES reanudable",
		},
	)
	if err != nil {
		t.Fatalf("BuildDomainWorkRequiredTestPlanV0: %v", err)
	}
	audioTTS := requiredTestByRefForTestV0(
		plan.RequiredTests,
		"opes-audio-tts-resumable-job-ref-policy-opes-audio",
	)
	if audioTTS.TestRef == "" ||
		!requiredTestHasExternalRefForTestV0(audioTTS, "required_test_name", "audio_tts_resumable") ||
		!requiredTestHasExternalRefForTestV0(audioTTS, "required_evidence", "audio_tts_operational_manifest") ||
		!requiredTestHasExternalRefForTestV0(audioTTS, "required_evidence", "progress_heartbeat") ||
		!requiredTestHasExternalRefForTestV0(audioTTS, "required_evidence", "provider_timeout") ||
		!requiredTestHasExternalRefForTestV0(audioTTS, "required_evidence", "resume_without_duplicate_valid_mp3") ||
		!stringInRequiredTestRefsForTestV0(audioTTS.AcceptanceCriteriaRefs, "opes-required-audio-tts-resumable") ||
		!stringInRequiredTestRefsForTestV0(audioTTS.EvidenceRefs, "opes-final-evidence:audio_tts_resumable") {
		t.Fatalf("audio_tts_required_test=%+v", audioTTS)
	}
	criteria := strings.Join(audioTTS.AcceptanceCriteria, "\n")
	for _, want := range []string{
		"heartbeat",
		"provider_timeout",
		"reanudacion segura",
		"MP3 validos preservados",
		"skipped_valid_mp3_refs",
		"generated_mp3_refs",
		"No se duplican MP3 validos",
		"unico audio_ref final vigente",
		"retry_from_phase=tts",
		"no ready",
	} {
		if !strings.Contains(criteria, want) {
			t.Fatalf("audio_tts_criteria=%q falta %s", criteria, want)
		}
	}
}

func TestOPESRequiredTestPolicyV0FinalTemarioIncluyeAudioTTSReanudable(t *testing.T) {
	plan, err := (OPESRequiredTestPolicyV0{}).BuildDomainWorkRequiredTestPlanV0(
		context.Background(),
		orquestadomainwork.DomainWorkJobRequestV0{
			CorrelationID:  "corr-policy-opes-final-audio",
			IdempotencyKey: "idem-policy-opes-final-audio",
			RequestedBy:    "orquesta",
			DomainRef:      "opes",
			WorkKind:       "finalize_temario_package",
			WorkRefs:       []string{"job-ref-policy-opes-final-audio"},
			Objective:      "cerrar temario OPES con audio validado",
		},
	)
	if err != nil {
		t.Fatalf("BuildDomainWorkRequiredTestPlanV0: %v", err)
	}
	audioTTS := requiredTestByRefForTestV0(
		plan.RequiredTests,
		"opes-audio-tts-resumable-job-ref-policy-opes-final-audio",
	)
	if audioTTS.TestRef == "" ||
		!requiredTestHasExternalRefForTestV0(audioTTS, "required_evidence", "resume_without_duplicate_valid_mp3") ||
		!stringInRequiredTestRefsForTestV0(audioTTS.EvidenceRefs, "opes-expected-evidence-resume-without-duplicate-valid-mp3") {
		t.Fatalf("audio_tts_required_test=%+v", audioTTS)
	}
}
