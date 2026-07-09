# Fallos estructurales del inventario de bugs - encargo para Codex

Fecha: 2026-07-10 (actualizado tras el cierre de sesion remota, HEAD
`43aea232b`, indice de sesion ampliado a S1-S14).
Autor: Claude (revision transversal solicitada por el operador).
Fuentes: `docs/inventario_bugs_orquesta_2026-06-30.md` (completo),
`docs/incidencias/incidencias_sesion_codex_remoto_orquesta_2026-07-10.md`
(S1-S14),
`docs/incidencias/incidencia_orquesta_goal_first_remote_checkpoint_control_2026-07-10.md`.

Estado operativo al escribir esto: local, remoto y GitHub sincronizados en
`43aea232b` sobre `trabajo/plataforma-agentes`, pero el binario Orquesta vivo
en remoto es anterior al codigo pusheado
(`binary_sha256=9541e2f0...`, ver S14). No se desplego porque hay
goals/app-server/tests antiguos vivos y la parada por `runs/control` no es
fiable (208C). Esa es exactamente la dependencia circular que este documento
ordena romper.

Proposito: este documento NO pide revisar bugs sueltos. Identifica los fallos
estructurales que explican por que el inventario sigue generando bugs nuevos
sobre los mismos ejes, y define cortes de trabajo accionables para Codex con
criterio de cierre verificable. Cada corte debe citar este documento y el
fallo `F<n>` que ataca.

Verificacion contra codigo: todas las afirmaciones sobre simbolos, patches y
comportamiento estan contrastadas con el arbol en `43aea232b` el 2026-07-10
(rg/git ls-files/lectura de los ficheros citados). Tres afirmaciones de la
primera version quedaron corregidas tras esa revision: F2 (la deteccion de
no-propagacion ya existe; el hueco es proyecciones-vs-identidad-runtime y la
falta de actuador de proceso en `runs/control`), F6 (Gemini ya tiene
normalizador tolerante; el fallo es la duplicacion por proveedor) y el
ejemplo de IDs duplicados del fallo meta.

## Lectura global

El inventario tiene ~250 filas; mas del 90% figuran `cerrado`. Pero los bugs
vivos de hoy (BUG-058, BUG-065, BUG-066 residual, BUG-075, BUG-079, BUG-165,
BUG-208 A-E, CODEX-HOME-TOKEN-INVALIDADO) caen casi todos sobre el mismo eje
que ya produjo BUG-035, BUG-056, BUG-069, BUG-071, BUG-090, BUG-096, BUG-101,
BUG-105 y BUG-131. La secuencia 065 (2026-07-01) -> 165 (2026-07-04) -> 208
(2026-07-10) es el mismo fallo estructural reproducido tres veces en diez
dias, cada vez en una superficie distinta. Conclusion: los cierres estan
funcionando como reconciliaciones puntuales anadidas en cada borde, no como
colapso de las fuentes de verdad que divergen. Mientras eso no cambie, el
inventario seguira creciendo con variantes del mismo bug.

## F1 - Fuente de verdad fragmentada del ciclo de vida goal-first

Patron: el estado de un goal vive repartido en al menos cuatro sitios que
divergen entre si: (1) estado persistido/statefile y stores, (2) artefactos de
filesystem (`checkpoint_started_*.txt`, `orquesta_goal_result_*.json`),
(3) proceso real (tmux/app-server/pids/sockets) y (4) proyecciones publicas
(`autoprogramming/status`, `observe_goal`, `director/stats`, CLI). Cada bug de
esta familia se cierra anadiendo una reconciliacion mas entre dos de esos
puntos, y el siguiente bug aparece entre otros dos.

Evidencia: BUG-035, BUG-045, BUG-056, BUG-069, BUG-071, BUG-072, BUG-078,
BUG-090, BUG-096, BUG-101, BUG-105, BUG-130, BUG-131 (que ya nombro esta raiz
textualmente), BUG-141, BUG-165, BUG-198 y BUG-208B (goal con
`orquesta_goal_result status=blocked` en disco mientras `observe` publica
`goal_status=running`; 504 parcial que mezclaba running con cierre blocked).

