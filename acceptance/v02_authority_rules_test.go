package acceptance_test

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const v02FixturePath = "acceptance/fixtures/v02_authority_rules.json"

type v02Fixture struct {
	SchemaVersion           int                `json:"schema_version"`
	ReceiptSchemaVersion    int                `json:"receipt_schema_version"`
	ContractID              string             `json:"contract_id"`
	TrustedBaseGitCommitOID string             `json:"trusted_base_git_commit_oid"`
	Command                 string             `json:"command"`
	ExecutionArgv           []string           `json:"execution_argv"`
	OutputPath              string             `json:"output_path"`
	ReceiptPath             string             `json:"receipt_path"`
	CandidateSubjects       []string           `json:"candidate_subjects"`
	ProductModule           string             `json:"product_module"`
	ProductRoots            []string           `json:"product_roots"`
	LegacyImportPrefixes    []string           `json:"legacy_import_prefixes"`
	FrozenSurfaces          []v02FrozenSurface `json:"frozen_surfaces"`
	Lifecycle               v02Lifecycle       `json:"lifecycle"`
}

type v02FrozenSurface struct {
	Root            string   `json:"root"`
	ExcludePrefixes []string `json:"exclude_prefixes"`
	FileCount       int      `json:"file_count"`
	Digest          string   `json:"digest"`
}

type v02Lifecycle struct {
	Package               string   `json:"package"`
	Type                  string   `json:"type"`
	StateType             string   `json:"state_type"`
	WorkItemType          string   `json:"work_item_type"`
	RequiredFields        []string `json:"required_fields"`
	WriterPackage         string   `json:"writer_package"`
	WriterType            string   `json:"writer_type"`
	ExecutionPackage      string   `json:"execution_package"`
	ExecutionType         string   `json:"execution_type"`
	ExecutionFields       []string `json:"execution_required_fields"`
	AgentPortType         string   `json:"agent_port_type"`
	ForbiddenGoalTerms    []string `json:"forbidden_goal_field_terms"`
	MutationMethods       []string `json:"mutation_methods"`
	ReadyMethod           string   `json:"ready_method"`
	SchedulerMethod       string   `json:"scheduler_method"`
	ClaimMethod           string   `json:"claim_method"`
	AllowedSchedulerTypes []string `json:"allowed_scheduler_types"`
}

type v02GoFile struct {
	Path        string
	PackagePath string
	Syntax      *ast.File
	Test        bool
}

type v02SourceSet struct {
	Root       string
	Module     string
	FileSet    *token.FileSet
	Files      []v02GoFile
	ByPackage  map[string][]*ast.File
	PackageDir map[string]string
}

func TestAcceptanceV02AuthorityRules(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v02Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v02FixturePath)))
	if fixture.SchemaVersion != 1 || fixture.ReceiptSchemaVersion != 3 || fixture.ContractID != "AC-V02-AUTHORITY-RULES" ||
		fixture.TrustedBaseGitCommitOID != "a8bff609492f312fe2d6bf8ccccde02b6e5c8426" ||
		fixture.Command != "go test -mod=vendor -count=1 ./acceptance -run '^TestAcceptanceV02AuthorityRules$'" ||
		!stringSlicesEqual(fixture.ExecutionArgv, []string{"go", "test", "-mod=vendor", "-count=1", "./acceptance", "-run", "^TestAcceptanceV02AuthorityRules$"}) ||
		fixture.OutputPath != "product/evidence/v02_authority_rules.output.txt" || fixture.ReceiptPath != "product/evidence/v02_authority_rules.json" ||
		fixture.ProductModule != "orquesta" || len(fixture.ProductRoots) == 0 || len(fixture.CandidateSubjects) == 0 {
		t.Fatalf("invalid V02 fixture header: %+v", fixture)
	}

	t.Run("legacy_surfaces_match_frozen_fixture", func(t *testing.T) {
		for _, surface := range fixture.FrozenSurfaces {
			count, digest := v02SurfaceDigest(t, repositoryRoot, surface)
			if count != surface.FileCount || digest != surface.Digest {
				t.Errorf("frozen surface %s changed: files=%d digest=%s; want files=%d digest=%s", surface.Root, count, digest, surface.FileCount, surface.Digest)
			}
		}
	})

	sources := v02LoadSources(t, repositoryRoot, fixture.ProductModule, fixture.ProductRoots)
	t.Run("new_product_has_no_legacy_import", func(t *testing.T) {
		for _, file := range sources.Files {
			for _, spec := range file.Syntax.Imports {
				importPath, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					t.Fatalf("%s: invalid import literal: %v", file.Path, err)
				}
				for _, prefix := range fixture.LegacyImportPrefixes {
					if importPath == prefix || strings.HasPrefix(importPath, strings.TrimSuffix(prefix, "/")+"/") {
						position := sources.FileSet.Position(spec.Pos())
						t.Errorf("%s:%d imports frozen legacy package %q", file.Path, position.Line, importPath)
					}
				}
			}
		}
	})

	t.Run("goal_is_the_only_lifecycle_authority", func(t *testing.T) {
		v02AssertUniqueGoal(t, sources, fixture.Lifecycle)
	})

	t.Run("execution_and_provider_identity_stay_outside_goal", func(t *testing.T) {
		v02AssertExecutionAndProviderSeparation(t, sources, fixture.Lifecycle)
	})

	t.Run("application_orchestrator_is_the_only_writer_and_scheduler", func(t *testing.T) {
		v02AssertSingleWriterAndScheduler(t, sources, fixture.Lifecycle)
	})
}

