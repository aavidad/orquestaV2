<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Arquitectura de resolución de modelos y gestión de flota

## Estado de este documento

Obsoleto como fuente de diseño activa desde `2026-04-11`.

Motivo:

- este documento empujaba una estrategia local centrada en Qwen como preferencia operativa
- esa recomendación ya no coincide con la smoke canónica de Orquesta ni con la doctrina actual de microprogramación dirigida
- mantenerlo como si siguiera vigente haría que otros agentes vuelvan a abrir código en una dirección ya descartada

## Sustitución canónica

La referencia vigente pasa a ser:

- `docs/BIBLIA_APP_ORQUESTA.md`
- `docs/diseno_microprogramacion_dirigida_agentes.md`
- `docs/matriz_modelo_y_razonamiento.md`
- `docs/orquesta_v1_roadmap.md`

## Decisión operativa vigente

- `gemma4:26b` es la opción local preferente y la única aprobada por ahora en la smoke canónica servida por Orquesta
- la vía actual `ollama-cli + tmux` se conserva como compatibilidad y experimentación
- la vía canónica objetivo para local es `pool + slots + agentes lógicos por perfil_tarea`
- Qwen y otros modelos locales siguen siendo compatibles, pero quedan en estado experimental mientras no superen la smoke canónica por la app
