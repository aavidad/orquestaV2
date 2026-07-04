<!--
Plan de mejora continua — Orquesta (fase 2 tras el plan pericial)
Autor: dirección Claude (Fable 5), decisión del propietario 2026-07-04
Origen: docs/informe_pericial_claude_orquesta_2026-07-03.md (cerrado) y
revisión experta de huecos y técnicas del campo.
Formato: cada sección MEJ-TASK-NNN es ejecutable por la automejora de
Orquesta (etiquetas canónicas del parser: Objetivo/Estado/Alcance/Criterios/
Tests). Pilotar con la receta de docs/bitacora_correccion_pericial_2026-07-03.md.
Paralelización por contrato (AGENTS.md): lanzar en ola todo lo que no comparta
Alcance.
-->

# Plan de mejora continua — huecos y técnicas

## Cómo usar este plan

- Grupo A (MEJ-TASK-101..106): huecos estructurales detectados por la
  dirección. Grupo B (MEJ-TASK-201..207): técnicas del campo con buen ratio
  esfuerzo/impacto.
- Olas sugeridas originalmente por Alcance disjunto:
  - Ola 1: 202 + 203 + 205 + 206 (integrada).
  - Ola 2: 201 + 204 + 104 (integrada con validacion local/fakes).
  - Ola 3: 102 + 105 + 207 (congelada por decision del operador).
  - Condicionales: 101 (cierre OPES real opt-in), 103 (supersedida por
    director de escalada para la forma residente).
  - Cerrada posterior: 106 (deuda residual/ratchets cerrada por Codex local en
    modo deuda gobernada). La retirada de `legacy_director_loop` queda como
    decision separada del operador, condicionada a checklist verificable.
- Cada cierre actualiza la bitácora pericial y no abre frentes fuera de su
  Alcance.
- Corte vigente 2026-07-03 noche: la cola de automejora queda congelada. No
  relanzar pilotajes ni tareas nuevas desde este documento salvo decision
  explicita del operador. Para evitar falsos relanzamientos, las tareas ya
  integradas o congeladas quedan con `Estado:` cerrado/aparcado, que el parser
  de automejora trata como no ejecutable.
- Auditoria 2026-07-04: tras la orden de paralelizar todo lo que no pise otros
  trabajos, solo queda lanzable `MEJ-101` si el operador confirma OPES temporal
  opt-in; `MEJ-102/103/105/207/T286-EXP` siguen aparcadas o supersedidas y no se
  deben arrancar automaticamente. La limpieza documental de historicos falsos se
  ejecuto por Orquesta como tarea aparte y queda registrada en el handoff.

## MEJ-TASK-101 ciclo-opes-real-validacion-campo

Objetivo: ejecutar y documentar el primer ciclo OPES completo sobre el nucleo nuevo contra OPES temporal aislado (nunca productivo): un tema de temario con derivados (tests, audio fake o preflight, HTML) pasando por goal-first, gates de cierre vigentes (visual, linguistico, html_ampliado, links) y reconciliacion por result materializado, siguiendo docs/runbooks/resultado_smoke_opes_derivados_goal_first_real_2026-06-28.md como base pero sobre HEAD actual. Producir runbook con resultados, bugs encontrados como filas de inventario y veredicto de si la autonomia de dominio queda validada en campo.

Estado: aparcado hasta decision operativa de cierre OPES real opt-in.

Alcance:

- `docs/runbooks`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `external/opes`

Criterios:

- OPES productivo intacto; solo temporal aislado con confirmaciones opt-in
- cada fallo observado queda como fila de inventario con evidencia
- veredicto explicito: autonomia de dominio validada o lista de bloqueos
- sin cambios de codigo en este ciclo: solo ejecucion, evidencia y triaje

Tests:

- el runbook nuevo existe y declara comandos reproducibles
- `git diff --stat` toca solo docs/ y external/opes

## MEJ-TASK-102 meta-director-olas-disjuntas

