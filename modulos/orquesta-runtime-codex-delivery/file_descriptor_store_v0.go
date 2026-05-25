package orquestaruntimecodexdelivery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	orquestadirectoragentfilesource "orquesta/modulos/orquesta-director-agent-file-source"
)

const (
	fileCodexReceiptDescriptorStoreSchemaVersionV0 = "orquesta.runtime.codex_delivery.receipts.file.v0"
	fileCodexReceiptDescriptorStoreNameV0          = "codex_receipt_descriptors_v0.json"
)

type FileCodexReceiptDescriptorStoreV0 struct {
	mu          sync.Mutex
	path        string
	descriptors []CodexReceiptDescriptorV0
}

type fileCodexReceiptDescriptorSnapshotV0 struct {
	SchemaVersion string                     `json:"schema_version"`
	Descriptors   []CodexReceiptDescriptorV0 `json:"descriptors"`
}

func NewFileCodexReceiptDescriptorStoreV0(dir string) (*FileCodexReceiptDescriptorStoreV0, error) {
	dir = filepath.Clean(dir)
	if dir == "." || dir == "" || !filepath.IsAbs(dir) {
		return nil, fmt.Errorf("codex_receipt_descriptor_file_store: dir_invalid")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("codex_receipt_descriptor_file_store: dir_unavailable")
	}
	path := filepath.Join(dir, fileCodexReceiptDescriptorStoreNameV0)
	descriptors, err := loadFileCodexReceiptDescriptorsV0(path)
	if err != nil {
		return nil, err
	}
	return &FileCodexReceiptDescriptorStoreV0{
		path:        path,
		descriptors: descriptors,
	}, nil
}

func (store *FileCodexReceiptDescriptorStoreV0) RecordCodexReceiptDescriptorV0(
	ctx context.Context,
	descriptor CodexReceiptDescriptorV0,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	descriptor = normalizeCodexReceiptDescriptorV0(descriptor)
	if err := validateCodexReceiptDescriptorV0(descriptor); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	next := append([]CodexReceiptDescriptorV0(nil), store.descriptors...)
	replaced := false
	for index := range next {
		if next[index].DescriptorRef == descriptor.DescriptorRef {
			next[index] = descriptor
			replaced = true
			break
		}
	}
	if !replaced {
		next = append(next, descriptor)
	}
	if err := persistFileCodexReceiptDescriptorsV0(store.path, next); err != nil {
		return err
	}
	store.descriptors = next
	return nil
}

func (store *FileCodexReceiptDescriptorStoreV0) ListCodexReceiptDescriptorsV0(
	ctx context.Context,
	request CodexReceiptDescriptorRequestV0,
) ([]CodexReceiptDescriptorV0, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	request = normalizeCodexReceiptDescriptorRequestV0(request)
	store.mu.Lock()
	defer store.mu.Unlock()
	result := make([]CodexReceiptDescriptorV0, 0, len(store.descriptors))
	for _, descriptor := range store.descriptors {
		if codexReceiptDescriptorMatchesRequestV0(descriptor, request) {
			result = append(result, descriptor)
		}
	}
	return append([]CodexReceiptDescriptorV0(nil), result...), nil
}

func (store *FileCodexReceiptDescriptorStoreV0) RecordDirectorAgentDecisionFileConsumptionV0(
	ctx context.Context,
	receipt orquestadirectoragentfilesource.DirectorAgentDecisionSidecarReceiptV0,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	receipt = normalizeDirectorDecisionSidecarReceiptV0(receipt)
	if err := validateDirectorDecisionSidecarReceiptForStoreV0(receipt); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	next := append([]CodexReceiptDescriptorV0(nil), store.descriptors...)
	for index := range next {
		if !codexReceiptDescriptorMatchesDecisionSidecarV0(next[index], receipt) {
			continue
		}
		next[index].DirectorDecisionSidecarReceipt = &receipt
		if err := persistFileCodexReceiptDescriptorsV0(store.path, next); err != nil {
			return err
		}
		store.descriptors = next
		return nil
	}
	return fmt.Errorf("codex_receipt_descriptor_file_store: director_decision_sidecar_unknown")
}

func loadFileCodexReceiptDescriptorsV0(path string) ([]CodexReceiptDescriptorV0, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("codex_receipt_descriptor_file_store: read_failed")
	}
	var snapshot fileCodexReceiptDescriptorSnapshotV0
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("codex_receipt_descriptor_file_store: json_invalid")
	}
	if snapshot.SchemaVersion != fileCodexReceiptDescriptorStoreSchemaVersionV0 {
		return nil, fmt.Errorf("codex_receipt_descriptor_file_store: schema_invalid")
	}
	descriptors := make([]CodexReceiptDescriptorV0, 0, len(snapshot.Descriptors))
	for _, descriptor := range snapshot.Descriptors {
		descriptor = normalizeCodexReceiptDescriptorV0(descriptor)
		if err := validateCodexReceiptDescriptorV0(descriptor); err != nil {
			return nil, fmt.Errorf("codex_receipt_descriptor_file_store: descriptor_invalid")
		}
		if descriptor.DirectorDecisionSidecarReceipt != nil {
			receipt := normalizeDirectorDecisionSidecarReceiptV0(*descriptor.DirectorDecisionSidecarReceipt)
			if err := validateDirectorDecisionSidecarReceiptForStoreV0(receipt); err != nil {
				return nil, fmt.Errorf("codex_receipt_descriptor_file_store: sidecar_receipt_invalid")
			}
			descriptor.DirectorDecisionSidecarReceipt = &receipt
		}
		descriptors = append(descriptors, descriptor)
	}
	return descriptors, nil
}

func persistFileCodexReceiptDescriptorsV0(
	path string,
	descriptors []CodexReceiptDescriptorV0,
) error {
	snapshot := fileCodexReceiptDescriptorSnapshotV0{
		SchemaVersion: fileCodexReceiptDescriptorStoreSchemaVersionV0,
		Descriptors:   append([]CodexReceiptDescriptorV0(nil), descriptors...),
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("codex_receipt_descriptor_file_store: json_failed")
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("codex_receipt_descriptor_file_store: write_failed")
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("codex_receipt_descriptor_file_store: rename_failed")
	}
	return nil
}
