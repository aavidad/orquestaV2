# Borrador de rebase APG-001 idle self-improvement

Estado: borrador de merge pendiente.

Motivo: el goal recibio la foto:

- `backlog_scan_ref:scan-ref-backlog-72c73a98bd54`
- `backlog_scan_epoch:backlog-scan-epoch-feee914e7ce2`
- `backlog_scan_doc:modulos/orquesta-autoprogramming/docs/tareas.md:line:27:sha256:55c7ad4bcc6cd71d41ec12ebdbdc6e19e54c3c0e47cfd2d3947f302398bc3582`
- `backlog_task_id_ref:task-id-ref-backlog-c84160567669`
- `backlog_task_id_range:T263`
- `backlog_scan_reservation_ref:reservation-ref-backlog-scan-doc-merge-9b9f5f8061bd`
- `backlog_scan_reservation_ref:reservation-ref-backlog-task-id-4f2d7716ca0e`

Durante este goal, la linea 27 de
`modulos/orquesta-autoprogramming/docs/tareas.md` sigue apuntando a APG-001,
pero ya no corresponde al contrato idle self-improvement de la foto recibida.
Hashes actuales observados antes de conservar la propuesta como borrador:

- linea 27 `sha256:5e10e2b23158a192c811a3398a97304c1930e04f168298d3d050ee45fcecbbbe`
- `sha256:dae39a299e09796432a2614212217f52784b8fc53070d980d56de3e3be39d3a3`

Por tanto no se inserta `## T263` ni se consume el rango reservado. La propuesta
queda conservada como borrador hasta rebase/merge documental con la foto nueva.

Bloque Escaneo backlog nuevos propuesto:

- task_ref: `task-ref-backlog-scanner`
- titulo: `Escaneo backlog nuevos`
- refs requeridas:
  - `backlog_scan_ref:scan-ref-backlog-72c73a98bd54`
  - `backlog_scan_epoch:backlog-scan-epoch-feee914e7ce2`
  - `backlog_scan_doc:modulos/orquesta-autoprogramming/docs/tareas.md:line:27:sha256:55c7ad4bcc6cd71d41ec12ebdbdc6e19e54c3c0e47cfd2d3947f302398bc3582`
  - `backlog_scan_reservation_ref:reservation-ref-backlog-scan-doc-merge-9b9f5f8061bd`
  - `backlog_scan_reservation_ref:reservation-ref-backlog-task-id-4f2d7716ca0e`

Artefactos implementados sin consumir `T263`:

- `idle_self_improvement_v0.go`
- `idle_self_improvement_v0_test.go`
- `idle_self_improvement_apg001_test.go`
- documentacion local de contratos, pruebas y decisiones.
