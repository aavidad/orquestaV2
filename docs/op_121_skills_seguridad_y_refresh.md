# OP-121 - Seguridad basica y refresh dinamico de skills

## Alcance implementado

Este bloque cubre las tareas `#375` y `#376` de `OP-121`.

Ya queda resuelto:

- metadata de seguridad para skills con herramientas externas
- aprobacion explicita antes de activar skills externas
- notificacion en caliente a agentes activos del rol afectado

No cubre todavia:

- creacion automatica de skills faltantes (`#377`)
- instalacion/ejecucion real de herramientas externas por skill
- sandbox de ejecucion por skill

## Metadata de seguridad

Cada skill puede declarar ahora:

- `origen`
  - `builtin`
  - `local`
  - `third_party`
- `nivel_riesgo`
  - `bajo`
  - `medio`
  - `alto`
- `requiere_aprobacion`

Regla base:

- si la skill es `builtin`, no requiere aprobacion adicional
- si la skill es `local` o `third_party`, se trata como skill externa

## Politica para skills externas

Al crear o editar una skill externa:

- Orquesta la deja `inactiva`
- `requiere_aprobacion = true`
- si no se indica riesgo, sube como minimo a `medio`

Al activar una skill externa:

- solo `admin` puede activarla
- al activarla, deja de estar pendiente de aprobacion

Consecuencia practica:

- un agente puede registrar una skill externa sin habilitarla de inmediato
- la aprobacion operativa queda separada del alta en catalogo
- el historial deja claro quien la aprobo al quedar trazado en auditoria y versionado

## Refresh dinamico

Cuando una skill cambia por:

- `crear`
- `actualizar`
- `activar/desactivar`

Orquesta emite un mensaje `skills_refresh` en `runtime_mailbox` para los agentes con sesion activa del rol afectado.

Payload base:

- `skill_id`
- `tipo_agente`
- `nombre`
- `motivo`
- `origen`
- `nivel_riesgo`
- `requiere_aprobacion`
- `activa`

Esto permite que un agente ya trabajando:

- detecte que el catalogo ha cambiado
- consulte el orquestador en caliente
- vea una skill nueva o un cambio de estado sin reiniciar la sesion

## Superficie tocada

- `db/reglas.go`
- `db/skills_catalog.go`
- `db/skills_runtime_refresh.go`
- `db/catalogo_versionado.go`
- `db/schema.go`
- `db/post_migraciones_compat.go`
- `cmd/api.go`
- `cmd/catalogo_briefing.go`

## Verificacion

Tests focalizados en verde:

```bash
go test ./db -run 'Test(GuardarYListarSkills|SkillsOrdenadasPorPrioridadYEscenarioYConVersionado|CrearSkillRechazaDuplicadoEquivalente|SkillExternaQuedaPendienteYSoloAdminLaActiva|CrearSkillNotificaRefreshAMailboxDeAgentesActivosDelRol)'
go test ./cmd -run 'Test(APICatalogoMutaciones|APISkillRechazaDuplicadoEquivalente|APISkillExternaQuedaPendienteYSoloAdminLaActiva)'
```
