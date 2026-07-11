# Resultado durable del replay T9104 / BUG-226

Estado final: cerrado y aceptado el 2026-07-11.

## Identidad

- binario final reconsultado: commit `6c3ec8e95a4d1bca1f914ac02780f29d7f50ed12`;
- run: `request-ref-autoprogramming-backlog-t9104-verificar-muestra-clasificacion-s13-ed0ec8ae`;
- goal externo: `019f508b-c4ae-7910-9f75-b299c799ae49`;
- goal interno: `goal-ref-task-autoprogramming-c0ae86a2ae2e-g01`;
- atestacion: `goal-required-test-attestation-ref-2880387ca7bf7606fe30139c701fe3ff665ba69ed9f45b0eac47bcbef54e08ab`.

## Veredicto

```text
run_status=cerrada
goal_status=complete
closure_status=accepted
closure_accepted=true
recommended_action=no_action_closed
closure_issues=[]
shutdown_ready=true
agents_in_flight=0
checkpoints_pending=0
```

El atestador independiente ejecuto:

```bash
go test -count=1 ./cmd/orquesta-server \
  -run '^TestServerStackSupervisorV0DeclaraPreparacionGoalFirstCompletaV0$'
```

Resultado `passed`, `exit_code=0`; el hash de `docs` fue identico antes y
despues. El cache de modulos se materializo como copia privada offline y el
workdir hermetico fue eliminado tras el test.

## Huellas

- primera observacion aceptada: SHA-256
  `bed8bd1571cee70fa337632fd03a6c95d664d7b4bd85ed1a95930cba336c8051`;
- reconsulta final sin issues: SHA-256
  `95dba666001ae7faa0f8d4358064ced469b9188db0c5f512ad9f0df713aeb342`;
- output de test independiente: SHA-256
  `d85ab161322e40e4a3db96f39013b8d4375e62e8f975792d5ec46992d417cfbf`;
- receipt de atestacion: SHA-256
  `cf116a3e5105d256708a6232782d1850129346dcce20e3304709d5e3b275fb17`;
- shutdown final: SHA-256
  `f1cf5c6e7388ee1be380e2e8c2f53f1b1f1bc5a7e1ee668b55c4210c0a0f6d0d`.

La secuencia completa y los intentos fallidos estan documentados en
`docs/incidencias/incidencia_orquesta_t9104_wakeup_material_progress_shutdown_stale_2026-07-11.md`.
