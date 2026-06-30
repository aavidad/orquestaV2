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
		!requirement.ProgressHeartbeatRequired ||
		!requirement.ProviderTimeoutRequired ||
		requirement.ProviderNoProgressTimeoutSeconds != 300 ||
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
				CapabilityRef:                    " cap-tts-001 ",
				Kind:                             " tts ",
				Available:                        true,
				NetworkReady:                     true,
				ToolPathReady:                    true,
				ProviderQuotaReady:               true,
				CommandTimeoutSeconds:            1800,
				ProgressHeartbeatReady:           true,
				ProviderTimeoutReady:             true,
				ProviderNoProgressTimeoutSeconds: 300,
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

func TestDomainWorkExternalCapabilityEvaluationV0BloqueaAudioSinHeartbeatProveedor(t *testing.T) {
	request := validDomainWorkJobRequestForTestV0()
	request.WorkKind = "generate_audio_asset"

	evaluation := EvaluateDomainWorkExternalCapabilitiesV0(
		request,
		[]DomainWorkExternalCapabilityV0{{
			CapabilityRef:         "tts-edge-ready-without-heartbeat",
			Kind:                  "tts",
			Available:             true,
			NetworkReady:          true,
			ToolPathReady:         true,
			ProviderQuotaReady:    true,
			CommandTimeoutSeconds: 1800,
		}},
	)

	if evaluation.Ready ||
		evaluation.OperationalReason != "external_capability_missing:speech_synthesis:progress_heartbeat_ready" ||
		len(evaluation.MissingRequirements) != 1 {
		t.Fatalf("evaluation=%+v", evaluation)
	}
}

func TestDomainWorkExternalCapabilityEvaluationV0BloqueaAudioSinProviderTimeout(t *testing.T) {
	request := validDomainWorkJobRequestForTestV0()
	request.WorkKind = "generate_audio_asset"

	evaluation := EvaluateDomainWorkExternalCapabilitiesV0(
		request,
		[]DomainWorkExternalCapabilityV0{{
			CapabilityRef:          "tts-edge-ready-without-timeout",
			Kind:                   "tts",
			Available:              true,
			NetworkReady:           true,
			ToolPathReady:          true,
			ProviderQuotaReady:     true,
			CommandTimeoutSeconds:  1800,
			ProgressHeartbeatReady: true,
		}},
	)

	if evaluation.Ready ||
		evaluation.OperationalReason != "external_capability_missing:speech_synthesis:provider_timeout_ready" ||
		len(evaluation.MissingRequirements) != 1 {
		t.Fatalf("evaluation=%+v", evaluation)
	}
}

func TestDomainWorkExternalCapabilityEvaluationV0BloqueaAudioConVentanaNoAvanceDemasiadoLarga(t *testing.T) {
	request := validDomainWorkJobRequestForTestV0()
	request.WorkKind = "generate_audio_asset"

	evaluation := EvaluateDomainWorkExternalCapabilitiesV0(
		request,
		[]DomainWorkExternalCapabilityV0{{
			CapabilityRef:                    "tts-edge-ready-slow-timeout",
			Kind:                             "tts",
			Available:                        true,
			NetworkReady:                     true,
			ToolPathReady:                    true,
			ProviderQuotaReady:               true,
			CommandTimeoutSeconds:            1800,
			ProgressHeartbeatReady:           true,
			ProviderTimeoutReady:             true,
			ProviderNoProgressTimeoutSeconds: 900,
		}},
	)

	if evaluation.Ready ||
		evaluation.OperationalReason != "external_capability_missing:speech_synthesis:provider_no_progress_timeout_seconds" ||
		len(evaluation.MissingRequirements) != 1 {
		t.Fatalf("evaluation=%+v", evaluation)
	}
}

