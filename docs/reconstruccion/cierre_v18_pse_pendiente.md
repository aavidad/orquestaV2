# V18: cierre P/S/E acreditado

V18 `independent_reviews` queda acreditada por
`product/evidence/v18_independent_reviews.json`. La cadena válida es P
`f0640009145e0ea28e8bbc955ebed2510cb02ce6`, S2
`32ee17e407006d9e0aeb46557b1e160769dd4848` y E2
`9d3469bad1433c1f83150292acdd7e5eef205352`; el candidate es
`sha256:286e824a7173ee277d1e406c7eea67482f26738fa8eda6e2a9ed0b8d7f7a9664` y
el output byte-idéntico ligado es
`sha256:2f7b7a6b614387bf37fe1465acc77d05d97adc1ac64eaeabe1f0944ae69b0ca2`.

Bases separadas:

- autoridad contractual V17: `4428f46dd6b48659a4fb871a66cb72927f41cb93`;
- delta integrado V18: `e311a97e4fdcafcf81c6114dbd904af4ed9293ad`.

La separación evita que el candidato V18 reatribuyera cambios V35--V37 ya
integrados en `main`. El delta sí conserva
`product/evidence/real_codex_mcp_e2e.json`: es evidencia genérica renovada y
sujeto canónico del candidato. Su ratificación queda enlazada al lease
[`lease_receipt.json`](/home/alberto/Trabajo/orquesta-rebuild/.orquesta-runtime/control-plane/runs/v18-seal-20260723T011000Z/lease_receipt.json)
(`sha256:a11a7ec31cdbd807a2eeb190d58e80ec0620aa3c6062fa626f5f725a8e592985`) y a la
revisión independiente [`review_receipt.json`](/home/alberto/Trabajo/orquesta-rebuild/.orquesta-runtime/control-plane/runs/v18-seal-20260723T011000Z/review_receipt.json)
(`sha256:ae9b5f2eb94fca720f562c87a6c90a8cadfa573d0725d7247bf0f05541b2c18d`);
no es evidencia V18 ni una mutación prohibida.

El primer E fue invalidado antes de acreditar por el falso rojo global
`BUG-REBUILD-20260723-333`; la cadena anterior no se reutilizó. E2 volvió a
ejecutar el argv desde `detached_clean`, pasó la suite raíz y aceptación y fue
revisada `GO` en
`.orquesta-runtime/control-plane/runs/v18-seal-20260723T011000Z/e2_review_receipt.json`
(`sha256:98a62d5a0f983f1e3509c1a46b0fb89de52b42c409749e40091869011add877b`).
El siguiente frente es V19; `L-TRACE` y `L-SEAL` siguen vivos hasta que el
operador registre su liberación.
