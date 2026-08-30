package tooling

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"unicode/utf8"
)

const (
	BootstrapCapability = "TLS-14"

	BootstrapKindTool     BootstrapSubjectKind = "tool"
	BootstrapKindSkill    BootstrapSubjectKind = "skill"
	BootstrapKindPlugin   BootstrapSubjectKind = "plugin"
	BootstrapKindRulePack BootstrapSubjectKind = "rulepack"

	BootstrapSurfaceDisabled = "disabled_default"
	BootstrapBrokerRequested = "requested_opt_in"
	BootstrapProjectionScope = "caller_supplied_projection"

	DoctorProjectionMatch DoctorStatus = "projection_match"
	DoctorProjectionDrift DoctorStatus = "projection_drift"

	ErrorBootstrapInvalid = "tooling.bootstrap_invalid"
)

type BootstrapSubjectKind string
type DoctorStatus string

// BootstrapSubject is desired disposition only. It contains no location,
// executable, endpoint, credential, activation flag, or installed-state claim.
type BootstrapSubject struct {
	Kind    BootstrapSubjectKind `json:"kind"`
	ID      string               `json:"id"`
	Version string               `json:"version"`
	Digest  string               `json:"digest"`
}

// BootstrapSelectionContext binds the desired manifest to the exact curated
// visibility context used to validate it. Matching this context is not an
// application authorization decision.
type BootstrapSelectionContext struct {
	CapabilityRef string `json:"capability_ref"`
	ProductID     string `json:"product_id,omitempty"`
	ProjectRef    string `json:"project_ref,omitempty"`
	Role          string `json:"role,omitempty"`
	GoalRef       string `json:"goal_ref,omitempty"`
}

type ToolingBootstrapSpec struct {
	Version               string                    `json:"version"`
	CuratedCatalogDigest  string                    `json:"curated_catalog_digest"`
	RulePackCatalogDigest string                    `json:"rulepack_catalog_digest"`
	SelectionContext      BootstrapSelectionContext `json:"selection_context"`
	BrokerOptIn           bool                      `json:"broker_opt_in"`
	UIEnabled             bool                      `json:"ui_enabled"`
	ExternalPortsEnabled  bool                      `json:"external_ports_enabled"`
	Subjects              []BootstrapSubject        `json:"subjects"`
}

// BootstrapDispositionSnapshot records only the accepted desired manifest.
// It deliberately has no filesystem, download, attempt, approval or effect fields.
type BootstrapDispositionSnapshot struct {
	Ref                   string                    `json:"ref"`
	Digest                string                    `json:"digest"`
	Capability            string                    `json:"capability"`
	BootstrapVersion      string                    `json:"bootstrap_version"`
	ManifestDigest        string                    `json:"manifest_digest"`
	CuratedCatalogDigest  string                    `json:"curated_catalog_digest"`
	RulePackCatalogDigest string                    `json:"rulepack_catalog_digest"`
	SelectionContext      BootstrapSelectionContext `json:"selection_context"`
	BrokerDisposition     string                    `json:"broker_disposition"`
	UIDisposition         string                    `json:"ui_disposition"`
	ExternalPorts         string                    `json:"external_ports"`
	Subjects              []BootstrapSubject        `json:"subjects"`
}

// ObservedBootstrapProjection is a complete caller-supplied projection. The
// doctor rehashes it; it does not discover or attest physical machine state.
type ObservedBootstrapProjection struct {
	Snapshot       ToolingBootstrapSpec `json:"snapshot"`
	SnapshotDigest string               `json:"snapshot_digest"`
}

type BootstrapDiagnostic struct {
	Code     string               `json:"code"`
	Kind     BootstrapSubjectKind `json:"kind,omitempty"`
	ID       string               `json:"id,omitempty"`
	Version  string               `json:"version,omitempty"`
	Expected string               `json:"expected,omitempty"`
	Observed string               `json:"observed,omitempty"`
}

