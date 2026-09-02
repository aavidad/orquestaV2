//go:build v38_physical

package acceptance_test

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"orquesta/internal/candidateseal"
)

const (
	physicalGateBReceiptEnv  = "ORQUESTA_V38_GATE_B_RECEIPT"
	physicalGateBBindingEnv  = "ORQUESTA_V38_GATE_B_BINDING"
	physicalGateBEvidenceEnv = "ORQUESTA_V38_GATE_B_EVIDENCE_DIR"
)

// TestV38AgentFirecrackerSingleVMPhysicalReceipt is intentionally available
// only behind v38_physical. An explicit physical invocation without all three
// inputs fails; a normal unit suite can neither skip nor fabricate Gate B.
func TestV38AgentFirecrackerSingleVMPhysicalReceipt(t *testing.T) {
	receiptPath := requiredPhysicalGateBPath(t, physicalGateBReceiptEnv)
	bindingPath := requiredPhysicalGateBPath(t, physicalGateBBindingEnv)
	evidenceRoot := requiredPhysicalGateBPath(t, physicalGateBEvidenceEnv)
	evidenceInfo, evidenceErr := os.Lstat(evidenceRoot)
	if evidenceErr != nil || !evidenceInfo.IsDir() || evidenceInfo.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("%s must name a real directory: %v", physicalGateBEvidenceEnv, evidenceErr)
	}

	receiptBytes := readPhysicalGateBRegular(t, receiptPath)
	receipt, err := candidateseal.DecodePhysicalGateBReceipt(receiptBytes)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := candidateseal.DecodeGateBBinding(readPhysicalGateBRegular(t, bindingPath))
	if err != nil {
		t.Fatal(err)
	}
	subjects := make(map[string][]byte)
	for _, role := range candidateseal.RequiredPhysicalGateBSubjectRoles() {
		subjects[role] = readPhysicalGateBRegular(t, filepath.Join(evidenceRoot, role))
	}
	if err := candidateseal.VerifyPhysicalGateBReceipt(receipt, binding, subjects); err != nil {
		t.Fatal(err)
	}

	run := receipt.Run
	want := candidateseal.PhysicalGateBRun{
		ProjectRef: "project:default", GoalRef: "goal:e5aa533c285ee4fdc6727fa86ac367ff",
		WorkItemRef:        "work-item:c87fa56888d135e84d3e7017f823f319",
		ExecutionRef:       "execution:189f66f40fb6dd5d30301bf211035086",
		ActionRef:          "action:launch:execution:189f66f40fb6dd5d30301bf211035086",
		EffectIntentRef:    "effect-intent:action:launch:execution:189f66f40fb6dd5d30301bf211035086",
		EffectIntentDigest: "7661e558c18d93d61f2ba5a42c8be3054f5bb4d6a60fb3c95a0ca99075f2d35a",
		EffectAttemptRef:   "effect-attempt:action:launch:execution:189f66f40fb6dd5d30301bf211035086:claim:143b6491a2f4795489eced1f91ba78fb",
		PlanGeneration:     1, WorkItemGeneration: 1, ActionFence: 1,
	}
	if run.ProjectRef != want.ProjectRef || run.GoalRef != want.GoalRef ||
		run.WorkItemRef != want.WorkItemRef || run.ExecutionRef != want.ExecutionRef ||
		run.ActionRef != want.ActionRef || run.EffectIntentRef != want.EffectIntentRef ||
		run.EffectIntentDigest != want.EffectIntentDigest || run.EffectAttemptRef != want.EffectAttemptRef ||
		run.PlanGeneration != want.PlanGeneration || run.WorkItemGeneration != want.WorkItemGeneration ||
		run.ActionFence != want.ActionFence {
		t.Fatalf("physical receipt belongs to another run: %+v", run)
	}
	if run.ResultMarker != "ORQUESTA_V38_C46_OK_c5b97edf-899d-47e1-9795-677e9fc29603" ||
		!reflect.DeepEqual(receipt.Proof.AllowedServices, []string{"controlled_egress_proxy", "orquesta_broker"}) {
		t.Fatal("real Codex result or vsock service set drifted")
	}
	t.Logf("B12_GATE_B_PHYSICAL_OK receipt=%s", receipt.ReceiptSHA256)
}

func requiredPhysicalGateBPath(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" || !filepath.IsAbs(value) || filepath.Clean(value) != value {
		t.Fatalf("%s must be an absolute clean path", name)
	}
	return value
}

func readPhysicalGateBRegular(t *testing.T, path string) []byte {
	t.Helper()
	directory, name := filepath.Split(path)
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatalf("open physical evidence root %q: %v", directory, err)
	}
	defer root.Close()
	before, err := root.Lstat(name)
	if err != nil || !before.Mode().IsRegular() || before.Size() <= 0 || before.Size() > 16<<20 {
		t.Fatalf("physical evidence %q is absent or invalid: %v", path, err)
	}
	file, err := root.Open(name)
	if err != nil {
		t.Fatalf("open physical evidence %q: %v", path, err)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(before, opened) {
		t.Fatalf("physical evidence %q changed before open", path)
	}
	content, err := io.ReadAll(io.LimitReader(file, 16<<20+1))
	after, afterErr := root.Lstat(name)
	if err != nil || afterErr != nil || len(content) > 16<<20 || int64(len(content)) != opened.Size() ||
		!os.SameFile(opened, after) {
		t.Fatalf("read physical evidence %q: %v", path, err)
	}
	return content
}
