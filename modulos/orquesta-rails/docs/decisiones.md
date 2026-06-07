# Decisiones locales: orquesta-rails

## RAILS-D015 - Reactivacion acotada por frontera y campo

Fecha: 2026-05-27

Decision: T15 queda reconciliada como cierre documental: el servidor usa
`ORQUESTA_DETAIL_PROHIBITED_RAILS=on` por defecto en modo produccion con scope
acotado, mientras `ORQUESTA_SECURITY_MODE=programming` preserva apertura `off`.

Motivo: los ACKs cerrados de T15 ya implementaron el default, el scope y la
effective config. Mantener la seccion como pendiente relanza trabajo cerrado y
confunde deuda futura con el contrato ya probado.

Impacto: `orquesta-rails` sigue siendo el owner neutral del helper; los
consumidores solo declaran frontera/campo. Nuevos scopes requieren matriz
externa propia y no pueden usar T15 como paraguas generico.

Estado: aceptada.

## RAILS-D016 - Rails offline hasta nueva orden

Fecha: 2026-06-02

Decision: los rails reutilizables se conservan, pero quedan inoperativos hasta
nueva orden. `ORQUESTA_RAILS_MODE` normaliza cualquier valor a `offline`;
`audit`, `enforced`, overrides heredados y
`ORQUESTA_DETAIL_PROHIBITED_RAILS=on` no reactivan bloqueo ni redaccion basada
en rails blandos.
El servidor proyecta `ORQUESTA_RAILS_MODE=offline` y
`ORQUESTA_DETAIL_PROHIBITED_RAILS=off` como configuracion efectiva.

Motivo: los filtros de detalle y `pending rail` estaban parando o degradando
trabajo valido por vocabulario operativo, nombres cercanos o refs utiles. Hasta
nueva orden, esos casos deben llegar al director/agente para reparacion o
aprovechamiento, no cortar el flujo.

Impacto: no existe reactivacion por variable de entorno. La vuelta de un rail a
produccion requiere tarea futura explicita, owner claro, cambio de codigo o
configuracion canonica y prueba viva de que no corta trabajo aprovechable. Las
validaciones estructurales que una composicion necesite deben declararse fuera
del rail blando y no reutilizar estas listas como veto automatico.

Estado: aceptada hasta nueva orden.

## RAILS-D017 - Redaccion no bloqueante separada de rails blandos

Fecha: 2026-06-04

Decision: `RedactOperationalTextForFieldV0` queda separada de
`RailsEnforcedV0()`. Los detectores y bloqueos de detalle siguen offline, pero
la redaccion puede sanitizar valores sensibles efectivos antes de publicar
diagnosticos, summaries o artefactos de salida.

Motivo: apagar rails blandos no debe implicar filtrar tokens, credenciales,
bearer, DSN con password o material privado. A la vez, la redaccion no puede
usar marcadores genericos como `prompt`, `payload`, HOME, provider/model o
vocabulario operativo para ocultar trabajo aprovechable.

Impacto: la redaccion devuelve texto seguro y `changed=true` como evidencia de
proyeccion; no decide bloqueo, fallo, descarte ni parada de agentes. La
deteccion de rails blandos conserva la politica offline vigente.

Estado: aceptada.
