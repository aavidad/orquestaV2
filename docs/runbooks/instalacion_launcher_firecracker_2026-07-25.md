# Instalación del launcher Firecracker del atestador

Fecha: 2026-07-25.

## Alcance y estado

Este procedimiento prepara el launcher root de Firecracker para el perfil
canónico `host-128g-16`. La instalación es deliberadamente bifásica:

1. `--apply` instala candidatos inmutables y prepara primitivas del host, pero
   no enlaza, arranca ni habilita el servicio productivo.
2. `--activate` solo acepta un recibo root-owned que declare superado el E2E
   físico del candidato exacto.

El instalador no cambia, detiene ni redimensiona Bubblewrap. Tampoco borra
versiones anteriores, runtime residual, namespaces o cgroups existentes. Una
ruta existente que no cumpla el contrato se rechaza para revisión operativa.

En este corte aún no existe en el repositorio un ejecutor canónico del E2E
Firecracker físico de 16 microVM. Por tanto se puede preparar y arrancar
temporalmente el candidato, pero **no se debe crear el recibo ni ejecutar
`--activate`** hasta incorporar y ejecutar ese E2E. El recibo no sustituye la
evidencia: es el gate de promoción de una evidencia externa todavía pendiente.

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

En 128 GiB, el techo de cgroups deja 48 GiB nominales al host, launcher,
page cache y variaciones. La reserva operativa canónica de 2 GiB participa en
el preflight, pero no se suma como memoria guest. Dieciséis es un techo estático,
no admisión dinámica: si el host tiene presión real, el scheduler debe lanzar
menos trabajo.

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
- launcher compilado desde el mismo corte.
- los seis SHA-256 esperados, obtenidos de la revisión/build reproducible y
  fijados antes de invocar el instalador;
- UID/GID del consumidor no-root y UID/GID exclusivos del jail. UID y GID del
  consumidor deben ser distintos de los del jail; los del jail no pueden ser
  cero.

El peer Unix se valida con UID y GID efectivos, no con grupos suplementarios:
el proceso consumidor debe conectar exactamente con `allowed-uid` y
`allowed-gid`.

El instalador nunca llama a `sudo`. `--apply`, `--check` y `--activate` exigen
que el operador ya esté en una sesión root. `--dry-run` es no-root.

No se debe calcular un hash de una fuente arbitraria y pasarlo inmediatamente
como si fuera procedencia. Los hashes esperados son entradas de confianza. El
instalador compara todos antes de ejecutar `firecracker --version` o
`jailer --version`.

Si cualquier modo se ejecuta con EUID 0, las seis fuentes deben ser
`root:root`, sin escritura de grupo/otros, link count 1 y con todos sus
ancestros `root:root` no sustituibles. Esto también cubre launcher, kernel e
initrd, que serán ejecutables directa o indirectamente después. Una ruta bajo
un directorio de usuario o `/tmp` se rechaza aunque sus bytes tengan el hash
correcto. Preparar previamente un staging root-owned dedicado.

El instalador Bash no conserva un descriptor ejecutable para las seis fuentes
durante toda la transacción. Reduce la ventana con hashes esperados, metadatos
estables, ancestros no sustituibles y reverificación de la copia instalada,
pero un proceso root concurrente sigue dentro de la frontera de confianza del
operador. No ejecutar otra mutación root sobre el staging durante
`--apply`, `--check` o `--activate`. El launcher instalado sí abre y fija sus
assets mediante descriptor antes de servir peticiones. Sustituir esta parte del
instalador por `openat2`/`fexecve` queda como endurecimiento P2 separado; no se
presenta aquí como garantía ya cerrada.

## 1. Dry-run no-root

Todas las entradas son flags explícitos; el instalador no admite configuración
por variables de entorno.

