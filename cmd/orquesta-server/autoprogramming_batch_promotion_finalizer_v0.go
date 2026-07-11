package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
)

const serverAutoprogrammingBatchPromotionReceiptSchemaV0 = "orquesta.server.autoprogramming_batch_promotion_receipt.v0"

type serverAutoprogrammingBatchPromotionFinalizerV0 struct {
	CanonicalWorkDir string
	ReceiptDir       string
	gitOutput        func(context.Context, string, ...string) (string, error)
}

type serverAutoprogrammingBatchPromotionReceiptV0 struct {
	SchemaVersion      string   `json:"schema_version"`
	ReceiptRef         string   `json:"receipt_ref"`
	BatchRef           string   `json:"batch_ref"`
	GateGeneration     uint64   `json:"gate_generation"`
	ClaimRef           string   `json:"claim_ref"`
	IntegratedRevision string   `json:"integrated_revision"`
	CanonicalWorkDir   string   `json:"canonical_work_dir"`
	CanonicalClean     bool     `json:"canonical_clean"`
	EvidenceRefs       []string `json:"evidence_refs"`
}

var _ orquestaappcodexstack.AutoprogrammingBatchPromotionFinalizerPortV0 = (*serverAutoprogrammingBatchPromotionFinalizerV0)(nil)
var _ orquestaappcodexstack.AutoprogrammingBatchPromotionClaimReconcilerPortV0 = (*serverAutoprogrammingBatchPromotionFinalizerV0)(nil)

func newServerAutoprogrammingBatchPromotionFinalizerV0(
	canonicalWorkDir string,
	receiptDir string,
) *serverAutoprogrammingBatchPromotionFinalizerV0 {
	canonicalWorkDir, _ = filepath.Abs(strings.TrimSpace(canonicalWorkDir))
	receiptDir, _ = filepath.Abs(strings.TrimSpace(receiptDir))
	return &serverAutoprogrammingBatchPromotionFinalizerV0{
		CanonicalWorkDir: filepath.Clean(canonicalWorkDir),
		ReceiptDir:       filepath.Clean(receiptDir),
		gitOutput:        serverAutoprogrammingBatchGitOutputV0,
	}
}

func (finalizer *serverAutoprogrammingBatchPromotionFinalizerV0) FinalizeAutoprogrammingBatchPromotionV0(
	ctx context.Context,
	request orquestaappcodexstack.AutoprogrammingBatchPromotionRequestV0,
) (orquestaappcodexstack.AutoprogrammingBatchPromotionResultV0, error) {
	if err := finalizer.validateRequestV0(request); err != nil {
		return orquestaappcodexstack.AutoprogrammingBatchPromotionResultV0{}, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return orquestaappcodexstack.AutoprogrammingBatchPromotionResultV0{}, err
	}
	if err := os.MkdirAll(finalizer.ReceiptDir, 0o700); err != nil {
		return orquestaappcodexstack.AutoprogrammingBatchPromotionResultV0{}, fmt.Errorf("autoprogramming_batch_promotion_receipt_dir")
	}
	return finalizer.withIntegrationLockV0(func() (orquestaappcodexstack.AutoprogrammingBatchPromotionResultV0, error) {
		if receipt, found, err := finalizer.loadReceiptV0(request); err != nil {
			return orquestaappcodexstack.AutoprogrammingBatchPromotionResultV0{}, err
		} else if found {
			return serverAutoprogrammingBatchPromotionResultFromReceiptV0(receipt), nil
		}
		status, err := finalizer.gitOutputV0(ctx, "status", "--porcelain", "--untracked-files=all")
		if err != nil || strings.TrimSpace(status) != "" {
			return orquestaappcodexstack.AutoprogrammingBatchPromotionResultV0{}, fmt.Errorf("autoprogramming_batch_promotion_canonical_dirty")
		}
		head, err := finalizer.gitOutputV0(ctx, "rev-parse", "--verify", "HEAD^{commit}")
		if err != nil || strings.TrimSpace(head) != strings.TrimSpace(request.IntegratedRevision) {
			return orquestaappcodexstack.AutoprogrammingBatchPromotionResultV0{}, fmt.Errorf("autoprogramming_batch_promotion_integrated_revision_mismatch")
		}
		receipt := finalizer.receiptForRequestV0(request)
		if err := finalizer.writeReceiptV0(receipt); err != nil {
			return orquestaappcodexstack.AutoprogrammingBatchPromotionResultV0{}, err
		}
		return serverAutoprogrammingBatchPromotionResultFromReceiptV0(receipt), nil
	})
}

