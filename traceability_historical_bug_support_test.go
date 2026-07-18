package orquesta_test

import (
	"encoding/json"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func traceReadHistoricalBugRows(t *testing.T, path string) []traceHistoricalBugRow {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var entries []traceHistoricalBugRow
	traceScanStrictJSONL(t, file, path, func(decoder *json.Decoder, lineNumber int) {
		var entry traceHistoricalBugRow
		if err := decoder.Decode(&entry); err != nil {
			t.Fatalf("decode %s:%d: %v", path, lineNumber, err)
		}
		entries = append(entries, entry)
	})
	return entries
}

func traceReadHistoricalBugOccurrences(t *testing.T, path string) []traceHistoricalBugOccurrence {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var entries []traceHistoricalBugOccurrence
	traceScanStrictJSONL(t, file, path, func(decoder *json.Decoder, lineNumber int) {
		var entry traceHistoricalBugOccurrence
		if err := decoder.Decode(&entry); err != nil {
			t.Fatalf("decode %s:%d: %v", path, lineNumber, err)
		}
		entries = append(entries, entry)
	})
	return entries
}

func traceReadHistoricalBugIDs(t *testing.T, path string) []traceHistoricalBugID {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var entries []traceHistoricalBugID
	traceScanStrictJSONL(t, file, path, func(decoder *json.Decoder, lineNumber int) {
		var entry traceHistoricalBugID
		if err := decoder.Decode(&entry); err != nil {
			t.Fatalf("decode %s:%d: %v", path, lineNumber, err)
		}
		entries = append(entries, entry)
	})
	return entries
}

func traceReadHistoricalBugIDReviews(t *testing.T, path string) []traceHistoricalBugIDReview {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var entries []traceHistoricalBugIDReview
	traceScanStrictJSONL(t, file, path, func(decoder *json.Decoder, lineNumber int) {
		var entry traceHistoricalBugIDReview
		if err := decoder.Decode(&entry); err != nil {
			t.Fatalf("decode %s:%d: %v", path, lineNumber, err)
		}
		entries = append(entries, entry)
	})
	return entries
}

func traceValidateHistoricalBugRows(
	t *testing.T,
	rows []traceHistoricalBugRow,
	bugSources map[string]traceSourceDisposition,
	accepted map[string]struct{},
	richRowSourceRefs []string,
) map[string]traceHistoricalBugRow {
	t.Helper()
	richRowSources := make(map[string]struct{}, len(richRowSourceRefs))
	for _, sourceRef := range richRowSourceRefs {
		if _, duplicate := richRowSources[sourceRef]; duplicate {
			t.Fatalf("duplicate rich historical bug source %q", sourceRef)
		}
		if _, exists := bugSources[sourceRef]; !exists {
			t.Fatalf("rich historical bug source %q lacks bug role", sourceRef)
		}
		richRowSources[sourceRef] = struct{}{}
	}
	sourceLines := make(map[string][]string, len(bugSources))
	detectedRows := make(map[string]struct{})
	for sourceRef := range bugSources {
		lines := traceReadSourceLines(t, sourceRef)
		sourceLines[sourceRef] = lines
		if _, reviewedRichSource := richRowSources[sourceRef]; !reviewedRichSource {
			continue
		}
		for index, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "| BUG-ORQ-") {
				detectedRows[traceHistoricalBugRowKey(sourceRef, index+1)] = struct{}{}
			}
		}
	}
	byKey := make(map[string]traceHistoricalBugRow, len(rows))
	literalTotals := make(map[string]int)
	for _, row := range rows {
		literalTotals[row.SourceBugID]++
	}
	literalSeen := make(map[string]int)
	previousKey := ""
	for _, row := range rows {
		lines, sourceExists := sourceLines[row.SourceRef]
		_, reviewedRichSource := richRowSources[row.SourceRef]
		key := traceHistoricalBugRowKey(row.SourceRef, row.SourceLine)
		if row.SchemaVersion != 1 || !sourceExists || !reviewedRichSource || key <= previousKey ||
			row.SourceLine < 1 || row.SourceLine > len(lines) || row.DetectionKind != "bug_table_row" ||
			strings.TrimSpace(row.SourceBugID) == "" || strings.TrimSpace(row.OriginalState) == "" ||
			strings.TrimSpace(row.Area) == "" || strings.TrimSpace(row.Symptom) == "" ||
			strings.TrimSpace(row.ArchitecturalHypothesis) == "" || strings.TrimSpace(row.Action) == "" ||
			strings.TrimSpace(row.ReviewNote) == "" || row.ClosureEvidence != "not_verified" {
			t.Fatalf("invalid or unordered rich historical bug row: %#v", row)
		}
		previousKey = key
		line := lines[row.SourceLine-1]
		textSHA := "sha256:" + traceSHA256Hex(line)
		wantEntryRef := traceHistoricalBugEntryRef(row.SourceRef, row.SourceLine, textSHA)
		if row.TextSHA256 != textSHA || row.EntryRef != wantEntryRef {
			t.Fatalf("rich historical bug row identity drift at %s:%d", row.SourceRef, row.SourceLine)
		}
		firstCell := traceMarkdownFirstCell(line)
		if firstCell != row.SourceBugID {
			t.Fatalf("rich historical bug row %s:%d source ID=%q, want first cell %q",
				row.SourceRef, row.SourceLine, row.SourceBugID, firstCell)
		}
		wantReported := traceExpandHistoricalBugRef(row.SourceBugID)
		if !reflect.DeepEqual(row.ReportedIDs, wantReported) {
			t.Fatalf("rich historical bug row %s:%d reported IDs=%v, want %v",
				row.SourceRef, row.SourceLine, row.ReportedIDs, wantReported)
		}
		literalSeen[row.SourceBugID]++
		if row.DuplicateIDOccurrence != literalSeen[row.SourceBugID] ||
			row.DuplicateIDCount != literalTotals[row.SourceBugID] {
			t.Fatalf("rich historical bug row %s:%d duplicate ordinal/count drift", row.SourceRef, row.SourceLine)
		}
		cells := traceMarkdownCells(line)
		if row.SourceRef == "docs/inventario_bugs_orquesta_2026-06-30.md" {
			if !traceStringSliceContains(cells, row.OriginalState) || !traceStringSliceContains(cells, row.Area) {
				t.Fatalf("rich historical bug row %s:%d state/area are not exact cells", row.SourceRef, row.SourceLine)
			}
		} else {
			wantState := traceDocumentField(lines, "Estado:")
			wantArea := strings.TrimSpace(strings.TrimPrefix(lines[0], "# Incidencia:"))
			if row.OriginalState != wantState || row.Area != wantArea {
				t.Fatalf("enriched rich historical bug row %s:%d state/area=%q/%q, want exact document fields %q/%q",
					row.SourceRef, row.SourceLine, row.OriginalState, row.Area, wantState, wantArea)
			}
		}
		wantSection := traceNearestLevelTwoHeading(lines, row.SourceLine-1)
		if row.SourceSection != wantSection {
			t.Fatalf("rich historical bug row %s:%d section=%q, want %q",
				row.SourceRef, row.SourceLine, row.SourceSection, wantSection)
		}
		if len(row.CapabilityIDs) == 0 || !sort.StringsAreSorted(row.CapabilityIDs) {
			t.Fatalf("rich historical bug row %s:%d capabilities empty or unsorted: %v",
				row.SourceRef, row.SourceLine, row.CapabilityIDs)
		}
		for index, capabilityID := range row.CapabilityIDs {
			if index > 0 && capabilityID == row.CapabilityIDs[index-1] {
				t.Fatalf("rich historical bug row %s:%d repeats capability %q", row.SourceRef, row.SourceLine, capabilityID)
			}
			if _, exists := accepted[capabilityID]; !exists {
				t.Fatalf("rich historical bug row %s:%d maps non-accepted capability %q",
					row.SourceRef, row.SourceLine, capabilityID)
			}
		}
		wantDisposition := "historical_lesson_candidate"
		if strings.HasPrefix(row.LessonTestRef, "planned:") {
			wantDisposition = "historical_pending_lesson"
			wantPlanned := "planned:lesson-tests/" + strings.TrimPrefix(row.TextSHA256, "sha256:")[:20]
			if row.LessonTestRef != wantPlanned || len(row.CitedExistingTestRefs) != 0 {
				t.Fatalf("rich historical bug row %s:%d invalid planned lesson", row.SourceRef, row.SourceLine)
			}
		} else {
			if !traceStringSliceContains(row.CitedExistingTestRefs, row.LessonTestRef) {
				t.Fatalf("rich historical bug row %s:%d primary lesson is not cited", row.SourceRef, row.SourceLine)
			}
			traceRequireExistingLessonRef(t, row.LessonTestRef)
		}
		if row.Disposition != wantDisposition {
			t.Fatalf("rich historical bug row %s:%d disposition=%q, want %q",
				row.SourceRef, row.SourceLine, row.Disposition, wantDisposition)
		}
		if _, duplicate := byKey[key]; duplicate {
			t.Fatalf("duplicate rich historical bug row at %s:%d", row.SourceRef, row.SourceLine)
		}
		byKey[key] = row
		delete(detectedRows, key)
	}
	if len(detectedRows) != 0 {
		missing := make([]string, 0, len(detectedRows))
		for key := range detectedRows {
			missing = append(missing, key)
		}
		sort.Strings(missing)
		t.Fatalf("rich historical bug rows omit physical inventory rows: %v", missing)
	}
	return byKey
}

