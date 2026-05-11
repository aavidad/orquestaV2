package orquestai18ndocs

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

const (
	appSpecSourceContractV0 = "AppSpecV0"
	i18nLoaderContractV0    = "i18n-loader/v0"
	defaultLocaleV0         = "es-ES"
)

var (
	webRequiredKeysV0 = []string{
		"app.title",
		"app.action.primary",
		"docs.user.title",
		"docs.user.content",
		"docs.user.overview.title",
		"docs.user.overview.content",
		"docs.developer.title",
		"docs.developer.content",
		"docs.developer.architecture.title",
		"docs.developer.architecture.content",
		"docs.systems.title",
		"docs.systems.content",
		"docs.systems.operations.title",
		"docs.systems.operations.content",
	}
	factoryRequiredKeysV0 = []string{
		"factory.plan.created",
		"factory.warning.review",
	}
	requiredDocTypesV0 = []string{
		"user_manual",
		"developer_manual",
		"systems_manual",
	}
)

type PlanSeedV0 struct {
	AppIDHint          string
	AppTitle           string
	PrimaryActionLabel string
	UILocales          []string
	DocsLocales        []string
	UIDefaultLocale    string
	DocsDefaultLocale  string
}

func BuildAppI18nDocsPlanV0(seed PlanSeedV0) AppI18nDocsPlanV0 {
	normalized := normalizePlanSeedV0(seed)
	webBundle := buildI18nBundleV0(normalized, "web", "app", normalized.UIDefaultLocale, normalized.UILocales, webRequiredKeysV0, buildWebMessagesV0)
	factoryBundle := buildI18nBundleV0(normalized, "factory", "factory", normalized.UIDefaultLocale, normalized.UILocales, factoryRequiredKeysV0, buildFactoryMessagesV0)

	return AppI18nDocsPlanV0{
		ContractVersion:     AppI18nDocsPlanContractVersionV0,
		AppIDHint:           normalized.AppIDHint,
		SourceContract:      appSpecSourceContractV0,
		UIDefaultLocale:     normalized.UIDefaultLocale,
		DocsDefaultLocale:   normalized.DocsDefaultLocale,
		WebBundle:           &webBundle,
		FactoryBundle:       &factoryBundle,
		DocsBundle:          buildDocsBundleV0(normalized),
		SkeletonLoaderShape: buildSkeletonLoaderShapeV0(normalized.UIDefaultLocale, webBundle, factoryBundle),
		Warnings:            []I18nDocsWarningV0{},
	}
}

func normalizePlanSeedV0(seed PlanSeedV0) PlanSeedV0 {
	seed.AppTitle = defaultStringV0(seed.AppTitle, "Aplicacion")
	seed.AppIDHint = stableSlugV0(defaultStringV0(seed.AppIDHint, seed.AppTitle))
	seed.PrimaryActionLabel = defaultStringV0(seed.PrimaryActionLabel, "Crear elemento")
	seed.UIDefaultLocale = defaultStringV0(seed.UIDefaultLocale, defaultLocaleV0)
	seed.DocsDefaultLocale = defaultStringV0(seed.DocsDefaultLocale, seed.UIDefaultLocale)
	seed.UILocales = normalizeLocalesV0(seed.UILocales, seed.UIDefaultLocale)
	seed.DocsLocales = normalizeLocalesV0(seed.DocsLocales, seed.DocsDefaultLocale)
	return seed
}

func buildI18nBundleV0(seed PlanSeedV0, scope, namespace, defaultLocale string, locales []string, requiredKeys []string, messages func(string, PlanSeedV0) map[string]string) I18nBundleV0 {
	catalogs := make(map[string]I18nCatalogV0, len(locales))
	for _, locale := range locales {
		catalogs[locale] = I18nCatalogV0{
			Locale:   locale,
			Messages: messages(locale, seed),
			Metadata: map[string]interface{}{"source": "plan-seed-v0"},
		}
	}
	return I18nBundleV0{
		StructureVersion: I18nBundleStructureVersionV0,
		Scope:            scope,
		DefaultLocale:    defaultLocale,
		Locales:          append([]string(nil), locales...),
		Catalogs:         catalogs,
		RequiredKeys:     append([]string(nil), requiredKeys...),
		Namespace:        namespace,
	}
}

func buildWebMessagesV0(locale string, seed PlanSeedV0) map[string]string {
	if isEnglishLocaleV0(locale) {
		return map[string]string{
			"app.title":                           seed.AppTitle,
			"app.action.primary":                  seed.PrimaryActionLabel,
			"docs.user.title":                     "User manual",
			"docs.user.content":                   "Initial user manual content.",
			"docs.user.overview.title":            "Overview",
			"docs.user.overview.content":          "Initial functional overview.",
			"docs.developer.title":                "Developer manual",
			"docs.developer.content":              "Initial developer manual content.",
			"docs.developer.architecture.title":   "Architecture",
			"docs.developer.architecture.content": "Initial technical overview.",
			"docs.systems.title":                  "Systems manual",
			"docs.systems.content":                "Initial systems manual content.",
			"docs.systems.operations.title":       "Operations",
			"docs.systems.operations.content":     "Initial operations overview.",
		}
	}
	return map[string]string{
		"app.title":                           seed.AppTitle,
		"app.action.primary":                  seed.PrimaryActionLabel,
		"docs.user.title":                     "Manual de usuario",
		"docs.user.content":                   "Contenido inicial del manual de usuario.",
		"docs.user.overview.title":            "Resumen",
		"docs.user.overview.content":          "Resumen funcional inicial.",
		"docs.developer.title":                "Manual de desarrollador",
		"docs.developer.content":              "Contenido inicial del manual de desarrollador.",
		"docs.developer.architecture.title":   "Arquitectura",
		"docs.developer.architecture.content": "Resumen tecnico inicial.",
		"docs.systems.title":                  "Manual de sistemas",
		"docs.systems.content":                "Contenido inicial del manual de sistemas.",
		"docs.systems.operations.title":       "Operaciones",
		"docs.systems.operations.content":     "Resumen operativo inicial.",
	}
}

