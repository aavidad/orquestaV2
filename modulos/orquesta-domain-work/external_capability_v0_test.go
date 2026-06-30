package orquestadomainwork

import (
	"context"
	"testing"
)

func TestDomainWorkExternalCapabilityRequirementsV0AudioRequiereSpeechSynthesis(t *testing.T) {
	request := validDomainWorkJobRequestForTestV0()
	request.WorkKind = " tts_topic "
	request.ExternalRefs = []DomainWorkExternalRefV0{
		{Kind: " run_ref ", Ref: " run-ref-tts-001 "},
		{Kind: "run_ref", Ref: "run-ref-tts-001"},
	}
	request.EvidenceRefs = []string{" evidence-ref-tts-001 ", "evidence-ref-tts-001"}

	query := BuildDomainWorkExternalCapabilityQueryV0(request)

	if query.SchemaVersion != DomainWorkExternalCapabilityQuerySchemaV0 ||
		query.WorkKind != "tts_topic" ||
		query.ArtifactType != DomainWorkArtifactTypeAudioAssetV0 ||
		len(query.Requirements) != 1 {
		t.Fatalf("query=%+v", query)
	}
	requirement := query.Requirements[0]
	if requirement.Kind != DomainWorkExternalCapabilityKindSpeechSynthesisV0 ||
		requirement.CapabilityRef != DomainWorkExternalCapabilityKindSpeechSynthesisV0 ||
		requirement.Reason != DomainWorkExternalCapabilityReasonArtifactRequiresCapabilityV0 ||
		!requirement.Required ||
		!requirement.NetworkRequired ||
		!requirement.ToolPathRequired ||
		!requirement.ProviderQuotaSensitive ||
		requirement.CommandTimeoutSeconds != 1800 ||
		requirement.WorkKind != "tts_topic" ||
		requirement.ArtifactType != DomainWorkArtifactTypeAudioAssetV0 ||
		len(requirement.ExternalRefs) != 1 ||
		len(requirement.EvidenceRefs) != 1 {
		t.Fatalf("requirement=%+v", requirement)
	}
}

func TestDomainWorkExternalCapabilityEvaluationV0BloqueaAudioSinTTS(t *testing.T) {
	request := validDomainWorkJobRequestForTestV0()
	request.WorkKind = "generate_audio_asset"

	evaluation := EvaluateDomainWorkExternalCapabilitiesV0(request, nil)

	if evaluation.Ready ||
		evaluation.OperationalReason != "external_capability_missing:speech_synthesis" ||
		len(evaluation.MissingRequirements) != 1 ||
		len(evaluation.Issues) != 1 ||
		evaluation.Issues[0].Code != ErrDomainWorkExternalCapabilityMissingV0 ||
		evaluation.Issues[0].Field != "external_capabilities" {
		t.Fatalf("evaluation=%+v", evaluation)
	}
}

func TestDomainWorkExternalCapabilityEvaluationV0AceptaDeclaracionTTSAlias(t *testing.T) {
	request := validDomainWorkJobRequestForTestV0()
	request.WorkKind = "generate_topic_audio"

	evaluation := EvaluateDomainWorkExternalCapabilitiesV0(
		request,
		[]DomainWorkExternalCapabilityV0{
			{
				CapabilityRef:         " cap-tts-001 ",
				Kind:                  " tts ",
				Available:             true,
				NetworkReady:          true,
				ToolPathReady:         true,
				ProviderQuotaReady:    true,
				CommandTimeoutSeconds: 1800,
				ExternalRefs: []DomainWorkExternalRefV0{{
					Kind: "edge_host_ref",
					Ref:  "edge-host-ref-001",
				}},
				EvidenceRefs: []string{" evidence-tts-ready-001 "},
			},
		},
	)

	if !evaluation.Ready ||
		len(evaluation.Issues) != 0 ||
		len(evaluation.MissingRequirements) != 0 ||
		len(evaluation.MatchedCapabilities) != 1 ||
		evaluation.MatchedCapabilities[0].Kind != DomainWorkExternalCapabilityKindSpeechSynthesisV0 ||
		evaluation.MatchedCapabilities[0].CapabilityRef != "cap-tts-001" ||
		len(evaluation.MatchedCapabilities[0].EvidenceRefs) != 1 {
		t.Fatalf("evaluation=%+v", evaluation)
	}
}

