# Guia de continuacion para agentes - programacion autonoma de Orquesta

Autor: Claude Fable (revisor/director), 2026-07-10.
Para: agentes con menos contexto/capacidad (Claude en modo barato, subagentes,
o cualquier sesion nueva) que continuen la programacion mientras Codex no
tiene cuota. Objetivo final del operador: que Orquesta se programe a si misma;
esta guia cubre lo que falta para llegar ahi.

## Como usar esta guia

1. Lee "Reglas de oro" y "Que NO tocar" completas antes del primer edit.
2. Ejecuta UN paso cada vez, en el orden dado. Cada paso es completo:
   codigo + test + commit + push. Si la sesion muere a mitad de paso,
   `git status` te dice donde quedaste; si el diff es confuso, `git stash`
   y repite el paso desde cero.
3. Al terminar cada paso: marca el checkbox en este documento (editar la
   linea `- [ ]` a `- [x]`) EN EL MISMO COMMIT del paso.
4. Estado global: `git log --oneline -15`, `docs/inventario_bugs_estado_vivo.md`
   y este documento. No necesitas leer el inventario historico (3900 lineas).

## Reglas de oro

- NUNCA ejecutes `go test ./...` global: mata la sesion por memoria.
  Verifica SOLO los paquetes tocados: `go test -count=1 ./modulos/<paquete>`.
  Para verificacion amplia existe `scripts/orquesta_test_batches.sh` (dos
  pases, aislado); solo al final de una etapa, nunca por paso.
- Un paso = un commit + `git push origin trabajo/plataforma-agentes`
  inmediato. Nunca acumules dos pasos sin pushear.
- Mensaje de commit: espanol, prefijo `fix:`/`feat:`/`docs:`/`test:`, y
  termina con la linea `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`.
- Antes de editar un fichero, leelo. Despues de editar: test focal del
  paquete + `git diff --check` (espacios colgantes rompen guards).
- No te fies de ningun verde autodeclarado (tuyo ni de nadie): ejecuta el
  test tu mismo y mira el `ok` real. Este proyecto ya sufrio dos falsos
  verdes por fixtures inventados (F3-R2 y 208H).
- Si un paso te pide crear ficheros nuevos en un modulo, replica el estilo
  del modulo (nombres `*_v0.go`, tests `*_v0_test.go`, sin dependencias
  nuevas fuera de stdlib).
- Si algo no cuadra con esta guia (simbolo que no existe, test que ya
  cubre el caso), NO fuerces: anota el hallazgo en la seccion "Bitacora de
  desviaciones" al final, commitea y sigue con el paso siguiente.

## Que NO tocar

- Worktrees de Codex (solo LECTURA): todo lo que cuelga de
  `/home/alberto/Trabajo/orquesta-worktrees/` y
  `/home/alberto/Trabajo/orquesta-wt-lease-generation-20260710`.
  En particular 208H (`wip/attestation-208h-20260710`, 36 ficheros sin
  commitear) es de Codex: no lo integres ni lo rehagas.
- El servidor remoto `srv1651826` y `uso-app`: prohibido conectar.
- `scripts/orquesta_server_drain.sh`, `orquesta_server_deploy.sh`,
  `orquesta_server_ctl.sh`: cerrados y con guards; no los modifiques.
- No lances smokes reales (`*_real.sh` con doble confirm): consumen cuota
  Codex que no hay. Quedan para la etapa D.
- No subas el ratchet de envs (`env_vars_orquesta`, limite 513): si un test
  de budget falla por tu cambio, tu cambio sobra una env.

## Contexto minimo (30 segundos)

Orquesta es un director de agentes goal-first en Go (modulos en `modulos/`,
server en `cmd/orquesta-server`). Hoy: el circuito goal-first local YA
funciona (smoke verde end-to-end tras el fix del lease `01cb27d77`). Falta,
para autonomia: que TODAS las superficies publiquen estado desde el
veredicto causal unico (F1), que `runs/control` pueda parar de verdad un
backend (F2), la atestacion de tests (208H, de Codex), y el piloto real.
Analisis completo: `docs/analisis_fallos_estructurales_orquesta_2026-07-10.md`.

## ETAPA A - Adoptar el veredicto causal F1 en las superficies

