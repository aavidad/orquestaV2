# orquesta-opes-director

Adaptador OPES para producir trabajos causales desde artefactos de dominio ya
recibidos por Orquesta.

Responsabilidad:

- observar registros de entrega `DomainWork` de la composicion;
- expandir `document_plan` aceptados mediante `orquesta-document-plan-expander`;
- convertir `followup_refs`, `rework_refs`, faltas obligatorias y rechazos en
  nuevos `DomainWorkJobRequestV0`;
- tratar `pending_refs` como alias de seguimiento causal pendiente;
- no convertir rechazos transitorios de envio (`429`, timeouts, retry budget,
  puerto no disponible) en trabajos de correccion antes de que pueda reintentarse
  el envio original;
- evitar cierres falsos de paquetes OPES con `pendiente_continuar`;
- liberar un paquete final solo si `CompleteJob=true`, no quedan pendientes y
  viajan evidencias del `manifest_cierre.json` completo: HTML, RAG, audio,
  tests, visual y QA;
- crear rework causal `finalize_temario_package` cuando un paquete final llega
  con `CompleteJob=true` pero sin evidencias completas de manifest de cierre;
- preservar estados estructurados de avance parcial del registro OPES, por
  ejemplo `texto_minimo_B_ok_pendiente_assets_html_tests_rag_audio_qa`, cuando
  vienen en el artefacto causal;
- exigir checkpoint durable cuando un artefacto OPES procedente de `goal_first`
  quiera asentarse como `settled_text` o `settled_final`; si solo hay heartbeat
  o no hay checkpoint, publicar `goal-first-topic-checkpoint-required` y rework
  causal en vez de cerrar por estado implicito;
- publicar un texto ya asentado como `texto_asentado_pendiente_derivados` y
  `operational_status=waiting`, no como `working`, para evitar reescritura
  tardia sin rework causal mientras quedan derivados por completar;
- aplicar el actualizador del registro OPES antes de persistir el job
  `update_topic_registry`, de modo que un fallo de herramienta no bloquee el
  siguiente tick por idempotencia prematura;
- acotar la lectura por `correlation_id` cuando el operador lo proporciona;
- mantener idempotencia estable sin tocar core ni internals de OPES.

Fuera de alcance:

- ejecutar agentes;
- leer colas OPES amplias;
- publicar en produccion;
- decidir por heuristicas de texto libre.