Objetivo: primer corte del meta-director: que una instancia de Orquesta pueda aceptar un objetivo compuesto por varias secciones de backlog pendientes, calcular cruces de Alcance entre ellas (interseccion de rutas normalizadas), y lanzar en paralelo como goals separados las que sean disjuntas, serializando las que se cruzan, con limite configurable de goals simultaneos reutilizando ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_MAX_REQUESTS. La integracion de resultados sigue siendo por revision (no auto-merge). Publicar en autoprogramming/status la ola activa: goals lanzados, en espera por cruce y terminados.

Estado: aparcado/congelado por orden del operador; no relanzar desde automejora.

Alcance:

- `modulos/orquesta-server`
- `modulos/orquesta-autoprogramming`
- `cmd/orquesta-server`
- `modulos/orquesta-mcp`

Criterios:

- dos secciones con Alcance disjunto se lanzan en paralelo en la misma instancia, cubierto por test con fakes
- dos secciones con Alcance cruzado se serializan con evidencia del cruce detectado
- el limite de simultaneos se respeta y se publica en status
- sin auto-merge: cada goal conserva write-set y result propios

Tests:

- `go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-autoprogramming`
- `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-mcp`
- `go test -count=1 ./`
- `go build ./...`

## MEJ-TASK-103 segundo-backend-goal-claude

Objetivo: primer corte de redundancia de proveedor: adaptador goal-first opt-in para Claude Code CLI como segundo backend del contrato CodexGoalStarterPortV0/ObserverPortV0 (o puerto neutral equivalente renombrado), reutilizando el patron app_server_tmux donde aplique o sesion CLI supervisada donde no, con el mismo contrato de result durable ORQUESTA_GOAL_RESULT_V0, write-set y required tests. Sin tocar el camino Codex por defecto; seleccion por configuracion explicita. Registrar la incidencia T18 (tier Gemini) como fuera de alcance de este corte.

Estado: aparcado/supersedido para la forma residente por el director de escalada; no relanzar sin credenciales y decision explicita.

Avance 2026-07-04 noche 12: los conectores CLI opt-in de Claude y Gemini ya
incluyen en su prompt operativo el contrato neutral de resultado durable
`orquesta_goal_result.v0` cuando el objetivo, criterios o tests pidan
goal-first/result durable. Ademas `orquesta-runtime-claude` ya tiene un primer
backend goal-first offline de fichero/control que implementa
`GoalWorkLauncherPortV0` y `GoalWorkObservationPortV0`: materializa spec/prompt
en runtime aislado y observa `orquesta_goal_result*.json` dentro del write-set.
No cambia defaults ni convierte Claude/Gemini en backend residente productivo.
La brecha de MEJ-103 sigue abierta hasta cablear seleccion opt-in por
composicion/env, proceso real Claude y smoke fake/real de lanzamiento.

Avance 2026-07-04 noche 14: `cmd/orquesta-server` ya reconoce
`ORQUESTA_CODEX_GOAL_BACKEND=claude_file_control` como backend goal-first
opt-in neutral sin anadir variables nuevas, respetando el ratchet MEJ-106 de
configuracion. La composicion expone los puertos `GoalWorkLauncherPortV0` y
`GoalWorkObservationPortV0` desde `orquesta-runtime-claude`, materializa
spec/prompt bajo control fuera del proyecto por defecto y deriva
`ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_GOAL_FIRST_ENABLED=true` cuando el
backend Claude esta configurado y no hay override explicito. Codex sigue siendo
el camino operativo normal cuando se configura
`ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`; valores no soportados siguen
fallando en arranque con diagnostico explicito. La brecha restante se reduce a
proceso Claude real supervisado con shutdown/control equivalente a Codex, smoke
fake/real opt-in de lanzamiento y backend propio Gemini.

