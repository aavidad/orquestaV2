package orquestai18ndocs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	AppI18nDocsPlanContractVersionV0 = "v0"
	I18nBundleStructureVersionV0     = "i18n-bundle/v0"
	DocsBundleStructureVersionV0     = "docs-bundle/v0"

	ErrDefaultLocaleFueraDeCatalogo = "default_locale_fuera_de_catalogo"
	ErrLocaleSinCatalogo            = "locale_sin_catalogo"
	ErrLocaleNoCoincide             = "locale_no_coincide"
	ErrClaveRequeridaFaltante       = "clave_requerida_faltante"
	ErrTipoDocumentoFaltante        = "tipo_documento_faltante"
	ErrFallbackLocaleInvalido       = "fallback_locale_invalido"
)

type AppI18nDocsPlanV0 struct {
	ContractVersion     string                    `json:"contract_version"`
	AppIDHint           string                    `json:"app_id_hint"`
	SourceContract      string                    `json:"source_contract"`
	UIDefaultLocale     string                    `json:"ui_default_locale"`
	DocsDefaultLocale   string                    `json:"docs_default_locale"`
	WebBundle           *I18nBundleV0             `json:"web_bundle,omitempty"`
	FactoryBundle       *I18nBundleV0             `json:"factory_bundle,omitempty"`
	DocsBundle          *DocsBundleV0             `json:"docs_bundle,omitempty"`
	SkeletonLoaderShape I18nSkeletonLoaderShapeV0 `json:"skeleton_loader_shape"`
	Warnings            []I18nDocsWarningV0       `json:"warnings"`
}

type I18nBundleV0 struct {
	StructureVersion string                   `json:"structure_version"`
	Scope            string                   `json:"scope"`
	DefaultLocale    string                   `json:"default_locale"`
	Locales          []string                 `json:"locales"`
	Catalogs         map[string]I18nCatalogV0 `json:"catalogs"`
	RequiredKeys     []string                 `json:"required_keys"`
	Namespace        string                   `json:"namespace"`
}

type I18nCatalogV0 struct {
	Locale   string                 `json:"locale"`
	Messages map[string]string      `json:"messages"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type DocsBundleV0 struct {
	StructureVersion string           `json:"structure_version"`
	DefaultLocale    string           `json:"default_locale"`
	Locales          []string         `json:"locales"`
	Docs             []GeneratedDocV0 `json:"docs"`
	RequiredDocTypes []string         `json:"required_doc_types"`
	TemplateSet      string           `json:"template_set"`
}

type GeneratedDocV0 struct {
	DocID      string         `json:"doc_id"`
	DocType    string         `json:"doc_type"`
	Locale     string         `json:"locale"`
	TitleKey   string         `json:"title_key"`
	ContentKey string         `json:"content_key"`
	Format     string         `json:"format"`
	Sections   []DocSectionV0 `json:"sections"`
}

type DocSectionV0 struct {
	SectionID  string `json:"section_id"`
	TitleKey   string `json:"title_key"`
	ContentKey string `json:"content_key"`
	Order      int    `json:"order"`
	Required   bool   `json:"required"`
}

type I18nSkeletonLoaderShapeV0 struct {
	StructureVersion      string   `json:"structure_version"`
	LoaderContract        string   `json:"loader_contract"`
	CatalogPathPattern    string   `json:"catalog_path_pattern"`
	DefaultLocaleStrategy string   `json:"default_locale_strategy"`
	FallbackLocale        string   `json:"fallback_locale"`
	RequiredNamespaces    []string `json:"required_namespaces"`
	RequiredKeysHash      string   `json:"required_keys_hash"`
}

type I18nDocsWarningV0 struct {
	Code       string `json:"code"`
	MessageKey string `json:"message_key"`
}

type I18nDocsValidationIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"path,omitempty"`
	Message string `json:"message"`
}

func DecodeAppI18nDocsPlanV0(data []byte) (AppI18nDocsPlanV0, []I18nDocsValidationIssueV0) {
	var plan AppI18nDocsPlanV0
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&plan); err != nil {
		return plan, []I18nDocsValidationIssueV0{{
			Code:    "app_i18n_docs_plan_invalido",
			Message: fmt.Sprintf("plan JSON invalido: %v", err),
		}}
	}
	return plan, ValidateAppI18nDocsPlanV0(plan)
}

