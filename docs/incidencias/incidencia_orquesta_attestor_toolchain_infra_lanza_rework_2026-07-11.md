# Incidencia 208AA: el attestor sin toolchain lanza rework de codigo

Fecha: 2026-07-11
Estado: en verificacion E2E
Area: nucleo goal-first / atestacion independiente / rework

## Resumen

Un goal real de autoprogramacion termino correctamente y materializo su unico
artefacto. La atestacion independiente ejecuto el `go` allowlisted en un entorno
hermetico, pero ese binario era el bootstrap Go 1.25.5 y el modulo exige Go
1.25.11. Con red desactivada intento descargar el toolchain y fallo. Orquesta
bloqueo correctamente el cierre, pero clasifico el fallo como tests fallidos y
lanzo automaticamente un segundo Codex de rework. El agente no podia reparar la
configuracion externa y solo podia volver a gastar tokens sobre codigo ya verde.

Un segundo intento con el binario Go 1.25.11 exacto descubrio otra causa de la
misma familia: el adapter crea un `GOMODCACHE` nuevo y vacio por atestacion,
fuerza `GOPROXY=off` y no siembra un snapshot de modulos verificado. El test raiz
no pudo resolver `golang.org/x/text v0.38.0`, aunque ya estaba instalado en la
cache local del operador. La atestacion hermetica actual solo es funcional para
fixtures sin dependencias externas o con una cache sembrada fuera del contrato.

## Evidencia retenida

- Run: `autoprog-core-causal-authority-guard-20260711-b`.
- Goal inicial: `goal-ref-task-autoprogramming-e3403e1a9910-g01`.
- Goal de rework inutil:
  `goal-ref-task-autoprogramming-e3403e1a9910-g01-rework-1`.
- Estado aislado:
  `/tmp/orquesta-self-core-guard-runtime/state/orchestration-state/`.
- Recibos de atestacion: ambos `status=failed`, hashes antes/despues iguales y
  `exit_code=1`.
- Logs:
  `/tmp/orquesta-self-core-guard-runtime/attestation/evidence/commands/`.
- Error literal: `go: download go1.25.11 for linux/amd64: toolchain not available`.
- El mismo diff pasa fuera del agente:
  `go test -count=1 . -run TestCausalVerdictAuthorityCallersRemainExplicitV0`
  y `go test -count=1 ./modulos/orquesta-estado-vivo`.
- `POST /api/v0/autoprogramming/goal/observe` publico goal `complete`, closure
  `blocked`, `autoprogramming_observe_goal_state_invalid`; el marcador durable
  ya apuntaba al rework `running`.
- El shutdown cooperativo elimino servidor, tmux y procesos del runtime
  aislado. No se toco OPES, `uso-app` ni ningun servicio externo.
- Segundo runtime retenido:
  `/tmp/orquesta-self-core-guard-runtime-accepted/`. Su log de atestacion
  conserva `module lookup disabled by GOPROXY=off` para `golang.org/x/text`.
- Tras eliminar el backend, la primera orden de control quedo
  `control_not_propagated_to_goal_backend`; observe reconcilio a `blocked` y la
  segunda orden confirmo `stopped`. El endpoint shutdown publico `ready` dos
  veces sin terminar el proceso; el PTY propietario lo cerro con SIGINT. Esta
  parte amplia evidencia el residual ya abierto 208A-D.
- El guard AST se integra por verificacion independiente aunque el run no pudo
  cerrar `accepted`; la excepcion queda ligada a esta incidencia.

## Lectura estructural

No es un fallo del test ni una razon para modificar el artefacto. Hay dos
fronteras que hoy se mezclan:

1. La composicion acepta un ejecutable allowlisted sin demostrar, con el mismo
   entorno hermetico del attestor, que puede ejecutar el toolchain requerido.
2. El cierre reduce cualquier atestacion `failed` a rework de implementacion,
   aunque el fallo proceda del runner, toolchain, cache o configuracion de
   confianza y el implementador no pueda cambiarlo.

## Criterio de cierre

1. La composicion hace preflight del comando allowlisted con el entorno
   hermetico real antes de lanzar un implementador que exige atestacion.
2. El entorno hermetico usa un snapshot/cache de dependencias content-addressed
   y de solo lectura, sembrado antes del test; conserva red desactivada y no
   hereda la cache mutable del implementador.
3. El resultado distingue al menos fallo de test de fallo de infraestructura
   del attestor con reason code durable y accionable.
4. Un fallo de infraestructura bloquea/pausa para reparacion de composicion;
   nunca consume el presupuesto de rework de codigo.
5. Tras corregir la configuracion se puede reanudar o reatestar el mismo
   artefacto sin pedir a Codex que lo reescriba.
6. Observe/status muestran el goal activo de rework o el bloqueo de
   infraestructura sin `goal_state_invalid` ambiguo.
7. Pruebas focales cubren bootstrap sin toolchain, dependencia presente en el
   snapshot y dependencia ausente. Las dos primeras cierran sin red; la ultima
   bloquea antes de lanzar o sin rework de codigo.

## Avance integrado

`97d1d913a` incorpora el corte neutral: `failure_code` durable, reason code
`goal_required_test_attestor_infrastructure_failed`, cierre bloqueado sin
`NeedsRework` para infraestructura y compatibilidad de identidad con recibos
fallidos legacy.

`350b48da7` incorpora el preflight obligatorio previo al launcher, snapshot de
modulos de solo lectura con hash verificado y el mismo entorno hermetico para
preflight y test (`GOPROXY=off`, `GOTOOLCHAIN=local`, sin entorno heredado).

El primer E2E sobre ese commit encontro una tercera causa de la misma familia:
el entorno hermetico no definia `PATH`. El `go test` congelado arrancaba, pero
los tests que invocan una herramienta declarada (`git`) por nombre fallaban con
`executable file not found in $PATH`. El implementador habia pasado porque su
entorno si tenia `PATH`. El attestor persistio un claim fallido generico, no un
recibo, y el ciclo volvio a marcar `NeedsRework=true`.

Evidencia del E2E:

- runtime: `/tmp/orquesta-208aa-e2e-runtime`;
- run: `autoprog-attestor-e2e-close-20260711`;
- goal: `goal-ref-task-autoprogramming-c202da3d0f17-g01`;
- los dos preflights y el hash del snapshot fueron validos;
- el comando exacto fallo bajo `env -i` sin `PATH` y paso con un `PATH`
  determinista limitado a los directorios de Go y Git;
- `runs/control` detuvo el rework y shutdown retiro el backend en el primer
  intento, devolvio `backend_still_running`, y cerro el servidor en el segundo.

El corte siguiente construye `PATH` exclusivamente desde los directorios de
`allowed_commands` y `git_command_path`, ordenados y sin heredar el valor del
padre. Tambien hace que cualquier error operativo devuelto por el attestor se
persista como `goal_required_test_attestor_infrastructure_failed`; replay
conserva ese codigo y no vuelve a lanzar attestor ni rework. Falta el E2E final
`accepted` para cerrar la incidencia.
