# Incidencia: carreras del harness de shutdown

Fecha: 2026-07-11.
Estado: BUG-215, BUG-216 y BUG-217 cerrados localmente.
Alcance: tests de `cmd/orquesta-server`.

## BUG-ORQ-20260711-215

`go test -race ./cmd/orquesta-server` detecto lecturas y escrituras concurrentes
de los contadores `shutdownCalls` y `statusCalls` en dos handlers HTTP de test.
No era una carrera de produccion. Los contadores usan ahora `sync/atomic`, sin
ampliar timeouts ni cambiar el cliente.

Evidencia: diez repeticiones focales con `-race` y paquete completo verde tras
el parche.

## BUG-ORQ-20260711-216

En la misma primera bateria, el test de forced cleanup con daemon muerto
devolvio `forced_cleanup_failed` solo bajo instrumentacion. Tras
`tmux kill-session`, el proceso app-server exacto podia seguir vivo un instante
mientras su socket ya no era observable; la fase posterior exigia de nuevo el
socket y devolvia `generation_observation_transient`.

El cleanup usa ahora el PID exacto ya validado por marker, lease y `startRef`
para esa fase posterior a `kill-session`. No aumenta timeouts, no relaja la
identidad y no publica el error interno. Evidencia: focal `-race -count=50`,
cuatro variantes de stop con `-race -count=20`, y modulo app-server normal y
`-race`, todos verdes.

## BUG-ORQ-20260711-217

La simulacion determinista de fallos conserva un guard estricto de 60 segundos
en ejecucion normal, pero el race detector la llevo a 79 segundos sin fallar
ninguna asercion funcional. Dos helpers con build tags `race`/`!race` omiten
solo el guard de latencia bajo instrumentacion; la simulacion, el replay y las
aserciones siguen ejecutandose. Evidencia: normal 7.120 s y `-race` 69.264 s,
ambos verdes.
