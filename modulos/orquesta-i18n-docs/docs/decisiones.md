# Decisiones locales: orquesta-i18n-docs

Las decisiones de este archivo solo afectan a `orquesta-i18n-docs`. Si afectan a otro modulo, deben elevarse al director y registrarse en `../../CONTRATOS.md`.

## Plantilla

```text
Fecha:
Decision:
Motivo:
Alternativas:
Impacto:
Contratos afectados:
Estado:
```

## Decisiones activas

```text
Fecha: 2026-05-04
Decision: El flujo "nueva app" siempre genera catalogos i18n y documentacion inicial; no existe modo sin i18n.
Motivo: `SolicitarNuevaApp v0` declara i18n y documentacion como defaults explicitos, y este modulo tiene la responsabilidad de bundles, idiomas y plantillas.
Alternativas: Tratar i18n/docs como paso opcional posterior; permitir textos visibles hardcodeados en skeleton.
Impacto: Todo `AppSpecV0` consumido por este modulo debe traer idiomas de UI y docs declarados o normalizados por `orquesta-factory`.
Contratos afectados: GenerarI18nDocsIniciales v0, AppI18nDocsPlanV0, I18nBundleV0, DocsBundleV0.
Estado: aceptada
```

```text
Fecha: 2026-05-04
Decision: El primer generador ejecutable de planes i18n/docs usa `PlanSeedV0` local como semilla minima y `BuildAppI18nDocsPlanV0` como builder puro.
Motivo: Permite producir AppI18nDocsPlanV0 validable sin acoplarse a internals de factory, sin filesystem, sin LLM, sin DB, sin runtime y sin plantillas reales.
Alternativas: Importar AppSpecV0 de factory; construir planes solo desde fixtures JSON; esperar al adaptador completo del puerto.
Impacto: El adaptador futuro de `GenerarI18nDocsIniciales v0` debe mapear el contrato publico AppSpecV0 a PlanSeedV0 y delegar la construccion del plan en este builder.
Contratos afectados: PlanSeedV0, GenerarI18nDocsIniciales v0, AppI18nDocsPlanV0, I18nBundleV0, DocsBundleV0, I18nSkeletonLoaderShapeV0.
Estado: aceptada
```

```text
Fecha: 2026-05-04
Decision: Los catalogos i18n de web y factory comparten una unica estructura de bundle versionada.
Motivo: AGENTS.md prohibe duplicar formatos de bundle y exige que las claves sean estables y testeables.
Alternativas: Un formato por consumidor; claves generadas libremente por plantilla.
Impacto: `orquesta-web` y `orquesta-factory` consumen el mismo `I18nBundleV0`; las diferencias por runtime o framework quedan fuera del contrato local.
Contratos afectados: I18nBundleV0, I18nCatalogV0.
Estado: aceptada
```

```text
Fecha: 2026-05-04
Decision: Skeleton y loader deben usar la misma ruta logica, version, locales y claves base de catalogo.
Motivo: La estructura unica evita que la app arranque con claves distintas a las generadas en el skeleton.
Alternativas: Generar skeleton con mensajes embebidos y loader con catalogos externos.
Impacto: El contrato `I18nSkeletonLoaderShapeV0` define una huella comparable entre salida generada y loader previsto.
Contratos afectados: I18nSkeletonLoaderShapeV0, I18nBundleV0.
Estado: aceptada
```

```text
Fecha: 2026-05-04
Decision: La documentacion generada se modela como bundle versionado con idioma declarado por documento.
Motivo: AGENTS.md prohibe generar documentacion sin idioma declarado.
Alternativas: Documentacion monolingue por defecto implicito; idioma solo a nivel de proyecto.
Impacto: Cada `GeneratedDocV0` declara `locale`, `doc_type`, `title_key` y `content_key`; el texto final se resuelve desde catalogos o plantillas localizadas.
Contratos afectados: DocsBundleV0, GeneratedDocV0.
Estado: aceptada
```

```text
Fecha: 2026-05-04
Decision: El primer corte material de I18N-009 usa JSON Schema local para fijar la forma serializable de `AppI18nDocsPlanV0`, `I18nBundleV0` y `DocsBundleV0`.
Motivo: La tarea requiere fixtures validables sin implementar todavia el generador Go ni escribir filesystem.
Alternativas: Esperar al generador antes de crear schemas; validar solo con ejemplos documentales.
Impacto: Los adaptadores futuros podran validar estructura base y fixtures locales desde `docs/schemas`; las reglas relacionales dinamicas quedan para harness de contrato posterior.
Contratos afectados: AppI18nDocsPlanV0, I18nBundleV0, DocsBundleV0.
Estado: aceptada
```

```text
Fecha: 2026-05-04
Decision: `GenerarI18nDocsIniciales v0` queda tratado localmente como contrato compartido ya promovido por direccion en `../../CONTRATOS.md`.
Motivo: `orquesta-factory` consume este puerto para obtener el plan serializable de i18n/docs desde `AppSpecV0` validada.
Alternativas: Mantenerlo solo como contrato local documentado hasta implementar generador.
Impacto: I18N-001 queda cerrada como completada/promovida; el detalle canonico sigue en `docs/contratos.md` y los schemas/fixtures canonicos locales siguen bajo `docs/schemas` y `docs/fixtures`.
Contratos afectados: GenerarI18nDocsIniciales v0, AppI18nDocsPlanV0, I18nBundleV0, DocsBundleV0.
Estado: aceptada
```

```text
Fecha: 2026-05-04
Decision: I18N-010 incorpora un harness Go relacional inicial sobre DTOs serializables de i18n/docs, separado del generador y de cualquier adaptador productivo.
Motivo: JSON Schema fija la forma, pero no puede expresar de forma portable la paridad entre `locales`, `catalogs`, `required_keys`, `required_doc_types` y `fallback_locale`.
Alternativas: Mantener esas invariantes como riesgo documental hasta implementar el generador; duplicar validacion en adaptadores consumidores.
Impacto: `ValidateAppI18nDocsPlanV0` pasa a ser la verificacion local canonica de relaciones v0 para fixtures y planes construidos en memoria.
Contratos afectados: AppI18nDocsPlanV0, I18nBundleV0, DocsBundleV0, I18nSkeletonLoaderShapeV0.
Estado: aceptada
```
