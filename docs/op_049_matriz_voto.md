<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# OP-049 — Matriz de voto

Este documento resume que se espera que validen `Codex2` y `Codex3` sobre `OP-049`.

## 1. Preguntas a responder

Cada agente revisor debe posicionarse sobre estas decisiones:

1. Aceptar o rechazar que Orquesta sea el nucleo transversal del workspace `~/Trabajo`.
2. Aceptar o rechazar arquitectura hexagonal con BD como adaptador.
3. Aceptar o rechazar capa de conectores para LLM y forge.
4. Aceptar o rechazar sesiones reanudables por proyecto con `external_session_id` y `resumen_continuidad`.
5. Aceptar o rechazar `worktrees + locks` como base del paralelismo seguro.
6. Aceptar o rechazar fases explicitas de investigacion a seguimiento.
7. Aceptar o rechazar gobierno Git desde Orquesta: commit, push, merge, progreso y ramas.
8. Aceptar o rechazar MCP como interfaz de integracion y contexto.
9. Aceptar o rechazar memoria de proyecto y deteccion de deriva como capacidades de primer nivel.
10. Aceptar o rechazar refactorizacion competitiva con backup y dos revisores no autores.

## 2. Criterios de revision

El voto no debe basarse en gusto personal. Debe responder a:

- coherencia tecnica
- compatibilidad con lo ya implementado
- coste de evolucion razonable
- seguridad
- mantenibilidad
- utilidad real para el flujo diario

## 3. Formato de voto esperado

Cada agente debe emitir:

- posicion global: `acuerdo`, `desacuerdo` o `abstencion`
- lista corta de puntos correctos
- lista corta de riesgos o ajustes
- recomendacion de orden de implementacion

## 4. Orden sugerido de implementacion

1. Hexagonalizacion progresiva y puertos.
2. Locks y worktrees.
3. Memoria, fuentes y deriva.
4. Fases y progreso.
5. Gobierno Git.
6. MCP y conectores avanzados.
7. Automatizacion de runtime y app completa.

## 5. Riesgos que deben revisar

- que la complejidad no supere la capacidad de mantenimiento
- que SQLite no quede expuesto a demasiada concurrencia
- que la capa de conectores no acabe duplicando logica del nucleo
- que Terminator y sesiones manuales no introduzcan ambiguedad
- que el calculo de progreso no se convierta en una metrica enganosa
