# Incidencia: ola Codex sin progreso y diagnostico excesivo

Fecha: 2026-07-10

## Hecho observado

Dos olas locales, lanzadas por `orquesta-server codex-launch-wave` con
write-sets disjuntos y sin crear aplicaciones temporales, no materializaron
cambios ni `codex_last_message.txt` durante aproximadamente 98 segundos:

- `wave-s13-code-guard-20260710` (effort `medium`): 969811 bytes de stderr.
- `wave-s13-audit-20260710` (effort `low`): 85171 bytes de stderr.

Las dos fueron paradas mediante `codex-wave-stop --force` con la confirmacion
de su propia `wave_ref`. El estado posterior fue `stopped`, sin procesos
`codex exec` de esas olas ni cambios en el arbol de trabajo.

## Impacto

El control de proceso funciono: el runtime fue observable, la parada quedo
confirmada y no hubo limpieza destructiva. El problema es de autonomia y
economia: el supervisor no tenia un presupuesto de diagnostico/progreso que
detuviera o replanificara una ola que consume salida interna sin artefacto,
ACK ni resultado compacto.

No se interpreta como cierre de S13. La tarea de guard de artefactos y la
auditoria quedaron sin entrega y se replanifican desde evidencia local.

## Hipotesis estructural

La politica actual solo limita la lectura publica de logs y permite observar
`stderr_bytes`; no convierte la ausencia sostenida de progreso observable en
una transicion causal de `running` a `blocked/rework`. Por tanto el operador
todavia debe decidir manualmente cuando cortar una ola que aparenta estar viva
pero no produce una entrega verificable.

## Siguiente correccion

Anadir al plano de composicion un presupuesto opt-in para olas Codex que mida
tiempo, crecimiento de diagnostico y al menos una senal de progreso durable
(ACK, mensaje final o cambio dentro del write-set). Al agotarse, debe emitir
un recibo durable `no_progress_diagnostic_budget_exhausted`, solicitar parada
cooperativa y preparar rework; el core no debe conocer logs ni procesos.

La prueba de cierre debe lanzar un comando controlado que solo produzca stderr
sin escribir entrega, comprobar la transicion y verificar que una ola con ACK
o resultado no se corta por el mismo umbral.
