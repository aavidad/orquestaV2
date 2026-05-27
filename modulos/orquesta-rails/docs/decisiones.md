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
