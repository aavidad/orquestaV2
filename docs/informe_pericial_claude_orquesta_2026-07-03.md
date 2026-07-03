<!--
Informe pericial — Orquesta (núcleo y proceso)
Autor del análisis: revisión externa asistida (Claude Code, Fable 5)
Fecha: 2026-07-03
Alcance: núcleo de orquestación, ciclo de vida goal-first, estado vivo,
proceso de desarrollo. Los conectores se tratan solo donde afectan al núcleo.
Responde a: docs/auditoria_claude_estado_orquesta_2026-07-03.md (informe
preliminar del agente programador).
-->

# Informe pericial — por qué la app autónoma aún no es estable

## Veredicto

**El problema no es de lógica ni de estructura de capas: es de arquitectura de
estado y de proceso de desarrollo.** El repo compila entero, las fronteras
hexagonales se cumplen y están testeadas, y la autonomía funciona de punta a
punta en condiciones controladas (el smoke Nueva App del 2026-07-03 cerró un
goal completo y materializó una app de 28 ficheros sin intervención humana).
Lo que falta es *fiabilidad* de esa autonomía, y la causa raíz es medible.

## Datos objetivos de la revisión

| Métrica | Valor | Lectura |
| --- | --- | --- |
| `go build ./...` | limpio | no hay problema de build ni código roto |
| Tests de frontera (`architecture_boundaries_test.go`, 8 tests) | verdes | la disciplina hexagonal es real, no doctrinal |
| LOC producción / tests | ~280k / ~251k | ratio casi 1:1; cobertura seria |
| Edad del proyecto | primer commit 2026-03-17 (3,5 meses) | 2.359 commits, picos de 100+/día |
| Bugs inventariados 30-06 a 02-07 | **130 únicos en 3 días** (51+44+51); 11 abiertos | el bucle incidencia→fix funciona, pero es reactivo |
| Interfaces de estado (stores/registries/ledgers/markers/snapshots/colas) | **64** de 248 interfaces totales | el estado vivo no tiene dueño único |
| Endpoints de status/observación distintos | **≥15** (`autoprogramming/status`, `director/stats`, `queue/global-status`, `domain-work/status`, `readiness`, `observe`…) | cada uno re-deriva su propia foto |
| Variables de entorno `ORQUESTA_*` | **500** | política acumulada por flag |
| Flags de confirmación en smokes | 44 en 37 scripts | la ejecución real es cara y excepcional |
| Módulos `*director*` | **17** (4 generaciones de ciclo vivas) | ninguna migración fue sustractiva |
| Adaptador backend Codex Goal (`codex_goal_app_server_*`, tmux, websocket) | ~2.000 LOC en `cmd/orquesta-server` | el componente más crítico vive fuera de la disciplina modular |

## Hallazgos

### P1 — No existe fuente de verdad única del ciclo de vida · severidad: crítica (causa raíz principal)

El estado "¿este trabajo está vivo/cerrado/bloqueado/pendiente de rework?" se
reconstruye desde: `RunStore`, `GoalWorkStateStore`, run markers,
`WorkflowTaskStore` + wait state, `ProcessRegistry`, snapshots de proceso,
sesión tmux, statefile, outbox ledgers en fichero, receipts, checkpoints,
`work_delivery.json` y JSONs sueltos en `docs/`. Orquesta es de facto un
sistema distribuido (daemon + tmux + app-server + agentes + filesystem +
colas) sin espina única de eventos/estado.

Leído como conjunto, el inventario de bugs no contiene 130 problemas
distintos: la inmensa mayoría son variantes de *un mismo fallo* — dos
componentes derivan el estado vivo desde fuentes distintas y no coinciden
(`running_stale` con procesos vivos, `shutdown_ready=true` con proceso vivo,
`percent=100` con goal bloqueado, ACK sin cierre, receipt `invalid` con
artefactos útiles en disco).

La hipótesis del agente en su informe preliminar (converger a un único grafo
`run_ref → goal_ref → external_goal_ref → backend/process →
checkpoint/artifact → receipt → terminal/rework`) es **correcta y queda
confirmada con números**. Ese grafo hoy existe solo implícitamente, repartido
en 64 contratos.

### P2 — Cuatro generaciones del director conviven · severidad: alta

Loop legacy de `app-director-service`, Director Operativo V1, espina neutral
V2 (`director-cycle → runner → scheduler → outbox`) y goal-first (Codex Goal
como director interno). Cada refundación fue razonable por sí sola — y
goal-first es la decisión *acertada* — pero mantener las cuatro vivas
multiplica la superficie: cada observador de estado debe entender N ciclos,
cada bug se arregla en N sitios y los tests fijan el comportamiento de todos.
El esfuerzo que debía endurecer *un* camino se ha ido a mantener coherentes
cuatro. La doctrina lo institucionaliza: "no se borra sin evidencia
equivalente" — sin fecha de retirada, eso es acumulación indefinida.

### P3 — El componente más crítico vive en el sitio menos gobernado · severidad: alta

El adaptador real del backend goal (`cmd/orquesta-server/codex_goal_app_server_v0.go`,
`codex_goal_app_server_tmux_v0.go`, `codex_goal_app_server_websocket_protocol_v0.go`)
está en la raíz de composición (251 ficheros), fuera de la disciplina modular
y de los tests de frontera. Es exactamente donde se concentran los bugs de
lifecycle (BUG-023, 029, 033, 043, 044 y el residual de shutdown del smoke
Nueva App). No tiene máquina de estados explícita: los estados
Ensure/Launch/Observe/Shutdown se comprueban de forma dispersa.

### P4 — La validación se desplazó a incidentes, no a un camino E2E ejercitado · severidad: alta (proceso)

