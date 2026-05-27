package main

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

type backlogScanEntryObservationV0 struct {
	Ref       string
	Date      string
	Ordinal   int
	Line      int
	BlockHash string
}

var backlogScanEntryHeadingPatternV0 = regexp.MustCompile(`^##\s+Escaneo backlog\s+(\d{4}-\d{2}-\d{2})(?:\s+(.+))?$`)

func backlogScanEntryRefsForDocumentV0(
	path string,
	content string,
) ([]backlogScanEntryObservationV0, []orquestaserver.BacklogScanEntryIssueV0) {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	entries := make([]backlogScanEntryObservationV0, 0)
	issues := make([]orquestaserver.BacklogScanEntryIssueV0, 0)
	for index, line := range lines {
		heading := strings.TrimSpace(line)
		matches := backlogScanEntryHeadingPatternV0.FindStringSubmatch(heading)
		if len(matches) == 0 {
			continue
		}
		next := len(lines)
		for cursor := index + 1; cursor < len(lines); cursor++ {
			if strings.HasPrefix(strings.TrimSpace(lines[cursor]), "## ") {
				next = cursor
				break
			}
		}
		blockHash := backlogScanHashV0(strings.Join(lines[index:next], "\n"))[:12]
		entry := backlogScanEntryObservationV0{
			Date:      matches[1],
			Ordinal:   backlogScanEntryOrdinalV0(matches[2]),
			Line:      index + 1,
			BlockHash: blockHash,
		}
		entry.Ref = backlogScanEntryRefV0(path, entry.Date, entry.Ordinal, blockHash)
		entries = append(entries, entry)
		if entry.Ordinal <= 0 {
			issues = append(issues, backlogScanEntryIssueV0(
				"backlog_scan_entry_order_ambiguous",
				entry,
				"entrada de escaneo sin ordinal publico derivable",
			))
		}
	}
	return entries, append(issues, backlogScanEntryOrderIssuesV0(entries)...)
}

func backlogScanEntryDigestV0(entries []backlogScanEntryObservationV0) string {
	if len(entries) == 0 {
		return ""
	}
	parts := make([]string, 0, len(entries))
	for _, entry := range entries {
		parts = append(parts, entry.Ref+"|"+strconv.Itoa(entry.Line))
	}
	return backlogScanHashV0(strings.Join(parts, "\n"))[:16]
}

func backlogScanPublicEntriesV0(
	entries []backlogScanEntryObservationV0,
) []orquestaserver.BacklogScanEntryV0 {
	out := make([]orquestaserver.BacklogScanEntryV0, 0, len(entries))
	for _, entry := range entries {
		out = append(out, orquestaserver.BacklogScanEntryV0{
			Ref:       entry.Ref,
			Date:      entry.Date,
			Ordinal:   entry.Ordinal,
			Line:      entry.Line,
			BlockHash: entry.BlockHash,
		})
	}
	return out
}

func backlogScanEntryOrderIssuesV0(
	entries []backlogScanEntryObservationV0,
) []orquestaserver.BacklogScanEntryIssueV0 {
	seen := map[string]backlogScanEntryObservationV0{}
	lastByDate := map[string]backlogScanEntryObservationV0{}
	issues := make([]orquestaserver.BacklogScanEntryIssueV0, 0)
	for _, entry := range entries {
		if entry.Ordinal <= 0 {
			continue
		}
		key := entry.Date + "|" + strconv.Itoa(entry.Ordinal)
		if previous, ok := seen[key]; ok {
			issues = append(issues,
				backlogScanEntryIssueV0("backlog_scan_entry_duplicate", previous, "ordinal de escaneo duplicado"),
				backlogScanEntryIssueV0("backlog_scan_entry_duplicate", entry, "ordinal de escaneo duplicado"),
			)
		}
		seen[key] = entry
		if previous, ok := lastByDate[entry.Date]; ok &&
			(entry.Ordinal <= previous.Ordinal || entry.Ordinal > previous.Ordinal+1) {
			issues = append(issues, backlogScanEntryIssueV0(
				"backlog_scan_entry_order_ambiguous",
				entry,
				"salto u orden no monotono de ordinales de escaneo",
			))
		}
		lastByDate[entry.Date] = entry
	}
	return issues
}

func backlogScanEntryIssueV0(
	code string,
	entry backlogScanEntryObservationV0,
	message string,
) orquestaserver.BacklogScanEntryIssueV0 {
	return orquestaserver.BacklogScanEntryIssueV0{
		Code:         code,
		ScanEntryRef: entry.Ref,
		Line:         entry.Line,
		Message:      message,
	}
}

func backlogScanEntryRefV0(path string, date string, ordinal int, blockHash string) string {
	ordinalText := "ordinal-" + strconv.Itoa(ordinal)
	if ordinal <= 0 {
		ordinalText = "ordinal-unknown"
	}
	return "scan-entry-ref-" + backlogScanHashV0(
		strings.TrimSpace(path) + "|" + date + "|" + ordinalText + "|" + blockHash,
	)[:16]
}

func backlogScanEntryOrdinalV0(value string) int {
	normalized := idleSelfImprovementHeadingRefV0(value)
	for _, entry := range backlogScanEntryOrdinalsV0() {
		if strings.Contains(normalized, entry.Token) {
			return entry.Ordinal
		}
	}
	return 0
}

type backlogScanEntryOrdinalTokenV0 struct {
	Token   string
	Ordinal int
}

func backlogScanEntryOrdinalsV0() []backlogScanEntryOrdinalTokenV0 {
	units := []backlogScanEntryOrdinalTokenV0{
		{"primera", 1}, {"segunda", 2}, {"tercera", 3},
		{"cuarta", 4}, {"quinta", 5}, {"sexta", 6},
		{"septima", 7}, {"octava", 8}, {"novena", 9},
	}
	ordinals := append([]backlogScanEntryOrdinalTokenV0(nil), units...)
	ordinals = append(ordinals, []backlogScanEntryOrdinalTokenV0{
		{"decima", 10}, {"undecima", 11}, {"duodecima", 12},
		{"decimoprimera", 11}, {"decimosegunda", 12},
		{"decimotercera", 13}, {"decimocuarta", 14},
		{"decimoquinta", 15}, {"decimosexta", 16},
		{"decimoseptima", 17}, {"decimoctava", 18},
		{"decimonovena", 19},
	}...)
	tens := []backlogScanEntryOrdinalTokenV0{
		{"vigesima", 20}, {"trigesima", 30},
		{"cuadragesima", 40}, {"quincuagesima", 50},
		{"sexagesima", 60}, {"septuagesima", 70},
		{"octogesima", 80}, {"nonagesima", 90},
	}
	for _, ten := range tens {
		ordinals = append(ordinals, ten)
		masculine := strings.TrimSuffix(ten.Token, "a") + "o"
		for _, unit := range units {
			ordinals = append(ordinals,
				backlogScanEntryOrdinalTokenV0{ten.Token + "-" + unit.Token, ten.Ordinal + unit.Ordinal},
				backlogScanEntryOrdinalTokenV0{masculine + "-" + unit.Token, ten.Ordinal + unit.Ordinal},
				backlogScanEntryOrdinalTokenV0{
					strings.ReplaceAll(masculine+unit.Token, "oo", "o"),
					ten.Ordinal + unit.Ordinal,
				},
			)
		}
	}
	sort.SliceStable(ordinals, func(i, j int) bool {
		return len(ordinals[i].Token) > len(ordinals[j].Token)
	})
	return ordinals
}
