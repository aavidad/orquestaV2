# Borrador de rebase APG-005 idle self-improvement

Estado: borrador de merge pendiente.

Motivo: el goal recibio la foto:

- `backlog_scan_ref:scan-ref-backlog-368a95054517`
- `backlog_scan_epoch:backlog-scan-epoch-8dc06f56b7fc`
- `backlog_scan_doc:modulos/orquesta-autoprogramming/docs/tareas.md:line:42:sha256:55c7ad4bcc6cd71d41ec12ebdbdc6e19e54c3c0e47cfd2d3947f302398bc3582`
- `backlog_task_id_ref:task-id-ref-backlog-48cf16dc7592`
- `backlog_task_id_range:T263`
- `backlog_scan_reservation_ref:reservation-ref-backlog-scan-doc-merge-9e9c403aac4f`
- `backlog_scan_reservation_ref:reservation-ref-backlog-task-id-911abe614485`

Durante este goal, el documento `modulos/orquesta-autoprogramming/docs/tareas.md`
no coincide con la huella declarada. Evidencia local observada:

- documento completo: `sha256:dae39a299e09796432a2614212217f52784b8fc53070d980d56de3e3be39d3a3`
- linea 42 actual: `sha256:0de9f12ba7facf571283b58fd0bc5cf1e69f8e5cec3cff6079eba1982f05a575`

Por tanto no se inserta `## T263` ni se consume el rango reservado. La propuesta
queda conservada como borrador hasta rebase/merge documental con la foto nueva.

Bloque Escaneo backlog nuevos propuesto:

- task_ref: `task-ref-backlog-scanner`
- titulo: `Escaneo backlog nuevos`
- refs requeridas:
  - `backlog_scan_ref:scan-ref-backlog-368a95054517`
  - `backlog_scan_epoch:backlog-scan-epoch-8dc06f56b7fc`
  - `backlog_scan_doc:modulos/orquesta-autoprogramming/docs/tareas.md:line:42:sha256:55c7ad4bcc6cd71d41ec12ebdbdc6e19e54c3c0e47cfd2d3947f302398bc3582`
  - `backlog_scan_reservation_ref:reservation-ref-backlog-scan-doc-merge-9e9c403aac4f`
  - `backlog_scan_reservation_ref:reservation-ref-backlog-task-id-911abe614485`

Artefactos implementados sin consumir `T263`:

- `idle_self_improvement_v0.go`
- `idle_self_improvement_v0_test.go`
- `idle_self_improvement_apg005_test.go`
- documentacion local de contratos, pruebas y decisiones.
