package main

import (
	"context"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestadomainworkfile "orquesta/modulos/orquesta-domain-work-file"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaopesdirector "orquesta/modulos/orquesta-opes-director"
	orquestaopestopicregistry "orquesta/modulos/orquesta-opes-topic-registry"
)

func TestServerOPESCausalProducerV0CreaFollowupDesdeLedgerYNoDuplica(t *testing.T) {
	ctx := context.Background()
	ledger := orquestaappcodexstack.NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
	if err := ledger.RecordDomainWorkArtifactSubmissionV0(ctx, orquestaappcodexstack.DomainWorkArtifactSubmissionRecordV0{
		IdempotencyKey: "submission-final-package-pending-001",
		Status:         orquestaappcodexstack.DomainWorkArtifactSubmissionStatusAcceptedV0,
		DomainRef:      orquestaopesdirector.OPESCausalProducerDefaultDomainRefV0,
		JobRef:         "job-source-final-package-001",
		ArtifactRef:    "artifact-final-package-001",
		ArtifactType:   orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0,
		CorrelationID:  "corr-final-package-pending-001",
		ReceiptRef:     "receipt-final-package-pending-001",
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "estado", Value: "pendiente_continuar"},
			{Name: "target_artifact_type", Value: orquestadomainwork.DomainWorkArtifactTypeAudioAssetV0},
			{Name: "followup_refs", Values: []string{"audio-tema-01"}},
			{Name: "course_id", Value: "curso-a2-informatica"},
			{Name: "topic_id", Value: "tema-01"},
		},
		EvidenceRefs: []string{"evidence-ref-final-package-pending-001"},
	}); err != nil {
		t.Fatalf("RecordDomainWorkArtifactSubmissionV0: %v", err)
	}

	creator, err := orquestadomainworkfile.NewFileDomainWorkJobCreatorV0(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileDomainWorkJobCreatorV0: %v", err)
	}
	stack := &orquestaappcodexstack.StackV0{
		DomainWork: orquestamcp.NewMCPDomainWorkToolExecutorV0(creator, creator),
		DomainDelivery: orquestaappcodexstack.DomainWorkDeliveryBridgeConfigV0{
			Enabled: true,
			Ledger:  ledger,
		},
	}
	producer := newServerOPESCausalProducerV0(stack)
	if producer == nil {
		t.Fatalf("productor causal OPES no cableado")
	}

	first, err := producer.ProduceV0(ctx, orquestaopesdirector.OPESCausalProducerRequestV0{
		DomainRef:  orquestaopesdirector.OPESCausalProducerDefaultDomainRefV0,
		MaxActions: 5,
	})
	if err != nil {
		t.Fatalf("ProduceV0 first: %v", err)
	}
	if len(first.CreatedJobs) != 2 {
		t.Fatalf("CreatedJobs first=%+v issues=%+v", first.CreatedJobs, first.Issues)
	}
	if !serverOPESCausalProducerCreatedWorkKindForTestV0(first.CreatedJobs, "generate_audio_asset") ||
		!serverOPESCausalProducerCreatedWorkKindForTestV0(first.CreatedJobs, "update_topic_registry") {
		t.Fatalf("CreatedJobs first=%+v", first.CreatedJobs)
	}

	second, err := producer.ProduceV0(ctx, orquestaopesdirector.OPESCausalProducerRequestV0{
		DomainRef:  orquestaopesdirector.OPESCausalProducerDefaultDomainRefV0,
		MaxActions: 5,
	})
	if err != nil {
		t.Fatalf("ProduceV0 second: %v", err)
	}
	if len(second.CreatedJobs) != 0 || len(second.SkippedRefs) != 2 {
		t.Fatalf("second CreatedJobs=%+v SkippedRefs=%+v", second.CreatedJobs, second.SkippedRefs)
	}
}

func serverOPESCausalProducerCreatedWorkKindForTestV0(
	jobs []orquestadomainwork.DomainWorkJobV0,
	workKind string,
) bool {
	for _, job := range jobs {
		if job.WorkKind == workKind {
			return true
		}
	}
	return false
}

func TestServerOPESTopicRegistryUpdaterV0AplicaDesdeDomainWork(t *testing.T) {
	runner := &fakeServerTopicRegistryRunnerV0{}
	updater := serverOPESTopicRegistryUpdaterV0{
		config: opesTopicRegistryConfigV0{
			Enabled:  true,
			ToolPath: "/tmp/registro_trabajo_temas.py",
			AgentID:  "orquesta-registro",
			Force:    true,
		},
		runner: runner,
	}
	result, err := updater.ApplyOPESCausalTopicRegistryUpdateV0(context.Background(),
		orquestadomainwork.DomainWorkJobRequestV0{
			WorkKind: "update_topic_registry",
			InputFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "registry_action", Value: "release"},
				{Name: "course_id", Value: "curso-a2"},
				{Name: "topic_id", Value: "tema-001"},
				{Name: "proposed_status", Value: "paquete_final_local_verificable"},
				{Name: "done_refs", Values: []string{"artifact-001"}},
			},
			EvidenceRefs: []string{"evidence-job-001"},
		})
	if err != nil {
		t.Fatalf("ApplyOPESCausalTopicRegistryUpdateV0: %v", err)
	}
	if result.Status != orquestaopestopicregistry.TopicRegistryUpdateStatusAppliedV0 ||
		len(runner.invocations) != 1 ||
		runner.invocations[0].ToolPath != "/tmp/registro_trabajo_temas.py" ||
		runner.invocations[0].Args[0] != "release" ||
		!serverTopicRegistryArgsContainForTestV0(runner.invocations[0].Args, "--force") {
		t.Fatalf("result=%+v invocations=%+v", result, runner.invocations)
	}
}

type fakeServerTopicRegistryRunnerV0 struct {
	invocations []orquestaopestopicregistry.TopicRegistryCommandInvocationV0
}

func (runner *fakeServerTopicRegistryRunnerV0) RunTopicRegistryCommandV0(
	_ context.Context,
	invocation orquestaopestopicregistry.TopicRegistryCommandInvocationV0,
) (orquestaopestopicregistry.TopicRegistryCommandResultV0, error) {
	runner.invocations = append(runner.invocations, invocation)
	return orquestaopestopicregistry.TopicRegistryCommandResultV0{}, nil
}

func serverTopicRegistryArgsContainForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
