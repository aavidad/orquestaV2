package acceptance_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func v18AssertKnownDependencySeams(t *testing.T, repositoryRoot string) {
	t.Helper()
	wantTypes := []v18TypeContract{
		{Directory: "internal/ports", Name: "AgentLaunchReceipt", Fields: []string{
			"ExecutionRef", "GoalRef", "WorkItemRef", "PlanGeneration", "AppSpecGeneration",
			"ExecutionAttempt", "SpecHash", "ReceiptRef",
		}},
		{Directory: "internal/application", Name: "ExecutionRecord", Fields: []string{"LaunchReceiptRef"}},
		{Directory: "internal/application", Name: "ChangeSet", Fields: []string{
			"Ref", "ExecutionRef", "ExecutionAttempt", "PlanGeneration", "AppSpecGeneration", "SpecHash",
			"TreeOID", "DiffDigest", "WriteSetDigest", "ParentChangeRef",
		}},
		{Directory: "internal/application", Name: "IntegrateChangeRequest", Fields: []string{
			"RequestRef", "GoalRef", "ChangeRef", "ExpectedTargetOID",
		}},
	}
	for _, required := range wantTypes {
		v18RequireProductionType(t, filepath.Join(repositoryRoot, required.Directory), required, "V18 dependency")
	}
	goalSource := v16ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "goal"))
	for _, marker := range []string{"ReworkOf", "ApplyReplan"} {
		if !strings.Contains(goalSource, marker) {
			t.Errorf("V18 dependency seam %s missing", marker)
		}
	}
}

func v18AssertProductModel(t *testing.T, repositoryRoot string, fixture v18Fixture) {
	t.Helper()
	v18RequireProductionType(t, filepath.Join(repositoryRoot, fixture.SubjectContract.Directory), fixture.SubjectContract, "V18_RED")
	functions, methods := v18ProductionFunctions(
		t, filepath.Join(repositoryRoot, fixture.SubjectContract.Directory), fixture.SubjectContract.Name,
	)
	if !functions[fixture.SubjectContract.Constructor] {
		t.Errorf("V18_RED %s constructor missing", fixture.SubjectContract.Constructor)
	}
	if !methods[fixture.SubjectContract.DigestMethod] {
		t.Errorf("V18_RED %s.%s missing", fixture.SubjectContract.Name, fixture.SubjectContract.DigestMethod)
	}
	for _, required := range fixture.RequiredTypeContracts {
		v18RequireProductionType(t, filepath.Join(repositoryRoot, required.Directory), required, "V18_RED")
	}
}

func v18RequireProductionType(t *testing.T, directory string, required v18TypeContract, prefix string) {
	t.Helper()
	fields, found := v18ProductionTypeFields(t, directory, required.Name)
	if !found {
		t.Errorf("%s production type %s missing", prefix, required.Name)
		return
	}
	for _, field := range required.Fields {
		if !fields[field] {
			t.Errorf("%s %s lacks %s", prefix, required.Name, field)
		}
	}
}

func v18ProductionTypeFields(t *testing.T, directory, typeName string) (map[string]bool, bool) {
	t.Helper()
	for _, filename := range v16ProductionGoFiles(t, directory, false) {
		parsed, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", filename, err)
		}
		for _, declaration := range parsed.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.TYPE {
				continue
			}
			for _, specification := range general.Specs {
				typeSpec, ok := specification.(*ast.TypeSpec)
				if !ok || typeSpec.Name.Name != typeName {
					continue
				}
				structure, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					return nil, true
				}
				fields := make(map[string]bool)
				for _, field := range structure.Fields.List {
					for _, name := range field.Names {
						fields[name.Name] = true
					}
				}
				return fields, true
			}
		}
	}
	return nil, false
}

func v18ProductionFunctions(t *testing.T, directory, receiverType string) (map[string]bool, map[string]bool) {
	t.Helper()
	functions, methods := map[string]bool{}, map[string]bool{}
	for _, filename := range v16ProductionGoFiles(t, directory, false) {
		parsed, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", filename, err)
		}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if function.Recv == nil {
				functions[function.Name.Name] = true
				continue
			}
			typeName := function.Recv.List[0].Type
			if pointer, ok := typeName.(*ast.StarExpr); ok {
				typeName = pointer.X
			}
			if identifier, ok := typeName.(*ast.Ident); ok && identifier.Name == receiverType {
				methods[function.Name.Name] = true
			}
		}
	}
	return functions, methods
}

func v18AssertFlowAuthority(t *testing.T, repositoryRoot string, fixture v18Fixture) {
	t.Helper()
	reviewSource := v16ReadProductionGoOptional(t, filepath.Join(repositoryRoot, "internal", "review"))
	for _, marker := range fixture.RequiredReviewMarkers {
		if !strings.Contains(reviewSource, marker) {
			t.Errorf("V18_RED review domain lacks %q", marker)
		}
	}
	applicationSource := v16ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "application"))
	for _, marker := range fixture.RequiredApplicationMarkers {
		if !strings.Contains(applicationSource, marker) {
			t.Errorf("V18_RED application flow lacks %q", marker)
		}
	}
	authoritySource := reviewSource + applicationSource + v16ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "ports"))
	for _, forbidden := range fixture.ForbiddenPrivateAuthorities {
		if strings.Contains(authoritySource, "type "+forbidden+" ") {
			t.Errorf("V18 private authority %s is forbidden", forbidden)
		}
	}
}

func v18AssertBehaviorTests(t *testing.T, repositoryRoot string, fixture v18Fixture) {
	t.Helper()
	testSource := v18ReadGoTestsOptional(t,
		filepath.Join(repositoryRoot, "internal", "review"),
		filepath.Join(repositoryRoot, "internal", "application"),
		filepath.Join(repositoryRoot, "internal", "adapters", "state", "sqlite"),
		filepath.Join(repositoryRoot, "internal", "bootstrap"),
	)
	for _, name := range fixture.RequiredBehaviorTests {
		if count := strings.Count(testSource, "func "+name+"("); count != 1 {
			t.Errorf("V18_RED executable behavior test %s definitions=%d, want 1", name, count)
		}
	}
}

func v18ReadGoTestsOptional(t *testing.T, directories ...string) string {
	t.Helper()
	var source strings.Builder
	for _, directory := range directories {
		for _, filename := range v16ProductionGoFiles(t, directory, true) {
			content, err := os.ReadFile(filename)
			if err != nil {
				t.Fatalf("read %s: %v", filename, err)
			}
			source.Write(content)
			source.WriteByte('\n')
		}
	}
	return source.String()
}