Que hay ya (verificado en codigo en `43aea232b`):
`ActiveShutdownWorkCleanerPortV0` en
`modulos/orquesta-server-shutdown/contracts_v0.go` (cierre de BUG-131),
`NormalizeStoppedServerSnapshotV0` en
`modulos/orquesta-server/status_process_stale_v0.go`, modulo neutral
`modulos/orquesta-estado-vivo`, y los patches focales 2026-07-10: workdir
inexistente (`workdir_unavailable` en
`orquesta-runtime-codex-appserver/codex_goal_app_server_v0.go`), rework
residente (`goal_first_resident_rework_v0_test.go` en
`orquesta-app-codex-stack`) y normalizacion del 504
(`mcpObserveAppDirectorGoalNormalizeTimeoutPartialV0` suprime cierre stale y
fuerza `recommended_action=observe_later` si el goal sigue
`running/accepted`, con test
`TestMCPAutoprogrammingObserveGoalHTTPHandlerV0TimeoutNoPublicaClosureBloqueadoSiGoalSigueRunningV0`).
Dato que apoya el patron: hay 116 ficheros Go en `modulos`/`cmd` que
mencionan reconciliacion; son reconciliaciones locales por superficie, no un
reconciliador unico.

Accion para Codex: definir UN reconciliador causal de goal-first como
contrato unico (no otra proyeccion): dado `run_ref/goal_ref`, debe leer
estado persistido + artefactos del write-set + identidad runtime viva y
emitir un veredicto unico tipado (`running_confirmed`, `terminal_by_artifact`,
`process_dead_state_stale`, `divergent_needs_repair`) que `observe`, `status`,
`director/stats` y el supervisor residente consuman en vez de recalcular cada
uno su version. Regla dura: ninguna superficie puede publicar `running` si el
reconciliador no confirma proceso vivo; ningun resultado terminal durable en
disco puede convivir con `running` publicado mas de un tick.

Criterio de cierre: repro por API en remoto del escenario BUG-208B
(resultado blocked en disco + observe) devolviendo el veredicto reconciliado,
y test de simulacion que inyecte las cuatro fuentes en desacuerdo.

## F2 - Control plane que no confirma la parada del backend real

Patron (precisado tras revisar el codigo): `runs/control stop/cancel` SI
detecta hoy la no-propagacion y la publica honestamente --
`enrichRunControlGoalBackendResultV0` en
`modulos/orquesta-mcp/run_control_tool_executor_v0.go` emite
`control_not_propagated_to_goal_backend` con `Estado=error` y
`recommended_action=observe_goal_backend_before_declaring_stopped` cuando el
backend sigue activo. El hueco estructural esta en dos sitios: (1) esa
verificacion consulta proyecciones (`director/stats` antes/despues), no la
identidad runtime real (pid/sesion tmux/socket); (2) la ruta forzada
(`reconcileGoalStateAfterForcedControlV0`) reconcilia ESTADO a terminal, pero
no tiene actuador de proceso: la maquinaria que de verdad mata el backend
(SIGTERM cooperativo, kill-session, cleaner) vive solo en el contrato de
shutdown (`ActiveShutdownWorkCleanerPortV0`) y `runs/control` no puede
invocarla. Por eso en remoto el forced dejo `goal_status_after=running`.

Evidencia: BUG-063, BUG-121/122/123 (catalogo run control), BUG-165 (Sueldos
forced stop), BUG-207 (crash ResetStdio en forced stop), BUG-208C / S5 de la
sesion remota 2026-07-10.

Que hay ya (verificado en codigo): forced stop real verde en local
(`smoke_goal_first_forced_stop_backend_real.sh`), SIGTERM cooperativo + FIFO
`stdin.pipe` en `app_server_tmux`
(`codex_goal_app_server_tmux_v0.go`), identidad
`ActiveShutdownWorkIdentityPortV0` para dedupe, y la deteccion honesta de
no-propagacion descrita arriba.

Accion para Codex: dar a `runs/control` acceso al mismo actuador de proceso
que ya usa shutdown: cuando el control detecta backend activo tras stop
(el caso `control_not_propagated_to_goal_backend`), debe poder escalar a la
parada por identidad runtime via los puertos ActiveShutdownWork ya
existentes, y solo publicar `status=stopped` tras confirmar muerte del
proceso/sesion. En remoto, el caso BUG-208C sugiere ademas que el control
apuntaba a un backend cuya identidad ya no casaba con el proceso vivo
(worktree retirado, S1): el control debe fallar con causa de identidad, no
con un generico no-propagado.

