package orquestaweb

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const WizardBotReplySchemaV0 = "web_nueva_app_wizard_bot_reply.v0"

type WizardBotTurnV0 struct {
	SessionRef string `json:"session_ref"`
	UserText   string `json:"user_text"`
	Locale     string `json:"locale,omitempty"`
}

type WizardBotReplyV0 struct {
	SchemaVersion string             `json:"schema_version"`
	Say           string             `json:"say"`
	TurnResult    WizardTurnResultV0 `json:"turn_result"`
	FilledAnswers []WizardAnswerV0   `json:"filled_answers,omitempty"`
	GroundingRefs []string           `json:"grounding_refs"`
	LLMStatus     string             `json:"llm_status,omitempty"`
	CostNotice    string             `json:"cost_notice,omitempty"`
}

type WizardBotLLMAssistPortV0 interface {
	AssistWizardBotTurnV0(context.Context, WizardBotLLMRequestV0) (WizardBotLLMResultV0, error)
}

type WizardBotLLMRequestV0 struct {
	SessionRef        string                        `json:"session_ref"`
	UserText          string                        `json:"user_text"`
	Locale            string                        `json:"locale,omitempty"`
	TurnResult        WizardTurnResultV0            `json:"turn_result"`
	GroundingSnippets []WizardBotGroundingSnippetV0 `json:"grounding_snippets"`
	AllowedAnswers    []WizardBotAllowedAnswerV0    `json:"allowed_answers"`
}

type WizardBotGroundingSnippetV0 struct {
	Ref  string `json:"ref"`
	Kind string `json:"kind"`
	Text string `json:"text"`
}

type WizardBotAllowedAnswerV0 struct {
	QuestionRef string `json:"question_ref"`
	UserChoice  string `json:"user_choice"`
}

type WizardBotLLMResultV0 struct {
	Status        string           `json:"status"`
	Say           string           `json:"say,omitempty"`
	FilledAnswers []WizardAnswerV0 `json:"filled_answers,omitempty"`
	GroundingRefs []string         `json:"grounding_refs,omitempty"`
	EvidenceRefs  []string         `json:"evidence_refs,omitempty"`
}

type wizardRAGCorpusEntryV0 struct {
	Ref      string
	Kind     string
	Key      string
	Question string
	Option   string
	Texts    []string
	Synonyms []string
}

type wizardLexicalMatchV0 struct {
	Entry wizardRAGCorpusEntryV0
	Score int
}

func NewWebNuevaAppWizardBotReplyV0(
	session WebNuevaAppIntakeSessionV0,
	turn WizardBotTurnV0,
) (WebNuevaAppIntakeSessionV0, WizardBotReplyV0) {
	return NewWebNuevaAppWizardBotReplyWithLLMV0(context.Background(), session, turn, nil)
}