type BootstrapDoctorReport struct {
	Status                 DoctorStatus          `json:"status"`
	Scope                  string                `json:"scope"`
	DesiredSnapshotDigest  string                `json:"desired_snapshot_digest"`
	ObservedSnapshotDigest string                `json:"observed_snapshot_digest,omitempty"`
	Diagnostics            []BootstrapDiagnostic `json:"diagnostics"`
}

type ToolingBootstrapDoctor struct {
	spec     ToolingBootstrapSpec
	digest   string
	snapshot BootstrapDispositionSnapshot
}

// NewToolingBootstrapDoctor validates one exact desired disposition against
// the already-built immutable catalogs. No install or activation is attempted.
func NewToolingBootstrapDoctor(
	curated *CuratedCatalog,
	context CuratedContext,
	rulepacks *RulePackCatalog,
	source ToolingBootstrapSpec,
) (*ToolingBootstrapDoctor, error) {
	wantedContext := bootstrapSelectionContext(context)
	if curated == nil || rulepacks == nil || source.CuratedCatalogDigest != curated.Digest() ||
		source.RulePackCatalogDigest != rulepacks.Digest() || !validSurfaceDigest(source.CuratedCatalogDigest) ||
		!validSurfaceDigest(source.RulePackCatalogDigest) || !context.valid || !context.scope.valid ||
		context.capability.String() != BootstrapCapability || source.SelectionContext != wantedContext ||
		source.UIEnabled || source.ExternalPortsEnabled || !validBootstrapSpecUTF8(source) {
		return nil, contractError(ErrorBootstrapInvalid, "catalogs_or_surfaces")
	}
	if err := curated.validateContext(context); err != nil {
		return nil, contractError(ErrorBootstrapInvalid, "curated_context")
	}
	if _, err := parseVersion(source.Version); err != nil || len(source.Subjects) == 0 {
		return nil, contractError(ErrorBootstrapInvalid, "version_or_subjects")
	}
	spec := source
	spec.Subjects = append([]BootstrapSubject(nil), source.Subjects...)
	sort.Slice(spec.Subjects, func(i, j int) bool {
		return bootstrapSubjectKey(spec.Subjects[i]) < bootstrapSubjectKey(spec.Subjects[j])
	})
	seen := make(map[string]struct{}, len(spec.Subjects))
	effectiveRulepacks, err := rulepacks.Resolve(context.scope)
	if err != nil {
		return nil, contractError(ErrorBootstrapInvalid, "rulepack_context")
	}
	rulepackByKey := make(map[string]EffectiveRulePackItem, len(effectiveRulepacks))
	for _, item := range effectiveRulepacks {
		rulepackByKey[registryKey(item.PackID, item.PackVersion)] = item
	}
	for _, subject := range spec.Subjects {
		key := bootstrapSubjectKey(subject)
		if _, duplicate := seen[key]; duplicate || !validToolID(subject.ID) || !validSurfaceDigest(subject.Digest) {
			return nil, contractError(ErrorBootstrapInvalid, "subject")
		}
		if _, err := parseVersion(subject.Version); err != nil {
			return nil, contractError(ErrorBootstrapInvalid, "subject_version")
		}
		seen[key] = struct{}{}
		if !bootstrapSubjectExists(curated, context, rulepackByKey, subject) {
			return nil, contractError(ErrorBootstrapInvalid, "subject_not_selected")
		}
	}
	digest := bootstrapSpecDigest(spec)
	if digest == "" {
		return nil, contractError(ErrorBootstrapInvalid, "manifest_digest")
	}
	broker := BootstrapSurfaceDisabled
	if spec.BrokerOptIn {
		broker = BootstrapBrokerRequested
	}
	snapshot := BootstrapDispositionSnapshot{
		Capability: BootstrapCapability, BootstrapVersion: spec.Version, ManifestDigest: digest,
		CuratedCatalogDigest: spec.CuratedCatalogDigest, RulePackCatalogDigest: spec.RulePackCatalogDigest,
		SelectionContext:  spec.SelectionContext,
		BrokerDisposition: broker, UIDisposition: BootstrapSurfaceDisabled, ExternalPorts: BootstrapSurfaceDisabled,
		Subjects: append([]BootstrapSubject(nil), spec.Subjects...),
	}
	snapshot.Digest = bootstrapSnapshotDigest(snapshot)
	if snapshot.Digest == "" {
		return nil, contractError(ErrorBootstrapInvalid, "snapshot_digest")
	}
	snapshot.Ref = "snapshot:tooling-bootstrap:" + snapshot.Digest[len("sha256:"):]
	return &ToolingBootstrapDoctor{spec: spec, digest: digest, snapshot: snapshot}, nil
}

