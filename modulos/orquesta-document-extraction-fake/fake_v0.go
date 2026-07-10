package orquestadocumentextractionfake

import (
	"context"
	"fmt"
	"sync"

	orquestadocumentextraction "orquesta/modulos/orquesta-document-extraction"
)

type StaticDocumentSourceV0 struct {
	Identity orquestadocumentextraction.DocumentAdapterIdentityV0
	Material orquestadocumentextraction.DocumentSourceMaterialV0
}

func (fake StaticDocumentSourceV0) AdapterIdentityV0() orquestadocumentextraction.DocumentAdapterIdentityV0 {
	return fake.Identity
}
func (fake StaticDocumentSourceV0) ResolveDocumentV0(_ context.Context, documentRef string) (orquestadocumentextraction.DocumentSourceMaterialV0, error) {
	if fake.Material.DocumentRef != documentRef {
		return orquestadocumentextraction.DocumentSourceMaterialV0{}, fmt.Errorf("document_not_found")
	}
	return fake.Material, nil
}

type PassthroughDocumentNormalizerV0 struct {
	Identity orquestadocumentextraction.DocumentAdapterIdentityV0
}

func (fake PassthroughDocumentNormalizerV0) AdapterIdentityV0() orquestadocumentextraction.DocumentAdapterIdentityV0 {
	return fake.Identity
}
func (fake PassthroughDocumentNormalizerV0) NormalizeDocumentV0(_ context.Context, source orquestadocumentextraction.DocumentSourceMaterialV0, _ orquestadocumentextraction.DocumentExtractionPolicyV0) (orquestadocumentextraction.DocumentNormalizedMaterialV0, error) {
	return orquestadocumentextraction.DocumentNormalizedMaterialV0{Source: source}, nil
}

type StaticDocumentParserV0 struct {
	Identity orquestadocumentextraction.DocumentAdapterIdentityV0
	Document orquestadocumentextraction.DocumentV0
}

func (fake StaticDocumentParserV0) AdapterIdentityV0() orquestadocumentextraction.DocumentAdapterIdentityV0 {
	return fake.Identity
}
func (fake StaticDocumentParserV0) ParseDocumentV0(_ context.Context, _ orquestadocumentextraction.DocumentNormalizedMaterialV0) (orquestadocumentextraction.DocumentV0, error) {
	return fake.Document, nil
}

type StaticDocumentLocalizerV0 struct {
	Identity  orquestadocumentextraction.DocumentAdapterIdentityV0
	Locations []orquestadocumentextraction.DocumentLocationV0
}

func (fake StaticDocumentLocalizerV0) AdapterIdentityV0() orquestadocumentextraction.DocumentAdapterIdentityV0 {
	return fake.Identity
}
func (fake StaticDocumentLocalizerV0) LocateDocumentSchemaV0(_ context.Context, _ orquestadocumentextraction.DocumentV0, _ orquestadocumentextraction.DocumentSchemaV0) ([]orquestadocumentextraction.DocumentLocationV0, error) {
	return append([]orquestadocumentextraction.DocumentLocationV0(nil), fake.Locations...), nil
}

type StaticDocumentSchemaExtractorV0 struct {
	Identity          orquestadocumentextraction.DocumentAdapterIdentityV0
	CandidatesBySlice map[string][]orquestadocumentextraction.DocumentFieldCandidateV0
}

func (fake StaticDocumentSchemaExtractorV0) AdapterIdentityV0() orquestadocumentextraction.DocumentAdapterIdentityV0 {
	return fake.Identity
}
func (fake StaticDocumentSchemaExtractorV0) ExtractDocumentSchemaSliceV0(_ context.Context, _ orquestadocumentextraction.DocumentV0, slice orquestadocumentextraction.DocumentSchemaSliceV0) ([]orquestadocumentextraction.DocumentFieldCandidateV0, error) {
	return append([]orquestadocumentextraction.DocumentFieldCandidateV0(nil), fake.CandidatesBySlice[slice.SliceRef]...), nil
}