func TestAcceptanceV02AuthorityRulesReceipt(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	evidenceAssertReceiptV3(t, repositoryRoot, evidenceReceiptV3Expectation{
		Contract:                "AC-V02-AUTHORITY-RULES",
		FixturePath:             v02FixturePath,
		ReceiptPath:             "product/evidence/v02_authority_rules.json",
		ExecutedNotBefore:       "2026-07-14T00:00:00+02:00",
		TrustedBaseGitCommitOID: "a8bff609492f312fe2d6bf8ccccde02b6e5c8426",
	})
}

func v02SurfaceDigest(t *testing.T, repositoryRoot string, surface v02FrozenSurface) (int, string) {
	t.Helper()
	root := filepath.Join(repositoryRoot, filepath.FromSlash(surface.Root))
	excluded := make(map[string]struct{}, len(surface.ExcludePrefixes))
	for _, prefix := range surface.ExcludePrefixes {
		excluded[strings.TrimSuffix(filepath.ToSlash(prefix), "/")] = struct{}{}
	}
	var files []string
	err := filepath.WalkDir(root, func(filename string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(repositoryRoot, filename)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if v02PathExcluded(relative, excluded) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return errors.New("symlink forbidden in frozen surface: " + relative)
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return errors.New("non-regular file in frozen surface: " + relative)
		}
		files = append(files, relative)
		return nil
	})
	if err != nil {
		t.Fatalf("scan frozen surface %s: %v", surface.Root, err)
	}
	sort.Strings(files)
	digest := sha256.New()
	for _, relative := range files {
		absolute := filepath.Join(repositoryRoot, filepath.FromSlash(relative))
		info, err := os.Stat(absolute)
		if err != nil {
			t.Fatalf("stat frozen file %s: %v", relative, err)
		}
		evidenceWriteFrame(t, digest, []byte(relative))
		executable := byte(0)
		if info.Mode()&0o111 != 0 {
			executable = 1
		}
		evidenceWriteFrame(t, digest, []byte{executable})
		handle, err := os.Open(absolute)
		if err != nil {
			t.Fatalf("open frozen file %s: %v", relative, err)
		}
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(info.Size()))
		if _, err := digest.Write(size[:]); err != nil {
			handle.Close()
			t.Fatalf("hash frozen file size %s: %v", relative, err)
		}
		if _, err := io.Copy(digest, handle); err != nil {
			handle.Close()
			t.Fatalf("hash frozen file %s: %v", relative, err)
		}
		if err := handle.Close(); err != nil {
			t.Fatalf("close frozen file %s: %v", relative, err)
		}
	}
	return len(files), "sha256:" + hex.EncodeToString(digest.Sum(nil))
}

func v02PathExcluded(relative string, excluded map[string]struct{}) bool {
	for prefix := range excluded {
		if relative == prefix || strings.HasPrefix(relative, prefix+"/") {
			return true
		}
	}
	return false
}

