# Contexto local: orquesta-director-agent-workflow

Lee este archivo antes de trabajar en este modulo.

## Reglas

- Adaptar decisiones compactas del director a comandos publicos del workflow.
- No lanzar procesos, no leer ficheros, no persistir y no elegir proveedor,
  modelo, HOME ni credenciales.
- Mantener funciones pequenas y tests locales.
- `orquesta-director-agent` sigue siendo puro; este modulo es el puente hacia
  `orquesta-core-workflow`.

## Validacion

```bash
go test -count=1 ./modulos/orquesta-director-agent-workflow
```