func NewWebNuevaAppWizardBotReplyWithLLMV0(
	ctx context.Context,
	session WebNuevaAppIntakeSessionV0,
	turn WizardBotTurnV0,
	assistant WizardBotLLMAssistPortV0,
) (WebNuevaAppIntakeSessionV0, WizardBotReplyV0) {
	session = refreshWebNuevaAppIntakeSessionV0(session)
	if trimV0(turn.SessionRef) != "" {
		session.SessionRef = trimV0(turn.SessionRef)
		session.SessionID = trimV0(turn.SessionRef)
	}
	if trimV0(turn.Locale) != "" {
		session.Form.Locale = trimV0(turn.Locale)
	}
	text := trimV0(turn.UserText)
	current := NewWebNuevaAppWizardTurnResultV0(session)
	corpus := NewWebNuevaAppWizardRAGCorpusV0(current)
	index := newWizardLexicalIndexV0(corpus)

	if wizardBotAcceptsRecommendationsV0(text) {
		session, turns := AcceptWebNuevaAppWizardRecommendationsV0(session, 12)
		applied := NewWebNuevaAppWizardTurnResultV0(session)
		if len(turns) > 0 {
			applied = turns[len(turns)-1]
		}
		return session, wizardBotReplyV0(turn.Locale, applied, nil, wizardBotSayForTurnV0(turn.Locale, applied, corpus, "He aplicado las recomendaciones del wizard."), wizardGroundingRefsForTurnV0(applied))
	}

	if query, ok := wizardBotComprehensionQueryV0(text); ok {
		matches := index.Search(query, 3)
		if len(matches) == 0 {
			say := wizardBotUnknownSayV0(turn.Locale, query, current, corpus)
			return session, wizardBotReplyV0(turn.Locale, current, nil, say, wizardGroundingRefsForTurnV0(current))
		}
		refs := wizardMatchRefsV0(matches)
		say := wizardBotSayForMatchesV0(turn.Locale, matches, current, corpus)
		return session, wizardBotReplyV0(turn.Locale, current, nil, say, appendUniqueStringsV0(refs, wizardGroundingRefsForTurnV0(current)...))
	}

	answers := wizardBotDeterministicAnswersV0(text, current)
	if len(answers) > 0 {
		var applied WizardTurnResultV0
		session, applied = ApplyWebNuevaAppWizardAnswersV0(session, answers)
		return session, wizardBotReplyV0(turn.Locale, applied, answers, wizardBotSayForTurnV0(turn.Locale, applied, NewWebNuevaAppWizardRAGCorpusV0(applied), "He registrado tu respuesta."), wizardGroundingRefsForAnswersV0(applied, answers))
	}
	if assistant != nil && text != "" {
		if assistedSession, reply, ok := wizardBotTryLLMAssistV0(ctx, session, turn, current, corpus, assistant); ok {
			return assistedSession, reply
		}
	}

	return session, wizardBotReplyV0(turn.Locale, current, nil, wizardBotSayForTurnV0(turn.Locale, current, corpus, ""), wizardGroundingRefsForTurnV0(current))
}

func NewWebNuevaAppWizardRAGCorpusV0(turn WizardTurnResultV0) []wizardRAGCorpusEntryV0 {
	catalog := NewNuevaAppI18nCatalogV0()
	allQuestions, _ := webNuevaAppWizardAllGapQuestionsWithResolvedV0(WebNuevaAppFormV0{
		Locale:      "es-ES",
		Nombre:      "Nueva app",
		Objetivo:    "agenda tienda mapa inventario notas tareas finanzas contactos reservas salud educacion comunidad iot galeria facturacion app de equipo con datos servidor movil",
		TipoApp:     "web",
		Descripcion: "flujo diario con datos, integraciones y usuarios compartidos",
		Datos:       WebNuevaAppDatosFormV0{DBRequired: true},
		Deploy:      WebNuevaAppDeployFormV0{Target: "desktop"},
		Integraciones: []WebNuevaAppConnectorFormV0{{
			Tipo:   "api",
			Nombre: "externa",
		}},
		UsuariosObjetivo: []string{"equipo"},
	})
	questions := dedupeWizardQuestionsV0(append(allQuestions, turn.Questions...))
	var out []wizardRAGCorpusEntryV0
	for _, question := range questions {
		out = append(out, wizardRAGCorpusEntryV0{
			Ref:      "wizard-corpus:question:" + question.QuestionRef,
			Kind:     "question",
			Key:      question.PromptKey,
			Question: question.QuestionRef,
			Texts:    wizardCatalogTextsV0(catalog, question.PromptKey, question.WhyKey, question.HelpKey),
			Synonyms: []string{question.QuestionRef, question.Field, question.TopicGroup},
		})
		for _, option := range question.Options {
			texts := wizardCatalogTextsV0(catalog, option.LabelKey, option.HelpKey, option.ExampleKey, option.RationaleKey)
			out = append(out, wizardRAGCorpusEntryV0{
				Ref:      "wizard-corpus:option:" + question.QuestionRef + ":" + option.Value,
				Kind:     "option",
				Key:      option.LabelKey,
				Question: question.QuestionRef,
				Option:   option.Value,
				Texts:    append([]string{option.Value}, texts...),
				Synonyms: wizardSynonymsForValueV0(option.Value, texts...),
			})
		}
	}
	for _, defaultValue := range WebNuevaAppWizardEngineeringDefaultsV0() {
		out = append(out, wizardRAGCorpusEntryV0{
			Ref:      "wizard-corpus:default:" + defaultValue.Value,
			Kind:     "default",
			Key:      defaultValue.Value,
			Texts:    append([]string{defaultValue.Area, defaultValue.Value}, wizardCatalogTextsV0(catalog, defaultValue.WhyKey, defaultValue.HelpKey, defaultValue.ExampleKey)...),
			Synonyms: wizardSynonymsForValueV0(defaultValue.Value, defaultValue.Area),
		})
	}
	return dedupeWizardCorpusEntriesV0(out)
}

