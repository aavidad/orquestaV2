package orquestaappchange

func PrepareAppChangeRequestV0(request AppChangeRequestV0) AppChangeRequestV0 {
	return normalizeAppChangeRequestV0(request)
}

func ValidateAppChangeRequestV0(request AppChangeRequestV0) []AppChangeIssueV0 {
	return validateAppChangeRequestV0(normalizeAppChangeRequestV0(request))
}
