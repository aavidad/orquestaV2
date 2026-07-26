# Instalación del launcher Firecracker del atestador

Fecha: 2026-07-25.

## Alcance y estado

Este procedimiento prepara el launcher root de Firecracker para el perfil
canónico `host-128g-16`. La instalación es deliberadamente bifásica:

1. `--apply` instala candidatos inmutables y prepara primitivas del host, pero
   no enlaza, arranca ni habilita el servicio productivo.
2. `--activate` solo acepta el receipt V2 y la evidencia root-owned del E2E
   físico superado por el candidato exacto.

El instalador no cambia, detiene ni redimensiona Bubblewrap. Tampoco borra
versiones anteriores, runtime residual, namespaces o cgroups existentes. Una
ruta existente que no cumpla el contrato se rechaza para revisión operativa.

El ejecutor canónico ya existe en
`cmd/orquesta-firecracker-attestor-e2e`. A fecha de este corte **no se ha
ejecutado físicamente**. El toolchain fijado
`/srv/orquesta-self/toolchains/go1.25.11` ya está instalado y el build host
reproducible dispone de su receipt de procedencia. Quedan el stage privilegiado
y la secuencia física `1 + 16`. Por tanto todavía no hay activación, evidencia
física ni receipt de activación V2 válido. Ese receipt no sustituye la
evidencia: ambos son entradas obligatorias de `--activate`.

La reserva concurrente de CID vsock para agentes generales es otro corte y no
forma parte de este launcher del atestador. Existe contrato y adaptador SQL
transaccional restart-safe en estado `planned_not_applied`, pero sus tablas aún
no están integradas en la migración canónica ni hay wiring físico. Ningún paso
de este runbook debe crear un vsock, elegir un CID, aplicar ese schema
manualmente o añadir NIC/TAP/NAT. El `TestAttestor` conserva red y vsock
ausentes.

## Capacidad fijada

Los valores salen del registro y del perfil vigente
`docs/perfil_firecracker_host_128g_2026-07-25.md`:

| Recurso | Por microVM | Máximo de 16 |
|---|---:|---:|
| Memoria guest | 4096 MiB | 64 GiB |
| `memory.max` del cgroup | 5 GiB | 80 GiB |
| CPU, periodo 100000 µs | cuota 200000 µs (2 vCPU) | 32 vCPU nominales |
| PIDs | 512 | 8192 |
| Sujeto | 512 MiB | 8 GiB |

El gate exige al menos `16 × 5 GiB + 2 GiB = 82 GiB` físicos. El host de esta
instalación, comercializado como 128 GB, expone 122,97 GiB: deja unos 40,97 GiB
antes de descontar consumo vivo del host. La reserva operativa canónica de
2 GiB ya participa en el preflight. Dieciséis es un techo estático, no admisión
dinámica: si el host tiene presión real, el scheduler debe lanzar menos trabajo.

El envelope del launcher deriva el input raw máximo como `512 MiB + 1 MiB de
metadata + 1024 bytes de framing`. La salida capturada queda en 64 MiB y el
drive de resultados conserva el máximo físico de 4 MiB del protocolo.

## Disco y bind persistente

`/run` de este host solo ofrece unos 13 GiB, insuficientes para el peor caso de
16 ejecuciones. Los datos viven en disco:

```text
/srv/orquesta-self/runtime/firecracker-attestor
    bind rw,nosuid
/run/orquesta
```

El segundo path mantiene el socket canónico
`/run/orquesta/firecracker-launcher.sock` y satisface el contrato del launcher,
que exige que socket y runtime tengan el mismo directorio. El helper verifica
con `findmnt` el target y `FSROOT` exactos y también la identidad de
dispositivo/inode. Si el target ya contiene datos o es otro mountpoint, falla y
no lo reemplaza.

El bind permite `exec` y dispositivos: `noexec` impediría ejecutar el jailer y
`nodev` inutilizaría el `/dev/kvm` que jailer crea dentro del chroot. No se crea
tmpfs ni se afirma que las microVM residan en RAM; llevar el runtime a memoria
queda como evolución separada con dimensionado y recovery propios.

Antes de instalar, el preflight exige espacio en el filesystem de `/srv` para
16 veces: input máximo, output drive, kernel, initrd `guest.cpio.gz`, 128 MiB
de overhead y un 10 % adicional. No reserva el espacio; evita una promoción
obviamente imposible.

