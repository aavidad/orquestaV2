# Ola complementaria de autoprogramacion 10x6 - 2026-05-25

Este documento es un mapa operativo para el Director. No cierra la ola, no
declara pruebas reales pasadas y no amplia write-set: solo ordena secciones,
ownership y evidencias esperadas para completar capacidad total de 10 agentes
padre con hasta 6 subagentes gobernados por padre.

## Frontera

- Orquesta sigue siendo nucleo reutilizable; Codex, web y autoprogramacion son
  composiciones consumidoras.
- El modulo `orquesta-autoprogramming` conserva contratos puros: valida,
  agrupa, particiona y genera trabajo programable, pero no arranca agentes,
  lee repositorios ni ejecuta tests.
- La capacidad 10x6 queda expresada como 10 tareas padre y fanout maximo 6.
  Cualquier ampliacion por encima de ese mapa debe entrar como decision
  explicita de request, plan o composicion.
- Cada subarbol debe conservar `parent_agent_ref`, refs opacas, write-set,
  tests requeridos y ACK compacto.

## Mapa de padres

| Padre | Area primaria | Entrega esperada |
| --- | --- | --- |
| P01 | `modulos/orquesta-core-workflow` | Rails neutrales y outbox/replay sin producto. |
| P02 | `modulos/orquesta-orchestration-core` | Waits, plan-state, recursion y cierre causal offline. |
| P03 | `modulos/orquesta-autoprogramming` | Contratos, particionado, backlog scanner y review gate. |
| P04 | `modulos/orquesta-app-director-service` | Entrada/reentrada del Director y cierre por puerto. |
| P05 | `modulos/orquesta-app-codex-stack` | Composicion Codex opt-in, closure source y supervisor. |
| P06 | `modulos/orquesta-server` y `cmd/orquesta-server` | Residente, cola, shutdown y wiring opt-in. |
| P07 | `modulos/orquesta-web` | Panel fino: cola, runs, stats, control y errores publicos. |
| P08 | Docs y runbooks | Handoffs, matriz de pruebas y deuda verificable. |
| P09 | Smokes y pruebas reales | Scripts opt-in, fake-runtime y evidencia durable. |
| P10 | Integracion transversal | Conflictos de write-set, replan y estado vivo post-ola. |

## Plantilla de 6 subagentes por padre

Cada padre puede usar hasta seis hijos cuando el runtime/composicion lo autorice:

1. estructura e indice del area;
2. lectura de codigo vigente y contratos locales;
3. implementacion focal;
4. pruebas focales y diagnostico;
5. revision de riesgos, rails y frontera hexagonal;
6. docs, ACK y coordinacion de conflictos.

Si un padre no necesita seis hijos, debe dejar slots libres. No se improvisan
hijos fuera de refs materializadas por Orquesta.

## Plan de secciones para entregas

Cada entrega de la ola debe conservar estas secciones, en este orden:

1. `Alcance`: write-set usado y motivo si toca mas de un area.
2. `Contexto leido`: AGENTS locales y docs vigentes relevantes.
3. `Cambio`: rutas tocadas y razon tecnica.
4. `Pruebas`: comando exacto, resultado y si fue focal o obligatorio.
5. `Evidencia`: refs de ACK, artifacts o docs creados.
6. `Bloqueos`: conflictos de write-set, pruebas pendientes o replan necesario.

## Invariantes de cierre

- No cerrar por transcript: solo ACK, eventos, tests y evidencia durable.
- No esperar todos los agentes vivos si hay `WaitAgentRefs`, cohorte, ola o
  parent task.
- No ampliar la capacidad vigente por encima de 10x6 sin decision explicita.
- No declarar OPES, Codex real, recursion real nueva o smoke residente como
  cerrado sin prueba opt-in propia.