Criterio de cierre: repro/API remota de stop y forced stop sobre un goal con
backend vivo y sobre un goal con backend huerfano; en ambos, estado final
coherente entre control, observe, status y `ps`.

## F3 - Verificacion amplia contaminada por estado vivo (208E)

Patron: `go test ./...` y las tandas amplias corren en el mismo host que
servidores residentes, app-servers, sandboxes y goals anteriores; fallan por
concurrencia y ruido (`claude_goal_result_invalid`,
`gemini_goal_result_invalid`, umbrales temporales), y eso bloquea declarar
verde cualquier cierre remoto. Es tambien la causa de que casi todos los
cierres del inventario digan "cerrado local, pendiente verificacion remota":
el residual remoto nunca se puede pagar porque no hay harness limpio.

Evidencia: BUG-208E / S9, S14 (el deploy del binario nuevo quedo bloqueado
precisamente porque no hay drain fiable de los procesos vivos), BUG-172
(disco lleno por caches), BUG-196 (GOCACHE read-only), BUG-133 (test escribia
en /tmp), nota operativa de sandbox DNS en BUG-079, y la operativa local
documentada de que `go test ./...` global tumba sesiones.

Accion para Codex: un corte unico "harness remoto aislado o drain gobernado":
(a) inventario de procesos Orquesta vivos con clasificacion (protegido /
drenable), (b) comando de drain gobernado que use los puertos de shutdown ya
existentes y NUNCA toque `uso-app`, (c) perfil de test aislado (TMPDIR,
GOCACHE, CODEX_HOME, puertos) reutilizable por deploy y nightly. Verificado
en codigo: hoy solo un script de `scripts/` exporta `GOCACHE` propio
(`smoke_self_programming_composite_goal_first.sh`); no existe un perfil de
aislamiento reutilizable. No mezclar este corte con limpieza de codigo
muerto (regla S2).

Restriccion clave aprendida en S14: el drain NO puede depender de
`runs/control`, porque 208C demuestra que esa parada no es fiable; debe
operar a nivel de proceso/identidad runtime (puertos de shutdown, SIGTERM
cooperativo, kill de sesion tmux propia) con backup previo de artefactos,
como ya hace el cleaner de shutdown en local. Si el drain se disenara sobre
el control plane, quedaria bloqueado por F2 y la dependencia circular
seguiria: sin drain no hay deploy, sin deploy no llegan los fixes de control.

Criterio de cierre: tanda amplia remota verde dos veces seguidas con el
perfil aislado, con receipt JSON del drain previo.

## F4 - El write-set no gobierna todo el ciclo (scheduling, rutas, git)

Patron: el write-set se declara y se valida en algunos bordes, pero no
gobierna (1) el scheduling paralelo -- se aceptaron dos tareas con write-set
`scripts` solapado en paralelo (S7/208D), (2) la generacion de rutas durables
-- resultados colgados bajo ficheros `.go`/`.sh` (BUG-195, BUG-198-ruta) y
checkpoints escritos en el arbol fuente (S10), ni (3) la frontera con git.
Conteo verificado con `git ls-files` en `43aea232b`: los 37 ficheros de S13
son los del patron exacto `*goal-ref-task-autoprogramming*`; con el patron
amplio (`checkpoint_started_goal-ref*` / `orquesta_goal_result_goal-ref*`,
incluyendo la familia `goal-ref-autoprogramming-backlog-*`) hay 61 ficheros
versionados, repartidos en al menos 10 directorios `docs/` de modulos, `cmd`,
`scripts/docs` e incluso `docs/evals` y `docs/historico`.

Evidencia: BUG-009, BUG-036, BUG-102, BUG-164, BUG-167, BUG-195, BUG-198,
S7/S10/S13 de la sesion 2026-07-10.

Que hay ya (verificado en codigo): secuenciacion de write-sets solapados con
codigos `write_set_overlap_sequenced` y `declared_write_set_overlap_sequenced`
en `modulos/orquesta-autoprogramming/autoprogramming_partition_policy_v0.go`,
salto de scopes fichero para la ruta durable, `workspace_write_guard` en
validacion de recibos.