El reconciliador unico ya existe:
`DerivarVeredictoCausalV0(evidencias []EvidenciaEstadoV0) VeredictoCausalV0`
en `modulos/orquesta-estado-vivo/veredicto_causal_v0.go`. Contratos:
`modulos/orquesta-estado-vivo/docs/contratos.md` (leer primero).
Referencia de consumo YA hecha (copiar el patron):
`modulos/orquesta-server/operational_status_estado_vivo_v0.go` recibe
`evidencias []orquestaestadovivo.EvidenciaEstadoV0` y deriva el veredicto.
Fuente de evidencias en el stack:
`modulos/orquesta-app-codex-stack/evidencia_estado_procesos_v0.go`
(`ListarEvidenciasEstadoV0`).

Regla que implanta cada paso: una superficie NO publica `running` si el
veredicto no es `running` confirmado; si hay resultado terminal durable, la
superficie publica el terminal, no `running`. El veredicto viaja como
campos nuevos opcionales (`causal_verdict`, `causal_reason_code`), sin
romper campos existentes.

- [x] PASO A1: veredicto en `observe_goal` (MCP).
  Ficheros: `modulos/orquesta-mcp/observe_app_director_goal_tool_v0.go` y su
  test. El executor debe aceptar (opcional, inyectada) una fuente de
  evidencias con la misma interfaz que usa operational-status; si esta
  presente, deriva el veredicto y: (a) anade `causal_verdict` +
  `causal_reason_code` al resultado; (b) si el veredicto contradice
  `goal_status=running` (clase terminal-by-artifact o process-dead), cambia
  `recommended_action` a `reconcile_goal_state` y NO publica running como
  estado limpio. Sin fuente inyectada: comportamiento actual intacto.
  Verifica: `go test -count=1 ./modulos/orquesta-mcp`.

- [x] PASO A2: cablear la fuente real de A1 en la composicion.
  Ficheros: donde el stack construye el executor de observe (buscar con
  `rg -ln "ObserveAppDirectorGoal" modulos/orquesta-app-codex-stack cmd/orquesta-server`).
  Pasa el `AutoprogrammingEstadoVivoSource`/adaptador de evidencias ya
  existente (mismo que consume operational-status) al executor MCP.
  Test de wiring en el stack (patron: buscar tests `TestBuildStack*Cablea*`).
  Verifica: `go test -count=1 ./modulos/orquesta-app-codex-stack`.

- [x] PASO A3: veredicto en `autoprogramming/status`.
  Ficheros: `rg -ln "autoprogramming.status" modulos/orquesta-mcp` para
  localizar el executor. Mismo contrato que A1: campos opcionales + nunca
  running contradicho. Verifica: `go test -count=1 ./modulos/orquesta-mcp`.

- [x] PASO A4: veredicto en `director/stats`.
  Ficheros: `modulos/orquesta-mcp/director_stats_*` (localizar con rg).
  Mismo contrato. Verifica: `go test -count=1 ./modulos/orquesta-mcp`.

- [ ] PASO A5: el supervisor residente consulta el veredicto antes de
  decidir rework/timeout. Ficheros: buscar
  `rg -ln "RunSupervisorGoalFirstResident" modulos cmd` y el punto donde
  decide rework por bloqueo recuperable (referencia:
  `goal_first_resident_rework_v0.go` en el stack). Si el veredicto dice
  process-dead-state-stale, el supervisor trata el goal como terminal
  reconciliable (no espera infinita). Verifica: focal del stack.

## ETAPA B - Actuador de proceso para runs/control (F2)

Contexto: `enrichRunControlGoalBackendResultV0` en
`modulos/orquesta-mcp/run_control_tool_executor_v0.go` detecta
`control_not_propagated_to_goal_backend` pero no puede actuar. La
maquinaria de parada real existe en el contrato de shutdown:
`ActiveShutdownWorkCleanerPortV0` e `IdentityPortV0` en
`modulos/orquesta-server-shutdown/contracts_v0.go`, implementados por el
backend tmux y cableados en
`cmd/orquesta-server/codex_goal_active_shutdown_work_v0.go`.

- [ ] PASO B1: puerto opcional de escalada en el executor de run control.
  Anade al executor MCP un campo opcional `BackendStopEscalator` (interfaz
  nueva pequena en orquesta-mcp con un metodo que recibe ctx + run/goal
  refs y devuelve resultado tipado stopped/residual/error). Cuando el
  control detecta no-propagacion Y `input.Forced`, invoca el escalador si
  esta inyectado; solo publica `status=stopped` si el escalador confirma;
  si no, conserva el error actual. Tests con escalador fake (confirma,
  residual, error). Verifica: `go test -count=1 ./modulos/orquesta-mcp`.

