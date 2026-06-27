package orquestaruntimecodexdelivery

import (
	"os"
	"path/filepath"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func artifactPathInvalidIssueV0() orquestaruntime.ExternalAgentConnectorErrorV0 {
	return orquestaruntime.ExternalAgentConnectorErrorV0{
		Code:     orquestaruntime.ExternalAgentConnectorErrorCodeV0(orquestaruntimecodex.CodexConnectorAckArtifactV0),
		Field:    "files",
		Evidence: []string{"artifact_path_invalid"},
	}
}

func TestCodexReceiptAckIssuesAllRecoverableV0ReceiptFaltanteEsRecuperable(t *testing.T) {
	issues := []orquestaruntime.ExternalAgentConnectorErrorV0{{
		Code:     orquestaruntime.ExternalAgentConnectorErrorCodeV0(orquestaruntimecodex.CodexConnectorAckArtifactV0),
		Field:    "test_receipts",
		Evidence: []string{"missing_required_test_receipt"},
	}}
	if !codexReceiptAckIssuesAllRecoverableV0(issues) {
		t.Fatalf("missing_required_test_receipt debe llegar a review/rework, no tumbar el tick")
	}
}

func TestCodexReceiptAckIssuesAllRecoverableV0DetalleSensibleRedactableEsRecuperable(t *testing.T) {
	issues := []orquestaruntime.ExternalAgentConnectorErrorV0{{
		Code:     orquestaruntime.ExternalAgentConnectorErrorCodeV0(orquestaruntimecodex.CodexConnectorAckForbiddenV0),
		Field:    "agent_ack",
		Evidence: []string{"forbidden_sensitive_detail"},
	}}
	if !codexReceiptAckIssuesAllRecoverableV0(issues) {
		t.Fatalf("forbidden_sensitive_detail debe conservarse como gate-issue redactado")
	}
}

func TestCodexReceiptAckIssuesAllRecoverableV0SalidaCrudaEsDura(t *testing.T) {
	issues := []orquestaruntime.ExternalAgentConnectorErrorV0{{
		Code:     orquestaruntime.ExternalAgentConnectorErrorCodeV0(orquestaruntimecodex.CodexConnectorAckForbiddenV0),
		Field:    "test_receipts",
		Evidence: []string{"raw_test_output_forbidden"},
	}}
	if codexReceiptAckIssuesAllRecoverableV0(issues) {
		t.Fatalf("raw_test_output_forbidden no debe suavizarse")
	}
}

func TestCodexReceiptAckIssuesAllRecoverableV0RutaNormalizableEsRecuperable(t *testing.T) {
	issues := []orquestaruntime.ExternalAgentConnectorErrorV0{artifactPathInvalidIssueV0()}
	if !codexReceiptAckIssuesAllRecoverableV0(issues) {
		t.Fatalf("artifact_path_invalid debe ser recuperable")
	}
}

func TestCodexReceiptAckIssuesAllRecoverableV0CorrelacionEsDura(t *testing.T) {
	issues := []orquestaruntime.ExternalAgentConnectorErrorV0{{
		Code:  orquestaruntime.ExternalAgentConnectorErrorCodeV0(orquestaruntimecodex.CodexConnectorAckCorrelationV0),
		Field: "agent_ack",
	}}
	if codexReceiptAckIssuesAllRecoverableV0(issues) {
		t.Fatalf("correlacion ajena (causalidad rota) NO debe ser recuperable")
	}
}

func TestCodexReceiptAckIssuesAllRecoverableV0ArtefactoProhibidoEsDuro(t *testing.T) {
	issues := []orquestaruntime.ExternalAgentConnectorErrorV0{{
		Code:     orquestaruntime.ExternalAgentConnectorErrorCodeV0(orquestaruntimecodex.CodexConnectorAckArtifactV0),
		Field:    "files",
		Evidence: []string{"local_artifact_excluded:control_file"},
	}}
	if codexReceiptAckIssuesAllRecoverableV0(issues) {
		t.Fatalf("artefacto de control/fuera de write-set NO debe ser recuperable")
	}
}

func TestCodexReceiptAckIssueGateRefsV0ProyectaGateIssue(t *testing.T) {
	refs := codexReceiptAckIssueGateRefsV0([]orquestaruntime.ExternalAgentConnectorErrorV0{artifactPathInvalidIssueV0()})
	wantBase := "gate-issue:ack_files"
	wantEvidence := "gate-issue:ack_files:artifact_path_invalid"
	if !stringInSliceV0(refs, wantBase) || !stringInSliceV0(refs, wantEvidence) {
		t.Fatalf("refs=%v quiere %q y %q", refs, wantBase, wantEvidence)
	}
}

func TestCleanCodexReceiptAckFilePathInProjectV0AbsolutaDentroNormaliza(t *testing.T) {
	project := t.TempDir()
	abs := filepath.Join(project, "salidas", "tema.md")
	got, ok := cleanCodexReceiptAckFilePathInProjectV0(abs, project)
	if !ok || got != "salidas/tema.md" {
		t.Fatalf("got=%q ok=%v, quiere salidas/tema.md", got, ok)
	}
}

func TestCleanCodexReceiptAckFilePathInProjectV0AbsolutaFueraRechaza(t *testing.T) {
	project := t.TempDir()
	if _, ok := cleanCodexReceiptAckFilePathInProjectV0("/etc/passwd", project); ok {
		t.Fatalf("ruta fuera del proyecto debe rechazarse")
	}
}

func TestCleanCodexReceiptAckFilePathInProjectV0RelativaSigueValida(t *testing.T) {
	project := t.TempDir()
	got, ok := cleanCodexReceiptAckFilePathInProjectV0("salidas/tema.md", project)
	if !ok || got != "salidas/tema.md" {
		t.Fatalf("got=%q ok=%v", got, ok)
	}
}

func TestVerifyCodexReceiptAckFilesEvidenceV0FicheroAusenteConOtroPresenteNoCorta(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "real.md"), []byte("contenido real"), 0o600); err != nil {
		t.Fatalf("escribir fichero real: %v", err)
	}
	refs, err := verifyCodexReceiptAckFilesEvidenceV0(CodexReceiptWorktreeVerificationRequestV0{
		ProjectWorkDir: project,
		AckFiles:       []string{"real.md", "fantasma.md"},
	})
	if err != nil {
		t.Fatalf("no debe cortar cuando hay entrega real: %v", err)
	}
	if !stringInSliceV0(refs, "gate-issue:ack_file_missing:fantasma.md") {
		t.Fatalf("refs=%v debe conservar el ausente como gate-issue", refs)
	}
}

func TestVerifyCodexReceiptAckFilesEvidenceV0TodosAusentesCorta(t *testing.T) {
	project := t.TempDir()
	_, err := verifyCodexReceiptAckFilesEvidenceV0(CodexReceiptWorktreeVerificationRequestV0{
		ProjectWorkDir: project,
		AckFiles:       []string{"fantasma.md"},
	})
	if err == nil {
		t.Fatalf("sin entrega real debe cortar en duro")
	}
}

func stringInSliceV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
