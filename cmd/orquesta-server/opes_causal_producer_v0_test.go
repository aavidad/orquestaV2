package main

import (
	"context"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestadomainworkfile "orquesta/modulos/orquesta-domain-work-file"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaopesdirector "orquesta/modulos/orquesta-opes-director"
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
	if len(first.CreatedJobs) != 1 {
		t.Fatalf("CreatedJobs first=%+v issues=%+v", first.CreatedJobs, first.Issues)
	}
	if first.CreatedJobs[0].WorkKind != "generate_audio_asset" {
		t.Fatalf("work_kind=%s", first.CreatedJobs[0].WorkKind)
	}

	second, err := producer.ProduceV0(ctx, orquestaopesdirector.OPESCausalProducerRequestV0{
		DomainRef:  orquestaopesdirector.OPESCausalProducerDefaultDomainRefV0,
		MaxActions: 5,
	})
	if err != nil {
		t.Fatalf("ProduceV0 second: %v", err)
	}
	if len(second.CreatedJobs) != 0 || len(second.SkippedRefs) != 1 {
		t.Fatalf("second CreatedJobs=%+v SkippedRefs=%+v", second.CreatedJobs, second.SkippedRefs)
	}
}
