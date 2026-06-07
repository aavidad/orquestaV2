---
name: orquesta-programacion-tests
description: Planificar y ejecutar pruebas de software con Orquesta: pruebas focales, suite completa, fallos como tareas reparables y evidencia de validacion.
---

# Orquesta Programacion Tests

Usa esta skill cuando un cambio de codigo necesite validacion.

## Orden

1. Prueba focal del modulo tocado.
2. Prueba de contrato/arquitectura si cambia frontera.
3. Prueba de integracion si cambia wiring.
4. `git diff --check`.
5. Suite completa si el cambio es transversal.

## Fallos

- No tirar el trabajo por un fallo reparable.
- Separar fallo del cambio, fallo previo y fallo de entorno.
- Crear rework causal con comando, salida resumida y ruta probable.
- Si una prueba es demasiado estricta por texto/alias, corregir la prueba o el
  adaptador; no bloquear una entrega valida por coincidencia literal.

## Evidencia

Guardar comandos ejecutados, resultado, tests omitidos y motivo.
