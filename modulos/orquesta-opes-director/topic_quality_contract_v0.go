package orquestaopesdirector

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

const (
	OPESTopicQualityContractSchemaV0 = "opes_topic_quality_contract.v0"

	OPESTopicQualityStatusCompleteV0    = "complete"
	OPESTopicQualityStatusNeedsReworkV0 = "needs_rework"

	OPESTopicQualityLevelBV0         = "B"
	OPESTopicQualityMinWordsLevelBV0 = 10800

	ErrOPESTopicQualityWordsRequiredV0           = "opes_topic_words_required"
	ErrOPESTopicQualityCanonicalWordsRequiredV0  = "opes_canonical_word_count_required"
	ErrOPESTopicQualityMinWordsNotMetV0          = "needs_expansion_min_words_B"
	ErrOPESTopicQualityInvalidReportContractV0   = "invalid_report_contract"
	ErrOPESTopicQualityPublicMojibakeV0          = "opes_public_text_mojibake"
	ErrOPESTopicQualityPublicMetacommentV0       = "opes_public_text_metacomment"
	ErrOPESTopicQualityStudyScaffoldingV0        = "opes_public_text_study_scaffolding"
	ErrOPESTopicQualityStructuralContaminationV0 = "opes_public_text_structural_contamination"
	ErrOPESTopicQualityDidacticVisualRequiredV0  = "opes_visual_didactic_function_required"
)

type OPESTopicQualityContractRequestV0 struct {
	TopicRef               string
	Level                  string
	Text                   string
	CanonicalWordCount     int
	CanonicalWordCountText string
	WordCount              int
	WordCountText          string
	WordCountSource        string
	ReportStatus           string
	ReportText             string
	RequireDidacticVisual  bool
	Visuals                []OPESTopicQualityVisualEvidenceV0
	EvidenceRefs           []string
}

type OPESTopicQualityVisualEvidenceV0 struct {
	VisualRef        string
	Raster           bool
	AnchorRef        string
	DidacticFunction string
	EvidenceRefs     []string
}

type OPESTopicQualityContractResultV0 struct {
	SchemaVersion   string                           `json:"schema_version"`
	Status          string                           `json:"status"`
	TopicRef        string                           `json:"topic_ref,omitempty"`
	Level           string                           `json:"level,omitempty"`
	WordCount       int                              `json:"word_count,omitempty"`
	WordCountSource string                           `json:"word_count_source,omitempty"`
	MinWords        int                              `json:"min_words,omitempty"`
	ReportStatus    string                           `json:"report_status,omitempty"`
	EvidenceRefs    []string                         `json:"evidence_refs,omitempty"`
	Issues          []OPESTopicQualityIssueV0        `json:"issues,omitempty"`
	Visuals         []OPESTopicQualityVisualResultV0 `json:"visuals,omitempty"`
}

var (
	opesTopicQualityWordReV0           = regexp.MustCompile(`[\p{L}\p{N}]+([-'][\p{L}\p{N}]+)?`)
	opesTopicQualityCodeFenceReV0      = regexp.MustCompile("(?s)```.*?```")
	opesTopicQualityInlineCodeReV0     = regexp.MustCompile("`[^`]*`")
	opesTopicQualityMarkdownAnchorReV0 = regexp.MustCompile(`\{#[A-Za-z0-9][A-Za-z0-9_-]*\}`)
	opesTopicQualityHeadingTableReV0   = regexp.MustCompile(`(?m)^#{1,6}\s+.*\|.*\|`)
)

type OPESTopicQualityVisualResultV0 struct {
	VisualRef        string   `json:"visual_ref,omitempty"`
	Raster           bool     `json:"raster,omitempty"`
	AnchorRef        string   `json:"anchor_ref,omitempty"`
	DidacticFunction string   `json:"didactic_function,omitempty"`
	EvidenceRefs     []string `json:"evidence_refs,omitempty"`
}

