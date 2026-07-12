package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

const runtimeModelMutationReceiptSchemaV0 = "runtime_model_mutation_receipt.v0"

type runtimeModelMutationReceiptV0 struct {
	SchemaVersion string   `json:"schema_version"`
	ReceiptRef    string   `json:"receipt_ref"`
	OperationRef  string   `json:"operation_ref"`
	PayloadHash   string   `json:"payload_hash"`
	Action        string   `json:"action"`
	Model         string   `json:"model"`
	Status        string   `json:"status"`
	ResultStatus  string   `json:"result_status,omitempty"`
	EvidenceRefs  []string `json:"evidence_refs"`
}

type runtimeModelMutationReceiptStoreV0 interface {
	LoadRuntimeModelMutationReceiptV0(string) (runtimeModelMutationReceiptV0, bool, error)
	SaveRuntimeModelMutationReceiptV0(runtimeModelMutationReceiptV0) error
}

type fileRuntimeModelMutationReceiptStoreV0 struct {
	dir string
	mu  sync.Mutex
}

func newFileRuntimeModelMutationReceiptStoreV0(stateDir string) (*fileRuntimeModelMutationReceiptStoreV0, error) {
	dir := filepath.Join(strings.TrimSpace(stateDir), "runtime-model-mutation-receipts")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("runtime_model_receipt_store_unavailable")
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return nil, fmt.Errorf("runtime_model_receipt_store_unavailable")
	}
	return &fileRuntimeModelMutationReceiptStoreV0{dir: dir}, nil
}

func (store *fileRuntimeModelMutationReceiptStoreV0) LoadRuntimeModelMutationReceiptV0(ref string) (runtimeModelMutationReceiptV0, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	raw, err := os.ReadFile(store.pathV0(ref))
	if os.IsNotExist(err) {
		return runtimeModelMutationReceiptV0{}, false, nil
	}
	if err != nil || len(raw) > 64*1024 {
		return runtimeModelMutationReceiptV0{}, false, fmt.Errorf("runtime_model_receipt_read_failed")
	}
	var receipt runtimeModelMutationReceiptV0
	if json.Unmarshal(raw, &receipt) != nil || receipt.SchemaVersion != runtimeModelMutationReceiptSchemaV0 || receipt.ReceiptRef != ref {
		return runtimeModelMutationReceiptV0{}, false, fmt.Errorf("runtime_model_receipt_invalid")
	}
	return receipt, true, nil
}

func (store *fileRuntimeModelMutationReceiptStoreV0) SaveRuntimeModelMutationReceiptV0(receipt runtimeModelMutationReceiptV0) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	raw, err := json.Marshal(receipt)
	if err != nil {
		return fmt.Errorf("runtime_model_receipt_encode_failed")
	}
	if err := writeCommandDurableFileV0(store.pathV0(receipt.ReceiptRef), append(raw, '\n'), "runtime_model_receipt"); err != nil {
		return fmt.Errorf("runtime_model_receipt_write_failed")
	}
	return nil
}

func (store *fileRuntimeModelMutationReceiptStoreV0) pathV0(ref string) string {
	return filepath.Join(store.dir, filepath.Base(strings.TrimSpace(ref))+".json")
}

type governedRuntimeModelManagerV0 struct {
	inner   orquestaruntime.RuntimeModelManagerPortV0
	store   runtimeModelMutationReceiptStoreV0
	allowed map[string]struct{}
	mu      *sync.Mutex
}

func newGovernedRuntimeModelManagerV0(inner orquestaruntime.RuntimeModelManagerPortV0, store runtimeModelMutationReceiptStoreV0, allowed []string) governedRuntimeModelManagerV0 {
	models := make(map[string]struct{}, len(allowed))
	for _, model := range allowed {
		if model = strings.TrimSpace(model); model != "" {
			models[model] = struct{}{}
		}
	}
	return governedRuntimeModelManagerV0{inner: inner, store: store, allowed: models, mu: &sync.Mutex{}}
}

func (manager governedRuntimeModelManagerV0) ListRuntimeModelsV0(ctx context.Context, req orquestaruntime.RuntimeModelListRequestV0) (orquestaruntime.RuntimeModelListResultV0, error) {
	if err := validateRuntimeModelProviderV0(req.ProviderRef); err != nil {
		return orquestaruntime.RuntimeModelListResultV0{}, err
	}
	return manager.inner.ListRuntimeModelsV0(ctx, req)
}

func (manager governedRuntimeModelManagerV0) RuntimeModelStatusV0(ctx context.Context, req orquestaruntime.RuntimeModelListRequestV0) (orquestaruntime.RuntimeModelListResultV0, error) {
	if err := validateRuntimeModelProviderV0(req.ProviderRef); err != nil {
		return orquestaruntime.RuntimeModelListResultV0{}, err
	}
	return manager.inner.RuntimeModelStatusV0(ctx, req)
}

func (manager governedRuntimeModelManagerV0) PullRuntimeModelV0(ctx context.Context, req orquestaruntime.RuntimeModelActionRequestV0) (orquestaruntime.RuntimeModelActionResultV0, error) {
	return manager.mutateV0(ctx, "pull", req, manager.inner.PullRuntimeModelV0)
}

func (manager governedRuntimeModelManagerV0) ServeRuntimeModelV0(ctx context.Context, req orquestaruntime.RuntimeModelActionRequestV0) (orquestaruntime.RuntimeModelActionResultV0, error) {
	return manager.mutateV0(ctx, "serve", req, manager.inner.ServeRuntimeModelV0)
}

