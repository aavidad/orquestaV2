# Léeme antes de trabajar en Orquesta V2

Fecha: 2026-07-26.

Handoff vigente:
`HANDOFF_PARADA_ORQUESTAV2_2026-07-26.md`.

## Regla de un minuto

Este directorio contiene la reconstrucción nueva. No continúes, arregles ni
importes el runtime legacy.

```text
PRODUCTO NUEVO              CONSULTA LEGACY
internal/**                 /home/alberto/Trabajo/orquestaV2-legacy-consulta
cmd/orquesta/**             modulos/**
sdk/**                      cmd/orquesta-server/**
config/**                   Directores V1/V2 y Goal-first clásico
product/**                  scripts, smokes y documentos antiguos
acceptance/**
```

Si una búsqueda propone `orquesta/modulos/...`, `cmd/orquesta-server`,
`GoalWorkSpecV0`, `PlanState`, `WorkflowTaskV0` o `legacy_director_loop`, estás
leyendo legado. Detente y vuelve a la superficie nueva.

## Autoridad nueva

- lifecycle único: `internal/goal.Goal`;
- escritor único: `internal/application.Orchestrator`;
- scheduler único: composición de `internal/bootstrap`;
- binario productivo único: `cmd/orquesta`;
- configuración única: `config/registry.json`;
- contrato de producto: `product/roadmap.json`;
- evidencia de cierre: `product/evidence/**`.

`WorkItem`, ejecuciones, mailbox, reviews, council y effects pertenecen al mismo
Goal; no son Directores ni ciclos de vida alternativos.

## Estado real

- el roadmap contiene 37 verticales nuevas;
- V1-V22 tienen contrato ejecutable y están acreditadas;
- V23-V37 son 15 contratos planificados; V23 sigue abierto y solo cubre
  Wizard, intake, dossier, confirmación y
  generación/flujos declarados por `AC-V23-WIZARD`;
- Firecracker no es gate de V23. Permanece opt-in y diferido según
  `corte_alcance_v23_firecracker_diferido_2026-07-26.md`;
- no declares el repositorio verde por una compilación parcial, un documento o
  un smoke sin receipt.

## Qué significa “sin dependencia del legacy”

El producto nuevo no importa, arranca, adapta ni comparte estado con
`modulos/**` o `cmd/orquesta-server/**`.

Eso no significa que las 37 verticales sean independientes entre sí ni que
Orquesta no use infraestructura externa:

- las verticales tienen dependencias causales explícitas en el roadmap;
- DB, Git, proveedores de agentes, identidad, Forge, OPES, Firecracker y otras
  tecnologías entran por puertos y adaptadores opt-in;
- V35-V37 califican una app de videojuegos externa como consumidora; no la
  convierten en parte del núcleo;
- una dependencia externa solo acredita la vertical que la declara cuando su
  adapter y su E2E real pasan. No contamina el dominio ni autoriza un fallback
  al legacy.

La afirmación correcta es: **V1-V37 pertenecen al diseño nuevo y no dependen
del runtime legacy; V1-V22 están acreditadas y V23-V37 todavía no**.

## Legado y recuperación

En este equipo, `sparse-checkout` mantiene fuera de la vista el código y los
tests inequívocamente legacy. No se borraron:

- copia navegable:
  `/home/alberto/Trabajo/orquestaV2-legacy-consulta`;
- bundle:
  `/home/alberto/Trabajo/orquestaV2-backups/2026-07-26-pre-separacion-legacy/orquestaV2-completo.bundle`;
- rama y tag:
  `backup/pre-separacion-legacy-20260726`.

No desactives el sparse-checkout ni copies paquetes desde la consulta. Si una
lección antigua aporta valor, exprésala como contrato o test de la arquitectura
nueva.

## Gate antes de afirmar “compila”

Ejecuta:

```bash
git status --short --branch
go test -mod=vendor -run '^$' ./...
```

El segundo comando compila todos los paquetes visibles sin ejecutar la suite.
Solo si termina con código cero puede afirmarse que el árbol activo compila.

Para afirmar “todo verde” hay que ejecutar los tests exigidos por la vertical y
la verificación transversal aplicable. “Compila” y “suite completa verde” no
son equivalentes.

## Prohibiciones prácticas

- no materializar `modulos/**` para solucionar un import;
- no lanzar `cmd/orquesta-server`;
- no ejecutar smokes Goal-first o scripts del servidor clásico;
- no tratar documentos históricos o el inventario antiguo de bugs como backlog;
- no hacer depender V23 de Firecracker, KVM, launcher o microVM;
- no editar la copia de consulta.