func TestDomainWorkExternalCapabilityEvaluationV0ConservaRazonOperativaDeTTSNoDisponible(t *testing.T) {
	request := validDomainWorkJobRequestForTestV0()
	request.WorkKind = "synthesize_topic_audio"

	evaluation := EvaluateDomainWorkExternalCapabilitiesV0(
		request,
		[]DomainWorkExternalCapabilityV0{
			{
				CapabilityRef:     "tts-edge-host",
				Kind:              "text_to_speech",
				Available:         false,
				OperationalReason: "tts-edge-host-not-configured",
			},
		},
	)

	if evaluation.Ready ||
		evaluation.OperationalReason != "tts-edge-host-not-configured" ||
		len(evaluation.MissingRequirements) != 1 {
		t.Fatalf("evaluation=%+v", evaluation)
	}
}

func TestDomainWorkExternalCapabilityRequirementsV0RevisionRemotaRequierePerfilProveedor(t *testing.T) {
	request := validDomainWorkJobRequestForTestV0()
	request.WorkKind = "review_agent_independent"

	query := BuildDomainWorkExternalCapabilityQueryV0(request)

	if query.ArtifactType != DomainWorkArtifactTypeAgentReviewReportV0 ||
		len(query.Requirements) != 1 {
		t.Fatalf("query=%+v", query)
	}
	requirement := query.Requirements[0]
	if requirement.Kind != DomainWorkExternalCapabilityKindRemoteQAProviderV0 ||
		requirement.Reason != DomainWorkExternalCapabilityReasonWorkKindRequiresCapabilityV0 ||
		!requirement.Required ||
		!requirement.NetworkRequired ||
		!requirement.AuthStateRequired ||
		!requirement.ProviderQuotaSensitive ||
		requirement.ToolPathRequired ||
		requirement.CommandTimeoutSeconds != 1200 {
		t.Fatalf("requirement=%+v", requirement)
	}
}

func TestDomainWorkExternalCapabilityEvaluationV0BloqueaQASiAuthOCuotaNoEstanListas(t *testing.T) {
	request := validDomainWorkJobRequestForTestV0()
	request.WorkKind = "review_agent_independent"

	evaluation := EvaluateDomainWorkExternalCapabilitiesV0(
		request,
		[]DomainWorkExternalCapabilityV0{{
			CapabilityRef:         "qa-provider-ref-001",
			Kind:                  "qa_provider",
			Available:             true,
			NetworkReady:          true,
			AuthStateReady:        false,
			ProviderQuotaReady:    true,
			CommandTimeoutSeconds: 1200,
			OperationalReason:     "auth_state_expired",
		}},
	)

	if evaluation.Ready ||
		len(evaluation.MissingRequirements) != 1 ||
		evaluation.OperationalReason != "auth_state_expired" {
		t.Fatalf("evaluation=%+v", evaluation)
	}
}

func TestDomainWorkExternalCapabilitySourcePortV0EsHexagonal(t *testing.T) {
	var source DomainWorkExternalCapabilitySourcePortV0 = fakeDomainWorkExternalCapabilitySourceV0{}

	capabilities, err := source.ListDomainWorkExternalCapabilitiesV0(
		context.Background(),
		BuildDomainWorkExternalCapabilityQueryV0(validDomainWorkJobRequestForTestV0()),
	)

	if err != nil {
		t.Fatalf("ListDomainWorkExternalCapabilitiesV0: %v", err)
	}
	if len(capabilities) != 1 ||
		capabilities[0].Kind != DomainWorkExternalCapabilityKindSpeechSynthesisV0 ||
		!capabilities[0].Available {
		t.Fatalf("capabilities=%+v", capabilities)
	}
}

type fakeDomainWorkExternalCapabilitySourceV0 struct{}

func (fakeDomainWorkExternalCapabilitySourceV0) ListDomainWorkExternalCapabilitiesV0(
	context.Context,
	DomainWorkExternalCapabilityQueryV0,
) ([]DomainWorkExternalCapabilityV0, error) {
	return []DomainWorkExternalCapabilityV0{NormalizeDomainWorkExternalCapabilityV0(
		DomainWorkExternalCapabilityV0{
			CapabilityRef:         "capability-speech-synthesis",
			Kind:                  "speech_synthesis",
			Available:             true,
			NetworkReady:          true,
			ToolPathReady:         true,
			ProviderQuotaReady:    true,
			CommandTimeoutSeconds: 1800,
		},
	)}, nil
}