Accion para Codex (dos piezas):
1. Canonizar el destino de artefactos de ejecucion fuera del arbol fuente
   (directorio de evidencias/retencion por goal), y un guard que impida que
   `checkpoint_started_*`/`orquesta_goal_result_*` nuevos aparezcan bajo
   rutas versionables.
2. Auditoria gobernada de los 37 ficheros ya versionados: clasificar
   (evidencia historica citada / movible / retirable) y retirarlos con commit
   explicito y JSON de clasificacion, segun la garantia de poda vigente.

Criterio de cierre: guard en rojo si un goal materializa artefactos de
ejecucion en ruta versionable; repo sin ficheros de ejecucion nuevos tras un
smoke real; auditoria de los 37 con destino decidido.

## F5 - Deriva local/remoto sin protocolo unico verificado

Patron: el remoto ha estado operando con `origin` apuntando a un bundle
temporal (S3), con `ORQUESTA_CTL_WORKDIR` sobre un worktree retirado (S1/208A)
y con worktrees 24 commits por detras (BUG-194 nota). El deploy atomico ya
existe y se ha endurecido (BUG-201, 211, 216, 217, 218), pero el servidor no
autovalida su propia identidad de arbol/branch/binario al arrancar ni la
publica como bloqueo.

Evidencia: S1, S3, S14 (binario vivo `9541e2f0...` anterior al codigo
pusheado `43aea232b`; commit/push correcto no implica binario desplegado),
BUG-191 (goal complete sin commit_sha en GitHub), BUG-194, protocolo remoto
pendiente de validar en
`docs/runbooks/protocolo_git_remoto_orquesta_2026-07-02.md`.

Hueco verificado en codigo: `scripts/orquesta_server_ctl.sh` toma
`ORQUESTA_CTL_WORKDIR` con default `/srv/orquesta-self/worktrees/pilot-remoto-1`
(justo el worktree que el inventario documenta como stale) y solo comprueba
si existe `orquesta.config.json` dentro; no valida que el workdir exista, sea
worktree git vivo ni que no este retirado. En `cmd/orquesta-server` no hay
ninguna validacion de identidad de worktree al arrancar (la unica validacion
de workdir es la del backend app-server por goal, BUG-198). El deploy si
valida identidad (`verify_runtime_identity` en
`scripts/orquesta_server_deploy.sh` acepta `runtime_identity.binary_sha256`
anidado), pero eso solo protege el camino deploy, no un arranque por ctl
contra un workdir muerto.

Accion para Codex: startup guard de identidad de despliegue: al arrancar, el
servidor valida que su workdir existe, es worktree git valido, no esta
retirado/detached inesperado, y que el binario coincide con
`runtime_identity.binary_sha256` del receipt de deploy; si no, arranca en
modo `degraded_identity` que rechaza `prepare-run` amplio y lo publica en
status. Revisar tambien el default de `ORQUESTA_CTL_WORKDIR` en el ctl, que
hoy apunta a `pilot-remoto-1`. Ademas, fijar el remote GitHub canonico en el
servidor (o remote explicito adicional) como parte del receipt de deploy.

Criterio de cierre: repro remota: arrancar contra worktree retirado debe
producir bloqueo visible por API, no aceptar cuatro goals como el 2026-07-10.

## F6 - Contratos de cierre narrativos que siguen aceptando falsos verdes

Patron: cada vez que un contrato de cierre vive como texto/narrativa (prompt,
informe del agente, doc) en vez de como validador ejecutable unico, aparece
un falso verde: QA editorial OPES, `evidence_refs` con forma inesperada,
resultados de proveedor no normalizados. La familia OPES esta mayormente
cerrada con la terna QA ejecutable. En la frontera de proveedor, la revision
de codigo 2026-07-10 confirma que la tolerancia de BUG-170 SI existe hoy en
Claude Y en Gemini, pero como implementaciones paralelas duplicadas:
`normalizeClaudeGoalEvidenceRefsJSONValueV0` en `orquesta-runtime-claude` y
`normalizeGeminiGoalResultJSONV0`/`normalizeGeminiGoalEvidenceRefsJSONValueV0`
en `orquesta-runtime-gemini`. El fallo estructural es esa duplicacion: cada
mejora del normalizador hay que copiarla a mano por proveedor, y el proximo
proveedor (o el proximo campo con forma flexible) reabre la clase BUG-170.

