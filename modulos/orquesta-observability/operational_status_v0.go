package orquestaobservability

import "errors"

func (err OperationalStatusValidationErrorV0) Error() string {
	if len(err.Issues) == 0 {
		return ErrOperationalStatusQueryInvalidaV0
	}
	return err.Issues[0].Code
}

func DecodeOperationalStatusQueryV0(data []byte) (OperationalStatusQueryV0, error) {
	var query OperationalStatusQueryV0
	if err := decodeStrictJSONV0(data, &query); err != nil {
		return OperationalStatusQueryV0{}, operationalStatusValidationErrorV0(ErrOperationalStatusQueryInvalidaV0, "")
	}
	if err := ValidateOperationalStatusQueryV0(query); err != nil {
		return OperationalStatusQueryV0{}, err
	}
	return query, nil
}

func DecodeDiagnosticoCompactoV0(data []byte) (DiagnosticoCompactoV0, error) {
	var diagnostic DiagnosticoCompactoV0
	if err := decodeStrictJSONV0(data, &diagnostic); err != nil {
		return DiagnosticoCompactoV0{}, operationalStatusValidationErrorV0(ErrOperationalStatusQueryInvalidaV0, "")
	}
	if err := ValidateDiagnosticoCompactoV0(diagnostic); err != nil {
		return DiagnosticoCompactoV0{}, err
	}
	return diagnostic, nil
}

func ValidateOperationalStatusQueryV0(query OperationalStatusQueryV0) error {
	var issues []OperationalStatusValidationIssueV0
	validateOperationalStatusQueryV0(query, "", addOperationalStatusIssueFuncV0(&issues))
	if len(issues) > 0 {
		return OperationalStatusValidationErrorV0{Issues: issues}
	}
	return nil
}

func ValidateDiagnosticoCompactoV0(diagnostic DiagnosticoCompactoV0) error {
	var issues []OperationalStatusValidationIssueV0
	validateDiagnosticoCompactoV0(diagnostic, "", addOperationalStatusIssueFuncV0(&issues))
	if len(issues) > 0 {
		return OperationalStatusValidationErrorV0{Issues: issues}
	}
	return nil
}

func HasOperationalStatusIssueV0(err error, code string) bool {
	var validationErr OperationalStatusValidationErrorV0
	if !errors.As(err, &validationErr) {
		return false
	}
	for _, issue := range validationErr.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func addOperationalStatusIssueFuncV0(issues *[]OperationalStatusValidationIssueV0) func(string, string) {
	return func(code, field string) {
		*issues = append(*issues, OperationalStatusValidationIssueV0{Code: code, Field: field})
	}
}

func operationalStatusValidationErrorV0(code, field string) OperationalStatusValidationErrorV0 {
	return OperationalStatusValidationErrorV0{Issues: []OperationalStatusValidationIssueV0{{Code: code, Field: field}}}
}
