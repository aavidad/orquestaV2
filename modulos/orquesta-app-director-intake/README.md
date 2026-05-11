# orquesta-app-director-intake

Entrada limpia para pedir una app completa a Orquesta arrancando por director.

Responsabilidades:

- avanzar un wizard conversacional de `AppSpecRequestV0` con preguntas compactas
  del director;
- recibir una `AppSpecV0` validada;
- crear un `OrchestrationRunV0` mediante comandos del workflow;
- abrir `brainstorming_arquitectura`;
- registrar las solicitudes de brainstorm necesarias;
- exponer un `CandidateProviderPortV0` que pide el director o equipo de
  directores.

Si la autonomia de agentes es `alta`, prepara un equipo pequeno de directores
por areas. El provider emite una cohorte inicial de hasta cuatro candidatos de
trabajo por tick: suficiente para director, web, API y persistencia, pero sin
abrir contextos grandes.

No planifica microtareas fijas ni elige proveedor, cuenta, credenciales,
runtime, HOME o persistencia.

Validacion:

```bash
go test -count=1 ./modulos/orquesta-app-director-intake
```
