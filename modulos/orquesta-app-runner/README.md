# orquesta-app-runner

Adaptador pequeno de compatibilidad para preparar una vista previa de
orquestacion desde una `AppSpecV0` validada.

Responsabilidades:

- construir el `OrchestrationRunV0` inicial mediante comandos del workflow;
- publicar decision, contrato funcional y microtareas durables antes de
  programacion;
- abrir la fase de programacion para el primer corte operativo;
- crear el plan de microtareas con `orquesta-app-planner`;
- devolver un `CandidateProviderPortV0` listo para el loop del nucleo;
- ejecutar el run preparado con puertos inyectados solo en modo legacy/preview;
- bloquear con razon publica si el caller declara que el objetivo requiere
  Director V2, plan-state, waits por ola/cohorte, review/tests/cierre o
  recursion.

La entrada operativa preferente para apps nuevas es
`orquesta.apps.arrancar_director.v0`, que delega en
`orquesta-app-director-service`.

No elige runtime, proveedor, modelo, HOME, credenciales ni persistencia.

Validacion:

```bash
go test -count=1 ./modulos/orquesta-app-runner
```
