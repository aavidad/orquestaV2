package orquestacionnucleoapp

func errorV0(code string, field string, message string) ErrorV0 {
	return ErrorV0{
		Code:    code,
		Field:   field,
		Message: message,
	}
}
