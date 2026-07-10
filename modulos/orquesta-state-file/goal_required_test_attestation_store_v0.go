package orquestastatefile

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

type goalRequiredTestAttestationDocumentV0 struct {
	SchemaVersion  string                                     `json:"schema_version"`
	RunRef         string                                     `json:"run_ref"`
	AttestationRef string                                     `json:"attestation_ref"`
	Attestation    orquestagoal.GoalRequiredTestAttestationV0 `json:"attestation"`
}

func (store *StoreV0) SaveGoalRequiredTestAttestationV0(ctx context.Context, attestation orquestagoal.GoalRequiredTestAttestationV0) error {
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
	path := store.goalRequiredTestAttestationPathV0(attestation.RunRef, attestation.AttestationRef)
	existing, ok, err := readJSONFileV0[goalRequiredTestAttestationDocumentV0](path)
	if err != nil {
		return err
	}
	if ok {
		current, err := validateGoalRequiredTestAttestationDocumentV0(existing, attestation.RunRef, attestation.AttestationRef)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(current, attestation) {
			return storeErrorV0("goal_required_test_attestation", "attestation existente con contrato distinto")
		}
		return nil
	}
	return writeJSONAtomicV0(path, goalRequiredTestAttestationDocumentV0{
		SchemaVersion:  orquestagoal.GoalRequiredTestAttestationSchemaV0,
		RunRef:         attestation.RunRef,
		AttestationRef: attestation.AttestationRef,
		Attestation:    attestation,
	})
}

func (store *StoreV0) ListGoalRequiredTestAttestationsV0(ctx context.Context, query orquestagoal.GoalRequiredTestAttestationQueryV0) ([]orquestagoal.GoalRequiredTestAttestationV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if query.RunRef == "" || query.GoalRef == "" || query.RevisionRef == "" {
		return nil, invalidErrorV0("goal_required_test_attestation.query", "run_ref, goal_ref y revision_ref requeridos")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	dir := filepath.Join(store.rootDir, goalRequiredTestAttestationsDirV0, hashRefsV0(query.RunRef))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := []orquestagoal.GoalRequiredTestAttestationV0{}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		document, ok, err := readJSONFileV0[goalRequiredTestAttestationDocumentV0](filepath.Join(dir, entry.Name()))
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
		if attestation.GoalRef == query.GoalRef && attestation.RevisionRef == query.RevisionRef {
			out = append(out, attestation)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AttestationRef < out[j].AttestationRef })
	return out, nil
}

func validateGoalRequiredTestAttestationDocumentV0(document goalRequiredTestAttestationDocumentV0, runRef, attestationRef string) (orquestagoal.GoalRequiredTestAttestationV0, error) {
	if document.SchemaVersion != orquestagoal.GoalRequiredTestAttestationSchemaV0 || document.RunRef != runRef || document.AttestationRef != attestationRef {
		return orquestagoal.GoalRequiredTestAttestationV0{}, storeErrorV0("goal_required_test_attestation", "documento inconsistente")
	}
	attestation := orquestagoal.NormalizeGoalRequiredTestAttestationV0(document.Attestation)
	if issues := orquestagoal.ValidateGoalRequiredTestAttestationV0(attestation); len(issues) > 0 || attestation.RunRef != runRef || attestation.AttestationRef != attestationRef {
		return orquestagoal.GoalRequiredTestAttestationV0{}, storeErrorV0("goal_required_test_attestation", "attestation interna invalida")
	}
	return attestation, nil
}
