# orquesta-autoprogramming

Mini-proyecto puro para validar solicitudes de autoprogramacion y resultados de
revision de codigo sin contaminar el nucleo de orquestacion.

Incluye:

- `AutoprogrammingRequestV0` y `ValidateAutoprogrammingRequestV0`;
- agrupacion compacta de tareas por area con
  `GroupAutoprogrammingTasksByAreaV0`;
- `AutoprogrammingReviewGateInputV0` y
  `EvaluateAutoprogrammingReviewGateV0`, con decision explicita de aceptar,
  conservar salida aprovechable y pedir follow-up;
- `BuildAutoprogrammingProgrammableWorkV0` para transformar solicitudes
  validadas en `WorkProfileV0`/`WorkflowTaskV0` por grupo sin solapar
  `write_set` salvo solapes seguros secuenciados, compactando textos largos
  antes del DTO durable para mantener tareas manejables;
- politica pura de particionado que normaliza aliases de area, declara
  reparaciones de rutas ambiguas y pospone tareas cuando hay trabajos vivos
  sobre el mismo `write_set`;
- `BuildAutoprogrammingSelfImprovementRequestV0` para convertir fallos
  observados por director/agentes en automejoras secundarias de baja prioridad,
  con evidencia, write-set propio y sin bloquear el trabajo principal;
- errores publicos tipados para issues de contrato.

Fuera de alcance:

- crear runs, arrancar agentes, ejecutar tests, leer repositorios o tocar disco;
- decidir proveedor, modelo, runtime, DB, HOME, OAuth, cuotas o estrategia de
  merge;
- depender de `orquesta-orchestration-core` o de adaptadores.

Validacion local:

```sh
go test -count=1 ./modulos/orquesta-autoprogramming
```
