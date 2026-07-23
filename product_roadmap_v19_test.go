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

func TestProductRoadmapV19ScopeAndControlledRedContract(t *testing.T) {
	index := readRoadmapTestIndex(t)
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
	for _, id := range roadmapV19OwnedCapabilities {
		entry := index.entries[id]
		if entry.Status != "declared" || len(entry.EvidenceRefs) != 0 ||
			!reflect.DeepEqual(entry.Dependencies, vertical.DependsOn) ||
			!reflect.DeepEqual(entry.AcceptanceContracts, vertical.AcceptanceContracts) {
			t.Errorf("V19 capability %s is not exact controlled-red ownership: %#v", id, entry)
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
	if contract.Status != "planned" || contract.TestRef != "planned:acceptance/v19_council_test.go" ||
		contract.Fixture != "planned:fixtures/v19_council" || contract.Receipt != "" {
		t.Fatalf("V19 must remain controlled red until product exists: %#v", contract)
	}
}

func TestV19FocalEvidenceRejectsZeroTestPackagePass(t *testing.T) {
	const name = "TestProductRoadmapV19ScopeAndControlledRedContract"
	packagePassOnly := `{"Action":"pass","Package":"orquesta"}`
	wrongTest := `{"Action":"run","Package":"orquesta","Test":"TestProductRoadmapV19ScopeAndExecutableContract"}
{"Action":"pass","Package":"orquesta","Test":"TestProductRoadmapV19ScopeAndExecutableContract"}`
	exact := `{"Action":"run","Package":"orquesta","Test":"TestProductRoadmapV19ScopeAndControlledRedContract"}
{"Action":"pass","Package":"orquesta","Test":"TestProductRoadmapV19ScopeAndControlledRedContract"}`
	if v19HasExactRunPass(packagePassOnly, name) || v19HasExactRunPass(wrongTest, name) ||
		!v19HasExactRunPass(exact, name) {
		t.Fatal("V19 focal evidence accepted package PASS without exact test run/pass")
	}
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
