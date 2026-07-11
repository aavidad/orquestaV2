# Incidencia local 2026-07-11: preflight, consumo y scope en autoprogramacion

Estado: `BUG-ORQ-20260711-221/222/223/224/225` cerrados localmente;
`BUG-ORQ-20260711-226` abierto.
Alcance: solo Orquesta local. No se toco remoto, OPES ni produccion.

## Contexto

Se uso el binario local de Orquesta en `91c3dbff0` para encargar por
`POST /api/v0/autoprogramming/prepare-run` una tarea real de limpieza de
configuracion. El estado y runtime aislados quedan retenidos bajo
`/tmp/orquesta-self-hermes-20260711` mientras esta incidencia sea util.

## BUG-ORQ-20260711-221: `run` publicaba readiness sin backend Codex valido

El primer servidor arranco con
`ORQUESTA_CODEX_COMMAND=/usr/local/bin/codex`, ruta absoluta inexistente.
Publico readiness verde y acepto `run-hermes-config-20260711-001`, pero el
backend termino `invalid` con `codex_app_server_tmux_socket_timeout`. El log
causal contiene `No such file or directory`.

Causa: `startServerCommandV0` ejecutaba `validateCodexCommandAvailableV0`,
pero `runServerCommandV0` no. El timeout de socket ocultaba un error de
configuracion demostrable antes de abrir el listener.

Cierre local: `run` usa el mismo preflight despues de resolver identidad. El
orden conserva `degraded_identity`, que no necesita construir runtime Codex.
Pruebas focales: `go test -count=1 ./cmd/orquesta-server -run CodexCommand`.

## BUG-ORQ-20260711-222: goal estrecho consumio mas de 1,1 M tokens sin cierre

El run `run-run-codex-preflight-20260711-001`, goal externo
`019f4fa2-e2d1-7e62-9f4c-b3be100a7833`, materializo un diff de 65 lineas y
paso el test focal, pero no escribio result durable. El contador vivo alcanzo
al menos `1.194.800` tokens. El observador publico
`checkpoint_only_high_consumption` y
`goal-observer-high-consumption-stop-requested`, pero el app-server/tmux
continuo vivo hasta un `POST /api/v0/runs/control` forzado. Ese control si
confirmo `goal_status_after=blocked`, `goal_control_signal_confirmed=true` y
elimino todos los procesos del goal.

Cierre local: `GoalCooperativeStopRequestV0` declara de forma tipada
`RequireConfirmedBackendStop`. Las dos rutas de alto consumo lo activan y el
adaptador de composicion invoca el executor `runs/control` forzado ya existente,
que gobierna escalador, reobservacion, reconciliacion e idempotencia. Solo
publica `Requested=true` con estado final stopped y señal de backend confirmada;
la via cooperativa no marcada conserva su comportamiento.

Segundo intento controlado: desde el binario `3d2fbf693`, el run
`run-bug222-confirmed-stop-20260711-001` y goal externo
`019f4fb4-99f7-7632-b2c7-abc209bc4bd2` consumieron `300.297` tokens sin tocar
ningun fichero. El supervisor lo corto por `runs/control` forzado; el resultado
confirmo `goal_status_after=blocked` y `goal_control_signal_confirmed=true`.
No se reutiliza ese goal ni se declara avance de codigo.

## BUG-ORQ-20260711-226: autoprogramacion sin progreso material temprano

El segundo intento de `222` consumio `300.297` tokens sin tocar ficheros. La
parada confirmada evita que el proceso quede vivo, pero no corrige por si sola
la ineficiencia anterior al umbral. Este hallazgo queda separado de `222` para
no confundir control seguro con productividad.

Pendiente: exigir progreso material verificable por tramo (diff dentro del
write-set, test nuevo o resultado durable), contabilizar lecturas/reintentos y
replanificar con contexto mas estrecho antes de alcanzar el limite duro. No
usar un simple checkpoint de inicio como evidencia de progreso util.

Diseño para el corte posterior, sujeto a prueba empirica: checkpoint tipado con
secuencia, tokens acumulados, revision de contexto y una clase material
`diff|test|result|receipt|none`. Solo renuevan el tramo un diff nuevo verificado
contra baseline/write-set, un test durable, un result o un receipt causal. Un
warning temprano debe preceder al replan; el replan crea un Goal causal nuevo,
reduce contexto y conserva artefactos/baseline. Si sigue sin progreso, aplica
el hard stop confirmado de `222`. No inferir avance desde summary, nombres de
fichero, heartbeat ni refs que solo contengan la palabra checkpoint.

