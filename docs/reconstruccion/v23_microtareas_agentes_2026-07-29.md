# V23 dividido en microtareas para agentes

Fecha de corte: 2026-07-29. Base funcional:
`232ed4ef4485c596c55d708182244de207dee7bd`.

Manifiesto ejecutable:
`docs/reconstruccion/worksets/v23_agent_microtasks_v1.json`.

## Decisión

V23 deja de ser un frente abierto. Se divide en 13 tareas cerradas, cuatro
fases y dos olas paralelas explícitas. Cada tarea declara objetivo, fichero y
símbolo objetivo, precondiciones, postcondiciones, dependencias, read-set,
write-set, pruebas, negativos, artefactos y límites de tamaño/tiempo.

No se reimplementan los 24 scopes que la fixture ya declara integrados. El
trabajo pendiente real de V23 queda reducido a:

1. congelar el alcance correcto;
2. cerrar el replay exacto de la evaluación de gaps;
3. acreditar la generación de dossier ya existente;
4. acreditar la ayuda contextual ya existente;
5. recalcular la matriz por pruebas;
6. ejecutar gate, dos reviews independientes y seal.

## Alcance que cambia de dueño

No se elimina ninguna capacidad:

| Capacidad | Dueño nuevo | Motivo |
|---|---|---|
| `UI-05` | V24 | La web es parte del vertical de administración web. |
| `WIZ-13` | V24 | Marca, tema y apariencia pertenecen a presentación. |
| `WIZ-10` concreto | V28 | V23 conserva `work_existing` y refs opacas; el adaptador de Analizador es una integración externa. |

Esta transferencia evita tres errores repetidos: meter UI en el dominio,
convertir una integración externa en requisito del Wizard y hacer depender V24
de una V23 que solo podría cerrar después de V24.

Firecracker, Bubblewrap, perfiles HOME, selección de modelos, RAG, bases
vectoriales, skills y herramientas de contexto tampoco forman parte del gate
V23. Permanecen en el inventario transversal y en sus verticales/adaptadores;
no se descartan.

## Olas

| Ola | Tareas | Paralelismo |
|---|---|---|
| 0 · alcance | `v23_01_scope_freeze` | Serial. Fija autoridad antes de escribir. |
| 1A · núcleo | `v23_02_snapshot_application`, `v23_07_dossier_characterization`, `v23_08_help_characterization` | Paralelas; write-sets disjuntos. |
| 1B · snapshot durable | `v23_03` → `v23_04` → `v23_05` → `v23_06` | Serial por causalidad schema/store/recovery/publicación. |
| 2 · aceptación | `v23_09` → `v23_10` | Serial; fixture y candidato tienen escritor único. |
| 3 · revisión | `v23_11_primary_review` y `v23_12_adversarial_review` | Paralelas, agentes y launches distintos. |
| 4 · cierre | `v23_13_seal` | Serial; exige ambos ballots PASS. |

La concurrencia máxima útil del plan es tres agentes en la primera ola y dos
revisores en la última. Lanzar diez agentes no acelera este corte: seis tareas
comparten una cadena causal y cuatro escriben autoridades comunes.

## Reglas de ejecución

- Cada agente recibe una sola entrada de `tasks[]`, no el manifiesto completo.
- Antes de editar ejecuta el preflight exacto declarado en esa entrada.
- No amplía el write-set. Un fallo fuera de scope produce `blocked` y una tarea
  hija con contrato propio.
- Dossier y ayuda son tareas de caracterización. Solo pueden añadir sus tests
  de aceptación. No pueden “aprovechar” para rediseñar producción.
- El snapshot reutiliza `internal/wizard/gaps/result_snapshot.go`; queda
  prohibida una segunda serialización o reevaluar con el catálogo actual.
- `evaluation_replay_exact=true` solo puede aparecer después de restaurar y
  validar bytes históricos durables.
- Reviews primaria y adversarial deben usar agentes distintos del autor y entre
  sí. No corrigen código dentro del ballot.
- Cada entrega incluye commit pequeño en castellano, salida de pruebas,
  riesgos/bloqueos y siguiente acción.

## Proyección a `PlanSpec`

El manifiesto contiene información adicional que `PlanSpec` todavía no modela.
El emisor conserva dentro del plan ejecutable las fases, dependencias,
write-sets, pruebas, capacidad, esfuerzo y criticidad. El target, pre/post,
negativos y límites permanecen en el manifiesto como contrato del agente.

```bash
scripts/emitir_plan_v23_microtareas.sh \
  > /tmp/orquesta-v23-plan.json
```

Validación determinista, sin LLM:

```bash
scripts/test_v23_agent_microtasks.sh
```

El test comprueba conteos, DAG, dependencias, scopes pendientes, transferencias,
límites y ausencia de solapes dentro de cada ola paralela.

## Criterio de cierre

V23 no termina porque “parece al 85 %”. Termina cuando:

- las 13 tareas están en disposición terminal causal;
- los seis scopes diferidos quedan cerrados o transferidos de forma explícita;
- ningún scope ya integrado fue reabierto;
- ambos reviews independientes son PASS;
- `product/evidence/v23_wizard.json` acredita el mismo candidato probado;
- roadmap y fixture se promocionan en el último commit, nunca antes.

Si una tarea descubre un defecto nuevo, V23 no vuelve a crecer sin límite. El
Director crea una microtarea hija con write-set y prueba exactos, la inserta
antes del gate que dependa de ella y conserva el resto del DAG.
