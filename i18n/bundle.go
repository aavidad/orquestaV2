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
	dir         string
	defaultLang string

	mu    sync.RWMutex
	dicts map[string]map[string]string
}

func NewBundle(dir string, defaultLang string) *Bundle {
	defaultLang = NormalizeLang(defaultLang)
	if defaultLang == "" {
		defaultLang = DefaultLang
	}
	return &Bundle{
		dir:         dir,
		defaultLang: defaultLang,
		dicts:       map[string]map[string]string{},
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
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		lang := NormalizeLang(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
		if lang == "" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(b.dir, entry.Name()))
		if err != nil {
			return fmt.Errorf("leyendo diccionario %s: %w", entry.Name(), err)
		}
		var dict map[string]string
		if err := json.Unmarshal(raw, &dict); err != nil {
			return fmt.Errorf("parseando diccionario %s: %w", entry.Name(), err)
		}
		dicts[lang] = dict
	}

	if len(dicts) == 0 {
		return fmt.Errorf("no hay diccionarios i18n en %s", b.dir)
	}
	if _, ok := dicts[b.defaultLang]; !ok {
		return fmt.Errorf("falta el idioma por defecto %s en %s", b.defaultLang, b.dir)
	}

	b.mu.Lock()
	b.dicts = dicts
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