Avance de nucleo 2026-07-11:

- `ab1896d68` define en `orquesta-autoprogramming` la decision pura por tramo,
  con clases `diff|test|result|receipt|none` y escalado
  `warning -> replan_required -> hard_stop_required`. Conserva la historia de
  evidencias para impedir replay `A -> B -> A` y un cambio de revision exige
  tramo causal nuevo.
- `5e5ae3c2e` añade a `orquesta-goal` uso observado neutral y tipado por goal:
  tokens acumulados, runtime, fecha, fuente y evidencias. El valor cero no
  cambia el JSON historico.
- `7aee819f4` define documento/puerto CAS de progreso con identidad compuesta,
  baseline, hash de write-set, checkpoint y claves de accion deterministas.
- `a1dd2daad` persiste ese estado con CAS y escritura JSON atomica en el unico
  `orquesta-state-file`; cubre reinicio, replay, concurrencia y contrato
  inmutable.
- `f98c41228` transporta el uso acumulado por goal desde app-server sin
  mezclarlo con el total agregado del run.
- `4a56cc212` limita los replans causales; una recurrencia sin progreso escala
  a parada dura en vez de crear una cadena ilimitada.
- `0018b0eef` ejecuta warning, replan y hard stop desde el servidor residente,
  con parada confirmada, idempotencia y fallback al comportamiento anterior si
  faltan los nuevos puertos.
- `f6ba9fea5` mueve el puerto de evidencia al modulo puro, verifica el diff
  contra baseline y write-set desde el adaptador Codex y cablea clasificador y
  store CAS en la composicion real. Los cambios fuera del write-set no renuevan
  el presupuesto.
- `93b3715c8` separa el puerto de lectura del puerto CAS para que las
  proyecciones dependan solo del contrato minimo.
- `2b31e156f` hace que MCP, HTTP y `runs/control` proyecten la decision durable;
  si existe estado valido ya no recalculan consumo ni infieren checkpoints por
  nombres. Ausencia real conserva compatibilidad; error o estado invalido
  bloquea la inferencia y publica diagnostico. El error `not found` es tipado,
  no se reconoce por texto.
- `cb87c6108` completa la clase `test`: solo renueva progreso desde una
  `LastClosure` aceptada y persistida que cubra todos los tests requeridos con
  verificaciones `verified` e `independent`, principals y credenciales
  distintos y trust policy coincidente. `RequiredTestResults` autodeclarados
  por el implementador se ignoran expresamente.

Hallazgo estructural durante la integracion: `autoprogramming/status` tomaba
`UsageSummary.TotalTokens` agregado del run, mientras el app-server dispone de
`thread/goal/get.tokensUsed` por goal y solo lo reducia a `Summary` al superar
un umbral. No se pueden sumar ni intercambiar ambos contadores. La politica
nueva consumira exclusivamente la observacion tipada por goal; MCP quedara como
proyeccion de la decision persistida y no parseara summaries ni nombres.

Pendiente para cerrar `226`: ejecutar una prueba empirica con un goal real,
acotado y util. El revisor propone T9104 del backlog de piloto, con
`MAX_REQUESTS=1`, write-set documental estrecho y consentimiento previo del
operador para el gasto. Las suites
completas de `autoprogramming`, `state-file`, `mcp`, `app-gateway`, `server`,
`app-codex-stack` y `cmd/orquesta-server` pasan tras el cableado, pero esa
evidencia offline aun no cierra el bug operativo.

Incidencia de delegacion retenida: el primer worker Terra termino sin editar
porque el proveedor devolvio `model at capacity`; se cerro esa instancia y el
mismo write-set se relanzo con Sol `high`, que entrego el corte MCP y sus tests.
No se atribuye avance a la ejecucion fallida ni se cambia la politica de modelo
por este bloqueo externo puntual.

## BUG-ORQ-20260711-223: `observe` manual dio 500 durante observacion residente

Mientras el goal anterior estaba `running`, el observador residente produjo
ticks `ok` con una observacion. En paralelo, un
`POST /api/v0/autoprogramming/goal/observe` para el mismo `run_ref` devolvio
HTTP 500 con `autoprogramming_observe_goal_error`. No se perdio el goal, pero
la superficie publica no distinguio contencion/reintento de un fallo interno.