func bootstrapSelectionContext(context CuratedContext) BootstrapSelectionContext {
	return BootstrapSelectionContext{
		CapabilityRef: context.capability.String(), ProductID: context.scope.productID,
		ProjectRef: context.scope.projectRef.String(), Role: string(context.scope.role),
		GoalRef: context.scope.goalRef.String(),
	}
}

func (doctor *ToolingBootstrapDoctor) Manifest() ToolingBootstrapSpec {
	if doctor == nil {
		return ToolingBootstrapSpec{}
	}
	result := doctor.spec
	result.Subjects = append([]BootstrapSubject(nil), doctor.spec.Subjects...)
	return result
}

func (doctor *ToolingBootstrapDoctor) Snapshot() BootstrapDispositionSnapshot {
	if doctor == nil {
		return BootstrapDispositionSnapshot{}
	}
	result := doctor.snapshot
	result.Subjects = append([]BootstrapSubject(nil), doctor.snapshot.Subjects...)
	return result
}

func (doctor *ToolingBootstrapDoctor) Diagnose(observed ObservedBootstrapProjection) BootstrapDoctorReport {
	if doctor == nil {
		return BootstrapDoctorReport{Status: DoctorProjectionDrift, Scope: BootstrapProjectionScope, Diagnostics: []BootstrapDiagnostic{{Code: "doctor_unconfigured"}}}
	}
	projection := observed.Snapshot
	projection.Subjects = append([]BootstrapSubject(nil), observed.Snapshot.Subjects...)
	baseReport := BootstrapDoctorReport{Status: DoctorProjectionDrift, Scope: BootstrapProjectionScope, DesiredSnapshotDigest: doctor.digest}
	if !validObservedBootstrapProjection(observed) {
		baseReport.Diagnostics = []BootstrapDiagnostic{{Code: "projection_invalid"}}
		return baseReport
	}
	sort.Slice(projection.Subjects, func(i, j int) bool {
		return bootstrapSubjectObservationKey(projection.Subjects[i]) < bootstrapSubjectObservationKey(projection.Subjects[j])
	})
	observedDigest := bootstrapSpecDigest(projection)
	baseReport.ObservedSnapshotDigest = observedDigest
	diagnostics := make([]BootstrapDiagnostic, 0)
	if observed.SnapshotDigest != observedDigest {
		diagnostics = append(diagnostics, BootstrapDiagnostic{Code: "projection_digest_mismatch", Expected: observedDigest, Observed: observed.SnapshotDigest})
	}
	if observedDigest != doctor.digest {
		diagnostics = append(diagnostics, BootstrapDiagnostic{Code: "desired_snapshot_mismatch", Expected: doctor.digest, Observed: observedDigest})
	}
	if projection.Version != doctor.spec.Version {
		diagnostics = append(diagnostics, BootstrapDiagnostic{Code: "bootstrap_version_mismatch", Expected: doctor.spec.Version, Observed: projection.Version})
	}
	if projection.CuratedCatalogDigest != doctor.spec.CuratedCatalogDigest {
		diagnostics = append(diagnostics, BootstrapDiagnostic{Code: "curated_catalog_mismatch", Expected: doctor.spec.CuratedCatalogDigest, Observed: projection.CuratedCatalogDigest})
	}
	if projection.RulePackCatalogDigest != doctor.spec.RulePackCatalogDigest {
		diagnostics = append(diagnostics, BootstrapDiagnostic{Code: "rulepack_catalog_mismatch", Expected: doctor.spec.RulePackCatalogDigest, Observed: projection.RulePackCatalogDigest})
	}
	if projection.SelectionContext != doctor.spec.SelectionContext {
		diagnostics = append(diagnostics, BootstrapDiagnostic{Code: "selection_context_mismatch", Expected: bootstrapJSONValue(doctor.spec.SelectionContext), Observed: bootstrapJSONValue(projection.SelectionContext)})
	}
	if projection.BrokerOptIn != doctor.spec.BrokerOptIn {
		diagnostics = append(diagnostics, BootstrapDiagnostic{Code: "broker_disposition_mismatch", Expected: bootstrapJSONValue(doctor.spec.BrokerOptIn), Observed: bootstrapJSONValue(projection.BrokerOptIn)})
	}
	if projection.UIEnabled {
		diagnostics = append(diagnostics, BootstrapDiagnostic{Code: "ui_not_disabled", Expected: "false", Observed: "true"})
	}
	if projection.ExternalPortsEnabled {
		diagnostics = append(diagnostics, BootstrapDiagnostic{Code: "external_ports_not_disabled", Expected: "false", Observed: "true"})
	}
	expected := make(map[string]BootstrapSubject, len(doctor.spec.Subjects))
	for _, subject := range doctor.spec.Subjects {
		expected[bootstrapSubjectKey(subject)] = subject
	}
	observedSubjects := projection.Subjects
	seen := make(map[string]struct{}, len(observedSubjects))
	for _, subject := range observedSubjects {
		key := bootstrapSubjectKey(subject)
		wanted, found := expected[key]
		if _, duplicate := seen[key]; duplicate {
			diagnostics = append(diagnostics, diagnosticFor("subject_duplicate", subject, subject.Digest, subject.Digest))
			continue
		}
		seen[key] = struct{}{}
		if !found {
			diagnostics = append(diagnostics, diagnosticFor("subject_unexpected", subject, "", subject.Digest))
		} else if wanted.Digest != subject.Digest {
			diagnostics = append(diagnostics, diagnosticFor("subject_digest_mismatch", subject, wanted.Digest, subject.Digest))
		}
	}
	for key, subject := range expected {
		if _, found := seen[key]; !found {
			diagnostics = append(diagnostics, diagnosticFor("subject_missing", subject, subject.Digest, ""))
		}
	}
	sort.Slice(diagnostics, func(i, j int) bool {
		return bootstrapJSONValue(diagnostics[i]) < bootstrapJSONValue(diagnostics[j])
	})
	status := DoctorProjectionMatch
	if len(diagnostics) != 0 {
		status = DoctorProjectionDrift
	}
	baseReport.Status, baseReport.Diagnostics = status, diagnostics
	return baseReport
}

