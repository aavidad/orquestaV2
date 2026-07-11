package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
)

func TestAutoprogrammingBatchPromotionFinalizerV0BloqueaCanonicalDirtySinReceiptV0(t *testing.T) {
	repo := serverAutoprogrammingBatchPromotionGitRepoV0(t)
	receiptDir := filepath.Join(t.TempDir(), "integration-receipts")
	finalizer := newServerAutoprogrammingBatchPromotionFinalizerV0(repo, receiptDir)
	request := serverAutoprogrammingBatchPromotionRequestForTestV0(t, repo, receiptDir)
	if err := os.WriteFile(filepath.Join(repo, "dirty.txt"), []byte("dirty\n"), 0o600); err != nil {
		t.Fatalf("write dirty: %v", err)
	}
	if _, err := finalizer.FinalizeAutoprogrammingBatchPromotionV0(context.Background(), request); err == nil || err.Error() != "autoprogramming_batch_promotion_canonical_dirty" {
		t.Fatalf("Finalize error=%v", err)
	}
	if _, err := os.Stat(finalizer.receiptPathV0(finalizer.receiptForRequestV0(request).ReceiptRef)); !os.IsNotExist(err) {
		t.Fatalf("receipt creado con arbol dirty: %v", err)
	}
}

func TestAutoprogrammingBatchPromotionFinalizerV0BloqueaHEADDistintoV0(t *testing.T) {
	repo := serverAutoprogrammingBatchPromotionGitRepoV0(t)
	receiptDir := filepath.Join(t.TempDir(), "integration-receipts")
	finalizer := newServerAutoprogrammingBatchPromotionFinalizerV0(repo, receiptDir)
	request := serverAutoprogrammingBatchPromotionRequestForTestV0(t, repo, receiptDir)
	request.IntegratedRevision = strings.Repeat("a", 40)
	if _, err := finalizer.FinalizeAutoprogrammingBatchPromotionV0(context.Background(), request); err == nil || err.Error() != "autoprogramming_batch_promotion_integrated_revision_mismatch" {
		t.Fatalf("Finalize error=%v", err)
	}
	if _, err := os.Stat(finalizer.receiptPathV0(finalizer.receiptForRequestV0(request).ReceiptRef)); !os.IsNotExist(err) {
		t.Fatalf("receipt creado con HEAD distinto: %v", err)
	}
}

