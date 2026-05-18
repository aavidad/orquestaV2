# Contexto para agentes: orquesta-document-plan-expander

Lee este fichero antes de trabajar en este directorio.

## Reglas

- Mantener el paquete puro y determinista.
- No importar adaptadores concretos, Codex, runtime, web, MCP, DB, filesystem
  productivo, red ni proveedor/modelo.
- La unica dependencia de Orquesta prevista es `orquesta-domain-work`.
- El expander no ejecuta jobs ni decide contenido: solo convierte un
  `DomainDocumentPlanV0` validado en `DomainWorkJobRequestV0` derivados.
- No inventar taxonomia ni reglas de una app externa. El plan ya debe
  traer work kinds y refs opacas.
- No ocultar errores de validacion: si el plan o un job derivado no valida, el
  resultado debe volver sin jobs.

## Verificacion minima

```bash
go test -count=1 ./modulos/orquesta-document-plan-expander
```

Para cambios de frontera:

```bash
go test -count=1 ./...
git diff --check
```
