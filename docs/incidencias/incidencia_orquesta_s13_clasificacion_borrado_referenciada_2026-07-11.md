# BUG-ORQ-20260711-219: S13 clasificaba como borrables rutas catalogadas

Estado: abierto documental/operativo. Area: retencion S13. Fecha: 2026-07-11.

## Hallazgo

El manifiesto ampliado S13 marcaba 25 rutas como `candidato_borrar` aunque
estaban catalogadas por JSON. Al contrastarlo con la auditoria estricta, diez
ademas tienen conteos de referencias externas. Las otras quince no pertenecen
al ambito estricto, por lo que su ausencia de ese inventario no demuestra que
esten libres de referencias.

## Correccion documental

`docs/clasificacion_retencion_s13_2026-07-10.json` pasa esos 25 registros a
`archivar_condicionado`; conserva la referencia al manifiesto ampliado y, donde
existe, a la auditoria estricta y su conteo. La reconciliacion separa los
ambitos: 68 estricto, 103 ampliado, 61 comunes, 42 exclusivas del ampliado y 7
omitidas por este.

## Siguiente accion

Una ola de retencion gobernada debe comprobar referencias por ruta y definir un
destino trazable antes de archivar. Esta incidencia no autoriza borrar ni mover
archivos.

Evidencia: `docs/auditorias/reconciliacion_s13_retencion_ampliada_2026-07-11.json`.
