# V18: preparación P/S/E pendiente

V18 `independent_reviews` permanece sin acreditar. Esta preparación no crea
`product/evidence/v18_independent_reviews.json`, no captura output y no cambia
el roadmap ni las capacidades. El receipt V3 se genera como parte de E, después
de ejecutar S desde checkout `detached_clean`.

Bases separadas:

- autoridad contractual V17: `4428f46dd6b48659a4fb871a66cb72927f41cb93`;
- delta integrado V18: `e311a97e4fdcafcf81c6114dbd904af4ed9293ad`.

La separación evita que el candidato V18 reatribuyera cambios V35--V37 ya
integrados en `main`. El delta sí conserva
`product/evidence/real_codex_mcp_e2e.json`: es evidencia genérica renovada por
su responsable y sujeto canónico del candidato, no evidencia V18 ni una
mutación prohibida.

## Protocolo posterior

1. P ejecuta el `execution_argv` del fixture desde el candidato limpio y crea
   el commit inmutable con `product_delta_sealed_git_commit_oid` aún vacío.
2. S escribe únicamente el OID de P, cambia el estado a
   `p_product_delta_sealed_pending_evidence` y verifica el inventario exacto
   del delta; sigue sin output, receipt ni promoción.
3. E parte de checkout `detached_clean` de S, ejecuta exactamente el argv,
   conserva output y genera el receipt V3 fuera del sujeto; ambos se registran
   en el commit E. Solo entonces puede actualizar el roadmap y acreditar los
   cinco IDs V18.

El V18 real E2E sigue siendo requerido por el argv, pero no se ejecuta durante
esta preparación. Un fallo en P, S o E conserva diagnóstico y deja V18
`planned`/`declared`.
