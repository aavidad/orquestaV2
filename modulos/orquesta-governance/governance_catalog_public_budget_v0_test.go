package orquestagovernance

import "testing"

func TestQueryGovernanceCatalogPublicV0AplicaBudgetYMetadata(t *testing.T) {
	provider := GovernanceCatalogProviderFuncV0(func() (GovernanceCatalogV0, error) {
		return GovernanceCatalogV0{
			CatalogVersion: "v0-test-catalog",
			Catalogs: GovernanceCatalogBlocksV0{
				Effective: []GovernanceCatalogEntryV0{
					testPublicBudgetEntryV0("regla 1", "source-ref://effective/1"),
					testPublicBudgetEntryV0("regla 2", "source-ref://effective/2"),
					testPublicBudgetEntryV0("regla 3", "source-ref://effective/3"),
				},
				Proposed: []GovernanceCatalogEntryV0{
					testPublicBudgetInactiveEntryV0("propuesta 1", GovernanceCatalogStateProposedV0, "source-ref://proposed/1"),
				},
				Quarantine: []GovernanceCatalogEntryV0{
					testPublicBudgetInactiveEntryV0("cuarentena 1", GovernanceCatalogStateQuarantineV0, "source-ref://quarantine/1"),
				},
			},
		}, nil
	})

	response, err := QueryGovernanceCatalogPublicV0(provider, GovernanceCatalogPublicQueryRequestV0{
		Filters: GovernanceCatalogPublicQueryFiltersV0{Module: "orquesta-governance"},
		OutputBudget: GovernanceCatalogPublicOutputBudgetV0{
			MaxEntries: 2,
			MaxBytes:   GovernanceCatalogPublicDefaultMaxBytesV0,
		},
	})
	if err != nil {
		t.Fatalf("QueryGovernanceCatalogPublicV0() error = %v", err)
	}

	if response.SchemaVersion != GovernanceCatalogPublicSchemaV0 ||
		response.CatalogVersion != "v0-test-catalog" ||
		response.CurrentBlock != GovernanceCatalogCurrentBlockEffectiveV0 {
		t.Fatalf("metadata incompleta: %+v", response)
	}
	if response.Freshness.Status != GovernanceCatalogPublicFreshnessDerivedV0 ||
		len(response.SourceRefs) == 0 ||
		response.SourceRefs[0] != "source-ref://effective/1" {
		t.Fatalf("freshness/source_refs = %+v/%+v", response.Freshness, response.SourceRefs)
	}
	if len(response.Effective) != 2 ||
		response.Counters.Effective != 3 ||
		response.OutputBudget.Status != GovernanceCatalogPublicBudgetTruncatedV0 ||
		response.OutputBudget.Reason != "max_entries_exceeded" {
		t.Fatalf("budget efectivo = len:%d counters:%+v budget:%+v", len(response.Effective), response.Counters, response.OutputBudget)
	}
	if response.InactiveSummary.ProposedCount != 1 ||
		response.InactiveSummary.QuarantineCount != 1 ||
		response.InactiveSummary.ProposedRefs[0] != "source-ref://proposed/1" ||
		response.InactiveSummary.QuarantineRefs[0] != "source-ref://quarantine/1" {
		t.Fatalf("inactive summary = %+v", response.InactiveSummary)
	}
}

func TestQueryGovernanceCatalogPublicV0BudgetBytesTrunca(t *testing.T) {
	response, err := QueryGovernanceCatalogPublicV0(
		GovernanceCatalogProviderFuncV0(func() (GovernanceCatalogV0, error) {
			return GovernanceCatalogV0{
				Catalogs: GovernanceCatalogBlocksV0{
					Effective: []GovernanceCatalogEntryV0{
						testPublicBudgetEntryV0("regla bytes", "source-ref://effective/bytes"),
					},
				},
			}, nil
		}),
		GovernanceCatalogPublicQueryRequestV0{
			OutputBudget: GovernanceCatalogPublicOutputBudgetV0{MaxEntries: 10, MaxBytes: 1},
		},
	)
	if err != nil {
		t.Fatalf("QueryGovernanceCatalogPublicV0() error = %v", err)
	}
	if len(response.Effective) != 0 ||
		response.OutputBudget.Status != GovernanceCatalogPublicBudgetTruncatedV0 ||
		response.OutputBudget.Reason != "max_bytes_exceeded" {
		t.Fatalf("budget bytes = len:%d budget:%+v", len(response.Effective), response.OutputBudget)
	}
}

func TestQueryGovernanceCatalogPublicV0FreshnessReasonWithoutRefs(t *testing.T) {
	response, err := QueryGovernanceCatalogPublicV0(
		GovernanceCatalogProviderFuncV0(func() (GovernanceCatalogV0, error) {
			return GovernanceCatalogV0{}, nil
		}),
		GovernanceCatalogPublicQueryRequestV0{},
	)
	if err != nil {
		t.Fatalf("QueryGovernanceCatalogPublicV0() error = %v", err)
	}
	if response.Freshness.Status != GovernanceCatalogPublicFreshnessUnknownV0 ||
		response.Freshness.Reason != "catalog_source_refs_unavailable" {
		t.Fatalf("freshness = %+v", response.Freshness)
	}
	if response.OutputBudget.MaxEntries != GovernanceCatalogPublicDefaultMaxEntriesV0 ||
		response.OutputBudget.MaxBytes != GovernanceCatalogPublicDefaultMaxBytesV0 {
		t.Fatalf("defaults budget = %+v", response.OutputBudget)
	}
}

func testPublicBudgetEntryV0(name string, sourceRef string) GovernanceCatalogEntryV0 {
	entry := testEffectiveEntryV0(name, GovernanceScopeV0{
		Level:   "module",
		Modules: []string{"orquesta-governance"},
	})
	entry.Origin.SourceRef = sourceRef
	return entry
}

func testPublicBudgetInactiveEntryV0(name string, state string, sourceRef string) GovernanceCatalogEntryV0 {
	entry := testInactiveEntryV0(name, state, func(entry *GovernanceCatalogEntryV0) {
		entry.Scope.Modules = []string{"orquesta-governance"}
		entry.Origin.SourceRef = sourceRef
	})
	return entry
}
