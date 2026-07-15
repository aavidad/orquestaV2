package orquesta_test

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

type productManifest struct {
	SchemaVersion     int                        `json:"schema_version"`
	Product           string                     `json:"product"`
	ReleaseTarget     string                     `json:"release_target"`
	BaseCommit        string                     `json:"base_commit"`
	OperatorDecisions map[string]json.RawMessage `json:"operator_decisions"`
	Capabilities      []struct {
		ID              string `json:"id"`
		Status          string `json:"status"`
		CutoverRequired bool   `json:"cutover_required"`
		Phase           string `json:"phase"`
		AcceptanceRef   string `json:"acceptance_ref"`
		EvidenceRef     string `json:"evidence_ref,omitempty"`
	} `json:"capabilities"`
	Deferred []string `json:"deferred"`
}

type realE2EReceipt struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind"`
	Result        string `json:"result"`
	SourceSHA256  string `json:"source_sha256"`
	ExecutedAt    string `json:"executed_at"`
	Marker        string `json:"marker"`
	Command       string `json:"command"`
}

func TestAcceptedProductCapabilitiesHaveExecutableEvidence(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("repository root: %v", err)
	}
	payload, err := os.ReadFile(filepath.Join(root, "product", "capabilities.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest productManifest
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if err := requireProductJSONEOF(decoder); err != nil {
		t.Fatalf("manifest trailing content: %v", err)
	}
	baseCommit, baseErr := hex.DecodeString(manifest.BaseCommit)
	if manifest.SchemaVersion != 1 || manifest.Product != "orquesta" || manifest.ReleaseTarget != "minimal_functional_cut" ||
		baseErr != nil || len(baseCommit) != 20 || len(manifest.Capabilities) == 0 {
		t.Fatalf("invalid manifest header: schema=%d capabilities=%d", manifest.SchemaVersion, len(manifest.Capabilities))
	}
	expectedDecisions := map[string]string{
		"deployment":               "single_operator_local",
		"default_actor_ref":        "actor:local-owner",
		"default_project_ref":      "project:default",
		"state_adapter":            "sqlite",
		"artifact_adapter":         "filesystem",
		"first_real_agent_adapter": "codex",
		"public_binding":           "mcp_streamable_http",
		"legacy_runtime":           "read_only_and_stopped",
		"supported_platform":       "linux",
	}
	if len(manifest.OperatorDecisions) != len(expectedDecisions) {
		t.Fatalf("operator decisions = %d, want %d", len(manifest.OperatorDecisions), len(expectedDecisions))
	}
	for key, want := range expectedDecisions {
		value, found := manifest.OperatorDecisions[key]
		var got string
		if !found || json.Unmarshal(value, &got) != nil || got != want {
			t.Fatalf("operator decision %q = %s, want %q", key, value, want)
		}
	}
	expectedCapabilities := map[string]string{
		"CORE-INTENT":          "internal/goal/intent_test.go",
		"CORE-GOAL":            "internal/goal/goal_test.go",
		"CORE-WORK-ITEM":       "internal/goal/work_item_test.go",
		"CORE-SINGLE-WRITER":   "internal/application/orchestrator_test.go",
		"CORE-EVIDENCE":        "internal/application/closure_test.go",
		"IDENTITY-LOCAL-OWNER": "internal/identity/contracts_test.go",
		"IDENTITY-LOCAL-TOKEN": "internal/adapters/auth/localtoken/localtoken_test.go",
		"STATE-SQLITE":         "internal/adapters/state/sqlite/repository_test.go",
		"ARTIFACT-FILESYSTEM":  "internal/adapters/artifact/filesystem/store_test.go",
		"CONFIG-CANONICAL":     "internal/config/config_test.go",
		"AGENT-CONTRACT":       "internal/ports/agent_contract_test.go",
		"AGENT-CODEX":          "internal/bootstrap/codex_real_e2e_test.go",
		"API-MCP":              "internal/bootstrap/mcp_e2e_test.go",
		"OPS-RESTART-REPLAY":   "internal/bootstrap/restart_e2e_test.go",
		"OPS-SHUTDOWN":         "internal/bootstrap/shutdown_e2e_test.go",
		"ARCH-HEXAGONAL":       "architecture_rebuild_test.go",
		"SURFACE-I18N":         "internal/i18n/catalog_test.go",
	}
	if len(manifest.Capabilities) != len(expectedCapabilities) {
		t.Fatalf("capabilities = %d, want %d", len(manifest.Capabilities), len(expectedCapabilities))
	}

	ids := make(map[string]struct{}, len(manifest.Capabilities))
	for _, capability := range manifest.Capabilities {
		if strings.TrimSpace(capability.ID) == "" {
			t.Fatal("capability without id")
		}
		if _, duplicate := ids[capability.ID]; duplicate {
			t.Fatalf("duplicate capability %q", capability.ID)
		}
		ids[capability.ID] = struct{}{}
		if capability.Status != "accepted" {
			t.Fatalf("capability %q has unsupported status %q in accepted list", capability.ID, capability.Status)
		}
		wantRef, required := expectedCapabilities[capability.ID]
		if !required || capability.AcceptanceRef != wantRef {
			t.Fatalf("capability %q acceptance_ref = %q, want %q", capability.ID, capability.AcceptanceRef, wantRef)
		}
		if strings.TrimSpace(capability.Phase) == "" {
			t.Fatalf("capability %q has no phase", capability.ID)
		}
		requireRepositoryFile(t, root, capability.AcceptanceRef)
		if !strings.HasSuffix(capability.AcceptanceRef, "_test.go") {
			t.Fatalf("capability %q acceptance_ref is not an executable Go test: %q", capability.ID, capability.AcceptanceRef)
		}
		if capability.EvidenceRef != "" {
			requireRepositoryFile(t, root, capability.EvidenceRef)
		}
	}

	expectedDeferred := map[string]struct{}{
		"goal_dag_and_phase_templates":              {},
		"director_role_and_lease":                   {},
		"pause_resume_cancel_and_replan":            {},
		"budgets_permissions_and_effect_governance": {},
		"multiuser_accounts":                        {},
		"rbac":                                      {},
		"active_directory":                          {},
		"oidc":                                      {},
		"postgres_state_adapter":                    {},
		"credential_store":                          {},
		"workspace_and_git_adapters":                {},
		"forge":                                     {},
		"typed_http_api":                            {},
		"web_admin":                                 {},
		"rich_wizard":                               {},
		"application_factory":                       {},
		"council":                                   {},
		"independent_human_reviewers":               {},
		"tool_registry_and_sdk":                     {},
		"skill_registry_and_rulepacks":              {},
		"context_broker_and_domain_rag":             {},
		"token_cost_routing_and_evals":              {},
		"hermes_director":                           {},
		"claude":                                    {},
		"gemini":                                    {},
		"ollama":                                    {},
		"local_openai_compatible_runtime":           {},
		"research_and_document_connectors":          {},
		"notifications":                             {},
		"telemetry_and_idle_watchdog":               {},
		"backup_restore_and_migration":              {},
		"opes":                                      {},
		"deploy_plugins":                            {},
	}
	if len(manifest.Deferred) != len(expectedDeferred) {
		t.Fatalf("deferred capabilities = %d, want %d", len(manifest.Deferred), len(expectedDeferred))
	}
	deferred := make(map[string]struct{}, len(manifest.Deferred))
	for _, item := range manifest.Deferred {
		if strings.TrimSpace(item) == "" {
			t.Fatal("empty deferred capability")
		}
		if _, duplicate := deferred[item]; duplicate {
			t.Fatalf("duplicate deferred capability %q", item)
		}
		if _, accepted := ids[item]; accepted {
			t.Fatalf("capability %q is both accepted and deferred", item)
		}
		if _, required := expectedDeferred[item]; !required {
			t.Fatalf("unexpected deferred capability %q", item)
		}
		deferred[item] = struct{}{}
	}
}

func TestRealCodexReceiptMatchesCurrentProductSource(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("repository root: %v", err)
	}
	path := filepath.Join(root, "product", "evidence", "real_codex_mcp_e2e.json")
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read real Codex receipt: %v", err)
	}
	var receipt realE2EReceipt
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		t.Fatalf("decode real Codex receipt: %v", err)
	}
	if err := requireProductJSONEOF(decoder); err != nil {
		t.Fatalf("receipt trailing content: %v", err)
	}
	wantDigest, err := rebuildProductSourceDigest(root)
	if err != nil {
		t.Fatalf("source digest: %v", err)
	}
	executedAt, timestampErr := time.Parse(time.RFC3339, receipt.ExecutedAt)
	markerSuffix := strings.TrimPrefix(receipt.Marker, "ORQUESTA_CODEX_E2E_OK_")
	decodedMarker, markerErr := hex.DecodeString(markerSuffix)
	if receipt.SchemaVersion != 1 || receipt.Kind != "real_codex_mcp_e2e" || receipt.Result != "PASS" ||
		!strings.HasPrefix(receipt.Marker, "ORQUESTA_CODEX_E2E_OK_") || len(receipt.Marker) != len("ORQUESTA_CODEX_E2E_OK_")+32 ||
		markerErr != nil || len(decodedMarker) != 16 ||
		timestampErr != nil || executedAt.After(time.Now().Add(5*time.Minute)) ||
		!strings.Contains(receipt.Command, "go test -mod=vendor") ||
		!strings.Contains(receipt.Command, "TestRealCodexAdapterClosesGoalThroughProductionMCPServer") ||
		!strings.Contains(receipt.Command, "-orquesta-real-codex-config=") ||
		receipt.SourceSHA256 != wantDigest {
		t.Fatalf("real Codex receipt is stale or invalid: got=%+v current_source_sha256=%s", receipt, wantDigest)
	}
}

