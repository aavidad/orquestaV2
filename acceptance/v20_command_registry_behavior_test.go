package acceptance_test

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	commandcore "orquesta/internal/commands"
	"orquesta/internal/identity"
)

func v20AssertMigrationAndBehaviorGates(t *testing.T, repositoryRoot string, fixture v20Fixture) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(repositoryRoot, filepath.FromSlash(fixture.MigrationContract.FileGlob)))
	if err != nil || len(matches) != 1 {
		t.Errorf("V20 command audit migration matches=%v err=%v, want exactly one", matches, err)
		return
	}
	content, err := os.ReadFile(matches[0])
	if err != nil {
		t.Error(err)
		return
	}
	lower := strings.ToLower(string(content))
	for _, marker := range fixture.MigrationContract.RequiredMarkers {
		if !strings.Contains(lower, strings.ToLower(marker)) {
			t.Errorf("V20 migration lacks %q", marker)
		}
	}
}

func v20AssertPrivateInterfacesContainNoLifecycle(t *testing.T, repositoryRoot string, fixture v20Fixture) {
	t.Helper()
	for _, relative := range []string{
		"internal/interfaces/httpapi/interface.go", "internal/interfaces/mcp/v20_commands.go",
		"internal/interfaces/cli/runner.go", "sdk/commands/client.go",
	} {
		content, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range fixture.ForbiddenInterfaceImports {
			if strings.Contains(string(content), `"`+forbidden+`"`) {
				t.Errorf("V20 private interface %s imports %s", relative, forbidden)
			}
		}
	}
}

func v20AssertClosedSchema(t *testing.T, name string, encoded json.RawMessage) {
	t.Helper()
	var root any
	if err := json.Unmarshal(encoded, &root); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	var walk func(string, any)
	walk = func(path string, value any) {
		object, ok := value.(map[string]any)
		if !ok {
			t.Errorf("%s %s is not schema object", name, path)
			return
		}
		typeName, _ := object["type"].(string)
		switch typeName {
		case "object":
			if closed, ok := object["additionalProperties"].(bool); !ok || closed {
				t.Errorf("%s %s is not closed-world", name, path)
			}
			properties, ok := object["properties"].(map[string]any)
			if !ok {
				t.Errorf("%s %s lacks properties", name, path)
				return
			}
			for property, child := range properties {
				walk(path+"."+property, child)
			}
		case "array":
			walk(path+"[]", object["items"])
		case "string", "integer", "boolean":
		default:
			t.Errorf("%s %s unsupported type %q", name, path, typeName)
		}
	}
	walk("$", root)
}

func v20AssertRBACPermission(t *testing.T, raw string) {
	t.Helper()
	permission := identity.Permission(raw)
	if err := identity.ValidatePermission(permission); err != nil {
		t.Fatalf("registry permission %q unknown: %v", raw, err)
	}
	roles := []identity.Role{
		identity.RolePlatformAdmin, identity.RoleProjectOwner, identity.RoleProjectAdmin,
		identity.RoleContributor, identity.RoleReviewer, identity.RoleOperator, identity.RoleViewer,
	}
	allowed := v20AllowedRoles()[permission]
	if allowed == nil {
		t.Fatalf("V20 permission %q has no RBAC ratchet", raw)
	}
	for _, role := range roles {
		if got := identity.RoleAllows(role, permission); got != allowed[role] {
			t.Errorf("RoleAllows(%q,%q)=%t want=%t", role, permission, got, allowed[role])
		}
	}
	if identity.RoleAllows(identity.Role("unknown"), permission) ||
		identity.RoleAllows(identity.RolePlatformAdmin, identity.Permission("unknown")) {
		t.Error("RBAC default deny regressed")
	}
}

