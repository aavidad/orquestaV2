# orquesta-context

Responsabilidad: preparar contextos pequenos y versionados para agentes.

Incluye:

- manifiesto `ContextBundleV0`;
- clasificacion por fase, modulo y microtarea;
- refs opacas a docs locales, contratos y evidencias;
- limites de tamano;
- regla de consulta al director si falta informacion externa.

No lee filesystem productivo ni conoce proveedores. Los adaptadores futuros podran materializar las refs por disco, MCP, REST u otro conector.

Estado T15 2026-05-27: contexto required `ref_only` se resuelve por accion
explicita y evidencia ACK. La politica de detalle permite refs opacas y corta
solo valores sensibles, material crudo o rutas privadas en campos activados.
