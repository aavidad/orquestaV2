package orquestacore

import "strings"

const (
	FunctionContractErrWriteSetVacioV0                = "write_set_vacio"
	FunctionContractErrArchivoObjetivoFueraWriteSetV0 = "archivo_objetivo_fuera_de_write_set"
	FunctionContractErrDependenciaProhibidaV0         = "dependencia_prohibida"
	FunctionContractErrFormatoEntregaNoSoportadoV0    = "formato_entrega_no_soportado"
	FunctionContractErrCambioFueraWriteSetV0          = "cambio_fuera_de_write_set"
)

type FunctionContractValidationResultV0 struct {
	Contract FunctionContractV0        `json:"contract"`
	Errores  []FunctionContractErrorV0 `json:"errores,omitempty"`
}

type FunctionContractDeliveryValidationRequestV0 struct {
	Contract     FunctionContractV0 `json:"contract"`
	ChangedFiles []string           `json:"changed_files"`
}

func ValidateFunctionContractV0(contract FunctionContractV0) FunctionContractValidationResultV0 {
	normalized := normalizeFunctionContractV0(contract)
	result := FunctionContractValidationResultV0{Contract: normalized}

	if normalized.Titulo == "" ||
		normalized.Objetivo == "" ||
		normalized.ArchivoObjetivo == "" ||
		normalized.SimboloObjetivo == "" ||
		len(normalized.TestsObligatorios) == 0 ||
		len(normalized.CriterioCierre) == 0 {
		result.add(FunctionContractQueryErrIncompletoV0, "", nil)
	}
	if len(normalized.WriteSet) == 0 {
		result.add(FunctionContractErrWriteSetVacioV0, "write_set", nil)
	}
	if normalized.ArchivoObjetivo != "" &&
		len(normalized.WriteSet) > 0 &&
		!pathAllowedByWriteSetV0(normalized.ArchivoObjetivo, normalized.WriteSet) {
		result.add(FunctionContractErrArchivoObjetivoFueraWriteSetV0, "archivo_objetivo", []string{normalized.ArchivoObjetivo})
	}
	if !functionContractFormatoEntregaSoportadoV0(normalized.FormatoEntrega) {
		result.add(FunctionContractErrFormatoEntregaNoSoportadoV0, "formato_entrega", []string{normalized.FormatoEntrega})
	}
	if duplicated := firstIntersectionV0(normalized.DependenciasPermitidas, normalized.DependenciasProhibidas); duplicated != "" {
		result.add(FunctionContractErrDependenciaProhibidaV0, "dependencias_prohibidas", []string{duplicated})
	}

	if len(result.Errores) == 0 && normalized.Estado == "" {
		result.Contract.Estado = FunctionContractEstadoActivaV0
	}
	return result
}

func ValidateFunctionContractDeliveryV0(request FunctionContractDeliveryValidationRequestV0) FunctionContractValidationResultV0 {
	result := ValidateFunctionContractV0(request.Contract)
	writeSet := result.Contract.WriteSet
	for _, changed := range compactStringsCoreV0(request.ChangedFiles) {
		if !pathAllowedByWriteSetV0(changed, writeSet) {
			result.add(FunctionContractErrCambioFueraWriteSetV0, "changed_files", []string{changed})
		}
	}
	return result
}

func (result *FunctionContractValidationResultV0) add(code string, field string, evidence []string) {
	result.Errores = append(result.Errores, FunctionContractErrorV0{
		Code:      code,
		Message:   code,
		Field:     field,
		Retryable: false,
		Evidence:  append([]string(nil), evidence...),
	})
}

func normalizeFunctionContractV0(contract FunctionContractV0) FunctionContractV0 {
	contract.Titulo = strings.TrimSpace(contract.Titulo)
	contract.Objetivo = strings.TrimSpace(contract.Objetivo)
	contract.ArchivoObjetivo = normalizeContractPathV0(contract.ArchivoObjetivo)
	contract.SimboloObjetivo = strings.TrimSpace(contract.SimboloObjetivo)
	contract.Descripcion = strings.TrimSpace(contract.Descripcion)
	contract.WriteSet = compactContractPathsV0(contract.WriteSet)
	contract.DependenciasPermitidas = compactStringsCoreV0(contract.DependenciasPermitidas)
	contract.DependenciasProhibidas = compactStringsCoreV0(contract.DependenciasProhibidas)
	contract.Precondiciones = compactStringsCoreV0(contract.Precondiciones)
	contract.Postcondiciones = compactStringsCoreV0(contract.Postcondiciones)
	contract.TestsObligatorios = compactStringsCoreV0(contract.TestsObligatorios)
	contract.FormatoEntrega = strings.TrimSpace(contract.FormatoEntrega)
	contract.CriterioCierre = compactStringsCoreV0(contract.CriterioCierre)
	contract.PruebasContrato = compactStringsCoreV0(contract.PruebasContrato)
	contract.Estado = strings.TrimSpace(contract.Estado)
	contract.OrigenEvidencia = strings.TrimSpace(contract.OrigenEvidencia)
	return contract
}

func functionContractFormatoEntregaSoportadoV0(format string) bool {
	switch format {
	case FunctionContractFormatoPatchEvidenciaV0,
		FunctionContractFormatoPatchUnificadoV0,
		FunctionContractFormatoFicherosEvidenciaV0:
		return true
	default:
		return false
	}
}

func pathAllowedByWriteSetV0(path string, writeSet []string) bool {
	normalizedPath := normalizeContractPathV0(path)
	if normalizedPath == "" || strings.HasPrefix(normalizedPath, "../") || strings.HasPrefix(normalizedPath, "/") {
		return false
	}
	for _, scope := range writeSet {
		normalizedScope := normalizeContractPathV0(scope)
		if normalizedScope == "" || strings.HasPrefix(normalizedScope, "../") || strings.HasPrefix(normalizedScope, "/") {
			continue
		}
		if normalizedPath == normalizedScope || strings.HasPrefix(normalizedPath, strings.TrimSuffix(normalizedScope, "/")+"/") {
			return true
		}
	}
	return false
}

func normalizeContractPathV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "/")
	for strings.Contains(value, "//") {
		value = strings.ReplaceAll(value, "//", "/")
	}
	return strings.TrimSuffix(value, "/")
}

func compactContractPathsV0(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := normalizeContractPathV0(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func compactStringsCoreV0(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func firstIntersectionV0(left []string, right []string) string {
	seen := map[string]struct{}{}
	for _, value := range left {
		seen[value] = struct{}{}
	}
	for _, value := range right {
		if _, ok := seen[value]; ok {
			return value
		}
	}
	return ""
}