func v20AllowedRoles() map[identity.Permission]map[identity.Role]bool {
	platform := map[identity.Role]bool{identity.RolePlatformAdmin: true, identity.RoleProjectOwner: true, identity.RoleProjectAdmin: true}
	all := map[identity.Role]bool{identity.RolePlatformAdmin: true, identity.RoleProjectOwner: true, identity.RoleProjectAdmin: true, identity.RoleContributor: true, identity.RoleReviewer: true, identity.RoleOperator: true, identity.RoleViewer: true}
	return map[identity.Permission]map[identity.Role]bool{
		identity.PermissionProjectMembershipManage: platform,
		identity.PermissionGoalsCreate:             {identity.RolePlatformAdmin: true, identity.RoleProjectOwner: true, identity.RoleProjectAdmin: true, identity.RoleContributor: true, identity.RoleOperator: true},
		identity.PermissionGoalsAmend:              {identity.RolePlatformAdmin: true, identity.RoleProjectOwner: true, identity.RoleProjectAdmin: true, identity.RoleContributor: true},
		identity.PermissionGoalsGet:                all,
		identity.PermissionGoalsList:               all,
		identity.PermissionGoalsDirect:             {identity.RolePlatformAdmin: true, identity.RoleProjectOwner: true, identity.RoleProjectAdmin: true, identity.RoleOperator: true},
		identity.PermissionEffectsApprove:          {identity.RolePlatformAdmin: true, identity.RoleProjectOwner: true, identity.RoleProjectAdmin: true, identity.RoleReviewer: true, identity.RoleOperator: true},
		identity.PermissionChangesIntegrate:        {identity.RolePlatformAdmin: true, identity.RoleProjectOwner: true, identity.RoleProjectAdmin: true, identity.RoleOperator: true},
		identity.PermissionCouncilSkip:             {identity.RolePlatformAdmin: true, identity.RoleProjectOwner: true, identity.RoleOperator: true},
		identity.PermissionArtifactsRead:           all,
		identity.PermissionProjectStatus:           all,
	}
}

func v20JSONEqual(left, right []byte) bool {
	var one, two any
	return json.Unmarshal(left, &one) == nil && json.Unmarshal(right, &two) == nil && reflect.DeepEqual(one, two)
}

func TestInterfacesDependOnDispatcherAndNeverWriteLifecycle(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v20Fixture](t, filepath.Join(repositoryRoot, v20FixturePath))
	lifecycleCalls := make(map[string]bool, len(fixture.ForbiddenInterfaceMarkers))
	for _, marker := range fixture.ForbiddenInterfaceMarkers {
		lifecycleCalls[strings.TrimSuffix(marker, "(")] = true
	}
	for _, relative := range []string{
		"internal/interfaces/httpapi", "internal/interfaces/mcp", "internal/interfaces/cli",
	} {
		importsDispatcher := false
		v20WalkGo(t, filepath.Join(repositoryRoot, filepath.FromSlash(relative)), false, func(path string, file *ast.File) {
			for _, specification := range file.Imports {
				importPath, err := strconv.Unquote(specification.Path.Value)
				if err != nil {
					t.Fatalf("%s: %v", path, err)
				}
				if importPath == "orquesta/internal/commands" {
					importsDispatcher = true
				}
				for _, forbidden := range fixture.ForbiddenInterfaceImports {
					if importPath == forbidden || strings.HasPrefix(importPath, forbidden+"/") {
						t.Errorf("%s imports lifecycle authority %s", path, importPath)
					}
				}
			}
			ast.Inspect(file, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				name := ""
				switch callee := call.Fun.(type) {
				case *ast.Ident:
					name = callee.Name
				case *ast.SelectorExpr:
					name = callee.Sel.Name
				}
				if lifecycleCalls[name] {
					t.Errorf("%s calls lifecycle writer %s", path, name)
				}
				return true
			})
		})
		if !importsDispatcher {
			t.Errorf("%s does not depend on the canonical command dispatcher", relative)
		}
	}

	v20WalkGo(t, filepath.Join(repositoryRoot, "sdk", "commands"), false, func(path string, file *ast.File) {
		for _, specification := range file.Imports {
			importPath, err := strconv.Unquote(specification.Path.Value)
			if err != nil {
				t.Fatalf("%s: %v", path, err)
			}
			if strings.HasPrefix(importPath, "orquesta/internal/") {
				t.Errorf("%s SDK bypasses dispatcher boundary through %s", path, importPath)
			}
		}
	})
}