func v02LoadSources(t *testing.T, repositoryRoot, module string, roots []string) v02SourceSet {
	t.Helper()
	sources := v02SourceSet{
		Root: repositoryRoot, Module: module, FileSet: token.NewFileSet(),
		ByPackage: make(map[string][]*ast.File), PackageDir: make(map[string]string),
	}
	seen := make(map[string]struct{})
	for _, sourceRoot := range roots {
		absoluteRoot := filepath.Join(repositoryRoot, filepath.FromSlash(sourceRoot))
		err := filepath.WalkDir(absoluteRoot, func(filename string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&fs.ModeSymlink != 0 {
				return errors.New("symlink forbidden in product source: " + filename)
			}
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
				return nil
			}
			relative, err := filepath.Rel(repositoryRoot, filename)
			if err != nil {
				return err
			}
			relative = filepath.ToSlash(relative)
			if _, duplicate := seen[relative]; duplicate {
				return nil
			}
			seen[relative] = struct{}{}
			syntax, err := parser.ParseFile(sources.FileSet, filename, nil, parser.AllErrors)
			if err != nil {
				return err
			}
			directory := path.Dir(relative)
			packagePath := module + "/" + directory
			file := v02GoFile{Path: relative, PackagePath: packagePath, Syntax: syntax, Test: strings.HasSuffix(relative, "_test.go")}
			sources.Files = append(sources.Files, file)
			if !file.Test {
				sources.ByPackage[packagePath] = append(sources.ByPackage[packagePath], syntax)
				sources.PackageDir[packagePath] = filepath.Join(repositoryRoot, filepath.FromSlash(directory))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan product root %s: %v", sourceRoot, err)
		}
	}
	sort.Slice(sources.Files, func(i, j int) bool { return sources.Files[i].Path < sources.Files[j].Path })
	return sources
}

func v02AssertUniqueGoal(t *testing.T, sources v02SourceSet, lifecycle v02Lifecycle) {
	t.Helper()
	var goalTypes, stateTypes int
	goalFields := make(map[string]struct{})
	for _, file := range sources.Files {
		if file.Test {
			continue
		}
		for _, declaration := range file.Syntax.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.TYPE {
				continue
			}
			for _, spec := range general.Specs {
				typeSpec := spec.(*ast.TypeSpec)
				qualified := file.PackagePath + "." + typeSpec.Name.Name
				if v02VersionedGoalType(typeSpec.Name.Name) {
					t.Errorf("%s declares forbidden parallel Goal generation %s", file.Path, qualified)
				}
				if file.PackagePath == lifecycle.Package && typeSpec.Name.Name == lifecycle.Type {
					goalTypes++
					structure, ok := typeSpec.Type.(*ast.StructType)
					if !ok {
						t.Errorf("%s must be a struct aggregate", qualified)
						continue
					}
					for _, field := range structure.Fields.List {
						for _, name := range field.Names {
							goalFields[name.Name] = struct{}{}
						}
					}
				} else if typeSpec.Name.Name == lifecycle.Type {
					t.Errorf("second Goal aggregate declared at %s", qualified)
				}
				if file.PackagePath == lifecycle.Package && typeSpec.Name.Name == lifecycle.StateType {
					stateTypes++
				} else if typeSpec.Name.Name == lifecycle.StateType {
					t.Errorf("second Goal lifecycle state declared at %s", qualified)
				}
			}
		}
	}
	if goalTypes != 1 || stateTypes != 1 {
		t.Errorf("Goal authority declarations: Goal=%d GoalState=%d, want 1/1", goalTypes, stateTypes)
	}
	for _, field := range lifecycle.RequiredFields {
		if _, exists := goalFields[field]; !exists {
			t.Errorf("Goal aggregate lacks required authority field %q", field)
		}
	}
}

var v02ParallelGoalPattern = regexp.MustCompile(`(?i)(goal.*(?:next|v[0-9]+)|(?:next|v[0-9]+).*goal)`)

func v02VersionedGoalType(name string) bool {
	return v02ParallelGoalPattern.MatchString(name)
}