func TestDomainWorkExternalCapabilityEvaluationV0NoAceptaRefConKindIncorrecto(t *testing.T) {
	request := validDomainWorkJobRequestForTestV0()
	request.WorkKind = "review_agent_independent"

	evaluation := EvaluateDomainWorkExternalCapabilitiesV0(
		request,
		[]DomainWorkExternalCapabilityV0{
			{
				CapabilityRef:         DomainWorkExternalCapabilityKindRemoteQAProviderV0,
				Kind:                  DomainWorkExternalCapabilityKindSpeechSynthesisV0,
				Available:             true,
				NetworkReady:          true,
				AuthStateReady:        true,
				ProviderQuotaReady:    true,
				CommandTimeoutSeconds: 1200,
				OperationalReason:     "wrong_kind_should_not_explain_remote_qa",
			},
		},
	)

	if evaluation.Ready ||
		evaluation.OperationalReason != "external_capability_missing:remote_qa_provider" ||
		len(evaluation.MissingRequirements) != 1 ||
		len(evaluation.MatchedCapabilities) != 0 {
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

func TestDomainWorkExternalCapabilityRequirementsV0MatrizDirectorNoRequiereProveedorRemoto(t *testing.T) {
	for _, workKind := range []string{
		"review_director_consolidation",
		"review_consensus_director",
	} {
		t.Run(workKind, func(t *testing.T) {
			request := validDomainWorkJobRequestForTestV0()
			request.WorkKind = workKind

			query := BuildDomainWorkExternalCapabilityQueryV0(request)
			evaluation := EvaluateDomainWorkExternalCapabilitiesV0(request, nil)

			if query.ArtifactType != DomainWorkArtifactTypeDirectorReviewMatrixV0 ||
				len(query.Requirements) != 0 {
				t.Fatalf("query=%+v", query)
			}
			if !evaluation.Ready ||
				len(evaluation.MissingRequirements) != 0 ||
				len(evaluation.Issues) != 0 ||
				evaluation.OperationalReason != "" {
				t.Fatalf("evaluation=%+v", evaluation)
			}
		})
	}
}

func TestDomainWorkExternalCapabilityRequirementsV0RevisionFinalDirectorNoRequiereProveedorRemoto(t *testing.T) {
	request := validDomainWorkJobRequestForTestV0()
	request.WorkKind = "review_director_final"

	query := BuildDomainWorkExternalCapabilityQueryV0(request)
	evaluation := EvaluateDomainWorkExternalCapabilitiesV0(request, nil)

	if len(query.Requirements) != 0 {
		t.Fatalf("query=%+v", query)
	}
	if !evaluation.Ready ||
		len(evaluation.MissingRequirements) != 0 ||
		len(evaluation.Issues) != 0 ||
		evaluation.OperationalReason != "" {
		t.Fatalf("evaluation=%+v", evaluation)
	}
}

func TestDomainWorkExternalCapabilityRequirementsV0RevisionesRemotasSiguenRequiriendoProveedor(t *testing.T) {
	for _, workKind := range []string{
		"review_codex",
		"review_gemini",
		"review_claude",
		"review_pair_codex_gemini",
		"review_pair_codex_claude",
		"review_pair_gemini_claude",
	} {
		t.Run(workKind, func(t *testing.T) {
			request := validDomainWorkJobRequestForTestV0()
			request.WorkKind = workKind

			query := BuildDomainWorkExternalCapabilityQueryV0(request)
			evaluation := EvaluateDomainWorkExternalCapabilitiesV0(request, nil)

			if len(query.Requirements) != 1 ||
				query.Requirements[0].Kind != DomainWorkExternalCapabilityKindRemoteQAProviderV0 {
				t.Fatalf("query=%+v", query)
			}
			if evaluation.Ready ||
				evaluation.OperationalReason != "external_capability_missing:remote_qa_provider" ||
				len(evaluation.MissingRequirements) != 1 {
				t.Fatalf("evaluation=%+v", evaluation)
			}
		})
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
			CapabilityRef:                    "capability-speech-synthesis",
			Kind:                             "speech_synthesis",
			Available:                        true,
			NetworkReady:                     true,
			ToolPathReady:                    true,
			ProviderQuotaReady:               true,
			CommandTimeoutSeconds:            1800,
			ProgressHeartbeatReady:           true,
			ProviderTimeoutReady:             true,
			ProviderNoProgressTimeoutSeconds: 300,
		},
	)}, nil
}