Cierre local: `context.DeadlineExceeded` y `context.Canceled` devueltos por el
executor se proyectan como HTTP 504 tipado, parcial y recuperable, con
`recommended_action=observe_later`; errores reales siguen en 500 y no se
publican paths ni mensajes internos.

## BUG-ORQ-20260711-224: rework escribio fuera del write-set

El rework `run-run-preflight-rework-20260711-001` declaro exclusivamente
`commands.go` y `commands_test.go`, pero modifico tambien `daemon.go`. La
integracion detecto el desvio y retiro ese cambio antes de aceptar el diff.

Cierre local: prepare-run captura el baseline antes del Goal, lo persiste en el
store JSON atomico de `orquesta-state-file` y transporta una ref Goal tipada.
La promocion reutiliza `VerifyWorktreeWriteSetV0`; ruta ajena, baseline parcial,
ausente o divergente bloquean commit/archive. El store sobrevive reinicio,
acepta replay identico y rechaza la misma ref con contenido distinto.

## BUG-ORQ-20260711-225: el perfil aislado filtraba `umask 077` a los tests

La suite amplia bajo `scripts/lib/isolated_test_env.sh` dio dos falsos rojos
de seguridad: un secreto bajo un padre deliberadamente escribible por grupo y
un store con raiz permisiva parecian seguros. Los mismos focales pasaron fuera
del perfil. La causa reproducida es que `orquesta_use_isolated_test_env` cambia
el umask del shell llamador a `077` y no lo restaura; por ello `os.Mkdir(...,
0770/0777)` dentro de los tests materializa `0700` y nunca construye el caso
inseguro que pretende verificar.

Cierre local: el perfil guarda y restaura exactamente el umask del llamador en
exito y error, sin relajar los modos `0700/0600` del setup. La prueba del
script valida ambos caminos; los dos focales de permisos pasan de nuevo dentro
del perfil aislado.

Reapertura durante la verificacion de `BUG-226`: el runner
`orquesta_test_batches.sh` imponia por separado `umask 077` antes de invocar el
perfil y nunca restauraba la mascara antes de ejecutar `go test`. Dos pasadas
reprodujeron los falsos rojos en tres pruebas de seguridad; recibo retenido en
`/tmp/orquesta-bug226-batches-20260711/receipt.json`. La correccion restaura el
umask del operador tras crear las rutas privadas y el self-test comprueba la
mascara observada por el proceso `go` hijo. Ademas, el fixture de secreto usaba
`Mkdir(0770)` como si el modo resultante ignorase el umask; ahora aplica
`Chmod(0770)` explicito antes de verificar el rechazo. La misma correccion se
aplica a los fixtures `WriteFile(0644)` de Hermes y `Mkdir(0755)` del store de
tools: un test de permisos debe fijar el modo que afirma probar. Los tres
focales pasan incluso bajo `umask 077`; el runner completo se verifica de nuevo
por dos pasadas antes de volver a cerrar `225`.

Segundo cierre local: `bash scripts/test_orquesta_test_batches.sh` queda verde;
los tres focales pasan con `umask 077`; y dos pasadas reales de
`orquesta_test_batches.sh` sobre `cmd/orquesta-server` y
`orquesta-tool-capability-file` quedan verdes. Recibo:
`/tmp/orquesta-bug225-focal-fixed-20260711/receipt.json`.

## Evidencia y cierre del corte

- Runtime fallido: `/tmp/orquesta-self-hermes-20260711/runtime`.
- Runtime del goal de implementacion: `runtime-2`; estado: `state-2`.
- Runtime del rework: `runtime-3`; estado: `state-3`.
- El control forzado de ambos goals confirmo parada y no quedaron procesos
  `orquesta-server`, `codex app-server`, `orquesta-goal-*` ni tmux del corte.
- El diff integrado queda limitado a `cmd/orquesta-server/commands.go` y
  `cmd/orquesta-server/commands_test.go`.
- `bash scripts/test_orquesta_test_batches.sh`, los focales de permisos y
  `go test -count=1 ./...` bajo el perfil aislado quedan verdes tras cerrar
  `221/225`.
- Tras cerrar `222/223/224`, una segunda `go test -count=1 ./...` bajo perfil
  aislado queda verde. Tambien pasan con `-race` los paquetes
  `orquesta-state-file`, `orquesta-app-codex-stack`, `orquesta-mcp`,
  `orquesta-server` y los focales `HighConsumption|GoalRunControlStopper` de
  `cmd/orquesta-server`.

`BUG-226` no queda cerrado por la parada confirmada de `222`: control y
eficiencia tienen criterios independientes.
