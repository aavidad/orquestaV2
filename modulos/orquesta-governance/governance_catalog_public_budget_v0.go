package orquestagovernance

import (
	"encoding/json"
	"strings"
)

const (
	GovernanceCatalogPublicSchemaV0            = "governance_catalog_public_query.v0"
	GovernanceCatalogPublicBudgetCompleteV0    = "complete"
	GovernanceCatalogPublicBudgetTruncatedV0   = "truncated"
	GovernanceCatalogPublicFreshnessDerivedV0  = "derived_from_catalog_refs"
	GovernanceCatalogPublicFreshnessUnknownV0  = "unknown"
	GovernanceCatalogPublicDefaultMaxEntriesV0 = 50
	GovernanceCatalogPublicMaximumMaxEntriesV0 = 200
	GovernanceCatalogPublicDefaultMaxBytesV0   = 64 << 10
	GovernanceCatalogPublicMaximumMaxBytesV0   = 256 << 10
	governanceCatalogPublicSourceRefLimitV0    = 40
	governanceCatalogPublicInactiveRefsLimitV0 = 20
)

type GovernanceCatalogPublicOutputBudgetV0 struct {
	MaxEntries       int    `json:"max_entries"`
	MaxBytes         int    `json:"max_bytes"`
	MatchedEffective int    `json:"matched_effective"`
	ReturnedEntries  int    `json:"returned_entries"`
	ReturnedBytes    int    `json:"returned_bytes"`
	Status           string `json:"status"`
	Reason           string `json:"reason,omitempty"`
}

type GovernanceCatalogPublicFreshnessV0 struct {
	Status     string   `json:"status"`
	Reason     string   `json:"reason,omitempty"`
	SourceRefs []string `json:"source_refs,omitempty"`
}

type GovernanceCatalogPublicInactiveSummaryV0 struct {
	ProposedCount   int      `json:"proposed_count"`
	QuarantineCount int      `json:"quarantine_count"`
	ProposedRefs    []string `json:"proposed_refs,omitempty"`
	QuarantineRefs  []string `json:"quarantine_refs,omitempty"`
	RefsTruncated   bool     `json:"refs_truncated,omitempty"`
}

func NormalizeGovernanceCatalogPublicOutputBudgetV0(in GovernanceCatalogPublicOutputBudgetV0) GovernanceCatalogPublicOutputBudgetV0 {
	out := GovernanceCatalogPublicOutputBudgetV0{
		MaxEntries: in.MaxEntries,
		MaxBytes:   in.MaxBytes,
		Status:     GovernanceCatalogPublicBudgetCompleteV0,
	}
	if out.MaxEntries <= 0 {
		out.MaxEntries = GovernanceCatalogPublicDefaultMaxEntriesV0
	}
	if out.MaxEntries > GovernanceCatalogPublicMaximumMaxEntriesV0 {
		out.MaxEntries = GovernanceCatalogPublicMaximumMaxEntriesV0
	}
	if out.MaxBytes <= 0 {
		out.MaxBytes = GovernanceCatalogPublicDefaultMaxBytesV0
	}
	if out.MaxBytes > GovernanceCatalogPublicMaximumMaxBytesV0 {
		out.MaxBytes = GovernanceCatalogPublicMaximumMaxBytesV0
	}
	return out
}

func limitGovernanceCatalogPublicEffectiveV0(entries []GovernanceCatalogEntryV0, budget GovernanceCatalogPublicOutputBudgetV0) ([]GovernanceCatalogEntryV0, GovernanceCatalogPublicOutputBudgetV0) {
	budget = NormalizeGovernanceCatalogPublicOutputBudgetV0(budget)
	budget.MatchedEffective = len(entries)
	out := make([]GovernanceCatalogEntryV0, 0, minIntGovernanceCatalogPublicV0(len(entries), budget.MaxEntries))
	for _, entry := range entries {
		if len(out) >= budget.MaxEntries {
			budget.Status = GovernanceCatalogPublicBudgetTruncatedV0
			budget.Reason = "max_entries_exceeded"
			break
		}
		raw, err := json.Marshal(entry)
		if err != nil {
			budget.Status = GovernanceCatalogPublicBudgetTruncatedV0
			budget.Reason = "entry_not_serializable"
			break
		}
		if budget.ReturnedBytes+len(raw) > budget.MaxBytes {
			budget.Status = GovernanceCatalogPublicBudgetTruncatedV0
			budget.Reason = "max_bytes_exceeded"
			break
		}
		out = append(out, entry)
		budget.ReturnedBytes += len(raw)
	}
	budget.ReturnedEntries = len(out)
	return out, budget
}