## Prerrequisitos

- Linux amd64 con `/dev/kvm`.
- cgroup v2 con `cpu`, `memory` y `pids`.
- systemd, `iproute2`, `findmnt`, `mountpoint`, `python3` y utilidades GNU.
- binarios Firecracker y jailer exactamente `1.16.1`.
- kernel compatible.
- initrd reproducible `guest.cpio.gz` y su manifiesto completo generado por
  `scripts/build_firecracker_attestor_guest.sh`.
- launcher y supervisor E2E compilados desde el mismo corte.
- los siete SHA-256 esperados, obtenidos de la revisión/build reproducible y
  fijados antes de invocar el instalador;
- UID/GID del consumidor no-root y UID/GID exclusivos del jail. UID y GID del
  consumidor deben ser distintos de los del jail; los del jail no pueden ser
  cero.

El peer Unix se valida con UID y GID efectivos, no con grupos suplementarios:
el proceso consumidor debe conectar exactamente con `allowed-uid` y
`allowed-gid`.

El instalador nunca llama a `sudo`. `--apply`, `--check` y `--activate` exigen
que el operador ya esté en una sesión root. `--dry-run` es no-root. El wrapper
de handoff sí abre la ventana de privilegio con `sudo -v` antes de invocar su
secuencia; esa elevación pertenece al operador/wrapper, no al instalador.

No se debe calcular un hash de una fuente arbitraria y pasarlo inmediatamente
como si fuera procedencia. Los hashes esperados son entradas de confianza. El
instalador compara todos antes de ejecutar `firecracker --version` o
`jailer --version`.

Si cualquier modo se ejecuta con EUID 0, las siete fuentes deben ser
`root:root`, sin escritura de grupo/otros, link count 1 y con todos sus
ancestros `root:root` no sustituibles. Esto también cubre launcher, supervisor,
kernel e initrd, que serán ejecutables directa o indirectamente después. Una
ruta bajo un directorio de usuario o `/tmp` se rechaza aunque sus bytes tengan
el hash correcto. Preparar previamente un staging root-owned dedicado.

El instalador Bash no conserva un descriptor ejecutable para las siete fuentes
durante toda la transacción. Reduce la ventana con hashes esperados, metadatos
estables, ancestros no sustituibles y reverificación de la copia instalada,
pero un proceso root concurrente sigue dentro de la frontera de confianza del
operador. No ejecutar otra mutación root sobre el staging durante
`--apply`, `--check` o `--activate`. El launcher instalado sí abre y fija sus
assets mediante descriptor antes de servir peticiones. Sustituir esta parte del
instalador por `openat2`/`fexecve` queda como endurecimiento P2 separado; no se
presenta aquí como garantía ya cerrada. Antes de elevar privilegios, copiar
también el propio instalador a ese staging, verificar su SHA-256 y ejecutar la
copia root-owned; no ejecutar como root el script mutable del checkout.

## 1. Dry-run no-root

Todas las entradas son flags explícitos; el instalador no admite configuración
por variables de entorno.

```bash
scripts/install_firecracker_attestor_launcher.sh \
  --dry-run \
  --profile host-128g-16 \
  --launcher-source /ABS/build/orquesta-firecracker-launcher \
  --supervisor-source /ABS/build/orquesta-firecracker-attestor-e2e \
  --firecracker-source /ABS/firecracker-v1.16.1/firecracker \
  --jailer-source /ABS/firecracker-v1.16.1/jailer \
  --kernel-source /ABS/kernel/vmlinux \
  --guest-source /ABS/guest/guest.cpio.gz \
  --guest-manifest-source /ABS/guest/guest.manifest.json \
  --launcher-sha256 <LAUNCHER_SHA256_REVISADO> \
  --supervisor-sha256 <SUPERVISOR_SHA256_REVISADO> \
  --firecracker-sha256 <FIRECRACKER_SHA256_REVISADO> \
  --jailer-sha256 <JAILER_SHA256_REVISADO> \
  --kernel-sha256 <KERNEL_SHA256_REVISADO> \
  --guest-sha256 <GUEST_SHA256_REVISADO> \
  --guest-manifest-sha256 <MANIFEST_SHA256_REVISADO> \
  --allowed-uid 1000 \
  --allowed-gid 1000 \
  --jail-uid 65432 \
  --jail-gid 65432
```

