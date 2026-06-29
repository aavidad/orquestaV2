package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadecisioncouncil "orquesta/modulos/orquesta-decision-council"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestCodexStackReceiptDecisionCouncilVoteSourceV0LeeArchitectureVoteDesdeAckV0(t *testing.T) {
	ctx := context.Background()
	projectDir := t.TempDir()
	spec := codexStackResidentCouncilVoteSpecForTestV0("task-council-v-001", "votes/architecture_vote.json")
	writeCodexStackResidentCouncilVoteFileForTestV0(t, projectDir, "votes/architecture_vote.json", map[string]any{
		"artifact_type": "architecture_vote.v0",
		"payload_json": map[string]any{
			"task_ref":      "task-council-v-001",
			"vote_ref":      "vote-ref-agent-001",
			"option_ref":    "task-council-p-001",
			"position":      "accepted",
			"evidence_refs": []string{"evidence-ref-vote-body"},
		},
	})
	descriptor := codexStackResidentCouncilVoteDescriptorForTestV0(t, projectDir, spec, []string{"votes/architecture_vote.json"})
	source := codexStackReceiptDecisionCouncilVoteSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(descriptor),
	}

	result, err := source.BuildDecisionCouncilVotesV0(ctx, codexStackResidentCouncilVoteRequestForTestV0())
	if err != nil {
		t.Fatalf("BuildDecisionCouncilVotesV0: %v", err)
	}
	if len(result.PendingEvidenceRefs) != 0 {
		t.Fatalf("pending inesperado: %+v", result.PendingEvidenceRefs)
	}
	if len(result.Votes) != 1 {
		t.Fatalf("votes=%+v", result.Votes)
	}
	vote := result.Votes[0]
	if vote.TaskRef != "task-council-v-001" ||
		vote.OptionRef != "task-council-p-001" ||
		vote.Position != orquestadecisioncouncil.CouncilVoteApproveV0 {
		t.Fatalf("vote normalizado inesperado: %+v", vote)
	}
	if !stringInSetV0(vote.EvidenceRefs, "evidence-ref-vote-body") {
		t.Fatalf("evidence_refs=%v", vote.EvidenceRefs)
	}
}

func TestCodexStackReceiptDecisionCouncilVoteSourceV0ConservaPendientePorPathInvalidoV0(t *testing.T) {
	ctx := context.Background()
	projectDir := t.TempDir()
	spec := codexStackResidentCouncilVoteSpecForTestV0("task-council-v-001", "../architecture_vote.json")
	descriptor := codexStackResidentCouncilVoteDescriptorForTestV0(t, projectDir, spec, []string{"../architecture_vote.json"})
	source := codexStackReceiptDecisionCouncilVoteSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(descriptor),
	}

	result, err := source.BuildDecisionCouncilVotesV0(ctx, codexStackResidentCouncilVoteRequestForTestV0())
	if err != nil {
		t.Fatalf("BuildDecisionCouncilVotesV0: %v", err)
	}
	if len(result.Votes) != 0 || len(result.PendingEvidenceRefs) == 0 {
		t.Fatalf("result=%+v", result)
	}
	if !stringInSetV0(
		result.PendingEvidenceRefs,
		"evidence-ref-codex-stack-resident-council-vote-ack-pending-task-council-v-001",
	) {
		t.Fatalf("pending=%v", result.PendingEvidenceRefs)
	}
}

func TestCodexStackResidentCouncilVoteFromJSONV0RechazaArtifactTypeAjenoV0(t *testing.T) {
	body := `{"artifact_type":"document_plan","task_ref":"task-council-v-001","option_ref":"task-council-p-001","position":"approve"}`
	if vote, ok := codexStackResidentCouncilVoteFromJSONV0(body, codexStackResidentCouncilVoteRequestForTestV0()); ok {
		t.Fatalf("voto aceptado con artifact_type ajeno: %+v", vote)
	}
}

func TestBuildStackV0CableaVoteSourceRealSiHayReceiptStoreV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	if stack.DecisionCouncil.VoteSource == nil {
		t.Fatalf("VoteSource no cableado")
	}
	if _, ok := stack.DecisionCouncil.VoteSource.(codexStackReceiptDecisionCouncilVoteSourceV0); !ok {
		t.Fatalf("VoteSource inesperado: %T", stack.DecisionCouncil.VoteSource)
	}
}

func codexStackResidentCouncilVoteRequestForTestV0() DecisionCouncilVoteBuildRequestV0 {
	return DecisionCouncilVoteBuildRequestV0{
		Run: orquestacoreworkflow.OrchestrationRunV0{
			RunID: "run-resident-council-vote-source-001",
		},
		VoteTasks: []orquestacoreworkflow.WorkflowTaskV0{{
			TaskID: "task-council-v-001",
		}},
		OptionRefs: []string{"task-council-p-001"},
	}
}

func codexStackResidentCouncilVoteSpecForTestV0(
	taskRef string,
	fileRef string,
) orquestaruntime.ExternalAgentLaunchSpecV0 {
	return orquestaruntime.ExternalAgentLaunchSpecV0{
		RequestID:     "agent-ref-" + taskRef,
		CorrelationID: "corr-" + taskRef,
		AgentPacket: orquestaruntime.AgentStartPacketV0{
			RequestID:     "agent-ref-" + taskRef,
			CorrelationID: "corr-" + taskRef,
			TargetModule:  "nueva-app",
			Phase:         string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
			Task: orquestaruntime.AgentStartTaskV0{
				TaskRef:  taskRef,
				WriteSet: []string{fileRef},
			},
			DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
				MailboxRef:   "mailbox-ref-" + taskRef,
				AckRef:       "ack-ref-" + taskRef,
				ReadinessRef: "readiness-ref-" + taskRef,
			},
		},
	}
}

func codexStackResidentCouncilVoteDescriptorForTestV0(
	t *testing.T,
	projectDir string,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
	files []string,
) orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	t.Helper()
	ackPath := filepath.Join(t.TempDir(), orquestaruntimecodex.CodexAgentAckFileNameV0)
	ack := orquestaruntimecodex.CodexAgentAckV0{
		SchemaVersion: orquestaruntimecodex.CodexAgentAckSchemaVersionV0,
		RequestID:     spec.RequestID,
		CorrelationID: spec.CorrelationID,
		AckRef:        spec.AgentPacket.DeliveryRefs.AckRef,
		TargetModule:  spec.AgentPacket.TargetModule,
		TaskRef:       spec.AgentPacket.Task.TaskRef,
		Status:        "completed",
		Files:         orquestaruntimecodex.EvidenceListV0(files),
	}
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("marshal ack: %v", err)
	}
	if err := os.WriteFile(ackPath, data, 0o600); err != nil {
		t.Fatalf("write ack: %v", err)
	}
	return orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		DescriptorRef:  "receipt-ref-" + spec.AgentPacket.DeliveryRefs.AckRef,
		RunID:          "run-resident-council-vote-source-001",
		AgentRef:       spec.RequestID,
		Spec:           spec,
		AckPath:        ackPath,
		ProjectWorkDir: projectDir,
	}
}

func writeCodexStackResidentCouncilVoteFileForTestV0(
	t *testing.T,
	projectDir string,
	fileRef string,
	body map[string]any,
) {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal vote: %v", err)
	}
	path := filepath.Join(projectDir, filepath.FromSlash(fileRef))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir vote dir: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write vote: %v", err)
	}
}
