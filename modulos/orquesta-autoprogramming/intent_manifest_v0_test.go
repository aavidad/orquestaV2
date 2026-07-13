package orquestaautoprogramming

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func hasAutoprogrammingIssueCodeV0(issues []AutoprogrammingRequestIssueV0, want string) bool {
	for _, issue := range issues {
		if issue.Code == want {
			return true
		}
	}
	return false
}

func TestAutoprogrammingIntentManifestV0PreservesOriginalBytesAndRejectsTampering(t *testing.T) {
	manifest, issues := BuildAutoprogrammingIntentManifestV0(validAutoprogrammingRequestV0(nil))
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	manifest.RequestJSON = append(manifest.RequestJSON, ' ')
	if !hasAutoprogrammingIssueCodeV0(ValidateAutoprogrammingIntentManifestV0(manifest), "intent_manifest_sha256_mismatch") {
		t.Fatal("tampering accepted")
	}
	manifest.RequestSHA256 = strings.ToUpper(manifest.RequestSHA256)
	if !hasAutoprogrammingIssueCodeV0(ValidateAutoprogrammingIntentManifestV0(manifest), "intent_manifest_sha256_invalid") {
		t.Fatal("uppercase SHA accepted")
	}
}

func TestAutoprogrammingIntentManifestV0CanonicalEquivalentRequestsShareBytesV0(t *testing.T) {
	base := validAutoprogrammingRequestV0(nil)
	base.AreaAliases = []AutoprogrammingAreaAliasV0{{Alias: "Apps Core", Area: "Orchestration Core"}}
	first, issues := BuildAutoprogrammingIntentManifestV0(base)
	if len(issues) != 0 {
		t.Fatalf("base issues=%+v", issues)
	}

	variant := validAutoprogrammingRequestV0(nil)
	variant.RequestRef = "  " + variant.RequestRef + "  "
	variant.ProjectRef = " " + variant.ProjectRef
	variant.Tasks[0].TaskRef = " " + variant.Tasks[0].TaskRef + " "
	variant.WriteSet = append(variant.WriteSet, " "+variant.WriteSet[0]+" ")
	variant.RequiredTests = append(variant.RequiredTests, " "+variant.RequiredTests[0]+" ")
	variant.AreaAliases = []AutoprogrammingAreaAliasV0{
		{Alias: "app_core", Area: "orchestration_core"},
		{Alias: "applications-core", Area: "Orchestration Core"},
	}
	variant.MaxTaskRefs = 10
	variant.MaxAreas = 10
	variant.MaxWriteSetEntries = 10
	variant.MaxDelegationDepth = 1
	variant.MaxSubagentsPerAgent = 6
	variant.MaxRecursiveAgents = 60
	second, issues := BuildAutoprogrammingIntentManifestV0(variant)
	if len(issues) != 0 {
		t.Fatalf("variant issues=%+v", issues)
	}
	if first.RequestSHA256 != second.RequestSHA256 || !bytes.Equal(first.RequestJSON, second.RequestJSON) {
		t.Fatalf("equivalent requests diverged:\nfirst=%s\nsecond=%s", first.RequestJSON, second.RequestJSON)
	}
}

func TestAutoprogrammingIntentManifestV0MaterialOrderChangesIdentityV0(t *testing.T) {
	base := validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.RequiredTests = []string{"go test ./first", "go test ./second"}
	})
	first, issues := BuildAutoprogrammingIntentManifestV0(base)
	if len(issues) != 0 {
		t.Fatalf("base issues=%+v", issues)
	}
	base.RequiredTests[0], base.RequiredTests[1] = base.RequiredTests[1], base.RequiredTests[0]
	second, issues := BuildAutoprogrammingIntentManifestV0(base)
	if len(issues) != 0 {
		t.Fatalf("swapped issues=%+v", issues)
	}
	if first.RequestSHA256 == second.RequestSHA256 {
		t.Fatal("material required_tests order was erased")
	}
}

