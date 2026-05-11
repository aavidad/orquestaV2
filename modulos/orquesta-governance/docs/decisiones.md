# Decisiones locales: orquesta-governance

Las decisiones de este archivo solo afectan a `orquesta-governance`. Si afectan a otro modulo, deben elevarse al director y registrarse en `../../CONTRATOS.md`.

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

## Decisiones iniciales desde DB v1

```text
Fecha: 2026-05-04
Decision: Usar DB v1 como fuente forense para governance, pero no importar reglas, skills ni workflows como activos por defecto.
Motivo: La DB historica contiene reglas versionadas y workflows utiles, pero tambien refleja etapas antiguas, CLI como camino principal y decisiones de dominio que no deben gobernar OrquestaV2.
Alternativas: Importar todas las filas; ignorar la DB; copiar solo textos sueltos a docs globales.
Impacto: `orquesta-governance` debe construir `GovernanceCatalogV0` con origen, estado, version y criterio de promocion antes de activar cualquier regla.
Contratos afectados: GovernanceCatalog v0 compartido, DecisionRecord v0 futuro.
Estado: aceptada_director
```

```text
Fecha: 2026-05-04
Decision: `GovernanceCatalogV0` queda como catalogo documental no activo para la extraccion inicial, con historicos sujetos a revision.
Motivo: GOV-001 debe extraer reglas, skills y workflows de DB v1 sin activarlos automaticamente ni tocar contratos globales fuera del write-set permitido en aquel corte.
Alternativas: Activar reglas candidatas en V2; copiar todas las filas como canon; descartar DB v1 completa.
Impacto: Las entradas extraidas solo pueden convertirse en reglas efectivas mediante decision trazable, contrato versionado y criterio de promocion explicito.
Contratos afectados: `GovernanceCatalogV0`, `GovernanceCatalogEntryV0`.
Estado: aceptada_local; actualizada tras promocion global de `GovernanceCatalog v0` en `../../CONTRATOS.md`.
```

```text
Fecha: 2026-05-04
Decision: Posponer temporalmente la promocion de `GovernanceCatalogV0` a contrato global.
Motivo: GOV-001 solo extrae catalogo documental y no debe convertir reglas historicas en reglas efectivas. La promocion global necesita consumidor real en core/MCP y decision sobre reglas efectivas.
Alternativas: Promover ya el catalogo; activar reglas candidatas; posponer toda extraccion.
Impacto: En aquel corte, governance podia seguir con GOV-002/GOV-003 mientras la promocion global quedaba diferida.
Contratos afectados: GovernanceCatalogV0, DecisionRecord v0 futuro.
Estado: superada_director 2026-05-04; el director promovio `GovernanceCatalog v0` a contrato compartido en `../../CONTRATOS.md`.
```

```text
Fecha: 2026-05-04
Decision: `GovernanceCatalog v0` queda promovido a contrato compartido global, con detalle canonico en `orquesta-governance/docs/contratos.md`.
Motivo: El contrato ya tiene schema draft7, fixtures canonicos e invariantes suficientes para que core/MCP lo consuman como catalogo efectivo/propuesto/cuarentena sin activar historicos DB v1 por defecto.
Alternativas: Mantenerlo local hasta harness de consulta; promover solo despues de conectar core; duplicar detalle en `../../CONTRATOS.md`.
Impacto: `../../CONTRATOS.md` referencia el contrato compartido minimo y este modulo conserva el detalle canonico. Cambios incompatibles de tipos, estados o permisos requieren version nueva o `CONSULTA AL DIRECTOR`.
Contratos afectados: GovernanceCatalog v0 compartido, `GovernanceCatalogV0`, `GovernanceCatalogEntryV0`.
Estado: aceptada_director
```

