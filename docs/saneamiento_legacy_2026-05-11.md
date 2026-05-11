# Saneamiento legacy 2026-05-11

## Decision

El producto activo queda acotado a `modulos/`.

El nucleo de orquestacion se movio de `orquestacionnucleoapp/` a
`modulos/orquesta-orchestration-core/` para que tambien tenga contexto local de
miniproyecto y no dependa de una carpeta raiz especial.

## Motivo

El arbol raiz mezclaba tres cosas incompatibles:

- nucleo modular nuevo;
- aplicacion v1/v2 con `cmd`, `db`, control-plane y runtime acoplados;
- fuentes externas completas usadas como referencia.

Esa mezcla hacia que `go test ./...` entrase en suites legacy con procesos
PTY/TMUX y tests de persistencia/control-plane que no representan el producto
nuevo. Eso reproducia el problema de v1/v2: mucho codigo en la misma superficie,
fallos cruzados y bucles de parcheo.

## Eliminado del producto activo

- `cmd/`, `db/`, `internal/` y paquetes raiz v1/v2 acoplados a persistencia,
  runtime y control-plane.
- scripts operativos antiguos, Docker/Compose antiguos y servicios host.
- backups de codigo bajo `backups_orchestration/`.
- repos externos completos (`oh-my-codex-main/`, `claw-code-dev-rust/`) que solo
  deben usarse como referencia documental, no como codigo vivo dentro de Orquesta.

## Conservado

- `docs/` como memoria historica y reglas del proyecto.
- `modulos/` como frontera activa de producto.
- `ARQUITECTURA.md` como indice historico/global.
- Tag `archive/legacy-root-before-purge-2026-05-11` para recuperar el estado
  anterior si se necesita extraer una idea concreta.

## Regla de reutilizacion

No se restaura legacy por merge completo.

Si hace falta algo de v1/v2 o de una referencia externa:

1. se extrae una capacidad concreta;
2. se adapta a hexagonal/i18n/conectores;
3. se introduce en un miniproyecto bajo `modulos/`;
4. se prueba con `go test ./...`;
5. se documenta la decision.

## Validacion

Tras la purga:

- `go test ./...`
- `go build ./...`

ambos pasan sobre la app activa.