Revisar primero `expected_asset_digest=<SHA256>`. Es el digest canónico que
calcula el launcher sobre este JSON, sin salto final y conservando este orden:

```json
{"schema":"orquesta.firecracker-launcher.assets.v1","firecracker_sha256":"...","jailer_sha256":"...","kernel_sha256":"...","guest_sha256":"...","guest_manifest_sha256":"..."}
```

Ese valor se copia sin prefijo a la clave canónica de Orquesta
`test_attestor.microvm.expected_asset_digest`. Vincula el cliente con el juego
exacto de Firecracker, jailer, kernel, guest y manifiesto instalado. No se
calcula en configuración con una segunda convención.

Revisar también las cuatro secciones renderizadas:

- JSON canónico de 31 claves, sin desconocidas, con hashes SHA-256 fijados;
- helper de primitivas;
- unidad oneshot versionada de primitivas;
- unidad versionada y endurecida del launcher.

El instalador valida el manifiesto con el mismo esquema vigente del launcher:
digests, versiones acotadas de BusyBox y Go, commit, reproducibilidad, `unpacked_bytes`,
`minimum_guest_memory_mib`, contrato de memoria y
`toolchain_nobody_go_test`.

## 2. Stage con `--apply`

Ejecutar el mismo comando cambiando únicamente `--dry-run` por `--apply`, desde
una sesión root. La salida publica los paths content-addressed exactos.

El stage crea o verifica:

- launcher, supervisor E2E y helper en `/usr/local/libexec`, con nombre por
  hash y modo `0755`;
- Firecracker, jailer, kernel, `guest.cpio.gz` y manifiesto en
  `/usr/local/lib/orquesta/firecracker`, por hash;
- configuración root-owned `0400` en `/etc/orquesta/firecracker`;
- unidades versionadas root-owned `0444` en `/etc/systemd/system`;
- backing root `root:<allowed_gid>` `0750` y marker root-owned `0400`;
- bind exacto a `/run/orquesta`;
- netns fijado `/run/netns/orquesta-firecracker-attestor-empty`, `nsfs`,
  root-owned, un enlace y modo seguro `[46][04][04]` —el montaje puede exponer
  `0444` aunque se solicite `0600`—, solo loopback DOWN, sin direcciones ni
  rutas;
- parent cgroup v2 `/sys/fs/cgroup/orquesta-firecracker-attestor`, root-owned
  `0755`, vacío, con `cpu`, `memory` y `pids` delegados.

El launcher materializa por ejecución `cpu.max`, `memory.max`,
`memory.swap.max=0`, `memory.oom.group=1` y `pids.max`. El instalador exige que
el controlador de swap exista antes de promover.

`--apply` publica además `expected_asset_digest`, pero no ejecuta
`systemctl daemon-reload`, `start` ni `enable`, y no crea
`/etc/systemd/system/orquesta-firecracker-attestor.service`.

## 3. Verificación idempotente

Ejecutar el mismo juego de flags con `--check`. Debe terminar con:

```text
check=ok
```

Una segunda pasada de `--apply` con exactamente los mismos artefactos verifica
los inodes instalados y no los reemplaza. Un mismo nombre content-addressed con
bytes, ownership o modo distintos se rechaza.

El test fake incluye dos regresiones de procedencia: una fuente non-root no
puede pasar la guarda root-trusted y un SHA-256 esperado incorrecto corta antes
de ejecutar el fake `--version`.

Pruebas no-root del instalador:

```bash
bash -n \
  scripts/install_firecracker_attestor_launcher.sh \
  scripts/test_install_firecracker_attestor_launcher.sh
shellcheck \
  scripts/install_firecracker_attestor_launcher.sh \
  scripts/test_install_firecracker_attestor_launcher.sh
scripts/test_install_firecracker_attestor_launcher.sh
```

## 4. E2E físico del candidato staged