func traceExtractHistoricalBugOccurrences(
	t *testing.T,
	bugSources map[string]traceSourceDisposition,
	rowsByLine map[string]traceHistoricalBugRow,
	basePattern *regexp.Regexp,
	slashPattern *regexp.Regexp,
) []traceHistoricalBugOccurrence {
	t.Helper()
	sourceRefs := make([]string, 0, len(bugSources))
	for sourceRef := range bugSources {
		sourceRefs = append(sourceRefs, sourceRef)
	}
	sort.Strings(sourceRefs)
	entries := make([]traceHistoricalBugOccurrence, 0, 1500)
	seenRefs := make(map[string]struct{})
	for _, sourceRef := range sourceRefs {
		lines := traceReadSourceLines(t, sourceRef)
		for lineIndex, line := range lines {
			textSHA := "sha256:" + traceSHA256Hex(line)
			for _, match := range basePattern.FindAllStringIndex(line, -1) {
				start, end := match[0], match[1]
				baseRef := line[start:end]
				slashParts, rawEnd := traceHistoricalBugSlashParts(line, end, slashPattern)
				rawRef := line[start:rawEnd]
				bugIDs := traceExpandHistoricalBugBase(baseRef, slashParts)
				for componentIndex, bugID := range bugIDs {
					coverageKind := "narrative_only"
					if row, exists := rowsByLine[traceHistoricalBugRowKey(sourceRef, lineIndex+1)]; exists {
						rowColumn := strings.Index(line, row.SourceBugID) + 1
						if rowColumn == start+1 && traceStringSliceContains(row.ReportedIDs, bugID) {
							coverageKind = "row_covered"
						}
					}
					entry := traceHistoricalBugOccurrence{
						SchemaVersion:  1,
						BugID:          bugID,
						RawBugRef:      rawRef,
						ComponentIndex: componentIndex + 1,
						SourceRef:      sourceRef,
						SourceLine:     lineIndex + 1,
						SourceColumn:   start + 1,
						TextSHA256:     textSHA,
						CoverageKind:   coverageKind,
					}
					entry.OccurrenceRef = traceHistoricalBugOccurrenceRef(entry)
					if _, duplicate := seenRefs[entry.OccurrenceRef]; duplicate {
						t.Fatalf("duplicate historical bug occurrence ref %q", entry.OccurrenceRef)
					}
					seenRefs[entry.OccurrenceRef] = struct{}{}
					entries = append(entries, entry)
				}
			}
		}
	}
	return entries
}

