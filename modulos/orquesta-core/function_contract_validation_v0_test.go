package orquestacore

import "testing"

func TestValidateFunctionContractV0AceptaMinimoActivo(t *testing.T) {
	contract := validFunctionContractV0()

	result := ValidateFunctionContractV0(contract)

	if len(result.Errores) != 0 {
		t.Fatalf("errores=%+v", result.Errores)
	}
	if result.Contract.Estado != FunctionContractEstadoActivaV0 {
		t.Fatalf("estado=%q", result.Contract.Estado)
	}
	if len(result.Contract.WriteSet) != 1 || result.Contract.WriteSet[0] != "modulos/orquesta-core" {
		t.Fatalf("write_set normalizado=%+v", result.Contract.WriteSet)
	}
}

func TestValidateFunctionContractV0RechazaWriteSetVacio(t *testing.T) {
	contract := validFunctionContractV0()
	contract.WriteSet = nil

	result := ValidateFunctionContractV0(contract)

	assertFunctionContractErrorV0(t, result, FunctionContractErrWriteSetVacioV0)
}

func TestValidateFunctionContractV0RechazaArchivoFueraDeWriteSet(t *testing.T) {
	contract := validFunctionContractV0()
	contract.ArchivoObjetivo = "modulos/orquesta-runtime/launcher.go"

	result := ValidateFunctionContractV0(contract)

	assertFunctionContractErrorV0(t, result, FunctionContractErrArchivoObjetivoFueraWriteSetV0)
}

func TestValidateFunctionContractV0RechazaDependenciaProhibida(t *testing.T) {
	contract := validFunctionContractV0()
	contract.DependenciasPermitidas = []string{"context", "net/http"}
	contract.DependenciasProhibidas = []string{"net/http"}

	result := ValidateFunctionContractV0(contract)

	assertFunctionContractErrorV0(t, result, FunctionContractErrDependenciaProhibidaV0)
}

func TestValidateFunctionContractV0RechazaFormatoEntregaDesconocido(t *testing.T) {
	contract := validFunctionContractV0()
	contract.FormatoEntrega = "captura"

	result := ValidateFunctionContractV0(contract)

	assertFunctionContractErrorV0(t, result, FunctionContractErrFormatoEntregaNoSoportadoV0)
}

func TestValidateFunctionContractDeliveryV0RespetaWriteSet(t *testing.T) {
	result := ValidateFunctionContractDeliveryV0(FunctionContractDeliveryValidationRequestV0{
		Contract:     validFunctionContractV0(),
		ChangedFiles: []string{"modulos/orquesta-core/function_contract_validation_v0.go"},
	})

	if len(result.Errores) != 0 {
		t.Fatalf("errores=%+v", result.Errores)
	}
}

func TestValidateFunctionContractDeliveryV0RechazaCambioFueraWriteSet(t *testing.T) {
	result := ValidateFunctionContractDeliveryV0(FunctionContractDeliveryValidationRequestV0{
		Contract:     validFunctionContractV0(),
		ChangedFiles: []string{"modulos/orquesta-runtime/launcher.go"},
	})

	assertFunctionContractErrorV0(t, result, FunctionContractErrCambioFueraWriteSetV0)
}

func validFunctionContractV0() FunctionContractV0 {
	return FunctionContractV0{
		Titulo:            "Validar FunctionContract",
		Objetivo:          "Mantener contratos ejecutables de core.",
		ArchivoObjetivo:   "modulos/orquesta-core/function_contract_validation_v0.go",
		SimboloObjetivo:   "ValidateFunctionContractV0",
		WriteSet:          []string{" modulos/orquesta-core ", "modulos/orquesta-core"},
		TestsObligatorios: []string{"go test -count=1 ./modulos/orquesta-core"},
		FormatoEntrega:    FunctionContractFormatoPatchEvidenciaV0,
		CriterioCierre:    []string{"tests verdes"},
		Version:           0,
	}
}

func assertFunctionContractErrorV0(t *testing.T, result FunctionContractValidationResultV0, want string) {
	t.Helper()
	for _, err := range result.Errores {
		if err.Code == want {
			return
		}
	}
	t.Fatalf("error %q no encontrado en %+v", want, result.Errores)
}
