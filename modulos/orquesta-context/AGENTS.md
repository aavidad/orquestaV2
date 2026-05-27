# Contexto Codex: orquesta-context

Lee primero este archivo y `README.md`. Despues lee solo los docs locales necesarios para tu microtarea.

## Reglas comunes

- Contexto pequeno: este modulo prepara bundles pequenos, no prompts gigantes.
- Hexagonal siempre: filesystem, MCP, runtime, LLM y proveedores son adaptadores.
- i18n por defecto cuando haya texto de UI, app o documentacion generada.
- Los bundles pasan refs opacas y manifiestos versionados; no transportan HOME, OAuth, tokens, transcripts ni prompts completos.
- Problema grande: descomposicion primero; microtarea pequena despues.
- Funciones pequenas, nombres claros y tests de invariantes.
- No cruces `internal/`, structs privados ni detalles de proveedor de otro modulo.
- Si necesitas informacion o decision de otro grupo, emite `CONSULTA AL DIRECTOR`.

## Alcance local

Trabaja en:

- contratos de `ContextBundleV0`;
- clasificacion por fase/modulo/tarea;
- manifiestos pequenos para agentes;
- politica de consulta al director cuando falte contexto externo;
- limites de tamano y seguridad de refs.

## Responsabilidad y capa

Responsabilidad: construir contexto pequeno, versionado y verificable para
agentes. Capa: contrato neutral y materializacion por puertos; los adaptadores
externos leen disco, MCP, REST o runtime fuera del nucleo.

## Prohibido

- Leer filesystem productivo desde el nucleo del builder.
- Hardcodear Codex, proveedor, OAuth, HOME o motor de base de datos.
- Meter documentacion global completa en cada agente.
- Convertir el contexto en una sesion monolitica.
- Crear ficheros largos como centro de todo el sistema.
- Imports prohibidos: Codex, OPES, web, MCP, DB concreta, runtime real, HOME,
  OAuth, tokens, prompts o transcripts crudos dentro del builder neutral.

## Pruebas focales y docs vigentes

Usa:

```bash
go test -count=1 ./modulos/orquesta-context
```

Docs vigentes: `AGENTS.md`, `docs/estado_actual_2026-05-17.md`,
`docs/guia_nucleo_orquestacion_2026-05-17.md`, `docs/principio_orquesta_piensa_director.md`
y `modulos/orquesta-context/docs/*` cuando cambies contratos, pruebas o tareas
locales.

## Entrega

Cada cambio debe incluir:

- write-set pequeno;
- contrato o invariante afectada;
- prueba unitaria de clasificacion/limites;
- actualizacion de `docs/tareas.md` y `docs/pruebas.md` si cambia estado.
