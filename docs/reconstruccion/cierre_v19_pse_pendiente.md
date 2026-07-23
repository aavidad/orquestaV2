# V19 Council — cierre P/S/E pendiente

Estado P2: `implemented_unsealed`. Producto y tests Council existen; todavía no
hay acreditación V19.

- P: fixture schema 1 con envelope preparado para receipt V3, command/argv
  congelados y candidate subjects declarados. Roadmap sigue `planned`; seis
  capabilities siguen `declared` y sin `evidence_refs`.
- S: crear únicamente `product/evidence/v19_council_seal.json` con árbol,
  binario, configuración efectiva redactada, receipt V18, sujetos Council,
  LOC y 53 ficheros candidatos sobre 350 líneas. No crear receipt ni output.
- E: desde worktree `detached_clean`, ejecutar argv congelado y crear pareja
  `v19_council.output.txt` + receipt V3. Receipt debe enlazar fixture, hash de
  candidate y commit/tree sellado. Solo review externa `GO` promociona roadmap
  y capabilities.

No se permite placeholder de manifest, output o receipt. El candidate P2 no
incluye paths de S/E; antes de S se recalcula contra base y commit P2 sellado.

Deuda: límites observados no acreditan simplicidad. Inventario P2: producto
`3441` (dominio `387`, aplicación `1214`, SQLite/recovery `1835`, bootstrap de
producción `0`, MCP/transporte `5`); soporte de test separado `3731`
(bootstrap `668`, MCP `43`); 53 ficheros candidatos pasan de 350. El
ratchet V22 es producto `<=3650`, dominio `<=400`, aplicación `<=1300`,
SQLite/recovery `<=1850`, fichero `<=1500` y bootstrap de producción `<=400`,
MCP/transporte `<=50`, con el límite de fichero aplicado a producción y el
soporte de test inventariado por separado.
