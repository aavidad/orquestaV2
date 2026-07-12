package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestFileRuntimeModelMutationReceiptStoreV0RecuperaReceiptDurableV0(t *testing.T) {
	stateDir := t.TempDir()
	store, err := newFileRuntimeModelMutationReceiptStoreV0(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	receipt := runtimeModelMutationReceiptV0{
		SchemaVersion: runtimeModelMutationReceiptSchemaV0,
		ReceiptRef:    "runtime-model-mutation-receipt-v0-test",
		OperationRef:  "operation-ref-test",
		PayloadHash:   "payload-hash-test",
		Action:        "pull",
		Model:         "qwen2.5:7b",
		Status:        "intent_recorded",
		EvidenceRefs:  []string{"evidence-ref-test"},
	}
	if err := store.SaveRuntimeModelMutationReceiptV0(receipt); err != nil {
		t.Fatal(err)
	}
	reopened, err := newFileRuntimeModelMutationReceiptStoreV0(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := reopened.LoadRuntimeModelMutationReceiptV0(receipt.ReceiptRef)
	if err != nil || !ok || got.PayloadHash != receipt.PayloadHash || got.Status != "intent_recorded" {
		t.Fatalf("got=%+v ok=%v err=%v", got, ok, err)
	}
	info, err := os.Stat(filepath.Join(stateDir, "runtime-model-mutation-receipts", receipt.ReceiptRef+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("receipt mode=%v", info.Mode().Perm())
	}
}

func TestGovernedRuntimeModelManagerV0NoMutaSinIntentDurableV0(t *testing.T) {
	for _, action := range []string{"pull", "serve", "stop"} {
		t.Run(action, func(t *testing.T) {
			inner := &runtimeModelMutationFakeV0{}
			store := &runtimeModelReceiptStoreFakeV0{saveErr: errors.New("disk unavailable")}
			manager := newGovernedRuntimeModelManagerV0(inner, store, []string{"qwen2.5:7b"})
			request := runtimeModelMutationRequestForTestV0(action)
			var err error
			switch action {
			case "pull":
				_, err = manager.PullRuntimeModelV0(context.Background(), request)
			case "serve":
				_, err = manager.ServeRuntimeModelV0(context.Background(), request)
			case "stop":
				_, err = manager.StopRuntimeModelV0(context.Background(), request)
			}
			if err == nil || inner.mutations != 0 {
				t.Fatalf("action=%s err=%v mutations=%d", action, err, inner.mutations)
			}
		})
	}
}

func TestGovernedRuntimeModelManagerV0AllowlistReceiptYReplayV0(t *testing.T) {
	store := &runtimeModelReceiptStoreFakeV0{}
	inner := &runtimeModelMutationFakeV0{beforeMutation: func() {
		if len(store.receipts) != 1 {
			t.Fatalf("efecto ejecutado antes del intent durable: %+v", store.receipts)
		}
		for _, receipt := range store.receipts {
			if receipt.Status != "intent_recorded" {
				t.Fatalf("status antes de efecto=%s", receipt.Status)
			}
		}
	}}
	manager := newGovernedRuntimeModelManagerV0(inner, store, []string{"qwen2.5:7b"})
	request := runtimeModelMutationRequestForTestV0("pull")
	result, err := manager.PullRuntimeModelV0(context.Background(), request)
	if err != nil || !result.Accepted || inner.mutations != 1 || len(result.Evidence) != 1 || result.Evidence[0].Kind != "runtime_model_mutation_receipt" {
		t.Fatalf("result=%+v err=%v mutations=%d", result, err, inner.mutations)
	}
	replayed, err := manager.PullRuntimeModelV0(context.Background(), request)
	if err != nil || !replayed.Accepted || inner.mutations != 1 || replayed.Evidence[0].Ref != result.Evidence[0].Ref {
		t.Fatalf("replay=%+v err=%v mutations=%d", replayed, err, inner.mutations)
	}
	blocked := request
	blocked.OperationRef = "operation-ref-runtime-model-blocked"
	blocked.Model = "arbitrary:latest"
	if _, err := manager.PullRuntimeModelV0(context.Background(), blocked); err == nil || inner.mutations != 1 {
		t.Fatalf("modelo fuera de allowlist mutó: err=%v mutations=%d", err, inner.mutations)
	}
}

func runtimeModelMutationRequestForTestV0(action string) orquestaruntime.RuntimeModelActionRequestV0 {
	return orquestaruntime.RuntimeModelActionRequestV0{
		OperationRef: "operation-ref-runtime-model-" + action,
		ProviderRef:  "ollama",
		Model:        "qwen2.5:7b",
		Evidence:     []string{"evidence-ref-runtime-model-operator-approved"},
	}
}

type runtimeModelReceiptStoreFakeV0 struct {
	receipts map[string]runtimeModelMutationReceiptV0
	saveErr  error
}

func (store *runtimeModelReceiptStoreFakeV0) LoadRuntimeModelMutationReceiptV0(ref string) (runtimeModelMutationReceiptV0, bool, error) {
	receipt, ok := store.receipts[ref]
	return receipt, ok, nil
}

func (store *runtimeModelReceiptStoreFakeV0) SaveRuntimeModelMutationReceiptV0(receipt runtimeModelMutationReceiptV0) error {
	if store.saveErr != nil {
		return store.saveErr
	}
	if store.receipts == nil {
		store.receipts = map[string]runtimeModelMutationReceiptV0{}
	}
	store.receipts[receipt.ReceiptRef] = receipt
	return nil
}

type runtimeModelMutationFakeV0 struct {
	mutations      int
	beforeMutation func()
}

func (fake *runtimeModelMutationFakeV0) ListRuntimeModelsV0(context.Context, orquestaruntime.RuntimeModelListRequestV0) (orquestaruntime.RuntimeModelListResultV0, error) {
	return orquestaruntime.RuntimeModelListResultV0{}, nil
}

func (fake *runtimeModelMutationFakeV0) RuntimeModelStatusV0(context.Context, orquestaruntime.RuntimeModelListRequestV0) (orquestaruntime.RuntimeModelListResultV0, error) {
	return orquestaruntime.RuntimeModelListResultV0{}, nil
}

func (fake *runtimeModelMutationFakeV0) PullRuntimeModelV0(_ context.Context, req orquestaruntime.RuntimeModelActionRequestV0) (orquestaruntime.RuntimeModelActionResultV0, error) {
	return fake.mutateV0(req)
}

func (fake *runtimeModelMutationFakeV0) ServeRuntimeModelV0(_ context.Context, req orquestaruntime.RuntimeModelActionRequestV0) (orquestaruntime.RuntimeModelActionResultV0, error) {
	return fake.mutateV0(req)
}

func (fake *runtimeModelMutationFakeV0) StopRuntimeModelV0(_ context.Context, req orquestaruntime.RuntimeModelActionRequestV0) (orquestaruntime.RuntimeModelActionResultV0, error) {
	return fake.mutateV0(req)
}

func (fake *runtimeModelMutationFakeV0) mutateV0(req orquestaruntime.RuntimeModelActionRequestV0) (orquestaruntime.RuntimeModelActionResultV0, error) {
	if fake.beforeMutation != nil {
		fake.beforeMutation()
	}
	fake.mutations++
	return orquestaruntime.RuntimeModelActionResultV0{ProviderRef: "ollama", Model: req.Model, Accepted: true, Status: "accepted"}, nil
}