La secuencia obligatoria es: reconstruir launcher y supervisor desde un objeto
Git exacto con Go `1.25.11` fijado, publicar el recibo de build, verificar ese
recibo, `--apply`, `systemctl daemon-reload`, comprobar que el candidato está
**inactivo**, ejecutar el E2E físico `1 + 16`, validar cero residual y solo
entonces `--activate`.
No se arranca manualmente la unidad antes del E2E: el ejecutor inicia y detiene
la unidad versionada candidata y comprueba su identidad estable.

El bloque ejecutable histórico que construía y lanzaba `3451d310…` queda
retirado; no se sustituye aquí por una receta ejecutable. El operador debe usar
exclusivamente el wrapper identificado en el handoff actual, tras verificar
sus identidades estáticas y con la autorización privilegiada correspondiente.

`buildvcs=false` es deliberado para conservar bytes reproducibles, por lo que
`go version -m` puede mostrar `(devel)`. La procedencia no se infiere de ese
texto del ELF: vive en
`orquesta-firecracker-host-build.receipt.json`, publicado **después** de ambos
binarios como marcador de commit del bundle. El builder compila dos veces con
cachés aisladas desde un `git archive` privado y exacto; el recibo fija:

- commit y tree Git;
- SHA-256 del archivo fuente y del objeto commit;
- SHA-256 del builder exacto;
- versión, SHA-256 del ejecutable Go y SHA-256 del árbol completo del
  toolchain;
- receta y entorno cerrados (`vendor`, `trimpath`, `buildvcs=false`, `CGO=0`,
  `linux/amd64`, `GOENV=off`, `GOTOOLCHAIN=local`, red de módulos desactivada,
  locale/tiempo fijos y cachés privadas);
- tamaño, modo, paquete y SHA-256 de launcher y supervisor.

El driver privilegiado no confía en un JSON sustituible. Fija por constantes
el SHA-256 del recibo y del verificador, los copia primero a staging
`root:root`, vuelve a comprobar ambos hashes y ejecuta allí la validación
estricta contra los dos ELF staged. Un cambio de fuente, toolchain, receta,
campo JSON, binario o verificador corta antes de `--apply`. El recibo queda
fuera de los binarios que acredita, como exige la regla de sujetos inmutables.

### Handoff actual del candidato (evidencia estática; no E2E)

El bloque ejecutable anterior para `3451d3108c0c14b9539bec1ca134afd07a9beb19`
queda retirado por obsoleto. No debe reutilizarse ni inferirse de él una
activación, receipt, nonce, resultado PASS o ejecución física.

El candidato `815c1f295391a077a6023a10637a7d52afab1eb6` se ejecutó y falló
limpiamente antes de arrancar una microVM. La evidencia física mostró que
Linux 7.0 exponía `/proc/net/route` vacío en el netns limpio; el launcher lo
rechazaba por exigir la cabecera histórica. `21d256bf` corrige esa
representación sin aceptar rutas y `156fd972` evita que la ausencia legítima
de `runs` después de parar enmascare la causa primaria. Los incidentes quedan
registrados como `BUG-ORQ-20260727-570` y `BUG-ORQ-20260727-571`.

El único handoff vigente es el wrapper
`/home/alberto/Trabajo/orquesta/script/ejecutar_firecracker_root_v2.sh`, sobre
la revisión `ec6dfc98c764d6613a95401d4c7395b75063a547` y árbol
`829e39af0fef4e3c65241d83c3316a5f0eb2f1f7`. Sus identidades estáticas son:

```text
driver_sha256=38d4c3870c4246c13440ce1ea5e9578a1e2e50b8cff1df1bf28401c6fd161cc4
wrapper_sha256=c75e0660cd6d7dfc223ac75077218575276b7ec5bca69b807b4569fb4809d9f7
driver_source=/tmp/orquesta-firecracker-root-v2.rfuWOn/run-root-e2e.sh
driver_target=/srv/orquesta-self/operator/firecracker-root-e2e-ec6dfc98c764d6613a95401d4c7395b75063a547
asset_digest=7df880f7cbfbf82d076254b474c46c6e85065a921a819308482343613a83d1f0
candidate_unit=orquesta-firecracker-attestor-0a0b436cf1e452f072fb94509168b8b9897e3cea43ded5865150fc0aaa8f595a.service
```

El wrapper, no el instalador, solicita `sudo -v` como preflight de operador.
El driver permanece inactivo hasta validar la secuencia física `1 + 16` del
candidato exacto. Esta documentación solo fija el handoff estático: no afirma
que el wrapper se haya ejecutado, que exista evidencia/receipt, que haya PASS,
que se haya emitido nonce ni que la unidad esté activada.