func TestAutoprogrammingIntentManifestV0RejectsNonCanonicalJSONV0(t *testing.T) {
	manifest, issues := BuildAutoprogrammingIntentManifestV0(validAutoprogrammingRequestV0(nil))
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	unknown := append(append([]byte(nil), manifest.RequestJSON[:len(manifest.RequestJSON)-1]...), []byte(`,"unknown":1}`)...)
	duplicate := append([]byte(`{"request_ref":"duplicate",`), manifest.RequestJSON[1:]...)
	for name, raw := range map[string][]byte{"unknown": unknown, "duplicate": duplicate, "trailing": append(append([]byte(nil), manifest.RequestJSON...), ' ')} {
		t.Run(name, func(t *testing.T) {
			if _, gotIssues := AutoprogrammingIntentManifestFromRequestJSONV0(raw); len(gotIssues) == 0 {
				t.Fatalf("noncanonical JSON accepted: %s", raw)
			}
		})
	}
	manifest.RequestJSON = append(manifest.RequestJSON, ' ')
	sum := sha256.Sum256(manifest.RequestJSON)
	manifest.RequestSHA256 = hex.EncodeToString(sum[:])
	if !hasAutoprogrammingIssueCodeV0(ValidateAutoprogrammingIntentManifestV0(manifest), "intent_manifest_request_json_noncanonical") {
		t.Fatal("noncanonical manifest with matching SHA accepted")
	}
}

func TestAutoprogrammingIntentManifestV0EnforcesFilesystemPortableIdentityAndSizeV0(t *testing.T) {
	for name, requestRef := range map[string]string{
		"dot segments": "request-ref-a..b",
		"too long":     "request-ref-" + strings.Repeat("a", autoprogrammingIntentManifestRequestRefMaxBytesV0),
	} {
		t.Run(name, func(t *testing.T) {
			request := validAutoprogrammingRequestV0(nil)
			request.RequestRef = requestRef
			if _, issues := BuildAutoprogrammingIntentManifestV0(request); len(issues) == 0 {
				t.Fatalf("filesystem-incompatible request_ref accepted: %q", requestRef)
			}
		})
	}
	request := validAutoprogrammingRequestV0(nil)
	request.Tasks[0].Objective = strings.Repeat("x", AutoprogrammingIntentManifestMaxBytesV0)
	if _, issues := BuildAutoprogrammingIntentManifestV0(request); !hasAutoprogrammingIssueCodeV0(issues, "intent_manifest_request_json_too_large") {
		t.Fatalf("oversized manifest issues=%+v", issues)
	}
}

