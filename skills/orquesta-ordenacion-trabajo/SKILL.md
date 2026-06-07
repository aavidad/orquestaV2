---
name: orquesta-ordenacion-trabajo
description: Ordenar trabajo en Orquesta como sistema transversal: objetivo, backlog, plan, olas, tareas, agentes, artefactos, revisiones, pruebas y cierre.
---

# Orquesta Ordenacion Trabajo

Usa esta skill cuando haya que convertir un objetivo grande en trabajo
ejecutable por agentes.

## Esqueleto

1. `objective`: que quiere el operador.
2. `scope`: dominio, repo, app o curso consumidor.
3. `backlog`: piezas pendientes, reutilizables y bloqueos reales.
4. `plan`: fases y dependencias.
5. `wave`: lote ejecutable en paralelo.
6. `task`: unidad concreta con write-set/refs/criterios.
7. `agent`: rol asignado y contexto minimo.
8. `artifact`: entrega versionada o ref opaca.
9. `review`: revision independiente, pares o consejo.
10. `tests`: evidencia de validacion.
11. `closure`: decision del Director y pendientes futuros.

## Contratos Reales

Usa el esqueleto anterior con los contratos vivos de Orquesta:

- `BacklogInicialPropuestoV0`: arranque de apps nuevas desde especificacion.
- `WorkflowTaskV0`: unidad ejecutable con `write_set`, criterios, tests,
  dependencias, `parent_task_ref`, `cohort_ref` y `wave_ref`.
- `WorkProfileV0`: perfil neutral de trabajo (`code_study`,
  `implementation`, `refactor`, `required_tests`, `documentation`, `review`,
  `domain_work`).
- `OperationalDirectorPlanV0`: plan operativo del Director, fases, olas y
  delegacion.
- `OperationalDirectorPlanStateV0`: estado vivo del plan, ola/cohorte,
  agentes pendientes y cierre.
- `WorkflowTaskWaitStateV0`: espera acotada por agentes, ola, cohorte o parent;
  no esperar todos los agentes vivos del run si hay refs concretas.
- `director-runner`, `director-scheduler`, `core-workflow` y outbox: ciclo
  puro para transformar candidatos en comandos/eventos sin tocar runtime real.

No inventes una cola paralela si estos contratos cubren el caso. Si falta un
dato de producto, dejalo en el adaptador o composicion consumidora.

## Priorizacion

Ordenar por:

- bloqueos que impiden avanzar;
- tareas con dependencias aguas abajo;
- trabajo paralelizable barato;
- validaciones tempranas que evitan rehacer;
- cierre/limpieza antes de abrir frentes nuevos.

## Estados

Usar estados simples:

- `pending`;
- `running`;
- `delivered`;
- `needs_rework`;
- `accepted`;
- `closed`;
- `blocked_real`.

No usar `failed` para trabajo recuperable. Convertirlo en `needs_rework`,
`draft`, `evidence` o `followup`.

## Evidencia minima

Cada tarea debe poder responder:

- quien la hizo;
- que refs uso;
- que entrego;
- que pruebas paso;
- que queda;
- que decision tomo el Director.

## Consumidores

OPES, programacion, documentacion, web o cualquier app externa usan este
esqueleto. Cada dominio aporta sus skills y contratos propios.