func buildFactoryMessagesV0(locale string, _ PlanSeedV0) map[string]string {
	if isEnglishLocaleV0(locale) {
		return map[string]string{
			"factory.plan.created":   "Initial plan created",
			"factory.warning.review": "Review assumptions before generating files",
		}
	}
	return map[string]string{
		"factory.plan.created":   "Plan inicial creado",
		"factory.warning.review": "Revisar supuestos antes de generar archivos",
	}
}

func buildDocsBundleV0(seed PlanSeedV0) *DocsBundleV0 {
	docs := make([]GeneratedDocV0, 0, len(seed.DocsLocales)*len(requiredDocTypesV0))
	for _, locale := range seed.DocsLocales {
		docs = append(docs,
			generatedDocV0("user-manual", "user_manual", locale, "docs.user.title", "docs.user.content", DocSectionV0{
				SectionID:  "overview",
				TitleKey:   "docs.user.overview.title",
				ContentKey: "docs.user.overview.content",
				Order:      0,
				Required:   true,
			}),
			generatedDocV0("developer-manual", "developer_manual", locale, "docs.developer.title", "docs.developer.content", DocSectionV0{
				SectionID:  "architecture",
				TitleKey:   "docs.developer.architecture.title",
				ContentKey: "docs.developer.architecture.content",
				Order:      0,
				Required:   true,
			}),
			generatedDocV0("systems-manual", "systems_manual", locale, "docs.systems.title", "docs.systems.content", DocSectionV0{
				SectionID:  "operations",
				TitleKey:   "docs.systems.operations.title",
				ContentKey: "docs.systems.operations.content",
				Order:      0,
				Required:   true,
			}),
		)
	}
	return &DocsBundleV0{
		StructureVersion: DocsBundleStructureVersionV0,
		DefaultLocale:    seed.DocsDefaultLocale,
		Locales:          append([]string(nil), seed.DocsLocales...),
		Docs:             docs,
		RequiredDocTypes: append([]string(nil), requiredDocTypesV0...),
		TemplateSet:      "initial-docs/v0",
	}
}

func generatedDocV0(prefix, docType, locale, titleKey, contentKey string, section DocSectionV0) GeneratedDocV0 {
	return GeneratedDocV0{
		DocID:      prefix + "." + strings.ToLower(locale),
		DocType:    docType,
		Locale:     locale,
		TitleKey:   titleKey,
		ContentKey: contentKey,
		Format:     "markdown",
		Sections:   []DocSectionV0{section},
	}
}

func buildSkeletonLoaderShapeV0(fallbackLocale string, bundles ...I18nBundleV0) I18nSkeletonLoaderShapeV0 {
	namespaces := make([]string, 0, len(bundles))
	keys := make([]string, 0)
	for _, bundle := range bundles {
		namespaces = append(namespaces, bundle.Namespace)
		keys = append(keys, bundle.RequiredKeys...)
	}
	return I18nSkeletonLoaderShapeV0{
		StructureVersion:      I18nBundleStructureVersionV0,
		LoaderContract:        i18nLoaderContractV0,
		CatalogPathPattern:    "i18n/{locale}/{namespace}.json",
		DefaultLocaleStrategy: "app_spec",
		FallbackLocale:        fallbackLocale,
		RequiredNamespaces:    namespaces,
		RequiredKeysHash:      requiredKeysHashV0(keys),
	}
}

func requiredKeysHashV0(keys []string) string {
	normalized := append([]string(nil), keys...)
	sort.Strings(normalized)
	sum := sha256.Sum256([]byte(strings.Join(normalized, "\n")))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func normalizeLocalesV0(locales []string, defaultLocale string) []string {
	values := make([]string, 0, len(locales)+1)
	seen := map[string]bool{}
	for _, locale := range append([]string{defaultLocale}, locales...) {
		locale = strings.TrimSpace(locale)
		if locale == "" || seen[locale] {
			continue
		}
		seen[locale] = true
		values = append(values, locale)
	}
	return values
}

func defaultStringV0(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func stableSlugV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastHyphen := false
	for _, part := range value {
		if (part >= 'a' && part <= 'z') || (part >= '0' && part <= '9') {
			builder.WriteRune(part)
			lastHyphen = false
			continue
		}
		if !lastHyphen {
			builder.WriteByte('-')
			lastHyphen = true
		}
	}
	slug := strings.Trim(builder.String(), "-")
	if slug == "" {
		return "app"
	}
	return slug
}

func isEnglishLocaleV0(locale string) bool {
	return strings.HasPrefix(strings.ToLower(locale), "en")
}
