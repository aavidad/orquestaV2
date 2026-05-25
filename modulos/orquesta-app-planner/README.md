# orquesta-app-planner

Planificador pequeno para partir una peticion de app en microtareas de
programacion con write-set acotado y dependencias explicitas.

No pertenece al nucleo de workflow durable. Es un adaptador/servicio exterior
que prepara planes deterministas para compatibilidad. La ruta operativa
preferente de `AppSpecV0` para apps nuevas es el Director V2 mediante
`orquesta.apps.arrancar_director.v0`; este planner no demuestra plan-state,
waits por ola/cohorte, review/tests/cierre ni recursion.

Responsabilidades:

- crear un plan determinista de microtareas;
- exponer solo la ola lista segun entregas ya registradas;
- construir `SchedulableWorkCandidateV0` con claims de toda la ola para que el
  gate de concurrencia pueda detectar solapes;
- mantener la informacion operacional fuera del core.

Validacion local:

```bash
go test -count=1 ./modulos/orquesta-app-planner
```
