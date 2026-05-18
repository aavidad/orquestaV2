package orquestastatefile

import (
	"context"
	"reflect"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type requiredTestEvidenceDocumentV0 struct {
	SchemaVersion string                                       `json:"schema_version"`
	RunRef        string                                       `json:"run_ref"`
	EvidenceRef   string                                       `json:"evidence_ref"`
	Evidence      orquestacionnucleoapp.RequiredTestEvidenceV0 `json:"evidence"`
}

func (store *StoreV0) SaveRequiredTestEvidenceV0(
	ctx context.Context,
	evidence orquestacionnucleoapp.RequiredTestEvidenceV0,
) error {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	normalized, err := orquestacionnucleoapp.NewRequiredTestEvidenceV0(evidence)
	if err != nil {
		return invalidErrorV0("required_test_evidence", err.Error())
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.saveRequiredTestEvidenceLockedV0(normalized)
}

func (store *StoreV0) LoadRequiredTestEvidenceV0(
	ctx context.Context,
	runRef string,
	evidenceRefs []string,
) ([]orquestacionnucleoapp.RequiredTestEvidenceV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	runRef = normalizeRefV0(runRef)
	if runRef == "" {
		return nil, invalidErrorV0("run_ref", "run_ref requerido")
	}
	refs := compactStringsV0(evidenceRefs)
	store.mu.Lock()
	defer store.mu.Unlock()
	out := make([]orquestacionnucleoapp.RequiredTestEvidenceV0, 0, len(refs))
	for _, evidenceRef := range refs {
		document, ok, err := readJSONFileV0[requiredTestEvidenceDocumentV0](store.requiredTestEvidencePathV0(runRef, evidenceRef))
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, storeErrorV0("required_test_evidence", "evidencia no encontrada")
		}
		evidence, err := validateRequiredTestEvidenceDocumentV0(document, runRef, evidenceRef)
		if err != nil {
			return nil, err
		}
		out = append(out, evidence)
	}
	return out, nil
}

func (store *StoreV0) saveRequiredTestEvidenceLockedV0(
	evidence orquestacionnucleoapp.RequiredTestEvidenceV0,
) error {
	runRef := normalizeRefV0(evidence.RunRef)
	evidenceRef := normalizeRefV0(evidence.EvidenceRef)
	path := store.requiredTestEvidencePathV0(runRef, evidenceRef)
	existing, ok, err := readJSONFileV0[requiredTestEvidenceDocumentV0](path)
	if err != nil {
		return err
	}
	if ok {
		existingEvidence, err := validateRequiredTestEvidenceDocumentV0(existing, runRef, evidenceRef)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(existingEvidence, evidence) {
			return storeErrorV0("required_test_evidence", "evidencia de test existente con contrato distinto")
		}
		return nil
	}
	return writeJSONAtomicV0(path, requiredTestEvidenceDocumentV0{
		SchemaVersion: requiredTestEvidenceDocumentSchemaV0,
		RunRef:        runRef,
		EvidenceRef:   evidenceRef,
		Evidence:      evidence,
	})
}

func validateRequiredTestEvidenceDocumentV0(
	document requiredTestEvidenceDocumentV0,
	expectedRunRef string,
	expectedEvidenceRef string,
) (orquestacionnucleoapp.RequiredTestEvidenceV0, error) {
	if document.SchemaVersion != requiredTestEvidenceDocumentSchemaV0 {
		return orquestacionnucleoapp.RequiredTestEvidenceV0{}, storeErrorV0("required_test_evidence.schema_version", "schema_version invalida")
	}
	if document.RunRef != expectedRunRef || document.EvidenceRef != expectedEvidenceRef {
		return orquestacionnucleoapp.RequiredTestEvidenceV0{}, storeErrorV0("required_test_evidence.ref", "ref inconsistente")
	}
	evidence, err := orquestacionnucleoapp.NewRequiredTestEvidenceV0(document.Evidence)
	if err != nil {
		return orquestacionnucleoapp.RequiredTestEvidenceV0{}, err
	}
	if evidence.RunRef != expectedRunRef || evidence.EvidenceRef != expectedEvidenceRef {
		return orquestacionnucleoapp.RequiredTestEvidenceV0{}, storeErrorV0("required_test_evidence.state_ref", "ref interna inconsistente")
	}
	return evidence, nil
}
