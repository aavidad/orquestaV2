package orquestatoolcapabilityfile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"syscall"

	toolcapability "orquesta/modulos/orquesta-tool-capability"
)

const (
	toolOperationFileStateSchemaV0 = "tool_operation_file_state.v0"
	toolOperationFileStateNameV0   = "tool-operation-state-v0.json"
	toolOperationFileLockNameV0    = "tool-operation-state-v0.lock"
	maxToolOperationFileStateV0    = 16 << 20
)

// ToolOperationFileStoreV0 is a durable reference adapter. It serializes all
// clients and processes through flock and atomically replaces one state file.
type ToolOperationFileStoreV0 struct {
	root      string
	statePath string
	lockPath  string
}

func NewToolOperationFileStoreV0(root string) (*ToolOperationFileStoreV0, error) {
	if !filepath.IsAbs(root) {
		return nil, errors.New("tool operation store root must be absolute")
	}
	root = filepath.Clean(root)
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("create tool operation store root: %w", err)
	}
	info, err := os.Lstat(root)
	if err != nil {
		return nil, fmt.Errorf("inspect tool operation store root: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("tool operation store root must be a real directory")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("tool operation store root must not be group/world accessible")
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || filepath.Clean(resolved) != root {
		return nil, errors.New("tool operation store root must not traverse symlinks")
	}
	return &ToolOperationFileStoreV0{
		root:      root,
		statePath: filepath.Join(root, toolOperationFileStateNameV0),
		lockPath:  filepath.Join(root, toolOperationFileLockNameV0),
	}, nil
}

func (store *ToolOperationFileStoreV0) ClaimToolOperationV0(
	ctx context.Context,
	record toolcapability.ToolOperationRecordV0,
) (toolcapability.ToolOperationRecordV0, bool, error) {
	if issues := toolcapability.ValidateToolOperationRecordV0(record); len(issues) > 0 {
		return toolcapability.ToolOperationRecordV0{}, false, fmt.Errorf("invalid tool operation record: %+v", issues)
	}
	var result toolcapability.ToolOperationRecordV0
	var acquired bool
	err := store.withLockedStateV0(ctx, func(state *toolOperationFileStateV0) (bool, error) {
		if existing, found := state.Records[record.Plan.OperationRef]; found {
			result = existing
			return false, nil
		}
		if operationRef, found := state.TargetLeases[record.Plan.TargetRef]; found && operationRef != record.Plan.OperationRef {
			return false, fmt.Errorf("%s", toolcapability.ErrToolCapabilityTargetBusyV0)
		}
		if operationRef, found := state.ReceiptIndex[record.Receipt.ReceiptRef]; found && operationRef != record.Plan.OperationRef {
			return false, errors.New("tool operation receipt ref collision")
		}
		state.Records[record.Plan.OperationRef] = record
		state.ReceiptIndex[record.Receipt.ReceiptRef] = record.Plan.OperationRef
		if !toolcapability.ToolReceiptStatusIsTargetTerminalV0(record.Receipt.Status) {
			state.TargetLeases[record.Plan.TargetRef] = record.Plan.OperationRef
		}
		result = record
		acquired = true
		return true, nil
	})
	return result, acquired, err
}

func (store *ToolOperationFileStoreV0) GetToolOperationRecordV0(
	ctx context.Context,
	operationRef string,
) (toolcapability.ToolOperationRecordV0, bool, error) {
	var result toolcapability.ToolOperationRecordV0
	var found bool
	err := store.withLockedStateV0(ctx, func(state *toolOperationFileStateV0) (bool, error) {
		result, found = state.Records[operationRef]
		return false, nil
	})
	return result, found, err
}

func (store *ToolOperationFileStoreV0) GetToolOperationRecordByReceiptRefV0(
	ctx context.Context,
	receiptRef string,
) (toolcapability.ToolOperationRecordV0, bool, error) {
	var result toolcapability.ToolOperationRecordV0
	var found bool
	err := store.withLockedStateV0(ctx, func(state *toolOperationFileStateV0) (bool, error) {
		operationRef, indexed := state.ReceiptIndex[receiptRef]
		if !indexed {
			return false, nil
		}
		result, found = state.Records[operationRef]
		return false, nil
	})
	return result, found, err
}

func (store *ToolOperationFileStoreV0) CompareAndSwapToolOperationV0(
	ctx context.Context,
	operationRef string,
	expectedVersion uint64,
	receipt toolcapability.ToolOperationReceiptV0,
) (toolcapability.ToolOperationRecordV0, bool, error) {
	var result toolcapability.ToolOperationRecordV0
	var swapped bool
	err := store.withLockedStateV0(ctx, func(state *toolOperationFileStateV0) (bool, error) {
		current, found := state.Records[operationRef]
		if !found || current.Receipt.StateVersion != expectedVersion {
			result = current
			return false, nil
		}
		if receipt.StateVersion != expectedVersion+1 || receipt.OperationRef != operationRef ||
			receipt.PlanFingerprint != current.Plan.PlanFingerprint || receipt.ReceiptRef != current.Receipt.ReceiptRef {
			return false, errors.New("invalid tool operation CAS transition")
		}
		next := current
		next.Receipt = receipt
		if issues := toolcapability.ValidateToolOperationRecordV0(next); len(issues) > 0 {
			return false, fmt.Errorf("invalid tool operation record: %+v", issues)
		}
		state.Records[operationRef] = next
		if toolcapability.ToolReceiptStatusIsTargetTerminalV0(receipt.Status) {
			delete(state.TargetLeases, current.Plan.TargetRef)
		} else {
			state.TargetLeases[current.Plan.TargetRef] = operationRef
		}
		result = next
		swapped = true
		return true, nil
	})
	return result, swapped, err
}

func (store *ToolOperationFileStoreV0) ListToolOperationReceiptsV0(
	ctx context.Context,
	filter toolcapability.ToolOperationReceiptFilterV0,
) ([]toolcapability.ToolOperationReceiptV0, error) {
	var receipts []toolcapability.ToolOperationReceiptV0
	err := store.withLockedStateV0(ctx, func(state *toolOperationFileStateV0) (bool, error) {
		for _, record := range state.Records {
			receipt := record.Receipt
			if (filter.ReceiptRef == "" || receipt.ReceiptRef == filter.ReceiptRef) &&
				(filter.AppRef == "" || receipt.AppRef == filter.AppRef) &&
				(filter.BindingRef == "" || receipt.BindingRef == filter.BindingRef) &&
				(filter.ToolRef == "" || receipt.ToolRef == filter.ToolRef) &&
				(filter.Operation == "" || receipt.Operation == filter.Operation) &&
				(filter.IdempotencyKey == "" || receipt.IdempotencyKey == filter.IdempotencyKey) {
				receipts = append(receipts, receipt)
			}
		}
		return false, nil
	})
	sort.Slice(receipts, func(left, right int) bool { return receipts[left].ReceiptRef < receipts[right].ReceiptRef })
	return receipts, err
}

type toolOperationFileStateV0 struct {
	SchemaVersion string
	Records       map[string]toolcapability.ToolOperationRecordV0
	ReceiptIndex  map[string]string
	TargetLeases  map[string]string
}

func newToolOperationFileStateV0() toolOperationFileStateV0 {
	return toolOperationFileStateV0{
		SchemaVersion: toolOperationFileStateSchemaV0,
		Records:       make(map[string]toolcapability.ToolOperationRecordV0),
		ReceiptIndex:  make(map[string]string),
		TargetLeases:  make(map[string]string),
	}
}

func (store *ToolOperationFileStoreV0) withLockedStateV0(
	ctx context.Context,
	operation func(*toolOperationFileStateV0) (dirty bool, err error),
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	lockFD, err := syscall.Open(store.lockPath, syscall.O_CREAT|syscall.O_RDWR|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return fmt.Errorf("open tool operation lock: %w", err)
	}
	defer syscall.Close(lockFD)
	var lockStat syscall.Stat_t
	if err := syscall.Fstat(lockFD, &lockStat); err != nil || lockStat.Mode&syscall.S_IFMT != syscall.S_IFREG || lockStat.Mode&0o077 != 0 {
		return errors.New("tool operation lock must be a private regular file")
	}
	if err := syscall.Flock(lockFD, syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock tool operation state: %w", err)
	}
	defer syscall.Flock(lockFD, syscall.LOCK_UN)
	if err := ctx.Err(); err != nil {
		return err
	}
	state, err := store.loadStateV0()
	if err != nil {
		return err
	}
	dirty, err := operation(&state)
	if err != nil {
		return err
	}
	if !dirty {
		return nil
	}
	return store.saveStateV0(state)
}

func (store *ToolOperationFileStoreV0) loadStateV0() (toolOperationFileStateV0, error) {
	fd, err := syscall.Open(store.statePath, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if errors.Is(err, syscall.ENOENT) {
		return newToolOperationFileStateV0(), nil
	}
	if err != nil {
		return toolOperationFileStateV0{}, fmt.Errorf("open tool operation state: %w", err)
	}
	file := os.NewFile(uintptr(fd), store.statePath)
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || info.Size() > maxToolOperationFileStateV0 {
		return toolOperationFileStateV0{}, errors.New("tool operation state must be a private bounded regular file")
	}
	decoder := json.NewDecoder(io.LimitReader(file, maxToolOperationFileStateV0))
	decoder.DisallowUnknownFields()
	var state toolOperationFileStateV0
	if err := decoder.Decode(&state); err != nil {
		return toolOperationFileStateV0{}, fmt.Errorf("decode tool operation state: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return toolOperationFileStateV0{}, errors.New("tool operation state contains trailing data")
	}
	if err := validateToolOperationFileStateV0(state); err != nil {
		return toolOperationFileStateV0{}, err
	}
	return state, nil
}

func validateToolOperationFileStateV0(state toolOperationFileStateV0) error {
	if state.SchemaVersion != toolOperationFileStateSchemaV0 || state.Records == nil || state.ReceiptIndex == nil || state.TargetLeases == nil {
		return errors.New("invalid tool operation file state schema")
	}
	for operationRef, record := range state.Records {
		if operationRef != record.Plan.OperationRef {
			return errors.New("tool operation index mismatch")
		}
		if issues := toolcapability.ValidateToolOperationRecordV0(record); len(issues) > 0 {
			return fmt.Errorf("invalid durable tool operation record: %+v", issues)
		}
		if state.ReceiptIndex[record.Receipt.ReceiptRef] != operationRef {
			return errors.New("tool operation receipt index mismatch")
		}
		leaseOperationRef, leased := state.TargetLeases[record.Plan.TargetRef]
		if toolcapability.ToolReceiptStatusIsTargetTerminalV0(record.Receipt.Status) {
			if leased && leaseOperationRef == operationRef {
				return errors.New("terminal tool operation retains target lease")
			}
		} else if !leased || leaseOperationRef != operationRef {
			return errors.New("active tool operation lost target lease")
		}
	}
	for receiptRef, operationRef := range state.ReceiptIndex {
		record, found := state.Records[operationRef]
		if !found || record.Receipt.ReceiptRef != receiptRef {
			return errors.New("orphan tool operation receipt index")
		}
	}
	for targetRef, operationRef := range state.TargetLeases {
		record, found := state.Records[operationRef]
		if !found || record.Plan.TargetRef != targetRef || toolcapability.ToolReceiptStatusIsTargetTerminalV0(record.Receipt.Status) {
			return errors.New("orphan tool operation target lease")
		}
	}
	return nil
}

func (store *ToolOperationFileStoreV0) saveStateV0(state toolOperationFileStateV0) error {
	if err := validateToolOperationFileStateV0(state); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(store.root, ".tool-operation-state-*.tmp")
	if err != nil {
		return fmt.Errorf("create tool operation state temp: %w", err)
	}
	temporaryPath := temporary.Name()
	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(state); err != nil {
		temporary.Close()
		return fmt.Errorf("encode tool operation state: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync tool operation state: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close tool operation state: %w", err)
	}
	if err := os.Rename(temporaryPath, store.statePath); err != nil {
		return fmt.Errorf("replace tool operation state: %w", err)
	}
	removeTemporary = false
	directory, err := os.Open(store.root)
	if err != nil {
		return fmt.Errorf("open tool operation store root: %w", err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync tool operation store root: %w", err)
	}
	return nil
}