El E2E realiza una atestación física y después una ola física de 16; verifica
límites, `memory.swap.max=0`, red/API/vsock/serial ausentes y deja unidad,
procesos, cgroup y directorios de runs sin residuo. No habilita el alias
productivo ni toca Bubblewrap. El estado actual bloquea este paso porque falta
la evidencia `1 + 16`; no se debe fabricar un receipt manual. El SHA del
supervisor se obtiene del binario construido con flags reproducibles y se
entrega al instalador como `--supervisor-sha256`. `--apply` instala la copia
root-owned content-addressed y `--activate` exige que la evidencia física cite
exactamente ese hash.

La spec es JSON estricto (sin claves desconocidas) y debe ser un fichero regular
`root:root`, enlace único, `0400`, con ruta absoluta canónica y ancestros
`root:root` sin escritura de grupo/otros. Su forma exacta es:

```json
{"candidate":{"unit_name":"orquesta-firecracker-attestor-36fbd58046da0b9ad0fd5e68188412b98324bd0ad1a478e51d2d648f829d7d1d.service","unit_path":"/etc/systemd/system/orquesta-firecracker-attestor-36fbd58046da0b9ad0fd5e68188412b98324bd0ad1a478e51d2d648f829d7d1d.service","unit_sha256":"36fbd58046da0b9ad0fd5e68188412b98324bd0ad1a478e51d2d648f829d7d1d","primitives_unit_path":"/etc/systemd/system/orquesta-firecracker-primitives-<PRIMITIVES_UNIT_SHA256>.service","primitives_unit_sha256":"<PRIMITIVES_UNIT_SHA256>","launcher_path":"/usr/local/libexec/orquesta-firecracker-launcher-<LAUNCHER_SHA256>","launcher_sha256":"<LAUNCHER_SHA256>","launcher_socket_path":"/run/orquesta/firecracker-launcher.sock","config_path":"/etc/orquesta/firecracker/launcher-<CONFIG_SHA256>.json","config_sha256":"<CONFIG_SHA256>","supervisor_path":"/usr/local/libexec/orquesta-firecracker-attestor-e2e-<SUPERVISOR_SHA256>","supervisor_sha256":"<SUPERVISOR_SHA256>","runtime_root":"/run/orquesta","cgroup_root":"/sys/fs/cgroup","parent_cgroup":"orquesta-firecracker-attestor","netns_path":"/run/netns/orquesta-firecracker-attestor-empty","asset_digest":"7df880f7cbfbf82d076254b474c46c6e85065a921a819308482343613a83d1f0"},"policy_digest":"","evidence_path":"/var/lib/orquesta/firecracker-e2e/evidence-<NONCE>.json","receipt_path":"/var/lib/orquesta/firecracker-e2e/receipt-<NONCE>.txt","phase_timeout":"10m","cleanup_timeout":"60s","stable_for":"2s","poll_interval":"250ms","child_uid":1000,"child_gid":1000,"workload":{"socket_path":"/run/orquesta/firecracker-launcher.sock","expected_asset_digest":"7df880f7cbfbf82d076254b474c46c6e85065a921a819308482343613a83d1f0","work_root":"/var/lib/orquesta/firecracker-e2e/work-<NONCE>","git_command":"/usr/bin/git","test_sleep":"15s","attestation_timeout":"5m","attestation_cleanup_timeout":"60s","max_output_bytes":67108864,"max_subject_bytes":536870912}}
```

`candidate.launcher_socket_path` debe ser idéntico a
`workload.socket_path`; `candidate.asset_digest` debe ser idéntico a
`workload.expected_asset_digest`. El supervisor es un binario content-addressed
root-owned y sus ancestros son seguros. El `work_root` ya existe, es hijo
privado `0700`, vacío y propiedad del UID/GID no-root hijo. Los padres de
`evidence_path` y `receipt_path` son `root:root` seguros; los outputs se crean
sin sobrescribirlos.

`policy_digest` puede ser `""` en la spec inicial. La fase física de una VM
descubre el digest real y la fase de 16 exige exactamente el mismo valor; el
receipt y la evidencia finales siempre contienen un SHA-256 válido.

