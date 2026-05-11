# Contexto Codex: orquesta-capacity

## Reglas comunes

- Contexto pequeno: usa este modulo y solo contratos globales citados por la tarea.
- Hexagonal: conecta por puertos, DTOs/eventos y conectores.
- i18n por defecto cuando haya texto de UI, app o documentacion generada.
- Persistencia, runtime, LLM, filesystem y deploy son adaptadores.
- Problema grande: descomposicion primero; microtarea pequena despues.
- No cruces `internal/`, tablas, structs privados ni detalles de proveedor de otro modulo.
- Si necesitas informacion o decision de otro grupo, emite `CONSULTA AL DIRECTOR`.
- Al arrancar aqui, lee `README.md` y los docs locales necesarios en `docs/`.

## Alcance

Trabaja en politicas de modelo, razonamiento, pools, cuotas y escalado.

## Reglas

- No existe tabla estatica rol -> modelo.
- `xhigh` es excepcional y requiere evidencia.
- La cuota debe distinguir real, estimada y obsoleta.
- Los modelos locales se habilitan por pruebas y score, no por deseo.

## Prohibido

- Inventar cuota como verdad.
- Atar politica a un proveedor.
- Saltar handoff si la ventana no alcanza.