func (finalizer *serverAutoprogrammingBatchPromotionFinalizerV0) ReconcileAutoprogrammingBatchPromotionV0(
	ctx context.Context,
	request orquestaappcodexstack.AutoprogrammingBatchPromotionRequestV0,
) (orquestaappcodexstack.AutoprogrammingBatchPromotionResultV0, bool, error) {
	if err := finalizer.validateRequestV0(request); err != nil {
		return orquestaappcodexstack.AutoprogrammingBatchPromotionResultV0{}, false, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return orquestaappcodexstack.AutoprogrammingBatchPromotionResultV0{}, false, err
	}
	receipt, found, err := finalizer.loadReceiptV0(request)
	if err != nil || !found {
		return orquestaappcodexstack.AutoprogrammingBatchPromotionResultV0{}, found, err
	}
	return serverAutoprogrammingBatchPromotionResultFromReceiptV0(receipt), true, nil
}

func (finalizer *serverAutoprogrammingBatchPromotionFinalizerV0) validateRequestV0(request orquestaappcodexstack.AutoprogrammingBatchPromotionRequestV0) error {
	canonical, err := filepath.Abs(strings.TrimSpace(request.CanonicalWorkDir))
	if err != nil || canonical != finalizer.CanonicalWorkDir || strings.TrimSpace(request.CanonicalWorkDir) != canonical {
		return fmt.Errorf("autoprogramming_batch_promotion_canonical_work_dir_mismatch")
	}
	resolved, err := filepath.EvalSymlinks(canonical)
	if err != nil || filepath.Clean(resolved) != canonical {
		return fmt.Errorf("autoprogramming_batch_promotion_canonical_work_dir_invalid")
	}
	receiptDir, err := filepath.Abs(strings.TrimSpace(request.ReceiptDir))
	if err != nil || receiptDir != finalizer.ReceiptDir || strings.TrimSpace(request.ReceiptDir) != receiptDir {
		return fmt.Errorf("autoprogramming_batch_promotion_receipt_dir_mismatch")
	}
	if strings.TrimSpace(request.BatchRef) == "" || request.GateGeneration == 0 ||
		strings.TrimSpace(request.ClaimRef) == "" || strings.TrimSpace(request.IntegratedRevision) == "" {
		return fmt.Errorf("autoprogramming_batch_promotion_request_invalid")
	}
	return nil
}

