# Incidencia 208AA: el attestor sin toolchain lanza rework de codigo

Fecha: 2026-07-11
Estado: abierto
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
