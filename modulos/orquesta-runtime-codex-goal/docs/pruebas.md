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
- el prompt separa direccion interna de Codex Goal y gobierno externo de
  Orquesta.
- `cmd/orquesta-server` prueba el wiring de composicion opt-in
  `app_server_proxy`: solo expone launcher/observer cuando hay backend
  configurado, mapea `thread/start`, `thread/goal/set`, `turn/start` y
  `thread/goal/get`, y no usa `codex exec`.
