package orquesta_test

import (
	"os"
	"reflect"
	"sort"
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