func bootstrapSubjectExists(
	curated *CuratedCatalog,
	context CuratedContext,
	rulepacks map[string]EffectiveRulePackItem,
	subject BootstrapSubject,
) bool {
	switch subject.Kind {
	case BootstrapKindTool:
		entry, found, err := curated.ResolveTool(context, subject.ID, subject.Version)
		return err == nil && found && entry.Digest == subject.Digest
	case BootstrapKindSkill:
		entry, found, err := curated.ResolveSkill(context, subject.ID, subject.Version)
		return err == nil && found && !entry.Revoked && entry.ReleaseDigest == subject.Digest
	case BootstrapKindPlugin:
		entry, found, err := curated.ResolvePlugin(context, subject.ID, subject.Version)
		return err == nil && found && entry.Digest == subject.Digest
	case BootstrapKindRulePack:
		entry, found := rulepacks[registryKey(subject.ID, subject.Version)]
		return found && entry.PackDigest == subject.Digest
	default:
		return false
	}
}

func bootstrapSubjectKey(subject BootstrapSubject) string {
	return string(subject.Kind) + "\x00" + subject.ID + "\x00" + subject.Version
}

func bootstrapSubjectObservationKey(subject BootstrapSubject) string {
	return bootstrapSubjectKey(subject) + "\x00" + subject.Digest
}