func traceHistoricalBugSlashParts(line string, offset int, slashPattern *regexp.Regexp) ([]string, int) {
	var parts []string
	cursor := offset
	for cursor < len(line) {
		match := slashPattern.FindStringIndex(line[cursor:])
		if match == nil || match[0] != 0 {
			break
		}
		part := line[cursor+1 : cursor+match[1]]
		if part == "BUG" && cursor+match[1] < len(line) && line[cursor+match[1]] == '-' {
			break
		}
		parts = append(parts, part)
		cursor += match[1]
	}
	return parts, cursor
}

func traceExpandHistoricalBugRef(rawRef string) []string {
	parts := strings.Split(rawRef, "/")
	return traceExpandHistoricalBugBase(parts[0], parts[1:])
}

func traceExpandHistoricalBugBase(baseRef string, slashParts []string) []string {
	if groups := traceHistoricalBugLetterRangeRE.FindStringSubmatch(baseRef); groups != nil {
		start, end := groups[3][0], groups[4][0]
		if start <= end {
			ids := make([]string, 0, int(end-start)+1)
			for suffix := start; suffix <= end; suffix++ {
				ids = append(ids, groups[1]+"-"+groups[2]+string(suffix))
			}
			return ids
		}
	}
	lastHyphen := strings.LastIndex(baseRef, "-")
	if lastHyphen < 0 {
		return []string{baseRef}
	}
	prefix := baseRef[:lastHyphen+1]
	firstSuffix := baseRef[lastHyphen+1:]
	digitPrefix := ""
	for index := 0; index < len(firstSuffix) && firstSuffix[index] >= '0' && firstSuffix[index] <= '9'; index++ {
		digitPrefix = firstSuffix[:index+1]
	}
	ids := []string{baseRef}
	for _, part := range slashParts {
		suffix := part
		lettersOnly := part != ""
		for _, char := range part {
			if char < 'A' || char > 'Z' {
				lettersOnly = false
				break
			}
		}
		if lettersOnly && digitPrefix != "" {
			suffix = digitPrefix + part
		}
		ids = append(ids, prefix+suffix)
	}
	return ids
}