func TestAutoprogrammingBatchPromotionFinalizerV0AtestaRevisionLimpiaSinGitMutadorV0(t *testing.T) {
	repo := serverAutoprogrammingBatchPromotionGitRepoV0(t)
	receiptDir := filepath.Join(t.TempDir(), "integration-receipts")
	finalizer := newServerAutoprogrammingBatchPromotionFinalizerV0(repo, receiptDir)
	request := serverAutoprogrammingBatchPromotionRequestForTestV0(t, repo, receiptDir)
	headBefore := serverAutoprogrammingBatchPromotionGitV0(t, repo, "rev-parse", "HEAD")

	result, err := finalizer.FinalizeAutoprogrammingBatchPromotionV0(context.Background(), request)
	if err != nil || !result.CanonicalClean || result.IntegratedRevision != headBefore || result.ReceiptRef == "" ||
		strings.ContainsAny(result.ReceiptRef, `/\\`) || strings.HasSuffix(result.ReceiptRef, ".json") ||
		!serverAutoprogrammingBatchHasRefV0(result.EvidenceRefs, "evidence-ref-autoprogramming-batch-promotion-clean") {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if headAfter := serverAutoprogrammingBatchPromotionGitV0(t, repo, "rev-parse", "HEAD"); headAfter != headBefore {
		t.Fatalf("HEAD mutado: got=%s want=%s", headAfter, headBefore)
	}
	if status := serverAutoprogrammingBatchPromotionGitV0(t, repo, "status", "--porcelain", "--untracked-files=all"); status != "" {
		t.Fatalf("canonical no limpio: %q", status)
	}
	if _, err := os.Stat(finalizer.receiptPathV0(result.ReceiptRef)); err != nil {
		t.Fatalf("receipt durable: %v", err)
	}
	serverAutoprogrammingBatchRecordPromotionReceiptV0(t, result)
}

func TestAutoprogrammingBatchPromotionFinalizerV0ReconcilesReceiptSinSegundoGitV0(t *testing.T) {
	repo := serverAutoprogrammingBatchPromotionGitRepoV0(t)
	receiptDir := filepath.Join(t.TempDir(), "integration-receipts")
	finalizer := newServerAutoprogrammingBatchPromotionFinalizerV0(repo, receiptDir)
	request := serverAutoprogrammingBatchPromotionRequestForTestV0(t, repo, receiptDir)
	gitCalls := 0
	finalizer.gitOutput = func(ctx context.Context, root string, args ...string) (string, error) {
		gitCalls++
		return serverAutoprogrammingBatchGitOutputV0(ctx, root, args...)
	}
	first, err := finalizer.FinalizeAutoprogrammingBatchPromotionV0(context.Background(), request)
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	callsAfterEffect := gitCalls
	finalizer.gitOutput = func(context.Context, string, ...string) (string, error) {
		gitCalls++
		return "", errors.New("git no debe ejecutarse durante reconcile")
	}
	replayed, found, err := finalizer.ReconcileAutoprogrammingBatchPromotionV0(context.Background(), request)
	if err != nil || !found || replayed.ReceiptRef != first.ReceiptRef || replayed.IntegratedRevision != first.IntegratedRevision || gitCalls != callsAfterEffect {
		t.Fatalf("replay=%+v found=%v err=%v git_calls=%d want=%d", replayed, found, err, gitCalls, callsAfterEffect)
	}
}

func serverAutoprogrammingBatchPromotionRequestForTestV0(t *testing.T, repo string, receiptDir string) orquestaappcodexstack.AutoprogrammingBatchPromotionRequestV0 {
	t.Helper()
	return orquestaappcodexstack.AutoprogrammingBatchPromotionRequestV0{
		BatchRef:           "batch-ref-autoprogramming-promotion-001",
		GateGeneration:     1,
		ClaimRef:           "claim-ref-autoprogramming-promotion-001",
		IntegratedRevision: serverAutoprogrammingBatchPromotionGitV0(t, repo, "rev-parse", "HEAD"),
		CanonicalWorkDir:   repo,
		ReceiptDir:         receiptDir,
		EvidenceRefs:       []string{"evidence-ref-autoprogramming-batch-gate-passed"},
	}
}

func serverAutoprogrammingBatchPromotionGitRepoV0(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	serverAutoprogrammingBatchPromotionGitV0(t, repo, "init")
	serverAutoprogrammingBatchPromotionGitV0(t, repo, "config", "user.email", "batch@example.test")
	serverAutoprogrammingBatchPromotionGitV0(t, repo, "config", "user.name", "Batch Test")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("batch promotion\n"), 0o600); err != nil {
		t.Fatalf("write README: %v", err)
	}
	serverAutoprogrammingBatchPromotionGitV0(t, repo, "add", "README.md")
	serverAutoprogrammingBatchPromotionGitV0(t, repo, "commit", "-m", "initial")
	return repo
}

func serverAutoprogrammingBatchPromotionGitV0(t *testing.T, repo string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repo}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

func serverAutoprogrammingBatchRecordPromotionReceiptV0(
	t *testing.T,
	result orquestaappcodexstack.AutoprogrammingBatchPromotionResultV0,
) {
	t.Helper()
	command := "go test ./..."
	sum := sha256.Sum256([]byte(command))
	test := orquestaautoprogramming.AutoprogrammingBatchTestV0{Command: command, SHA256: hex.EncodeToString(sum[:])}
	testHash := orquestaautoprogramming.AutoprogrammingBatchTestHashV0(test)
	batch := serverAutoprogrammingBatchClaimedTestV0(t, result.IntegratedRevision, test, testHash, "claim-ref-promotion-contract-test")
	batch = serverAutoprogrammingBatchMustTransitionV0(t, orquestaautoprogramming.RecordAutoprogrammingBatchTestReceiptV0(
		batch, batch.StoreVersion, "record-promotion-contract-test", result.IntegratedRevision, testHash,
		"claim-ref-promotion-contract-test", "receipt-ref-promotion-contract-test", orquestaautoprogramming.AutoprogrammingBatchTestReceiptPassedV0,
	))
	batch = serverAutoprogrammingBatchMustTransitionV0(t, orquestaautoprogramming.ClaimAutoprogrammingBatchPromotionV0(
		batch, batch.StoreVersion, "claim-promotion-contract", result.ClaimRef, result.IntegratedRevision,
	))
	recorded := orquestaautoprogramming.RegisterAutoprogrammingBatchPromotionV0(
		batch, batch.StoreVersion, "record-promotion-contract", result.ClaimRef, result.IntegratedRevision, result.ReceiptRef,
	)
	if !recorded.Accepted || recorded.Batch.PromotionReceipt.ReceiptRef != result.ReceiptRef {
		t.Fatalf("receipt de finalizer rechazado por agregado: result=%+v transition=%+v", result, recorded)
	}
}
