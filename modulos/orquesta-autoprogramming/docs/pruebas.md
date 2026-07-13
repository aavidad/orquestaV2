# Pruebas: orquesta-autoprogramming

## Local

```sh
go test -count=1 ./modulos/orquesta-autoprogramming
```

Cobertura actual:

- acepta solicitud pequena con worktree aislada;
- valida fixtures versionadas de solicitud real web/MCP mediante
  `AutoprogrammingRequestSourceV0` y construye trabajo programable preservando
  refs opacas de origen;
- acepta ola de 10 tareas padre dentro de los limites vigentes;
- acepta ola amplia de 21 tareas padre cuando `max_task_refs`, `max_areas` y
  `max_write_set_entries` vienen declarados por contrato;
- rechaza contratos de limites negativos o superiores a 40;
- rechaza presupuestos de delegacion fuera de rango y preserva limites
  explicitos validos en `WorkflowTaskV0`;
- rechaza branch/ref/isolation incompletos;
- rechaza alcance demasiado amplio;
- rechaza `write_set` vacio o inseguro;
- agrupa tareas por area normalizada;
- preserva contratos explicitos por tarea: objetivo, contexto, context refs,
  criterios, tests y reglas compactas;
- compacta textos largos de tareas antes de `WorkflowTaskV0` y mantiene el
  payload durable por debajo del limite del core;
- transforma solicitudes validas en `WorkProfileV0`/`WorkflowTaskV0` con refs
  opacas y pruebas requeridas preservadas;
- inyecta guards de contrato/incidencia explicitos: startup-lock/codex_wrapper
  agrega tests de regresion, criterios de variables
  `ORQUESTA_CODEX_STARTUP_LOCK_*`, `context_refs` y `function_contract_refs`;
- clasifica tareas de autoprogramacion goal-first sin runtime: legacy
  compatible, bloqueada por capacidades Goal, lista para Goal, cubierta por
  Goal y legacy requerido;
- genera `GoalWorkSpecV0` valido cuando la clasificacion queda `goal_ready`,
  conserva `write_set`, pruebas requeridas, criterios, refs opacas y rule refs
  hard de contratos conocidos, y no genera specs si faltan capacidades Goal;
- `v1` materializa perfiles de trabajo por tipo de app/area/tarea, prioriza
  `task_ref` sobre area y conserva fallback `v0`;
- particiona `write_set` por area, normaliza aliases y rechaza rutas no
  asignables;
- secuencia rutas que coinciden con varias areas compatibles;
- pospone tareas cuyo `write_set` solapa trabajos vivos y conserva `depends_on`
  causal;
- rechaza trabajos vivos con refs imposibles o rutas inseguras;
- review gate acepta ACK completado con tests verdes;
- review gate rechaza ACK ausente, tests fallidos y tests ausentes;
- review gate acepta como rail blando reutilizable una entrega con tests verdes,
  ficheros grandes, ficheros fuera del `write_set` o destinos faltantes;
- review gate mantiene ACK ausente o tests fallidos como bloqueo de cierre con
  evidencia compacta;
- politica de capacidad usa tokens operativos para OPES/riesgo alto y no escala
  a `xhigh` por subcadenas inocuas como `scopes`;
- arquitectura impide importar core, runtime, DB, `cmd` o adaptadores.
- reconciliacion T208 no requiere codigo en este modulo; la evidencia se valida
  con la bateria cruzada del paquete y con ACK que declara
  `contexto_ref_only_resuelto`.
- automejora idle v0 resuelve defaults: 60 segundos por defecto, `0` desactiva
  solo el disparo por reloj idle y `target_queue` permite preparar trabajo
  secundario cuando hay capacidad libre y la cola visible esta por debajo del
  objetivo;
- planner de backlog salta tareas ya visibles en cola, filtra secciones
  narrativas explicitas y cualquier entrada sin `task_ref`, aunque conserve
  evidencia documental del scanner, y puede crear una tarea scanner `Escaneo backlog nuevos` con
  epoch, reservas, lineas y hashes de documentos; no duplica el scanner si ya
  esta visible por `task_ref` o por `section_ref`;
- proyeccion publica de trabajo externo distingue `outbox_pending`,
  `wait_external` y `external_process_verified`.

## Integracion focal

```sh
go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-mcp ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-orchestration-core
```

## Reconciliacion T208

```sh
go test -count=1 ./cmd/orquesta-guardian ./cmd/orquesta-server ./modulos/orquesta-autoprogramming ./modulos/orquesta-runtime-worktree ./modulos/orquesta-runtime-codex ./modulos/orquesta-server
```