Avance 2026-07-04 noche 15: `ORQUESTA_CODEX_GOAL_BACKEND=claude_process`
activa un backend Claude goal-first con proceso real supervisado por
`ProcessRuntimeConnectorV0`. Reutiliza el mismo contrato durable del backend
file-control, genera wrapper CLI por goal, lanza el comando Claude configurado
por `ORQUESTA_CLAUDE_COMMAND`, observa el resultado
`orquesta_goal_result_v0.json` y bloquea como
`claude_goal_process_stopped_without_result` si el proceso termina sin result
durable. Hay smoke fake de servidor con proceso real local. La brecha restante
ya no es lanzamiento de proceso, sino control/shutdown persistente equivalente
al backend Codex, smoke opt-in contra Claude real con credenciales y backend
propio Gemini.

Avance 2026-07-04 noche 16: `claude_process` ya expone control/stop en vida
del servidor por `GoalBackendControlPortV0`. `ClaudeGoalProcessBackendV0`
transporta `StopClaudeGoalV0`, detiene el `ProcessRuntimeConnectorV0`
asociado al `goal_ref`, devuelve `status=blocked`, `goal_status_set=true`,
`backend_stopped=true` y evidencias de stop solicitado/completado. El servidor
lo adapta al mismo puerto usado por run-control goal-first, de modo que un stop
forzado puede terminalizar un goal Claude activo igual que el backend Codex en
la misma instancia. La brecha pendiente queda acotada a persistencia/adopcion
del proceso tras reinicio, smoke opt-in contra Claude real con credenciales y
backend propio Gemini.

Avance 2026-07-04 noche 17: Gemini alcanza paridad offline/fake con el corte
Claude goal-first. `orquesta-runtime-gemini` incorpora backend
`GeminiGoalBackendV0` file-control y `GeminiGoalProcessBackendV0` con proceso
supervisado, wrapper por goal, observacion de `orquesta_goal_result_v0.json`,
bloqueo `gemini_goal_process_stopped_without_result` si el proceso termina sin
resultado durable y `StopGeminiGoalV0` para parar procesos vivos en la
instancia actual. `cmd/orquesta-server` reconoce
`ORQUESTA_CODEX_GOAL_BACKEND=gemini_file_control` y `gemini_process`, deriva
goal-first idle desde esos valores y adapta el control Gemini al mismo puerto
neutral de run-control. Codex sigue siendo el camino por defecto cuando se
configura `app_server_tmux`, y no se anaden variables `ORQUESTA_*` nuevas. La
brecha pendiente queda acotada a smoke opt-in contra Gemini real con
credenciales/tier valido, persistencia/adopcion de procesos Claude/Gemini tras
reinicio y una prueba real amplia de shutdown/control con proveedor externo.

Avance 2026-07-04 noche 18: queda cerrado el residual offline/fake de
persistencia/adopcion tras reinicio para `claude_process` y `gemini_process`.
Ambos backends escriben un manifiesto interno por goal en el runtime aislado
con refs opacas del `ProcessRuntimeConnectorV0` y PID solo interno, sin
publicarlo en specs/prompts/issues/evidencias publicas. Si una instancia nueva
no conserva el `process_ref` en memoria, carga el manifiesto, llama a
`AdoptProcessV0`, recupera el proceso vivo y puede observarlo o pararlo por el
mismo `GoalBackendControlPortV0`. La cobertura focal verifica backend
reiniciado, mapa en memoria vacio, adopcion y stop efectivo para Claude y
Gemini. La brecha pendiente queda reducida a smoke opt-in contra proveedores
reales con credenciales/tier validos y prueba real amplia de shutdown/control
con proveedor externo.

