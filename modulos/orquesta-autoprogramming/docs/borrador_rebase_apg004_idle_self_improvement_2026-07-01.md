# Borrador de rebase APG-004 idle self-improvement

Estado: borrador de merge pendiente.

Motivo: el goal recibio la foto:

- `backlog_scan_ref:scan-ref-backlog-2e2bfd30581c`
- `backlog_scan_epoch:backlog-scan-epoch-7d58c5b5552a`
- `backlog_scan_doc:modulos/orquesta-autoprogramming/docs/tareas.md:line:38:sha256:55c7ad4bcc6cd71d41ec12ebdbdc6e19e54c3c0e47cfd2d3947f302398bc3582`
- `backlog_task_id_ref:task-id-ref-backlog-5279219cc308`
- `backlog_task_id_range:T263`
- `backlog_scan_reservation_ref:reservation-ref-backlog-scan-doc-merge-540fce675096`
- `backlog_scan_reservation_ref:reservation-ref-backlog-task-id-02081ab71912`

Durante este goal, el documento `modulos/orquesta-autoprogramming/docs/tareas.md`
no coincide con la huella declarada. Evidencia local observada:

- documento completo: `sha256:dae39a299e09796432a2614212217f52784b8fc53070d980d56de3e3be39d3a3`
- linea 38 actual: `sha256:1c31f9346ae29b6fd5fe0db73c19d5d3f3e36cc088d21936c589c5cf8b1a67fc`

Por tanto no se inserta `## T263` ni se consume el rango reservado. La propuesta
queda conservada como borrador hasta rebase/merge documental con la foto nueva.

Bloque Escaneo backlog nuevos propuesto:

- task_ref: `task-ref-backlog-scanner`
- titulo: `Escaneo backlog nuevos`
- refs requeridas:
  - `backlog_scan_ref:scan-ref-backlog-2e2bfd30581c`
  - `backlog_scan_epoch:backlog-scan-epoch-7d58c5b5552a`
  - `backlog_scan_doc:modulos/orquesta-autoprogramming/docs/tareas.md:line:38:sha256:55c7ad4bcc6cd71d41ec12ebdbdc6e19e54c3c0e47cfd2d3947f302398bc3582`
  - `backlog_scan_reservation_ref:reservation-ref-backlog-scan-doc-merge-540fce675096`
  - `backlog_scan_reservation_ref:reservation-ref-backlog-task-id-02081ab71912`
  - `backlog_task_id_ref:task-id-ref-backlog-5279219cc308`
  - `backlog_task_id_range:T263`

Artefactos implementados sin consumir `T263`:

- `idle_self_improvement_v0.go`
- `idle_self_improvement_v0_test.go`
- `self_improvement_apg004_test.go`
- documentacion local de contratos, pruebas y decisiones.
