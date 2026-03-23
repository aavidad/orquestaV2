<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# OP-050 — Diseno de presupuestos de sesion

## 1. Objetivo

Definir como debe modelar Orquesta el presupuesto operativo de una sesion para:

- anticipar handoff antes del agotamiento
- evitar perdida de contexto por limite de runtime
- separar politica de reparto de la telemetria concreta del proveedor

Este documento cubre la parte de `Codex3`: presupuestos de sesion y ventanas temporales.
Los pools de capacidad y reparto jerarquico quedan en el bloque complementario de `Codex2`.

## 2. Principios

1. El presupuesto es una politica de Orquesta, no del runtime.
2. La telemetria del proveedor es una fuente, no la fuente de verdad unica.
3. Si un proveedor no expone tokens o creditos, Orquesta debe seguir operando con tiempo, eventos y handoff manual.
4. Ninguna decision critica debe depender de una metrica no fiable sin marcar su calidad.
5. El handoff debe activarse antes del agotamiento, no despues.

## 3. Tipos de presupuesto

### 3.1. Presupuesto de sesion

Limite aplicado a una sesion concreta.

Ejemplos:

- `5h` maximas de ventana temporal
- `200k` tokens estimados
- `80%` de contexto consumido
- `50` mensajes maximos

### 3.2. Presupuesto de periodo

Limite acumulado por ventana de tiempo.

Ejemplos:

- limite diario
- limite semanal
- limite mensual

### 3.3. Presupuesto de seguridad

Reserva obligatoria para no agotar la capacidad al 100%.

Ejemplos:

- `20%` de tokens de margen
- `30m` de tiempo reservado
- `10` mensajes de colchón

## 4. Calidad de telemetria

Cada medida debe llevar calidad declarada:

- `directa`: viene del proveedor o runtime
- `derivada`: calculada por Orquesta desde eventos o historico
- `manual`: configurada por operador
- `desconocida`: no hay telemetria suficiente

Esto evita acoplar el nucleo a una falsa precision.

## 5. Estados de presupuesto

Para cada sesion, Orquesta debe calcular:

- `ok`
- `aviso`
- `handoff_recomendado`
- `handoff_obligatorio`
- `agotado`

Regla inicial propuesta:

1. `aviso` cuando se consume el 70% del presupuesto efectivo
2. `handoff_recomendado` al 85%
3. `handoff_obligatorio` al 95%
4. `agotado` al alcanzar o superar el 100%

Si solo existe telemetria temporal:

- `aviso` a partir del 70% de la ventana
- `handoff_recomendado` a partir del 85%
- `handoff_obligatorio` a partir del 95%

## 6. Politica de handoff

Cuando una sesion entra en `handoff_recomendado` o superior, Orquesta debe exigir:

- refresco de `heartbeat`
- guardado de `external_session_id` si existe
- guardado de `resumen_continuidad`
- referencia a tarea, propuesta y rama activa
- identificacion del siguiente agente o pool candidato

Cuando entra en `handoff_obligatorio`:

- no se deben abrir subtareas largas nuevas
- no se debe iniciar investigacion pesada nueva
- se debe priorizar cierre atomico o traspaso

## 7. Modelo minimo propuesto

### 7.1. Tabla `presupuestos_sesion`

Campos minimos:

- `id`
- `sesion_id`
- `tipo`
- `unidad`
- `limite_valor`
- `reserva_valor`
- `warning_ratio`
- `handoff_ratio`
- `hard_ratio`
- `fuente`
- `calidad_telemetria`
- `activo`
- `created_at`
- `updated_at`

Notas:

- `tipo`: `tiempo`, `tokens`, `mensajes`, `creditos`, `otro`
- `unidad`: `segundos`, `tokens`, `mensajes`, `creditos`, `porcentaje`
- `fuente`: `runtime`, `orquesta`, `manual`, `mixta`

### 7.2. Tabla `consumos_sesion`

Campos minimos:

- `id`
- `sesion_id`
- `presupuesto_id`
- `valor_consumido`
- `valor_restante`
- `estado`
- `observado_en`
- `calidad_telemetria`
- `detalle_json`

### 7.3. Tabla `politicas_handoff`

Campos minimos:

- `id`
- `proyecto_id`
- `pool_id`
- `tipo_presupuesto`
- `accion`
- `umbral_ratio`
- `requiere_resumen`
- `requiere_external_session_id`
- `requiere_asignacion_destino`
- `activa`

## 8. Separacion correcta de responsabilidades

### Nucleo

- decide estado de presupuesto
- aplica politicas de handoff
- resuelve cuando bloquear, avisar o permitir continuidad

### Adaptador runtime

- obtiene telemetria si el proveedor la expone
- normaliza medidas a un contrato comun
- declara calidad de la telemetria

### Persistencia

- almacena politicas, snapshots y estados
- no decide reglas de negocio

## 9. Integracion futura con MCP

Resources candidatos:

- `orquesta://sesiones/{id}/presupuesto`
- `orquesta://agentes/{agente}/presupuesto`
- `orquesta://pools/{pool}/presupuesto`

Tools candidatas:

- `orquesta.sesion.presupuesto.ver`
- `orquesta.sesion.presupuesto.actualizar`
- `orquesta.sesion.handoff.forzar`

Prompts candidatos:

- `handoff_preventivo`
- `continuidad_por_presupuesto`

## 10. Riesgos a vigilar

- asumir precision de tokens donde el runtime no la da
- mezclar presupuesto de sesion con capacidad global del pool
- bloquear trabajo util por umbrales demasiado agresivos
- esconder la politica dentro de conectores o SQL directo

## 11. Orden recomendado

1. Consolidar contrato comun de telemetria de presupuesto
2. Introducir tablas aditivas de politicas y snapshots
3. Calcular estado de presupuesto en servicio de aplicacion
4. Exponer lectura por CLI/web/MCP
5. Activar handoff preventivo con gates graduales