type StaticDocumentEvidenceLocatorV0 struct {
	Identity            orquestadocumentextraction.DocumentAdapterIdentityV0
	EvidenceByCandidate map[string][]orquestadocumentextraction.DocumentEvidenceV0
}

func (fake StaticDocumentEvidenceLocatorV0) AdapterIdentityV0() orquestadocumentextraction.DocumentAdapterIdentityV0 {
	return fake.Identity
}
func (fake StaticDocumentEvidenceLocatorV0) LocateDocumentEvidenceV0(_ context.Context, _ orquestadocumentextraction.DocumentV0, candidate orquestadocumentextraction.DocumentFieldCandidateV0) ([]orquestadocumentextraction.DocumentEvidenceV0, error) {
	return append([]orquestadocumentextraction.DocumentEvidenceV0(nil), fake.EvidenceByCandidate[candidate.CandidateRef]...), nil
}

type AcceptingDocumentValidatorV0 struct {
	Identity       orquestadocumentextraction.DocumentAdapterIdentityV0
	RequiresReview bool
}

func (fake AcceptingDocumentValidatorV0) AdapterIdentityV0() orquestadocumentextraction.DocumentAdapterIdentityV0 {
	return fake.Identity
}
func (fake AcceptingDocumentValidatorV0) ValidateDocumentFieldV0(_ context.Context, _ orquestadocumentextraction.DocumentSchemaFieldV0, candidate orquestadocumentextraction.DocumentFieldCandidateV0, _ string) (orquestadocumentextraction.DocumentValidationResultV0, error) {
	return orquestadocumentextraction.DocumentValidationResultV0{Candidate: candidate, Accepted: true, RequiresHumanReview: fake.RequiresReview, ValidationRefs: []string{"fake:validation"}}, nil
}

type AcceptingDocumentHumanReviewV0 struct {
	Identity orquestadocumentextraction.DocumentAdapterIdentityV0
}

func (fake AcceptingDocumentHumanReviewV0) AdapterIdentityV0() orquestadocumentextraction.DocumentAdapterIdentityV0 {
	return fake.Identity
}
func (fake AcceptingDocumentHumanReviewV0) ReviewDocumentFieldV0(_ context.Context, _ orquestadocumentextraction.DocumentSchemaFieldV0, candidate orquestadocumentextraction.DocumentFieldCandidateV0) (orquestadocumentextraction.DocumentHumanReviewResultV0, error) {
	return orquestadocumentextraction.DocumentHumanReviewResultV0{Candidate: candidate, Accepted: true, DecisionRef: "fake:review"}, nil
}

type RecordingDocumentExporterV0 struct {
	Identity orquestadocumentextraction.DocumentAdapterIdentityV0
	mu       sync.Mutex
	Requests []orquestadocumentextraction.DocumentExportRequestV0
	Artifact orquestadocumentextraction.DocumentExportArtifactV0
}

func (fake *RecordingDocumentExporterV0) AdapterIdentityV0() orquestadocumentextraction.DocumentAdapterIdentityV0 {
	return fake.Identity
}
func (fake *RecordingDocumentExporterV0) ExportDocumentFieldsV0(_ context.Context, request orquestadocumentextraction.DocumentExportRequestV0) (orquestadocumentextraction.DocumentExportArtifactV0, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.Requests = append(fake.Requests, request)
	return fake.Artifact, nil
}
func (fake *RecordingDocumentExporterV0) CallCountV0() int {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	return len(fake.Requests)
}

type InMemoryDocumentReceiptStoreV0 struct {
	Identity orquestadocumentextraction.DocumentAdapterIdentityV0
	mu       sync.Mutex
	Receipts []orquestadocumentextraction.DocumentExtractionReceiptV0
}

