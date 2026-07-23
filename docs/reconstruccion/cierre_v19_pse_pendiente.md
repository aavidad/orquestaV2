# V19 Council — cierre P/S/E pendiente

Estado P2: `implemented_unsealed`. Producto y tests Council existen; todavía no
hay acreditación V19.

- P: fixture schema 1 con envelope preparado para receipt V3, command/argv
  congelados y candidate subjects declarados. Roadmap sigue `planned`; seis
  capabilities siguen `declared` y sin `evidence_refs`.
- S: crear únicamente `product/evidence/v19_council_seal.json` con árbol,
  binario, configuración efectiva redactada, receipt V18, sujetos Council,
  LOC y 43 ficheros sobre 350 líneas. No crear receipt ni output.
- E: desde worktree `detached_clean`, ejecutar argv congelado y crear pareja
  `v19_council.output.txt` + receipt V3. Receipt debe enlazar fixture, hash de
  candidate y commit/tree sellado. Solo review externa `GO` promociona roadmap
  y capabilities.

No se permite placeholder de manifest, output o receipt. El candidate P2 no
incluye paths de S/E; antes de S se recalcula contra base y commit P2 sellado.

Deuda: límites observados no acreditan simplicidad. Inventario P2: producto
`3419` (dominio `387`, aplicación `1214`, SQLite/recovery `1818`, bootstrap de
producción `0`); bootstrap/test separado `545`; 43 ficheros pasan de 350. El
ratchet V22 es dominio `>=400`, SQLite/recovery `>=1850`, fichero `<=1500` y
bootstrap de producción cap 400, con test/bootstrap como métrica separada.
