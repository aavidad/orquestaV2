# Incidencias de sesion Codex/Orquesta remoto

Fecha: 2026-07-10  
Estado: abierto  
Objetivo de la sesion: dejar Orquesta trabajando en remoto, conservar los
avances de los agentes, documentar fallos estructurales y preparar commit/push
sin tocar `uso-app` ni otros servicios productivos.

Este documento agrupa todos los fallos observados en la sesion para revision de
Claude. No sustituye a las incidencias detalladas; sirve como indice operativo
para detectar patrones estructurales.

Lectura transversal: [analisis de fallos estructurales](../analisis_fallos_estructurales_orquesta_2026-07-10.md).
Registro historico/vivo: [inventario de bugs](../inventario_bugs_orquesta_2026-06-30.md).

## Fallos observados

### S1 - Servidor remoto vivo con worktree obsoleto

El proceso remoto de Orquesta estaba vivo, pero arrancado con
`ORQUESTA_CTL_WORKDIR=/srv/orquesta-self/worktrees/orquesta-repair-checkpoint-20260709T215549Z`,
un worktree que habia sido retirado en una limpieza previa.

Estado: mitigado en runtime, no cerrado estructuralmente. Se reinicio solo
Orquesta apuntando a `/srv/orquesta-self/worktrees/orquesta`, sin tocar
`uso-app`. Falta que el servidor detecte y bloquee esta condicion por si mismo.

Relacion: `BUG-ORQ-20260710-208A`.

### S2 - Limpieza operativa confundida con limpieza de codigo muerto

La orden de limpiar se interpreto en parte como limpieza de worktrees/agentes
vivos. El operador aclaro que se referia a codigo muerto, historicos falsos,
duplicados y variables/envs, no a eliminar artefactos o workdirs de ejecucion
activa.

Estado: abierto como riesgo operativo. Antes de borrar cualquier worktree,
checkpoint, runtime o cache hay que verificar procesos, refs, manifests,
`agent_ack`, `director_decisions`, `outbox` y evidencias.

Relacion: regla de higiene en `AGENTS.md` y `BUG-ORQ-20260710-208A`.

### S3 - Remoto no estaba sincronizado con GitHub

El remoto tenia `origin=/tmp/orquesta-self.bundle`, no el remoto GitHub. Local
y GitHub estaban en `e715812f4`, mientras el remoto seguia en
`16224f637b` con cambios sin commitear generados por agentes.

Estado: abierto hasta commit/push remoto. No se debe asumir que `git pull` en
remoto trae GitHub mientras `origin` apunte al bundle temporal.

### S4 - Goal-first remoto acepto trabajo amplio y quedo inconsistente

El `POST /api/v0/autoprogramming/prepare-run` remoto acepto cuatro goals
paralelos. Varios solo materializaron `checkpoint_started`; algunos escribieron
`orquesta_goal_result` con `status=blocked`, pero la API seguia exponiendo
`goal_status=running`.

Estado: abierto. Incidencia detallada en
`docs/incidencias/incidencia_orquesta_goal_first_remote_checkpoint_control_2026-07-10.md`.

Relacion: `BUG-ORQ-20260710-208B`.

### S5 - `runs/control` no paro el backend goal-first

El intento de parar `goal-04` por `/api/v0/runs/control` devolvio
`control_not_propagated_to_goal_backend`. El reintento con `forced=true`
mantuvo `goal_status_after=running`.

Estado: abierto. La reparacion actual normaliza parte de la observacion y
secuencia write-sets, pero no cierra todavia la propagacion efectiva de
stop/cancel al backend.

Relacion: `BUG-ORQ-20260710-208C`.

### S6 - `observe_goal` devolvio 504 con snapshot parcial contradictorio

`POST /api/v0/autoprogramming/goal/observe` devolvio
`autoprogramming_observe_goal_timeout`; el parcial mezclaba
`goal_status=running` con `closure_status=blocked`.

Estado: mitigado por patch focal: en timeout parcial con goal running ya no se
publica cierre bloqueado ni replan falso. Falta desplegar y probar por API en
remoto.

Relacion: `BUG-ORQ-20260710-208B`.

### S7 - Batch paralelo acepto write-sets solapados

La request tenia al menos dos tareas con write-set `scripts`. Orquesta las
lanzo en paralelo en vez de secuenciarlas, rechazarlas o pedir replan.

Estado: mitigado por patch focal en `orquesta-autoprogramming`: los write-sets
declarados solapados se secuencian. Falta desplegar y demostrar con repro/API.

Relacion: `BUG-ORQ-20260710-208D`.

