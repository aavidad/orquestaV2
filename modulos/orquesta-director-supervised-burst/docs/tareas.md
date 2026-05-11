# Tareas: orquesta-director-supervised-burst

## Backlog local

### DSB-000 - Contrato local

Write-set: docs locales.

Cierre:

- contrato, decisiones, pruebas y tareas documentadas.

### DSB-001 - Caso de uso de rafaga

Write-set: tipos, validacion, helpers y caso de uso.

Cierre:

- `RunDirectorSupervisedBurstV0` implementado por puertos, sin adaptadores operativos.

### DSB-002 - Pruebas de parada

Write-set: tests locales.

Cierre:

- continuidad, outbox, max_steps, error de builder y error de paso cubiertos.

### DSB-003 - Registro global

Write-set: `modulos/README.md`, `modulos/CONTRATOS.md`, estado del nucleo y verificador.

Cierre:

- el modulo entra en `verificar_nucleo_orquesta_v2.sh`.
