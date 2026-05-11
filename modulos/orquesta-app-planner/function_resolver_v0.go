package orquestaappplanner

import orquestaruntime "orquesta/modulos/orquesta-runtime"

type AppPlanFunctionContractResolverV0 struct {
	Plan AppMicrotaskPlanV0
}

var _ orquestaruntime.FunctionContractResolverV0 = AppPlanFunctionContractResolverV0{}

func (resolver AppPlanFunctionContractResolverV0) ResolveFunctionContractV0(
	taskRef string,
) (*orquestaruntime.RuntimeFunctionContractV0, error) {
	if err := validateAppMicrotaskPlanV0(resolver.Plan); err != nil {
		return nil, err
	}
	unit, ok := findAppPlanUnitByTaskRefV0(resolver.Plan, taskRef)
	if !ok {
		return nil, AppPlannerIssueV0{Field: "task_ref"}
	}
	contract := RuntimeFunctionContractForUnitV0(unit)
	return &contract, nil
}
