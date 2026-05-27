package orquestacli

import i18ndocs "orquesta/modulos/orquesta-i18n-docs"

const (
	cliPublicErrOptionInvalidV0   = i18ndocs.PublicErrorCLIOptionInvalidV0
	cliPublicErrInputTooLargeV0   = i18ndocs.PublicErrorCLIInputTooLargeV0
	cliPublicErrConfigInvalidV0   = i18ndocs.PublicErrorCLIConfigInvalidV0
	cliPublicErrContractMissingV0 = i18ndocs.PublicErrorCLIContractMissingV0
	cliPublicErrResponseInvalidV0 = i18ndocs.PublicErrorResponseInvalidV0
	cliPublicErrTransportV0       = i18ndocs.PublicErrorTransportV0
	cliPublicErrNotSerializableV0 = i18ndocs.PublicErrorCLINotSerializableV0
)

func CliPublicErrorDescriptorV0(code string) (i18ndocs.PublicErrorDescriptorV0, bool) {
	return i18ndocs.PublicErrorDescriptorByCodeV0(code)
}

func CliPublicErrorCodeKnownV0(code string) bool {
	return i18ndocs.PublicErrorCodeKnownV0(code)
}