### S8 - Agentes remotos fallaron aplicando parches por contexto stale

El app-server registro errores de `apply_patch` porque esperaba lineas antiguas
en:

- `scripts/smoke_opes_domain_work_real.sh`
- `scripts/orquesta_server_deploy.sh`

Estado: abierto como sintoma de concurrencia/write-set/contexto stale. Parte
del codigo resultante es recuperable y ya esta en el worktree remoto, pero el
backend no cerro el goal correctamente.

Relacion: `BUG-ORQ-20260710-208`.

### S9 - Prueba amplia remota falla con procesos residentes vivos

La verificacion amplia remota fallo bajo concurrencia:

- `cmd/orquesta-server`: `claude_goal_result_invalid`
- `cmd/orquesta-server`: `gemini_goal_result_invalid`
- `cmd/orquesta-server`: child de compilacion terminado en flaky harness
- `modulos/orquesta-app-codex-stack`: umbral temporal excedido en
  `TestSimulacionDeterministaFallosGoalFirstV0`

Paquetes focales si pasaron:

- `modulos/orquesta-autoprogramming`
- `modulos/orquesta-mcp`
- `modulos/orquesta-runtime-codex-appserver`
- `modulos/orquesta-app-codex-stack` focal
- `modulos/orquesta-web`

Estado: abierto. No se puede cerrar el frente remoto hasta tener drain/cleanup
gobernado o harness aislado.

Relacion: `BUG-ORQ-20260710-208E`.

### S10 - Artefactos generados quedaron en el worktree

Quedaron un checkpoint y un resultado generated bajo `scripts/`:

- `scripts/checkpoint_started_goal-ref-task-autoprogramming-f0ad7bb152dd-g04.txt`
- `scripts/docs/orquesta_goal_result_goal-ref-task-autoprogramming-f0ad7bb152dd-g04.json`

Estado: mitigado. Se movieron a backup remoto:
`/srv/orquesta-self/backups/active-goals-20260710T-doc-before-commit/generated-artifacts/`.
No deben commitearse como fuente.

### S11 - Telegram/Hermes remoto no es fiable para control operativo aun

Durante los cortes anteriores Telegram/Hermes reporto fallos de auth de Codex
(`Codex auth is missing access_token`) y el bot no daba estado util del
programador. En esta sesion se decidio aplazar Telegram/Hermes hasta cerrar el
nucleo/control de Orquesta.

Estado: aplazado, no cerrado. Debe quedar detras de goal-first/control/observe.

### S12 - Limpieza de variables/codigo muerto sigue incompleta como corte remoto

Hay avances en consolidacion de envs y scripts, pero el remoto aun necesita un
corte gobernado para migrar perfil remoto, secretos, defaults y codigo muerto.
No debe mezclarse con parada de agentes vivos ni con OPES productivo.

Estado: abierto. Alimentado por:

- `docs/auditoria_envs_pisadas_2026-07-04.md`
- `docs/auditoria_codigo_muerto_duplicado_2026-07-04.md`
- `docs/instrucciones_director_codex_2026-07-09.md`

### S13 - Artefactos de ejecucion ya versionados en el repo

Tras el push se comprobo que el repo remoto contiene 37 ficheros ya versionados
con patrones de ejecucion:

- `checkpoint_started_goal-ref-task-autoprogramming-*.txt`
- `orquesta_goal_result_goal-ref-task-autoprogramming-*.json`

Afectan a rutas como:

- `cmd/orquesta-server/docs/`
- `modulos/orquesta-web/docs/`
- `modulos/orquesta-autoprogramming/docs/`
- `modulos/orquesta-operator-telegram/docs/`
- `scripts/docs/`

Estado: abierto. No se borran en caliente porque pueden estar citados por
incidencias, commits o evidencias de agentes. Necesitan una auditoria gobernada:
clasificar si son evidencia historica valida, moverlos a una carpeta de
evidencias/retencion o retirarlos del arbol fuente con commit explicito.

Relacion: limpieza de historicos falsos y `BUG-ORQ-20260710-208E`.

### S14 - Codigo pusheado pero binario remoto vivo no desplegado

Local, GitHub y remoto quedaron sincronizados en `f379e4b95`, pero la API
`/api/status` sigue reportando el proceso vivo
`/srv/orquesta-self/runtime/orquesta-server-claude` arrancado antes del commit,
con `binary_sha256=9541e2f0019819e58577e8c2fcbc202e602f636321e3a66f42ad618e0a876b47`.