```bash
scripts/install_firecracker_attestor_launcher.sh \
  --dry-run \
  --profile host-128g-16 \
  --launcher-source /ABS/build/orquesta-firecracker-launcher \
  --firecracker-source /ABS/firecracker-v1.16.1/firecracker \
  --jailer-source /ABS/firecracker-v1.16.1/jailer \
  --kernel-source /ABS/kernel/vmlinux \
  --guest-source /ABS/guest/guest.cpio.gz \
  --guest-manifest-source /ABS/guest/guest.manifest.json \
  --launcher-sha256 <LAUNCHER_SHA256_REVISADO> \
  --firecracker-sha256 <FIRECRACKER_SHA256_REVISADO> \
  --jailer-sha256 <JAILER_SHA256_REVISADO> \
  --kernel-sha256 <KERNEL_SHA256_REVISADO> \
  --guest-sha256 <GUEST_SHA256_REVISADO> \
  --guest-manifest-sha256 <MANIFEST_SHA256_REVISADO> \
  --allowed-uid 1000 \
  --allowed-gid 1000 \
  --jail-uid 65534 \
  --jail-gid 65534
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

- launcher y helper en `/usr/local/libexec`, con nombre por hash y modo `0755`;
- Firecracker, jailer, kernel, `guest.cpio.gz` y manifiesto en
  `/usr/local/lib/orquesta/firecracker`, por hash;
- configuración root-owned `0400` en `/etc/orquesta/firecracker`;
- unidades versionadas root-owned `0444` en `/etc/systemd/system`;
- backing root `root:<allowed_gid>` `0750` y marker root-owned `0400`;
- bind exacto a `/run/orquesta`;
- netns fijado `/run/netns/orquesta-firecracker-attestor-empty`, `nsfs`
  root-owned `0600`, solo loopback DOWN, sin direcciones ni rutas;
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

## 4. Arranque temporal del candidato

`--apply` imprime `unit_path` y `primitives_unit_path`. El basename de
`unit_path` es una unidad versionada ya visible para systemd, pero no habilitada.
Desde una sesión root:

```bash
systemctl daemon-reload
systemctl start orquesta-firecracker-attestor-<UNIT_SHA256>.service
systemctl is-active --quiet \
  orquesta-firecracker-attestor-<UNIT_SHA256>.service
```

Esto arranca el candidato versionado directamente; no crea ni cambia el enlace
productivo y no toca Bubblewrap. La unidad depende de su preparador versionado.

El E2E futuro que autorice promoción deberá ejecutar mediante el socket
canónico, como el UID/GID permitido, al menos:

1. una atestación real que arranque el initrd, ejecute un test y devuelva
   evidencia válida;
2. una ola concurrente de 16 solicitudes con guest 4096 MiB, cgroup 5 GiB,
   512 PIDs y cuota 200000/100000;
3. observación durante la ola de `memory.swap.max=0` en cada cgroup;
4. ausencia de red guest, API socket, vsock y consola/serial;
5. al terminar, cero procesos Firecracker/jailer propios, cero hijos del parent
   cgroup y cero directorios de runs residuales.

Después de capturar la evidencia, detener solo la unidad versionada probada:

```bash
systemctl stop orquesta-firecracker-attestor-<UNIT_SHA256>.service
systemctl is-active --quiet \
  orquesta-firecracker-attestor-<UNIT_SHA256>.service
```

El segundo comando debe devolver estado no activo. No deshabilitar ni parar
Bubblewrap.

Hasta que exista el ejecutor canónico de esos cinco puntos, el procedimiento se
detiene aquí. No se improvisa un recibo manual.

## 5. Recibo y activación futura

Cuando el E2E canónico exista y quede verde, su cierre debe escribir un fichero
root-owned, regular, `0400`, link count 1 y con estas líneas exactas:

```text
schema=orquesta_firecracker_activation_receipt.v1
status=passed
config_sha256=<CONFIG_SHA256>
unit_sha256=<UNIT_SHA256>
primitives_unit_sha256=<PRIMITIVES_UNIT_SHA256>
launcher_sha256=<LAUNCHER_SHA256>
asset_digest=<EXPECTED_ASSET_DIGEST_PUBLICADO_POR_APPLY>
e2e_suite=orquesta.firecracker-attestor.physical-16.v1
max_concurrent_runs=16
physical_microvm_count=16
all_attestations_valid=true
zero_residual_runs=true
network_absent=true
memory_swap_max_zero=true
```

Entonces se ejecuta el mismo comando explícito de instalación con
`--activate` en lugar de `--apply` y se añade:

```text
--e2e-receipt /ABS/evidencia/firecracker-activation.receipt
```

La activación:

1. vuelve a verificar artefactos, runtime, bind, netns y cgroup;
2. instala una copia content-addressed del recibo;
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
