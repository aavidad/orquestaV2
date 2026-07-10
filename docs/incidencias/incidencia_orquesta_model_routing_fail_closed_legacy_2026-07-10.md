# Incidencia: 208K model routing invalida configuraciones existentes

Fecha: 2026-07-10. Estado: abierto.

## Sintoma

La rama `fix/model-routing-p0-p1-sol-20260710` incorpora la politica de
modelos/esfuerzo, pero trata `model_routing` ausente como conector invalido.
Eso rompe configuraciones y fixtures existentes antes de que puedan aplicar la
politica conservadora por defecto.

Reproduccion ejecutada en ese worktree:

```bash
go test -count=1 \
  ./modulos/orquesta-capacity \
  ./modulos/orquesta-runtime-codex \
  ./modulos/orquesta-runtime-claude \
  ./modulos/orquesta-app-codex-stack
```

Resultado: `orquesta-capacity` verde; fallan Codex con
`codex_connector_value_invalido: model_routing`, Claude con
`claude_goal_model_routing_invalid`/`claude_connector_value_invalido:
model_routing`, y flujos del stack que ya no llegan a lanzar u observar.

## Impacto

La regla de ahorro de tokens no puede convertir una configuracion opcional en
un veto de ejecucion. Sin una normalizacion canónica, la introduccion de la
politica hace fallar smokes, lifecycle y tests que no declaraban routing por
ser anteriores al contrato. Esto contradice la regla de no cortar trabajo
recuperable por una forma omitida.

La auditoria del codigo confirma el mecanismo: `modelRoutingPolicyFromConfigV0`
crea refs y esfuerzos por defecto, pero `modelRoutingAliasesV0` devuelve un
mapa vacio cuando el fichero no declara aliases. `exactModelRoutingAliasV0`
rechaza entonces el resultado; Codex y Claude duplican despues ese veto al
validar sus receipts. La normalizacion debe producir politica y aliases
canónicos juntos, en la composicion comun, antes de los dos resolvers.

## Correccion requerida

Definir una unica normalizacion de composicion: cuando no haya routing
explicito, materializar politica conservadora (`trivial/low`,
`normal/medium`, `complejo/high`, `critico/high`; `xhigh` solo con
autorizacion causal). Solo valores contradictorios o un `xhigh` no autorizado
deben devolver error tipado. Los resolvers Codex y Claude deben consumir el
mismo resultado normalizado; no duplicar defaults ni convertir ausencia en
`connector_value_invalido`.

## Revision posterior del WIP

La revision del worktree `wip/remote-main-20260710` confirma que el problema
sigue presente con nombres nuevos: `codexModelRoutingFromProjectConfigFileV0`
y `claudeModelRoutingFromProjectConfigFileV0` generan la politica tipada por
defecto, pero inicializan `ModelAlias` desde el fichero y por tanto lo dejan
vacio sin seccion explicita. Los dos resolvers buscan despues el
`SelectedModelRef` en ese mapa y bloquean el launch.

La correccion debe ser precisa: si la seccion de routing esta completamente
ausente, el adaptador de composicion materializa los tres aliases canonicos
del proveedor junto a la politica por defecto. Si la seccion esta presente,
un alias faltante sigue siendo configuracion parcial y debe fallar cerrado.
Asi no se reintroduce una eleccion por `PATH`, env global o herencia de padre.

## Criterio de cierre

- Los paquetes de la reproduccion pasan con configuracion legacy sin
  `model_routing` y con routing explicito.
- Hay pruebas adversariales de `xhigh` no autorizado y de valor contradictorio.
- La politica efectiva queda en receipt/evidencia, sin introducir proveedor ni
  modelo en el nucleo neutral.
- El autor publica un commit y revisor reejecuta los paquetes afectados.
