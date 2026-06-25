# Pruebas

Validacion local:

```bash
go test -count=1 ./modulos/orquesta-goal
```

Cobertura actual:

- spec valido se normaliza a `orquesta_goal_work_spec.v0`;
- falta de objetivo bloquea validacion;
- write-set absoluto o con `..` se rechaza por seguridad;
- refs con rutas locales sensibles se rechazan;
- observation request sin `goal_ref` o con `external_goal_ref` absoluto se
  rechaza;
- result con refs de artefacto, evidencia, test o receipt invalidas se rechaza;
- result con estado `accepted` se rechaza porque `accepted` pertenece al recibo
  de lanzamiento, no a la observacion del goal;
- reglas blandas advisory no bloquean el contrato;
- `complete` con spec invalido no valida cierre;
- `complete` sin `goal_ref` causal igual al spec no valida cierre;
- `complete` sin evidencias requeridas no valida cierre.
