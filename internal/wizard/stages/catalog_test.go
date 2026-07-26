package stages

import (
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestBuiltInCatalogContainsExactlySixCompleteTemplates(t *testing.T) {
	t.Parallel()

	catalog := BuiltIn()
	if got, want := catalog.Version().String(), currentVersionValue; got != want {
		t.Fatalf("version = %q, want %q", got, want)
	}
	templates := catalog.Templates()
	if got, want := len(templates), 6; got != want {
		t.Fatalf("templates = %d, want %d", got, want)
	}
	if got := templateRefStrings(templates); !reflect.DeepEqual(got, requiredTemplateValues) {
		t.Fatalf("template refs = %v, want %v", got, requiredTemplateValues)
	}

	for _, template := range templates {
		if got := roadmapRefStrings(template.RoadmapCapabilityRefs()); !reflect.DeepEqual(
			got,
			[]string{"WIZ-11", "WIZ-24"},
		) {
			t.Fatalf("%s roadmap refs = %v", template.Ref().String(), got)
		}
		if len(template.Stages()) < 3 {
			t.Fatalf("%s has fewer than three stages", template.Ref().String())
		}
		if len(template.Units()) < len(template.Stages()) {
			t.Fatalf("%s has fewer units than stages", template.Ref().String())
		}
		for _, unit := range template.Units() {
			if unit.Role().String() == "" || len(unit.WriteSet()) == 0 ||
				len(unit.RequiredTests()) == 0 ||
				len(unit.AcceptanceCriteria()) == 0 || len(unit.Effects()) == 0 {
				t.Fatalf("%s unit %s is incomplete", template.Ref().String(), unit.Ref().String())
			}
			if !validExecutionPolicy(unit.Policy()) {
				t.Fatalf("%s unit %s has invalid policy", template.Ref().String(), unit.Ref().String())
			}
		}
	}
}

func TestBuiltInTemplatesExposeExplicitGovernedEffects(t *testing.T) {
	t.Parallel()

	catalog := BuiltIn()
	assertTemplateHasEffect(t, catalog, "template:research", EffectRequestExternal, false)
	assertTemplateHasEffect(t, catalog, "template:build_app", EffectMutateWorkspace, false)
	assertTemplateHasEffect(t, catalog, "template:change_app", EffectMutateWorkspace, false)
	assertTemplateHasEffect(t, catalog, "template:domain_production", EffectPublish, true)
	assertTemplateHasEffect(t, catalog, "template:deploy", EffectMutateExternal, true)
	assertTemplateHasEffect(t, catalog, "template:self_change", EffectMutateSelf, true)
}

func TestCatalogAndNestedAccessorsReturnDefensiveCopies(t *testing.T) {
	t.Parallel()

	catalog := BuiltIn()
	first := catalog.Templates()
	originalRef := first[0].Ref()
	first[0] = Template{}
	if got := catalog.Templates()[0].Ref(); got != originalRef {
		t.Fatalf("catalog mutated through Templates: %q", got.String())
	}

	template, ok := catalog.Template(mustTemplateRef("template:build_app"))
	if !ok {
		t.Fatal("build_app missing")
	}
	stages := template.Stages()
	units := template.Units()
	originalStage := stages[0].Ref()
	originalUnit := units[0].Ref()
	stages[0].dependsOn = append(stages[0].dependsOn, mustStageRef("stage:test.fake"))
	units[0].dependsOn = append(units[0].dependsOn, mustUnitRef("unit:test.fake"))
	units[0].writeSet[0].Path = "mutated"
	units[0].requiredTests[0].CriterionRefs[0] = mustCriterionRef("criterion:test.fake")
	units[0].acceptanceCriteria[0] = Criterion{}
	units[0].effects[0] = Effect{}

	fresh, _ := catalog.Template(mustTemplateRef("template:build_app"))
	if fresh.Stages()[0].Ref() != originalStage ||
		fresh.Units()[0].Ref() != originalUnit ||
		fresh.Units()[0].WriteSet()[0].Path == "mutated" ||
		fresh.Units()[0].RequiredTests()[0].CriterionRefs[0].String() ==
			"criterion:test.fake" {
		t.Fatal("catalog mutated through nested accessor")
	}
}

func TestCatalogSupportsConcurrentReadAndCopyMutation(t *testing.T) {
	t.Parallel()

	catalog := BuiltIn()
	var workers sync.WaitGroup
	for worker := 0; worker < 32; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for attempt := 0; attempt < 50; attempt++ {
				templates := catalog.Templates()
				templates[0].units[0].writeSet[0].Path = "private_copy"
				fresh := catalog.Templates()
				if fresh[0].units[0].writeSet[0].Path == "private_copy" {
					t.Errorf("worker mutation reached catalog")
					return
				}
			}
		}()
	}
	workers.Wait()
}

