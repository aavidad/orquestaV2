# Patron SDK de tools/capabilities

Fecha: 2026-07-10. Estado: V0 offline integrable por puertos; sin materializador
real, transporte ni composicion productiva.

## Patron canonico

`intent refs -> authorities -> verified snapshot -> durable plan/claim -> effect adapter -> reconcile/CAS`

1. **Intent**: `GeneratedAppToolAttachRequestV0` solo transporta app/binding,
   manifest/bundle, config, write-set, modo, operacion e idempotency refs. Nunca
   transporta un binding ni permisos elegidos por el caller.
2. **Authorities**: la composicion resuelve el binding por
   `GeneratedAppToolBindingAuthorityPortV0`, el manifest por registry y el
   bundle por `ToolBundleSnapshotResolverPortV0`. Los refs devueltos deben
   coincidir exactamente con los solicitados. El validator recibe el plan con
   binding resuelto por autoridad.
3. **Snapshot**: el resolver verifica hashes y publica `SnapshotRef`
   content-addressed, `HandleRef` opaco y `VerificationRef`. El materializador
   consume el handle del plan, no vuelve a abrir rutas del request.
4. **Plan/claim**: el store persiste atomicamente `ToolOperationRecordV0`
   completo: plan inmutable, receipt y lease de destino. Deduplica por
   `operation_ref`; serializa por `target_ref=app+tool+binding`; completa por
   CAS sobre `state_version`.
5. **Effects**: materialize, uninstall y rollback son adaptadores opt-in. Deben
   deduplicar durablemente por `operation_ref`. Rollback recibe el plan previo
   exacto, incluido su snapshot/handle.
6. **Recovery**: `ReconcileGeneratedAppToolOperationV0` inspecciona el efecto;
   `ResumeGeneratedAppToolOperationV0` solo reintenta tras reconciliar
   `not_applied` y adquirir `resuming` por CAS. `claimed`, `resuming` y
   `recovery_required` retienen el lease; estados terminales lo liberan.

## Hashes y fingerprint

- `ToolBundleV0.content_hash` cubre manifest y bundle salvo el propio hash:
  artifacts+hashes, i18n, migrations, tests y compatibilidad.
- La codificacion canonica es `s<bytes>:<utf8>;`, `b0;`/`b1;`,
  `a<count>[<values>]` y `o<count>{<sorted-key><value>}`. Objetos ordenan claves
  y colecciones equivalentes son conjuntos ordenados. Duplicados se rechazan.
- `plan_fingerprint` cubre operacion/idempotencia, manifest/capability/tool,
  bundle ref+content hash, snapshot ref, config, write-set, binding autorizado,
  modo y previous receipt. El handle se excluye para permitir reemitir un
  handle seguro del mismo snapshot.
- La misma idempotency key con fingerprint distinto es conflicto, nunca replay.

## Garantias honestas

- Claim atomico + lease evitan dos efectos concurrentes desde el usecase.
- No existe exactly-once distribuido automatico entre store y efecto externo.
  Un crash puede dejar `claimed` o `resuming` antes o despues del efecto.
- Reconcile distingue `applied`, `not_applied` y `unknown`. `applied` completa
  sin repetir; `not_applied` puede reanudarse con el mismo `operation_ref`;
  `unknown` queda `recovery_required`.
- At-least-once solo aparece en `Resume` explicito y exige materializador
  idempotente. Exactly-once requiere transaccion comun o deduplicacion durable
  equivalente en el adaptador.
- Errores operativos posteriores al claim intentan una transicion durable a
  `recovery_required`. Si el propio store no esta disponible, el claim previo
  sigue siendo la referencia de reconciliacion; no se relanza silenciosamente.

## Seguridad de adaptadores

- Un snapshot filesystem debe partir de un root confiable no controlado por el
  request, abrir con no-follow/openat o handles seguros, rechazar symlinks y
  verificar bytes antes de publicar el handle. El handle debe sobrevivir o
  poder reconciliarse durante recovery.
- `AllowedWriteSet` y `WriteSet` usan paths relativos canonicos por segmentos.
  URI, drives Windows relativos/absolutos, UNC, tilde, NUL, barras invertidas y
  traversal se rechazan.
- `StatusGeneratedAppToolV0` exige siempre `app_ref+tool_ref`; binding,
  operacion, receipt e idempotency son refinamientos opcionales.
- Upgrade exige `PreviousReceiptRef` exacto, estado previo `attached/upgraded`,
  app/binding/tool/capability coincidentes y `UpgradeFromRefs` compatible.

## Adaptador durable de referencia

`modulos/orquesta-tool-capability-file` implementa el store con `flock` entre
procesos, estado atomico, `fsync`, root privado, `O_NOFOLLOW`, validacion de
indices/leases y limites de lectura. Es referencia offline, no wiring
productivo ni snapshot resolver.

## Checklist

- Registrar binding autorizado fuera del request.
- Publicar manifest, bundle hash y snapshot verificado.
- Implementar materializador idempotente y reconciliable por `operation_ref`.
- Persistir plan+receipt+lease atomicamente y usar CAS para cada transicion.
- Probar payload conflictivo, lease de destino, crash antes/despues del efecto,
  resume concurrente, upgrade/rollback exacto y store multiproceso.
- Mantener transports y filesystem fuera del core.

## Limites pendientes

- No hay resolver/materializador filesystem real ni connector remoto. Cuando
  exista debe aportar pruebas propias de root/handle/symlink/hash y recovery.
- El file-store no esta cableado en servidor o composicion productiva.
- No hay MCP/API global ni integracion funcional con `document-extraction`.