var traceHistoricalBugLetterRangeRE = regexp.MustCompile(`^(BUG-ORQ(?:-[A-Z0-9]+)*)-([0-9]+)([A-Z])-([A-Z])$`)

func traceHistoricalBugEntryRef(sourceRef string, sourceLine int, textSHA string) string {
	subject := sourceRef + "\n" + strconv.Itoa(sourceLine) + "\n" + textSHA
	return "BUGENTRY-" + traceSHA256Hex(subject)[:24]
}

func traceHistoricalBugOccurrenceRef(entry traceHistoricalBugOccurrence) string {
	subject := strings.Join([]string{
		entry.BugID,
		entry.RawBugRef,
		strconv.Itoa(entry.ComponentIndex),
		entry.SourceRef,
		strconv.Itoa(entry.SourceLine),
		strconv.Itoa(entry.SourceColumn),
		entry.TextSHA256,
	}, "\n")
	return "BUGOCC-" + traceSHA256Hex(subject)[:24]
}

func traceHistoricalBugRowCapabilities(rows []traceHistoricalBugRow) map[string][]string {
	sets := make(map[string]map[string]struct{})
	for _, row := range rows {
		for _, bugID := range row.ReportedIDs {
			if sets[bugID] == nil {
				sets[bugID] = make(map[string]struct{})
			}
			for _, capabilityID := range row.CapabilityIDs {
				sets[bugID][capabilityID] = struct{}{}
			}
		}
	}
	result := make(map[string][]string, len(sets))
	for bugID, values := range sets {
		for capabilityID := range values {
			result[bugID] = append(result[bugID], capabilityID)
		}
		sort.Strings(result[bugID])
	}
	return result
}

