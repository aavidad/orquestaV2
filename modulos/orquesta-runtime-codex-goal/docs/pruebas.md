# Pruebas

```bash
go test -count=1 ./modulos/orquesta-runtime-codex-goal
```

Cobertura actual:

- el packet incluye objetivo, contexto, reglas, write-set, tests, criterios de
  aceptacion, artefactos, evidencias, presupuesto y politicas de cierre/rework;
- el launcher rechaza specs invalidos;
- el launcher llama al puerto real inyectado;
- el observer valida requests, llama al puerto inyectado y devuelve
  `GoalWorkResultV0`;
- el observer rechaza backend ausente, errores del backend, refs invalidas y
  `goal_ref` cruzado;
- el launcher y observer preservan `IssueCode` de backend cuando el transporte
  real falla, para no degradar diagnosticos como socket o standalone ausente;
- el prompt separa direccion interna de Codex Goal y gobierno externo de
  Orquesta.
- `cmd/orquesta-server` prueba el wiring de composicion opt-in con backends
  app-server: solo expone launcher/observer cuando hay backend
  configurado, mapea `thread/start`, `thread/goal/set`, `turn/start`,
  `thread/goal/get` y `thread/read`, extrae el marcador
  `ORQUESTA_GOAL_RESULT_V0` de un `agentMessage` final o fallback sin phase,
  usa `orquesta_goal_result_v0.json` como resultado durable preferente cuando
  existe con `goal_ref` coincidente, y no usa `codex exec`.
- smoke real opt-in documentado:
  `docs/runbooks/smoke_goal_first_app_server_real_2026-06-25.md` y
  `scripts/smoke_goal_first_app_server_real.sh`.
