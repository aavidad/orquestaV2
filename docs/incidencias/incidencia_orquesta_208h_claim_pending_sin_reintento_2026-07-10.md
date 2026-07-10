# Incidencia: 208J claim de atestacion pendiente sin recuperacion

Fecha: 2026-07-10.
Estado: abierto, bloqueante para aceptar `BUG-ORQ-20260710-208H` y ejecutar
los pasos D1/D2 de autonomia.

## Hallazgo

La rama candidata `wip/attestation-208h-20260710` adquiere un claim durable
antes de lanzar el atestador independiente. Si el atestador devuelve error, el
claim queda en estado `pending`. En la siguiente observacion,
`AcquireGoalRequiredTestAttestationClaimV0` encuentra ese claim, devuelve
`Acquired=false` y el ciclo lo salta. No existe transicion de fallo, liberacion,
caducidad ni recuperacion del owner.

La validacion final permanece bloqueada por falta de receipt, pero el sistema
no puede volver a ejecutar el test ni publicar una causa durable de rework.
Un error transitorio, timeout o muerte del proceso de atestacion deja el goal
atascado indefinidamente. Esto contradice el objetivo de autonomia de 208H.

## Evidencia verificada

- `modulos/orquesta-goal/lifecycle_v0.go`: el ciclo adquiere el claim, y solo
  llama a `CompleteGoalRequiredTestAttestationClaimV0` tras recibir un receipt
  valido. Un error del atestador retorna antes de persistir estado final.
- `modulos/orquesta-state-file/goal_required_test_attestation_store_v0.go`:
  `Acquire...` conserva un claim existente con `Acquired=false`; los unicos
  estados admitidos son `pending` y `completed`.
- No hay puerto `Release`, `Fail`, `Recover` ni lease con vencimiento para ese
  claim.
- Revision independiente ejecutada sobre la rama candidata: los siete focales
  declarados pasan, la bateria aislada de seis paquetes pasa dos veces
  consecutivas (6 paquetes y 12 ejecuciones, `two_consecutive_passes_passed`)
  y `TestNeutralOrchestrationPackagesDoNotImportProductAdapters` pasa. La
  cache aislada de esa revision se elimina tras registrar este resumen. Ninguna
  de esas pruebas fuerza un error del atestador despues de adquirir el claim.

## Reproduccion minima que falta como test

1. Preparar un goal terminal con atestacion independiente requerida.
2. Hacer que el store conceda el claim y que el atestador devuelva error.
3. Observar el mismo goal otra vez.
4. Comprobar que no queda un `pending` silencioso: debe existir una transicion
   durable recuperable y una accion tipada (`retry`, `rework` o `blocked`), sin
   aceptar el cierre por resultados autodeclarados.

Tambien debe cubrirse muerte de proceso tras adquirir el claim y dos observers
concurrentes, conservando un solo atestador activo por test/revision.

## Correccion estructural requerida

Extender el contrato de claim, sin introducir filesystem ni runtime en
`orquesta-goal`, con una maquina de estados durable: `pending`, `completed` y
un estado recuperable de fallo/expiracion con owner, instante y evidencia. El
store concreto aplica CAS/lock para reclamar o recuperar solo un claim vencido;
el lifecycle persiste el fallo del atestador y devuelve una accion causal. La
recuperacion debe comprobar primero receipts inmutables existentes, nunca
borrarlos ni sustituirlos.

No es valido resolverlo borrando ficheros `pending`, reintentando en bucle ni
degradando a `required_test_results` declarados por el implementador.

## Criterio de cierre

- Tests focales de error, timeout/muerte y concurrencia que fallen con el
  contrato actual y prueben el nuevo comportamiento.
- El estado y la accion quedan durables tras recrear `orquesta-state-file`.
- Los focales de 208H y dos pases aislados por lotes vuelven a ser verdes.
- Un revisor independiente reejecuta los tests antes de integrar 208H.
