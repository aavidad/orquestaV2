package i18n

import "errors"

var (
	ErrCatalogUnavailable = errors.New("i18n.catalog_unavailable")
	ErrCatalogInvalid     = errors.New("i18n.catalog_invalid")
	ErrManifestInvalid    = errors.New("i18n.manifest_invalid")
	ErrLocaleInvalid      = errors.New("i18n.locale_invalid")
	ErrKeyMissing         = errors.New("i18n.key_missing")
	ErrMessageKind        = errors.New("i18n.message_kind_invalid")
	ErrCurrencyInvalid    = errors.New("i18n.currency_invalid")
	ErrTimeZoneInvalid    = errors.New("i18n.timezone_invalid")
)
