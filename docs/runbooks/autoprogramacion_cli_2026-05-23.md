# Autoprogramacion CLI - 2026-05-23

## Alcance

Este runbook valida el borde CLI de autoprogramacion. El binario es un adaptador
fino: parsea argumentos, lee JSON de entrada, invoca clientes REST existentes y
emite `CliOutputEnvelopeV0` automatizable. No decide negocio, no persiste estado,
no lee DB, no arranca runtime y no usa fallback local.

Write-set del corte:

- `modulos/orquesta-cli`
- `cmd/orquesta-cli`
- `docs/runbooks/autoprogramacion_cli_2026-05-23.md`

## Comandos cubiertos

- `app spec solicitar`
- `app spec bootstrap`
- `doctor contratos`
- `contratos funcion listar`
- `contratos funcion ver`
- `contratos funcion registrar` bloqueado por contrato
- `gobernanza catalogo listar`
- `gobernanza catalogo ver`

## Contrato externo

La CLI consume solo rutas publicas versionadas:

- `POST /api/v0/apps/spec`
- `POST /api/v0/director/bootstrap/appspec`
- `POST /api/v0/operational-status/query`
- `POST /api/v0/core/function-contracts/list`
- `POST /api/v0/core/function-contracts/view`
- `POST /api/v0/governance/catalog/query`

Todas las llamadas REST propagan `X-Correlation-ID`, usan JSON y devuelven
envelopes publicos sin cuerpos privados de errores remotos. Las refs son opacas:
la CLI solo transporta `project_ref`, `app_spec_ref`, `function_contract_ref` y
refs equivalentes declaradas por contratos propietarios.

## Fronteras

- `cmd/orquesta-cli` solo delega en `RunOrquestaCLIV0`.
- `modulos/orquesta-cli` contiene parseo tecnico, lectura de JSON y clientes por
  contrato.
- Las reglas de dominio permanecen en `orquesta-factory`, `orquesta-director`,
  `orquesta-observability`, `orquesta-core` y `orquesta-governance`.
- `registrar FunctionContractV0` conserva el bloqueo publico
  `registrar_function_contract_bloqueado`.

## Validacion del corte

Ejecutar desde la raiz del repo:

```bash
go test -count=1 ./modulos/orquesta-cli
```

Criterios de aceptacion manual:

- El binario no reintroduce control-plane legacy.
- No hay imports de DB, runtime, HOME, tokens ni proveedores reales en la CLI.
- Los comandos mutantes usan contratos publicos y claves idempotentes cuando el
  contrato las declara.
- Los comandos read-only no mutan estado ni activan reglas.
- La ayuda y errores visibles usan espanol y codigos/i18n estables.
- Los argumentos posicionales sobrantes fallan como `opcion_invalida`.

## Resultado 2026-05-23

Validado con la bateria focal obligatoria del paquete:

- `go test -count=1 ./modulos/orquesta-cli`
- `validar criterios de aceptacion del cambio`
- `validar contrato externo de dominio`
