<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Handoff Codex1 — 2026-03-22

## Estado general

Trabajo orientado a dejar Orquesta preparada para relevo y trabajo paralelo con varios agentes.

## Decisiones y documentación ya dejadas

- `ARQUITECTURA.md`
- `docs/README.md`
- `docs/orquesta_v1_vision.md`
- `docs/op_049_matriz_voto.md`
- `docs/orquesta_v1_roadmap.md`
- `docs/operacion_agentes_manuales.md`
- `docs/op_050_orquestador_jerarquico.md`
- `docs/op_050_matriz_voto.md`
- `docs/op_052_control_presupuesto_y_relevo.md`
- `docs/diseno_minimo_pools_y_presupuestos.md`
- `docs/matriz_modelo_y_razonamiento.md`
- `docs/diseno_control_activo_agentes.md`

## Propuestas relevantes

- `OP-049`
  Arquitectura objetivo de Orquesta v1.
  Estado observado: `consenso`.

- `OP-048`
  Conector GitHub para alta de proyectos.
  Sigue abierta.

- `OP-050` y `OP-051`
  Han quedado dos propuestas muy parecidas sobre orquestador jerárquico y presupuesto de sesión.
  Conviene consolidarlas en una sola y cerrar la duplicada con criterio explícito.

- `OP-052`
  Debe usarse para votar de forma separada la política concreta de presupuesto restante, estados amarillo/rojo y relevo preventivo.

## Cambios de código ya hechos

### Votación y consenso

Se endureció la lógica de consenso:

- mínimo de votos configurables
- mínimo de votos de acuerdo de agentes no autores
- cierre manual a `consenso` también validado por política

Claves de configuración añadidas:

- `propuesta_min_votes=2`
- `propuesta_min_non_author_votes=2`

Tests añadidos:

- `db/propuestas_consenso_test.go`

Verificación hecha:

```bash
env GOCACHE=/tmp/orquesta-gocache go test ./db -run 'TestConsenso|TestCerrarPropuestaConsenso' -v -count=1 -timeout=30s
env GOCACHE=/tmp/orquesta-gocache go test ./cmd -count=1
```

### Briefing de autonomía segura

Se añadió al briefing y a la documentación:

- acciones normales sin pedir permiso previo
- acciones destructivas o peligrosas: consultar antes con Orquesta o con otro agente del mismo proyecto

### Terminator y sesiones manuales

Existen wrappers:

- `scripts/cargar_agentes.sh`
- `scripts/terminator_agentes.sh`
- `scripts/agente_console.sh`
- `scripts/agentes.orquestador.plan`

Se trabajó para:

- usar worktree por agente cuando comparten proyecto
- guardar `external_session_id`
- guardar `resumen_continuidad`
- usar `cwd` aislado por agente

## Riesgos o deuda abierta

### 1. Agentes duplicados por mayúsculas/minúsculas

En la BD real existen:

- `Codex1` y `codex1`
- `Codex2` y `codex2`

Esto afecta:

- `agentes`
- `sesiones`
- `tareas`
- `audit_log`
- `votos`
- `asignaciones`
- `worktrees`

No se ha hecho la fusión todavía porque requiere:

- backup previo
- plan de merge transaccional
- decisión del nombre canónico

### 2. Duplicidad de propuestas OP-050/OP-051

La idea quedó registrada dos veces por contención de SQLite y reintentos.
Hay que decidir cuál se conserva.

Además, la propuesta formal para votar el diseño de control activo de agentes vivos puede requerir reintento si vuelve a coincidir con contención de SQLite.
El texto objetivo es:

- título:
  `Control activo de agentes vivos desde Orquesta`
- descripción:
  `Decidir el canal de control para arrancar, enviar instrucciones, pausar, continuar y hacer handoff sobre agentes vivos. Ver docs/diseno_control_activo_agentes.md`

### 3. Terminator

El entorno gráfico obligó a varios intentos.
El uso real quedó delegado al usuario.
Los wrappers están más robustos, pero no se ha cerrado todavía un arranque completamente fiable desde esta sesión.

### 4. Control activo de agentes vivos

