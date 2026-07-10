# Incidencia: 208K model routing invalida configuraciones existentes

Fecha: 2026-07-10. Estado: cerrado localmente en `4faf2fd2f`.

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

## Correccion local BUG-208K

En `fix/model-routing-legacy-aliases-20260710`, los campos de proyecto de
routing pasan a ser punteros: solo un bloque JSON ausente se materializa con
los aliases canonicos de composicion. Para Codex son `luna -> gpt-5.6-luna`,
`terra -> gpt-5.6-terra` y `sol -> gpt-5.6-sol`; para Claude,
`haiku -> haiku-4.5`, `sonnet -> sonnet-5` y `fable -> fable-5`.

Un bloque presente, incluso `{}`, no recibe aliases: conserva el rechazo
fail-closed del resolver. Las pruebas focales tambien conservan la exigencia
causal para critical y `xhigh`.

Evidencia local 2026-07-10 ejecutada con caches aisladas bajo
`/tmp/orquesta-review-208k-cache`:

```bash
env GOTOOLCHAIN=local GOCACHE=/tmp/orquesta-review-208k-cache/go-cache GOTMPDIR=/tmp/orquesta-review-208k-cache/go-tmp GOMODCACHE=/tmp/orquesta-review-208k-cache/go-mod go test -count=1 \
  ./modulos/orquesta-capacity \
  ./modulos/orquesta-runtime-codex \
  ./modulos/orquesta-runtime-claude \
  ./modulos/orquesta-app-codex-stack \
  ./cmd/orquesta-server
```

Resultado: compilan y pasan `orquesta-capacity`, `orquesta-runtime-codex` y
`orquesta-runtime-claude`; desaparecen los tres errores de compilacion 208K.
La ejecucion conjunta sigue roja por fallos ajenos ya presentes en
`orquesta-app-codex-stack` (descriptores de uso vacios y flujos que quedan en
`wait_unhandled_outbox`/`programacion`). No constituye cierre de la incidencia.

## Criterio de cierre

- Los paquetes de la reproduccion pasan con configuracion legacy sin
  `model_routing` y con routing explicito.
- Hay pruebas adversariales de `xhigh` no autorizado y de valor contradictorio.
- La politica efectiva queda en receipt/evidencia, sin introducir proveedor ni
  modelo en el nucleo neutral.
- El autor publica un commit y revisor reejecuta los paquetes afectados.

## Verificacion independiente del candidato

El revisor ejecuto en el host, con Go 1.25.11 y caches aisladas bajo
`/tmp/orquesta-review-208k-cache`, sin depender de la red de los sandboxes:

```bash
go test -count=1 \
  ./modulos/orquesta-capacity \
  ./modulos/orquesta-runtime-codex \
  ./modulos/orquesta-runtime-claude \
  ./modulos/orquesta-app-codex-stack

go test -count=1 ./cmd/orquesta-server \
  -run 'Test(Codex|Claude)ModelRoutingConfigV0|TestCodexRuntimeConfigV0'
```

Ambos comandos terminaron verdes. El paquete completo
`orquesta-app-codex-stack` cubre ahora tambien el fixture OPES de `xhigh`: la
ruta declara de forma causal nivel, motivo, evidencia y autorizacion, en vez
de heredar el esfuerzo desde un campo runtime legacy. La ejecucion completa de
`cmd/orquesta-server` arranca fakes amplios ajenos al write-set y se conserva
como verificacion transversal posterior; no se usa como falso requisito ya
cumplido para cerrar esta incidencia.

## Cierre local

El candidato se integro en `trabajo/plataforma-agentes` como `4faf2fd2f`.
El revisor repitio sobre esa rama, con caches aisladas bajo
`/tmp/orquesta-review-208k-main-cache`:

```bash
go test -count=1 \
  ./modulos/orquesta-capacity \
  ./modulos/orquesta-runtime-codex \
  ./modulos/orquesta-runtime-claude \
  ./modulos/orquesta-app-codex-stack

go test -count=1 ./cmd/orquesta-server \
  -run 'Test(Codex|Claude)ModelRoutingConfigV0|TestCodexRuntimeConfigV0'
```

Todo verde. El cierre cubre el defecto 208K: compatibilidad de routing ausente,
rechazo de configuracion parcial, ausencia de herencia global y autorizacion
causal de `xhigh`. No acredita por si solo pendientes de deploy, atestacion o
limpieza de variables, que conservan sus entradas separadas.

## Reparacion mecanica de expectativas 2026-07-10

Se actualizaron las pruebas focales para reflejar que
`ORQUESTA_CODEX_REASONING_EFFORT` no inicializa el runtime global y que el
routing Claude parcial falla cerrado mediante error con modelo vacio, sin
exigir `decision.Rejected`.

La prueba solicitada no pudo ejecutarse: el entorno no pudo descargar Go
1.25.11 y, usando el toolchain local, tampoco pudo descargar
`golang.org/x/text@v0.38.0`; la red esta bloqueada (`proxy.golang.org`,
`network is unreachable`). `git diff --check` pasa.

## Reparacion del helper de stack 2026-07-10

`codexStackBaseConfigForTestV0` ahora declara el routing tipado canonico de
Codex: politica estricta con refs `luna/terra/sol`, esfuerzos
`low/medium/high/high`, aliases `gpt-5.6-luna/terra/sol` y rutas por tarea
vacias. Conserva `Model` y `ReasoningEffort` legacy para fixtures antiguas;
el stack debe resolver por `ModelRouting`.

La prueba focal solicitada con caches bajo `/tmp` no pudo arrancar porque el
entorno no permite descargar `golang.org/x/text@v0.38.0` desde
`proxy.golang.org` (DNS/socket bloqueado). `git diff --check` pasa. El bug
208K sigue abierto hasta una ejecucion con dependencias disponibles.

## Normalizacion de frontera 2026-07-10

`BuildStackV0` normaliza ahora, antes de validar o cablear perfiles, solo los
`CodexModelRoutingConfigV0` y `ClaudeModelRoutingConfigV0` Go completamente
cero. Materializa las politicas estrictas y aliases canonicos de la
composicion para Codex (`luna/terra/sol`) y Claude (`haiku/sonnet/fable`). Los
maps vacios pero inicializados y cualquier otro bloque parcial no son cero: no
reciben defaults y el resolver conserva el rechazo fail-closed. Los defaults
del servidor delegan en la misma composicion para no duplicar la politica.

Se añadieron focales para el caso cero y el parcial. La bateria requerida
`go test -count=1 ./modulos/orquesta-app-codex-stack`, con caches aisladas en
`/tmp/orquesta-208k-model-routing-cache`, no llego a compilar por la misma
dependencia indisponible (`golang.org/x/text@v0.38.0`); no acredita cierre.
