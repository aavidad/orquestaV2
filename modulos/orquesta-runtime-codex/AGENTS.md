# Contexto Codex: orquesta-runtime-codex

## Reglas comunes

- Contexto pequeno: usa este modulo y solo los contratos publicos citados.
- Hexagonal: este modulo es un conector externo, no una extension del core.
- i18n por defecto cuando haya texto de usuario; los errores tecnicos usan `message_key`.
- No accedas a DB ni a tablas.
- No importes `internal/` de otro modulo.
- No hardcodees HOME, OAuth, modelo, proveedor, cuenta, token ni ruta de Codex.
- Si hace falta una decision de otro modulo, emite `CONSULTA AL DIRECTOR`.

## Alcance

Resolver de agente externo para Codex CLI. Convierte una `ExternalAgentLaunchSpecV0` opaca en un `ProcessRuntimeLaunchRequestV0` ejecutable mediante configuracion opt-in del operador.

## Reglas

- Codex es un conector, no una dependencia del nucleo.
- El core nunca recibe command paths, HOME, OAuth, modelos ni secretos.
- El paquete de agente es `AgentStartPacketV0`; el agente debe leerlo y responder con ACK.
- El wrapper operacional puede contener rutas opt-in, pero nunca debe serializarse como contrato de core.
- Ficheros pequenos y funciones pequenas.

## Prohibido

- Defaults de proveedor/modelo/HOME.
- Heredar `PATH`, `HOME` o entorno completo desde Orquesta.
- Escribir fuera del runtime/project workdir recibido por configuracion explicita.
- Marcar un proceso controlado como cobertura de agente real.
