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

## Rework local 2026-07-10

Se implemento el primer corte en `codex-launch-wave`/`codex-wave-status`, sin
tocar el core: flags opt-in de segundos sin progreso, bytes de diagnostico y
write-set; baseline de ese write-set; solicitud cooperativa durable; y un
recibo con razon `no_progress_diagnostic_budget_exhausted` y `rework_ref`.

Dos pruebas focales lo ejercen con un Codex falso: una ola que solo escribe
stderr termina en `stop_requested` con recibo/rework; otra que publica
`codex_last_message.txt` se conserva. La revision posterior confirma que el
supervisor residente goal-first ya tiene el equivalente gobernado por
`reconcileIdleSelfImprovementGoalProgressV0`: bloquea consumo creciente sin
progreso, persiste `blocked/rework`, solicita parada cooperativa y despierta el
supervisor; sus tres focales pasan. Por tanto no se debe duplicar esa politica.

El residual de esta incidencia queda acotado a `codex-launch-wave`, una utilidad
breakglass que no entra al ciclo goal-first y cuyo presupuesto se evalua al usar
`codex-wave-status`. Si se conserva para operaciones reales, debe materializar
una tarea gobernada o un observador residente; no se presenta como autonomia del
nucleo.

En el mismo intento real se lanzaron cinco olas con write-sets disjuntos. Cuatro
no entregaron artefactos y se pararon con su propio `wave_ref`; la quinta dejo
el adaptador PDF parcial que despues se completo y verifico de forma focal.
La evidencia runtime queda bajo `.orquesta-runtime/codex-waves/` (ignorada por
Git); no se versionan logs ni resultados de ejecucion.