func TestTemplateConstructionHasDeterministicOrder(t *testing.T) {
	t.Parallel()

	original, _ := BuiltIn().Template(mustTemplateRef("template:build_app"))
	stages := original.Stages()
	units := original.Units()
	reverseStages(stages)
	reverseUnits(units)

	reordered, err := NewTemplate(TemplateInput{
		Ref: original.Ref(), RoadmapCapabilityRefs: reverseRoadmapRefs(
			original.RoadmapCapabilityRefs(),
		),
		Stages: stages, Units: units,
	})
	if err != nil {
		t.Fatalf("rebuild template: %v", err)
	}
	if !reflect.DeepEqual(reordered, original) {
		t.Fatalf("template depends on input order\nreordered=%#v\noriginal=%#v", reordered, original)
	}

	templates := BuiltIn().Templates()
	reverseTemplates(templates)
	catalog, err := NewCatalog(CatalogInput{
		Version: CurrentVersion(), Templates: templates,
	})
	if err != nil {
		t.Fatalf("rebuild catalog: %v", err)
	}
	if got := templateRefStrings(catalog.Templates()); !reflect.DeepEqual(
		got,
		requiredTemplateValues,
	) {
		t.Fatalf("catalog order = %v", got)
	}
}

func TestTemplateRejectsMissingDependencyAndCycles(t *testing.T) {
	t.Parallel()

	base, _ := BuiltIn().Template(mustTemplateRef("template:change_app"))
	stages := base.Stages()
	stages[1].dependsOn = []StageRef{mustStageRef("stage:change_app.missing")}
	_, err := NewTemplate(TemplateInput{
		Ref: base.Ref(), RoadmapCapabilityRefs: base.RoadmapCapabilityRefs(),
		Stages: stages, Units: base.Units(),
	})
	assertErrorCode(t, err, ErrorReferenceNotFound)

	stages = base.Stages()
	stages[0].dependsOn = []StageRef{stages[1].ref}
	_, err = NewTemplate(TemplateInput{
		Ref: base.Ref(), RoadmapCapabilityRefs: base.RoadmapCapabilityRefs(),
		Stages: stages, Units: base.Units(),
	})
	assertErrorCode(t, err, ErrorDependencyCycle)
}

func TestTemplateRejectsUnorderedWriteSetConflict(t *testing.T) {
	t.Parallel()

	base, _ := BuiltIn().Template(mustTemplateRef("template:build_app"))
	units := base.Units()
	var domainIndex, interfacesIndex int
	for index := range units {
		switch units[index].Ref().String() {
		case "unit:build_app.implement_domain":
			domainIndex = index
		case "unit:build_app.implement_interfaces":
			interfacesIndex = index
		}
	}
	units[interfacesIndex].writeSet = []WriteScope{{Path: "source/domain/model"}}
	if units[domainIndex].stageRef != units[interfacesIndex].stageRef {
		t.Fatal("test units are not concurrent")
	}
	_, err := NewTemplate(TemplateInput{
		Ref: base.Ref(), RoadmapCapabilityRefs: base.RoadmapCapabilityRefs(),
		Stages: base.Stages(), Units: units,
	})
	assertErrorCode(t, err, ErrorWriteSetConflict)
}

func TestUnitRejectsEmptyTestsAndUnresolvedCriterion(t *testing.T) {
	t.Parallel()

	base, _ := BuiltIn().Template(mustTemplateRef("template:research"))
	unit := base.Units()[0]
	_, err := NewUnit(UnitInput{
		Ref: unit.Ref(), StageRef: unit.StageRef(), DependsOn: unit.DependsOn(),
		Role: unit.Role(), WriteSet: unit.WriteSet(),
		AcceptanceCriteria: unit.AcceptanceCriteria(), Effects: unit.Effects(),
		Policy: unit.Policy(),
	})
	assertErrorCode(t, err, ErrorInvalidArgument)

	tests := unit.RequiredTests()
	tests[0].CriterionRefs = []CriterionRef{
		mustCriterionRef("criterion:research.frame_question.missing"),
	}
	_, err = NewUnit(UnitInput{
		Ref: unit.Ref(), StageRef: unit.StageRef(), DependsOn: unit.DependsOn(),
		Role: unit.Role(), WriteSet: unit.WriteSet(), RequiredTests: tests,
		AcceptanceCriteria: unit.AcceptanceCriteria(), Effects: unit.Effects(),
		Policy: unit.Policy(),
	})
	assertErrorCode(t, err, ErrorReferenceNotFound)
}

