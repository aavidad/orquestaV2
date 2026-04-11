<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Uso actual de la app Orquesta

## Objetivo

Explicar cómo se usa hoy Orquesta mientras la app completa todavía no controla por sí sola todo el ciclo de vida de los agentes.

## Capas actuales

Orquesta se usa hoy por tres vías complementarias, pero con un único plano operativo válido:

1. Servicio/daemon
   Es la fuente de verdad operativa para sesiones, tareas, propuestas, votos y control de agentes.

2. Web
   Es un cliente HTTP sobre el servicio, con dashboard y acciones de gestión.

3. CLI y API HTTP/JSON
   Actúan como clientes del servicio. El modo local queda solo para recuperación explícita.

## Arranque del panel web

```bash
cd ~/Trabajo/orquesta
./orquesta serve
```

Por defecto queda en:

```text
http://127.0.0.1:16543
```

## Qué ofrece hoy la web

Rutas HTML disponibles:

- `/`
  Dashboard.
- `/tareas`
  Lista, alta, detalle y acciones de tareas.
- `/propuestas`
  Lista, alta, detalle y acciones de propuestas.

La web actual sirve para:

- ver progreso general
- ver tareas en progreso
- crear tareas
- reasignar o cambiar estado de tareas desde sus vistas
- crear propuestas
- votar o cerrar propuestas desde sus vistas

## Qué ofrece hoy la API

Rutas JSON principales:

- `/api/status`
- `/api/agentes`
- `/api/reglas`
- `/api/skills`
- `/api/workflows`
- `/api/proyectos`
- `/api/conectores`
- `/api/asignaciones`
- `/api/asignaciones/activar`
- `/api/locks`
- `/api/tareas`
- `/api/propuestas`
- `/api/worktrees`
- `/api/sesiones/inicio`
- `/api/sesiones/guardar`
- `/api/sesiones/fin`
- `/api/sesiones/continuar`
- `/api/agente/preparar`
- `/api/agente/tick`

La API ya cubre más superficie que la web.
Por eso, mientras la interfaz gráfica no llegue a todo, la combinación correcta es:

- servicio/daemon como fuente de verdad
- web y CLI como clientes del servicio
- API para integración con escritorio, automatizaciones y control plane
- OpenClaw Gateway como adaptador saliente opcional de notificaciones, configurado por `openclaw_gateway_url`, `openclaw_gateway_token` y `openclaw_gateway_operator`

Consultas de briefing de agentes ya cubiertas por API:

- `GET /api/reglas?tipo_agente=programador`
- `GET /api/skills?agente=Codex1`
- `GET /api/workflows?tipo_agente=programador`
- `GET /api/workflows?tipo_agente=programador&nombre=inicio-sesion`

Estado actual del catalogo de skills:

- indice ordenado por `prioridad`, `escenario` y `nombre`
- metadata minima ya expuesta para skills: `escenario`, `prioridad`, `aliases_json`, `herramientas_json`
- mutaciones versionadas y auditadas
- anti-duplicado funcional por equivalencia canonica, no solo por nombre exacto

## Flujo correcto para un agente manual

Entrada manual de compatibilidad o recuperación:

```bash
scripts/inicio_agente.sh <agente>
```

Ejemplos:

```bash
scripts/inicio_agente.sh Codex2
scripts/inicio_agente.sh Codex3 --tarea 155
scripts/inicio_agente.sh antigravity --proyecto orquestador --no-auto
```

Ese wrapper hace:

1. `orquesta sesion inicio`
2. muestra tareas activas del agente
3. inicia la tarea si se le pasa o si hay una única candidata clara

## Regla práctica para documentadores

Los agentes documentadores como `antigravity` deben usar:

- la web para seguir estado general
- la CLI para iniciar sesión, tomar tarea y votar
- la API solo cuando se documente o se pruebe integración

## Limitaciones actuales

- la web todavía no cubre todo el modelo de proyectos, conectores y control activo de agentes
- el arranque autónomo completo de agentes sigue en cierre progresivo aunque el control plane y el autobootstrap ya operan
- una instalación nueva ya no debe levantar flota legacy por seed implícito: para pruebas locales de Ollama o flotas específicas, los agentes se registran explícitamente desde la app/API
- la app de escritorio aún no existe como producto terminado
- parte del gobierno operativo sigue pasando por CLI y scripts

## Estado actual de Ollama local

La política vigente para agentes locales de Ollama es dual:

1. Vía experimental y de compatibilidad
   - `agente` -> `ollama-cli` -> `tmux`
   - útil para smokes, depuración, comparación de candidatos y rescate

2. Vía canónica objetivo
   - `pool local`
   - `slots`
   - agentes lógicos por `perfil_tarea`
   - microprogramación dirigida servida por la app

Regla operativa:

- la vía experimental no desaparece de golpe
- pero la promoción de modelos locales y el camino preferente de producción deben moverse al pool compartido gobernado por la app
- con recursos actuales, `gemma4:26b` es el worker local preferente y Qwen queda en estado experimental mientras no supere la smoke canónica de Orquesta

## Política de Acceso a Persistencia (AP-077)

No se permite el acceso directo a la base de datos (p. ej. mediante `sqlite3`) para realizar mutaciones o escrituras en el flujo normal de trabajo. 

La lectura e inspección directa es excepcional y solo se tolera mientras la CLI/API no proporcione la observabilidad y administración necesarias. Cuando la cobertura sea total, el acceso externo quedará bloqueado. Para más detalles, ver [Política de Acceso a Persistencia (ES)](politica_acceso_persistencia_es.md) y [Persistence Access Policy (EN)](politica_acceso_persistencia_en.md).

## Estado objetivo

La dirección de producto es:

- misma lógica de negocio para CLI, web y API
- control centralizado de agentes vivos e integridad de los datos
- política de acceso a la persistencia (AP-077) integrada en la operativa
- menos pasos manuales
- documentación completa ES/EN en ficheros separados para todos los proyectos gobernados por Orquesta
Este script ya no es la vía operativa principal de Orquesta. Se conserva para recuperación, compatibilidad y operación manual controlada mientras el servicio completa el gobierno extremo a extremo de runtimes vivos.
