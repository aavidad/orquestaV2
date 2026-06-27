package main

import i18ndocs "orquesta/modulos/orquesta-i18n-docs"

const (
	serverPublicErrMethodNotAllowedV0   = i18ndocs.PublicErrorMCPMethodNotAllowedV0
	serverPublicErrBodyTooLargeV0       = i18ndocs.PublicErrorBodyTooLargeV0
	serverPublicErrBodyTrailingDataV0   = i18ndocs.PublicErrorBodyTrailingDataV0
	serverPublicErrContentTypeInvalidV0 = i18ndocs.PublicErrorContentTypeInvalidV0
	serverPublicErrAcceptInvalidV0      = i18ndocs.PublicErrorAcceptInvalidV0
	serverPublicErrMCPParamsTooLargeV0  = i18ndocs.PublicErrorMCPParamsTooLargeV0
)

func serverPublicErrorDescriptorV0(code string) (i18ndocs.PublicErrorDescriptorV0, bool) {
	return i18ndocs.PublicErrorDescriptorByCodeV0(code)
}
