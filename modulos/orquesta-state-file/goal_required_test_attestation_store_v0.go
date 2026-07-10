package orquestastatefile

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	goalRequiredTestFinalSnapshotDocumentSchemaV0    = "orquesta_state_file.goal_required_test_final_snapshot.v0"
	goalRequiredTestAttestationClaimDocumentSchemaV0 = "orquesta_state_file.goal_required_test_attestation_claim.v0"
)

type goalRequiredTestAttestationDocumentV0 struct {
	SchemaVersion  string                                     `json:"schema_version"`
	RunRef         string                                     `json:"run_ref"`
	AttestationRef string                                     `json:"attestation_ref"`
	Attestation    orquestagoal.GoalRequiredTestAttestationV0 `json:"attestation"`
}

type goalRequiredTestFinalSnapshotDocumentV0 struct {
	SchemaVersion string                                       `json:"schema_version"`
	RunRef        string                                       `json:"run_ref"`
	GoalRef       string                                       `json:"goal_ref"`
	Snapshot      orquestagoal.GoalRequiredTestFinalSnapshotV0 `json:"snapshot"`
}

type goalRequiredTestAttestationClaimDocumentV0 struct {
	SchemaVersion string                                          `json:"schema_version"`
	RunRef        string                                          `json:"run_ref"`
	ClaimRef      string                                          `json:"claim_ref"`
	Claim         orquestagoal.GoalRequiredTestAttestationClaimV0 `json:"claim"`
}

func (store *StoreV0) SaveGoalRequiredTestAttestationV0(
	ctx context.Context,
	attestation orquestagoal.GoalRequiredTestAttestationV0,
) error {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	attestation = orquestagoal.NormalizeGoalRequiredTestAttestationV0(attestation)
	if issues := orquestagoal.ValidateGoalRequiredTestAttestationV0(attestation); len(issues) > 0 {
		return invalidErrorV0("goal_required_test_attestation", "attestation invalida")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return withProcessFileLockV0(ctx, store.goalRequiredTestAttestationRunLockPathV0(attestation.RunRef), func() error {
		return store.saveGoalRequiredTestAttestationLockedV0(attestation)
	})
}

func (store *StoreV0) saveGoalRequiredTestAttestationLockedV0(
	attestation orquestagoal.GoalRequiredTestAttestationV0,
) error {
	path := store.goalRequiredTestAttestationCanonicalPathV0(
		attestation.RunRef, attestation.GoalRef, attestation.RevisionRef, attestation.TestRef,
	)
	existing, ok, err := readJSONFileV0[goalRequiredTestAttestationDocumentV0](path)
	if err != nil {
		return err
	}
	if ok {
		current, err := validateGoalRequiredTestAttestationDocumentV0(existing, attestation.RunRef, existing.AttestationRef)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(current, attestation) {
			return storeErrorV0("goal_required_test_attestation", "receipt canonico existente con contrato distinto")
		}
		return nil
	}
	return writeJSONAtomicV0(path, goalRequiredTestAttestationDocumentV0{
		SchemaVersion: orquestagoal.GoalRequiredTestAttestationSchemaV0,
		RunRef:        attestation.RunRef, AttestationRef: attestation.AttestationRef, Attestation: attestation,
	})
}

func (store *StoreV0) ListGoalRequiredTestAttestationsV0(
	ctx context.Context,
	query orquestagoal.GoalRequiredTestAttestationQueryV0,
) ([]orquestagoal.GoalRequiredTestAttestationV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	query.RunRef = strings.TrimSpace(query.RunRef)
	query.GoalRef = strings.TrimSpace(query.GoalRef)
	query.RevisionRef = strings.TrimSpace(query.RevisionRef)
	if query.RunRef == "" || query.GoalRef == "" {
		return nil, invalidErrorV0("goal_required_test_attestation.query", "run_ref y goal_ref requeridos")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	out := []orquestagoal.GoalRequiredTestAttestationV0{}
	err := withProcessFileLockV0(ctx, store.goalRequiredTestAttestationRunLockPathV0(query.RunRef), func() error {
		var err error
		out, err = store.listGoalRequiredTestAttestationsLockedV0(ctx, query)
		return err
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AttestationRef < out[j].AttestationRef })
	return out, nil
}

func (store *StoreV0) listGoalRequiredTestAttestationsLockedV0(
	ctx context.Context,
	query orquestagoal.GoalRequiredTestAttestationQueryV0,
) ([]orquestagoal.GoalRequiredTestAttestationV0, error) {
	dirs := []string{
		store.goalRequiredTestAttestationReceiptsDirV0(query.RunRef),
		filepath.Join(store.rootDir, goalRequiredTestAttestationsDirV0, hashRefsV0(query.RunRef)),
	}
	out := []orquestagoal.GoalRequiredTestAttestationV0{}
	seenPaths := map[string]bool{}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			path := filepath.Join(dir, entry.Name())
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" || seenPaths[path] {
				continue
			}
			seenPaths[path] = true
			document, ok, err := readJSONFileV0[goalRequiredTestAttestationDocumentV0](path)
			if err != nil || !ok {
				if err != nil {
					return nil, err
				}
				continue
			}
			attestation, err := validateGoalRequiredTestAttestationDocumentV0(document, query.RunRef, document.AttestationRef)
			if err != nil {
				return nil, err
			}
			if attestation.GoalRef == query.GoalRef && (query.RevisionRef == "" || attestation.RevisionRef == query.RevisionRef) {
				out = append(out, attestation)
			}
		}
	}
	return out, nil
}

