# Resultado físico Firecracker 1 + 16 — 2026-07-27

## Resultado

El candidato construido desde `001f13f6f5673d8aefdfb91ebef727aa92b5debf`
completó el E2E físico de una microVM y la ola paralela de 16 microVM. El
driver terminó con estado `0`, verificó la limpieza y activó:

```text
orquesta-firecracker-attestor-57ede79dee780873a0012f3518eeea668de2135a78a3ca4478d7c3c555a7bb09.service
```

Estado observado tras la activación:

```text
ActiveState=active
SubState=running
Result=success
NRestarts=0
runs_residuales=0
cgroups_hijos_residuales=0
vmm_residuales=0
```

Evidencia durable root-only:

```text
/var/lib/orquesta/firecracker-e2e/evidence-001f13f6f567-b675061f5d1d50631f4d5586c1c61fbf.json
/var/lib/orquesta/firecracker-e2e/receipt-001f13f6f567-b675061f5d1d50631f4d5586c1c61fbf.txt
/var/lib/orquesta/firecracker-e2e/e2e-001f13f6f567-b675061f5d1d50631f4d5586c1c61fbf.log
```

## Causa cerrada

Dos capturas syscall independientes coincidieron: Jailer completaba montaje,
`pivot_root`, dispositivos, `setgid` y `setgroups`, pero `setuid` devolvía
`EPERM`. La hipótesis anterior sobre el modo del VMM era falsa y quedó
revertida.

Para desbloquear el primer E2E se usó un perfil bootstrap explícito sin
`CapabilityBoundingSet` y con `NoNewPrivileges=no`. Esto acredita el arranque,
la carga y la limpieza, pero no demuestra cuál de las dos directivas eliminaba
la capacidad efectiva ni constituye el perfil endurecido final.

## Pendiente separado

1. Integrar la bitácora durable `firecrackeraudit` en launcher/supervisor.
2. Reintroducir bounding y NNP de uno en uno, repitiendo el E2E físico.
3. Mantener el candidato actual como bootstrap funcional hasta tener evidencia
   equivalente del perfil endurecido.