func wizardCatalogTextsV0(catalog NuevaAppI18nCatalogV0, keys ...string) []string {
	var out []string
	for _, key := range compactStringsV0(keys) {
		out = append(out, key)
		for _, locale := range []string{NuevaAppI18nDefaultLocaleV0, NuevaAppI18nEnglishLocaleV0} {
			if text := catalog.lookupExact(locale, key); text != "" {
				out = append(out, text)
			}
		}
	}
	return compactStringsV0(out)
}

func dedupeWizardCorpusEntriesV0(entries []wizardRAGCorpusEntryV0) []wizardRAGCorpusEntryV0 {
	seen := map[string]bool{}
	out := make([]wizardRAGCorpusEntryV0, 0, len(entries))
	for _, entry := range entries {
		if trimV0(entry.Ref) == "" || seen[entry.Ref] {
			continue
		}
		seen[entry.Ref] = true
		out = append(out, entry)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Ref < out[j].Ref })
	return out
}

type wizardLexicalIndexV0 struct {
	entries []wizardRAGCorpusEntryV0
	tokens  map[string]map[int]bool
}

func newWizardLexicalIndexV0(entries []wizardRAGCorpusEntryV0) wizardLexicalIndexV0 {
	index := wizardLexicalIndexV0{entries: entries, tokens: map[string]map[int]bool{}}
	for i, entry := range entries {
		for _, token := range wizardEntryTokensV0(entry) {
			if index.tokens[token] == nil {
				index.tokens[token] = map[int]bool{}
			}
			index.tokens[token][i] = true
		}
	}
	return index
}

func (index wizardLexicalIndexV0) Search(query string, limit int) []wizardLexicalMatchV0 {
	queryTokens := wizardNormalizedTokensV0(query)
	scores := map[int]int{}
	for _, token := range queryTokens {
		for candidate, ok := range index.tokens[token] {
			if ok {
				scores[candidate] += 4
			}
		}
		for indexedToken, candidates := range index.tokens {
			if token == indexedToken || !wizardTokensSimilarV0(token, indexedToken) {
				continue
			}
			for candidate := range candidates {
				scores[candidate]++
			}
		}
	}
	matches := make([]wizardLexicalMatchV0, 0, len(scores))
	for candidate, score := range scores {
		if score >= 2 {
			matches = append(matches, wizardLexicalMatchV0{Entry: index.entries[candidate], Score: score})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			return matches[i].Entry.Ref < matches[j].Entry.Ref
		}
		return matches[i].Score > matches[j].Score
	})
	if limit > 0 && len(matches) > limit {
		return matches[:limit]
	}
	return matches
}

func wizardEntryTokensV0(entry wizardRAGCorpusEntryV0) []string {
	return wizardNormalizedTokensV0(strings.Join(append(append([]string{entry.Ref, entry.Key, entry.Question, entry.Option}, entry.Texts...), entry.Synonyms...), " "))
}

