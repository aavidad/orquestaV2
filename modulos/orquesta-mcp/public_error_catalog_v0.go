package orquestamcp

import (
	"strings"

	i18ndocs "orquesta/modulos/orquesta-i18n-docs"
)

const (
	MCPPublicErrMethodNotAllowedV0   = i18ndocs.PublicErrorMethodNotAllowedV0
	MCPPublicErrPathUnsupportedV0    = i18ndocs.PublicErrorPathUnsupportedV0
	MCPPublicErrBodyInvalidV0        = i18ndocs.PublicErrorBodyInvalidV0
	MCPPublicErrBodyTooLargeV0       = i18ndocs.PublicErrorBodyTooLargeV0
	MCPPublicErrBodyTrailingDataV0   = i18ndocs.PublicErrorBodyTrailingDataV0
	MCPPublicErrContentTypeInvalidV0 = i18ndocs.PublicErrorContentTypeInvalidV0
	MCPPublicErrExecutorV0           = i18ndocs.PublicErrorExecutorV0
	MCPPublicErrTransportUnboundV0   = i18ndocs.PublicErrorMCPTransportUnboundV0
	MCPPublicErrTransportInputV0     = i18ndocs.PublicErrorMCPTransportInputV0
	MCPPublicErrTransportPortV0      = i18ndocs.PublicErrorMCPTransportPortV0
	MCPPublicErrTransportSchemaV0    = i18ndocs.PublicErrorMCPTransportSchemaV0
	MCPPublicErrOutputTooLargeV0     = i18ndocs.PublicErrorMCPOutputTooLargeV0
	MCPPublicErrResourceBlockedV0    = i18ndocs.PublicErrorMCPResourceBlockedV0
	MCPPublicErrToolBlockedV0        = i18ndocs.PublicErrorMCPToolBlockedV0
)

func MCPPublicErrorDescriptorV0(code string) (i18ndocs.PublicErrorDescriptorV0, bool) {
	return i18ndocs.PublicErrorDescriptorByCodeV0(code)
}

func MCPPublicErrorCodeKnownV0(code string) bool {
	return i18ndocs.PublicErrorCodeKnownV0(code)
}

func normalizeMCPPublicErrorCodeV0(code string, fallback string) string {
	return i18ndocs.NormalizePublicErrorCodeV0(code, fallback)
}

func publicMCPErrorCodeFromErrorV0(err error, fallback string) string {
	if err == nil {
		return normalizeMCPPublicErrorCodeV0("", fallback)
	}
	code := strings.TrimSpace(err.Error())
	return normalizeMCPPublicErrorCodeV0(code, fallback)
}
