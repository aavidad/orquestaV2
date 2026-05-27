# Rotacion experimental de sesiones por handoff

Estado: contrato opt-in de baja prioridad para T207.

## Frontera

La rotacion de sesiones no es rail estricto del servidor. Ningun proceso vivo se
corta por este contrato. El nucleo puro conserva refs opacas, estado y eventos;
la epoca de sesion vive en adaptadores/runtime como `session_epoch`.

## Contrato

La composicion que quiera probar rotacion construye un
`RuntimeSessionRotationRequestV0` con:

- `mode=opt_in_low_priority`;
- `session_epoch` opaca;
- `run_ref`, `task_ref`, `agent_ref` y `session_ref` opacas;
- `handoff` versionado `runtime_session_rotation_handoff.v0`;
- evidencias causales compactas.

El handoff aceptable contiene objetivo original, avance real, siguiente accion,
pendientes, `outgoing_ack_ref`, `outgoing_status=handoff_ready`,
`pending_status`, refs causales y evidencias. Puede incluir archivos tocados
relativos al workspace, pruebas requeridas y riesgos. No debe contener HOME,
secretos, rutas absolutas, transcripts, prompts ni detalles de proveedor.

## Decisiones

`EvaluateRuntimeSessionRotationV0` y
`BuildSessionRotationDirectiveV0` aplican estas reglas:

- sin opt-in: continuar la sesion actual;
- handoff ausente o insuficiente: pedir al director completar handoff o
  continuar la sesion actual;
- handoff aceptado: permitir lanzar sesion de relevo con contexto acotado;
- `stop_current` queda `false` en todos los casos de este corte.
- cada decision publica `comparison_metric` con baseline
  `continue_current_session`, accion candidata, senales compactas de
  continuidad/coste/calidad y refs de evidencia. Es una metrica de experimento,
  no una orden de cortar el proceso actual.

## Verificacion focal

```bash
go test -count=1 ./modulos/orquesta-runtime ./modulos/orquesta-orchestration-core
```

El smoke real de relevo queda pendiente. Para cerrarlo debe demostrar
relanzamiento opt-in sin duplicar trabajo, preservando refs causales,
parent/child refs y `WaitAgentRefs` acotado.
