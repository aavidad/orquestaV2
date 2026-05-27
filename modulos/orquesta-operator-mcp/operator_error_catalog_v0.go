package orquestaoperatormcp

import i18ndocs "orquesta/modulos/orquesta-i18n-docs"

const (
	i18nPublicOperatorRequiredFieldV0 = i18ndocs.PublicErrorOperatorRequiredFieldV0
	i18nPublicOperatorOpaqueRefV0     = i18ndocs.PublicErrorOperatorOpaqueRefV0
	i18nPublicOperatorBudgetInvalidV0 = i18ndocs.PublicErrorOperatorBudgetInvalidV0
	i18nPublicOperatorLimitInvalidV0  = i18ndocs.PublicErrorOperatorLimitInvalidV0
	i18nPublicOperatorQuestionV0      = i18ndocs.PublicErrorOperatorQuestionV0
	i18nPublicOperatorSectionV0       = i18ndocs.PublicErrorOperatorSectionV0
	i18nPublicOperatorPortV0          = i18ndocs.PublicErrorOperatorPortV0
	i18nPublicOperatorPortErrorV0     = i18ndocs.PublicErrorOperatorPortErrorV0
	i18nPublicOperatorConnectorV0     = i18ndocs.PublicErrorOperatorConnectorV0
	i18nPublicOperatorTimeoutV0       = i18ndocs.PublicErrorOperatorTimeoutV0
	i18nPublicOperatorCancelledV0     = i18ndocs.PublicErrorOperatorCancelledV0
)

func OperatorMCPPublicErrorDescriptorV0(code string) (i18ndocs.PublicErrorDescriptorV0, bool) {
	return i18ndocs.PublicErrorDescriptorByCodeV0(code)
}

func OperatorMCPPublicErrorCodeKnownV0(code string) bool {
	return i18ndocs.PublicErrorCodeKnownV0(code)
}