## 5. Activación tras evidencia real

Con receipt V2 y evidencia producidos por ese E2E, se ejecuta el mismo comando
explícito de instalación con `--activate` y ambos flags:

```text
--e2e-receipt /ABS/evidencia/firecracker-activation.receipt
--e2e-evidence /ABS/evidencia/firecracker-e2e.json
```

La activación:

1. vuelve a verificar artefactos, runtime, bind, netns y cgroup;
2. instala copias content-addressed de la evidencia y del recibo ligado a ella;
3. cambia atómicamente el symlink canónico a la unidad ya probada;
4. habilita la unidad versionada real, nunca el alias enlazado que systemd
   rechaza, arranca esa unidad y valida que quede activa;
5. detiene y deshabilita la unidad previa solo si su estado anterior lo exige;
6. si falla, restaura exactamente el symlink anterior y los estados
   activo/inactivo y habilitado/deshabilitado de ambas unidades; un estado
   previo distinto de esos cuatro estados se rechaza antes de mutar.

## Endurecimiento y excepciones necesarias

La unidad principal usa `ProtectSystem=strict`, `ProtectHome=yes`,
`PrivateTmp=yes`, mount namespace privado, device policy cerrada salvo
`/dev/kvm` y `/dev/net/tun` con `rwm`, más `/dev/null` y `/dev/urandom`,
familias `AF_UNIX`/`AF_NETLINK`,
`IPAddressDeny=any`, bounding de capabilities y `NoNewPrivileges=yes`.

No se pueden activar en el launcher:

- `PrivateDevices=yes`: oculta KVM;
- `PrivateNetwork=yes`: interfiere con el `setns` gobernado del jailer;
- `ProtectControlGroups=yes`: impide materializar cgroups por run;
- `RestrictNamespaces=yes`: impide jailer/netns;
- un `SystemCallFilter` no acreditado: puede cortar mount, setns, KVM o cgroup.

Preparar el bind y el netns desde `ExecStartPre` de esa unidad endurecida
tampoco funciona: las restricciones de filesystem crean un mount namespace y
los mounts del prestart no sobreviven de forma fiable al `ExecStart`. Por eso
hay una unidad oneshot versionada separada. Esa unidad usa
`PrivateMounts=no`, `ProtectSystem=no`, `PrivateNetwork=no`,
`ProtectControlGroups=no` y `RestrictNamespaces=no`, con las capabilities
acotadas a `CHOWN`, `DAC_OVERRIDE`, `FOWNER`, `NET_ADMIN` y `SYS_ADMIN`.
La unidad principal depende causalmente de ella y repite un `--check` sin
mutaciones antes de arrancar.

Estas excepciones son frontera explícita de systemd, no bypass oculto.
El permiso `m` de KVM/TUN es necesario porque jailer 1.16.1 crea esos nodos
dentro del chroot; `DevicePolicy=closed` sin `m`, o sin TUN, deja el servicio
aparentemente endurecido pero rompe el arranque real.

## Rollback

### Rollback Bubblewrap de emergencia

Firecracker continúa como proveedor principal; Bubblewrap es solo rollback. El
rollback seguro usa el mismo binario nuevo y la configuración
`/home/alberto/Trabajo/.orquesta-runtime-v2-v23/Codex12/config/orquesta.rollback-bubblewrap-65536.toml`
con SHA-256 `c9028735362b6a95c5bc7589851ab56358b3d9656fee59ab51e2f1f41ef7df2f`.
El preflight real de Bubblewrap (SHA `cc4e8c8…`) superó 44.992 argumentos.
No volver al binario `5b5` tras migraciones 017--019 ni restaurar o tocar
SQLite/WAL. Esta alternativa no acredita Firecracker activo ni un E2E físico.

Los candidatos anteriores nunca se borran. Antes de una promoción manual,
guardar target y estados previos:

```bash
readlink /etc/systemd/system/orquesta-firecracker-attestor.service
systemctl is-enabled orquesta-firecracker-attestor-<SHA_ANTERIOR>.service
systemctl is-active orquesta-firecracker-attestor-<SHA_ANTERIOR>.service
```

El rollback normal desde un candidato activo hacia una versión anterior que
estaba habilitada y activa se hace sobre los nombres versionados reales. Nunca
se pasa el alias canónico a `enable` o `disable`:

```bash
sudo bash <<'ROLLBACK'
set -euo pipefail
umask 0077

CANONICAL_UNIT=/etc/systemd/system/orquesta-firecracker-attestor.service
CANDIDATE_UNIT=orquesta-firecracker-attestor-<SHA_CANDIDATO>.service
PREVIOUS_UNIT=orquesta-firecracker-attestor-<SHA_ANTERIOR>.service
CANDIDATE_PATH=/etc/systemd/system/$CANDIDATE_UNIT
PREVIOUS_PATH=/etc/systemd/system/$PREVIOUS_UNIT

for UNIT_PATH in "$CANDIDATE_PATH" "$PREVIOUS_PATH"; do
  UNIT_NAME="$(basename "$UNIT_PATH")"
  [[ "$UNIT_NAME" =~ ^orquesta-firecracker-attestor-([0-9a-f]{64})[.]service$ ]]
  test "$(realpath -e "$UNIT_PATH")" = "$UNIT_PATH"
  test -f "$UNIT_PATH"
  test ! -L "$UNIT_PATH"
  test "$(sha256sum "$UNIT_PATH" | awk '{print $1}')" = "${BASH_REMATCH[1]}"
done
test "$(realpath -e "$CANONICAL_UNIT")" = "$CANDIDATE_PATH"

systemctl stop "$CANDIDATE_UNIT"
systemctl disable "$CANDIDATE_UNIT"

ROLLBACK_DIR="$(mktemp -d \
  /etc/systemd/system/.orquesta-firecracker-rollback.XXXXXXXX)"
cleanup_rollback_dir() {
  if [[ -n "${ROLLBACK_DIR:-}" && -d "$ROLLBACK_DIR" ]]; then
    if [[ -L "$ROLLBACK_DIR/unit" ]]; then
      unlink "$ROLLBACK_DIR/unit"
    fi
    rmdir "$ROLLBACK_DIR"
  fi
}
trap cleanup_rollback_dir EXIT
ln -s "$PREVIOUS_PATH" "$ROLLBACK_DIR/unit"
mv -T \
  "$ROLLBACK_DIR/unit" \
  "$CANONICAL_UNIT"
rmdir "$ROLLBACK_DIR"
ROLLBACK_DIR=""
systemctl daemon-reload
systemctl enable "$PREVIOUS_UNIT"
systemctl restart "$PREVIOUS_UNIT"

test "$(realpath -e "$CANONICAL_UNIT")" = "$PREVIOUS_PATH"
systemctl is-enabled --quiet "$PREVIOUS_UNIT"
systemctl is-active --quiet "$PREVIOUS_UNIT"
! systemctl is-enabled --quiet "$CANDIDATE_UNIT"
! systemctl is-active --quiet "$CANDIDATE_UNIT"
ROLLBACK
```

Antes de ejecutar, comprobar que ambos nombres cumplen exactamente
`orquesta-firecracker-attestor-<64 hex>.service` y que el SHA-256 del fichero
coincide con los 64 hex del nombre. El instalador hace esa allowlist y esa
comprobación antes de consultar o mutar systemd; un alias previo hacia otro
servicio falla sin `stop`, `disable`, `restart` ni cambio del enlace.

Si los estados guardados de la versión anterior no eran `enabled` y `active`,
restaurar esos estados registrados en lugar de los supuestos del bloque. El
rollback automático durante `--activate` admite solo los estados exactos
`enabled`/`disabled` y `active`/`inactive`; si no puede restaurarlos informa
`activation_failed_rollback_incomplete`.

Si era la primera activación, el rollback consiste en
`systemctl stop "$CANDIDATE_UNIT"`, `systemctl disable "$CANDIDATE_UNIT"` y
retirar solo el symlink canónico después de comprobar que no hay procesos ni
runs. Después se hace `daemon-reload` y se verifican candidato inactivo,
candidato deshabilitado y alias ausente. Los artefactos, recibos y unidades
versionadas se conservan para auditoría y una reactivación recuperable.

No desmontar ni borrar backing, marker, netns o parent cgroup como parte de un
rollback normal. Si alguno queda malformado, registrar evidencia y tratarlo
como limpieza gobernada separada.
