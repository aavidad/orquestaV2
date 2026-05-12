# Decision: anclaje obligatorio al objetivo actual

Fecha: 2026-05-12.

## Problema

En una prueba real de autoprogramacion, el director recibio una peticion concreta pero genero microtareas historicas/genericas. Las decisiones eran validas por forma, pero no demostraban relacion con el objetivo actual.

## Decision

Toda microtarea de programacion emitida por el director debe incluir en `acceptance_criteria` una linea `objetivo_actual: ...` con la capacidad concreta pedida y tokens principales de la solicitud.

El stack Codex valida ese anclaje antes de aplicar decisiones:

- si hay contexto compacto de la solicitud y falta `objetivo_actual`, la decision se rechaza;
- si `objetivo_actual` no menciona ningun token util del objetivo real, la decision se rechaza;
- las reglas de fases, refs causales, hexagonal, i18n y conectores siguen siendo obligatorias.

## Motivo

Esto evita que el director reutilice planes antiguos o tareas validas solo de manera estructural. La correccion queda en el contrato y en una policy verificable, no en supervision manual.

## Alcance

Aplica a microtareas de `programacion`. El contexto llega por puerto hexagonal como `ObjectiveHints`, sin acoplar el nucleo a proveedor, modelo, DB, runtime ni UI.
