/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"orquesta/i18n"
)

const (
	languagePolicyDefaultKey          = "language.policy.default_language"
	languagePolicyDocsMultilangKey    = "language.policy.documentation_multilang"
	languagePolicyAppsMultilangKey    = "language.policy.apps_multilang"
	languagePolicyDocsDefaultKey      = "language.policy.documentation_default_language"
	languagePolicyAppsDefaultKey      = "language.policy.apps_default_language"
	languagePolicyAllowedLanguagesKey = "language.policy.allowed_languages"
	languagePolicyNotesKey            = "language.policy.notes"
	languagePolicyUpdatedByKey        = "language.policy.updated_by"
	languagePolicyUpdatedAtKey        = "language.policy.updated_at"
	languageMatrixPrefix              = "language.matrix."
	languageMatrixProjectPrefix       = "language.matrix.project."
	languageMatrixTaskPrefix          = "language.matrix.task."
	languageMatrixContextDocs         = "docs"
	languageMatrixContextApps         = "apps"
	languageMatrixContextAll          = "all"
)

type LanguagePolicy struct {
	DefaultLanguage          string    `json:"default_language"`
	DocumentationMultilang   bool      `json:"documentation_multilang"`
	AppsMultilang            bool      `json:"apps_multilang"`
	DocumentationDefaultLang string    `json:"documentation_default_language"`
	AppsDefaultLang          string    `json:"apps_default_language"`
	AllowedLanguages         []string  `json:"allowed_languages"`
	Notes                    string    `json:"notes"`
	UpdatedBy                string    `json:"updated_by"`
	UpdatedAt                time.Time `json:"updated_at"`
}

type LanguageMatrixEntry struct {
	Scope     string    `json:"scope"`
	Selector  string    `json:"selector"`
	Context   string    `json:"context"`
	Language  string    `json:"language"`
	Reason    string    `json:"reason"`
	UpdatedBy string    `json:"updated_by"`
	UpdatedAt time.Time `json:"updated_at"`
	ConfigKey string    `json:"-"`
}

type LanguageResolution struct {
	Proyecto string
	TareaID  *int64
	Contexto string
	Idioma   string
	Origen   string
	Entrada  *LanguageMatrixEntry
	Politica *LanguagePolicy
}

func defaultLanguagePolicy() *LanguagePolicy {
	return &LanguagePolicy{
		DefaultLanguage:          "es",
		DocumentationMultilang:   true,
		AppsMultilang:            true,
		DocumentationDefaultLang: "es",
		AppsDefaultLang:          "es",
		AllowedLanguages:         i18n.DefaultProjectSkeletonLanguages(),
	}
}

func normalizeLanguageCode(lang string) string {
	lang = strings.TrimSpace(strings.ToLower(lang))
	lang = strings.ReplaceAll(lang, "_", "-")
	if lang == "" {
		return ""
	}
	if idx := strings.IndexByte(lang, '-'); idx >= 0 {
		lang = lang[:idx]
	}
	return lang
}

func normalizeContext(ctx string) string {
	ctx = strings.TrimSpace(strings.ToLower(ctx))
	switch ctx {
	case "", "all", "todo", "ambos":
		return languageMatrixContextAll
	case "docs", "documentacion":
		return languageMatrixContextDocs
	case "apps", "app", "aplicaciones":
		return languageMatrixContextApps
	default:
		return ctx
	}
}

func normalizeSelectorKind(kind string) string {
	kind = strings.TrimSpace(strings.ToLower(kind))
	switch kind {
	case "project", "proyecto":
		return "project"
	case "task", "tarea":
		return "task"
	default:
		return ""
	}
}

