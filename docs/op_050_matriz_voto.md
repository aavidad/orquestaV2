<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# OP-050 — Matriz de voto sobre pools y presupuesto de sesión

## Objetivo

Que los agentes revisores voten opciones concretas sobre:

- pools/licencias
- presupuesto de sesión
- handoff preventivo
- uso del porcentaje restante

## 1. Preguntas de voto

### A. ¿Cómo modelar la capacidad?

Opciones:

- `A1`
  Licencias rígidas por proveedor.
  Simple, pero poco flexible.

- `A2`
  Pools de capacidad por proveedor/plan/runtime.
  Flexible, ampliable y mejor para crecimiento.

Recomendación inicial: `A2`.

### B. ¿Qué métrica principal usar para handoff?

Opciones:

- `B1`
  Solo tiempo absoluto restante.

- `B2`
  Solo porcentaje restante de la ventana.

- `B3`
  Modelo mixto:
  porcentaje restante + tiempo absoluto + contexto de tarea.

Recomendación inicial: `B3`.

### C. ¿Cuándo debe activarse el handoff preventivo?

Opciones:

- `C1`
  Cuando quede menos de un porcentaje fijo.
  Ejemplo: `10%`.

- `C2`
  Cuando quede menos de tiempo fijo.
  Ejemplo: `30 min`.

- `C3`
  Regla combinada.
  Ejemplo: `<=10%` o `<=30 min`, lo que ocurra antes.

Recomendación inicial: `C3`.

### D. ¿Cómo obtener la telemetría?

Opciones:

- `D1`
  Solo por CLI status.

- `D2`
  Solo por API/consola del proveedor.

- `D3`
  Mixto:
  CLI status + API + configuración manual + inferencia.

Recomendación inicial: `D3`.

### E. ¿Qué hacer cuando no hay telemetría fiable?

Opciones:

- `E1`
  No tomar decisiones automáticas.

- `E2`
  Usar una política manual por proveedor/plan.

- `E3`
  Política manual + inferencia operativa.

Recomendación inicial: `E3`.

### F. ¿Qué debe hacer Orquesta al entrar en zona de riesgo?

Opciones:

- `F1`
  Solo avisar.

- `F2`
  Avisar y bloquear nuevas tareas largas.

- `F3`
  Avisar, bloquear nuevas tareas largas y forzar checkpoint/handoff.

Recomendación inicial: `F3`.

## 2. Datos mínimos a guardar

Los revisores deben confirmar si esta base es suficiente:

- `window_kind`
- `reset_at`
- `remaining_seconds`
- `remaining_tokens`
- `remaining_messages`
- `remaining_credits`
- `budget_source`
- `raw_snapshot_json`

## 3. Criterio específico para nuestro caso

Para nuestro entorno real, usar solo porcentaje no basta.

Ejemplo:

- `6%` de una ventana semanal puede seguir siendo mucho tiempo
- `6%` de una ventana de 5 horas puede significar relevo inmediato

Por eso la métrica correcta para Orquesta parece ser:

- porcentaje restante
- tiempo restante absoluto
- tipo de ventana (`5h`, semanal, etc.)
- tamaño/riesgo de la tarea actual

## 4. Formato de voto esperado

Cada agente debe responder al menos:

- `A`: opción elegida
- `B`: opción elegida
- `C`: opción elegida
- `D`: opción elegida
- `E`: opción elegida
- `F`: opción elegida
- riesgos o ajustes

## 5. Recomendación inicial de Codex1

- `A2`
- `B3`
- `C3`
- `D3`
- `E3`
- `F3`
