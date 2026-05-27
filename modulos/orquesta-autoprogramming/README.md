# orquesta-autoprogramming

Mini-proyecto puro para validar solicitudes de autoprogramacion y resultados de
revision de codigo sin contaminar el nucleo de orquestacion.

Incluye:

- `AutoprogrammingRequestV0` y `ValidateAutoprogrammingRequestV0`;
- `AutoprogrammingRequestSourceV0` para validar que una solicitud real de
  entrada publica conserva superficie, transporte, identidad publica y refs de
  evidencia sin acoplarse a web, MCP ni HTTP concretos;
- agrupacion compacta de tareas por area con
  `GroupAutoprogrammingTasksByAreaV0`;
- `AutoprogrammingReviewGateInputV0` y
  `EvaluateAutoprogrammingReviewGateV0`, con decision explicita de aceptar,
  conservar salida aprovechable y pedir follow-up;
- `BuildAutoprogrammingProgrammableWorkV0` para transformar solicitudes
  validadas en `WorkProfileV0`/`WorkflowTaskV0` por grupo sin solapar
  `write_set` salvo solapes seguros secuenciados, compactando textos largos
  antes del DTO durable para mantener tareas manejables;
- `AutoprogrammingRequestV1` y `BuildAutoprogrammingProgrammableWorkV1` para
  declarar perfiles de trabajo por tipo de app, area o tarea sin romper `v0`;
- politica pura de particionado que normaliza aliases de area, declara
  reparaciones de rutas ambiguas y pospone tareas cuando hay trabajos vivos
  sobre el mismo `write_set`;
- `BuildAutoprogrammingSelfImprovementRequestV0` para convertir fallos
  observados por director/agentes en automejoras secundarias de baja prioridad,
  con evidencia, write-set propio y sin bloquear el trabajo principal;
- errores publicos tipados para issues de contrato.

Nota T208 2026-05-27: este modulo solo aporta contratos puros para
autoprogramacion. La reconciliacion del guardian break-glass queda cerrada como
umbrella documental; cualquier reparacion futura debe entrar como owner
especifico, no como regla nueva de runtime dentro de este modulo.

Fuera de alcance:

- crear runs, arrancar agentes, ejecutar tests, leer repositorios o tocar disco;
- decidir proveedor, modelo, runtime, DB, HOME, OAuth, cuotas o estrategia de
  merge;
- depender de `orquesta-orchestration-core` o de adaptadores.

Validacion local:

```sh
go test -count=1 ./modulos/orquesta-autoprogramming
```