- [ ] PASO B2: implementar el escalador real sobre los puertos de shutdown
  y cablearlo. Ficheros: adaptador nuevo en el stack o en
  `cmd/orquesta-server` (junto a `codex_goal_active_shutdown_work_v0.go`)
  que envuelva el `ActiveShutdownWorkCleanerPortV0` del backend goal
  configurado, filtrando por identidad (run_ref/work_ref) antes de limpiar.
  Wiring + test focal de que el executor MCP recibe el escalador real.
  Verifica: focal de `cmd/orquesta-server` con
  `-run 'TestRunControl|TestServerGoalWorkPorts'` y del stack.

## ETAPA C - Preparacion del piloto (sin cuota Codex)

- [ ] PASO C1: config canonica local minima. Crear `orquesta.config.json`
  en la raiz SOLO si no existe, con el minimo que el server valida (mirar
  `rg -n "orquesta.config.json" cmd/orquesta-server --type go | head` y el
  test de config canonica para el schema). Sin Telegram, sin remoto.
  Verifica: focal de config en cmd. Si el schema exige secretos que no
  tienes, deja el paso anotado en la bitacora y sigue.

- [ ] PASO C2: backlog acotado del piloto. Crear
  `docs/backlog_piloto_autonomia_2026-07-10.md` con 3 tareas PEQUENAS de
  este mismo roadmap (por ejemplo: A3 si quedo pendiente, un test de
  contrato faltante, una correccion de la bitacora de desviaciones), en el
  formato que el parser honra: `Objetivo:`, `Estado:`, `Alcance:` (=
  write-set, primer scope un directorio docs/ para el resultado durable),
  `Criterios:`, `Tests:`. OJO: el parser IGNORA "Write-set previsto:".
  No se lanza nada en este paso; solo queda listo.

## ETAPA D - Con cuota Codex recuperada (NO antes)

- [ ] PASO D1: smoke de confirmacion desde la rama principal:
  `ORQUESTA_KEEP_SMOKE_DIR=1 ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1
  ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1
  bash scripts/smoke_goal_first_app_server_real.sh` (en background, rc!=0
  es fallo). Verde = goal complete + closure accepted + sin procesos.
- [ ] PASO D2: piloto de autoprogramacion supervisado con el backlog de C2
  (receta completa: `docs/bitacora_correccion_pericial_2026-07-03.md`,
  seccion "Receta completa"; MAX_REQUESTS=1; endpoints status/observe son
  SOLO POST con body `{}`). El revisor (Claude) valida cada cierre
  ejecutando los tests declarados: 208H sigue abierto y NO se acepta un
  cierre sin reejecutar sus tests.
- [ ] PASO D3: `scripts/orquesta_test_batches.sh` dos pases verdes.
- App del operador: pendiente de que el operador describa la app; se lanza
  tras D1 por el flujo Nueva App.

## Trampas conocidas (leidas de incidentes reales)

- tmux: `display-message -t "=sesion"` sin `:` devuelve formatos vacios en
  tmux 3.6 (causa del bug del lease). El codigo ya esta corregido; no
  "simplifiques" selectores tmux.
- Los goals escriben su resultado durable bajo el PRIMER scope directorio
  del write-set: pon `docs/...` primero para no ensuciar `scripts/`.
- `checkpoint_started_*`/`orquesta_goal_result_*` NUNCA se commitean como
  fuente (hay 61 versionados historicos pendientes de auditoria; no anadas
  mas).
- `GOTMPDIR`/`GOCACHE` heredados pueden ser de solo lectura en sandbox:
  si un build falla raro, usa `scripts/lib/isolated_test_env.sh`
  (`source` + `orquesta_use_isolated_test_env <dir>`).
- Los tests de `cmd/orquesta-server` completos son pesados: usa `-run` con
  patrones focales y `-timeout` explicito.

## Bitacora de desviaciones

(Anotar aqui, con fecha y paso, todo lo que no cuadre con la guia.)

- 2026-07-10, PASO A1: el executor de observe_goal YA tenia `EstadoVivoSource`
  inyectable y un camino estado-vivo en la ruta de snapshot parcial
  (`observe_app_director_goal_estado_vivo_v0.go`); lo que faltaba y se anadio
  fue: campos `causal_verdict`/`causal_reason_code` en el resultado, derivar
  el veredicto tambien en la ruta principal de `Execute`, y la regla de no
  publicar `running` contradicho con `recommended_action=reconcile_goal_state`.
- 2026-07-10, PASO A1: `modulos/orquesta-mcp/shared_contracts_resource_v0_test.go`
  esta sin gofmt en HEAD (preexistente, no tocado en este paso).
