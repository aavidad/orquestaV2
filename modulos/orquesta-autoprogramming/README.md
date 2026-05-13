# orquesta-autoprogramming

Mini-proyecto puro para validar solicitudes de autoprogramacion y resultados de
revision de codigo sin contaminar el nucleo de orquestacion.

Incluye:

- `AutoprogrammingRequestV0` y `ValidateAutoprogrammingRequestV0`;
- agrupacion compacta de tareas por area con
  `GroupAutoprogrammingTasksByAreaV0`;
- `AutoprogrammingReviewGateInputV0` y
  `EvaluateAutoprogrammingReviewGateV0`;
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