func (finalizer *serverAutoprogrammingBatchPromotionFinalizerV0) withIntegrationLockV0(
	fn func() (orquestaappcodexstack.AutoprogrammingBatchPromotionResultV0, error),
) (orquestaappcodexstack.AutoprogrammingBatchPromotionResultV0, error) {
	lock, err := os.OpenFile(filepath.Join(finalizer.ReceiptDir, ".goal-workspace-integration.lock"), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return orquestaappcodexstack.AutoprogrammingBatchPromotionResultV0{}, fmt.Errorf("autoprogramming_batch_promotion_lock_open")
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return orquestaappcodexstack.AutoprogrammingBatchPromotionResultV0{}, fmt.Errorf("autoprogramming_batch_promotion_lock")
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	return fn()
}

func (finalizer *serverAutoprogrammingBatchPromotionFinalizerV0) receiptForRequestV0(request orquestaappcodexstack.AutoprogrammingBatchPromotionRequestV0) serverAutoprogrammingBatchPromotionReceiptV0 {
	identity := strings.Join([]string{request.BatchRef, fmt.Sprint(request.GateGeneration), request.ClaimRef, request.IntegratedRevision}, "\x00")
	sum := sha256.Sum256([]byte(identity))
	return serverAutoprogrammingBatchPromotionReceiptV0{
		SchemaVersion:      serverAutoprogrammingBatchPromotionReceiptSchemaV0,
		ReceiptRef:         "autoprogramming-batch-promotion-receipt-v0/" + hex.EncodeToString(sum[:16]) + ".json",
		BatchRef:           strings.TrimSpace(request.BatchRef),
		GateGeneration:     request.GateGeneration,
		ClaimRef:           strings.TrimSpace(request.ClaimRef),
		IntegratedRevision: strings.TrimSpace(request.IntegratedRevision),
		CanonicalWorkDir:   finalizer.CanonicalWorkDir,
		CanonicalClean:     true,
		EvidenceRefs:       compactServerAutoprogrammingBatchRefsV0(append(append([]string(nil), request.EvidenceRefs...), "evidence-ref-autoprogramming-batch-promotion-clean")),
	}
}

func (finalizer *serverAutoprogrammingBatchPromotionFinalizerV0) writeReceiptV0(receipt serverAutoprogrammingBatchPromotionReceiptV0) error {
	data, err := json.Marshal(receipt)
	if err != nil {
		return fmt.Errorf("autoprogramming_batch_promotion_receipt_encode")
	}
	if err := writeCommandDurableFileV0(finalizer.receiptPathV0(receipt.ReceiptRef), append(data, '\n'), "autoprogramming_batch_promotion_receipt"); err != nil {
		return err
	}
	return nil
}

func (finalizer *serverAutoprogrammingBatchPromotionFinalizerV0) loadReceiptV0(request orquestaappcodexstack.AutoprogrammingBatchPromotionRequestV0) (serverAutoprogrammingBatchPromotionReceiptV0, bool, error) {
	want := finalizer.receiptForRequestV0(request)
	raw, err := os.ReadFile(finalizer.receiptPathV0(want.ReceiptRef))
	if os.IsNotExist(err) {
		return serverAutoprogrammingBatchPromotionReceiptV0{}, false, nil
	}
	if err != nil {
		return serverAutoprogrammingBatchPromotionReceiptV0{}, false, fmt.Errorf("autoprogramming_batch_promotion_receipt_read")
	}
	var receipt serverAutoprogrammingBatchPromotionReceiptV0
	if json.Unmarshal(raw, &receipt) != nil || !serverAutoprogrammingBatchPromotionReceiptMatchesV0(receipt, want) {
		return serverAutoprogrammingBatchPromotionReceiptV0{}, false, fmt.Errorf("autoprogramming_batch_promotion_receipt_invalid")
	}
	return receipt, true, nil
}

func (finalizer *serverAutoprogrammingBatchPromotionFinalizerV0) receiptPathV0(receiptRef string) string {
	return filepath.Join(finalizer.ReceiptDir, filepath.Base(strings.TrimSpace(receiptRef)))
}

func (finalizer *serverAutoprogrammingBatchPromotionFinalizerV0) gitOutputV0(ctx context.Context, args ...string) (string, error) {
	if finalizer.gitOutput == nil {
		return "", fmt.Errorf("autoprogramming_batch_promotion_git_unavailable")
	}
	return finalizer.gitOutput(ctx, finalizer.CanonicalWorkDir, args...)
}

func serverAutoprogrammingBatchGitOutputV0(ctx context.Context, repo string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", append([]string{"-C", repo}, args...)...)
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=Never")
	output, err := command.Output()
	return strings.TrimSpace(string(output)), err
}

func serverAutoprogrammingBatchPromotionReceiptMatchesV0(got, want serverAutoprogrammingBatchPromotionReceiptV0) bool {
	return got.SchemaVersion == serverAutoprogrammingBatchPromotionReceiptSchemaV0 && got.ReceiptRef == want.ReceiptRef &&
		got.BatchRef == want.BatchRef && got.GateGeneration == want.GateGeneration && got.ClaimRef == want.ClaimRef &&
		got.IntegratedRevision == want.IntegratedRevision && got.CanonicalWorkDir == want.CanonicalWorkDir && got.CanonicalClean &&
		strings.Join(compactServerAutoprogrammingBatchRefsV0(got.EvidenceRefs), "\x00") == strings.Join(want.EvidenceRefs, "\x00")
}

func serverAutoprogrammingBatchPromotionResultFromReceiptV0(receipt serverAutoprogrammingBatchPromotionReceiptV0) orquestaappcodexstack.AutoprogrammingBatchPromotionResultV0 {
	return orquestaappcodexstack.AutoprogrammingBatchPromotionResultV0{
		BatchRef:           receipt.BatchRef,
		GateGeneration:     receipt.GateGeneration,
		ClaimRef:           receipt.ClaimRef,
		IntegratedRevision: receipt.IntegratedRevision,
		CanonicalClean:     receipt.CanonicalClean,
		ReceiptRef:         receipt.ReceiptRef,
		EvidenceRefs:       compactServerAutoprogrammingBatchRefsV0(append(append([]string(nil), receipt.EvidenceRefs...), receipt.ReceiptRef)),
	}
}