func requireProductJSONEOF(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("unexpected trailing JSON value")
	}
	return err
}

func requireRepositoryFile(t *testing.T, root, relative string) {
	t.Helper()
	clean := filepath.Clean(filepath.FromSlash(relative))
	if relative == "" || filepath.IsAbs(relative) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		t.Fatalf("unsafe repository evidence path %q", relative)
	}
	info, err := os.Lstat(filepath.Join(root, clean))
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("repository evidence path %q is not a regular file: %v", relative, err)
	}
}

func rebuildProductSourceDigest(root string) (string, error) {
	paths := []string{"architecture_rebuild_test.go", "product_manifest_test.go", "go.mod", "go.sum", "product/capabilities.json"}
	for _, directory := range []string{"internal", "cmd/orquesta", "config", "vendor"} {
		err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(directory)), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("source symlink forbidden: %s", path)
			}
			if entry.IsDir() {
				return nil
			}
			if !entry.Type().IsRegular() {
				return fmt.Errorf("non-regular source forbidden: %s", path)
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			paths = append(paths, filepath.ToSlash(relative))
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	sort.Strings(paths)
	digest := sha256.New()
	for _, relative := range paths {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			return "", err
		}
		writeDigestField(digest, []byte(relative))
		writeDigestField(digest, content)
	}
	return "sha256:" + hex.EncodeToString(digest.Sum(nil)), nil
}

func writeDigestField(digest interface{ Write([]byte) (int, error) }, value []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = digest.Write(length[:])
	_, _ = digest.Write(value)
}