func traceHistoricalBugRowLessonRefs(rows []traceHistoricalBugRow) map[string][]string {
	sets := make(map[string]map[string]struct{})
	for _, row := range rows {
		for _, bugID := range row.ReportedIDs {
			if sets[bugID] == nil {
				sets[bugID] = make(map[string]struct{})
			}
			sets[bugID][row.LessonTestRef] = struct{}{}
		}
	}
	result := make(map[string][]string, len(sets))
	for bugID, values := range sets {
		for lessonRef := range values {
			result[bugID] = append(result[bugID], lessonRef)
		}
		sort.Strings(result[bugID])
	}
	return result
}

func traceBuildHistoricalBugIDs(
	occurrencesByID map[string][]traceHistoricalBugOccurrence,
	rowCapabilities map[string][]string,
	rowLessonRefs map[string][]string,
	reviews map[string]traceHistoricalBugIDReview,
	bindings map[string]traceHistoricalBugIDReviewBinding,
) []traceHistoricalBugID {
	bugIDs := make([]string, 0, len(occurrencesByID))
	for bugID := range occurrencesByID {
		bugIDs = append(bugIDs, bugID)
	}
	sort.Strings(bugIDs)
	entries := make([]traceHistoricalBugID, 0, len(bugIDs))
	for _, bugID := range bugIDs {
		occurrences := occurrencesByID[bugID]
		refs := make([]string, 0, len(occurrences))
		rowCovered := false
		for _, occurrence := range occurrences {
			refs = append(refs, occurrence.OccurrenceRef)
			rowCovered = rowCovered || occurrence.CoverageKind == "row_covered"
		}
		entry := traceHistoricalBugID{
			SchemaVersion:     1,
			BugID:             bugID,
			Disposition:       "historical_lesson_pending",
			OccurrenceCount:   len(occurrences),
			OccurrencesSHA256: traceStringsDigest(refs),
			ClosureEvidence:   "not_verified",
		}
		if rowCovered {
			entry.CoverageKind = "row_covered"
			entry.CapabilityIDs = append([]string(nil), rowCapabilities[bugID]...)
			entry.LessonTestRef = traceHistoricalBugPrimaryLesson(rowLessonRefs[bugID])
			entry.ReviewNote = "Exact capability union and lesson candidate from every rich Markdown row reporting this normalized BUG-ORQ ID; legacy status is not rebuild closure."
		} else {
			review := reviews[bugID]
			entry.CoverageKind = "narrative_only"
			entry.CapabilityIDs = append([]string(nil), review.CapabilityIDs...)
			entry.LessonTestRef = "planned:historical-bug-lessons/" + traceSHA256Hex(bugID)[:20]
			entry.ReviewNote = review.ReviewNote
		}
		entry.LessonState = "candidate_not_accredited"
		if strings.HasPrefix(entry.LessonTestRef, "planned:") {
			entry.LessonState = "pending_invariant_test"
		}
		if binding, exists := bindings[bugID]; exists {
			entry.VerifiedCapabilityIDs = append([]string(nil), binding.VerifiedCapabilityIDs...)
			entry.RebuildEvidenceRefs = append([]string(nil), binding.RebuildEvidenceRefs...)
		}
		entries = append(entries, entry)
	}
	return entries
}

