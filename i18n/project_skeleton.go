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
)

const (
	projectSkeletonVersion     = 1
	projectSkeletonDirName     = "i18n"
	projectSkeletonConfigName  = "config.json"
	projectSkeletonReadmeName  = "README.md"
	projectSkeletonDomainMode  = "file_per_domain"
	projectSkeletonPathPattern = "i18n/<lang>/<domain>.json"
)

func DefaultProjectSkeletonLanguages() []string {
	return []string{"es", "en", "de", "fr", "it", "zh", "gl", "eu", "ca", "val"}
}

type ProjectSkeletonConfig struct {
	Version          int      `json:"version"`
	DefaultLanguage  string   `json:"default_language"`
	FallbackLanguage string   `json:"fallback_language"`
	Languages        []string `json:"languages"`
	Domains          []string `json:"domains"`
	DomainMode       string   `json:"domain_mode"`
	PathPattern      string   `json:"path_pattern"`
}

type ProjectSkeletonSpec struct {
	RootDir          string
	DefaultLanguage  string
	FallbackLanguage string
	Languages        []string
	Domains          []string
}

func DefaultProjectSkeletonDomains() []string {
	return []string{"common", "navigation", "actions", "validation", "errors"}
}

func NormalizeProjectSkeletonSpec(spec ProjectSkeletonSpec) (ProjectSkeletonSpec, error) {
	spec.RootDir = strings.TrimSpace(spec.RootDir)
	if spec.RootDir == "" {
		return ProjectSkeletonSpec{}, fmt.Errorf("root dir obligatorio")
	}

	spec.DefaultLanguage = NormalizeLang(spec.DefaultLanguage)
	if spec.DefaultLanguage == "" {
		spec.DefaultLanguage = DefaultLang
	}

	spec.FallbackLanguage = NormalizeLang(spec.FallbackLanguage)
	if spec.FallbackLanguage == "" {
		spec.FallbackLanguage = spec.DefaultLanguage
	}

	spec.Languages = normalizeUniqueLanguages(spec.Languages)
	if len(spec.Languages) == 0 {
		spec.Languages = DefaultProjectSkeletonLanguages()
	}
	spec.Languages = normalizeUniqueLanguages(append(spec.Languages, spec.DefaultLanguage, spec.FallbackLanguage))

	spec.Domains = normalizeDomains(spec.Domains)
	if len(spec.Domains) == 0 {
		spec.Domains = DefaultProjectSkeletonDomains()
	}

	return spec, nil
}

func BuildProjectSkeletonConfig(spec ProjectSkeletonSpec) (*ProjectSkeletonConfig, error) {
	spec, err := NormalizeProjectSkeletonSpec(spec)
	if err != nil {
		return nil, err
	}
	return &ProjectSkeletonConfig{
		Version:          projectSkeletonVersion,
		DefaultLanguage:  spec.DefaultLanguage,
		FallbackLanguage: spec.FallbackLanguage,
		Languages:        append([]string{}, spec.Languages...),
		Domains:          append([]string{}, spec.Domains...),
		DomainMode:       projectSkeletonDomainMode,
		PathPattern:      projectSkeletonPathPattern,
	}, nil
}

func MaterializeProjectSkeleton(spec ProjectSkeletonSpec) (*ProjectSkeletonConfig, error) {
	spec, err := NormalizeProjectSkeletonSpec(spec)
	if err != nil {
		return nil, err
	}

	i18nDir := filepath.Join(spec.RootDir, projectSkeletonDirName)
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		return nil, fmt.Errorf("crear directorio i18n: %w", err)
	}

	cfg, err := BuildProjectSkeletonConfig(spec)
	if err != nil {
		return nil, err
	}
	if err := writeJSONFile(filepath.Join(i18nDir, projectSkeletonConfigName), cfg); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(i18nDir, projectSkeletonReadmeName), []byte(projectSkeletonReadme(spec)), 0o644); err != nil {
		return nil, fmt.Errorf("escribir README i18n: %w", err)
	}

	for _, lang := range spec.Languages {
		langDir := filepath.Join(i18nDir, lang)
		if err := os.MkdirAll(langDir, 0o755); err != nil {
			return nil, fmt.Errorf("crear directorio idioma %s: %w", lang, err)
		}
		for _, domain := range spec.Domains {
			seed := seedDomainDictionary(lang, spec.DefaultLanguage, domain)
			if err := writeJSONFile(filepath.Join(langDir, domain+".json"), seed); err != nil {
				return nil, err
			}
		}
	}

	return cfg, nil
}

func writeJSONFile(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("serializar %s: %w", path, err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("escribir %s: %w", path, err)
	}
	return nil
}

func normalizeUniqueLanguages(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		lang := NormalizeLang(value)
		if lang == "" {
			continue
		}
		if _, ok := seen[lang]; ok {
			continue
		}
		seen[lang] = struct{}{}
		out = append(out, lang)
	}
	sort.Strings(out)
	return out
}

