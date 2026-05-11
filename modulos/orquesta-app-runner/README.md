# orquesta-app-runner

Adaptador pequeno para preparar una ejecucion de Orquesta desde una
`AppSpecV0` validada.

Responsabilidades:

- construir el `OrchestrationRunV0` inicial mediante comandos del workflow;
- publicar decision, contrato funcional y microtareas durables antes de
  programacion;
- abrir la fase de programacion para el primer corte operativo;
- crear el plan de microtareas con `orquesta-app-planner`;
- devolver un `CandidateProviderPortV0` listo para el loop del nucleo;
- ejecutar el run preparado con puertos inyectados, esperar progreso externo y
  avanzar la siguiente ola sin que web/MCP conozcan scheduler interno.

No elige runtime, proveedor, modelo, HOME, credenciales ni persistencia.

Validacion:

```bash
go test -count=1 ./modulos/orquesta-app-runner
```
