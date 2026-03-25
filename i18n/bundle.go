/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const DefaultLang = "es"

type Bundle struct {
	dir          string
	defaultLang  string
	fallbackLang string

	mu    sync.RWMutex
	dicts map[string]map[string]string
}

func NewBundle(dir string, defaultLang string) *Bundle {
	defaultLang = NormalizeLang(defaultLang)
	if defaultLang == "" {
		defaultLang = DefaultLang
	}
	return &Bundle{
		dir:          dir,
		defaultLang:  defaultLang,
		fallbackLang: defaultLang,
		dicts:        map[string]map[string]string{},
	}
}

func ResolveDir() string {
	if v := strings.TrimSpace(os.Getenv("ORQUESTA_I18N_DIR")); v != "" {
		return v
	}
	return "i18n"
}

func NormalizeLang(lang string) string {
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

func (b *Bundle) Reload() error {
	entries, err := os.ReadDir(b.dir)
	if err != nil {
		return fmt.Errorf("leyendo diccionarios i18n: %w", err)
	}

	dicts := map[string]map[string]string{}
	cfg, err := loadBundleConfig(b.dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := strings.TrimSpace(entry.Name())
		if entry.IsDir() {
			lang := NormalizeLang(name)
			if lang == "" {
				continue
			}
			dict, err := loadDomainDictionaries(filepath.Join(b.dir, name))
			if err != nil {
				return fmt.Errorf("leyendo diccionarios de %s: %w", name, err)
			}
			if len(dict) == 0 {
				continue
			}
			mergeDict(dicts, lang, dict)
			continue
		}
		if filepath.Ext(name) != ".json" || strings.EqualFold(name, projectSkeletonConfigName) {
			continue
		}
		lang := NormalizeLang(strings.TrimSuffix(name, filepath.Ext(name)))
		if lang == "" {
			continue
		}
		dict, err := loadDictionaryFile(filepath.Join(b.dir, name))
		if err != nil {
			return fmt.Errorf("leyendo diccionario %s: %w", name, err)
		}
		mergeDict(dicts, lang, dict)
	}

	if len(dicts) == 0 {
		return fmt.Errorf("no hay diccionarios i18n en %s", b.dir)
	}

	defaultLang, fallbackLang, err := resolveBundleLanguages(dicts, b.dir, b.defaultLang, cfg)
	if err != nil {
		return err
	}

	b.mu.Lock()
	b.dicts = dicts
	b.defaultLang = defaultLang
	b.fallbackLang = fallbackLang
	b.mu.Unlock()
	return nil
}

func (b *Bundle) SetDefaultLang(lang string) {
	lang = NormalizeLang(lang)
	if lang == "" {
		return
	}
	b.mu.Lock()
	b.defaultLang = lang
	if b.fallbackLang == "" {
		b.fallbackLang = lang
	}
	b.mu.Unlock()
}

func (b *Bundle) HasLang(lang string) bool {
	lang = NormalizeLang(lang)
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, ok := b.dicts[lang]
	return ok
}

func (b *Bundle) ResolveLang(lang string) string {
	lang = NormalizeLang(lang)
	b.mu.RLock()
	defer b.mu.RUnlock()
	if _, ok := b.dicts[lang]; ok {
		return lang
	}
	return b.defaultLang
}

func (b *Bundle) Languages() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	langs := make([]string, 0, len(b.dicts))
	for lang := range b.dicts {
		langs = append(langs, lang)
	}
	sort.Strings(langs)
	return langs
}

func (b *Bundle) T(lang string, key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	lang = b.ResolveLang(lang)

	b.mu.RLock()
	defer b.mu.RUnlock()
	if v := lookupKey(b.dicts[lang], key); v != "" {
		return v
	}
	if v := lookupKey(b.dicts[b.fallbackLang], key); v != "" {
		return v
	}
	if v := lookupKey(b.dicts[b.defaultLang], key); v != "" {
		return v
	}
	return key
}

func lookupKey(dict map[string]string, key string) string {
	if dict == nil {
		return ""
	}
	return strings.TrimSpace(dict[key])
}

func loadDomainDictionaries(dir string) (map[string]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	dict := map[string]string{}
	for _, name := range names {
		partial, err := loadDictionaryFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		for key, value := range partial {
			dict[key] = value
		}
	}
	return dict, nil
}

func loadDictionaryFile(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var dict map[string]string
	if err := json.Unmarshal(raw, &dict); err != nil {
		return nil, err
	}
	return dict, nil
}

func mergeDict(dicts map[string]map[string]string, lang string, dict map[string]string) {
	if len(dict) == 0 {
		return
	}
	current := dicts[lang]
	if current == nil {
		current = map[string]string{}
		dicts[lang] = current
	}
	for key, value := range dict {
		current[key] = value
	}
}

func loadBundleConfig(dir string) (*ProjectSkeletonConfig, error) {
	path := filepath.Join(dir, projectSkeletonConfigName)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("leyendo %s: %w", path, err)
	}
	var cfg ProjectSkeletonConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parseando %s: %w", path, err)
	}
	return &cfg, nil
}

func resolveBundleLanguages(dicts map[string]map[string]string, dir string, currentDefault string, cfg *ProjectSkeletonConfig) (string, string, error) {
	defaultLang := NormalizeLang(currentDefault)
	if defaultLang == "" {
		defaultLang = DefaultLang
	}
	fallbackLang := defaultLang
	if cfg != nil {
		if candidate := NormalizeLang(cfg.DefaultLanguage); candidate != "" {
			if _, ok := dicts[defaultLang]; !ok {
				defaultLang = candidate
			}
		}
		if candidate := NormalizeLang(cfg.FallbackLanguage); candidate != "" {
			fallbackLang = candidate
		}
	}
	if _, ok := dicts[defaultLang]; !ok {
		return "", "", fmt.Errorf("falta el idioma por defecto %s en %s", defaultLang, dir)
	}
	if _, ok := dicts[fallbackLang]; !ok {
		fallbackLang = defaultLang
	}
	return defaultLang, fallbackLang, nil
}