func v02AssertExecutionAndProviderSeparation(t *testing.T, sources v02SourceSet, lifecycle v02Lifecycle) {
	t.Helper()
	goalSpec, goalFile := v02FindType(t, sources, lifecycle.Package, lifecycle.Type)
	workItemSpec, workItemFile := v02FindType(t, sources, lifecycle.Package, lifecycle.WorkItemType)
	for _, subject := range []struct {
		spec *ast.TypeSpec
		file v02GoFile
	}{{goalSpec, goalFile}, {workItemSpec, workItemFile}} {
		structure, ok := subject.spec.Type.(*ast.StructType)
		if !ok {
			t.Errorf("%s.%s must remain a struct", subject.file.PackagePath, subject.spec.Name.Name)
			continue
		}
		for _, field := range structure.Fields.List {
			var names []string
			for _, name := range field.Names {
				names = append(names, name.Name)
			}
			var typeNames []string
			ast.Inspect(field.Type, func(node ast.Node) bool {
				if identifier, ok := node.(*ast.Ident); ok {
					typeNames = append(typeNames, identifier.Name)
				}
				return true
			})
			search := strings.ToLower(strings.Join(append(names, typeNames...), " "))
			for _, forbidden := range lifecycle.ForbiddenGoalTerms {
				if strings.Contains(search, strings.ToLower(forbidden)) {
					position := sources.FileSet.Position(field.Pos())
					t.Errorf("%s:%d embeds concrete provider/runtime concern %q in %s", subject.file.Path, position.Line, forbidden, subject.spec.Name.Name)
				}
			}
			if strings.Contains(search, strings.ToLower(lifecycle.ExecutionType)) {
				position := sources.FileSet.Position(field.Pos())
				t.Errorf("%s:%d embeds replaceable %s projection in %s authority", subject.file.Path, position.Line, lifecycle.ExecutionType, subject.spec.Name.Name)
			}
		}
	}

	executionSpec, _ := v02FindType(t, sources, lifecycle.ExecutionPackage, lifecycle.ExecutionType)
	execution, ok := executionSpec.Type.(*ast.StructType)
	if !ok {
		t.Fatalf("%s.%s must be a projection struct", lifecycle.ExecutionPackage, lifecycle.ExecutionType)
	}
	executionFields := make(map[string]struct{})
	for _, field := range execution.Fields.List {
		for _, name := range field.Names {
			executionFields[name.Name] = struct{}{}
		}
	}
	for _, required := range lifecycle.ExecutionFields {
		if _, exists := executionFields[required]; !exists {
			t.Errorf("%s lacks attempt/projection field %q", lifecycle.ExecutionType, required)
		}
	}

	portSpec, _ := v02FindType(t, sources, lifecycle.WriterPackage, lifecycle.AgentPortType)
	contract, ok := portSpec.Type.(*ast.InterfaceType)
	if !ok {
		t.Fatalf("%s.%s must be an outbound interface", lifecycle.WriterPackage, lifecycle.AgentPortType)
	}
	methods := make(map[string]struct{})
	for _, method := range contract.Methods.List {
		for _, name := range method.Names {
			methods[name.Name] = struct{}{}
		}
	}
	for _, required := range []string{"Capabilities", "Launch"} {
		if _, exists := methods[required]; !exists {
			t.Errorf("%s lacks provider boundary method %q", lifecycle.AgentPortType, required)
		}
	}

	for _, file := range sources.Files {
		if file.Test {
			continue
		}
		for _, spec := range file.Syntax.Imports {
			importPath, _ := strconv.Unquote(spec.Path.Value)
			if file.PackagePath == lifecycle.Package && strings.HasPrefix(importPath, sources.Module+"/internal/") {
				position := sources.FileSet.Position(spec.Pos())
				t.Errorf("%s:%d Goal domain imports concrete/internal boundary %q", file.Path, position.Line, importPath)
			}
			if file.PackagePath != lifecycle.WriterPackage || !strings.HasPrefix(importPath, sources.Module+"/internal/") {
				continue
			}
			if importPath == lifecycle.Package || importPath == sources.Module+"/internal/identity" || importPath == sources.Module+"/internal/ports" {
				continue
			}
			position := sources.FileSet.Position(spec.Pos())
			t.Errorf("%s:%d application authority imports outward/concrete package %q", file.Path, position.Line, importPath)
		}
	}
}

func v02FindType(t *testing.T, sources v02SourceSet, packagePath, name string) (*ast.TypeSpec, v02GoFile) {
	t.Helper()
	var found *ast.TypeSpec
	var foundFile v02GoFile
	for _, file := range sources.Files {
		if file.Test || file.PackagePath != packagePath {
			continue
		}
		for _, declaration := range file.Syntax.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.TYPE {
				continue
			}
			for _, raw := range general.Specs {
				spec := raw.(*ast.TypeSpec)
				if spec.Name.Name != name {
					continue
				}
				if found != nil {
					t.Fatalf("duplicate type %s.%s", packagePath, name)
				}
				found, foundFile = spec, file
			}
		}
	}
	if found == nil {
		t.Fatalf("missing type %s.%s", packagePath, name)
	}
	return found, foundFile
}

