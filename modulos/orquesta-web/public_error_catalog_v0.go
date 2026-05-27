package orquestaweb

import i18ndocs "orquesta/modulos/orquesta-i18n-docs"

const (
	webPublicErrMethodUnsupportedV0 = i18ndocs.PublicErrorMethodUnsupportedV0
	webPublicErrTransportV0         = i18ndocs.PublicErrorTransportV0
	webPublicErrResponseInvalidV0   = i18ndocs.PublicErrorResponseInvalidV0
	webPublicErrFormIncompleteV0    = i18ndocs.PublicErrorWebFormIncompleteV0
	webPublicErrTransportMissingV0  = i18ndocs.PublicErrorWebTransportMissingV0
	webPublicErrRunRefRequiredV0    = i18ndocs.PublicErrorRunRefRequiredV0
)

func WebPublicErrorDescriptorV0(code string) (i18ndocs.PublicErrorDescriptorV0, bool) {
	return i18ndocs.PublicErrorDescriptorByCodeV0(code)
}

func WebPublicErrorCodeKnownV0(code string) bool {
	return i18ndocs.PublicErrorCodeKnownV0(code)
}
