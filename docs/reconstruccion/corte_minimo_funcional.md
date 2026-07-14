# Corte mínimo funcional de la nueva Orquesta

Fecha: 2026-07-14

Estado: cerrado al 100 % de la definición mínima funcional. La suite raíz, el
E2E Codex real y todos los gates del árbol integrado pasaron el 2026-07-14.

## Resultado

Un único binario nuevo acredita este flujo por MCP real:

```text
intención -> Goal durable -> WorkItem -> ejecución -> artefacto
          -> atestación -> cierre durable
```

`Goal` y `WorkItem` son la única autoridad de lifecycle. Eventos, outbox,
dashboard y estado de ejecución son auditoría o proyecciones.

## Frontera del corte

- operador único `actor:local-owner`;
- proyecto inicial `project:default`;
- identidad y proyecto viajan como refs desde el primer comando;
- SQLite es adaptador de estado local y sustituible;
- filesystem content-addressed es adaptador de artefactos;
- Codex es el primer adaptador real de agente;
- plataforma de release acreditada: Linux; la recolección del árbol de procesos
  usa grupos de proceso Unix;
- MCP Streamable HTTP es la entrada pública acreditada;
- textos humanos usan catálogo i18n con español por defecto;
- configuración no sensible, credenciales y proyección efectiva conservan
  fronteras distintas;
- reinicio no repite trabajo terminal y shutdown no deja procesos propios ni
  descendientes en Linux.

Fuera del corte: AD, OIDC, RBAC completo, varios usuarios, Forge, web rica,
Wizard rico, Consejo, Hermes, OPES y paridad de proveedores.

## Aislamiento

| Concepto | Valor |
|---|---|
| Worktree nuevo | `/home/alberto/Trabajo/orquesta-rebuild` |
| Rama | `reconstruccion/minimo-funcional-20260714` |
| Base | `7576f60bd3b5424a6c1c19e4f319cb634b12f231` |
| Árbol antiguo | `/home/alberto/Trabajo/orquesta`, solo lectura |
| Runtime/estado nuevo | ruta temporal o instalación propia, nunca legacy |

El producto nuevo solo puede escribir `internal/`, `cmd/orquesta/`, `config/`,
`product/`, esta documentación, dependencias raíz y su guard de write-set. No
puede importar `orquesta/modulos/...`.

## Autoridad y ejecución

Motor determinista interno valida y persiste. Agentes proponen o producen
resultados por puertos; nunca reciben el repositorio ni escriben lifecycle.
Usar Codex como adaptador no convierte Codex en núcleo.

El Orquesta antiguo permanece detenido. Durante este corte se usa coordinación
Codex directa como excepción acotada: reactivar el runtime antiguo contradiría
la orden de aislamiento y volvería a crear trabajo sobre la autoridad que se
está sustituyendo.

## Definición de 100 %

`product/capabilities.json` es el manifest del corte. Cierre solo cuando:

1. cada `acceptance_ref` existe y pasa;
2. arquitectura impide imports legacy/concretos desde dominio/aplicación;
3. suite completa, race focal y `git diff --check` quedan verdes;
4. E2E usa cliente y servidor MCP reales;
5. E2E Codex real produce artefacto y Goal terminal acreditado;
6. restart/replay no duplica ejecución, artefacto ni cierre;
7. shutdown deja cero procesos y descendientes del producto nuevo en Linux;
8. árbol antiguo conserva su diff original.

No se usará porcentaje subjetivo.

La arquitectura y la operación se describen en
[`guia_arquitectura_operacion.md`](guia_arquitectura_operacion.md). El estado
de cada contrato y la evidencia ejecutada están en
[`mapa_aceptacion_evidencias.md`](mapa_aceptacion_evidencias.md). La evolución
posterior al corte, sin reescribir el núcleo, queda en
[`progresion_futura.md`](progresion_futura.md).

## Evidencia del 2026-07-14

- `go test -mod=vendor -count=1 ./...`: `PASS` sobre todo el repositorio.
- E2E real Codex por servidor MCP de producción: `PASS` en 3,54 s, con request
  y marcador criptográficamente únicos, autenticación Bearer, SQLite, CAS,
  artefacto, atestación y cierre leídos de vuelta por el cliente MCP oficial.
- Receipt ligado al digest de fuentes:
  `sha256:c674e4ce75f40b73759dcd960095ac8774693a937b70b5f29c9e0b3b7b92ce89`.
- `race` focal, `vet`, `git diff --check` y guard de write-set: `PASS`.
- Contrarrevisión final independiente de runtime, almacenamiento, SQLite y
  contratos: cero P0/P1 abiertos tras cerrar las regresiones de patrón MCP y
  symlink en el blob CAS final.

## Fuentes del estudio

Las fuentes completas permanecen aisladas en el árbol de estudio hasta su
integración documental limpia:

- catálogo: SHA-256
  `351c2258562424fc5c0dab9e0dcce0b623f9eabf42f2485b6ae45432694d4fd3`;
- ruta: SHA-256
  `483d947532d81841fa9caf58a176400552d2ea5482e8f93be375c07e4ae62b73`;
- orden de auditoría: SHA-256
  `252a84fdf5c68e61def6ae7d1e44d1039c1ca9297385af271d3fda7aed145d13`.