type v02TypeChecker struct {
	sources  v02SourceSet
	cache    map[string]*types.Package
	infos    map[string]*types.Info
	errors   map[string][]error
	checking map[string]bool
	fallback types.Importer
}

func v02AssertSingleWriterAndScheduler(t *testing.T, sources v02SourceSet, lifecycle v02Lifecycle) {
	t.Helper()
	checker := &v02TypeChecker{
		sources: sources, cache: make(map[string]*types.Package), infos: make(map[string]*types.Info),
		errors: make(map[string][]error), checking: make(map[string]bool), fallback: importer.Default(),
	}
	for packagePath := range sources.ByPackage {
		_, _ = checker.Import(packagePath)
	}
	for _, packagePath := range []string{lifecycle.Package, lifecycle.WriterPackage} {
		if len(checker.errors[packagePath]) != 0 {
			t.Fatalf("type-check authority package %s: %v", packagePath, checker.errors[packagePath][0])
		}
	}

	mutationSet := v02StringSet(lifecycle.MutationMethods)
	declaredMutations := make(map[string]struct{})
	seenMutations := make(map[string]struct{})
	seenSchedulerOperations := make(map[string]struct{})
	schedulerDeclarations := 0
	allowedSchedulerTypes := v02StringSet(lifecycle.AllowedSchedulerTypes)

	for _, file := range sources.Files {
		if file.Test {
			continue
		}
		info := checker.infos[file.PackagePath]
		for _, declaration := range file.Syntax.Decls {
			switch typed := declaration.(type) {
			case *ast.GenDecl:
				if typed.Tok != token.TYPE {
					continue
				}
				for _, spec := range typed.Specs {
					typeSpec := spec.(*ast.TypeSpec)
					if !strings.Contains(strings.ToLower(typeSpec.Name.Name), "scheduler") {
						continue
					}
					qualified := file.PackagePath + "." + typeSpec.Name.Name
					if _, allowed := allowedSchedulerTypes[qualified]; !allowed {
						t.Errorf("parallel scheduler type forbidden: %s", qualified)
					}
				}
			case *ast.FuncDecl:
				receiver := v02ReceiverName(info, typed)
				if file.PackagePath == lifecycle.Package && receiver == lifecycle.Type && v02ReturnsGoalAndError(typed, lifecycle.Type) {
					declaredMutations[typed.Name.Name] = struct{}{}
				}
				if typed.Name.Name == lifecycle.SchedulerMethod {
					schedulerDeclarations++
					if file.PackagePath != lifecycle.WriterPackage || receiver != lifecycle.WriterType {
						t.Errorf("scheduler authority %s.%s must belong to %s.%s", file.PackagePath, typed.Name.Name, lifecycle.WriterPackage, lifecycle.WriterType)
					}
				}
				v02InspectCalls(t, sources, file, typed, info, lifecycle, mutationSet, seenMutations, seenSchedulerOperations)
			}
		}
	}
	if !reflect.DeepEqual(declaredMutations, mutationSet) {
		t.Errorf("Goal mutation API = %v, frozen authority API = %v", v02SortedKeys(declaredMutations), v02SortedKeys(mutationSet))
	}
	if !reflect.DeepEqual(seenMutations, mutationSet) {
		t.Errorf("application mutation calls = %v, want every frozen mutation %v", v02SortedKeys(seenMutations), v02SortedKeys(mutationSet))
	}
	if schedulerDeclarations != 1 {
		t.Errorf("scheduler method declarations = %d, want 1", schedulerDeclarations)
	}
	wantSchedulerOperations := v02StringSet([]string{lifecycle.ReadyMethod, lifecycle.ClaimMethod})
	if !reflect.DeepEqual(seenSchedulerOperations, wantSchedulerOperations) {
		t.Errorf("application scheduler operations = %v, want %v", v02SortedKeys(seenSchedulerOperations), v02SortedKeys(wantSchedulerOperations))
	}
}