func TestSealedDependencyReceiptAndSourceOIDGateSchemaFreeze(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixturePath := filepath.Join(repositoryRoot, v20FixturePath)
	fixture := evidenceDecodeStrictJSON[v20Fixture](t, fixturePath)
	receipt := evidenceDecodeStrictJSON[evidenceReceiptV3](
		t, filepath.Join(repositoryRoot, filepath.FromSlash(fixture.PendingDependencyReceipt)),
	)
	gate := fixture.DependencyActivation
	if gate.Contract != receipt.Contract || gate.Result != receipt.Result ||
		gate.SourceWorktreeState != receipt.Execution.SourceWorktreeState ||
		gate.SealedOID != receipt.SealedSource.GitCommitOID ||
		gate.SealedOID != receipt.Execution.SourceGitCommitOID ||
		gate.EvidenceOID != fixture.TrustedBaseGitCommitOID ||
		gate.IntegrationHeadOID != fixture.TrustedBaseGitCommitOID {
		t.Fatalf("dependency gate does not bind the sealed receipt: gate=%+v receipt=%+v", gate, receipt)
	}
	for name, oid := range map[string]string{
		"product": gate.ProductOID, "sealed": gate.SealedOID,
		"evidence": gate.EvidenceOID, "integration": gate.IntegrationHeadOID,
	} {
		if len(oid) != 40 || strings.Trim(oid, "0123456789abcdef") != "" {
			t.Fatalf("%s oid=%q is not a frozen SHA-1 object identity", name, oid)
		}
		if _, err := evidenceGitCanonicalCommit(repositoryRoot, oid); err != nil {
			t.Fatalf("%s oid: %v", name, err)
		}
	}
	sealedParent, err := evidenceGit(repositoryRoot, "rev-parse", gate.SealedOID+"^")
	if err != nil || strings.TrimSpace(string(sealedParent)) != gate.ProductOID {
		t.Fatalf("sealed source parent=%q err=%v want product=%s", sealedParent, err, gate.ProductOID)
	}
	evidenceParent, err := evidenceGit(repositoryRoot, "rev-parse", gate.EvidenceOID+"^")
	if err != nil || strings.TrimSpace(string(evidenceParent)) != gate.SealedOID {
		t.Fatalf("evidence source parent=%q err=%v want sealed=%s", evidenceParent, err, gate.SealedOID)
	}
	evidenceAssertReceiptV3(t, repositoryRoot, evidenceReceiptV3Expectation{
		Contract: gate.Contract, FixturePath: "acceptance/" + v19FixturePath,
		ReceiptPath: fixture.PendingDependencyReceipt, ExecutedNotBefore: "2026-07-23T00:00:00Z",
		TrustedBaseGitCommitOID: "6f244a7594141c74dc28e095e0e5e32a05102826",
	})

	for name, document := range map[string]struct {
		path   string
		target any
	}{
		"fixture": {fixturePath, &v20Fixture{}},
		"registry": {
			filepath.Join(repositoryRoot, filepath.FromSlash(fixture.CanonicalRegistryPath)),
			&v20RegistryDocument{},
		},
	} {
		body, readErr := os.ReadFile(document.path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		var object map[string]any
		if json.Unmarshal(body, &object) != nil {
			t.Fatalf("%s is not JSON", name)
		}
		object["unsealed_schema_extension"] = true
		mutated, _ := json.Marshal(object)
		decoder := json.NewDecoder(bytes.NewReader(mutated))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(document.target); err == nil {
			t.Fatalf("%s schema accepted an undeclared field", name)
		}
	}
}

func TestRuntimeUsesGeneratedRegistryAndNeverLoadsMutableRegistryPath(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v20Fixture](t, filepath.Join(repositoryRoot, v20FixturePath))
	if fixture.RuntimeRegistryMode != "compiled_generated_only" || commandcore.RegistrySourceSHA256 == "" {
		t.Fatalf("runtime mode=%q digest=%q", fixture.RuntimeRegistryMode, commandcore.RegistrySourceSHA256)
	}
	commandsRoot := filepath.Join(repositoryRoot, "internal", "commands")
	forbiddenImports := map[string]bool{"embed": true, "io/fs": true, "os": true, "path/filepath": true}
	v20WalkGo(t, commandsRoot, false, func(path string, file *ast.File) {
		if strings.Contains(filepath.ToSlash(path), "/internal/commands/cmd/") {
			return
		}
		for _, specification := range file.Imports {
			importPath, err := strconv.Unquote(specification.Path.Value)
			if err != nil {
				t.Fatalf("%s: %v", path, err)
			}
			if forbiddenImports[importPath] {
				t.Errorf("runtime command package imports mutable registry capability %s in %s", importPath, path)
			}
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(content, []byte("registry.json")) {
			t.Errorf("runtime source %s names mutable registry path", path)
		}
	})
	generatedPath := filepath.Join(repositoryRoot, "internal", "commands", "definitions_generated.go")
	generated, err := os.ReadFile(generatedPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(generated, []byte("// Code generated by commandgen; DO NOT EDIT.")) ||
		!bytes.Contains(generated, []byte(`const RegistrySourceSHA256 = "`+commandcore.RegistrySourceSHA256+`"`)) {
		t.Fatal("compiled registry lacks matching generated source identity")
	}
	dispatcherSource, err := os.ReadFile(filepath.Join(commandsRoot, "dispatcher.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(dispatcherSource, []byte("cloneDefinitions(compiledDefinitions)")) ||
		!bytes.Contains(dispatcherSource, []byte("RegistryDigest: registryAdmissionDigest(definition)")) {
		t.Fatal("dispatcher is not bound to generated definitions and stable admission identity")
	}
}
