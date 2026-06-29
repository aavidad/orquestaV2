# orquesta-director-operativo

Contrato puro para el primer corte del Director Operativo V1; este modulo es
contrato puro, no runtime.

El objetivo es modelar lo que hoy hace bien un Codex coordinador externo:
mantener un plan vivo, dividir trabajo, lanzar subagentes, esperar, revisar,
replanificar y cerrar con pruebas. Este modulo solo define el plan operativo y
sus guardas. No arranca agentes ni conoce Codex, OPES, runtime, DB, web ni MCP.

Debe servir para:

- autoprogramacion de Orquesta;
- trabajos de dominio externos como OPES;
- futuros dominios que entren por contratos y refs opacas.

Tambien modela delegacion recursiva gobernada: un agente puede solicitar
subagentes, pero solo bajo presupuesto del director, profundidad maxima,
fanout maximo, parent/child refs y review posterior. No existe spawn libre.

Validacion:

```sh
go test -count=1 ./modulos/orquesta-director-operativo
```

## Indice operativo

- `README.md`: frontera, estado vigente y mapa rapido de owners.
- `docs/contratos.md`: DTOs, pasos, olas, reparacion y delegacion gobernada.
- `docs/pruebas.md`: cobertura local y pendientes que no pertenecen al contrato
  puro.
- `docs/ejemplos.md`: supuestos practicos, errores frecuentes y notas de test.
- `docs/tareas.md`: alias locales `DIR-OP-*` y su estado frente a cortes
  globales.

Estado tras el primer corte:

- este modulo llega hasta contrato puro, validacion y proyeccion a olas;
- la materializacion inicial vive en
  `modulos/orquesta-orchestration-core/operational_director_materializer_v0.go`;
- el materializador sigue limitado por diseno a items `launch_subagents` listos
  como workflow tasks y comandos `CreateMicrotask`, conservando linaje de
  ola/cohorte/parent y refs opacas compactas de dominio/evidencia como
  `context_refs`;
- waits por cohorte/ola/parent, ingesta acotada por `WaitAgentRefs`,
  review/rework/replan, tests requeridos durables, cierre causal y recursion
  Codex real ya tienen evidencia fuera de este modulo puro;
- este contrato puro no debe ampliarse con runtime: OPES temporal real de derivados/cierre
  quedo cerrado funcionalmente por goal-first; los residuales son calidad
  editorial, coste y automatizacion larga por conectores de dominio.

## Frontera vigente

| Pieza | Owner | Estado |
| --- | --- | --- |
| Contrato, validacion, presupuesto, pasos, olas y delegacion gobernada | `orquesta-director-operativo` | Vigente en este modulo puro. |
| `launch_subagents -> WorkflowTaskV0 -> CreateMicrotask` | `orquesta-orchestration-core` | Implementado como materializador parcial por puertos. |
| Wait por `wave_ref`, `cohort_ref` o `parent_task_ref` | `app-director-service` + stores | Cerrado para waits acotados; no esperar todo el run. |
| Review, required tests, replan y cierre causal | `app-director-service` + `orquesta-orchestration-core` | Cerrado offline/fake-runtime y con smokes Codex/no-OPES acotados. |
| Ola/cohorte Codex real amplia | `orquesta-app-codex-stack` | Cerrada por `CODEX-WAVE-REAL`; repetir solo ante regresion demostrada. |
| Recursion Codex real | `orquesta-app-codex-stack` | Cerrada por `CODEX-RECURSION-REAL`; no convierte Codex en nucleo. |
| OPES derivados hasta cierre | Conectores OPES opt-in | Cerrado funcionalmente en instancia temporal por runbook goal-first real; repetir solo ante regresion demostrada. |

## Mapa conceptual

```text
OperationalDirectorPlanV0
  -> OperationalDirectorWaveWorkV0
  -> WorkflowTaskV0 + CreateMicrotask
  -> wait por wave/cohort/parent
  -> review_deliveries
  -> run_required_tests si aplica
  -> replan_or_close / close por refs causales
```

Este modulo llega solo hasta las dos primeras piezas. El resto vive en
`orquesta-orchestration-core`, `app-director-service` y composiciones opt-in.

Guias locales:

- `docs/contratos.md`: forma y frontera de los DTOs.
- `docs/pruebas.md`: cobertura y bateria recomendada.
- `docs/ejemplos.md`: supuestos practicos, errores frecuentes y notas de test.
