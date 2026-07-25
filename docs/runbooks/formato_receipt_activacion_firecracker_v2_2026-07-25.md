# Receipt V2 de activación Firecracker

Este documento sustituye únicamente el bloque de receipt V1 del runbook de
instalación. `--apply` no cambia y `--activate` solo admite el schema exacto
`orquesta_firecracker_activation_receipt.v2`.

El productor del E2E físico debe calcular:

- `evidence_sha256`: SHA-256 en minúsculas del artefacto inmutable que conserva
  la evidencia completa del smoke físico de 16 microVM;
- `policy_digest`: SHA-256 en minúsculas de la política efectiva que consumió
  ese mismo E2E.

El receipt tiene exactamente estas líneas, en este orden y con un único salto
de línea final:

```text
schema=orquesta_firecracker_activation_receipt.v2
status=passed
config_sha256=<sha256>
unit_sha256=<sha256>
primitives_unit_sha256=<sha256>
launcher_sha256=<sha256>
asset_digest=<sha256>
evidence_sha256=<sha256>
policy_digest=<sha256>
e2e_suite=orquesta.firecracker-attestor.physical-16.v1
max_concurrent_runs=16
physical_microvm_count=16
concurrent_high_water=16
all_attestations_valid=true
zero_residual_runs=true
network_absent=true
api_absent=true
vsock_absent=true
serial_absent=true
memory_swap_max_zero=true
```

Antes de activarlo, el fichero debe ser regular, enlace único, `root:root`,
modo `0400`, ruta absoluta canónica y estar bajo ancestros reales `root:root`
sin escritura de grupo/otros. No se admiten symlinks ni valores equivalentes
con otro orden, mayúsculas o formato.
