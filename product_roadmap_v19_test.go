package orquesta_test

import (
	"bufio"
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
)

var roadmapV19OwnedCapabilities = []string{"EVD-07", "GOV-11", "GOV-13", "GOV-14", "STG-06", "STG-08"}

type roadmapV19Fixture struct {
	ImplementationStatus string   `json:"implementation_status"`
	Command              string   `json:"command"`
	ReceiptPath          string   `json:"receipt_path"`
	OutputPath           string   `json:"output_path"`
	OwnedCapabilityIDs   []string `json:"owned_capability_ids"`
}

func TestProductRoadmapV19ScopeAndLifecycleContract(t *testing.T) {
	index := readRoadmapTestIndex(t)
	fixture := readRoadmapV19Fixture(t)
	assertRoadmapV18Lifecycle(t, index, readRoadmapV18Fixture(t))
	if _, err := os.Stat("product/evidence/v18_independent_reviews.json"); err != nil {
		t.Fatalf("V19 dependency receipt is not accredited: %v", err)
	}

	var owned []string
	for _, entry := range index.document.CapabilityEntries {
		if entry.OwnerContext == "council" && entry.Decision == "accept" {
			owned = append(owned, entry.ID)
		}
	}
	sort.Strings(owned)
	if !reflect.DeepEqual(owned, roadmapV19OwnedCapabilities) {
		t.Fatalf("V19 accepted ownership=%v want=%v", owned, roadmapV19OwnedCapabilities)
	}
	vertical := index.verticals["council"]
	_, receiptErr := os.Stat(fixture.ReceiptPath)
	receiptExists := receiptErr == nil
	if receiptErr != nil && !os.IsNotExist(receiptErr) {
		t.Fatal(receiptErr)
	}
	wantEvidence := []string{"acceptance/v19_council_test.go", "acceptance/fixtures/v19_council.json", fixture.ReceiptPath}
	for _, id := range roadmapV19OwnedCapabilities {
		entry := index.entries[id]
		if !reflect.DeepEqual(entry.Dependencies, vertical.DependsOn) ||
			!reflect.DeepEqual(entry.AcceptanceContracts, vertical.AcceptanceContracts) {
			t.Errorf("V19 capability %s has invalid ownership: %#v", id, entry)
		}
		if receiptExists && (entry.Status != "accredited" || !reflect.DeepEqual(entry.EvidenceRefs, wantEvidence)) {
			t.Errorf("V19 capability %s has invalid post-E evidence: %#v", id, entry)
		}
		if !receiptExists && (entry.Status != "declared" || len(entry.EvidenceRefs) != 0) {
			t.Errorf("V19 capability %s has premature pre-E evidence: %#v", id, entry)
		}
	}

	var mapped []string
	for _, mapping := range index.document.DeferredMappings {
		if mapping.ID == "council" {
			mapped = append(mapped, mapping.CapabilityIDs...)
		}
	}
	sort.Strings(mapped)
	if !reflect.DeepEqual(mapped, roadmapV19OwnedCapabilities) {
		t.Fatalf("V19 deferred mapping=%v want exact=%v", mapped, roadmapV19OwnedCapabilities)
	}

	contract := index.contracts["AC-V19-COUNCIL"]
	preEAssertions := []string{
		"auto required and operator-skip policies have isolated E2Es",
		"ballot identity derives from accredited launch",
		"skip records principal reason time and spec hash",
		"P implementation is present and unsealed; roadmap stays planned until S/E receipt",
	}
	if receiptExists {
		postEAssertions := append([]string(nil), preEAssertions...)
		postEAssertions[3] = "P/S/E binds exact product, sealed source, output and receipt before accreditation"
		if contract.Status != "executable" || contract.TestRef != "acceptance/v19_council_test.go" ||
			contract.Command != fixture.Command || contract.Fixture != "acceptance/fixtures/v19_council.json" ||
			contract.Receipt != fixture.ReceiptPath || !reflect.DeepEqual(contract.Assertions, postEAssertions) {
			t.Fatalf("V19 roadmap contract is not exact post-E: %#v", contract)
		}
	} else if contract.Status != "planned" || contract.TestRef != "planned:acceptance/v19_council_test.go" ||
		contract.Command != "planned:go test -mod=vendor -count=1 . ./acceptance -run '^TestAcceptanceV19Council$'" ||
		contract.Fixture != "planned:fixtures/v19_council" || contract.Receipt != "" ||
		!reflect.DeepEqual(contract.Assertions, preEAssertions) {
		t.Fatalf("V19 roadmap contract must remain non-executable before E: %#v", contract)
	}
}

