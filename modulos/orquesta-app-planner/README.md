# orquesta-app-planner

Planificador pequeno para partir una peticion de app en microtareas de
programacion con write-set acotado y dependencias explicitas.

No pertenece al nucleo de workflow durable. Es un adaptador/servicio exterior que
prepara candidatos para `orquestacionnucleoapp.CandidateProviderPortV0`.

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
