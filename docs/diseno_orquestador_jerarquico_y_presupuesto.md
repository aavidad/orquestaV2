<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Diseño integrado de orquestador jerárquico y presupuesto de sesión

## Objetivo

Unificar en un solo diseño operativo dos decisiones que ya existen por separado en Orquesta:

- reparto jerárquico por pools de capacidad
- control de presupuesto de sesión y handoff preventivo

La idea no es duplicar `OP-050`, sino dejar claro cómo conviven ambas piezas dentro del plano de control real de Orquesta.

## Base reutilizada

Este documento integra lo ya definido en:

- `docs/op_050_orquestador_jerarquico.md`
- `docs/op_050_presupuestos_sesion.md`
- `docs/diseno_minimo_pools_y_presupuestos.md`
- `docs/op_052_control_presupuesto_y_relevo.md`
- `docs/diseno_control_activo_agentes.md`

## Problema conjunto

Orquesta no solo reparte tareas entre agentes.
También debe decidir:

- qué pool o runtime conviene usar para cada trabajo
- cuánto margen operativo queda antes de cortar una sesión
- cuándo conviene relevar a otro agente sin perder continuidad

Si estas dos decisiones se modelan por separado, aparecen fallos típicos:

- un pool parece disponible pero sus sesiones reales están cerca del agotamiento
- se abre un agente hijo aunque el presupuesto restante ya no permite trabajo útil
- el handoff se decide tarde porque el sistema mira solo tareas y no presupuesto

## Modelo integrado

## 1. Nivel superior: orquestador global

El orquestador superior decide estrategia de workspace:

- prioridades de proyecto
- reparto entre pools
- reserva de capacidad
- elección de modelo o runtime cuando haya varias opciones

No asigna trabajo “a una licencia concreta” en crudo.
Asigna contra un pool con política, capacidad y telemetría propias.

## 2. Nivel intermedio: pools de capacidad

Cada pool representa una fuente operativa de capacidad.

Ejemplos:

- `codex`
- `claude`
- `android`
- futuros pools para modelos locales o conectores remotos

Campos conceptuales mínimos:

- identidad del pool
- proveedor/runtime
- capacidad total y reservada
- si permite agentes hijos
- política de delegación y handoff
- fuente y calidad de telemetría

## 3. Nivel inferior: sesión y agente activo

La unidad operativa real sigue siendo la sesión viva de un agente.

Cada sesión debe poder relacionarse con:

- agente
- pool
- proyecto
- modelo si aplica
- presupuesto y consumo observados
- continuidad necesaria para relevo

## Separación correcta de responsabilidades

## 1. Pool

Responde a:

- cuánta capacidad global hay
- qué modelos o runtimes están disponibles
- qué política de reparto y sobrecoste existe

No debe confundirse con el presupuesto concreto de una sesión.

## 2. Presupuesto de sesión

Responde a:

- cuánto margen operativo le queda a una sesión concreta
- qué calidad tiene esa telemetría
- si se debe avisar, recomendar handoff o exigir relevo

No debe cargarse con decisiones globales de reparto entre pools.

## 3. Supervisor

El supervisor es quien cruza ambas piezas:

- detecta si un pool tiene hueco real
- comprueba si la sesión elegida tiene margen suficiente
- evita arrancar trabajo largo sobre sesiones degradadas
- fuerza checkpoint e inicia handoff preventivo cuando toca

## Flujo operativo recomendado

## 1. Selección de pool

Antes de asignar o arrancar:

1. se listan pools compatibles con la tarea
2. se filtran por capacidad disponible
3. se aplican restricciones de coste, modelo o política
4. se elige el pool más adecuado

## 2. Validación de presupuesto

Antes de abrir trabajo útil:

1. se inspecciona el presupuesto de la sesión candidata
2. se mide su estado: `ok`, `aviso`, `handoff_recomendado`, `handoff_obligatorio`, `agotado`
3. si la sesión está degradada, no se abre trabajo largo nuevo

## 3. Handoff preventivo

Si la sesión entra en umbral de relevo:

1. se fuerza actualización de continuidad
2. se guarda `external_session_id` si existe
3. se guarda resumen estructurado
4. se resuelve candidato de relevo
5. el relevo se intenta primero dentro del pool o del proyecto según política

## Política de decisión recomendada

## 1. Regla principal

La asignación no debe mirar solo capacidad libre.
Debe mirar:

- capacidad del pool
- estado del presupuesto de la sesión
- calidad de telemetría disponible
- longitud o riesgo estimado del trabajo

## 2. Regla conservadora

Si la telemetría es mala o parcial:

- mejor avisar antes
- exigir continuidad antes
- evitar subtareas largas nuevas

## 3. Regla de relevo

Orden recomendado de búsqueda:

1. agente del mismo proyecto con sesión sana
2. agente del mismo pool con capacidad y presupuesto suficiente
3. mejor agente disponible del proyecto en otro pool

## Datos que deben quedar trazados

Para que el modelo sirva de verdad, el sistema debe poder reconstruir:

- qué pool fue elegido
- por qué se eligió
- qué presupuesto tenía la sesión al empezar
- cuándo cambió a estado de aviso o handoff
- qué continuidad se guardó
- qué agente o pool recibió el relevo

## Encaje con el control activo

Este diseño no sustituye el plano de control de agentes; lo alimenta.

Encaja con:

- `runtime_handles`
- `runtime_orders`
- `runtime_mailbox`
- `runtime_checkpoints`

En términos prácticos:

- el presupuesto no dispara acciones mágicas fuera del control plane
- genera señales y políticas que el supervisor traduce a órdenes

## Qué no debe hacerse

- decidir handoff solo por intuición manual si ya hay datos suficientes
- mezclar límites globales del pool con consumo de una sesión concreta
- abrir agentes hijos ignorando el presupuesto restante real
- esconder la lógica dentro de SQL o de un conector específico
- acoplar la decisión de relevo a un proveedor concreto

## Fases de implantación recomendadas

### Fase 1

- modelo de pools
- modelo de presupuestos de sesión
- lectura homogénea por CLI/API

### Fase 2

- cálculo de estado de presupuesto
- visibilidad en dashboard y diagnóstico
- advertencias operativas

### Fase 3

- handoff preventivo con políticas
- selección automática de relevo
- integración completa con órdenes del control plane

## Criterio de aceptación

El diseño estará bien aplicado si Orquesta puede responder, sin acceso lateral ni interpretación manual excesiva, a estas preguntas:

- qué pool conviene usar para esta tarea
- qué margen real tiene la sesión actual
- si se puede seguir o hay que relevar
- a quién se debe pasar el trabajo y por qué