func TestProductRoadmapV19LifecycleHasExactEvidencePaths(t *testing.T) {
	fixture := readRoadmapV19Fixture(t)
	wantSeal := fixture.ImplementationStatus == "sealed_unexecuted"
	_, receiptErr := os.Lstat(fixture.ReceiptPath)
	wantExecution := receiptErr == nil
	if receiptErr != nil && !os.IsNotExist(receiptErr) {
		t.Fatal(receiptErr)
	}
	for _, expectation := range []struct {
		path string
		want bool
	}{
		{"product/evidence/v19_council_seal.json", wantSeal},
		{fixture.ReceiptPath, wantExecution},
		{fixture.OutputPath, wantExecution},
	} {
		_, err := os.Lstat(expectation.path)
		if expectation.want && err != nil {
			t.Fatalf("V19 expected evidence %s: %v", expectation.path, err)
		}
		if !expectation.want && !os.IsNotExist(err) {
			t.Fatalf("V19 unexpected evidence %s: %v", expectation.path, err)
		}
	}
}

func TestV19EvidenceBelongsOnlyToCouncilCapabilities(t *testing.T) {
	index, fixture := readRoadmapTestIndex(t), readRoadmapV19Fixture(t)
	_, receiptErr := os.Stat(fixture.ReceiptPath)
	receiptExists := receiptErr == nil
	if receiptErr != nil && !os.IsNotExist(receiptErr) {
		t.Fatal(receiptErr)
	}
	wantEvidence := []string{"acceptance/v19_council_test.go", "acceptance/fixtures/v19_council.json", fixture.ReceiptPath}
	evidence := roadmapSetOf(append(append([]string(nil), wantEvidence...), fixture.OutputPath, "product/evidence/v19_council_seal.json")...)
	assertRoadmapEvidenceExclusive(t, index.document.CapabilityEntries, roadmapSetOf(fixture.OwnedCapabilityIDs...), evidence,
		func(t *testing.T, entry roadmapEntry) {
			if receiptExists && (entry.Status != "accredited" || !reflect.DeepEqual(entry.EvidenceRefs, wantEvidence)) {
				t.Errorf("owned V19 capability %s lacks exact post-E evidence: %#v", entry.ID, entry)
			}
			if !receiptExists && (entry.Status != "declared" || len(entry.EvidenceRefs) != 0) {
				t.Errorf("owned V19 capability %s has premature evidence: %#v", entry.ID, entry)
			}
		})
}

func TestV19FocalEvidenceRejectsZeroTestPackagePass(t *testing.T) {
	const name = "TestProductRoadmapV19ScopeAndLifecycleContract"
	packagePassOnly := `{"Action":"pass","Package":"orquesta"}`
	wrongTest := `{"Action":"run","Package":"orquesta","Test":"TestProductRoadmapV19ScopeAndExecutableContract"}
{"Action":"pass","Package":"orquesta","Test":"TestProductRoadmapV19ScopeAndExecutableContract"}`
	exact := `{"Action":"run","Package":"orquesta","Test":"TestProductRoadmapV19ScopeAndLifecycleContract"}
{"Action":"pass","Package":"orquesta","Test":"TestProductRoadmapV19ScopeAndLifecycleContract"}`
	if v19HasExactRunPass(packagePassOnly, name) || v19HasExactRunPass(wrongTest, name) ||
		!v19HasExactRunPass(exact, name) {
		t.Fatal("V19 focal evidence accepted package PASS without exact test run/pass")
	}
}

func readRoadmapV19Fixture(t *testing.T) roadmapV19Fixture {
	t.Helper()
	content, err := os.ReadFile("acceptance/fixtures/v19_council.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture roadmapV19Fixture
	if err := json.Unmarshal(content, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func v19HasExactRunPass(output, testName string) bool {
	run, pass := false, false
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		var event struct {
			Action string `json:"Action"`
			Test   string `json:"Test"`
		}
		if json.Unmarshal(scanner.Bytes(), &event) != nil || event.Test != testName {
			continue
		}
		run = run || event.Action == "run"
		pass = pass || event.Action == "pass"
	}
	return scanner.Err() == nil && run && pass
}
