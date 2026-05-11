package orquestagovernance

import (
	"errors"
	"testing"
)

func TestQueryEffectiveGovernanceCatalogV0FiltersEffectiveAndCountsInactive(t *testing.T) {
	catalog := GovernanceCatalogV0{
		Catalogs: GovernanceCatalogBlocksV0{
			Effective: []GovernanceCatalogEntryV0{
				testEffectiveEntryV0("regla core review", GovernanceScopeV0{
					Level:   "module",
					Modules: []string{"orquesta-core"},
					Roles:   []string{"programador"},
					Phases:  []string{"review"},
					Tags:    []string{"contracts", "microtask"},
				}),
				testEffectiveEntryV0("regla factory review", GovernanceScopeV0{
					Level:   "module",
					Modules: []string{"orquesta-factory"},
					Roles:   []string{"programador"},
					Phases:  []string{"review"},
					Tags:    []string{"contracts"},
				}),
			},
			Proposed: []GovernanceCatalogEntryV0{
				testInactiveEntryV0("candidato core", GovernanceCatalogStateProposedV0),
				testInactiveEntryV0("candidato factory", GovernanceCatalogStateProposedV0, func(entry *GovernanceCatalogEntryV0) {
					entry.Scope.Modules = []string{"orquesta-factory"}
				}),
			},
			Quarantine: []GovernanceCatalogEntryV0{
				testInactiveEntryV0("historico core", GovernanceCatalogStateQuarantineV0),
				testInactiveEntryV0("historico otro tag", GovernanceCatalogStateQuarantineV0, func(entry *GovernanceCatalogEntryV0) {
					entry.Scope.Tags = []string{"legacy"}
				}),
			},
		},
	}

	result, err := QueryEffectiveGovernanceCatalogV0(catalog, GovernanceCatalogQueryV0{
		Module: "orquesta-core",
		Role:   "programador",
		Phase:  "review",
		Tags:   []string{"contracts"},
	})
	if err != nil {
		t.Fatalf("QueryEffectiveGovernanceCatalogV0() error = %v", err)
	}

	if result.Counters.Proposed != 1 || result.Counters.Quarantine != 1 {
		t.Fatalf("inactive counters = proposed:%d quarantine:%d, want 1/1", result.Counters.Proposed, result.Counters.Quarantine)
	}
	if result.Counters.Effective != 1 || len(result.Effective) != 1 {
		t.Fatalf("effective matches = counter:%d len:%d, want 1/1", result.Counters.Effective, len(result.Effective))
	}
	if result.Effective[0].Name != "regla core review" {
		t.Fatalf("effective[0].Name = %q, want regla core review", result.Effective[0].Name)
	}
}

func TestQueryEffectiveGovernanceCatalogV0IgnoresInactiveAsEffective(t *testing.T) {
	catalog := GovernanceCatalogV0{
		Catalogs: GovernanceCatalogBlocksV0{
			Proposed: []GovernanceCatalogEntryV0{
				testInactiveEntryV0("candidato coincidente", GovernanceCatalogStateProposedV0),
			},
			Quarantine: []GovernanceCatalogEntryV0{
				testInactiveEntryV0("cuarentena coincidente", GovernanceCatalogStateQuarantineV0),
			},
		},
	}

	result, err := QueryEffectiveGovernanceCatalogV0(catalog, GovernanceCatalogQueryV0{
		Module: "orquesta-core",
		Role:   "programador",
		Phase:  "review",
		Tags:   []string{"contracts"},
	})
	if err != nil {
		t.Fatalf("QueryEffectiveGovernanceCatalogV0() error = %v", err)
	}

	if len(result.Effective) != 0 || result.Counters.Effective != 0 {
		t.Fatalf("inactive entries treated as effective: len:%d counter:%d", len(result.Effective), result.Counters.Effective)
	}
	if result.Counters.Proposed != 1 || result.Counters.Quarantine != 1 {
		t.Fatalf("inactive counters = proposed:%d quarantine:%d, want 1/1", result.Counters.Proposed, result.Counters.Quarantine)
	}
}

func TestQueryEffectiveGovernanceCatalogV0RejectsProposedInEffectiveBlock(t *testing.T) {
	entry := testEffectiveEntryV0("estado mal ubicado", GovernanceScopeV0{Level: "global"})
	entry.Status.CatalogState = GovernanceCatalogStateProposedV0

	_, err := QueryEffectiveGovernanceCatalogV0(GovernanceCatalogV0{
		Catalogs: GovernanceCatalogBlocksV0{
			Effective: []GovernanceCatalogEntryV0{entry},
		},
	}, GovernanceCatalogQueryV0{})
	if !errors.Is(err, GovernanceCatalogForbiddenActivationErrorV0) {
		t.Fatalf("error = %v, want %v", err, GovernanceCatalogForbiddenActivationErrorV0)
	}
}