func traceValidateHistoricalBugRebuildEvidence(t *testing.T, entry traceHistoricalBugID) {
	t.Helper()
	if len(entry.VerifiedCapabilityIDs) == 0 {
		if len(entry.RebuildEvidenceRefs) != 0 || entry.ClosureEvidence != "not_verified" {
			t.Fatalf("historical bug %q has closure without verified capability evidence: %#v", entry.BugID, entry)
		}
		return
	}
	if !sort.StringsAreSorted(entry.VerifiedCapabilityIDs) || len(entry.RebuildEvidenceRefs) == 0 {
		t.Fatalf("historical bug %q has invalid rebuild evidence: %#v", entry.BugID, entry)
	}
	capabilities := make(map[string]struct{}, len(entry.CapabilityIDs))
	for _, capabilityID := range entry.CapabilityIDs {
		capabilities[capabilityID] = struct{}{}
	}
	for index, capabilityID := range entry.VerifiedCapabilityIDs {
		if index > 0 && capabilityID == entry.VerifiedCapabilityIDs[index-1] {
			t.Fatalf("historical bug %q repeats verified capability %q", entry.BugID, capabilityID)
		}
		if _, exists := capabilities[capabilityID]; !exists {
			t.Fatalf("historical bug %q verifies unrelated capability %q", entry.BugID, capabilityID)
		}
	}
	seenEvidence := make(map[string]struct{}, len(entry.RebuildEvidenceRefs))
	for _, evidenceRef := range entry.RebuildEvidenceRefs {
		if strings.TrimSpace(evidenceRef) == "" {
			t.Fatalf("historical bug %q has empty rebuild evidence", entry.BugID)
		}
		if _, duplicate := seenEvidence[evidenceRef]; duplicate {
			t.Fatalf("historical bug %q repeats rebuild evidence %q", entry.BugID, evidenceRef)
		}
		seenEvidence[evidenceRef] = struct{}{}
	}
	if entry.ClosureEvidence != "not_verified" {
		t.Fatalf("historical bug %q inferred legacy closure from capability coverage: %#v", entry.BugID, entry)
	}
}

func traceHistoricalBugPrimaryLesson(refs []string) string {
	for _, ref := range refs {
		if !strings.HasPrefix(ref, "planned:") {
			return ref
		}
	}
	if len(refs) == 0 {
		return ""
	}
	return refs[0]
}

func traceHistoricalBugRowKey(sourceRef string, sourceLine int) string {
	return sourceRef + "\x00" + strconv.Itoa(sourceLine)
}

func traceMarkdownFirstCell(line string) string {
	cells := traceMarkdownCells(line)
	if len(cells) == 0 {
		return ""
	}
	return cells[0]
}

func traceMarkdownCells(line string) []string {
	parts := strings.Split(line, "|")
	if len(parts) < 3 || strings.TrimSpace(parts[0]) != "" {
		return nil
	}
	parts = parts[1:]
	if strings.TrimSpace(parts[len(parts)-1]) == "" {
		parts = parts[:len(parts)-1]
	}
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
	}
	return parts
}

func traceNearestLevelTwoHeading(lines []string, beforeLineIndex int) string {
	for index := beforeLineIndex - 1; index >= 0; index-- {
		if strings.HasPrefix(lines[index], "## ") {
			return strings.TrimSpace(strings.TrimPrefix(lines[index], "## "))
		}
	}
	return "not_declared"
}

func traceDocumentField(lines []string, prefix string) string {
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

func traceRequireExistingLessonRef(t *testing.T, ref string) {
	t.Helper()
	parts := strings.Split(ref, "#")
	if len(parts) > 2 || strings.TrimSpace(parts[0]) == "" {
		t.Fatalf("invalid historical bug lesson ref %q", ref)
	}
	if _, err := os.Stat(parts[0]); err != nil {
		t.Fatalf("historical bug lesson ref %q: %v", ref, err)
	}
	if len(parts) == 2 {
		if !strings.HasSuffix(parts[0], "_test.go") || !strings.HasPrefix(parts[1], "Test") {
			t.Fatalf("invalid Go historical bug lesson ref %q", ref)
		}
		traceRequireGoTestRef(t, ref, true)
	}
}