Avance 2026-07-04 noche 19: `claude_process` queda validado tambien contra
Claude Code real en smoke opt-in acotado. El test
`TestClaudeGoalProcessBackendV0RealOptInEscribeResultadoDurableV0`, activado
con `SMOKE_CLAUDE_GOAL_PROCESS_REAL=1`, lanzo Claude Code `2.1.201`, escribio
artefacto en proyecto temporal, produjo `orquesta_goal_result_v0.json`
parseable con `status=complete`, `artifact_refs`, `materialized_artifacts` con
`artifact_ref` y `required_test_results=passed`. A raiz del primer intento real
se endurecio el protocolo de prompt para exigir JSON puro sin markdown/fences y
`artifact_ref` no vacio en `materialized_artifacts`. Gemini CLI `0.45.1` sigue
bloqueado por `IneligibleTierError/UNSUPPORTED_CLIENT`; el diagnostico queda
como frontera externa ya cubierta por `BUG-ORQ-20260703-140`, no como bug nuevo
de Orquesta. Runbook:
`docs/runbooks/smoke_goal_first_provider_process_real_2026-07-04.md`.

Avance 2026-07-04 noche 20: `claude_process` queda validado tambien por la
superficie HTTP real de `cmd/orquesta-server`. El smoke opt-in
`scripts/smoke_goal_first_claude_process_server_real.sh` arranca un servidor
temporal con `ORQUESTA_CODEX_GOAL_BACKEND=claude_process`, lanza
`POST /api/v0/apps/director`, observa por
`POST /api/v0/apps/director/goal/observe` y cierra en `poll=31` con
`goal_status=complete`, `run_status=cerrada`, `closure_status=accepted` y
`closure_accepted=true`. La ejecucion retenida en
`/tmp/orquesta-claude-process-server.x5N8pq` produjo 9 `artifact_refs` y 16
`evidence_refs` reconciliadas por Orquesta; el result durable contiene 3 refs
requeridas, 13 paths, 3 artefactos materializados y 1 test requerido `passed`.
Durante el corte se corrigio una forma recuperable de proveedor
(`evidence_refs` como objetos `{ref, description}`) mediante normalizacion
acotada en Claude/Gemini y se fijo `--safe-mode` por defecto en el smoke para
evitar heredar MCPs locales de Claude. Gemini real sigue pendiente por
`IneligibleTierError/UNSUPPORTED_CLIENT`; la prueba real de
`runs/control`/shutdown con proveedor externo vivo o lento sigue como residual
separado de `BUG-ORQ-20260704-165`.

Alcance:

- `modulos/orquesta-runtime-claude`
- `modulos/orquesta-goal`
- `cmd/orquesta-server`
- `docs/runbooks`

Criterios:

- con backend claude configurado, un goal acotado fake/real opt-in lanza, observa y reconcilia por result durable
- sin configuracion explicita nada cambia para Codex
- el contrato neutral no gana dependencias especificas de proveedor
- runbook con precondiciones de credenciales y smoke opt-in

Tests:

- `go test -count=1 ./modulos/orquesta-runtime-claude ./modulos/orquesta-goal`
- `go test -count=1 ./cmd/orquesta-server`
- `go test -count=1 ./`
- `go build ./...`

## MEJ-TASK-104 gobernador-presupuesto-goals

Objetivo: convertir la politica frugal en decision medida: un gobernador de presupuesto que, con las metricas 801A (context_budget), el prompt_cache 801C y un presupuesto operativo declarado por configuracion (tokens o goals por dia), decida antes de lanzar automejora idle si procede lanzar, aplazar con razon budget_deferred o degradar a tarea mas barata. Publicar en autoprogramming/status el presupuesto restante estimado y los aplazamientos con razon. Sin env vars nuevas mas alla del presupuesto declarado.

Estado: cerrado por MEJ-104/T290 y smoke real acotado del 2026-07-04. El smoke
temporal valido `budget_deferred` en `/api/v0/server/status` y
`/api/v0/autoprogramming/status` con backend `app_server_tmux` configurado y sin
lanzar goal Codex.

Alcance:

- `modulos/orquesta-server`
- `modulos/orquesta-autoprogramming`
- `cmd/orquesta-server`
- `modulos/orquesta-mcp`

Criterios:

