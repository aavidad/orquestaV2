# Auditoría de variables de entorno que se pisan — 2026-07-04

Autor: Claude (director), a petición del operador tras el incidente T7A
(`umbral pedido 450000, efectivo 100000 defaulted`). Alimenta la TAREA-8
(config canónica) de `docs/instrucciones_director_codex_2026-07-04.md`.

## Método

- Registro central: 236 envs `ORQUESTA_*` en
  `cmd/orquesta-server/server_env_registry_v0.go`.
- Lectura real: `os.Getenv` en cmd/ y modulos/ (fuera de tests).
- La métrica de deuda cuenta 511 nombres en total: la diferencia son envs
  leídas FUERA del registro central (guardian, smokes, scripts) — primera
  conclusión en sí misma.

## Pisadas confirmadas (mismo concepto, varios nombres)

1. **Home de Codex, TRES variantes**: `ORQUESTA_CODEX_CODE_HOME` (la
   canónica que usan los pilotajes), `ORQUESTA_CODEX_HOME`
   (`codex_wave_config_v0.go`) y `CODEX_HOME` sin prefijo
   (`codex_env_v0.go`). Riesgo real: un operador exporta una y el
   componente lee otra → mismo patrón que T7A. Acción: una canónica +
   alias con aviso `deprecated_env_used`.
2. **Base URL de OPES, dos variantes**: `ORQUESTA_OPES_BASE_URL` y
   `OPES_BASE_URL` (`opes_bridge_config.go`). Los scripts de pilotaje ya
   exportan AMBAS "por si acaso" — evidencia de que la duplicidad confunde.
3. **Timeouts con unidades mezcladas**: `ORQUESTA_CODEX_GOAL_TIMEOUT_MS`
   (milisegundos) convive con `ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS`,
   `SMOKE_CLAUDE_GOAL_PROCESS_TIMEOUT_SECONDS` y
   `SMOKE_GEMINI_GOAL_PROCESS_TIMEOUT_SECONDS` (segundos). Trampa de x1000
   esperando a ocurrir. Acción: sufijo de unidad único (_MS o _SECONDS) en
   toda la superficie; alias temporales.
4. **Familia fuera del registro central**: `cmd/orquesta-guardian` lee ~27
   `ORQUESTA_GUARDIAN_*` directamente de `os.Getenv` sin pasar por el
   registro → invisibles para `effective_config` y para el eco de
   configuración; los smokes (`SMOKE_*`) igual. Acción: registrarlas o
   declararlas explícitamente "solo-proceso-hijo".

## Familias con mayor superficie (candidatas a sección de fichero, no envs)

| Familia | nº envs |
|---|---|
| OPES_BRIDGE | 47 |
| CODEX_WAVE | 26 |
| SERVER_IDLE | 21 |
| OPES_REGISTRY | 15 |
| CODEBASE_BROKER | 10 |
| DOMAIN_WORK | 9 |

Estas 6 familias suman ~128 envs (~25% del total): son configuración de
componente, no operativa; deben vivir como secciones del fichero canónico
(TAREA-8.4) y desaparecer como envs.

## Relación con T7A

El umbral `ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS`
está registrado y su lectura es única (sin pisada de nombre); su fallo fue
de PROYECCIÓN (daemon no hereda el env del shell) — clase distinta, cubierta
por TAREA-8.1/8.2 (fichero canónico + guard `config_projection_mismatch`).
Las pisadas de este documento son la otra mitad de la clase: mismo valor,
nombres distintos. Ambas se cierran con la misma medicina: fuente única
tipada + eco por setting + ratchet bidireccional (8.3).

## Acciones entregadas a Codex

- TAREA-8.4 usa esta auditoría como lista inicial de la ola 1:
  consolidar las 3 variantes de codex home, las 2 de OPES base URL y
  normalizar unidades de timeout (alias + aviso, sin romper).
- TAREA-8.3 añade el caso (d): env leída fuera del registro = rojo, salvo
  lista blanca explícita "solo-proceso-hijo".