Evidencia: BUG-058 (vivo), BUG-061, BUG-064, BUG-067, BUG-070, BUG-093,
BUG-095, BUG-097, BUG-108/109/110, BUG-170, fallos S9 de la tanda remota.
Nota sobre S9: los strings `claude_goal_result_invalid` y
`gemini_goal_result_invalid` no aparecen en las fuentes de
`cmd/orquesta-server` (solo como constantes en los modulos runtime), lo que
refuerza la hipotesis de que la tanda amplia remota fallo por contaminacion
de F3 (procesos vivos) y no por un hueco nuevo de contrato; confirmarlo tras
el drain, no antes.

Accion para Codex: extraer el normalizador tolerante de `orquesta_goal_result`
a una pieza neutral compartida (un solo sitio con corpus de fixtures reales
por proveedor Claude/Gemini/Codex), que los tres backends consuman, y que
degrade a `recuperable + repair_receipt` en vez de `invalid` terminal cuando
la desviacion es de forma y no de contenido. Prioridad por detras de F3:
primero confirmar con harness limpio si S9 era ruido.

Criterio de cierre: un unico normalizador con fixtures por proveedor en tests
focales; ningun `goal_result_invalid` terminal por desviacion de forma
recuperable; tanda amplia limpia sin esos codigos tras el drain.

## Fallo meta - El propio inventario como fuente de verdad degradada

El inventario mezcla tabla, avances cronologicos y notas; hay IDs duplicados
con estados contradictorios (BUG-ORQ-20260704-165 figura `abierto` y
`cerrado` en filas distintas; BUG-ORQ-20260702-121/122/123 estan reutilizados
para bugs diferentes -- 121 es a la vez "Run control/Goal backend" y
"OPES/final package topic quality" --; y existe un `BUG-ORQ-20260709-208`
local distinto del `BUG-ORQ-20260710-208` remoto, que solo se distinguen por
la fecha embebida). La "Lectura vigente" corrige a mano lo que la tabla dice. Esto ya obligo a una regla especial (prevalece la lectura
sobre filas antiguas) y hace imposible contar bugs vivos por maquina.

Accion para Codex (barata, alto retorno): no reescribir la historia; anadir
un indice generado `docs/inventario_bugs_estado_vivo.md` (o JSON) con una
fila canonica por ID vivo y su residual exacto, regenerable por script, y
regla de no reutilizar IDs. El inventario actual queda como historial.

## Orden de ataque propuesto

La situacion S14 fija el primer paso: hoy el sistema esta en un ciclo
bloqueado (no se despliega porque no hay drain fiable; los fixes de control
no llegan al binario vivo porque no se despliega). El punto de corte del
ciclo es un drain a nivel de proceso que no dependa del control plane.

1. F3 (drain gobernado + harness aislado): rompe el ciclo de S14 y
   desbloquea la verificacion de todo lo demas; sin el, ningun cierre remoto
   es demostrable.
2. F5 (deploy del binario ya pusheado + startup guard de identidad):
   inmediatamente despues del drain; corta la clase S1/S3/S14 antes de
   reanudar autoprogramacion.
3. F1 + F2 juntos (reconciliador causal + control confirmado): son el nucleo
   de BUG-208 y de la orden vigente del operador (que el director no se
   atranque); F2 depende del veredicto de F1. Revalidar por API tras el
   deploy los patches focales del 2026-07-10 que ya viajan en `43aea232b`.
4. F4 (write-set/artefactos versionados): pieza 1 con el deploy; la
   auditoria de los 37 ficheros como corte separado gobernado.
5. F6 (normalizador por proveedor): tras F3, cuando se pueda distinguir
   contrato real de ruido.
6. Fallo meta: en cualquier hueco; no bloquea.

Regla transversal: cada corte cita `F<n>`, deja test focal + repro/API, y no
mezcla frentes (en particular, nada de limpieza de codigo muerto dentro de
cortes operativos, regla S2).