Patrón actual: tests offline/fake-runtime verdes + smokes reales caros y
gateados por confirmación manual (44 flags). Consecuencia: los bugs solo
afloran cuando OPES (el consumidor real) choca con ellos, produciendo tandas
de 40-50 bugs por contacto con la realidad. Los "falsos verdes" (`ready` que
no es ready, `100%` que no es 100%) son la diferencia entre fake-runtime y
realidad. El bucle incidencia→bug→test→cierre funciona (11 abiertos de 146
filas) pero es estructuralmente reactivo.

### P5 — El sistema acumula compensaciones en vez de colapsar causas · severidad: media-alta

500 env vars, ≥15 proyecciones de status, decenas de códigos diagnóstico con
`recommended_action`. Cada fix añade superficie observable en lugar de
eliminar una fuente de divergencia. La señal de mejora real sería que esos
contadores *bajen*.

### P6 — Deuda documental que genera defectos · severidad: media

`ARQUITECTURA.md` afirma persistencia principal sobre PostgreSQL; la realidad
del código es ledgers y markers en filesystem (`orquesta-persistence`,
`orquesta-state-file`). Con programadores LLM que leen la doctrina como
fuente de verdad, la documentación stale produce bugs directamente. El "orden
de autoridad documental" de 5 niveles es en sí mismo un síntoma.

### Lo que NO es el problema (fortalezas verificadas)

- La lógica micro: las funciones revisadas (`StartAppDirectorV0`,
  `startAppDirectorGoalFirstV0`, loop residente) son defensivas y cuidadosas.
- La disciplina hexagonal: real, testeada y verde.
- La cobertura de tests y la ausencia de código muerto.
- La elección goal-first: adelgazar Orquesta a compilar spec + lanzar +
  observar + validar cierre es la simplificación correcta.
- La autonomía funcional: el smoke Nueva App cerró goal, run y closure
  aceptado de punta a punta.

## Respuestas a las 5 preguntas del informe preliminar

1. **Nueva App generó Python pidiendo Go** → combinación: el `GoalWorkSpecV0`
   debe llevar lenguaje/framework como restricción dura (no preferencia en
   prosa del prompt) **y** el validador de cierre debe rechazarlo como
   required test. Un contrato que no se valida en el cierre no es contrato.
2. **Contrato mínimo terminal de `GoalWorkResultV0` para Nueva App** →
   manifest obligatorio con lenguaje, layout esperado (hexagonal), tests
   ejecutados con evidencia y los tres manuales. Sin manifest válido:
   `rework`, nunca `complete`.
3. **¿Readiness debe bloquear por backend goal degradado?** → separar
   readiness por capacidad: `ready_for_observe` (status utilizable) vs
   `ready_for_launch` (backend goal sano). Para lanzar goal real, bloquear es
   correcto; para solo observar, no.
4. **`shutdown_ready=true` con proceso vivo** → definir `ready` como "sin
   trabajo vivo + limpieza solicitada" y que el wrapper espere la salida del
   PID como observable separado. Es el mismo contrato ya aplicado al pane
   tmux en BUG-023; aplicarlo un nivel arriba, al proceso servidor. Bug de
   contrato temporal, no de lógica.
5. **Contratos MCP del remoto** → dirección correcta, pero cada campo de
   evidencia nuevo es más proyección: que todos lean del grafo único (R1),
   no que cada tool re-derive.

## Recomendaciones, en orden

1. **R1 — Materializar el grafo operativo único como código.** Un solo módulo
   dueño de la proyección de ciclo de vida (`run_ref → goal_ref → proceso →
   receipt → terminal`), alimentado por eventos, y **prohibir por test de
   frontera** que los endpoints de status deriven estado vivo desde otra
   fuente. Es el troceo más rentable: ataca de raíz a ~70% del inventario.
2. **R2 — Fecha de retirada al loop legacy y congelar la espina V2.**
   Goal-first como único camino de producción; legacy solo tras opt-in con
   sunset documentado; la espina V2 sin features nuevas salvo que se
   convierta en el interior de goal-first.
3. **R3 — Sacar el backend Codex Goal de `cmd/` a un módulo propio** con
   máquina de estados explícita (`Ensure → Launch → Observe → Shutdown` con
   estados nombrados) y cobertura de frontera.
4. **R4 — Smoke E2E real barato y periódico** (el de Nueva App, aislado,
   nightly). Casi todos los bugs del inventario los habría cazado un smoke
   recurrente antes de que OPES los sufriera. La fiabilidad se gana
   ejercitando el bucle, no añadiendo proyecciones.
5. **R5 — Regla de higiene:** cada cierre de bug debería *eliminar* una
   fuente de verdad, un flag o una proyección, no añadirlos. Medir nº de env
   vars (hoy 500) y nº de endpoints de status (hoy ≥15) como métrica de deuda.
6. **R6 — Sincronizar doctrina con realidad** (persistencia file-based vs
   "PostgreSQL"; consolidar el orden de autoridad documental). Los
   programadores son LLMs: doc stale = generador directo de defectos.

## Conclusión

La dirección es buena en lo esencial: goal-first es correcto, la disciplina
de fronteras es real y la autonomía ya funciona de punta a punta en
condiciones controladas. El proyecto no está a "meses" de una app autónoma:
está a **una consolidación de estado (R1) y una retirada de generaciones
(R2)** de distancia. El riesgo real no es técnico sino de trayectoria: si se
sigue cerrando bugs uno a uno añadiendo diagnósticos y flags, el inventario
crecerá al ritmo del contacto con la realidad (~50/día) indefinidamente. Con
R1-R4 ejecutadas, la mayor parte de ese inventario desaparece por
construcción.
