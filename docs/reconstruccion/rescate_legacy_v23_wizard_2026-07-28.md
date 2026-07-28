# Rescate legacy para V23 Wizard — 2026-07-28

Estado: **inventario vigente de consulta; no acredita V23**.

Este corte compara en solo lectura el Wizard de OrquestaV2 con las copias
visibles bajo `/home/alberto/Trabajo`. No autoriza imports desde legacy,
dual-write, estados por canal ni promoción de capabilities. La fuente de
producto sigue siendo `/home/alberto/Trabajo/orquestaV2`.

## Resultado del barrido

`orquesta`, `orquesta-rebuild`, `orquesta-old-director-runtime` y
`orquesta-rebuild-worktrees/v23-wizard` contienen la misma generación útil del
Wizard. Los cuatro ficheros web principales del worktree V23 coinciden con el
legacy final; `wizard_v0.go` de `orquesta-app-director-intake` tiene el mismo
SHA-256 en las cuatro copias principales. `orquesta-autonomia-clean` solo
aporta un helper anterior, no otro Wizard completo.

No existe una solución V23 más avanzada escondida. Las ideas valiosas ya están
mayoritariamente reimplementadas con contratos más fuertes en OrquestaV2.

## Mapa de reutilización

| ID | Solución legacy | Estado en OrquestaV2 | Decisión |
| --- | --- | --- | --- |
| `WIZ-01` | `request_kind` y `project_source` distinguen app nueva, GitHub y ruta local. | Existe foundation neutral de análisis, todavía sin wiring público. | Adoptar contrato y negativos; no copiar DTOs web/factory. |
| `WIZ-02` | U1–U12, T1–T8, R1–R8 y hasta cuatro preguntas por turno. | Catálogos tipados, gaps causales y política de rondas externa. | Ya integrado con mejor frontera; no copiar. |
| `WIZ-05` | Aceptación conjunta de recomendaciones. | Comando y writer durable, atómico y con replay exacto. | No reutilizar el bucle legacy ni su límite mágico `12`. |
| `WIZ-06` | Inventario amplio de defaults técnicos. | Propuestas visibles y tipadas dentro del evaluador. | Conservar contenido útil; rechazar aplicación silenciosa. |
| `WIZ-09` | Sesión web mutable. | Revisión esperada, cadena durable y reapertura causal. | Descartar el mecanismo legacy. |
| `WIZ-10` | Solo valida la procedencia new/GitHub/local. | `internal/application/app_analysis*.go` recibe análisis verificable, aún aislado. | Legacy no analiza código: usar solo sus negativos de fuente. |
| `WIZ-13` | No hay branding o tema visual productivo. | Tampoco existe aún contrato Wizard específico. | Implementación nueva; no hay código que rescatar. |
| `WIZ-16` | Slot filling determinista y asistente opcional. | Texto libre ya es durable, pero falta la interacción pública completa. | Reutilizar contratos y tests; rehacer sobre el intake único. |
| `WIZ-17` | Ayuda, ejemplos, glosario ES/EN y corpus RAG. | Catálogo tipado con cobertura ES/EN; falta experiencia pública completa. | Reusar inventario editorial y casos de aceptación, no sesión. |
| `WIZ-19` | Modo determinista con puerto LLM opcional y degradación. | No hay bot Wizard público equivalente. | Adoptar la idea hexagonal; reescribir con autoridad/autorización V2. |
| `WIZ-20` | Exclusión por hechos y reevaluación. | Dependencias transitivas y revisiones causales más fuertes. | Ya integrado; no copiar. |
| `WIZ-25` | Packs combinables elegidos por palabras clave. | Packs explícitos, combinables, versionados y sin inferencia textual. | Conservar taxonomía; rechazar activación heurística. |
| `UI-05` | Wizard web por pasos, chat, ayuda y dossier. | Sin superficie web V23 final. | Adaptar UX y pruebas; no copiar el render monolítico. |

## Fuentes legacy útiles

- Motor, taxonomía y packs:
  - `/home/alberto/Trabajo/orquesta/modulos/orquesta-web/nueva_app_wizard_gaps_v0.go`
  - `/home/alberto/Trabajo/orquesta/modulos/orquesta-web/nueva_app_wizard_turn_v0.go`
  - `/home/alberto/Trabajo/orquesta/modulos/orquesta-web/nueva_app_wizard_types_v0.go`