```text
Fecha: 2026-05-04
Decision: Usar referencias forenses por tabla/rol/categoria/nombre, no IDs numericos de DB v1.
Motivo: Los IDs historicos son artefactos de SQLite y no deben convertirse en identidad publica ni afectar compatibilidad futura.
Alternativas: Conservar `id`/`*_id` como claves externas; generar codigos canonicos nuevos en GOV-001.
Impacto: El catalogo conserva trazabilidad suficiente sin acoplarse a la base historica. Cualquier identificador estable futuro debe definirse en contrato separado.
Contratos afectados: `GovernanceCatalogEntryV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Implementar la primera consulta Go de `GovernanceCatalogV0` como funcion pura sobre DTOs en memoria.
Motivo: Core y MCP necesitan un harness compacto antes de conectar consumidores reales, pero governance no debe leer DB v1 ni filesystem productivo en runtime.
Alternativas: Leer fixtures JSON desde disco en runtime; consultar SQLite v1 readonly; devolver un catalogo plano sin validar estados efectivos.
Impacto: `QueryEffectiveGovernanceCatalogV0` devuelve entradas efectivas filtradas por modulo/rol/fase/tags, cuenta `proposed` y `quarantine` que coinciden con el mismo filtro, e invalida entradas mal ubicadas en `effective`.
Contratos afectados: `GovernanceCatalogV0`, `GovernanceCatalogEntryV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Anyadir `scope.tags` como campo opcional para consulta documental compacta.
Motivo: El contrato ya exigia reglas consultables por alcance; el microcorte GOV-006 necesita tags sin introducir permisos ni estados nuevos.
Alternativas: Inferir tags desde nombres/resumen; bloquear el filtro por falta de campo; crear version v1 del contrato.
Impacto: Las entradas sin tags siguen siendo validas; los tags solo filtran cuando el consumidor los solicita y no hacen activa una entrada `proposed` o `quarantine`.
Contratos afectados: `GovernanceCatalogV0`, `GovernanceCatalogEntryV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Clasificar reglas municipales, financieras, CLI y rutas documentales v1 como `quarantine` salvo que expresen un principio reutilizable.
Motivo: DBV1-000 indica que SQLite, CLI como camino principal, reglas municipales/ContaGrx y rutas v1 no son doctrina global de OrquestaV2.
Alternativas: Promocionarlas como reglas por defecto; eliminarlas del catalogo.
Impacto: Se conserva evidencia historica para futuras plantillas de dominio, pero no gobierna V2 ni se ofrece como catalogo efectivo.
Contratos afectados: `GovernanceCatalogV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Publicar `GovernanceCatalogV0` en tres bloques explicitos: `effective`, `proposed` y `quarantine`.
Motivo: Un unico listado documental facilita confundir candidatos DB v1 con reglas vivas. La separacion obliga a que solo `effective` gobierne consumidores futuros y deja los historicos como evidencia o cuarentena.
Alternativas: Mantener una lista plana con `proposed_status`; activar candidatos marcados como `promote_candidate`; dejar toda la clasificacion solo en Markdown.
Impacto: El schema canonico draft7 exige estado, version, origen forense y criterio de promocion por entrada. En v0 no hay entradas DB v1 efectivas.
Contratos afectados: `GovernanceCatalogV0`, `GovernanceCatalogEntryV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Una entrada historica con `origin.source_type=dbv1` no puede aparecer en `catalogs.effective` sin `review_state=approved_effective`, `decision_ref` y `promoted_at`.
Motivo: La trazabilidad forense es evidencia, no autoridad normativa. Activar una regla, skill o workflow historico sin revision importaria reglas v1 como canon vivo.
Alternativas: Confiar en el campo de version historica; aceptar `promote_candidate` como efectivo; prohibir toda promocion futura de DB v1.
Impacto: Los consumidores futuros deben validar el schema o regla equivalente antes de tratar una entrada como efectiva.
Contratos afectados: `GovernanceCatalogV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: El control de secretos del catalogo debe declararse con marcadores estructurados y no puede depender solo de busquedas de texto libre.
Motivo: Governance publica reglas y evidencia; no debe filtrar secretos por heuristica textual como unica barrera, ni copiar tokens, credenciales, rutas HOME reales o datos de cuenta runtime.
Alternativas: Usar solo `rg`/palabras prohibidas; no registrar controles de secretos en el contrato; delegarlo a runtime.
Impacto: El schema canonico exige `secret_control_policy.free_text_scan_only_allowed=false` y `secret_controls.review_method` estructurado por entrada.
Contratos afectados: `GovernanceCatalogV0`, `GovernanceCatalogEntryV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Tratar las tablas de versiones v1 como evidencia debil de versionado, no como historial funcional completo.
Motivo: La lectura readonly muestra `version_num=1`, actor `orquesta` y accion `seed` en todas las versiones observadas; ademas hay 4 reglas y 1 skill sin fila de version historica.
Alternativas: Inferir cambios historicos no presentes; ignorar tablas `_versiones`.
Impacto: V2 debe exigir version documental real en `GovernanceCatalogV0`, pero no debe asumir que DB v1 contiene evolucion completa.
Contratos afectados: `GovernanceCatalogEntryV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Promocionar OP-001..OP-124 solo como evidencia historica reutilizable, nunca como mandato directo.
Motivo: La documentacion historica contiene ideas validas, pero tambien decisiones rechazadas, duplicadas, obsoletas o ligadas a supuestos de v1. Si esas OP mandaran por si mismas, volveriamos a importar errores de contexto, CLI y arquitectura abandonada.
Alternativas: Ignorar por completo las OP historicas; tratarlas como reglas vivas; promover automaticamente las que tuvieron consenso.
Impacto: `DecisionPromotionPolicyV0` fija que las OP y votos historicos solo sirven como evidencia, con minimo de votos configurable por fase, rechazo expreso de duplicadas/rechazadas y escalado al director si hay empate, riesgo alto o intento de volverlas efectivas.
Contratos afectados: `DecisionPromotionPolicyV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Recuperar el sistema de votacion como `ArchitectureVoteV0` con opciones explicitas, umbrales por fase y escalado obligatorio para contratos compartidos.
Motivo: El historico muestra que votar sin contrato minimo, sin opciones cerradas o sin criterio de cierre produce bucles. La votacion V2 debe servir para brainstorming y cambios compartidos sin depender del nucleo ni de la CLI v1.
Alternativas: Mantener solo narrativa libre; votar por comentarios sin DTO minimo; esperar a que el core defina el workflow.
Impacto: `ArchitectureVoteV0` deja un formato documental/contractual compacto que puede exponerse por REST/MCP/CLI en el futuro. El cierre local solo vale para alcance acotado; si toca arquitectura global o contrato compartido, decide el director.
Contratos afectados: `ArchitectureVoteV0`, `DecisionPromotionPolicyV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Exponer `GovernanceCatalog v0` por un puerto publico read-only, pequeno y local al modulo antes de cualquier integracion con core.
Motivo: CLI necesita una superficie publica para consultar governance efectiva sin acoplarse al detalle interno del catalogo ni esperar al nucleo. El slice debe ser compacto, no activar historicos y seguir siendo usable con proveedor inyectable o fixture canonico.
Alternativas: Esperar a un puerto del core; devolver el catalogo completo sin filtros; leer DB v1 readonly desde el adaptador; hacer que CLI importe el DTO interno y llame a la funcion pura directamente.
Impacto: `GovernanceCatalogPublicQuery v0` queda definido con filtros por modulo, rol, fase y tags, y una respuesta compacta que solo devuelve `effective` junto a contadores de `proposed` y `quarantine`. La ruta/version local recomendada es `POST /api/v0/governance/catalog/query` y necesita promocion del director si pasa a contrato compartido entre modulos.
Contratos afectados: `GovernanceCatalogV0`, `GovernanceCatalogPublicQuery v0`.
Estado: aceptada_local
```