func normalizeDomains(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		domain := strings.TrimSpace(strings.ToLower(value))
		if domain == "" {
			continue
		}
		domain = strings.ReplaceAll(domain, " ", "_")
		if _, ok := seen[domain]; ok {
			continue
		}
		seen[domain] = struct{}{}
		out = append(out, domain)
	}
	sort.Strings(out)
	return out
}

func projectSkeletonReadme(spec ProjectSkeletonSpec) string {
	return strings.TrimSpace(fmt.Sprintf(`
# i18n

Contrato base de internacionalizacion del proyecto.

Reglas:
- idioma por defecto: %s
- fallback inicial: %s
- estructura por idioma y por dominio
- un fichero JSON por dominio dentro de cada idioma
- claves estables y reutilizables; no mezclar textos de modulos distintos sin dominio

Patron esperado:
- i18n/config.json
- i18n/<idioma>/<dominio>.json

Dominios semilla:
- %s
`, spec.DefaultLanguage, spec.FallbackLanguage, strings.Join(spec.Domains, ", "))) + "\n"
}

func seedDomainDictionary(lang, defaultLang, domain string) map[string]string {
	lang = NormalizeLang(lang)
	defaultLang = NormalizeLang(defaultLang)
	base := seedDictionaryByDomain(defaultLang, domain)
	if specific := seedDictionaryByDomain(lang, domain); len(specific) > 0 {
		return specific
	}
	return base
}

func seedDictionaryByDomain(lang, domain string) map[string]string {
	switch domain {
	case "common":
		return mapByLang(lang,
			map[string]string{
				"app.title":          "Aplicacion base",
				"app.subtitle":       "Proyecto generado por Orquesta",
				"status.ready":       "Listo",
				"status.loading":     "Cargando",
				"status.empty":       "Sin datos",
				"status.unavailable": "No disponible",
			},
			map[string]string{
				"app.title":          "Base application",
				"app.subtitle":       "Project generated by Orquesta",
				"status.ready":       "Ready",
				"status.loading":     "Loading",
				"status.empty":       "No data",
				"status.unavailable": "Unavailable",
			},
		)
	case "navigation":
		return mapByLang(lang,
			map[string]string{
				"nav.home":      "Inicio",
				"nav.dashboard": "Panel",
				"nav.settings":  "Configuracion",
				"nav.help":      "Ayuda",
				"nav.back":      "Volver",
			},
			map[string]string{
				"nav.home":      "Home",
				"nav.dashboard": "Dashboard",
				"nav.settings":  "Settings",
				"nav.help":      "Help",
				"nav.back":      "Back",
			},
		)
	case "actions":
		return mapByLang(lang,
			map[string]string{
				"action.accept": "Aceptar",
				"action.cancel": "Cancelar",
				"action.save":   "Guardar",
				"action.close":  "Cerrar",
				"action.retry":  "Reintentar",
			},
			map[string]string{
				"action.accept": "Accept",
				"action.cancel": "Cancel",
				"action.save":   "Save",
				"action.close":  "Close",
				"action.retry":  "Retry",
			},
		)
	case "validation":
		return mapByLang(lang,
			map[string]string{
				"validation.required":      "Campo obligatorio",
				"validation.invalid_email": "Correo no valido",
				"validation.min_length":    "Longitud minima no alcanzada",
				"validation.max_length":    "Longitud maxima superada",
			},
			map[string]string{
				"validation.required":      "Required field",
				"validation.invalid_email": "Invalid email",
				"validation.min_length":    "Minimum length not reached",
				"validation.max_length":    "Maximum length exceeded",
			},
		)
	case "errors":
		return mapByLang(lang,
			map[string]string{
				"error.generic":      "Ha ocurrido un error",
				"error.network":      "No se pudo completar la peticion",
				"error.unauthorized": "No autorizado",
				"error.forbidden":    "Acceso denegado",
				"error.not_found":    "Recurso no encontrado",
			},
			map[string]string{
				"error.generic":      "An error occurred",
				"error.network":      "The request could not be completed",
				"error.unauthorized": "Unauthorized",
				"error.forbidden":    "Access denied",
				"error.not_found":    "Resource not found",
			},
		)
	default:
		return map[string]string{
			domain + ".title":       fallbackCopy(lang, "", "Titulo", "Title"),
			domain + ".description": fallbackCopy(lang, "", "Descripcion base", "Base description"),
		}
	}
}

func mapByLang(lang string, es map[string]string, en map[string]string) map[string]string {
	if lang == "en" {
		return en
	}
	return es
}

func fallbackCopy(lang, defaultLang, esValue, enValue string) string {
	if NormalizeLang(lang) == "en" {
		return enValue
	}
	if NormalizeLang(defaultLang) == "en" {
		return enValue
	}
	return esValue
}
