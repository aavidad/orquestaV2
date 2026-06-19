---
name: orquesta-programacion-revision
description: Revisar cambios de codigo con agentes, consejo y votacion: bugs, regresiones, arquitectura, riesgos y rework causal antes de aceptar.
---

# Orquesta Programacion Revision

Usa esta skill para revisar cambios de software antes de cierre.

## Prioridad

1. Bugs o regresiones.
2. Incumplimiento de arquitectura hexagonal estricta.
3. Tests faltantes.
4. Seguridad o efectos externos.
5. Limpieza y mantenimiento.

## Agentes

- tecnico: correctness, concurrencia, errores y APIs;
- arquitectura: hexagonal estricta, domain/application/ports/adapters/bootstrap,
  handlers finos, puertos, adaptadores, configuracion;
- producto/UI: experiencia, i18n, accesibilidad;
- tests: cobertura focal, fixtures y pruebas de regresion.

## Corte De Aceptacion

La revision no debe aceptar una app generada por Orquesta si:

- domain/application importan adapters, HTTP, DB, filesystem, runtime, LLM o UI;
- un handler/adaptador contiene composicion de repositorios, autenticacion,
  fixtures, reglas de negocio o configuracion global;
- el wiring no vive en bootstrap/cmd o en una superficie canonica equivalente.

## Votacion

Si hay varias soluciones, pedir candidatos y votos. El Director decide, fusiona
o pide rework; no descarta candidatos recuperables.

## Entrega

Findings con ruta/linea, severidad, razon, prueba esperada y decision.