func TestQueryEffectiveGovernanceCatalogV0RejectsEffectiveWithoutDecision(t *testing.T) {
	entry := testEffectiveEntryV0("sin decision", GovernanceScopeV0{Level: "global"})
	entry.Promotion.DecisionRef = nil

	_, err := QueryEffectiveGovernanceCatalogV0(GovernanceCatalogV0{
		Catalogs: GovernanceCatalogBlocksV0{
			Effective: []GovernanceCatalogEntryV0{entry},
		},
	}, GovernanceCatalogQueryV0{})
	if !errors.Is(err, GovernanceEntryEffectiveWithoutDecisionV0) {
		t.Fatalf("error = %v, want %v", err, GovernanceEntryEffectiveWithoutDecisionV0)
	}
}

func TestQueryEffectiveGovernanceCatalogV0RejectsEffectiveInProposedBlock(t *testing.T) {
	entry := testEffectiveEntryV0("efectiva mal ubicada", GovernanceScopeV0{Level: "global"})

	_, err := QueryEffectiveGovernanceCatalogV0(GovernanceCatalogV0{
		Catalogs: GovernanceCatalogBlocksV0{
			Proposed: []GovernanceCatalogEntryV0{entry},
		},
	}, GovernanceCatalogQueryV0{})
	if !errors.Is(err, GovernanceCatalogForbiddenActivationErrorV0) {
		t.Fatalf("error = %v, want %v", err, GovernanceCatalogForbiddenActivationErrorV0)
	}
}

func TestQueryEffectiveGovernanceCatalogV0RejectsProposedInQuarantineBlock(t *testing.T) {
	entry := testInactiveEntryV0("propuesta mal ubicada", GovernanceCatalogStateProposedV0)

	_, err := QueryEffectiveGovernanceCatalogV0(GovernanceCatalogV0{
		Catalogs: GovernanceCatalogBlocksV0{
			Quarantine: []GovernanceCatalogEntryV0{entry},
		},
	}, GovernanceCatalogQueryV0{})
	if !errors.Is(err, GovernanceCatalogForbiddenActivationErrorV0) {
		t.Fatalf("error = %v, want %v", err, GovernanceCatalogForbiddenActivationErrorV0)
	}
}

func TestQueryEffectiveGovernanceCatalogV0RejectsQuarantineInProposedBlock(t *testing.T) {
	entry := testInactiveEntryV0("cuarentena mal ubicada", GovernanceCatalogStateQuarantineV0)

	_, err := QueryEffectiveGovernanceCatalogV0(GovernanceCatalogV0{
		Catalogs: GovernanceCatalogBlocksV0{
			Proposed: []GovernanceCatalogEntryV0{entry},
		},
	}, GovernanceCatalogQueryV0{})
	if !errors.Is(err, GovernanceCatalogForbiddenActivationErrorV0) {
		t.Fatalf("error = %v, want %v", err, GovernanceCatalogForbiddenActivationErrorV0)
	}
}

func testEffectiveEntryV0(name string, scope GovernanceScopeV0) GovernanceCatalogEntryV0 {
	return GovernanceCatalogEntryV0{
		Kind:    "rule",
		Name:    name,
		Summary: "Regla efectiva de prueba para consulta pura en memoria.",
		Scope:   scope,
		Status: GovernanceStatusV0{
			CatalogState: GovernanceCatalogStateEffectiveV0,
			ReviewState:  GovernanceReviewStateApprovedEffectiveV0,
		},
		Version: GovernanceVersionV0{
			ContractVersion:       GovernanceCatalogVersionV0,
			DocumentVersion:       "v0-test",
			SourceVersionEvidence: "test",
		},
		Origin: GovernanceOriginV0{
			SourceType:           "local_doc",
			SourceRef:            "docs/pruebas.md#GOV-006",
			ForensicInventoryRef: "GOV-006",
			LegacyPublicIDPolicy: "no_legacy_ids_as_canon",
		},
		Promotion: GovernancePromotionV0{
			Criterion:                "Promocion de prueba para validar consulta efectiva.",
			DecisionRef:              testStringPointerV0("GOV-006-TEST"),
			PromotedAt:               testStringPointerV0("2026-05-04"),
			HistoricalReviewRequired: true,
		},
		SecretControls: GovernanceSecretControlsV0{
			ContainsSecretMaterial:   false,
			ReviewMethod:             "structured_policy",
			StructuredMarkersChecked: []string{"secret_kind"},
		},
	}
}

func testInactiveEntryV0(name string, catalogState string, options ...func(*GovernanceCatalogEntryV0)) GovernanceCatalogEntryV0 {
	entry := testEffectiveEntryV0(name, GovernanceScopeV0{
		Level:   "module",
		Modules: []string{"orquesta-core"},
		Roles:   []string{"programador"},
		Phases:  []string{"review"},
		Tags:    []string{"contracts"},
	})
	entry.Status.CatalogState = catalogState
	entry.Status.ReviewState = testInactiveReviewStateV0(catalogState)
	entry.Promotion.DecisionRef = nil
	entry.Promotion.PromotedAt = nil
	for _, option := range options {
		option(&entry)
	}
	return entry
}

func testInactiveReviewStateV0(catalogState string) string {
	if catalogState == GovernanceCatalogStateQuarantineV0 {
		return "quarantined"
	}
	return "pending_review"
}

func testStringPointerV0(value string) *string {
	return &value
}
