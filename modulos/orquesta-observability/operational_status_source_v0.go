package orquestaobservability

type OperationalStatusQuerySourceV0 interface {
	QueryOperationalStatusV0(OperationalStatusQueryV0) (DiagnosticoCompactoV0, error)
}

func FilterDiagnosticoCompactoForQueryV0(
	diagnostic DiagnosticoCompactoV0,
	query OperationalStatusQueryV0,
) (DiagnosticoCompactoV0, error) {
	if err := ValidateOperationalStatusQueryV0(query); err != nil {
		return DiagnosticoCompactoV0{}, err
	}
	if err := ValidateDiagnosticoCompactoV0(diagnostic); err != nil {
		return DiagnosticoCompactoV0{}, err
	}
	filtered := filterDiagnosticoCompactoForQueryV0(diagnostic, query)
	if err := ValidateDiagnosticoCompactoV0(filtered); err != nil {
		return DiagnosticoCompactoV0{}, err
	}
	return filtered, nil
}
