package orquestastatefile

import (
	"context"
	"reflect"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
)

const autoprogrammingBatchDocumentSchemaV0 = "orquesta_state_file.autoprogramming_batch.v0"

type autoprogrammingBatchDocumentV0 struct {
	SchemaVersion string                                         `json:"schema_version"`
	BatchRef      string                                         `json:"batch_ref"`
	Batch         orquestaautoprogramming.AutoprogrammingBatchV0 `json:"batch"`
}

func (store *StoreV0) LoadAutoprogrammingBatchV0(
	ctx context.Context,
	batchRef string,
) (orquestaautoprogramming.AutoprogrammingBatchV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return orquestaautoprogramming.AutoprogrammingBatchV0{}, err
	}
	batchRef = normalizeRefV0(batchRef)
	if batchRef == "" {
		return orquestaautoprogramming.AutoprogrammingBatchV0{}, invalidErrorV0("autoprogramming_batch.batch_ref", "batch_ref requerido")
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	path := store.autoprogrammingBatchPathV0(batchRef)
	var document autoprogrammingBatchDocumentV0
	var found bool
	err := withProcessFileLockV0(ctx, path+".lock", func() error {
		var readErr error
		document, found, readErr = readJSONFileV0[autoprogrammingBatchDocumentV0](path)
		return readErr
	})
	if err != nil {
		return orquestaautoprogramming.AutoprogrammingBatchV0{}, err
	}
	if !found {
		return orquestaautoprogramming.AutoprogrammingBatchV0{}, storeErrorV0("autoprogramming_batch", "batch no encontrado")
	}
	return validateAutoprogrammingBatchDocumentV0(document, batchRef)
}

func (store *StoreV0) CompareAndSwapAutoprogrammingBatchV0(
	ctx context.Context,
	expectedVersion uint64,
	batch orquestaautoprogramming.AutoprogrammingBatchV0,
) (orquestaautoprogramming.AutoprogrammingBatchV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return orquestaautoprogramming.AutoprogrammingBatchV0{}, err
	}
	batch = orquestaautoprogramming.NormalizeAutoprogrammingBatchV0(batch)
	if batch.BatchRef == "" {
		return orquestaautoprogramming.AutoprogrammingBatchV0{}, invalidErrorV0("autoprogramming_batch.batch_ref", "batch_ref requerido")
	}
	if batch.StoreVersion != expectedVersion+1 {
		return orquestaautoprogramming.AutoprogrammingBatchV0{}, invalidErrorV0("autoprogramming_batch.store_version", "candidate.store_version debe ser expected_version + 1")
	}
	validation := orquestaautoprogramming.ValidateAutoprogrammingBatchV0(batch)
	if !validation.Accepted {
		return orquestaautoprogramming.AutoprogrammingBatchV0{}, invalidErrorV0("autoprogramming_batch", "batch invalido")
	}
	batch = validation.Batch

	store.mu.Lock()
	defer store.mu.Unlock()
	path := store.autoprogrammingBatchPathV0(batch.BatchRef)
	var saved orquestaautoprogramming.AutoprogrammingBatchV0
	// The repository lock helper is multiprocess on Linux. Its !linux fallback
	// only leaves StoreV0's per-instance mutex, so cross-instance CAS is not serialized.
	err := withProcessFileLockV0(ctx, path+".lock", func() error {
		document, found, readErr := readJSONFileV0[autoprogrammingBatchDocumentV0](path)
		if readErr != nil {
			return readErr
		}
		if !found {
			if expectedVersion != 0 || !autoprogrammingBatchInitialStateV0(batch) {
				return autoprogrammingBatchCASConflictV0()
			}
			saved = batch
			return store.writeAutoprogrammingBatchV0(path, saved)
		}

		existing, validateErr := validateAutoprogrammingBatchDocumentV0(document, batch.BatchRef)
		if validateErr != nil {
			return validateErr
		}
		if reflect.DeepEqual(existing, batch) {
			saved = existing
			return nil
		}
		if existing.PlanHash != batch.PlanHash || existing.StoreVersion != expectedVersion ||
			!autoprogrammingBatchEvidenceAppendOnlyV0(existing, batch) ||
			!autoprogrammingBatchExactSuccessorV0(existing, batch) {
			return autoprogrammingBatchCASConflictV0()
		}
		saved = batch
		return store.writeAutoprogrammingBatchV0(path, saved)
	})
	if err != nil {
		return orquestaautoprogramming.AutoprogrammingBatchV0{}, err
	}
	return saved, nil
}