func (fake *InMemoryDocumentReceiptStoreV0) AdapterIdentityV0() orquestadocumentextraction.DocumentAdapterIdentityV0 {
	return fake.Identity
}
func (fake *InMemoryDocumentReceiptStoreV0) StoreDocumentExtractionReceiptV0(_ context.Context, receipt orquestadocumentextraction.DocumentExtractionReceiptV0) (string, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	receipt.ReceiptRef = fmt.Sprintf("receipt:%d", len(fake.Receipts)+1)
	fake.Receipts = append(fake.Receipts, receipt)
	return receipt.ReceiptRef, nil
}
func (fake *InMemoryDocumentReceiptStoreV0) LastReceiptV0() (orquestadocumentextraction.DocumentExtractionReceiptV0, bool) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.Receipts) == 0 {
		return orquestadocumentextraction.DocumentExtractionReceiptV0{}, false
	}
	return fake.Receipts[len(fake.Receipts)-1], true
}

type InMemoryDocumentToolOperationStoreV0 struct {
	Identity    orquestadocumentextraction.DocumentAdapterIdentityV0
	mu          sync.Mutex
	Operations  map[string]orquestadocumentextraction.DocumentToolOperationV0
	Idempotency map[string]string
}

func NewInMemoryDocumentToolOperationStoreV0(identity orquestadocumentextraction.DocumentAdapterIdentityV0) *InMemoryDocumentToolOperationStoreV0 {
	return &InMemoryDocumentToolOperationStoreV0{Identity: identity, Operations: map[string]orquestadocumentextraction.DocumentToolOperationV0{}, Idempotency: map[string]string{}}
}
func (fake *InMemoryDocumentToolOperationStoreV0) AdapterIdentityV0() orquestadocumentextraction.DocumentAdapterIdentityV0 {
	return fake.Identity
}
func (fake *InMemoryDocumentToolOperationStoreV0) SubmitDocumentToolCommandV0(_ context.Context, command orquestadocumentextraction.DocumentToolCommandV0) (orquestadocumentextraction.DocumentToolOperationV0, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if operationRef, ok := fake.Idempotency[command.IdempotencyKey]; ok {
		return fake.Operations[operationRef], nil
	}
	operationRef := "operation:" + command.CommandRef
	operation := orquestadocumentextraction.DocumentToolOperationV0{OperationRef: operationRef, CommandRef: command.CommandRef, Surface: command.Surface, IdempotencyKey: command.IdempotencyKey, State: orquestadocumentextraction.DocumentToolOperationQueuedV0}
	fake.Operations[operationRef] = operation
	fake.Idempotency[command.IdempotencyKey] = operationRef
	return operation, nil
}
func (fake *InMemoryDocumentToolOperationStoreV0) LoadDocumentToolOperationV0(_ context.Context, operationRef string) (orquestadocumentextraction.DocumentToolOperationV0, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	operation, ok := fake.Operations[operationRef]
	if !ok {
		return orquestadocumentextraction.DocumentToolOperationV0{}, fmt.Errorf("document_operation_not_found")
	}
	return operation, nil
}

type StaticDocumentToolCapabilityRegistryV0 struct {
	Identity     orquestadocumentextraction.DocumentAdapterIdentityV0
	Capabilities []orquestadocumentextraction.DocumentToolCapabilityDescriptorV0
}

func (fake StaticDocumentToolCapabilityRegistryV0) AdapterIdentityV0() orquestadocumentextraction.DocumentAdapterIdentityV0 {
	return fake.Identity
}
func (fake StaticDocumentToolCapabilityRegistryV0) DiscoverDocumentToolCapabilitiesV0(_ context.Context, _ orquestadocumentextraction.DocumentToolDiscoveryRequestV0) ([]orquestadocumentextraction.DocumentToolCapabilityDescriptorV0, error) {
	return append([]orquestadocumentextraction.DocumentToolCapabilityDescriptorV0(nil), fake.Capabilities...), nil
}
