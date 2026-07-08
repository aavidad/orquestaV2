---
name: orquesta-programacion-autonoma
description: Programar cambios de software con Orquesta como director: descomponer trabajo, lanzar agentes/subagentes, conservar avances, probar, limpiar worktree y cerrar con evidencia.
---

# Orquesta Programacion Autonoma

Usa esta skill cuando el objetivo sea programar, corregir o cerrar trabajo de
codigo en cualquier repo consumidor de Orquesta.

## Flujo

1. Lee `AGENTS.md` del repo y del modulo.
2. Declara write-set estrecho.
3. Divide en tareas independientes y lanza agentes si el runtime lo permite.
4. Pide finales compactos: cambios, rutas, pruebas, bloqueos reales.
5. Integra sin pisar cambios ajenos.
6. Ejecuta pruebas focales y luego la verificacion minima del repo.
7. Limpia artefactos generados sin borrar trabajo util.
8. Documenta decisiones o pendientes verificables.

## Reglas

- No metas reglas de dominio dentro del nucleo generico.
- No conviertas alias, nombres cercanos, contexto reducido o formato reparable
  en fallo automatico.
- Conserva entregas parciales como borrador, evidencia o tarea derivada.
- Los cortes fuertes son seguridad real, causalidad rota, refs imposibles,
  datos sensibles efectivos o efectos externos no autorizados.
- Si una regla bloquea trabajo valido, el Director decide si se ignora, se
  repara o se saca del camino de ejecucion.

## Programacion

- Aplicar `orquesta-programacion-minima` si la tarea trae esa skill_ref o el
  Director la activa para comparar diff minimo. No convertirla en default amplio
  sin A/B en golden tasks.
- Preferir patrones locales del repo.
- Toda app generada o modificada por Orquesta debe ser hexagonal estricta:
  separar domain, application/usecases, ports, adapters y bootstrap.
- Domain/application no importan adapters, HTTP, DB, filesystem, runtime, LLM ni
  UI; dependen de puertos/interfaces.
- Handlers y adaptadores son finos: traducen transporte, validan forma y llaman
  casos de uso; no construyen repositorios, autenticacion, fixtures, reglas de
  negocio ni configuracion global.
- El wiring vive en bootstrap/cmd o superficie de composicion canonica.
- No hardcodear configuracion que deba cambiarse sin redeploy.
- Mantener i18n para texto visible de UI.
- Antes de produccion, probar en local o contra instancia temporal.

## Entrega

Informe breve con:

- agentes usados;
- archivos cambiados;
- pruebas ejecutadas;
- worktree limpio o motivo real si no puede quedar limpio;
- pendientes futuros no bloqueantes.