- con presupuesto agotado, la automejora idle aplaza con budget_deferred visible en status, no lanza en silencio
- con presupuesto disponible el comportamiento actual no cambia
- el gasto estimado por goal usa metricas reales 801A, no constantes inventadas
- test con fakes de las tres decisiones: lanzar, aplazar, degradar

Tests:

- `go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-autoprogramming`
- `go test -count=1 ./modulos/orquesta-mcp ./cmd/orquesta-server`
- `go test -count=1 ./`
- `go build ./...`

## MEJ-TASK-105 memoria-entre-goals-fitness

Objetivo: primer corte de memoria entre goals: al cerrar un goal (aceptado o rework), destilar una leccion compacta y anonima de rutas privadas (clase de tarea, modulo, resultado, tests fallados/pasados, tokens 801A) a un almacen file-based append-only bajo el state dir, y permitir que el compilador de GoalWorkSpec inyecte como context_refs las 3 lecciones mas relevantes por clase de tarea/modulo. Base para el fitness por materias de ARQUITECTURA.md sin construir aun el scheduler ponderado.

Estado: aparcado/congelado por orden del operador; no relanzar desde automejora.

Alcance:

- `modulos/orquesta-autoprogramming`
- `modulos/orquesta-server`
- `modulos/orquesta-app-codex-stack`
- `cmd/orquesta-server`

Criterios:

- las lecciones no contienen rutas absolutas, tokens ni prompts completos
- un goal nuevo de la misma clase recibe hasta 3 lecciones como refs compactas
- el almacen es append-only, acotado por retencion configurable, y su ausencia no rompe nada
- sin store de base de datos nueva: file-based bajo state dir existente

Tests:

- `go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-server`
- `go test -count=1 ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`
- `go test -count=1 ./`
- `go build ./...`

## MEJ-TASK-106 deuda-residual-gobernada

Objetivo: gobernar la deuda residual conocida sin big-bang: (1) plan de troceo incremental del hub orquesta-core-workflow actualizando docs/runbooks/plan_troceo_hubs_orquesta_2026-07-01.md con el primer subpaquete concreto a extraer y su ratchet; (2) ratchet descendente de env vars: test raiz que fija el numero actual de ORQUESTA_* (medido con scripts/orquesta_metricas_deuda.sh) como maximo y obliga a bajar para anadir; (3) checklist de retirada del legacy_director_loop condicionada a ventana §9 verde y segundo backend goal, documentada para decision del operador, sin borrar codigo en esta tarea.

Estado: cerrado por Codex local en modo deuda gobernada; retirada legacy no ejecutada y queda condicionada a checklist/decision explicita.

Alcance:

- `docs/runbooks`
- `env_vars_budget_test.go`
- `scripts`

Criterios:

- el ratchet de env vars falla si el conteo sube y pasa si baja o se mantiene
- el plan de troceo nombra el primer subpaquete, sus ficheros y su test focal
- la checklist de retirada legacy lista condiciones verificables, no fechas blandas
- ningun codigo de produccion cambia en esta tarea

Tests:

- `go test -count=1 ./ -run TestEnvVarsBudget`
- `go test -count=1 ./`
- `bash scripts/orquesta_metricas_deuda.sh`

## MEJ-TASK-201 simulacion-determinista-fallos

Objetivo: simulador determinista del ciclo goal-first con inyeccion sistematica de fallos, estilo FoundationDB: con reloj falso, backend falso y stores in-memory existentes, un arnes que ejecuta el ciclo completo (lanzar, observar, materializar result, reconciliar, cerrar) matando el backend, retrasando la materializacion, duplicando observaciones o cortando el proceso en CADA punto de transicion enumerable, y verifica los invariantes: ningun goal queda running eterno, ningun terminal desaparece sin estado visible, ningun conflicto se resuelve en silencio, ninguna transicion ilegal de TransicionBackendV0. Semilla reproducible: cada fallo detectado imprime la secuencia exacta para reproducirlo.

Estado: cerrado por T291/df2bed2f, integrado y verificado con simulador determinista.

Alcance:

- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-server`
- `modulos/orquesta-estado-vivo`

Criterios:

- el arnes enumera los puntos de inyeccion desde las transiciones reales, no lista manual
- corre en menos de 60 segundos en modo corto y es determinista por semilla
- cada invariante violado falla con la secuencia reproducible
- entra en go test normal (modo corto) sin flags nuevos

Tests:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestSimulacion'`
- `go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-estado-vivo`
- `go test -count=1 ./`
- `go build ./...`

## MEJ-TASK-202 property-based-nucleo-puro

Objetivo: tests basados en propiedades (pgregory.net/rapid, dependencia solo de test) para las funciones puras del nucleo: ConstruirProyeccionCicloVidaV0 (determinismo bajo permutacion de evidencias, ningun conflicto silencioso, monotonia de precedencias), TransicionBackendV0 (sin transiciones ilegales alcanzables, idempotencia de observaciones repetidas) y ReconcileExternalWorkPublicStatusV0 (estado publico nunca completed sin evidencia terminal). Minimo 4 propiedades por funcion con generadores de entradas.

Estado: cerrado por T287/a9f3b455, integrado y verificado.

Alcance:

- `modulos/orquesta-estado-vivo`
- `modulos/orquesta-runtime-codex-appserver`
- `modulos/orquesta-run-coordinator`
- `go.mod`
- `go.sum`

Criterios:

- rapid solo como dependencia de test; go build ./... sin cambios de dependencias de produccion
- cada propiedad documenta el invariante en una linea
- una regresion sintetica (mutacion manual local) es cazada por al menos una propiedad antes de commitear
- corren en modo corto en menos de 30 segundos

Tests:

- `go test -count=1 ./modulos/orquesta-estado-vivo ./modulos/orquesta-runtime-codex-appserver ./modulos/orquesta-run-coordinator`
- `go test -count=1 ./`
- `go build ./...`

## MEJ-TASK-203 mutation-testing-piloto

Objetivo: piloto de mutation testing acotado y opt-in: script scripts/orquesta_mutation_pilot.sh que ejecute go-mutesting (o herramienta Go equivalente activa) SOLO sobre modulos/orquesta-estado-vivo y modulos/orquesta-goal, publique el mutation score por paquete en un JSON datado bajo el directorio de resultados nightly, y documente en runbook el umbral aceptable propuesto y los mutantes supervivientes mas graves como candidatos a test nuevo. No entra en CI por defecto: solo opt-in manual/nightly extendido.

Estado: cerrado por T288/a9f3b455, integrado y verificado como piloto opt-in.

Alcance:

- `scripts`
- `docs/runbooks`

Criterios:

- opt-in explicito; sin dependencia obligatoria nueva en go.mod
- score por paquete reproducible y datado
- runbook con umbral propuesto y top de mutantes supervivientes
- no toca codigo de produccion

Tests:

- `bash -n scripts/orquesta_mutation_pilot.sh`
- test shell focal del script
- `git diff --check`

## MEJ-TASK-204 tests-congelados-actor-critico

Objetivo: separar quien define los tests de quien implementa en la automejora idle: modo opt-in en el que, antes de lanzar el goal implementador, se lanza un goal barato definidor que SOLO escribe los required tests (ficheros _test.go nuevos con casos que deben pasar) y los congela con hash en el GoalWorkSpec del implementador; el implementador no puede modificar los tests congelados (la validacion de cierre rechaza diffs sobre ellos con reason frozen_tests_modified) y su cierre exige que pasen. Aplicable primero a tareas de backlog con Alcance de un solo modulo.

Estado: cerrado por T292/0ce95763, integrado y verificado.

Alcance:

