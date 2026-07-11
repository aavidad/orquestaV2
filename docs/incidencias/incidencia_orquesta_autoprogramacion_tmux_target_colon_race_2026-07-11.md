# Incidencia 208AD: smoke local de autoprogramacion pierde el backend Codex tras un recibo tmux invalido

Fecha: 2026-07-11
ID: BUG-ORQ-20260711-208AD
Estado: abierto, en investigacion
Area: autoprogramacion goal-first / runtime Codex app-server / tmux

## Resumen

El smoke local de autoprogramacion acepto el goal
`goal-ref-task-autoprogramming-f66d290bbc33-g01` durante `prepare-run`, pero
su `launch_receipt` quedo `invalid` con el issue
`codex_app_server_tmux_has_session_failed`. La observacion posterior lo dejo
`blocked` con `goal_backend_gone_without_result`; no existe resultado de
trabajo del backend que permita cerrar el run.

Esto reabre una observacion operativa de tmux distinta del cierre local de
208Y: no hay todavia causa confirmada, cambio de codigo ni prueba de regresion
que expliquen este caso.

## Evidencia retenida

Ruta retenida: `/tmp/orquesta-cleanup-goal-20260711`.

- Request/run: `autoprog-cleanup-deadcode-appserver-20260711`.
- Goal aceptado: `goal-ref-task-autoprogramming-f66d290bbc33-g01`.
- Estado persistido:
  `state-live/orchestration-state/app_director_goal_states/43f3546f57d5500cb881614cb66321e672a575decaa67d8f5562fed90bbdb0c3.json`.
- El `launch_receipt` persistido declara `status=invalid` e incluye
  `codex_app_server_tmux_has_session_failed`.
- El resultado persistido y `observe-1.json` declaran `status=blocked` y
  `summary=goal_backend_gone_without_result`; la API recomienda `replan`.

La respuesta de `prepare-run` acepto el goal antes de esa transicion. El
recibo de lanzamiento invalido y el bloqueo posterior son los hechos que esta
incidencia conserva; no prueban por si solos que tmux haya creado, perdido o
seleccionado una sesion concreta.

## Hipotesis verificable

Durante una carrera, tmux puede variar el mensaje `can't find session` al
consultar un target exacto que contiene `:`. Esa variacion podria estar siendo
clasificada como `codex_app_server_tmux_has_session_failed` en vez de una
ausencia transitoria recuperable.

Es una hipotesis, no una causa cerrada ni una propuesta de solucion. Tambien
deben descartarse una sesion realmente ausente, un target distinto al esperado,
una carrera de lifecycle ajena a `:`, o una clasificacion incorrecta posterior
al comando tmux.

## Siguiente investigacion

1. Reproducir de forma aislada la carrera con tmux real y un target exacto que
   contenga `:`, conservando argv, target, codigo de salida y stdout/stderr
   crudos de cada consulta.
2. Correlacionar esa secuencia con generation/lease, owner state y la
   transicion del receipt para distinguir ausencia real de clasificacion
   transitoria.
3. Solo si el repro demuestra el contrato, anadir prueba de regresion y un
   cambio acotado; no relajar errores tmux no verificados.

## Criterio de cierre

- Hay una causa demostrada por repro o por evidencia equivalente, no inferida
  del texto del error.
- Existe commit y prueba que cubran la variacion concreta sin aceptar errores
  tmux ajenos.
- El smoke local vuelve a completar el ciclo del goal con receipt valido y
  resultado terminal verificable, sin `goal_backend_gone_without_result`.
