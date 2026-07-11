# T9104: progreso material desacoplado y shutdown stale

Fecha: 2026-07-11

Estado: `BUG-ORQ-20260711-227/228` cerrados localmente; replay real pendiente.

## Alcance y evidencia retenida

La prueba empirica autorizada para cerrar `BUG-226` se ejecuto con un unico
goal util `T9104`, `MAX_REQUESTS=1`, worktree temporal y runtime aislado. No se
toco remoto, OPES ni ninguna aplicacion productiva.

- runtime: `/tmp/orquesta-bug226-t9104-runtime`
- worktree: `/tmp/orquesta-bug226-t9104-worktree`
- status inicial: `status-001.json`
- paradas HTTP: `shutdown-unbound.json` a `shutdown-unbound-4.json`
- control forzado confirmado: `run-control-unbound.json`
- goal externo abortado: `019f5040-faee-78c3-9941-6d3d2fd6d82e`

Ese goal no se reutiliza. Tras confirmar la parada por `runs/control`, el
servidor temporal se cerro cooperativamente con `SIGINT` porque el segundo bug
impedia alcanzar `shutdown_ready`.

## BUG-ORQ-20260711-228: el decorador ocultaba el puerto material

El servidor vivo publico `material_progress_state_reader_unbound`. El store
file-based implementaba `MaterialProgressStateStorePortV0`, pero
`serverWakeupGoalStateStoreV0` solo reexponia los contratos Goal. El stack
obtiene el reader mediante type assertion sobre el store ya decorado, por lo
que el governor, MCP y `runs/control` perdian la misma capacidad al activar el
relay residente.

Cierre local:

- el decorador delega lectura y CAS al puerto material de su `inner`;
- no se amplia el contrato Goal ni se obliga a los fakes legacy a implementar
  persistencia material;
- el test de wiring con relay real exige reader en MCP y `runs/control`.

## BUG-ORQ-20260711-227: accion de shutdown anterior bloqueaba un snapshot nuevo

Tras la parada confirmada del backend, el shutdown de aplicacion devolvia un
estado actual sin trabajo. El wrapper HTTP seguia fusionando
`cleanup_required` de la llamada anterior porque `MarkShutdownSnapshotV0`
reemplazaba refs y contador, pero no `ShutdownGoalActions`. El resultado era un
`stop_pending` indefinido con todos los contadores a cero.

Cierre local:

- cada snapshot nuevo invalida las acciones derivadas del anterior;
- la respuesta downstream vuelve a persistir las acciones actuales si siguen
  siendo necesarias;
- una regresion secuencial demuestra `backend_still_running -> snapshot vacio
  -> ready`, sin relajar el guard que conserva trabajo realmente observado.

## Verificacion local

```text
git diff --check
go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server
ok orquesta/cmd/orquesta-server 60.628s
ok orquesta/modulos/orquesta-server 0.444s
```

## Criterio de cierre empirico

1. Compilar un binario nuevo desde un commit limpio.
2. Crear worktree, estado, runtime y goal refs nuevos.
3. Ejecutar de nuevo `T9104` con un solo request.
4. Confirmar que no aparece `material_progress_state_reader_unbound`, que el
   estado material se persiste/proyecta y que el cierre usa tests independientes.
5. Confirmar que el shutdown termina en `shutdown_ready=true` sin señal manual.
6. Conservar recibos y actualizar `BUG-226/227/228`; no declarar cerrado el
   nucleo solo por los tests focales.