func governanceCatalogPublicFreshnessV0(catalog GovernanceCatalogV0) GovernanceCatalogPublicFreshnessV0 {
	refs, truncated := governanceCatalogPublicSourceRefsV0(catalog)
	if len(refs) == 0 {
		return GovernanceCatalogPublicFreshnessV0{
			Status: GovernanceCatalogPublicFreshnessUnknownV0,
			Reason: "catalog_source_refs_unavailable",
		}
	}
	freshness := GovernanceCatalogPublicFreshnessV0{
		Status:     GovernanceCatalogPublicFreshnessDerivedV0,
		SourceRefs: refs,
	}
	if truncated {
		freshness.Reason = "source_refs_truncated"
	}
	return freshness
}

func governanceCatalogPublicSourceRefsV0(catalog GovernanceCatalogV0) ([]string, bool) {
	refs := make([]string, 0, governanceCatalogPublicSourceRefLimitV0)
	seen := map[string]struct{}{}
	add := func(value string) bool {
		value = strings.TrimSpace(value)
		if value == "" {
			return false
		}
		if _, ok := seen[value]; ok {
			return false
		}
		if len(refs) >= governanceCatalogPublicSourceRefLimitV0 {
			return true
		}
		seen[value] = struct{}{}
		refs = append(refs, value)
		return false
	}
	for _, block := range [][]GovernanceCatalogEntryV0{catalog.Catalogs.Effective, catalog.Catalogs.Proposed, catalog.Catalogs.Quarantine} {
		for _, entry := range block {
			if add(entry.Origin.SourceRef) || add(governanceStringPointerValueV0(entry.Promotion.DecisionRef)) {
				return refs, true
			}
		}
	}
	return refs, false
}

func governanceCatalogPublicInactiveSummaryV0(catalog GovernanceCatalogV0, query GovernanceCatalogQueryV0) GovernanceCatalogPublicInactiveSummaryV0 {
	summary := GovernanceCatalogPublicInactiveSummaryV0{}
	summary.ProposedCount, summary.ProposedRefs, summary.RefsTruncated = governanceCatalogPublicInactiveRefsV0(catalog.Catalogs.Proposed, query, summary.RefsTruncated)
	summary.QuarantineCount, summary.QuarantineRefs, summary.RefsTruncated = governanceCatalogPublicInactiveRefsV0(catalog.Catalogs.Quarantine, query, summary.RefsTruncated)
	return summary
}

func governanceCatalogPublicInactiveRefsV0(entries []GovernanceCatalogEntryV0, query GovernanceCatalogQueryV0, alreadyTruncated bool) (int, []string, bool) {
	refs := []string{}
	seen := map[string]struct{}{}
	count := 0
	truncated := alreadyTruncated
	for _, entry := range entries {
		if !governanceEntryMatchesQueryV0(entry, query) {
			continue
		}
		count++
		ref := strings.TrimSpace(entry.Origin.SourceRef)
		if ref == "" {
			continue
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		if len(refs) >= governanceCatalogPublicInactiveRefsLimitV0 {
			truncated = true
			continue
		}
		seen[ref] = struct{}{}
		refs = append(refs, ref)
	}
	return count, refs, truncated
}

func governanceStringPointerValueV0(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func minIntGovernanceCatalogPublicV0(a, b int) int {
	if a < b {
		return a
	}
	return b
}
