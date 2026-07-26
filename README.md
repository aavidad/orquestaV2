# Orquesta V2

Reconstrucción de Orquesta sobre un único lifecycle `Goal`, un único escritor
de aplicación y un único scheduler. El producto nuevo no importa, arranca ni
adapta el runtime anterior.

## Autoridad vigente

Lee en este orden:

1. `AGENTS.md`;
2. `docs/reconstruccion/LEEME_AGENTE_ORQUESTAV2.md`;
3. `product/roadmap.json`;
4. `product/capabilities.json` y `product/evidence/**`;
5. `docs/reconstruccion/ruta_total_100.md`.

Último handoff:
`docs/reconstruccion/HANDOFF_PARADA_ORQUESTAV2_2026-07-26.md`.

V1-V22 son verticales acreditadas de esta reconstrucción. V23 permanece abierto
hasta que su contrato y su receipt queden sellados; no se declara terminado por
documentación ni por un smoke parcial.

## Superficie de producto

- dominio y aplicación: `internal/goal`, `internal/application`;
- puertos y adaptadores: `internal/ports`, `internal/adapters`;
- interfaces y composición: `internal/interfaces`, `internal/bootstrap`;
- único binario productivo: `cmd/orquesta`;
- contratos públicos: `sdk`;
- catálogo, roadmap y evidencias: `product`;
- aceptación transversal: `acceptance`.

OPES, programación con Codex y otras aplicaciones son consumidores o
adaptadores. No definen el núcleo.

## Legado

El código anterior no está en la vista activa de este equipo. Se conserva
íntegro y consultable en:

```text
/home/alberto/Trabajo/orquestaV2-legacy-consulta
```

La copia, el bundle, las referencias de rescate y la reversión están descritos
en `docs/reconstruccion/separacion_legacy_consulta_2026-07-26.md`. El legado
sirve para extraer lecciones, fixtures y semántica; nunca como autoridad ni
como atajo de implementación.

## Verificación

```bash
git diff --check
go test -mod=vendor -count=1 ./...
```

Las pruebas reales y los smokes que requieren runtime, credenciales o
infraestructura son opt-in y deben ejecutarse según el contrato de la vertical
correspondiente.