func (manager governedRuntimeModelManagerV0) StopRuntimeModelV0(ctx context.Context, req orquestaruntime.RuntimeModelActionRequestV0) (orquestaruntime.RuntimeModelActionResultV0, error) {
	return manager.mutateV0(ctx, "stop", req, manager.inner.StopRuntimeModelV0)
}

func (manager governedRuntimeModelManagerV0) mutateV0(ctx context.Context, action string, req orquestaruntime.RuntimeModelActionRequestV0, invoke func(context.Context, orquestaruntime.RuntimeModelActionRequestV0) (orquestaruntime.RuntimeModelActionResultV0, error)) (orquestaruntime.RuntimeModelActionResultV0, error) {
	if manager.mu == nil {
		return orquestaruntime.RuntimeModelActionResultV0{}, fmt.Errorf("runtime_model_governance_unavailable")
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	receipt, err := manager.intentV0(action, req)
	if err != nil {
		return orquestaruntime.RuntimeModelActionResultV0{}, err
	}
	if prior, exists, err := manager.store.LoadRuntimeModelMutationReceiptV0(receipt.ReceiptRef); err != nil {
		return orquestaruntime.RuntimeModelActionResultV0{}, err
	} else if exists {
		if prior.PayloadHash != receipt.PayloadHash {
			return orquestaruntime.RuntimeModelActionResultV0{}, fmt.Errorf("runtime_model_operation_conflict")
		}
		if prior.Status == "accepted" {
			return runtimeModelReplayResultV0(req, prior), nil
		}
		return orquestaruntime.RuntimeModelActionResultV0{}, fmt.Errorf("runtime_model_operation_not_replayable")
	}
	if err := manager.store.SaveRuntimeModelMutationReceiptV0(receipt); err != nil {
		return orquestaruntime.RuntimeModelActionResultV0{}, err
	}
	result, callErr := invoke(ctx, req)
	if callErr != nil {
		receipt.Status = "failed"
		if err := manager.store.SaveRuntimeModelMutationReceiptV0(receipt); err != nil {
			return orquestaruntime.RuntimeModelActionResultV0{}, err
		}
		return orquestaruntime.RuntimeModelActionResultV0{}, callErr
	}
	receipt.Status = "accepted"
	receipt.ResultStatus = strings.TrimSpace(result.Status)
	if err := manager.store.SaveRuntimeModelMutationReceiptV0(receipt); err != nil {
		return orquestaruntime.RuntimeModelActionResultV0{}, err
	}
	result.Evidence = append(result.Evidence, orquestaruntime.RuntimeModelEvidence{Kind: "runtime_model_mutation_receipt", Ref: receipt.ReceiptRef})
	return result, nil
}

func (manager governedRuntimeModelManagerV0) intentV0(action string, req orquestaruntime.RuntimeModelActionRequestV0) (runtimeModelMutationReceiptV0, error) {
	model := strings.TrimSpace(req.Model)
	operationRef := strings.TrimSpace(req.OperationRef)
	if manager.inner == nil || manager.store == nil {
		return runtimeModelMutationReceiptV0{}, fmt.Errorf("runtime_model_governance_unavailable")
	}
	if err := validateRuntimeModelProviderV0(req.ProviderRef); err != nil {
		return runtimeModelMutationReceiptV0{}, err
	}
	if _, ok := manager.allowed[model]; !ok || model == "" {
		return runtimeModelMutationReceiptV0{}, fmt.Errorf("runtime_model_not_allowed")
	}
	if operationRef == "" || len(operationRef) > 256 || len(req.Evidence) == 0 || len(req.Evidence) > 32 {
		return runtimeModelMutationReceiptV0{}, fmt.Errorf("runtime_model_evidence_required")
	}
	evidence := make([]string, 0, len(req.Evidence))
	for _, ref := range req.Evidence {
		ref = strings.TrimSpace(ref)
		if ref == "" || len(ref) > 512 {
			return runtimeModelMutationReceiptV0{}, fmt.Errorf("runtime_model_evidence_invalid")
		}
		evidence = append(evidence, ref)
	}
	payload := strings.Join([]string{action, model, strings.TrimSpace(req.KeepAlive), strings.TrimSpace(req.ProviderRef), strings.TrimSpace(req.EndpointRef), strings.Join(evidence, "\x00")}, "\x01")
	payloadHash := sha256.Sum256([]byte(payload))
	operationHash := sha256.Sum256([]byte(operationRef))
	return runtimeModelMutationReceiptV0{
		SchemaVersion: runtimeModelMutationReceiptSchemaV0,
		ReceiptRef:    "runtime-model-mutation-receipt-v0-" + hex.EncodeToString(operationHash[:16]),
		OperationRef:  operationRef,
		PayloadHash:   hex.EncodeToString(payloadHash[:]),
		Action:        action,
		Model:         model,
		Status:        "intent_recorded",
		EvidenceRefs:  evidence,
	}, nil
}

func validateRuntimeModelProviderV0(provider string) error {
	provider = strings.TrimSpace(provider)
	if provider != "" && provider != "ollama" {
		return fmt.Errorf("runtime_model_provider_not_allowed")
	}
	return nil
}

func runtimeModelReplayResultV0(req orquestaruntime.RuntimeModelActionRequestV0, receipt runtimeModelMutationReceiptV0) orquestaruntime.RuntimeModelActionResultV0 {
	return orquestaruntime.RuntimeModelActionResultV0{
		ProviderRef: "ollama",
		EndpointRef: strings.TrimSpace(req.EndpointRef),
		Model:       receipt.Model,
		Accepted:    true,
		Status:      receipt.ResultStatus,
		Evidence: []orquestaruntime.RuntimeModelEvidence{{
			Kind: "runtime_model_mutation_receipt",
			Ref:  receipt.ReceiptRef,
		}},
	}
}