func (store *StoreV0) writeAutoprogrammingBatchV0(
	path string,
	batch orquestaautoprogramming.AutoprogrammingBatchV0,
) error {
	return writeJSONAtomicV0(path, autoprogrammingBatchDocumentV0{
		SchemaVersion: autoprogrammingBatchDocumentSchemaV0,
		BatchRef:      batch.BatchRef,
		Batch:         batch,
	})
}

func validateAutoprogrammingBatchDocumentV0(
	document autoprogrammingBatchDocumentV0,
	batchRef string,
) (orquestaautoprogramming.AutoprogrammingBatchV0, error) {
	if document.SchemaVersion != autoprogrammingBatchDocumentSchemaV0 || normalizeRefV0(document.BatchRef) != batchRef {
		return orquestaautoprogramming.AutoprogrammingBatchV0{}, storeErrorV0("autoprogramming_batch", "documento persistido inconsistente")
	}
	validation := orquestaautoprogramming.ValidateAutoprogrammingBatchV0(document.Batch)
	if !validation.Accepted || validation.Batch.BatchRef != batchRef {
		return orquestaautoprogramming.AutoprogrammingBatchV0{}, storeErrorV0("autoprogramming_batch", "batch persistido invalido")
	}
	return validation.Batch, nil
}

func autoprogrammingBatchCASConflictV0() error {
	return storeErrorV0("autoprogramming_batch.cas", "batch divergente o store_version no coincide")
}

func autoprogrammingBatchInitialStateV0(batch orquestaautoprogramming.AutoprogrammingBatchV0) bool {
	result := orquestaautoprogramming.NewAutoprogrammingBatchV0(orquestaautoprogramming.AutoprogrammingBatchPlanV0{
		BatchRef: batch.BatchRef, RequestRef: batch.RequestRef, ProjectRef: batch.ProjectRef,
		BaseRevision: batch.BaseRevision, Members: batch.Members, FrozenTests: batch.FrozenTests,
	})
	return result.Accepted && reflect.DeepEqual(result.Batch, batch)
}

func autoprogrammingBatchEvidenceAppendOnlyV0(
	current orquestaautoprogramming.AutoprogrammingBatchV0,
	next orquestaautoprogramming.AutoprogrammingBatchV0,
) bool {
	return autoprogrammingBatchContainsAllV0(next.TestClaims, current.TestClaims) &&
		autoprogrammingBatchContainsAllV0(next.TestReceipts, current.TestReceipts) &&
		autoprogrammingBatchContainsAllV0(next.ActionReceipts, current.ActionReceipts) &&
		autoprogrammingBatchIntegrationClaimsRetainedV0(current.IntegrationClaims, next.IntegrationClaims) &&
		autoprogrammingBatchPromotionClaimsRetainedV0(current.PromotionClaims, next.PromotionClaims)
}

