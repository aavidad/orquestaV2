package orquestaappplanner

import orquestaruntime "orquesta/modulos/orquesta-runtime"

func RuntimeFunctionContractForUnitV0(unit AppWorkUnitV0) orquestaruntime.RuntimeFunctionContractV0 {
	return orquestaruntime.RuntimeFunctionContractV0{
		SourceContract:    orquestaruntime.FunctionContractSourceV0,
		ContractRef:       "contract-" + unit.TaskRef,
		ContractVersion:   orquestaruntime.FunctionContractVersionV0,
		State:             orquestaruntime.FunctionContractStateActiveV0,
		Titulo:            unit.Title,
		Objetivo:          unit.Summary,
		ArchivoObjetivo:   firstWriteSetPathV0(unit.WriteSet),
		SimboloObjetivo:   unit.TaskRef,
		WriteSet:          append([]string(nil), unit.WriteSet...),
		TestsObligatorios: append([]string(nil), unit.RequiredTests...),
		CriterioCierre:    append([]string(nil), unit.AcceptanceCriteria...),
	}
}

func firstWriteSetPathV0(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
