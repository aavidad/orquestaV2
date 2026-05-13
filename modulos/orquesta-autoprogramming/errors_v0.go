package orquestaautoprogramming

const ErrAutoprogrammingInvalidoV0 = "autoprogramming_invalido"

type ErrorV0 struct {
	Code    string
	Field   string
	Message string
}

func (err ErrorV0) Error() string {
	return err.Code
}

func errorV0(code string, field string, message string) ErrorV0 {
	return ErrorV0{
		Code:    code,
		Field:   field,
		Message: message,
	}
}
