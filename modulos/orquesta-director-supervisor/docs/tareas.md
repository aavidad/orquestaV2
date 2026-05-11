# Tareas: orquesta-director-supervisor

## Backlog local

### DSV-000 - Contrato local

Write-set: docs locales.

Cierre:

- contrato, decisiones, pruebas y tareas documentadas.

### DSV-001 - Politica de decision v0

Write-set: tipos, validacion, helpers y caso de uso.

Cierre:

- `DecideDirectorSupervisorNextActionV0` implementado sin efectos externos.

### DSV-002 - Pruebas de estados

Write-set: tests locales.

Cierre:

- todos los status del ciclo quedan cubiertos.

### DSV-003 - Registro global

Write-set: `modulos/README.md`, `modulos/CONTRATOS.md`, estado del nucleo y verificador.

Cierre:

- el modulo entra en `verificar_nucleo_orquesta_v2.sh`.

### DSV-004 - Recomendacion autonoma local

Write-set: tipos, helpers, caso de uso, tests y docs locales.

Cierre:

- la decision expone `autonomous_recommendation`;
- outbox y candidatos pendientes recomiendan `wait` sin operador manual.
