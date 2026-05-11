# Tareas: orquesta-app-planner

## APP-PLAN-001

Objetivo: dividir una solicitud Go API + web en microtareas pequenas.

Estado: hecho.

Validacion: `go test -count=1 ./modulos/orquesta-app-planner`.

## APP-PLAN-002

Objetivo: exponer solo la ola lista para scheduler, con claims de toda la ola.

Estado: hecho.

Validacion: `TestAppPlanCandidateProviderV0ProduceComandosSchedulerValidos`.

## APP-PLAN-003

Objetivo: resolver contrato, evidencias y contexto pequeno para el lanzador de
agentes sin elegir proveedor, modelo, HOME ni credenciales reales.

Estado: en curso.

Validacion: `TestAppPlanResolversConstruyenRuntimeLaunchValidoV0`.

## APP-PLAN-004

Objetivo: conectar el planificador a `AppSpecV0` validada por factory.

Estado: hecho.

Validacion: `TestBuildGoAPIWebMicrotaskPlanFromAppSpecV0UsaContratoFactoryV0`.