Estado: abierto. No se hizo despliegue/restart del binario nuevo en este corte
porque habia goals/app-server/go tests antiguos vivos y el propio bug 208
demuestra que `runs/control` no propaga parada fiable al backend. Antes de
reanudar autoprogramacion hay que hacer drain/backup/cleanup gobernado solo de
Orquesta, desplegar el binario nuevo y revalidar API. No tocar `uso-app`.

Relacion: `BUG-ORQ-20260710-208C` y `208E`.

Actualizacion 01:34 UTC: ya no estaban visibles las generaciones app-server
que habian coexistido durante la tanda. Este dato reduce el riesgo inmediato de
interferir con esos procesos concretos, pero no cierra S14: no se verifico en
este corte que su desaparicion procediera de un drain gobernado, ni que el
binario Orquesta vivo se hubiera actualizado. Antes de desplegar o relanzar hay
que reobservar procesos, identidad del binario, state y receipts.

### S15 - Timeout parcial coexistio con estado durable `invalid`

Una observacion HTTP nueva devolvio un resultado parcial por timeout y, en la
misma ventana, `goal_status=invalid`; el estado `invalid` persistio. Son dos
hechos de capas distintas: el timeout caracteriza el intento de observacion y
`invalid` el ultimo estado goal-first durable. No debe normalizarse uno como si
fuera el otro ni inferirse `running` a partir del timeout.

Estado: abierto. El material durable queda aprovechable como diagnostico, no
como cierre aceptado. Falta correlacion causal por refs/timestamps entre la
respuesta HTTP, el store y la generacion backend. La causa exacta no esta
demostrada; se registra como subfallo `BUG-ORQ-20260710-208I`.

### S16 - Todas las generaciones app-server desaparecieron a las 01:34 UTC

A las 01:34 UTC dejaron de estar visibles todas las generaciones app-server
que se venian observando. El hecho temporal esta confirmado por la observacion
operativa; no hay receipt en estos tres documentos que demuestre si fue salida
cooperativa, limpieza externa, crash o reemplazo generacional.

Estado: cambio de estado observado, causa pendiente. No equivale a despliegue,
shutdown gobernado ni cierre de los goals. La siguiente accion segura es
capturar el estado vivo y la identidad del binario antes de relanzar, limpiar o
atribuir la desaparicion a una reparacion. Relacion: `BUG-ORQ-20260710-208C`,
`208E` y `208I`.

## Patrones estructurales detectados

- Varias fuentes de verdad para un goal: estado persistido, checkpoint/result
  en filesystem y proceso real backend.
- Control plane local puede escribir `stop_requested` sin detener backend.
- Observacion parcial mezcla estado vivo con cierre bloqueado.
- Paralelizacion no siempre respeta write-sets declarados.
- Limpieza operativa sin guardas puede borrar rutas que siguen referenciadas.
- Verificaciones amplias no estan aisladas de procesos residentes.
- Remoto puede quedar fuera de GitHub por remoto Git configurado a bundle.
- Artefactos de ejecucion pueden acabar versionados como si fueran fuente.
- Un commit/push correcto no implica que el binario remoto vivo este
  desplegado.

## Estado de mitigaciones ya hechas

- Reinicio gobernado solo de Orquesta apuntando al worktree canonico.
- Backup de artefactos generados antes de retirarlos del worktree.
- Documentacion detallada de `BUG-ORQ-20260710-208`.
- Patch focal para workdir inexistente en app-server.
- Patch focal para rework residente ante bloqueos operativos recuperables.
- Patch focal para secuenciar write-sets declarados solapados.
- Patch focal para no publicar cierre `blocked` desde un timeout parcial con
  `goal_status=running`.
- Tests focales de esos patches en verde.

## Pendiente inmediato

1. Capturar estado posterior a las 01:34 UTC: procesos propios, identidad del
   binario, goal states y receipts; no atribuir causa sin evidencia.
2. Conservar y correlacionar el timeout parcial/`invalid` de S15 por refs y
   tiempos; `invalid` durable no es cierre aceptado.
3. Commit remoto de los cambios recuperables de agentes y parches de Codex.
4. Cambiar remoto Git del servidor a GitHub o anadir remote GitHub explicito.
5. Push de la rama `trabajo/plataforma-agentes`.
6. Hacer drain gobernado solo si la reobservacion encuentra procesos Orquesta
   propios; no tocar `uso-app`.
7. Desplegar el binario Orquesta remoto y verificar su identidad.
8. Repro/API de `prepare-run`, `observe` y `runs/control` con write-sets
   solapados y stop/cancel.