func autoprogrammingBatchContainsAllV0[T any](values []T, required []T) bool {
	for _, want := range required {
		found := false
		for _, value := range values {
			if reflect.DeepEqual(value, want) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func autoprogrammingBatchIntegrationClaimsRetainedV0(
	current []orquestaautoprogramming.AutoprogrammingBatchIntegrationClaimV0,
	next []orquestaautoprogramming.AutoprogrammingBatchIntegrationClaimV0,
) bool {
	for _, want := range current {
		found := false
		for _, candidate := range next {
			if candidate.GateGeneration == want.GateGeneration && candidate.ClaimRef == want.ClaimRef && candidate.TaskRef == want.TaskRef {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func autoprogrammingBatchPromotionClaimsRetainedV0(
	current []orquestaautoprogramming.AutoprogrammingBatchPromotionClaimV0,
	next []orquestaautoprogramming.AutoprogrammingBatchPromotionClaimV0,
) bool {
	for _, want := range current {
		found := false
		for _, candidate := range next {
			if candidate.GateGeneration == want.GateGeneration && candidate.ClaimRef == want.ClaimRef {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func autoprogrammingBatchExactSuccessorV0(
	current orquestaautoprogramming.AutoprogrammingBatchV0,
	next orquestaautoprogramming.AutoprogrammingBatchV0,
) bool {
	idempotencyKey, ok := autoprogrammingBatchAddedActionKeyV0(current.ActionReceipts, next.ActionReceipts)
	if !ok {
		return false
	}
	matches := func(result orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0) bool {
		return result.Accepted && reflect.DeepEqual(result.Batch, next)
	}
	for _, member := range next.Members {
		if matches(orquestaautoprogramming.RegisterAutoprogrammingBatchLaunchV0(current, current.StoreVersion, idempotencyKey, member.TaskRef)) ||
			matches(orquestaautoprogramming.RegisterAutoprogrammingBatchFocalCloseV0(current, current.StoreVersion, idempotencyKey, member.TaskRef)) ||
			matches(orquestaautoprogramming.RequestAutoprogrammingBatchReworkV0(current, current.StoreVersion, idempotencyKey, member.TaskRef)) {
			return true
		}
	}
	for _, claim := range next.IntegrationClaims {
		if matches(orquestaautoprogramming.ClaimAutoprogrammingBatchIntegrationV0(current, current.StoreVersion, idempotencyKey, claim.ClaimRef, claim.TaskRef, claim.ParentRevision)) ||
			matches(orquestaautoprogramming.RegisterAutoprogrammingBatchIntegrationV0(current, current.StoreVersion, idempotencyKey, claim.ClaimRef, claim.TaskRef, claim.SourceRevision, claim.ParentRevision, claim.IntegrationRevision, claim.ReceiptRef)) {
			return true
		}
	}
	for _, claim := range next.TestClaims {
		if matches(orquestaautoprogramming.ClaimAutoprogrammingBatchTestV0(current, current.StoreVersion, idempotencyKey, claim.Revision, claim.TestHash, claim.ClaimRef)) {
			return true
		}
	}
	for _, receipt := range next.TestReceipts {
		if matches(orquestaautoprogramming.RecordAutoprogrammingBatchTestReceiptV0(current, current.StoreVersion, idempotencyKey, receipt.Revision, receipt.TestHash, receipt.ClaimRef, receipt.ReceiptRef, receipt.Status)) {
			return true
		}
	}
	for _, claim := range next.PromotionClaims {
		if matches(orquestaautoprogramming.ClaimAutoprogrammingBatchPromotionV0(current, current.StoreVersion, idempotencyKey, claim.ClaimRef, claim.Revision)) {
			return true
		}
	}
	return matches(orquestaautoprogramming.RegisterAutoprogrammingBatchPromotionV0(current, current.StoreVersion, idempotencyKey, next.PromotionReceipt.ClaimRef, next.PromotionReceipt.Revision, next.PromotionReceipt.ReceiptRef)) ||
		matches(orquestaautoprogramming.CloseAutoprogrammingBatchV0(current, current.StoreVersion, idempotencyKey)) ||
		matches(orquestaautoprogramming.BlockAutoprogrammingBatchV0(current, current.StoreVersion, idempotencyKey, next.BlockRef))
}

func autoprogrammingBatchAddedActionKeyV0(
	current []orquestaautoprogramming.AutoprogrammingBatchActionReceiptV0,
	next []orquestaautoprogramming.AutoprogrammingBatchActionReceiptV0,
) (string, bool) {
	if len(next) != len(current)+1 {
		return "", false
	}
	currentKeys := make(map[string]bool, len(current))
	for _, receipt := range current {
		currentKeys[receipt.IdempotencyKey] = true
	}
	for _, receipt := range next {
		if !currentKeys[receipt.IdempotencyKey] {
			return receipt.IdempotencyKey, true
		}
	}
	return "", false
}
