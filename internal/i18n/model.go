package i18n

const DefaultLocale = "es"

type Resolution struct {
	Requested string
	Locale    string
	Fallback  bool
}

type Manifest struct {
	SchemaVersion   int              `json:"schema_version"`
	DefaultLocale   string           `json:"default_locale"`
	FallbackLocale  string           `json:"fallback_locale"`
	EnabledLocales  []string         `json:"enabled_locales"`
	Catalogs        []CatalogSource  `json:"catalogs"`
	Surfaces        []Surface        `json:"surfaces"`
	PublicDocuments []PublicDocument `json:"public_documents"`
}

type CatalogSource struct {
	Locale string `json:"locale"`
	Path   string `json:"path"`
}

type Surface struct {
	ID            string   `json:"id"`
	State         string   `json:"state"`
	KeySources    []string `json:"key_sources"`
	LiteralPolicy string   `json:"literal_policy"`
}

type PublicDocument struct {
	ID      string            `json:"id"`
	Locales map[string]string `json:"locales"`
}
