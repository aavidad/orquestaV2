package jsonimport

import (
	"context"
	"encoding/json"
	"errors"
	"sort"

	"orquesta/internal/config"
)

// Importer builds deterministic plans and delegates the sole write to
// config.Manager. It stores neither source bytes nor migration plans.
type Importer struct {
	manager *config.Manager
}

// New validates dependencies and the fixed canonical table without reading or
// writing configuration state.
func New(options Options) (*Importer, error) {
	if options.Manager == nil {
		return nil, &Error{Code: ErrorDependenciesRequired}
	}
	return &Importer{manager: options.Manager}, nil
}

// Preview performs a strict, side-effect-free parse and accounts for every
// legacy leaf. A plan is ready only when it contains a mapped change and no
// deferred or secret-required leaves.
func (importer *Importer) Preview(ctx context.Context, source []byte) (Plan, error) {
	if importer == nil || importer.manager == nil {
		return Plan{}, &Error{Code: ErrorDependenciesRequired}
	}
	if err := readyContext(ctx); err != nil {
		return Plan{}, err
	}
	leaves, err := parseLegacyDocument(ctx, source)
	if err != nil {
		return Plan{}, err
	}
	index, _ := canonicalMappingIndex()

	plan := Plan{
		SourceSHA256:  bytesSHA256(source),
		MappingSHA256: canonicalMappingSHA256(),
		Entries:       make([]Entry, 0, len(leaves)),
		changes:       make([]config.Change, 0, len(leaves)),
	}
	for _, leaf := range leaves {
		mapping := index[leaf.path]
		entry := Entry{
			LegacyPath: mapping.LegacyPath, TargetKey: mapping.TargetKey,
			Transform: mapping.Transform, Disposition: mapping.Disposition, Reason: mapping.Reason,
		}
		switch mapping.Disposition {
		case DispositionMapped:
			value, err := applyTransform(mapping, leaf.value)
			if err != nil {
				return Plan{}, err
			}
			entry.Value = value
			plan.changes = append(plan.changes, config.Change{Key: mapping.TargetKey, Value: value})
		case DispositionDeferred:
			plan.UnresolvedPaths = append(plan.UnresolvedPaths, mapping.LegacyPath)
		case DispositionSecretRequired:
			plan.SecretRequiredPaths = append(plan.SecretRequiredPaths, mapping.LegacyPath)
		case DispositionSchema:
		default:
			return Plan{}, &Error{Code: ErrorMappingInvalid, Path: mapping.LegacyPath}
		}
		plan.Entries = append(plan.Entries, entry)
	}
	sort.Slice(plan.changes, func(left, right int) bool {
		return plan.changes[left].Key < plan.changes[right].Key
	})
	if err := validateCanonicalChanges(plan.changes); err != nil {
		return Plan{}, err
	}
	if len(plan.SecretRequiredPaths) != 0 {
		plan.SourceSHA256 = ""
	}
	plan.Ready = len(plan.changes) > 0 && len(plan.UnresolvedPaths) == 0 && len(plan.SecretRequiredPaths) == 0
	plan.PlanSHA256, err = hashPlan(plan)
	if err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func applyTransform(mapping Mapping, value any) (any, error) {
	if mapping.Transform != transformIdentityString {
		return nil, &Error{Code: ErrorMappingInvalid, Path: mapping.LegacyPath}
	}
	text, ok := value.(string)
	if !ok {
		return nil, &Error{Code: ErrorValueInvalid, Path: mapping.LegacyPath}
	}
	return text, nil
}

// validateCanonicalChanges reuses the registry's pure parser and cross-value
// validators. Preview therefore cannot claim readiness for a value that Apply
// would reject, without duplicating server or future key policy here.
func validateCanonicalChanges(changes []config.Change) error {
	if len(changes) == 0 {
		return nil
	}
	values := make(map[config.Key]any, len(changes))
	for _, item := range changes {
		if _, duplicate := values[item.Key]; duplicate {
			return &Error{Code: ErrorMappingInvalid, Path: string(item.Key), Cause: errors.New("legacy_target_duplicate")}
		}
		values[item.Key] = item.Value
	}
	rendered, err := config.RenderExplicit(values)
	if err != nil {
		return &Error{Code: ErrorValueInvalid, Cause: err}
	}
	if _, err := config.Resolve(config.ResolveOptions{TOML: rendered}); err != nil {
		return &Error{Code: ErrorValueInvalid, Cause: err}
	}
	return nil
}

// Apply recomputes and verifies the exact plan, then uses config.Manager for
// expected-revision CAS, durable idempotency and TOML rendering.
func (importer *Importer) Apply(ctx context.Context, request ApplyRequest) (config.UpdateResult, error) {
	if importer == nil || importer.manager == nil {
		return config.UpdateResult{}, &Error{Code: ErrorDependenciesRequired}
	}
	if err := readyContext(ctx); err != nil {
		return config.UpdateResult{}, err
	}
	if !request.Confirm {
		return config.UpdateResult{}, &Error{Code: ErrorConfirmationRequired}
	}
	plan, err := importer.Preview(ctx, request.Source)
	if err != nil {
		return config.UpdateResult{}, err
	}
	if plan.PlanSHA256 == "" || plan.PlanSHA256 != request.ExpectedPlanSHA256 {
		return config.UpdateResult{}, &Error{Code: ErrorPlanMismatch}
	}
	if !plan.Ready {
		return config.UpdateResult{}, &Error{Code: ErrorPlanNotReady}
	}
	return importer.manager.Update(ctx, config.UpdateRequest{
		ActorRef: request.ActorRef, RequestRef: request.RequestRef,
		ExpectedRevision: request.ExpectedRevision, Confirm: true,
		Changes: append([]config.Change(nil), plan.changes...),
	})
}

func hashPlan(plan Plan) (string, error) {
	type planDocument struct {
		SchemaVersion       int             `json:"schema_version"`
		DocumentType        string          `json:"document_type"`
		SourceSHA256        string          `json:"source_sha256"`
		MappingSHA256       string          `json:"mapping_sha256"`
		Ready               bool            `json:"ready"`
		Entries             []Entry         `json:"entries"`
		UnresolvedPaths     []string        `json:"unresolved_paths"`
		SecretRequiredPaths []string        `json:"secret_required_paths"`
		Changes             []config.Change `json:"changes"`
	}
	document := planDocument{
		SchemaVersion: 1, DocumentType: "orquesta.config.legacy-json-plan.v1",
		SourceSHA256: plan.SourceSHA256, MappingSHA256: plan.MappingSHA256, Ready: plan.Ready,
		Entries:             plan.Entries,
		UnresolvedPaths:     append([]string(nil), plan.UnresolvedPaths...),
		SecretRequiredPaths: append([]string(nil), plan.SecretRequiredPaths...),
		Changes:             plan.changes,
	}
	payload, err := json.Marshal(document)
	if err != nil {
		return "", &Error{Code: ErrorMappingInvalid, Cause: err}
	}
	return bytesSHA256(payload), nil
}
