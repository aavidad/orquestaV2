package main

import "testing"

func TestServerPublicErrorCatalogV0CubreMCPJSONRPCV0(t *testing.T) {
	for _, code := range []string{
		serverPublicErrMethodNotAllowedV0,
		serverPublicErrBodyTooLargeV0,
		serverPublicErrBodyTrailingDataV0,
		serverPublicErrContentTypeInvalidV0,
		serverPublicErrAcceptInvalidV0,
		serverPublicErrMCPParamsTooLargeV0,
	} {
		entry, ok := serverPublicErrorDescriptorV0(code)
		if !ok || entry.I18nKey == "" || entry.Severity == "" {
			t.Fatalf("codigo server fuera de catalogo: %s %+v", code, entry)
		}
	}
}