func TestUnitRejectsUnsafeWritePathAndUnguardedExternalMutation(t *testing.T) {
	t.Parallel()

	base, _ := BuiltIn().Template(mustTemplateRef("template:deploy"))
	unit := base.Units()[0]
	input := unitInput(unit)
	input.WriteSet = []WriteScope{{Path: "../escape"}}
	_, err := NewUnit(input)
	assertErrorCode(t, err, ErrorInvalidArgument)

	input = unitInput(unit)
	input.Effects = []Effect{{
		Ref: mustEffectRef("effect:deploy.unsafe"), Kind: EffectMutateExternal,
	}}
	_, err = NewUnit(input)
	assertErrorCode(t, err, ErrorInvalidArgument)
}

func TestCatalogRejectsMissingAndConflictingTemplateDefinitions(t *testing.T) {
	t.Parallel()

	templates := BuiltIn().Templates()
	_, err := NewCatalog(CatalogInput{
		Version: CurrentVersion(), Templates: templates[:len(templates)-1],
	})
	assertErrorCode(t, err, ErrorIncompleteCatalog)

	templates = BuiltIn().Templates()
	duplicate := cloneTemplates([]Template{templates[0]})[0]
	duplicate.stages[0].phase = PhaseRelease
	templates[len(templates)-1] = duplicate
	_, err = NewCatalog(CatalogInput{
		Version: CurrentVersion(), Templates: templates,
	})
	assertErrorCode(t, err, ErrorConflictingDefinition)
}

func TestPackageImportsStayPureAndDoNotReachForbiddenLayers(t *testing.T) {
	t.Parallel()

	entries, err := parser.ParseDir(
		token.NewFileSet(),
		".",
		nil,
		parser.ImportsOnly,
	)
	if err != nil {
		t.Fatalf("parse package: %v", err)
	}
	for _, parsedPackage := range entries {
		for fileName, file := range parsedPackage.Files {
			if strings.HasSuffix(fileName, "_test.go") {
				continue
			}
			for _, importSpec := range file.Imports {
				path, unquoteErr := strconv.Unquote(importSpec.Path.Value)
				if unquoteErr != nil {
					t.Fatalf("unquote import in %s: %v", fileName, unquoteErr)
				}
				if isForbiddenImport(path) {
					t.Fatalf("forbidden import %q in %s", path, fileName)
				}
			}
		}
	}
}

func assertTemplateHasEffect(
	t *testing.T,
	catalog Catalog,
	templateValue string,
	kind EffectKind,
	approval bool,
) {
	t.Helper()
	template, ok := catalog.Template(mustTemplateRef(templateValue))
	if !ok {
		t.Fatalf("%s missing", templateValue)
	}
	for _, unit := range template.Units() {
		for _, effect := range unit.Effects() {
			if effect.Kind == kind && effect.ApprovalRequired == approval {
				return
			}
		}
	}
	t.Fatalf("%s has no effect %q with approval=%v", templateValue, kind, approval)
}

func assertErrorCode(t *testing.T, err error, want ErrorCode) {
	t.Helper()
	if got := ErrorCodeOf(err); got != want {
		t.Fatalf("error code = %q, want %q; err=%v", got, want, err)
	}
}

func unitInput(unit Unit) UnitInput {
	return UnitInput{
		Ref: unit.Ref(), StageRef: unit.StageRef(), DependsOn: unit.DependsOn(),
		Role: unit.Role(), WriteSet: unit.WriteSet(),
		RequiredTests:      unit.RequiredTests(),
		AcceptanceCriteria: unit.AcceptanceCriteria(), Effects: unit.Effects(),
		Policy: unit.Policy(),
	}
}

func templateRefStrings(values []Template) []string {
	out := make([]string, len(values))
	for index, value := range values {
		out[index] = value.Ref().String()
	}
	return out
}

func roadmapRefStrings(values []RoadmapCapabilityRef) []string {
	out := make([]string, len(values))
	for index, value := range values {
		out[index] = value.String()
	}
	return out
}

func reverseStages(values []Stage) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}

func reverseUnits(values []Unit) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}

func reverseTemplates(values []Template) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}

func reverseRoadmapRefs(values []RoadmapCapabilityRef) []RoadmapCapabilityRef {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
	return values
}

func isForbiddenImport(path string) bool {
	return strings.HasPrefix(path, "orquesta/internal/application") ||
		strings.HasPrefix(path, "orquesta/internal/goal") ||
		strings.HasPrefix(path, "orquesta/internal/adapters") ||
		path == "os/exec" || path == "database/sql" || path == "net/http"
}