func (checker *v02TypeChecker) Import(importPath string) (*types.Package, error) {
	if cached := checker.cache[importPath]; cached != nil {
		return cached, nil
	}
	files := checker.sources.ByPackage[importPath]
	if len(files) == 0 {
		loaded, err := checker.fallback.Import(importPath)
		if err == nil {
			return loaded, nil
		}
		stub := types.NewPackage(importPath, path.Base(importPath))
		stub.MarkComplete()
		checker.cache[importPath] = stub
		return stub, nil
	}
	if checker.checking[importPath] {
		return nil, errors.New("local import cycle at " + importPath)
	}
	checker.checking[importPath] = true
	defer delete(checker.checking, importPath)
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue), Defs: make(map[*ast.Ident]types.Object),
		Uses: make(map[*ast.Ident]types.Object), Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}
	config := types.Config{
		Importer: checker, DisableUnusedImportCheck: true,
		Error: func(err error) { checker.errors[importPath] = append(checker.errors[importPath], err) },
	}
	checked, _ := config.Check(importPath, checker.sources.FileSet, files, info)
	if checked == nil {
		checked = types.NewPackage(importPath, files[0].Name.Name)
		checked.MarkComplete()
	}
	checker.cache[importPath] = checked
	checker.infos[importPath] = info
	return checked, nil
}

func v02InspectCalls(
	t *testing.T,
	sources v02SourceSet,
	file v02GoFile,
	function *ast.FuncDecl,
	info *types.Info,
	lifecycle v02Lifecycle,
	mutations map[string]struct{},
	seen map[string]struct{},
	seenSchedulerOperations map[string]struct{},
) {
	t.Helper()
	if function.Body == nil || info == nil {
		return
	}
	callerReceiver := v02ReceiverName(info, function)
	ast.Inspect(function.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		selection := info.Selections[selector]
		if selection == nil {
			return true
		}
		receiverPackage, receiverType := v02SelectionReceiver(selection)
		method := selection.Obj().Name()
		if receiverPackage == lifecycle.Package && receiverType == lifecycle.Type {
			if _, mutation := mutations[method]; mutation {
				seen[method] = struct{}{}
				if file.PackagePath != lifecycle.WriterPackage || callerReceiver != lifecycle.WriterType {
					position := sources.FileSet.Position(selector.Pos())
					t.Errorf("%s:%d invokes Goal.%s outside sole writer %s.%s", file.Path, position.Line, method, lifecycle.WriterPackage, lifecycle.WriterType)
				}
			}
			if method == lifecycle.ReadyMethod && file.PackagePath == lifecycle.WriterPackage && callerReceiver == lifecycle.WriterType {
				seenSchedulerOperations[method] = struct{}{}
			}
		}
		if method == lifecycle.ClaimMethod {
			if file.PackagePath != lifecycle.WriterPackage || callerReceiver != lifecycle.WriterType {
				position := sources.FileSet.Position(selector.Pos())
				t.Errorf("%s:%d claims scheduler work outside %s.%s", file.Path, position.Line, lifecycle.WriterPackage, lifecycle.WriterType)
			} else {
				seenSchedulerOperations[method] = struct{}{}
			}
		}
		return true
	})
}

func v02ReceiverName(info *types.Info, function *ast.FuncDecl) string {
	if info == nil || function.Recv == nil || len(function.Recv.List) != 1 {
		return ""
	}
	return v02NamedType(info.TypeOf(function.Recv.List[0].Type))
}

func v02SelectionReceiver(selection *types.Selection) (string, string) {
	receiver := selection.Recv()
	if pointer, ok := receiver.(*types.Pointer); ok {
		receiver = pointer.Elem()
	}
	named, ok := receiver.(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return "", ""
	}
	return named.Obj().Pkg().Path(), named.Obj().Name()
}

func v02NamedType(value types.Type) string {
	if pointer, ok := value.(*types.Pointer); ok {
		value = pointer.Elem()
	}
	if named, ok := value.(*types.Named); ok {
		return named.Obj().Name()
	}
	return ""
}

func v02ReturnsGoalAndError(function *ast.FuncDecl, goalType string) bool {
	if function.Type.Results == nil || len(function.Type.Results.List) != 2 {
		return false
	}
	first, firstOK := function.Type.Results.List[0].Type.(*ast.Ident)
	second, secondOK := function.Type.Results.List[1].Type.(*ast.Ident)
	return firstOK && secondOK && first.Name == goalType && second.Name == "error"
}

func v02StringSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func v02SortedKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