- Bot, ayuda y glosario:
  - `/home/alberto/Trabajo/orquesta/modulos/orquesta-web/nueva_app_wizard_bot_v0.go`
  - `/home/alberto/Trabajo/orquesta/modulos/orquesta-web/nueva_app_wizard_help_i18n_v0.go`
  - `/home/alberto/Trabajo/orquesta/modulos/orquesta-web/nueva_app_wizard_universal_i18n_v0.go`
  - `/home/alberto/Trabajo/orquesta/docs/wizard_glosario_generado.md`
- Dossier:
  - `/home/alberto/Trabajo/orquesta/modulos/orquesta-web/nueva_app_wizard_dossier_v0.go`
- App nueva o existente:
  - `/home/alberto/Trabajo/orquesta/modulos/orquesta-factory/project_source_v0.go`
  - `/home/alberto/Trabajo/orquesta/modulos/orquesta-web/nueva_app_endpoint_post_v0_test.go`
- Diseño consolidado:
  - `/home/alberto/Trabajo/orquesta/docs/diseno_wizard_programacion_2026-07-04.md`

`modulos/orquesta-factory/backlog_v0.go` y
`modulos/orquesta-app-planner/plan_large_v0.go` solo conservan valor histórico:
los seis templates y su compilación tipada ya viven en
`internal/wizard/stages` e `internal/application/wizard_stage_plan.go`.

## Pruebas legacy que siguen siendo buena especificación

En `modulos/orquesta-web/nueva_app_wizard_turn_v0_test.go`:

- `TestWizardPacksCombinadosSinDuplicadosV0`;
- `TestWizardPreguntaQueEsRespondeYNoAvanzaV0`;
- `TestWizardReevaluaExclusionesAlCambiarRespuestaV0`;
- `TestWizardTodaOpcionTieneAyudaV0`;
- `TestWizardDossierPrevioIncluyeArquitecturaI18NConectoresDocsEInfografiasV0`;
- `TestWizardRespuestasMismoTurnoNoPisanListasAcumulativasV0`.

En `modulos/orquesta-web/nueva_app_wizard_bot_v0_test.go`:

- `TestWizardBotGroundingEstrictoV0`;
- `TestWizardBotSlotFillingRegistraRespuestasV0`;
- `TestWizardBotSinProveedorFuncionaV0`;
- `TestWizardBotNoInventaOpcionesV0`;
- los casos de degradación por presupuesto agotado o fallo del proveedor.

Para `UI-05`, adaptar los criterios de:

- `modulos/orquesta-web/nueva_app_html_handler_v0_test.go`;
- `modulos/orquesta-web/nueva_app_browser_smoke_v0_test.go`;
- `modulos/orquesta-web/nueva_app_wizard_bot_endpoint_v0_test.go`.

Los tests se portan contra comandos y snapshots V2; no se ejecutan importando
paquetes legacy ni se usan como evidencia de cierre por sí solos.

## Código que no se copia

- `WebNuevaAppIntakeSessionV0`, `SessionID` y `SessionRef` como autoridades.
- `SpecPreview` o `LaunchReady` como disparadores implícitos.
- `DossierRef` derivado de strings: no es content-addressed.
- `orquesta-factory` u `orquesta-app-planner` completos.
- Packs activados por coincidencias de palabras.
- Defaults técnicos aplicados sin decisión visible.
- Normalizadores que ignoran acciones desconocidas o reparan ambigüedad en
  silencio.
- `nueva_app_html_render_v0.go`: supera 1.800 líneas y mezcla UI, JS,
  compatibilidad y lógica.

## Riesgos históricos retenidos

El inventario legacy documenta fallos que deben convertirse en negativos V23:

- opciones visibles que no alcanzaban el contrato ejecutable;
- filas/listas pisadas entre respuestas y U6/U12 compartiendo campo;
- placeholders i18n que parecían traducciones reales;
- schemas MCP distintos del contrato web;
- bot, formulario y sesión guiada con estados divergentes;
- dossier mostrado sin confirmación exacta;
- catálogo UI, Wizard, Factory y documentación como fuentes duplicadas.

La integración correcta usa el único `intake_ref`/revisión, los comandos
canónicos, el dossier durable content-addressed, confirmación exacta y Goal
causal. Por ello no queda código legacy recomendable para copiar literalmente:
quedan contrato/negativos de app existente, diseño del bot/glosario y criterios
UX de `UI-05`.