- `modulos/orquesta-autoprogramming`
- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-server`
- `cmd/orquesta-server`

Criterios:

- modo opt-in; sin el flag el flujo actual no cambia
- el implementador que toca un test congelado queda en rework con frozen_tests_modified
- el cierre exige tests congelados en verde con evidencia
- cubierto de punta a punta con fakes

Tests:

- `go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-app-codex-stack`
- `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`
- `go test -count=1 ./`
- `go build ./...`

## MEJ-TASK-205 golden-tasks-evals

Objetivo: banco de tareas doradas para medir deriva de calidad de agentes: definir en docs/evals/ un set inicial de 5 tareas acotadas con resultado esperado verificable (la Nueva App del smoke como primera; una tarea de script tipo T265; una de docs tipo T280; una de fix con test congelado; una de exploracion tipo mapa publico), un script scripts/orquesta_golden_evals.sh opt-in que las lance en serie o paralelo contra una instancia aislada usando la receta de pilotaje, y un evaluador que puntue cada resultado con los criterios declarados (tests pasan, ficheros esperados, sin fuera de write-set) a JSON datado. Runbook con cadencia recomendada semanal.

Estado: cerrado por T289/a9f3b455, integrado y verificado.

Alcance:

- `docs/evals`
- `scripts`
- `docs/runbooks`

Criterios:

- cada tarea dorada declara criterios verificables por script, no juicio manual
- el resultado es un JSON datado comparable entre ejecuciones
- opt-in con confirmacion; nunca toca OPES productivo
- las 5 tareas cubren clases distintas de trabajo

Tests:

- `bash -n scripts/orquesta_golden_evals.sh`
- test shell focal del script
- `git diff --check`

## MEJ-TASK-206 biblioteca-habilidades-curada

Objetivo: curacion de habilidades reutilizables estilo Voyager: un goal idle opcional que revisa results durables aceptados recientes, detecta artefactos reutilizables (scripts, patrones de test, plantillas de runbook), y propone su destilacion a skills/ como habilidad con contrato (nombre, cuando usarla, entrada/salida, validacion) via seccion de backlog nueva para revision, nunca copiando secretos ni rutas privadas. El compilador de GoalWorkSpec anade como context_refs las habilidades cuyo nombre/etiquetas casen con la clase de tarea.

Estado: cerrado por MEJ-206/a9f3b455 mediante reparacion acotada; no relanzar destilacion autonoma.

Alcance:

- `skills`
- `modulos/orquesta-autoprogramming`
- `cmd/orquesta-server`

Criterios:

- la destilacion siempre pasa por seccion de backlog y revision, no auto-commit de skills
- una habilidad casada por clase llega al spec como ref compacta, cubierto por test
- sin secretos ni rutas absolutas en habilidades
- con el directorio skills/ vacio nada cambia

Tests:

- `go test -count=1 ./modulos/orquesta-autoprogramming ./cmd/orquesta-server`
- `go test -count=1 ./`
- `go build ./...`

## MEJ-TASK-207 cascada-modelos-medida

Objetivo: convertir la escalera worker-barato/repair-helper/prime en decision medida estilo FrugalGPT: registrar por goal el modelo/reasoning usado y el desenlace (aceptado, rework, escalado) junto a las metricas 801A; producir una proyeccion en autoprogramming/status con el ratio de exito por escalon y coste medio; y permitir configurar umbral de escalado por evidencia (n fallos del barato en la clase de tarea) en lugar de heuristica fija. Sin cambiar el default actual hasta tener 2 semanas de datos.

Estado: aparcado/congelado por orden del operador; no cambiar defaults sin datos y decision explicita.

Alcance:

- `modulos/orquesta-autoprogramming`
- `modulos/orquesta-server`
- `modulos/orquesta-mcp`
- `cmd/orquesta-server`

Criterios:

- el registro por goal no publica prompts ni secretos
- la proyeccion de ratios por escalon es visible en status con datos falsos de test
- el umbral configurable no altera el comportamiento por defecto
- decision de cambio de default documentada como pendiente de datos, no aplicada

Tests:

- `go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-server`
- `go test -count=1 ./modulos/orquesta-mcp ./cmd/orquesta-server`
- `go test -count=1 ./`
- `go build ./...`