func TestInMemoryAutoprogrammingIntentManifestStoreV0IndexesNormalizedIdentityV0(t *testing.T) {
	manifest, issues := BuildAutoprogrammingIntentManifestV0(validAutoprogrammingRequestV0(nil))
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	manifest.RequestRef = " " + manifest.RequestRef + " "
	manifest.ManifestRef = " " + manifest.ManifestRef + " "
	store := NewInMemoryAutoprogrammingIntentManifestStoreV0()
	if _, err := store.CreateAutoprogrammingIntentManifestIfAbsentV0(nil, manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadAutoprogrammingIntentManifestV0(nil, strings.TrimSpace(manifest.RequestRef)); err != nil {
		t.Fatalf("normalized key not loadable: %v", err)
	}
}

func TestAutoprogrammingIntentManifestV0PreservesCanonicalPrepareRunEnvelopeV0(t *testing.T) {
	envelope := AutoprogrammingPrepareRunEnvelopeV0{
		RequestID:              " request-envelope-001 ",
		CorrelationID:          " corr-envelope-001 ",
		IdempotencyKey:         " idem-envelope-001 ",
		OccurredAt:             " 2026-07-13T10:00:00Z ",
		RequestedBy:            " operator-ref-001 ",
		DirectorExecutionMode:  " GOAL_FIRST ",
		AutoprogrammingRequest: validAutoprogrammingRequestV0(nil),
		PriorityScore:          70,
		RequiredSettings: []AutoprogrammingPrepareRunRequiredSettingV0{
			{Key: " ORQUESTA_PUBLIC_LIMIT "},
			{Key: "ORQUESTA_PUBLIC_LIMIT"},
		},
	}
	manifest, issues := BuildAutoprogrammingIntentManifestFromPrepareRunEnvelopeV0(envelope)
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if len(ValidateAutoprogrammingIntentManifestV0(manifest)) != 0 {
		t.Fatalf("manifest invalid: %+v", manifest)
	}
	var stored AutoprogrammingPrepareRunEnvelopeV0
	if err := json.Unmarshal(manifest.RequestJSON, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.SchemaVersion != AutoprogrammingPrepareRunEnvelopeSchemaV0 ||
		stored.RequestID != "request-envelope-001" ||
		stored.CorrelationID != "corr-envelope-001" ||
		stored.IdempotencyKey != "idem-envelope-001" ||
		stored.RequestedBy != "operator-ref-001" ||
		stored.DirectorExecutionMode != "goal_first" ||
		stored.PriorityScore != 70 ||
		len(stored.RequiredSettings) != 1 ||
		stored.RequiredSettings[0].Key != "ORQUESTA_PUBLIC_LIMIT" {
		t.Fatalf("stored=%+v", stored)
	}
}

func TestAutoprogrammingIntentManifestV0EnvelopeReplayAndConflictV0(t *testing.T) {
	envelope := AutoprogrammingPrepareRunEnvelopeV0{
		RequestID:              "request-envelope-replay-001",
		CorrelationID:          "corr-envelope-replay-001",
		IdempotencyKey:         "idem-envelope-replay-001",
		OccurredAt:             "2026-07-13T10:00:00Z",
		RequestedBy:            "operator-ref-envelope-replay-001",
		AutoprogrammingRequest: validAutoprogrammingRequestV0(nil),
	}
	first, issues := BuildAutoprogrammingIntentManifestFromPrepareRunEnvelopeV0(envelope)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	store := NewInMemoryAutoprogrammingIntentManifestStoreV0()
	stored, err := store.CreateAutoprogrammingIntentManifestIfAbsentV0(nil, first)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := store.CreateAutoprogrammingIntentManifestIfAbsentV0(nil, first)
	if err != nil || replayed.RequestSHA256 != stored.RequestSHA256 {
		t.Fatalf("replay=%+v err=%v", replayed, err)
	}
	envelope.PriorityScore = 99
	conflict, issues := BuildAutoprogrammingIntentManifestFromPrepareRunEnvelopeV0(envelope)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	if _, err := store.CreateAutoprogrammingIntentManifestIfAbsentV0(nil, conflict); !errors.Is(err, errInMemoryIntentManifestConflictV0) {
		t.Fatalf("conflict err=%v", err)
	}
}

func TestAutoprogrammingIntentManifestV0EnvelopeRespects480KiBCeilingV0(t *testing.T) {
	request := validAutoprogrammingRequestV0(nil)
	base, issues := BuildAutoprogrammingIntentManifestV0(request)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	request.Tasks[0].Objective = "x"
	withOneByte, issues := BuildAutoprogrammingIntentManifestV0(request)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	objectiveJSONOverhead := len(withOneByte.RequestJSON) - len(base.RequestJSON) - 1
	request.Tasks[0].Objective = strings.Repeat("x", AutoprogrammingIntentManifestMaxBytesV0-len(base.RequestJSON)-objectiveJSONOverhead)
	domainAtLimit, issues := BuildAutoprogrammingIntentManifestV0(request)
	if len(issues) != 0 || len(domainAtLimit.RequestJSON) != AutoprogrammingIntentManifestMaxBytesV0 {
		t.Fatalf("domain bytes=%d issues=%+v", len(domainAtLimit.RequestJSON), issues)
	}
	envelope := AutoprogrammingPrepareRunEnvelopeV0{
		RequestID:              "request-envelope-limit-001",
		CorrelationID:          "corr-envelope-limit-001",
		IdempotencyKey:         "idem-envelope-limit-001",
		OccurredAt:             "2026-07-13T10:00:00Z",
		RequestedBy:            "operator-ref-envelope-limit-001",
		AutoprogrammingRequest: request,
	}
	if _, issues := BuildAutoprogrammingIntentManifestFromPrepareRunEnvelopeV0(envelope); !hasAutoprogrammingIssueCodeV0(issues, "intent_manifest_request_json_too_large") {
		t.Fatalf("oversized envelope issues=%+v", issues)
	}
}