func wizardNormalizedTokensV0(value string) []string {
	normalized := normalizeGuidedNeedV0(wizardFoldAccentsV0(value))
	fields := strings.FieldsFunc(normalized, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	seen := map[string]bool{}
	var out []string
	for _, field := range fields {
		token := wizardSingularTokenV0(field)
		if len(token) < 2 || seen[token] {
			continue
		}
		seen[token] = true
		out = append(out, token)
	}
	return out
}

func wizardSingularTokenV0(token string) string {
	token = strings.TrimSpace(token)
	for _, suffix := range []string{"ciones", "es", "s"} {
		if len(token) > len(suffix)+3 && strings.HasSuffix(token, suffix) {
			if suffix == "ciones" {
				return strings.TrimSuffix(token, suffix) + "cion"
			}
			return strings.TrimSuffix(token, suffix)
		}
	}
	return token
}

func wizardFoldAccentsV0(value string) string {
	replacer := strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n", "Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U", "Ü", "U", "Ñ", "N")
	return replacer.Replace(value)
}

func wizardTokensSimilarV0(a, b string) bool {
	if len(a) < 4 || len(b) < 4 {
		return false
	}
	if strings.HasPrefix(a, b) || strings.HasPrefix(b, a) {
		return true
	}
	if absIntV0(len(a)-len(b)) > 2 {
		return false
	}
	return wizardEditDistanceAtMostOneV0(a, b)
}

func wizardEditDistanceAtMostOneV0(a, b string) bool {
	if a == b {
		return true
	}
	if absIntV0(len(a)-len(b)) > 1 {
		return false
	}
	i, j, edits := 0, 0, 0
	for i < len(a) && j < len(b) {
		if a[i] == b[j] {
			i++
			j++
			continue
		}
		edits++
		if edits > 1 {
			return false
		}
		if len(a) > len(b) {
			i++
		} else if len(b) > len(a) {
			j++
		} else {
			i++
			j++
		}
	}
	return edits+(len(a)-i)+(len(b)-j) <= 1
}

func absIntV0(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func wizardSynonymsForValueV0(value string, texts ...string) []string {
	base := []string{value, strings.ReplaceAll(value, "_", " ")}
	normalized := normalizeGuidedNeedV0(strings.Join(append([]string{value}, texts...), " "))
	switch {
	case guidedContainsAnyV0(normalized, "calendar", "calendario", "agenda"):
		base = append(base, "agenda", "calendario", "citas")
	case guidedContainsAnyV0(normalized, "log", "observabilidad"):
		base = append(base, "logs", "registro", "trazas")
	case guidedContainsAnyV0(normalized, "auth", "oauth", "identidad"):
		base = append(base, "auth", "login", "identidad", "acceso")
	case guidedContainsAnyV0(normalized, "contenedor", "deploy", "despliegue"):
		base = append(base, "deploy", "despliegue", "publicar")
	case guidedContainsAnyV0(normalized, "persistencia", "storage", "datos"):
		base = append(base, "datos", "storage", "guardado")
	}
	return compactStringsV0(base)
}

func wizardBotAcceptsRecommendationsV0(text string) bool {
	normalized := normalizeGuidedNeedV0(wizardFoldAccentsV0(text))
	return guidedContainsAnyV0(normalized, "acepta lo que recomiendes", "acepto lo recomendado", "recomendaciones", "lo recomendado")
}

func wizardBotComprehensionQueryV0(text string) (string, bool) {
	normalized := strings.Trim(normalizeGuidedNeedV0(wizardFoldAccentsV0(text)), " ?")
	for _, prefix := range []string{"que es ", "que significa ", "explica ", "opciones de ", "opciones para ", "que opciones hay de ", "cual me recomiendas para ", "cual recomiendas para "} {
		if strings.HasPrefix(normalized, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(normalized, prefix)), true
		}
	}
	return "", false
}

func wizardBotDeterministicAnswersV0(text string, turn WizardTurnResultV0) []WizardAnswerV0 {
	normalized := normalizeGuidedNeedV0(wizardFoldAccentsV0(text))
	if normalized == "" || len(turn.Questions) == 0 {
		return nil
	}
	parts := strings.FieldsFunc(normalized, func(r rune) bool { return r == ',' || r == ';' || r == '\n' })
	if len(parts) == 0 {
		parts = []string{normalized}
	}
	var answers []WizardAnswerV0
	for _, part := range parts {
		answer, ok := wizardBotAnswerForTextPartV0(strings.TrimSpace(part), turn)
		if ok && !wizardAnswerAlreadyFilledV0(answers, answer.QuestionRef) {
			answers = append(answers, answer)
		}
	}
	return answers
}

func wizardBotAnswerForTextPartV0(text string, turn WizardTurnResultV0) (WizardAnswerV0, bool) {
	if number, err := strconv.Atoi(text); err == nil && number > 0 && number <= len(turn.Questions[0].Options) {
		option := turn.Questions[0].Options[number-1]
		return WizardAnswerV0{QuestionRef: turn.Questions[0].QuestionRef, UserChoice: option.Value}, true
	}
	for _, question := range turn.Questions {
		if answer, ok := wizardBotSemanticAnswerV0(text, question); ok {
			return answer, true
		}
		for index, option := range question.Options {
			if text == strconv.Itoa(index+1) || wizardTextMatchesOptionV0(text, option) {
				return WizardAnswerV0{QuestionRef: question.QuestionRef, UserChoice: option.Value}, true
			}
		}
	}
	return WizardAnswerV0{}, false
}

func wizardBotSemanticAnswerV0(text string, question WizardQuestionV0) (WizardAnswerV0, bool) {
	hasOption := func(value string) bool {
		for _, option := range question.Options {
			if option.Value == value {
				return true
			}
		}
		return false
	}
	answer := func(value string) (WizardAnswerV0, bool) {
		if !hasOption(value) {
			return WizardAnswerV0{}, false
		}
		return WizardAnswerV0{QuestionRef: question.QuestionRef, UserChoice: value}, true
	}
	switch question.QuestionRef {
	case "wizard-u1-audiencia", "wizard-r8-usuarios-compartido":
		if guidedContainsAnyV0(text, "empresa", "equipo", "organizacion", "oficina", "usuarios") {
			return answer("equipo")
		}
	case "wizard-u7-integraciones":
		if guidedContainsAnyV0(text, "correo", "email", "mail", "api", "servicio externo") {
			return answer("api")
		}
	case "wizard-r3-integracion-agenda":
		if guidedContainsAnyV0(text, "agenda", "calendario", "calendar", "citas") {
			if guidedContainsAnyV0(text, "microsoft", "outlook", "365", "teams") {
				return answer("calendar_microsoft_365")
			}
			if guidedContainsAnyV0(text, "google", "workspace", "gmail") {
				return answer("calendar_google_workspace")
			}
			return answer("calendar_enterprise")
		}
	case "wizard-t2-identidad-corporativa":
		if guidedContainsAnyV0(text, "ad", "active directory", "ldap", "dominio windows") {
			return answer("ldap_bind")
		}
		if guidedContainsAnyV0(text, "oidc", "sso", "login corporativo") {
			return answer("oidc_sso")
		}
	}
	return WizardAnswerV0{}, false
}

func wizardTextMatchesOptionV0(text string, option WizardOptionV0) bool {
	catalog := NewNuevaAppI18nCatalogV0()
	candidates := append([]string{option.Value, strings.ReplaceAll(option.Value, "_", " ")}, wizardCatalogTextsV0(catalog, option.LabelKey, option.HelpKey)...)
	needleTokens := wizardNormalizedTokensV0(text)
	for _, candidate := range candidates {
		haystack := " " + strings.Join(wizardNormalizedTokensV0(candidate), " ") + " "
		if haystack == "  " {
			continue
		}
		matched := 0
		for _, token := range needleTokens {
			if strings.Contains(haystack, " "+token+" ") {
				matched++
			}
		}
		if matched > 0 && matched == len(needleTokens) {
			return true
		}
	}
	return false
}

func wizardAnswerAlreadyFilledV0(answers []WizardAnswerV0, questionRef string) bool {
	for _, answer := range answers {
		if answer.QuestionRef == questionRef {
			return true
		}
	}
	return false
}

func wizardBotReplyV0(locale string, turn WizardTurnResultV0, answers []WizardAnswerV0, say string, refs []string) WizardBotReplyV0 {
	return WizardBotReplyV0{
		SchemaVersion: WizardBotReplySchemaV0,
		Say:           trimV0(say),
		TurnResult:    turn,
		FilledAnswers: answers,
		GroundingRefs: compactStringsV0(refs),
	}
}

func wizardBotTryLLMAssistV0(
	ctx context.Context,
	session WebNuevaAppIntakeSessionV0,
	turn WizardBotTurnV0,
	current WizardTurnResultV0,
	corpus []wizardRAGCorpusEntryV0,
	assistant WizardBotLLMAssistPortV0,
) (WebNuevaAppIntakeSessionV0, WizardBotReplyV0, bool) {
	request := WizardBotLLMRequestV0{
		SessionRef:        session.SessionRef,
		UserText:          trimV0(turn.UserText),
		Locale:            trimV0(turn.Locale),
		TurnResult:        current,
		GroundingSnippets: wizardBotGroundingSnippetsV0(turn.Locale, current, corpus),
		AllowedAnswers:    wizardBotAllowedAnswersV0(current),
	}
	result, err := assistant.AssistWizardBotTurnV0(ctx, request)
	if err != nil || wizardBotLLMDegradedV0(result.Status) {
		say := wizardBotSayForTurnV0(turn.Locale, current, corpus, wizardBotLLMDegradedMessageV0(turn.Locale, result.Status))
		reply := wizardBotReplyV0(turn.Locale, current, nil, say, wizardGroundingRefsForTurnV0(current))
		reply.LLMStatus = wizardBotLLMStatusOrDefaultV0(result.Status, "provider_failed")
		return session, reply, true
	}
	answers := wizardBotValidLLMAnswersV0(result.FilledAnswers, current)
	refs := wizardBotValidLLMGroundingRefsV0(result.GroundingRefs, request.GroundingSnippets)
	if len(refs) == 0 {
		refs = wizardGroundingRefsForTurnV0(current)
	}
	if len(answers) > 0 {
		var applied WizardTurnResultV0
		session, applied = ApplyWebNuevaAppWizardAnswersV0(session, answers)
		say := trimV0(result.Say)
		if say == "" || !wizardBotLLMSayGroundedV0(say, refs) {
			say = wizardBotSayForTurnV0(turn.Locale, applied, NewWebNuevaAppWizardRAGCorpusV0(applied), "He registrado tu respuesta.")
		}
		reply := wizardBotReplyV0(turn.Locale, applied, answers, say, appendUniqueStringsV0(refs, wizardGroundingRefsForAnswersV0(applied, answers)...))
		reply.LLMStatus = "ok"
		reply.CostNotice = wizardBotCostNoticeV0(turn.Locale)
		return session, reply, true
	}
	if say := trimV0(result.Say); say != "" && wizardBotLLMSayGroundedV0(say, refs) {
		reply := wizardBotReplyV0(turn.Locale, current, nil, say, refs)
		reply.LLMStatus = "ok"
		reply.CostNotice = wizardBotCostNoticeV0(turn.Locale)
		return session, reply, true
	}
	return session, WizardBotReplyV0{}, false
}

func wizardBotGroundingSnippetsV0(locale string, turn WizardTurnResultV0, corpus []wizardRAGCorpusEntryV0) []WizardBotGroundingSnippetV0 {
	refs := wizardGroundingRefsForTurnV0(turn)
	out := make([]WizardBotGroundingSnippetV0, 0, len(refs))
	for _, ref := range refs {
		for _, entry := range corpus {
			if entry.Ref != ref {
				continue
			}
			out = append(out, WizardBotGroundingSnippetV0{Ref: entry.Ref, Kind: entry.Kind, Text: wizardBestEntryTextV0(locale, entry)})
			break
		}
	}
	return out
}

func wizardBotAllowedAnswersV0(turn WizardTurnResultV0) []WizardBotAllowedAnswerV0 {
	var out []WizardBotAllowedAnswerV0
	for _, question := range turn.Questions {
		for _, option := range question.Options {
			out = append(out, WizardBotAllowedAnswerV0{QuestionRef: question.QuestionRef, UserChoice: option.Value})
		}
	}
	return out
}

func wizardBotValidLLMAnswersV0(values []WizardAnswerV0, turn WizardTurnResultV0) []WizardAnswerV0 {
	allowed := map[string]bool{}
	for _, allowedAnswer := range wizardBotAllowedAnswersV0(turn) {
		allowed[allowedAnswer.QuestionRef+"\x00"+allowedAnswer.UserChoice] = true
	}
	var out []WizardAnswerV0
	for _, answer := range values {
		answer.QuestionRef = trimV0(answer.QuestionRef)
		answer.UserChoice = trimV0(answer.UserChoice)
		if allowed[answer.QuestionRef+"\x00"+answer.UserChoice] && !wizardAnswerAlreadyFilledV0(out, answer.QuestionRef) {
			out = append(out, answer)
		}
	}
	return out
}

func wizardBotValidLLMGroundingRefsV0(refs []string, snippets []WizardBotGroundingSnippetV0) []string {
	allowed := map[string]bool{}
	for _, snippet := range snippets {
		allowed[snippet.Ref] = true
	}
	var out []string
	for _, ref := range refs {
		ref = trimV0(ref)
		if allowed[ref] {
			out = append(out, ref)
		}
	}
	return compactStringsV0(out)
}

func wizardBotLLMSayGroundedV0(say string, refs []string) bool {
	return trimV0(say) != "" && len(refs) > 0
}

func wizardBotLLMDegradedV0(status string) bool {
	status = trimV0(status)
	return status == "budget_exhausted" || status == "provider_failed" || status == "disabled"
}

func wizardBotLLMStatusOrDefaultV0(status, fallback string) string {
	if status = trimV0(status); status != "" {
		return status
	}
	return fallback
}

func wizardBotLLMDegradedMessageV0(locale, status string) string {
	if trimV0(status) == "budget_exhausted" {
		return wizardBotLocaleTextV0(locale, "Sigo contigo en modo basico por presupuesto.", "I am continuing in basic mode because the budget is exhausted.")
	}
	return wizardBotLocaleTextV0(locale, "Sigo contigo en modo basico porque el proveedor no esta disponible.", "I am continuing in basic mode because the provider is unavailable.")
}

func wizardBotCostNoticeV0(locale string) string {
	return wizardBotLocaleTextV0(locale, "El chat con LLM consume presupuesto del bot.", "LLM chat consumes the bot budget.")
}

func wizardBotSayForMatchesV0(locale string, matches []wizardLexicalMatchV0, turn WizardTurnResultV0, corpus []wizardRAGCorpusEntryV0) string {
	var builder strings.Builder
	builder.WriteString(wizardBotLocaleTextV0(locale, "Esto es lo que cubre el catalogo:", "This is what the catalog covers:"))
	for _, match := range matches {
		builder.WriteString(" ")
		builder.WriteString(wizardBestEntryTextV0(locale, match.Entry))
	}
	builder.WriteString(" ")
	builder.WriteString(wizardBotTurnPromptV0(locale, turn, corpus))
	return builder.String()
}

func wizardBotUnknownSayV0(locale, query string, turn WizardTurnResultV0, corpus []wizardRAGCorpusEntryV0) string {
	return wizardBotLocaleTextV0(locale, "No lo se con el catalogo actual: ", "I do not know from the current catalog: ") + query + ". " + wizardBotTurnPromptV0(locale, turn, corpus)
}

func wizardBotSayForTurnV0(locale string, turn WizardTurnResultV0, corpus []wizardRAGCorpusEntryV0, prefix string) string {
	parts := compactStringsV0([]string{prefix, wizardBotTurnPromptV0(locale, turn, corpus)})
	return strings.Join(parts, " ")
}

func wizardBotTurnPromptV0(locale string, turn WizardTurnResultV0, corpus []wizardRAGCorpusEntryV0) string {
	if turn.LaunchReady {
		return wizardBotLocaleTextV0(locale, "La especificacion esta completa. Confirma explicitamente antes de lanzar.", "The specification is complete. Confirm explicitly before launch.")
	}
	if len(turn.Questions) == 0 {
		return wizardBotLocaleTextV0(locale, "No hay una pregunta abierta ahora.", "There is no open question now.")
	}
	question := turn.Questions[0]
	var builder strings.Builder
	builder.WriteString(wizardBotLocaleTextV0(locale, "Siguiente pregunta: ", "Next question: "))
	builder.WriteString(wizardEntryTextByRefV0(locale, corpus, "wizard-corpus:question:"+question.QuestionRef, question.PromptKey))
	builder.WriteString(" ")
	for index, option := range question.Options {
		builder.WriteString(strconv.Itoa(index + 1))
		builder.WriteString(". ")
		builder.WriteString(wizardEntryTextByRefV0(locale, corpus, "wizard-corpus:option:"+question.QuestionRef+":"+option.Value, option.Value))
		if option.Recommended {
			builder.WriteString(wizardBotLocaleTextV0(locale, " (recomendada)", " (recommended)"))
		}
		builder.WriteString(" ")
	}
	builder.WriteString(wizardBotLocaleTextV0(locale, "Responde con el numero, con tus palabras, o preguntame que significa cualquier cosa.", "Answer with the number, with your words, or ask what anything means."))
	return builder.String()
}

func wizardEntryTextByRefV0(locale string, corpus []wizardRAGCorpusEntryV0, ref string, fallback string) string {
	for _, entry := range corpus {
		if entry.Ref == ref {
			return wizardBestEntryTextV0(locale, entry)
		}
	}
	return fallback
}

func wizardBestEntryTextV0(locale string, entry wizardRAGCorpusEntryV0) string {
	prefer := func(text string) bool {
		if strings.HasPrefix(text, "nueva_app.") || text == entry.Option || text == entry.Key {
			return false
		}
		if strings.Contains(text, "_") && !strings.Contains(text, " ") {
			return false
		}
		return true
	}
	for _, text := range entry.Texts {
		if !prefer(text) {
			continue
		}
		if wizardLocaleLooksEnglishV0(locale) == strings.Contains(strings.ToLower(text), " asks ") {
			return text
		}
	}
	for _, text := range entry.Texts {
		if prefer(text) {
			return text
		}
	}
	return entry.Key
}

func wizardLocaleLooksEnglishV0(locale string) bool {
	return strings.HasPrefix(strings.ToLower(strings.ReplaceAll(locale, "_", "-")), "en")
}

func wizardBotLocaleTextV0(locale, es, en string) string {
	if wizardLocaleLooksEnglishV0(locale) {
		return en
	}
	return es
}

func wizardMatchRefsV0(matches []wizardLexicalMatchV0) []string {
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, match.Entry.Ref)
	}
	return compactStringsV0(out)
}

func wizardGroundingRefsForTurnV0(turn WizardTurnResultV0) []string {
	var refs []string
	for _, question := range turn.Questions {
		refs = append(refs, "wizard-corpus:question:"+question.QuestionRef)
		for _, option := range question.Options {
			refs = append(refs, "wizard-corpus:option:"+question.QuestionRef+":"+option.Value)
		}
	}
	for _, defaultValue := range turn.EngineeringDefaults {
		refs = append(refs, "wizard-corpus:default:"+defaultValue.Value)
	}
	return compactStringsV0(refs)
}

func wizardGroundingRefsForAnswersV0(turn WizardTurnResultV0, answers []WizardAnswerV0) []string {
	refs := wizardGroundingRefsForTurnV0(turn)
	for _, answer := range answers {
		refs = append(refs, "wizard-answer:"+answer.QuestionRef+":"+answer.UserChoice)
	}
	return compactStringsV0(refs)
}
