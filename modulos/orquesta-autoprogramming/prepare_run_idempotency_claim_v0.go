package orquestaautoprogramming

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

const AutoprogrammingPrepareRunIdempotencyClaimSchemaV0 = "orquesta_autoprogramming_prepare_run_idempotency_claim.v0"

var ErrAutoprogrammingPrepareRunIdempotencyClaimInvalidV0 = errors.New("prepare_run_idempotency_claim_invalid")
var ErrAutoprogrammingPrepareRunIdempotencyClaimConflictV0 = errors.New("prepare_run_idempotency_claim_conflict")
var ErrAutoprogrammingPrepareRunIdempotencyClaimUnavailableV0 = errors.New("prepare_run_idempotency_claim_unavailable")

// AutoprogrammingPrepareRunIdempotencyClaimV0 is an immutable projection of
// the command envelope onto its effective manifest. It prevents one public
// idempotency key from authorizing two different prepare_run commands.
type AutoprogrammingPrepareRunIdempotencyClaimV0 struct {
	SchemaVersion  string `json:"schema_version"`
	IdempotencyKey string `json:"idempotency_key"`
	CommandSHA256  string `json:"command_sha256"`
	RequestRef     string `json:"request_ref"`
	ManifestRef    string `json:"manifest_ref"`
	ManifestSHA256 string `json:"manifest_sha256"`
}

// AutoprogrammingPrepareRunIdempotencyClaimStorePortV0 is deliberately
// separate from the legacy manifest store contract. Compositions that accept a
// prepare_run envelope must inject an atomic create-once implementation.
type AutoprogrammingPrepareRunIdempotencyClaimStorePortV0 interface {
	CreateAutoprogrammingPrepareRunIdempotencyClaimIfAbsentV0(context.Context, AutoprogrammingPrepareRunIdempotencyClaimV0) (AutoprogrammingPrepareRunIdempotencyClaimV0, error)
}

func BuildAutoprogrammingPrepareRunIdempotencyClaimV0(
	idempotencyKey string,
	commandManifest AutoprogrammingIntentManifestV0,
	effectiveManifest AutoprogrammingIntentManifestV0,
) (AutoprogrammingPrepareRunIdempotencyClaimV0, []AutoprogrammingRequestIssueV0) {
	if issues := ValidateAutoprogrammingIntentManifestV0(commandManifest); len(issues) != 0 {
		return AutoprogrammingPrepareRunIdempotencyClaimV0{}, issues
	}
	commandContent, issues := ProjectAutoprogrammingIntentManifestContentV0(commandManifest)
	if len(issues) != 0 {
		return AutoprogrammingPrepareRunIdempotencyClaimV0{}, issues
	}
	if commandContent.Kind != AutoprogrammingIntentManifestContentPrepareRunEnvelopeV0 {
		return AutoprogrammingPrepareRunIdempotencyClaimV0{}, []AutoprogrammingRequestIssueV0{
			autoprogrammingRequestIssueV0("prepare_run_idempotency_command_envelope_required", "command_manifest", "prepare_run_idempotency_command_envelope_required"),
		}
	}
	if issues := ValidateAutoprogrammingIntentManifestV0(effectiveManifest); len(issues) != 0 {
		return AutoprogrammingPrepareRunIdempotencyClaimV0{}, issues
	}
	claim := AutoprogrammingPrepareRunIdempotencyClaimV0{
		SchemaVersion:  AutoprogrammingPrepareRunIdempotencyClaimSchemaV0,
		IdempotencyKey: idempotencyKey,
		CommandSHA256:  commandManifest.RequestSHA256,
		RequestRef:     effectiveManifest.RequestRef,
		ManifestRef:    effectiveManifest.ManifestRef,
		ManifestSHA256: effectiveManifest.RequestSHA256,
	}
	claim = NormalizeAutoprogrammingPrepareRunIdempotencyClaimV0(claim)
	return claim, ValidateAutoprogrammingPrepareRunIdempotencyClaimV0(claim)
}

func NormalizeAutoprogrammingPrepareRunIdempotencyClaimV0(
	claim AutoprogrammingPrepareRunIdempotencyClaimV0,
) AutoprogrammingPrepareRunIdempotencyClaimV0 {
	claim.SchemaVersion = strings.TrimSpace(claim.SchemaVersion)
	claim.IdempotencyKey = strings.TrimSpace(claim.IdempotencyKey)
	claim.CommandSHA256 = strings.TrimSpace(claim.CommandSHA256)
	claim.RequestRef = strings.TrimSpace(claim.RequestRef)
	claim.ManifestRef = strings.TrimSpace(claim.ManifestRef)
	claim.ManifestSHA256 = strings.TrimSpace(claim.ManifestSHA256)
	return claim
}

func ValidateAutoprogrammingPrepareRunIdempotencyClaimV0(
	claim AutoprogrammingPrepareRunIdempotencyClaimV0,
) []AutoprogrammingRequestIssueV0 {
	claim = NormalizeAutoprogrammingPrepareRunIdempotencyClaimV0(claim)
	var issues []AutoprogrammingRequestIssueV0
	if claim.SchemaVersion != AutoprogrammingPrepareRunIdempotencyClaimSchemaV0 {
		issues = append(issues, autoprogrammingRequestIssueV0("prepare_run_idempotency_claim_schema_invalid", "schema_version", "prepare_run_idempotency_claim_schema_invalid"))
	}
	if !autoprogrammingPrepareRunIdentityValueV0(claim.IdempotencyKey) {
		issues = append(issues, autoprogrammingRequestIssueV0("prepare_run_idempotency_key_invalid", "idempotency_key", "prepare_run_idempotency_key_invalid"))
	}
	if !autoprogrammingIntentManifestSHA256V0.MatchString(claim.CommandSHA256) {
		issues = append(issues, autoprogrammingRequestIssueV0("prepare_run_idempotency_command_sha256_invalid", "command_sha256", "prepare_run_idempotency_command_sha256_invalid"))
	}
	if !autoprogrammingIntentManifestRefTokenV0.MatchString(claim.RequestRef) || len(claim.RequestRef) > autoprogrammingIntentManifestRequestRefMaxBytesV0 || strings.Contains(claim.RequestRef, "..") {
		issues = append(issues, autoprogrammingRequestIssueV0("prepare_run_idempotency_request_ref_invalid", "request_ref", "prepare_run_idempotency_request_ref_invalid"))
	}
	if claim.ManifestRef != "intent-manifest-ref-"+claim.RequestRef {
		issues = append(issues, autoprogrammingRequestIssueV0("prepare_run_idempotency_manifest_ref_invalid", "manifest_ref", "prepare_run_idempotency_manifest_ref_invalid"))
	}
	if !autoprogrammingIntentManifestSHA256V0.MatchString(claim.ManifestSHA256) {
		issues = append(issues, autoprogrammingRequestIssueV0("prepare_run_idempotency_manifest_sha256_invalid", "manifest_sha256", "prepare_run_idempotency_manifest_sha256_invalid"))
	}
	return issues
}

func EqualAutoprogrammingPrepareRunIdempotencyClaimV0(
	left AutoprogrammingPrepareRunIdempotencyClaimV0,
	right AutoprogrammingPrepareRunIdempotencyClaimV0,
) bool {
	return NormalizeAutoprogrammingPrepareRunIdempotencyClaimV0(left) == NormalizeAutoprogrammingPrepareRunIdempotencyClaimV0(right)
}

func autoprogrammingPrepareRunIdentityValueV0(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || !utf8.ValidString(value) {
		return false
	}
	for _, current := range value {
		if unicode.IsControl(current) {
			return false
		}
	}
	return true
}