9. Solo despues, reactivar Telegram/Hermes y limpieza amplia de codigo muerto.

### S17 - Duplicacion generacional app-server al desaparecer tmux

Reproduccion confirmada por el operador: tmux desaparecio, el app-server siguio
vivo y `EnsureV0` rebindeo el mismo socket, dejando dos generaciones node/native
simultaneas. La implementacion parcial de generation lease conservaba seis
huecos altos: targets tmux no suficientemente ligados a identidad, señales por
PID/PGID expuestas a reuse, unlink de socket ambiguo, startup con PID cero, CAS
parcial y marker/lease globales al directorio.

Desbloqueo local completado en
`modulos/orquesta-runtime-codex-appserver`: marker/lease por socket; target tmux
exacto `=session`; identidad persistida de sesion, creación y pane PID/starttime;
CAS de todos los campos con serializacion por ruta; adopcion de restart solo con
marker exacto + proceso identificado + listener/respondiente; y limpieza stale
solo tras doble revalidacion de listener, owner e inode. Si tmux exacto falta y
el proceso identificado sigue vivo, se devuelve conflicto y no se señaliza ni
se hace unlink. Legacy ambiguo tambien queda en conflicto conservador.

Verificacion ejecutada con cache aislada fuera del repo:

- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver`: verde.
- CAS concurrente: `-count=100`, verde.
- focal target/PID-cero/CAS con `-race`: verde.
- `git diff --check -- modulos/orquesta-runtime-codex-appserver`: verde.

Limitaciones honestas: las pruebas usan listeners Unix reales, pero este
sandbox nego su creacion con `setsockopt: operation not permitted` tanto en el
TMPDIR aislado como bajo `/tmp`, de modo que esos casos quedaron `SKIP` aqui.
Ademas `.git/index.lock` es de solo lectura: no se pudo hacer add/commit/push.
No se observaron ni mutaron procesos vivos, no se abrieron goals y no se hizo
drain/deploy. Relacion: `BUG-ORQ-20260710-208J`.

Actualizacion r2 de S17: el log externo
`operator-generation-lease-test-20260710-r2.log` avanzo sin `SKIP` y dejo solo
dos defectos de test. `EnsureShutdownCleanupMigrado` no creaba el directorio
padre antes del listener. El test stale quedaba bloqueado porque el helper Unix
background heredaba las pipes stdout/stderr del comando tmux falso: el proceso
shell ya habia salido, pero `exec.Cmd.Run` no recibia EOF y nunca devolvia a
`EnsureV0`; por eso shutdown/cleanup no llegaban a ejecutarse. Se crea ahora el
padre desde el helper comun y el subproceso se lanza con stdio desacoplado desde
el inicio. No se oculto el fallo con skips, sleeps, timeout mayor, señalizacion
por PID ni unlink amplio. Verificacion local del modulo y focal `-race` verde;
los dos listeners siguen no ejecutables aqui por `EPERM`. Pendiente: r3 externa
sin `SKIP`; no hay cierre productivo, drain ni deploy.

### S18 - Cambio gobernado de remoto a local

El operador ordeno detener la programacion remota para no seguir ocupando disco
con caches y worktrees. Se interrumpieron solo las sesiones Codex/tests de
Orquesta. El shutdown HTTP del binario viejo encontro un goal durable en
`stop_requested` aunque no quedaban sesiones `orquesta-goal-*` ni procesos
app-server reales; devolvio `active_goals_present`. Tras verificar de nuevo el
ejecutable exacto `/srv/orquesta-self/runtime/orquesta-server-claude` y la
ausencia de backend, se envio `SIGINT` cooperativo solo a ese PID. Orquesta
remoto termino; no se toco `uso-app` ni ninguna otra aplicacion.

Antes de limpiar se preservaron cinco frentes como commits WIP y se trajeron a
ramas locales: main mixto, atestacion 208H, DAG de autonomia, F5 identidad y F6
decoder. Los cuatro worktrees temporales y sus ramas remotas se eliminaron solo
despues de verificar los SHA locales. `/tmp` remoto bajo del 77% al 34% y el
repo remoto quedo limpio en `6a8cb3e066`, igual que GitHub.

Estado: cerrado operativamente. El remoto queda apagado para Orquesta y no se
desplegara de nuevo hasta producir y verificar el ejecutable local.

### S19 - Revision independiente reabre la atestacion 208H

La primera implementacion de atestacion paso focales, pero la revision
independiente encontro cuatro huecos altos: no estaba activa en los specs reales
de autoprogramacion; la independencia se demostraba solo comparando strings de
agente; revision/hashes no quedaban ligados al checkout probado; y el lifecycle
podia aceptar cierre con policy activa si faltaban puertos. Tambien encontro
replay parcial no reintentable y store file-based sin exclusion multiproceso.

Estado: abierto y trasladado a reparacion local. El WIP no se integro ni se
conto como cierre de 208H.

### S20 - Revision independiente reabre el routing de modelos

La politica inicial no distinguia `complex` de `critical`, fabricaba defaults
si faltaban modelo/esfuerzo, permitia heredar `xhigh`, aceptaba builders Claude
con argumentos vacios y podia resolver el Codex global `0.128.0` en vez del
gestionado `0.144.1`. El receipt de routing Codex era opcional y no estaba
correlacionado con la decision real del launcher.

Estado: abierto y en reparacion local. `max` queda prohibido; `xhigh` debe exigir
autorizacion causal explicita y no puede proceder de env, padre, retry o hijo.
El ratchet global detecto ademas 521 lecturas `ORQUESTA_*` frente a limite 513;
no se elevara el limite para ocultar la deuda.

### S21 - Auditoria y limpieza de todas las ramas remotas

Se fetcheron todas las refs del repo del servidor antes de borrar. Habia siete
ramas: main, dos totalmente integradas y cuatro historicas divergentes con 17
commits de junio no alcanzables por main. Todas quedaron preservadas como ramas
`archive/server-*` locales y el tag de respaldo se verifico por SHA. El remoto
quedo despues con una sola rama (`trabajo/plataforma-agentes`) y un solo
worktree limpio. Una auditoria local separada clasifica ahora si alguno de los
17 commits conserva codigo aprovechable o esta supersedido. La auditoria
termino sin candidatos a cherry-pick: cuatro cambios funcionales ya estan
integrados o ampliados por series posteriores, uno tiene patch-id exacto en
main y los restantes son documentos/rails/contratos OPES historicos
incompatibles con la direccion vigente. Las refs `archive/server-*` se
conservan localmente como respaldo, pero no se mezclan con el codigo actual.

### S22 - Primer cierre integrado solo local

Se integraron localmente tres commits limpios y disjuntos: generacion unica
app-server, decoder neutral F6 y DAG durable de autonomia. La bateria conjunta
paso nueve paquetes, cuatro paquetes con `-race` y la frontera neutral. Estos
commits no se han pusheado ni desplegado; el remoto sigue en `6a8cb3e066` y
Orquesta permanece apagada.

### S23 - Contencion S13 y revalidacion local F3/F5

Se corrigio la deuda documental S13 sin borrar evidencia a ciegas. La auditoria
`docs/auditorias/s13_artefactos_ejecucion_versionados_2026-07-10.json` inventaria
68 rutas: 61 con nombre de checkpoint/resultado, 58 movibles cuando exista el
lector de recibos runtime, 5 historicas, 4 fixtures y 1 pendiente de triaje.
El script `scripts/orquesta_check_versioned_execution_artifacts.sh` compara el
arbol Git con esa auditoria y falla ante una ruta nueva no clasificada; su test
incluye un repositorio fixture con un candidato no clasificado. El prompt de
`codex-launch-wave` tambien ordena guardar recibos tecnicos solo en el runtime
del agente, nunca en el repositorio. Commits: `59d3be620`, `a268a87e9` y
`ca7a057b1`, todos pusheados a `trabajo/plataforma-agentes`.

La prueba local de dos olas S13 no produjo entrega verificable en unos 98 s y
acumulo diagnostico interno (969811 y 85171 bytes). Se pararon por
`codex-wave-stop --force`, confirmando `stopped` y sin procesos residuales. Se
registro `BUG-ORQ-20260710-208S`: falta que la composicion corte/replanifique
autonomamente por presupuesto de diagnostico/progreso; no se mezcla esa futura
pieza con core ni con F3/F5.

Revalidacion posterior, solo con harnesses sinteticos y caches bajo `/tmp`:

- `scripts/test_orquesta_server_drain.sh`: verde (`pidfd_real`, backup,
  tmux limpio, matriz fail-closed y lock exclusivo).
- `scripts/test_orquesta_server_deploy.sh`: verde.

No se ejecuto drain/deploy contra remoto, no se arranco `orquesta-server`, no
se creo una app temporal y no se toco `uso-app`. El primer intento del harness
de deploy solo evidencio que un `TMPDIR` aislado debe existir antes de
`mktemp`; al crear la raiz declarada el mismo test paso. La rama continua limpia
y sincronizada con `origin/trabajo/plataforma-agentes`.