type OPESTopicQualityIssueV0 struct {
	Code         string   `json:"code"`
	Field        string   `json:"field,omitempty"`
	Message      string   `json:"message,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

func ValidateOPESTopicQualityContractV0(
	request OPESTopicQualityContractRequestV0,
) OPESTopicQualityContractResultV0 {
	request = normalizeOPESTopicQualityContractRequestV0(request)
	result := OPESTopicQualityContractResultV0{
		SchemaVersion:   OPESTopicQualityContractSchemaV0,
		Status:          OPESTopicQualityStatusCompleteV0,
		TopicRef:        request.TopicRef,
		Level:           request.Level,
		WordCount:       opesTopicQualityWordCountV0(request),
		WordCountSource: opesTopicQualityWordCountSourceV0(request),
		MinWords:        opesTopicQualityMinWordsV0(request.Level),
		ReportStatus:    request.ReportStatus,
		EvidenceRefs:    compactStringsV0(request.EvidenceRefs),
		Visuals:         opesTopicQualityVisualResultsV0(request.Visuals),
	}
	if result.WordCount <= 0 {
		result.Issues = append(result.Issues, opesTopicQualityIssueV0(
			ErrOPESTopicQualityWordsRequiredV0,
			"word_count",
			"word count is required for OPES public topic quality validation",
		))
	}
	if opesTopicQualityWeakWordCountSourceV0(request.WordCountSource) &&
		parseOPESTopicQualityWordCountV0(request.CanonicalWordCountText) <= 0 &&
		request.CanonicalWordCount <= 0 {
		result.Issues = append(result.Issues, opesTopicQualityIssueV0(
			ErrOPESTopicQualityCanonicalWordsRequiredV0,
			"canonical_word_count",
			"OPES level closure requires canonical strict word count, not wc output",
		))
	}
	if result.MinWords > 0 && result.WordCount > 0 && result.WordCount < result.MinWords {
		result.Issues = append(result.Issues, opesTopicQualityIssueV0(
			ErrOPESTopicQualityMinWordsNotMetV0,
			"word_count",
			"topic text is below OPES level B minimum words",
		))
	}
	result.Issues = append(result.Issues, opesTopicQualityReportIssuesV0(request, result)...)
	for _, phrase := range opesTopicPublicMojibakeMatchesV0(request.Text) {
		result.Issues = append(result.Issues, OPESTopicQualityIssueV0{
			Code:    ErrOPESTopicQualityPublicMojibakeV0,
			Field:   "text",
			Message: phrase,
		})
	}
	for _, phrase := range opesTopicPublicMetacommentMatchesV0(request.Text) {
		result.Issues = append(result.Issues, OPESTopicQualityIssueV0{
			Code:    ErrOPESTopicQualityPublicMetacommentV0,
			Field:   "text",
			Message: phrase,
		})
	}
	for _, phrase := range opesTopicPublicStudyScaffoldingMatchesV0(request.Text) {
		result.Issues = append(result.Issues, OPESTopicQualityIssueV0{
			Code:    ErrOPESTopicQualityStudyScaffoldingV0,
			Field:   "text",
			Message: phrase,
		})
	}
	for _, phrase := range opesTopicPublicStructuralContaminationMatchesV0(request.Text) {
		result.Issues = append(result.Issues, OPESTopicQualityIssueV0{
			Code:    ErrOPESTopicQualityStructuralContaminationV0,
			Field:   "text",
			Message: phrase,
		})
	}
	if request.RequireDidacticVisual && !opesTopicHasDidacticVisualV0(request.Visuals) {
		result.Issues = append(result.Issues, opesTopicQualityIssueV0(
			ErrOPESTopicQualityDidacticVisualRequiredV0,
			"visuals",
			"raster visual evidence must include didactic function and topic anchor",
		))
	}
	if len(result.Issues) > 0 {
		result.Status = OPESTopicQualityStatusNeedsReworkV0
	}
	return result
}

func normalizeOPESTopicQualityContractRequestV0(
	request OPESTopicQualityContractRequestV0,
) OPESTopicQualityContractRequestV0 {
	request.TopicRef = strings.TrimSpace(request.TopicRef)
	request.Level = strings.ToUpper(strings.TrimSpace(request.Level))
	request.Text = strings.TrimSpace(request.Text)
	request.CanonicalWordCountText = strings.TrimSpace(request.CanonicalWordCountText)
	request.WordCountText = strings.TrimSpace(request.WordCountText)
	request.WordCountSource = strings.ToLower(strings.TrimSpace(request.WordCountSource))
	request.ReportStatus = strings.ToLower(strings.TrimSpace(request.ReportStatus))
	request.ReportText = strings.TrimSpace(request.ReportText)
	request.EvidenceRefs = compactStringsV0(request.EvidenceRefs)
	for index := range request.Visuals {
		request.Visuals[index].VisualRef = strings.TrimSpace(request.Visuals[index].VisualRef)
		request.Visuals[index].AnchorRef = strings.TrimSpace(request.Visuals[index].AnchorRef)
		request.Visuals[index].DidacticFunction = strings.TrimSpace(request.Visuals[index].DidacticFunction)
		request.Visuals[index].EvidenceRefs = compactStringsV0(request.Visuals[index].EvidenceRefs)
	}
	return request
}

func opesTopicQualityWordCountV0(request OPESTopicQualityContractRequestV0) int {
	if request.CanonicalWordCount > 0 {
		return request.CanonicalWordCount
	}
	if parsed := parseOPESTopicQualityWordCountV0(request.CanonicalWordCountText); parsed > 0 {
		return parsed
	}
	if request.WordCount > 0 {
		return request.WordCount
	}
	if parsed := parseOPESTopicQualityWordCountV0(request.WordCountText); parsed > 0 {
		return parsed
	}
	return countOPESTopicWordsV0(request.Text)
}

func opesTopicQualityMinWordsV0(level string) int {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case OPESTopicQualityLevelBV0:
		return OPESTopicQualityMinWordsLevelBV0
	default:
		return 0
	}
}

func opesTopicQualityWordCountSourceV0(request OPESTopicQualityContractRequestV0) string {
	if request.CanonicalWordCount > 0 || parseOPESTopicQualityWordCountV0(request.CanonicalWordCountText) > 0 {
		return "canonical"
	}
	if request.WordCountSource != "" {
		return request.WordCountSource
	}
	if request.WordCount > 0 || request.WordCountText != "" {
		return "declared"
	}
	if request.Text != "" {
		return "text"
	}
	return ""
}

func parseOPESTopicQualityWordCountV0(value string) int {
	token := firstOPESTopicQualityNumberTokenV0(value)
	if token == "" {
		return 0
	}
	normalized := strings.NewReplacer(".", "", ",", "", " ", "").Replace(token)
	parsed, err := strconv.Atoi(normalized)
	if err != nil || parsed <= 0 {
		return 0
	}
	return parsed
}

func firstOPESTopicQualityNumberTokenV0(value string) string {
	var b strings.Builder
	started := false
	for _, r := range value {
		if unicode.IsDigit(r) || (started && (r == '.' || r == ',' || unicode.IsSpace(r))) {
			b.WriteRune(r)
			started = true
			continue
		}
		if started {
			break
		}
	}
	return strings.TrimSpace(b.String())
}

func countOPESTopicWordsV0(value string) int {
	value = opesTopicQualityCodeFenceReV0.ReplaceAllString(value, " ")
	value = opesTopicQualityInlineCodeReV0.ReplaceAllString(value, " ")
	return len(opesTopicQualityWordReV0.FindAllString(value, -1))
}

func opesTopicQualityWeakWordCountSourceV0(source string) bool {
	source = strings.ToLower(strings.TrimSpace(source))
	switch source {
	case "wc", "wc -w", "wc_w", "shell_wc", "unix_wc":
		return true
	default:
		return strings.Contains(source, "wc -w") || strings.Contains(source, "wc")
	}
}

func opesTopicQualityReportIssuesV0(
	request OPESTopicQualityContractRequestV0,
	result OPESTopicQualityContractResultV0,
) []OPESTopicQualityIssueV0 {
	status := strings.ToLower(strings.TrimSpace(request.ReportStatus))
	textClaimsPass := opesTopicQualityReportClaimsPassV0(request.ReportText)
	if opesTopicQualityReportStatusFailsV0(status) && textClaimsPass {
		return []OPESTopicQualityIssueV0{opesTopicQualityIssueV0(
			ErrOPESTopicQualityInvalidReportContractV0,
			"report",
			"quality report is failing but its narrative claims the OPES minimum passes",
		)}
	}
	if opesTopicQualityReportStatusPassesV0(status) &&
		result.MinWords > 0 &&
		result.WordCount > 0 &&
		result.WordCount < result.MinWords {
		return []OPESTopicQualityIssueV0{opesTopicQualityIssueV0(
			ErrOPESTopicQualityInvalidReportContractV0,
			"report",
			"quality report claims pass while canonical word count is below minimum",
		)}
	}
	return nil
}

func opesTopicQualityReportStatusFailsV0(status string) bool {
	switch status {
	case "fail", "failed", "falla", "invalid", "needs_rework", "rework", "blocked":
		return true
	default:
		return false
	}
}

func opesTopicQualityReportStatusPassesV0(status string) bool {
	switch status {
	case "pass", "passed", "ok", "complete", "completed", "valid":
		return true
	default:
		return false
	}
}

func opesTopicQualityReportClaimsPassV0(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}
	patterns := []string{
		"supera 10.800",
		"supera el minimo",
		"supera el mínimo",
		"cumple el minimo",
		"cumple el mínimo",
		"alcanza el minimo",
		"alcanza el mínimo",
	}
	for _, pattern := range patterns {
		if strings.Contains(normalized, pattern) {
			return true
		}
	}
	return false
}

func opesTopicPublicMojibakeMatchesV0(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	patterns := []struct {
		pattern string
		label   string
	}{
		{pattern: "\u00c3", label: "mojibake:utf8_latin1_marker_c3"},
		{pattern: "\u00c2", label: "mojibake:utf8_latin1_marker_c2"},
		{pattern: "\ufffd", label: "mojibake:replacement_character"},
		{pattern: "\u00e2\u20ac", label: "mojibake:windows_punctuation_marker"},
	}
	var matches []string
	for _, pattern := range patterns {
		if strings.Contains(text, pattern.pattern) {
			matches = append(matches, pattern.label)
		}
	}
	normalized := strings.ToLower(text)
	orthographicCorruptions := []struct {
		pattern string
		label   string
	}{
		{pattern: "\u00f3rga\u00f1os", label: "orthographic_corruption:organos"},
		{pattern: "conf\u00edanza", label: "orthographic_corruption:confianza"},
		{pattern: "instituci\u00f3nal", label: "orthographic_corruption:institucional"},
		{pattern: "propuest\u00e1", label: "orthographic_corruption:propuesta"},
	}
	for _, pattern := range orthographicCorruptions {
		if strings.Contains(normalized, pattern.pattern) {
			matches = append(matches, pattern.label)
		}
	}
	return compactStringsV0(matches)
}

func opesTopicPublicMetacommentMatchesV0(text string) []string {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return nil
	}
	patterns := []string{
		"en examen",
		"para examen",
		"nota de test",
		"para este tema basta",
		"para el nivel de este tema",
		"lo evaluable en este tema",
		"debe estudiarse",
	}
	var matches []string
	for _, pattern := range patterns {
		if strings.Contains(normalized, pattern) {
			matches = append(matches, pattern)
		}
	}
	return compactStringsV0(matches)
}

func opesTopicPublicStudyScaffoldingMatchesV0(text string) []string {
	normalized := opesTopicQualityNormalizePublicPhraseTextV0(text)
	if normalized == "" {
		return nil
	}
	patterns := []string{
		"preguntas de recuperacion",
		"repaso espaciado",
		"dia 0",
		"mapa mental",
		"la respuesta debe empezar",
		"una respuesta fuerte empieza",
		"reconstruye sin mirar",
		"calendario de estudio",
		"trazabilidad b",
		"derivaciones futuras",
		"fuentes internas",
		"bloque de canon",
		"bloques de canon",
		"bloque maestro",
		"bloques de otros temas",
		"referencias a svg",
		"referencias a .md",
	}
	var matches []string
	for _, pattern := range patterns {
		if opesTopicQualityContainsPublicPhraseV0(normalized, pattern) {
			matches = append(matches, pattern)
		}
	}
	return compactStringsV0(matches)
}

func opesTopicPublicStructuralContaminationMatchesV0(text string) []string {
	normalized := opesTopicQualityNormalizePublicTextV0(text)
	if normalized == "" {
		return nil
	}
	patterns := []string{
		"bloque ajeno",
		"bloque de otro tema",
		"bloques de otro tema",
		"bloques de otros temas",
		"contenido injertado",
		"contaminacion cruzada",
		"contaminacion estructural",
		"fuente de tema ajeno",
		"fuentes de temas ajenos",
		"procedencia de bloques",
		"canon maestro",
		"bloque de canon",
		"bloque maestro",
		"derivacion futura",
		"derivaciones futuras",
		"trazabilidad b",
		"referencia interna a canon",
		"referencias internas a canones",
		"visual svg",
		"archivo svg",
		"referencia svg",
		"referencia a .svg",
		"referencias a .svg",
		"referencia a .md",
		"referencias a .md",
		"tema_",
		"/tema_",
		"02_temas/",
		"00_control/",
		"visuales_plan.md",
		"tema_ampliado.md",
		"tema_resumen.md",
	}
	var matches []string
	if opesTopicQualityMarkdownAnchorReV0.MatchString(text) {
		matches = append(matches, "markdown_anchor")
	}
	if opesTopicQualityHeadingTableReV0.MatchString(text) {
		matches = append(matches, "markdown_heading_table")
	}
	for _, pattern := range patterns {
		if strings.Contains(normalized, pattern) {
			matches = append(matches, pattern)
		}
	}
	return compactStringsV0(matches)
}

func opesTopicQualityNormalizePublicTextV0(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return ""
	}
	return opesTopicQualityStripPublicTextDiacriticsV0(text)
}

func opesTopicQualityNormalizePublicPhraseTextV0(text string) string {
	text = opesTopicQualityNormalizePublicTextV0(text)
	if text == "" {
		return ""
	}
	var b strings.Builder
	lastSpace := true
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastSpace = false
			continue
		}
		if !lastSpace {
			b.WriteByte(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

func opesTopicQualityStripPublicTextDiacriticsV0(text string) string {
	var b strings.Builder
	for _, r := range text {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		switch r {
		case '\u00e1', '\u00e0', '\u00e4', '\u00e2':
			b.WriteByte('a')
		case '\u00e9', '\u00e8', '\u00eb', '\u00ea':
			b.WriteByte('e')
		case '\u00ed', '\u00ec', '\u00ef', '\u00ee':
			b.WriteByte('i')
		case '\u00f3', '\u00f2', '\u00f6', '\u00f4':
			b.WriteByte('o')
		case '\u00fa', '\u00f9', '\u00fc', '\u00fb':
			b.WriteByte('u')
		case '\u00f1':
			b.WriteByte('n')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func opesTopicQualityContainsPublicPhraseV0(normalizedText string, pattern string) bool {
	normalizedPattern := opesTopicQualityNormalizePublicPhraseTextV0(pattern)
	if normalizedPattern == "" {
		return false
	}
	return strings.Contains(" "+normalizedText+" ", " "+normalizedPattern+" ")
}

func opesTopicHasDidacticVisualV0(visuals []OPESTopicQualityVisualEvidenceV0) bool {
	for _, visual := range visuals {
		if visual.Raster &&
			strings.TrimSpace(visual.AnchorRef) != "" &&
			strings.TrimSpace(visual.DidacticFunction) != "" {
			return true
		}
	}
	return false
}

func opesTopicQualityVisualResultsV0(
	visuals []OPESTopicQualityVisualEvidenceV0,
) []OPESTopicQualityVisualResultV0 {
	out := make([]OPESTopicQualityVisualResultV0, 0, len(visuals))
	for _, visual := range visuals {
		visual.VisualRef = strings.TrimSpace(visual.VisualRef)
		visual.AnchorRef = strings.TrimSpace(visual.AnchorRef)
		visual.DidacticFunction = strings.TrimSpace(visual.DidacticFunction)
		out = append(out, OPESTopicQualityVisualResultV0{
			VisualRef:        visual.VisualRef,
			Raster:           visual.Raster,
			AnchorRef:        visual.AnchorRef,
			DidacticFunction: visual.DidacticFunction,
			EvidenceRefs:     compactStringsV0(visual.EvidenceRefs),
		})
	}
	return out
}

func opesTopicQualityIssueV0(code string, field string, message string) OPESTopicQualityIssueV0 {
	return OPESTopicQualityIssueV0{
		Code:    strings.TrimSpace(code),
		Field:   strings.TrimSpace(field),
		Message: strings.TrimSpace(message),
	}
}