func configBoolOrDefault(key string, fallback bool) bool {
	v, err := ConfigGet(key)
	if err != nil {
		return fallback
	}
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "1", "true", "yes", "si", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func configStringOrDefault(key, fallback string) string {
	v, err := ConfigGet(key)
	if err != nil {
		return fallback
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}

func configCSVOrDefault(key string, fallback []string) []string {
	v, err := ConfigGet(key)
	if err != nil {
		return append([]string{}, fallback...)
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return append([]string{}, fallback...)
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if p := normalizeLanguageCode(part); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return append([]string{}, fallback...)
	}
	return out
}

func ensureLanguagePolicyDefaults() {
	_, _ = DB.Exec(`INSERT OR IGNORE INTO config (clave, valor) VALUES (?, ?)`, languagePolicyDefaultKey, "es")
	_, _ = DB.Exec(`INSERT OR IGNORE INTO config (clave, valor) VALUES (?, ?)`, languagePolicyDocsMultilangKey, "1")
	_, _ = DB.Exec(`INSERT OR IGNORE INTO config (clave, valor) VALUES (?, ?)`, languagePolicyAppsMultilangKey, "1")
	_, _ = DB.Exec(`INSERT OR IGNORE INTO config (clave, valor) VALUES (?, ?)`, languagePolicyDocsDefaultKey, "es")
	_, _ = DB.Exec(`INSERT OR IGNORE INTO config (clave, valor) VALUES (?, ?)`, languagePolicyAppsDefaultKey, "es")
	_, _ = DB.Exec(`INSERT OR IGNORE INTO config (clave, valor) VALUES (?, ?)`, languagePolicyAllowedLanguagesKey, strings.Join(i18n.DefaultProjectSkeletonLanguages(), ","))
}

func GetLanguagePolicy() (*LanguagePolicy, error) {
	p := defaultLanguagePolicy()
	p.DefaultLanguage = normalizeLanguageCode(configStringOrDefault(languagePolicyDefaultKey, p.DefaultLanguage))
	if p.DefaultLanguage == "" {
		p.DefaultLanguage = "es"
	}
	p.DocumentationMultilang = configBoolOrDefault(languagePolicyDocsMultilangKey, p.DocumentationMultilang)
	p.AppsMultilang = configBoolOrDefault(languagePolicyAppsMultilangKey, p.AppsMultilang)
	p.DocumentationDefaultLang = normalizeLanguageCode(configStringOrDefault(languagePolicyDocsDefaultKey, p.DocumentationDefaultLang))
	if p.DocumentationDefaultLang == "" {
		p.DocumentationDefaultLang = p.DefaultLanguage
	}
	p.AppsDefaultLang = normalizeLanguageCode(configStringOrDefault(languagePolicyAppsDefaultKey, p.AppsDefaultLang))
	if p.AppsDefaultLang == "" {
		p.AppsDefaultLang = p.DefaultLanguage
	}
	p.AllowedLanguages = configCSVOrDefault(languagePolicyAllowedLanguagesKey, p.AllowedLanguages)
	p.Notes = configStringOrDefault(languagePolicyNotesKey, "")
	p.UpdatedBy = configStringOrDefault(languagePolicyUpdatedByKey, "")
	if raw, err := ConfigGet(languagePolicyUpdatedAtKey); err == nil {
		if ts, parseErr := time.Parse(time.RFC3339Nano, strings.TrimSpace(raw)); parseErr == nil {
			p.UpdatedAt = ts
		}
	}
	return p, nil
}

func SetLanguagePolicy(p *LanguagePolicy, updatedBy string) error {
	if p == nil {
		return fmt.Errorf("politica de lenguaje nula")
	}
	if normalizeLanguageCode(p.DefaultLanguage) == "" {
		return fmt.Errorf("default_language es obligatorio")
	}
	if normalizeLanguageCode(p.DocumentationDefaultLang) == "" {
		p.DocumentationDefaultLang = p.DefaultLanguage
	}
	if normalizeLanguageCode(p.AppsDefaultLang) == "" {
		p.AppsDefaultLang = p.DefaultLanguage
	}

	p.DefaultLanguage = normalizeLanguageCode(p.DefaultLanguage)
	p.DocumentationDefaultLang = normalizeLanguageCode(p.DocumentationDefaultLang)
	p.AppsDefaultLang = normalizeLanguageCode(p.AppsDefaultLang)
	p.AllowedLanguages = normalizeLanguageList(p.AllowedLanguages, p.DefaultLanguage, p.DocumentationDefaultLang, p.AppsDefaultLang)
	p.UpdatedBy = strings.TrimSpace(updatedBy)
	p.UpdatedAt = time.Now().UTC()

	vals := map[string]string{
		languagePolicyDefaultKey:          p.DefaultLanguage,
		languagePolicyDocsMultilangKey:    boolString(p.DocumentationMultilang),
		languagePolicyAppsMultilangKey:    boolString(p.AppsMultilang),
		languagePolicyDocsDefaultKey:      p.DocumentationDefaultLang,
		languagePolicyAppsDefaultKey:      p.AppsDefaultLang,
		languagePolicyAllowedLanguagesKey: strings.Join(p.AllowedLanguages, ","),
		languagePolicyNotesKey:            strings.TrimSpace(p.Notes),
		languagePolicyUpdatedByKey:        p.UpdatedBy,
		languagePolicyUpdatedAtKey:        p.UpdatedAt.Format(time.RFC3339Nano),
	}
	for k, v := range vals {
		if err := ConfigSet(k, v); err != nil {
			return err
		}
	}
	Audit("orquesta", "actualizar_politica_lenguaje", "config", 0, p.DefaultLanguage)
	return nil
}

func normalizeLanguageList(values []string, required ...string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values)+len(required))
	add := func(v string) {
		v = normalizeLanguageCode(v)
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	for _, v := range values {
		add(v)
	}
	for _, v := range required {
		add(v)
	}
	sort.Strings(out)
	return out
}

func boolString(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

func matrixConfigKey(kind, selector, contexto string) (string, error) {
	kind = normalizeSelectorKind(kind)
	contexto = normalizeContext(contexto)
	selector = strings.TrimSpace(selector)
	if kind == "" || selector == "" {
		return "", fmt.Errorf("tipo y selector son obligatorios")
	}
	if contexto == "" {
		contexto = languageMatrixContextAll
	}
	if kind == "task" {
		return fmt.Sprintf("%s%s.%s", languageMatrixTaskPrefix, selector, contexto), nil
	}
	return fmt.Sprintf("%s%s.%s", languageMatrixProjectPrefix, selector, contexto), nil
}

func SetLanguageMatrixEntry(kind, selector, contexto, language, reason, updatedBy string) (*LanguageMatrixEntry, error) {
	kind = normalizeSelectorKind(kind)
	contexto = normalizeContext(contexto)
	selector = strings.TrimSpace(selector)
	language = normalizeLanguageCode(language)
	reason = strings.TrimSpace(reason)
	updatedBy = strings.TrimSpace(updatedBy)
	if kind == "" || selector == "" || language == "" {
		return nil, fmt.Errorf("tipo, selector e idioma son obligatorios")
	}
	policy, err := GetLanguagePolicy()
	if err != nil {
		return nil, err
	}
	if !languageAllowed(language, policy.AllowedLanguages) {
		return nil, fmt.Errorf("idioma %s no permitido por la politica global", language)
	}
	key, err := matrixConfigKey(kind, selector, contexto)
	if err != nil {
		return nil, err
	}
	entry := &LanguageMatrixEntry{
		Scope:     kind,
		Selector:  selector,
		Context:   contexto,
		Language:  language,
		Reason:    reason,
		UpdatedBy: updatedBy,
		UpdatedAt: time.Now().UTC(),
		ConfigKey: key,
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return nil, err
	}
	if err := ConfigSet(key, string(raw)); err != nil {
		return nil, err
	}
	Audit("orquesta", "actualizar_matriz_lenguaje", "config", 0, key+"="+language)
	return entry, nil
}

func DeleteLanguageMatrixEntry(kind, selector, contexto string) error {
	key, err := matrixConfigKey(kind, selector, contexto)
	if err != nil {
		return err
	}
	res, err := DB.Exec(`DELETE FROM config WHERE clave = ?`, key)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("entrada no encontrada")
	}
	Audit("orquesta", "borrar_matriz_lenguaje", "config", 0, key)
	return nil
}

func ListLanguageMatrixEntries() ([]*LanguageMatrixEntry, error) {
	rows, err := DB.Query(`SELECT clave, valor FROM config WHERE clave LIKE ? ORDER BY clave`, languageMatrixPrefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*LanguageMatrixEntry
	for rows.Next() {
		var key, raw string
		if err := rows.Scan(&key, &raw); err != nil {
			return nil, err
		}
		var entry LanguageMatrixEntry
		if err := json.Unmarshal([]byte(raw), &entry); err != nil {
			return nil, fmt.Errorf("parseando %s: %w", key, err)
		}
		entry.ConfigKey = key
		if entry.UpdatedAt.IsZero() {
			entry.UpdatedAt = time.Time{}
		}
		if entry.Scope == "" {
			if strings.HasPrefix(key, languageMatrixTaskPrefix) {
				entry.Scope = "task"
			} else {
				entry.Scope = "project"
			}
		}
		out = append(out, &entry)
	}
	return out, rows.Err()
}

func GetLanguageMatrixEntry(kind, selector, contexto string) (*LanguageMatrixEntry, error) {
	key, err := matrixConfigKey(kind, selector, contexto)
	if err != nil {
		return nil, err
	}
	var raw string
	if err := DB.QueryRow(`SELECT valor FROM config WHERE clave = ?`, key).Scan(&raw); err != nil {
		return nil, err
	}
	var entry LanguageMatrixEntry
	if err := json.Unmarshal([]byte(raw), &entry); err != nil {
		return nil, fmt.Errorf("parseando %s: %w", key, err)
	}
	entry.ConfigKey = key
	return &entry, nil
}

func languageAllowed(lang string, allowed []string) bool {
	lang = normalizeLanguageCode(lang)
	for _, candidate := range allowed {
		if normalizeLanguageCode(candidate) == lang {
			return true
		}
	}
	return false
}

func ResolveLanguage(project string, taskID *int64, contexto string) (*LanguageResolution, error) {
	contexto = normalizeContext(contexto)
	if contexto == "" {
		contexto = languageMatrixContextAll
	}
	policy, err := GetLanguagePolicy()
	if err != nil {
		return nil, err
	}
	res := &LanguageResolution{
		Proyecto: strings.TrimSpace(project),
		TareaID:  taskID,
		Contexto: contexto,
		Politica: policy,
	}

	if taskID != nil && *taskID > 0 {
		if entry, err := findLanguageEntry("task", fmt.Sprintf("%d", *taskID), contexto); err == nil {
			res.Idioma = entry.Language
			res.Origen = entry.ConfigKey
			res.Entrada = entry
			return res, nil
		}
	}
	if project = strings.TrimSpace(project); project != "" {
		if entry, err := findLanguageEntry("project", project, contexto); err == nil {
			res.Idioma = entry.Language
			res.Origen = entry.ConfigKey
			res.Entrada = entry
			return res, nil
		}
	}

	switch contexto {
	case languageMatrixContextDocs:
		res.Idioma = policy.DocumentationDefaultLang
		res.Origen = languagePolicyDocsDefaultKey
	case languageMatrixContextApps:
		res.Idioma = policy.AppsDefaultLang
		res.Origen = languagePolicyAppsDefaultKey
	default:
		res.Idioma = policy.DefaultLanguage
		res.Origen = languagePolicyDefaultKey
	}
	return res, nil
}

func findLanguageEntry(kind, selector, contexto string) (*LanguageMatrixEntry, error) {
	if entry, err := GetLanguageMatrixEntry(kind, selector, contexto); err == nil {
		return entry, nil
	}
	if contexto != languageMatrixContextAll {
		if entry, err := GetLanguageMatrixEntry(kind, selector, languageMatrixContextAll); err == nil {
			return entry, nil
		}
	}
	return nil, fmt.Errorf("no encontrada")
}
