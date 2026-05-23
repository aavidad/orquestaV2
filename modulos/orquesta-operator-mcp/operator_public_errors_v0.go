package orquestaoperatormcp

import "errors"

type OperatorMCPPublicErrorV0 struct {
	Code string
}

func NewOperatorMCPPublicErrorV0(code string) OperatorMCPPublicErrorV0 {
	return OperatorMCPPublicErrorV0{Code: code}
}

func (err OperatorMCPPublicErrorV0) Error() string {
	return err.Code
}

func PublicOperatorMCPErrorCodeV0(err error) (string, bool) {
	var publicErr OperatorMCPPublicErrorV0
	if errors.As(err, &publicErr) && publicErr.Code != "" {
		return publicErr.Code, true
	}
	return "", false
}
