# orquesta-core-replanner

Responsabilidad: politica pura de replanificacion del nucleo.

Este microproyecto decide como salir de trabajo basura, revisiones rechazadas, tareas bloqueadas o agentes fallidos sin improvisar bucles. No ejecuta agentes ni modifica el workflow principal directamente. Primero define contratos candidatos y pruebas puras; despues se promueven cortes pequenos a `orquesta-core-workflow`.

Incluye:

- propuestas de rework;
- division de microtareas;
- sustitucion logica de agente;
- escalado de capacidad recomendado;
- consultas al director cuando no hay decision segura.

No incluye:

- runtime real;
- DB concreta;
- proveedor/modelo/HOME/OAuth;
- prompts completos o transcripts;
- cambios directos de ficheros de otros modulos sin tarea separada.

## Arranque

La ruta vigente para agentes OrquestaV2 es el servidor residente y la cola
gobernada, no el wrapper local. `./arrancar_codex.sh` queda reservado a
compatibilidad historica o recuperacion manual con error publico cuando no haya
contrato versionado.
