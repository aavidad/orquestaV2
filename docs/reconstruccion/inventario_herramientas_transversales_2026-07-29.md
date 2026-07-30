# Inventario transversal de herramientas estudiadas

Fecha de corte: 2026-07-29.

Read-model estructurado:
`product/knowledge/tooling_adoption_v1.json`.

## Veredicto

Las herramientas no se han eliminado del diseño V2. El roadmap conserva
capacidades específicas:

- `TLS-01..14`: tools, resources, MCP, SDK, skills, plugins y doctor;
- `CTX-01..12`: contexto compacto, caveman, caché, FTS/BM25, embeddings,
  reranking, vector DB y evaluaciones;
- `EXT-03..22`: conectores web, documentos, datos, office, Git, comunicación,
  medios, shell y computer-use;
- `AGT-01..12`: providers, modelos, selección y disponibilidad;
- `OPS-15..20`: instalación, shutdown, retención, watchdog y observabilidad;
- `EVD-04/EVD-13`: atestación independiente y sandbox.

El problema es de aplicación, no de memoria documental: gran parte sigue en
estado `declared`. Antes de este inventario no existía una vista única que
distinguiera acreditado, parcial, candidato, condicional y deuda.

## Estado resumido

| Estado | Significado | Ejemplos |
|---|---|---|
| Acreditado | V2 tiene evidencia ejecutable. | MCP público, CredentialStore, Git/worktrees, Bubblewrap TestAttestor. |
| Parcial | Una parte está probada, falta el contrato completo. | refs fuera del prompt, broker de código, multi-HOME, observabilidad, familia de providers. |
| Candidato | Solución aceptada o caracterizada, sin seal V2. | Tool SDK, skills, plugins, caveman, FTS/BM25, Firecracker. |
| Condicional | Solo entra si gana un benchmark. | prompt cache, embeddings híbridos, reranking, vector DB. |
| Deuda | Necesidad aceptada aún no acreditada. | skill creator, doctor, evals, watchdog, retención y conectores de dominio. |

No debe decirse que Firecracker, skill creator o una base vectorial “ya
funcionan” por tener diseño o código parcial. Firecracker sigue como candidato
no acreditado: el read-model lo asigna a V38 `agent_runtime_elastic` mediante
`ORC-28`, mientras la decisión técnica `agent_microvm_network` permanece
`planned_not_applied`. Tampoco debe interpretarse `declared` como descarte.

## Decisiones

1. V23 no absorbe herramientas transversales. Su plan solo cierra Wizard.
2. V26 es dueño de Tool SDK, skills, skill creator, plugins y doctor.
3. V27 es dueño de contexto, caveman, broker, RAG y evaluaciones.
4. La opción de búsqueda inicial es `rg`/metadata y FTS5/BM25. Embeddings,
   reranking o vector DB exigen benchmark positivo.
5. Firecracker sigue como candidato a adaptador opt-in de aislamiento fuerte;
   no se afirma que funcione ni que esté acreditado. Su siguiente gate exige
   A+B+C sobre el mismo candidato: núcleo neutral, adaptador Firecracker y ola
   física 1/5/10/16/20. Bubblewrap continúa acreditado para TestAttestor; no se
   confunden ambos alcances.
6. Multi-HOME pertenece al adaptador Codex/composición, nunca al core.
7. Observabilidad, watchdog, shutdown y retención pertenecen a operación y no
   deciden contenido de agentes.
8. Conectores web, PDF, datos, presentaciones, media y computer-use entran por
   Tool SDK/plugin con permisos y receipts; nunca por imports en dominio.

## Uso sin tokens

```bash
scripts/consultar_herramientas.sh --all
scripts/consultar_herramientas.sh --status debt
scripts/consultar_herramientas.sh --capability CTX-03
scripts/consultar_herramientas.sh --term firecracker
```

La consulta combina este read-model con el estado actual de
`product/roadmap.json`. Así una promoción futura no deja el inventario
silenciosamente desfasado.

## Regla para nuevos estudios

Toda herramienta nueva debe añadirse al roadmap y a este read-model con:

- problema y alternativa simple;
- capability y vertical dueño;
- versión, fuente y licencia;
- permisos, secretos, red y efectos;
- coste de tokens, CPU, RAM, disco y operación;
- prueba mínima, benchmark si procede y plan de retirada;
- estado honesto: investigación no equivale a producto.