func ValidateAppI18nDocsPlanV0(plan AppI18nDocsPlanV0) []I18nDocsValidationIssueV0 {
	var issues []I18nDocsValidationIssueV0
	issues = append(issues, validateI18nBundleV0("web_bundle", plan.WebBundle)...)
	issues = append(issues, validateI18nBundleV0("factory_bundle", plan.FactoryBundle)...)
	issues = append(issues, validateDocsBundleV0("docs_bundle", plan.DocsBundle)...)
	issues = append(issues, validateFallbackLocaleV0(plan)...)
	return issues
}

func validateI18nBundleV0(path string, bundle *I18nBundleV0) []I18nDocsValidationIssueV0 {
	if bundle == nil {
		return nil
	}
	var issues []I18nDocsValidationIssueV0
	localeSet := stringSetV0(bundle.Locales)
	if !localeSet[bundle.DefaultLocale] {
		issues = append(issues, i18nDocsIssueV0(ErrDefaultLocaleFueraDeCatalogo, path+".default_locale", "default_locale debe estar incluido en locales"))
	}
	for _, locale := range bundle.Locales {
		catalog, ok := bundle.Catalogs[locale]
		if !ok {
			issues = append(issues, i18nDocsIssueV0(ErrLocaleSinCatalogo, path+".catalogs."+locale, "cada locale declarado debe tener catalogo"))
			continue
		}
		if catalog.Locale != "" && catalog.Locale != locale {
			issues = append(issues, i18nDocsIssueV0(ErrLocaleNoCoincide, path+".catalogs."+locale+".locale", "locale del catalogo debe coincidir con la clave"))
		}
		issues = append(issues, validateRequiredKeysV0(path+".catalogs."+locale, bundle.RequiredKeys, catalog.Messages)...)
	}
	for locale := range bundle.Catalogs {
		if !localeSet[locale] {
			issues = append(issues, i18nDocsIssueV0(ErrLocaleSinCatalogo, path+".catalogs."+locale, "catalogs no debe contener locales no declarados"))
		}
	}
	return issues
}

func validateRequiredKeysV0(path string, requiredKeys []string, messages map[string]string) []I18nDocsValidationIssueV0 {
	var issues []I18nDocsValidationIssueV0
	for _, key := range requiredKeys {
		if _, ok := messages[key]; !ok {
			issues = append(issues, i18nDocsIssueV0(ErrClaveRequeridaFaltante, path+".messages."+key, "catalogo incompleto para required_keys"))
		}
	}
	return issues
}

func validateDocsBundleV0(path string, bundle *DocsBundleV0) []I18nDocsValidationIssueV0 {
	if bundle == nil {
		return nil
	}
	var issues []I18nDocsValidationIssueV0
	localeSet := stringSetV0(bundle.Locales)
	if !localeSet[bundle.DefaultLocale] {
		issues = append(issues, i18nDocsIssueV0(ErrDefaultLocaleFueraDeCatalogo, path+".default_locale", "default_locale debe estar incluido en locales"))
	}
	covered := make(map[string]map[string]bool, len(bundle.Locales))
	for _, locale := range bundle.Locales {
		covered[locale] = make(map[string]bool, len(bundle.RequiredDocTypes))
	}
	for _, doc := range bundle.Docs {
		if _, ok := covered[doc.Locale]; ok {
			covered[doc.Locale][doc.DocType] = true
		}
	}
	for _, locale := range bundle.Locales {
		for _, docType := range bundle.RequiredDocTypes {
			if !covered[locale][docType] {
				issues = append(issues, i18nDocsIssueV0(ErrTipoDocumentoFaltante, path+".docs", "cada locale debe cubrir todos los required_doc_types"))
			}
		}
	}
	return issues
}

func validateFallbackLocaleV0(plan AppI18nDocsPlanV0) []I18nDocsValidationIssueV0 {
	fallback := strings.TrimSpace(plan.SkeletonLoaderShape.FallbackLocale)
	if fallback == "" {
		return nil
	}
	if plan.WebBundle != nil && stringSetV0(plan.WebBundle.Locales)[fallback] {
		return nil
	}
	if plan.FactoryBundle != nil && stringSetV0(plan.FactoryBundle.Locales)[fallback] {
		return nil
	}
	if plan.DocsBundle != nil && stringSetV0(plan.DocsBundle.Locales)[fallback] {
		return nil
	}
	return []I18nDocsValidationIssueV0{i18nDocsIssueV0(ErrFallbackLocaleInvalido, "skeleton_loader_shape.fallback_locale", "fallback_locale debe pertenecer a al menos un bundle aplicable")}
}

func stringSetV0(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func i18nDocsIssueV0(code, field, message string) I18nDocsValidationIssueV0 {
	return I18nDocsValidationIssueV0{
		Code:    code,
		Field:   field,
		Message: message,
	}
}
