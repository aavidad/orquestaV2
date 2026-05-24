package orquestacoreworkflow

func codeFieldErrorTextV0(code string, field string) string {
	if field == "" {
		return code
	}
	return code + ": " + field
}