Sigue faltando la pieza crítica para autonomía real de la app:

- Orquesta planifica y persiste bien
- pero todavía no empuja instrucciones a agentes ya vivos por sí sola
- el usuario sigue actuando como puente manual con las consolas abiertas

La siguiente fase debe cubrir:

- `agente arrancar`
- `agente enviar`
- `agente pausar`
- `agente continuar`
- `agente handoff`
- handles de runtime para PTY/API/MCP

### 5. Traslado fisico del repo

La arquitectura objetivo pide sacar `orquestador` a `~/Trabajo/orquestador`, pero eso no se ha ejecutado.

No debe hacerse mientras haya agentes activos en la ruta actual `~/Trabajo/PlataformaMunicipal/orquestador`.
Primero hay que pausar sesiones, guardar continuidad, hacer backup y actualizar rutas persistidas.

## Tareas abiertas creadas para otros agentes

- `#149` `Codex2`
  Votar `OP-050` y diseñar pools de capacidad.

- `#150` `Codex3`
  Votar `OP-050` y diseñar presupuestos de sesión.

- `#151` `Codex3`
  Implementar esquema mínimo de presupuestos de sesión.
  Estado observado al cierre de esta sesión: completada.

- `#152` `Codex2`
  Implementar esquema mínimo de pools y modelos.
  Estado observado al cierre de esta sesión: completada.

- `#153` `Codex1`
  Diseñar control activo de agentes vivos.

- `#156` `Codex1`
  Planificar traslado fisico de orquestador a `~/Trabajo`.
  Estado deseado: bloqueada hasta que no queden agentes activos en la ruta actual.

- `#154` `Codex2`
  Implementar envío autónomo de instrucciones a agentes.

- `#155` `Codex3`
  Diseñar handoff y reasignación sobre agentes vivos.

- `#157` `Codex3`
  Votar `OP-052` sobre presupuesto de sesión y relevo.

- `#158` `Codex2`
  Votar `OP-052` sobre presupuesto de sesión y relevo.

- `#159` `Codex1`
  Consolidar `OP-050` y `OP-051` en una sola decisión.

- `#160` `Codex1`
  Fusionar agentes duplicados por mayúsculas y minúsculas con backup previo.

## Recomendación de siguiente bloque

1. Consolidar `OP-050/OP-051`.
2. Diseñar esquema mínimo de:
   - `pools_capacidad`
   - `pool_modelos`
   - `presupuestos_sesion`
   - `politicas_modelo`
3. Definir política de telemetría por proveedor:
   - CLI status
   - API/console
   - configuración manual
   - inferencia
4. Separar voto explícito sobre:
   - porcentaje restante
   - tiempo absoluto restante
   - tipo de ventana
   - umbrales de checkpoint y handoff
5. Resolver motor de selección:
   - perfil de tarea
   - pool
   - modelo
   - razonamiento
6. Implementar control activo sobre agentes vivos.
7. Fusionar agentes duplicados `Codex*/codex*` con backup previo.

## No perder de vista

- las migraciones deben seguir siendo aditivas e idempotentes
- no borrar datos reales de la BD
- el núcleo debe seguir desacoplándose de SQLite
- las decisiones estructurales deben pasar por voto no autor

## Actualizacion de cierre del dia

- `OP-079` queda en consenso:
  estrategia hibrida de control de agentes.

- `OP-080` queda en consenso:
  modo servidor unico para eliminar `SQLITE_BUSY`.

- `OP-081` queda en consenso:
  RAEX debe ser incremental y no un gate rigido.

- `OP-082` queda abierta:
  observabilidad pasiva de runtimes y agentes hijos para que el panel muestre estado real sin interrumpir al agente.

- Documentos nuevos:
  - `docs/diseno_observabilidad_pasiva_runtimes_es.md`
  - `docs/diseno_observabilidad_pasiva_runtimes_en.md`

- Tareas abiertas asociadas:
  - `#240` votar `OP-082` (`Codex2`)
  - `#241` votar `OP-082` (`Codex3`)
  - `#242` implementar modelo base de `runtime_instances` y `runtime_telemetry` (`Codex1`)