func (store *StoreV0) FreezeGoalRequiredTestFinalSnapshotV0(
	ctx context.Context,
	snapshot orquestagoal.GoalRequiredTestFinalSnapshotV0,
) (orquestagoal.GoalRequiredTestFinalSnapshotV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	snapshot = orquestagoal.NormalizeGoalRequiredTestFinalSnapshotV0(snapshot)
	if issues := orquestagoal.ValidateGoalRequiredTestFinalSnapshotV0(snapshot); len(issues) > 0 {
		return orquestagoal.GoalRequiredTestFinalSnapshotV0{}, invalidErrorV0("goal_required_test_final_snapshot", "snapshot invalido")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	path := store.goalRequiredTestFinalSnapshotPathV0(snapshot.RunRef, snapshot.GoalRef)
	err := withProcessFileLockV0(ctx, store.goalRequiredTestAttestationRunLockPathV0(snapshot.RunRef), func() error {
		document, ok, err := readJSONFileV0[goalRequiredTestFinalSnapshotDocumentV0](path)
		if err != nil {
			return err
		}
		if ok {
			current, err := validateGoalRequiredTestFinalSnapshotDocumentV0(document, snapshot.RunRef, snapshot.GoalRef)
			if err != nil {
				return err
			}
			if !orquestagoal.GoalRequiredTestFinalSnapshotIdentityEqualV0(current, snapshot) {
				return storeErrorV0("goal_required_test_final_snapshot", "snapshot final inmutable ya congelado con otra foto")
			}
			snapshot = current
			return nil
		}
		return writeJSONAtomicV0(path, goalRequiredTestFinalSnapshotDocumentV0{
			SchemaVersion: goalRequiredTestFinalSnapshotDocumentSchemaV0,
			RunRef:        snapshot.RunRef, GoalRef: snapshot.GoalRef, Snapshot: snapshot,
		})
	})
	return snapshot, err
}

func (store *StoreV0) LoadGoalRequiredTestFinalSnapshotV0(
	ctx context.Context,
	runRef string,
	goalRef string,
) (orquestagoal.GoalRequiredTestFinalSnapshotV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	runRef = strings.TrimSpace(runRef)
	goalRef = strings.TrimSpace(goalRef)
	if runRef == "" || goalRef == "" {
		return orquestagoal.GoalRequiredTestFinalSnapshotV0{}, invalidErrorV0("goal_required_test_final_snapshot", "run_ref y goal_ref requeridos")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	var snapshot orquestagoal.GoalRequiredTestFinalSnapshotV0
	err := withProcessFileLockV0(ctx, store.goalRequiredTestAttestationRunLockPathV0(runRef), func() error {
		document, ok, err := readJSONFileV0[goalRequiredTestFinalSnapshotDocumentV0](store.goalRequiredTestFinalSnapshotPathV0(runRef, goalRef))
		if err != nil {
			return err
		}
		if !ok {
			return storeErrorV0("goal_required_test_final_snapshot", "snapshot final no encontrado")
		}
		snapshot, err = validateGoalRequiredTestFinalSnapshotDocumentV0(document, runRef, goalRef)
		return err
	})
	return snapshot, err
}

func (store *StoreV0) AcquireGoalRequiredTestAttestationClaimV0(
	ctx context.Context,
	request orquestagoal.GoalRequiredTestAttestationClaimRequestV0,
) (orquestagoal.GoalRequiredTestAttestationClaimResultV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.GoalRef = strings.TrimSpace(request.GoalRef)
	request.RevisionRef = strings.TrimSpace(request.RevisionRef)
	request.TestRef = strings.TrimSpace(request.TestRef)
	request.DefinitionSHA256 = strings.ToLower(strings.TrimSpace(request.DefinitionSHA256))
	claim := orquestagoal.NormalizeGoalRequiredTestAttestationClaimV0(orquestagoal.GoalRequiredTestAttestationClaimV0{
		ClaimRef: orquestagoal.GoalRequiredTestAttestationClaimRefV0(request),
		RunRef:   request.RunRef, GoalRef: request.GoalRef, RevisionRef: request.RevisionRef,
		TestRef: request.TestRef, DefinitionSHA256: request.DefinitionSHA256,
		Status:    orquestagoal.GoalRequiredTestAttestationClaimStatusPendingV0,
		ClaimedAt: time.Now().UTC().Format(time.RFC3339Nano),
	})
	if issues := orquestagoal.ValidateGoalRequiredTestAttestationClaimV0(claim); len(issues) > 0 {
		return orquestagoal.GoalRequiredTestAttestationClaimResultV0{}, invalidErrorV0("goal_required_test_attestation_claim", "claim invalido")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	result := orquestagoal.GoalRequiredTestAttestationClaimResultV0{}
	path := store.goalRequiredTestAttestationClaimPathV0(request.RunRef, request.GoalRef, request.RevisionRef, request.TestRef)
	err := withProcessFileLockV0(ctx, store.goalRequiredTestAttestationRunLockPathV0(request.RunRef), func() error {
		document, ok, err := readJSONFileV0[goalRequiredTestAttestationClaimDocumentV0](path)
		if err != nil {
			return err
		}
		if ok {
			current, err := validateGoalRequiredTestAttestationClaimDocumentV0(document, request.RunRef)
			if err != nil {
				return err
			}
			if current.ClaimRef != claim.ClaimRef || current.DefinitionSHA256 != claim.DefinitionSHA256 {
				return storeErrorV0("goal_required_test_attestation_claim", "claim canonico contradice definicion congelada")
			}
			result.Claim = current
			return nil
		}
		if err := writeJSONAtomicV0(path, goalRequiredTestAttestationClaimDocumentV0{
			SchemaVersion: goalRequiredTestAttestationClaimDocumentSchemaV0,
			RunRef:        request.RunRef, ClaimRef: claim.ClaimRef, Claim: claim,
		}); err != nil {
			return err
		}
		result = orquestagoal.GoalRequiredTestAttestationClaimResultV0{Claim: claim, Acquired: true}
		return nil
	})
	return result, err
}

func (store *StoreV0) CompleteGoalRequiredTestAttestationClaimV0(
	ctx context.Context,
	claim orquestagoal.GoalRequiredTestAttestationClaimV0,
	attestation orquestagoal.GoalRequiredTestAttestationV0,
) error {
	ctx = contextOrBackgroundV0(ctx)
	claim = orquestagoal.NormalizeGoalRequiredTestAttestationClaimV0(claim)
	attestation = orquestagoal.NormalizeGoalRequiredTestAttestationV0(attestation)
	if issues := orquestagoal.ValidateGoalRequiredTestAttestationClaimV0(claim); len(issues) > 0 ||
		len(orquestagoal.ValidateGoalRequiredTestAttestationV0(attestation)) > 0 {
		return invalidErrorV0("goal_required_test_attestation_claim", "claim o receipt invalido")
	}
	if claim.RunRef != attestation.RunRef || claim.GoalRef != attestation.GoalRef || claim.RevisionRef != attestation.RevisionRef ||
		claim.TestRef != attestation.TestRef || claim.DefinitionSHA256 != attestation.DefinitionSHA256 {
		return invalidErrorV0("goal_required_test_attestation_claim", "receipt no corresponde al claim")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return withProcessFileLockV0(ctx, store.goalRequiredTestAttestationRunLockPathV0(claim.RunRef), func() error {
		path := store.goalRequiredTestAttestationClaimPathV0(claim.RunRef, claim.GoalRef, claim.RevisionRef, claim.TestRef)
		document, ok, err := readJSONFileV0[goalRequiredTestAttestationClaimDocumentV0](path)
		if err != nil {
			return err
		}
		if !ok {
			return storeErrorV0("goal_required_test_attestation_claim", "claim durable no encontrado")
		}
		current, err := validateGoalRequiredTestAttestationClaimDocumentV0(document, claim.RunRef)
		if err != nil {
			return err
		}
		if current.ClaimRef != claim.ClaimRef {
			return storeErrorV0("goal_required_test_attestation_claim", "claim durable distinto")
		}
		if current.Status == orquestagoal.GoalRequiredTestAttestationClaimStatusCompletedV0 {
			if current.AttestationRef != attestation.AttestationRef {
				return storeErrorV0("goal_required_test_attestation_claim", "claim completado con receipt contradictorio")
			}
			return store.saveGoalRequiredTestAttestationLockedV0(attestation)
		}
		if err := store.saveGoalRequiredTestAttestationLockedV0(attestation); err != nil {
			return err
		}
		current.Status = orquestagoal.GoalRequiredTestAttestationClaimStatusCompletedV0
		current.AttestationRef = attestation.AttestationRef
		current.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
		return writeJSONAtomicV0(path, goalRequiredTestAttestationClaimDocumentV0{
			SchemaVersion: goalRequiredTestAttestationClaimDocumentSchemaV0,
			RunRef:        current.RunRef, ClaimRef: current.ClaimRef, Claim: current,
		})
	})
}

func (store *StoreV0) FailGoalRequiredTestAttestationClaimV0(
	ctx context.Context,
	claim orquestagoal.GoalRequiredTestAttestationClaimV0,
	failureCode string,
) (orquestagoal.GoalRequiredTestAttestationClaimV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	claim = orquestagoal.NormalizeGoalRequiredTestAttestationClaimV0(claim)
	failureCode = strings.TrimSpace(failureCode)
	if issues := orquestagoal.ValidateGoalRequiredTestAttestationClaimV0(claim); len(issues) > 0 || failureCode == "" {
		return orquestagoal.GoalRequiredTestAttestationClaimV0{}, invalidErrorV0("goal_required_test_attestation_claim", "claim o fallo invalido")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	var failed orquestagoal.GoalRequiredTestAttestationClaimV0
	err := withProcessFileLockV0(ctx, store.goalRequiredTestAttestationRunLockPathV0(claim.RunRef), func() error {
		path := store.goalRequiredTestAttestationClaimPathV0(claim.RunRef, claim.GoalRef, claim.RevisionRef, claim.TestRef)
		document, ok, err := readJSONFileV0[goalRequiredTestAttestationClaimDocumentV0](path)
		if err != nil {
			return err
		}
		if !ok {
			return storeErrorV0("goal_required_test_attestation_claim", "claim durable no encontrado")
		}
		current, err := validateGoalRequiredTestAttestationClaimDocumentV0(document, claim.RunRef)
		if err != nil {
			return err
		}
		if current.ClaimRef != claim.ClaimRef {
			return storeErrorV0("goal_required_test_attestation_claim", "claim durable distinto")
		}
		if current.Status == orquestagoal.GoalRequiredTestAttestationClaimStatusCompletedV0 {
			return storeErrorV0("goal_required_test_attestation_claim", "claim ya completado")
		}
		if current.Status == orquestagoal.GoalRequiredTestAttestationClaimStatusFailedV0 {
			failed = current
			return nil
		}
		current.Status = orquestagoal.GoalRequiredTestAttestationClaimStatusFailedV0
		current.FailureCode = failureCode
		current.FailedAt = time.Now().UTC().Format(time.RFC3339Nano)
		if err := writeJSONAtomicV0(path, goalRequiredTestAttestationClaimDocumentV0{SchemaVersion: goalRequiredTestAttestationClaimDocumentSchemaV0, RunRef: current.RunRef, ClaimRef: current.ClaimRef, Claim: current}); err != nil {
			return err
		}
		failed = current
		return nil
	})
	return failed, err
}

func (store *StoreV0) goalRequiredTestAttestationRunLockPathV0(runRef string) string {
	return filepath.Join(store.rootDir, goalRequiredTestAttestationsDirV0, hashRefsV0(runRef), ".run.lock")
}

func (store *StoreV0) goalRequiredTestAttestationReceiptsDirV0(runRef string) string {
	return filepath.Join(store.rootDir, goalRequiredTestAttestationsDirV0, hashRefsV0(runRef), "receipts")
}

func (store *StoreV0) goalRequiredTestAttestationCanonicalPathV0(runRef, goalRef, revisionRef, testRef string) string {
	return filepath.Join(store.goalRequiredTestAttestationReceiptsDirV0(runRef), hashRefsV0(goalRef, revisionRef, testRef)+".json")
}

func (store *StoreV0) goalRequiredTestFinalSnapshotPathV0(runRef, goalRef string) string {
	return filepath.Join(store.rootDir, goalRequiredTestAttestationsDirV0, hashRefsV0(runRef), "snapshots", hashRefsV0(goalRef)+".json")
}

func (store *StoreV0) goalRequiredTestAttestationClaimPathV0(runRef, goalRef, revisionRef, testRef string) string {
	return filepath.Join(store.rootDir, goalRequiredTestAttestationsDirV0, hashRefsV0(runRef), "claims", hashRefsV0(goalRef, revisionRef, testRef)+".json")
}

func validateGoalRequiredTestAttestationDocumentV0(
	document goalRequiredTestAttestationDocumentV0,
	runRef string,
	attestationRef string,
) (orquestagoal.GoalRequiredTestAttestationV0, error) {
	if document.SchemaVersion != orquestagoal.GoalRequiredTestAttestationSchemaV0 || document.RunRef != runRef || document.AttestationRef != attestationRef {
		return orquestagoal.GoalRequiredTestAttestationV0{}, storeErrorV0("goal_required_test_attestation", "documento inconsistente")
	}
	attestation := orquestagoal.NormalizeGoalRequiredTestAttestationV0(document.Attestation)
	if issues := orquestagoal.ValidateGoalRequiredTestAttestationV0(attestation); len(issues) > 0 || attestation.RunRef != runRef || attestation.AttestationRef != attestationRef {
		return orquestagoal.GoalRequiredTestAttestationV0{}, storeErrorV0("goal_required_test_attestation", "attestation interna invalida")
	}
	return attestation, nil
}

func validateGoalRequiredTestFinalSnapshotDocumentV0(
	document goalRequiredTestFinalSnapshotDocumentV0,
	runRef string,
	goalRef string,
) (orquestagoal.GoalRequiredTestFinalSnapshotV0, error) {
	if document.SchemaVersion != goalRequiredTestFinalSnapshotDocumentSchemaV0 || document.RunRef != runRef || document.GoalRef != goalRef {
		return orquestagoal.GoalRequiredTestFinalSnapshotV0{}, storeErrorV0("goal_required_test_final_snapshot", "documento inconsistente")
	}
	snapshot := orquestagoal.NormalizeGoalRequiredTestFinalSnapshotV0(document.Snapshot)
	if issues := orquestagoal.ValidateGoalRequiredTestFinalSnapshotV0(snapshot); len(issues) > 0 || snapshot.RunRef != runRef || snapshot.GoalRef != goalRef {
		return orquestagoal.GoalRequiredTestFinalSnapshotV0{}, storeErrorV0("goal_required_test_final_snapshot", "snapshot interno invalido")
	}
	return snapshot, nil
}

func validateGoalRequiredTestAttestationClaimDocumentV0(
	document goalRequiredTestAttestationClaimDocumentV0,
	runRef string,
) (orquestagoal.GoalRequiredTestAttestationClaimV0, error) {
	if document.SchemaVersion != goalRequiredTestAttestationClaimDocumentSchemaV0 || document.RunRef != runRef {
		return orquestagoal.GoalRequiredTestAttestationClaimV0{}, storeErrorV0("goal_required_test_attestation_claim", "documento inconsistente")
	}
	claim := orquestagoal.NormalizeGoalRequiredTestAttestationClaimV0(document.Claim)
	if issues := orquestagoal.ValidateGoalRequiredTestAttestationClaimV0(claim); len(issues) > 0 || claim.RunRef != runRef || claim.ClaimRef != document.ClaimRef {
		return orquestagoal.GoalRequiredTestAttestationClaimV0{}, storeErrorV0("goal_required_test_attestation_claim", "claim interno invalido")
	}
	return claim, nil
}
