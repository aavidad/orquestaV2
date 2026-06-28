# Contexto del Goal residente de Orquesta

Eres un Codex Goal lanzado por Orquesta para terminar Orquesta y sus conectores
en un entorno remoto aislado. Codex ejecuta; Orquesta dirige. No eres una sesion
manual permanente por SSH ni debes saltarte el contrato recibido desde
Orquesta.

## Objetivo

Madurar Orquesta hasta que pueda autoprogramarse, validar sus cambios y dejar
sus conectores principales listos: nucleo, Codex Goal, web, autoprogramacion,
conectores OPES, DomainWork/external-work y documentacion operativa.

## Frontera dura

- Trabaja solo dentro del repo Orquesta montado en `/workspace/project`.
- El estado, runtime, caches y artefactos de prueba viven bajo `/workspace`.
- No toques `uso-app`, OPES productivo, postgres productivo, nginx, servicios
  existentes ni `/home/berserk/deploy/opes`.
- No abras puertos externos. La web solo existe por loopback del host y tunel
  SSH.
- No subas nada a produccion ni ejecutes deploys reales.
- No borres artefactos de pruebas de temario o conectores: conservalos para
  revision y posible promocion manual posterior.
- Si necesitas probar OPES/temarios, usa fake, temporal o staging aislado bajo
  `/srv/orquesta-self` y documenta exactamente el alcance.

## Runtime

- Usa siempre `ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`.
- No uses `stdio`.
- No uses `app_server_proxy`.
- No caigas al loop legacy salvo que una prueba existente lo exija de forma
  explicita y aislada.
- Si falta Codex auth, tmux, socket o cuota, documenta el bloqueo y deja tarea
  accionable. No inventes otro transporte.
- Si Orquesta no puede construir o validar el `GoalWorkSpec`, no arranques
  trabajo manual equivalente: documenta el bloqueo para que Orquesta se repare.

## Forma de trabajo

- Mantén write-set estrecho por cambio.
- No empieces de cero: lee primero este contexto, `AGENTS.md`, los documentos
  vigentes citados por `AGENTS.md` y las incidencias `TAREA_OPES_ORQUESTA_*.md`
  que sigan abiertas.
- Prioriza arreglos que hagan Orquesta autosuficiente con Goal.
- Toca conectores OPES, web o programacion cuando sea necesario para cerrar
  Orquesta, pero las pruebas deben ser aisladas y sin promocion productiva.
- Ejecuta pruebas focales antes de cerrar cada cambio; para cambios
  transversales usa `git diff --check` y `go test -count=1 ./...` cuando sea
  viable.
- Deja commits locales claros; no hagas push ni despliegues productivos sin
  orden explicita del operador.
- Conserva evidencias compactas: cambios, pruebas, bloqueos y rutas de
  artefactos de prueba.

## Siguientes cierres

Trabaja en este orden salvo bloqueo real documentado:

1. Cerrar el perfil remoto aislado: servidor arrancado en contenedor, puerto
   solo `127.0.0.1:19039`, binds solo bajo `/srv/orquesta-self`, estado visible
   en `/api/v0/server/status` y sin variables OPES/DomainWork productivas.
2. Cerrar Goal por tmux: `app_server_tmux` debe lanzar, observar y cerrar una
   tarea real acotada. `stdio` y `app_server_proxy` no se usan en goal-first; si
   queda codigo legacy, debe estar documentado como historico o aislado por
   tests.
3. Cerrar autoprogramacion autosuficiente: Orquesta debe leer incidencias,
   convertirlas en `GoalWorkSpec`, ejecutar pruebas focales, conservar
   evidencia y producir commit local sin que el operador programe a mano salvo
   desbloqueo.
4. Cerrar reparacion automatica: si una prueba, smoke o validacion falla,
   Orquesta debe crear tarea causal de reparacion, relanzar con contexto
   compacto y documentar bloqueo solo si falta auth, cuota, runtime o permiso.
5. Cerrar web de Nueva App: ayuda por hover, mensajes de validacion en
   castellano, wizard conversacional que orienta hasta contrato terminado,
   opcion experto de datos/integraciones, arquitecturas amplias con hexagonal
   por defecto cuando encaje, calidad/accesibilidad seleccionable en ejecucion y
   manual profundo de opciones.
6. Cerrar conectores OPES/DomainWork en aislado: usar fakes, temporales o rutas
   bajo `/srv/orquesta-self`; no tocar OPES productivo. Las pruebas de temario
   conservan artefactos para revision y no promocionan produccion.
7. Cerrar documentacion operativa: actualizar runbooks, matriz de pruebas,
   estado actual y docs de arquitectura solo con evidencia real. Lo historico se
   marca como historico, no se borra sin trazabilidad.

## Criterio de cierre

No declares Orquesta terminada solo porque una prueba concreta pase. Debe quedar
claro que:

- el servidor remoto sigue aislado;
- los conectores no pueden tocar produccion por accidente;
- Goal/tmux funciona sin stdio;
- las tareas de autoprogramacion producen cambios verificables;
- los fallos se convierten en incidencias o nuevas tareas causales.
