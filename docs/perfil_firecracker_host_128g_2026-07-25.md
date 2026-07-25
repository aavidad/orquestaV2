# Perfil Firecracker para host de 128 GiB

Fecha de dimensionado: 2026-07-25.

Este perfil configura el atestador `microvm` para admitir hasta 16 ejecuciones
simultáneas en un host dedicado de 128 GiB. No cambia el default portable de
`test_attestor.max_concurrent_runs`, que sigue siendo 2.

```toml
[repository.local]
seed_path = "/ruta/absoluta/al/repositorio"

[runtime]
max_output_bytes = 67108864

[test_attestor]
provider = "microvm"
max_subject_bytes = 536870912
max_concurrent_runs = 16

[test_attestor.microvm]
launcher_socket = "/run/orquesta/firecracker-launcher.sock"
guest_memory_mib = 4096

[test_attestor.resources]
memory_max_bytes = 5368709120
pids_max = 512
cpu_quota_micros = 200000
```

El presupuesto nominal de cgroup es `16 × 5 GiB = 80 GiB`. En un host de
128 GiB deja 48 GiB nominales para el sistema, el launcher, page cache y
variación de carga. Cada guest recibe 4 GiB y el cgroup conserva 1 GiB adicional.
El guest puede alojar dos copias del sujeto máximo de 512 MiB, una salida máxima
de 64 MiB y 2 GiB de reserva operativa.

Con el periodo canónico de 100 000 microsegundos del protocolo Firecracker,
`cpu_quota_micros = 200000` representa 2 vCPU por microVM: 32 vCPU nominales
para 16 ejecuciones. La metadata canónica del cross-validator fija la cuota
máxima de una microVM en 3 200 000 microsegundos, equivalente a las 32 vCPU
máximas del protocolo actual; el loader rechaza antes del arranque cualquier
cuota microVM superior. Bubblewrap conserva el rango compartido de la clave
hasta 10 000 000 microsegundos: el límite específico no se le aplica.

Los defaults portables quedan en guest de 4 GiB, cgroup de 5 GiB, 512 PIDs y
dos ejecuciones. `runtime.max_output_bytes` conserva su default global porque
también limita las salidas de agentes; los 64 MiB anteriores son un override
explícito de este despliegue. El mínimo guest de 128 MiB, el margen de cgroup de
1 GiB, la reserva operativa de 2 GiB y la cuota máxima microVM viven en la
metadata del registro canónico y se publican en los artefactos generados; el
loader no mantiene una segunda copia de esos límites.

La configuración surtirá efecto cuando el despliegue active
`test_attestor.provider = "microvm"` y conecte el launcher. No modifica ni
redimensiona el proveedor Bubblewrap que esté vivo. La admisión dinámica basada
en memoria, CPU y presión reales del host sigue pendiente en el runner físico:
el límite de 16 es un techo estático, no una promesa de disponibilidad. Este
corte acredita validación de configuración y aritmética; no acredita un E2E de
16 microVM ni una prueba de carga Firecracker.
