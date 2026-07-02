# Borrador de rebase APG-003 idle self-improvement

Estado: borrador de merge pendiente.

Motivo: el goal recibio la foto:

- `backlog_scan_ref:scan-ref-backlog-58c1f26e2898`
- `backlog_scan_epoch:backlog-scan-epoch-68612dec3688`
- `backlog_scan_doc:modulos/orquesta-autoprogramming/docs/tareas.md:line:34:sha256:55c7ad4bcc6cd71d41ec12ebdbdc6e19e54c3c0e47cfd2d3947f302398bc3582`
- `backlog_task_id_ref:task-id-ref-backlog-1a44ad9065e5`
- `backlog_task_id_range:T263`
- `backlog_scan_reservation_ref:reservation-ref-backlog-scan-doc-merge-55b2a657b4e5`
- `backlog_scan_reservation_ref:reservation-ref-backlog-task-id-2fba20b72349`

Durante este goal, la linea 34 de
`modulos/orquesta-autoprogramming/docs/tareas.md` sigue apuntando a APG-003,
pero los hashes actuales no coinciden con la foto recibida:

- linea 34: `sha256:027b210bf82c8c45561f2fa1544cf44c01dd102a51c6ecbfe65ba893c61cc26d`
- `sha256:dae39a299e09796432a2614212217f52784b8fc53070d980d56de3e3be39d3a3`

Por tanto no se inserta `## T263` ni se consume el rango reservado. La propuesta
queda conservada como borrador hasta rebase/merge documental con la foto nueva.

Bloque Escaneo backlog nuevos propuesto:

- task_ref: `task-ref-backlog-scanner`
- titulo: `Escaneo backlog nuevos`
- refs requeridas:
  - `backlog_scan_ref:scan-ref-backlog-58c1f26e2898`
  - `backlog_scan_epoch:backlog-scan-epoch-68612dec3688`
  - `backlog_scan_doc:modulos/orquesta-autoprogramming/docs/tareas.md:line:34:sha256:55c7ad4bcc6cd71d41ec12ebdbdc6e19e54c3c0e47cfd2d3947f302398bc3582`
  - `backlog_scan_reservation_ref:reservation-ref-backlog-scan-doc-merge-55b2a657b4e5`
  - `backlog_scan_reservation_ref:reservation-ref-backlog-task-id-2fba20b72349`
  - `backlog_task_id_ref:task-id-ref-backlog-1a44ad9065e5`
  - `backlog_task_id_range:T263`

Artefactos implementados sin consumir `T263`:

- `idle_self_improvement_v0.go`
- `idle_self_improvement_v0_test.go`
- `idle_self_improvement_apg003_test.go`
- documentacion local de contratos, pruebas y decisiones.