func diagnosticFor(code string, subject BootstrapSubject, expected, observed string) BootstrapDiagnostic {
	return BootstrapDiagnostic{Code: code, Kind: subject.Kind, ID: subject.ID, Version: subject.Version, Expected: expected, Observed: observed}
}

func bootstrapSpecDigest(spec ToolingBootstrapSpec) string {
	if !validBootstrapSpecUTF8(spec) {
		return ""
	}
	return bootstrapJSONDigest(struct {
		Contract string               `json:"contract"`
		Spec     ToolingBootstrapSpec `json:"spec"`
	}{"orquesta.tooling.bootstrap-manifest.v2", spec})
}

func bootstrapSnapshotDigest(snapshot BootstrapDispositionSnapshot) string {
	if !validBootstrapSnapshotUTF8(snapshot) {
		return ""
	}
	snapshot.Ref, snapshot.Digest = "", ""
	return bootstrapJSONDigest(struct {
		Contract string                       `json:"contract"`
		Snapshot BootstrapDispositionSnapshot `json:"snapshot"`
	}{"orquesta.tooling.bootstrap-snapshot.v3", snapshot})
}

func bootstrapJSONDigest(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func validBootstrapSpecUTF8(spec ToolingBootstrapSpec) bool {
	if !validBootstrapStrings(
		spec.Version, spec.CuratedCatalogDigest, spec.RulePackCatalogDigest,
		spec.SelectionContext.CapabilityRef, spec.SelectionContext.ProductID,
		spec.SelectionContext.ProjectRef, spec.SelectionContext.Role, spec.SelectionContext.GoalRef,
	) {
		return false
	}
	for _, subject := range spec.Subjects {
		if !validBootstrapSubjectUTF8(subject) {
			return false
		}
	}
	return true
}

func validBootstrapSnapshotUTF8(snapshot BootstrapDispositionSnapshot) bool {
	if !validBootstrapStrings(
		snapshot.Ref, snapshot.Digest, snapshot.Capability, snapshot.BootstrapVersion, snapshot.ManifestDigest,
		snapshot.CuratedCatalogDigest, snapshot.RulePackCatalogDigest,
		snapshot.SelectionContext.CapabilityRef, snapshot.SelectionContext.ProductID,
		snapshot.SelectionContext.ProjectRef, snapshot.SelectionContext.Role, snapshot.SelectionContext.GoalRef,
		snapshot.BrokerDisposition, snapshot.UIDisposition, snapshot.ExternalPorts,
	) {
		return false
	}
	for _, subject := range snapshot.Subjects {
		if !validBootstrapSubjectUTF8(subject) {
			return false
		}
	}
	return true
}

func validObservedBootstrapProjection(observed ObservedBootstrapProjection) bool {
	if !validSurfaceDigest(observed.SnapshotDigest) || !validBootstrapSpecUTF8(observed.Snapshot) ||
		!validSurfaceDigest(observed.Snapshot.CuratedCatalogDigest) || !validSurfaceDigest(observed.Snapshot.RulePackCatalogDigest) {
		return false
	}
	for _, subject := range observed.Snapshot.Subjects {
		if !validSurfaceDigest(subject.Digest) {
			return false
		}
	}
	return true
}

func bootstrapJSONValue(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func validBootstrapSubjectUTF8(subject BootstrapSubject) bool {
	return validBootstrapStrings(string(subject.Kind), subject.ID, subject.Version, subject.Digest)
}

func validBootstrapStrings(values ...string) bool {
	for _, value := range values {
		if !utf8.ValidString(value) {
			return false
		}
	}
	return true
}
