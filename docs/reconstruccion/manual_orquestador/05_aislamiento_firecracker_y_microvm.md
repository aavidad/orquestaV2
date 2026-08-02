# 05. Aislamiento de agentes con Firecracker y microVM

> **Responsabilidad:** definir cómo construir el adaptador de agentes aislados
> con Firecracker y una microVM por agente.
>
> **Alcance:** confianza, activos medidos, red, credenciales, recursos, datos,
> lanzamiento, cierre, fallos, pruebas físicas y retirada.
>
> **No acredita:** este capítulo describe el objetivo y la ruta de
> implementación; no acredita que el entorno de ejecución de agentes
> Firecracker exista ni esté activado.

## Al terminar este capítulo, el lector sabrá…

- distinguir atestación de pruebas y ejecución aislada de agentes;
- definir la frontera de confianza y medir todos los activos de una microVM;
- limitar red, credenciales, recursos, entrada y salida de cada agente;
- recuperar, detener, inventariar y conservar una microVM sin perder pruebas;
- diseñar cohortes físicas que acrediten aislamiento y gestión elástica.

## Índice del capítulo

- [Vocabulario](#0-vocabulario-mínimo), [decisión fundamental](#1-decisión-fundamental), [estado actual](#2-estado-actual-honesto) y [mecanismos](#3-tres-mecanismos-distintos).
- [Confianza](#4-frontera-de-confianza), [activos](#5-activos-medidos), [anfitrión](#6-preparación-y-admisión-del-anfitrión) y [red](#7-red-sin-interfaz-solo-vsock).
- [Credenciales](#8-credenciales-mínimas-y-efímeras), [datos](#9-espacio-de-trabajo-y-artefactos), [ejecución](#10-secuencia-física-de-una-ejecución) y [conservación](#11-conservación-y-revisión).
- [Amenazas](#12-amenazas-y-controles), [recuperación](#13-fallos-y-recuperación), [oleadas](#14-oleadas-físicas-y-lógicas) y [diagnóstico](#15-diagnóstico-seguro).
- [Tareas](#16-plan-de-construcción-en-tareas-acotadas), [acreditación](#17-criterio-de-acreditación) y [fuentes](#18-lecciones-y-fuentes).

## 0. Vocabulario mínimo

- Firecracker: monitor de máquinas virtuales ligero que crea microVM sobre KVM.
- KVM: capacidad del núcleo Linux que permite virtualización asistida por el
  procesador.
- microVM: máquina virtual pequeña y de propósito acotado.
- programa de confinamiento (`jailer`): auxiliar de Firecracker que reduce
  privilegios y encierra el proceso.
- `virtio-vsock` o `vsock`: canal de comunicación entre anfitrión y máquina
  virtual que no requiere una interfaz de red.
- CID: número de contexto que encamina una conexión `vsock`; no es una
  credencial ni una prueba de identidad.
- CAS: almacén direccionado por contenido; cada objeto se recupera y verifica
  por su huella criptográfica.
- ciclo de vida (`lifecycle`): secuencia autoritativa de estados de una entidad.
- arrendamiento (`lease`): concesión temporal y exclusiva sobre un recurso.
- cerca (`fence`): ordinal que invalida una reserva o escritor anterior.
- buzón causal (`mailbox`): intercambio durable ligado a identidades,
  generación y ejecución exactas.
- recibo (`receipt`): registro estructurado de un intento, resultado o efecto.
- huella criptográfica (`digest` o `hash`): resumen verificable de código,
  imagen, configuración o artefacto.
- grupo de control: mecanismo Linux, también conocido como `cgroup`, que limita
  y contabiliza CPU, memoria, procesos y entrada/salida.
- conservado pendiente de revisión (`preserved_pending_review`): identificador
  literal que exige entorno detenido, inventariado y sellado, pero prohíbe
  borrarlo automáticamente.

## 1. Decisión fundamental

Firecracker es un adaptador de infraestructura, no parte del núcleo. La
aplicación pide «ejecutar un agente con estas restricciones» mediante puertos
neutrales. La composición elige Firecracker, un proceso local u otro proveedor.
El dominio no conoce `/dev/kvm`, imágenes, núcleos, zócalos, dispositivos,
identificadores de contexto virtual ni órdenes del sistema operativo.

La unidad de aislamiento es:

```text
una ejecución de agente -> una microVM -> un sistema de archivos aislado
```

No se introducen varios agentes dentro de una microVM para abaratar recursos.
Eso mezclaría identidades, credenciales, escritura, fallos y recibos. Un perfil
de proveedor puede servir sucesivamente a varias ejecuciones, pero cada
ejecución obtiene su propio entorno causal.

Firecracker tampoco se convierte en requisito universal. Es una opción de
aislamiento fuerte, activada por política, capacidad del anfitrión y riesgo.
El orquestador debe seguir funcionando con otros adaptadores conformes.

## 2. Estado actual honesto

| Pieza | Estado | Situación comprobada |
|---|---|---|
| Separación hexagonal del entorno de ejecución de agentes | ACREDITADO | Existen puertos neutrales de lanzamiento, observación y control. |
| Atestador de pruebas con Bubblewrap | ACREDITADO | V17 acredita la composición exacta de `TestAttestor` y su candidato sellado; no acredita Bubblewrap en general ni un entorno de agentes. |
| Cliente y lanzador externo Firecracker para atestación | PARCIAL | Hay código, protocolo, huésped y prueba física de atestación; su activación exige recibo del anfitrión y no acredita agentes. |
| Política de red causal del agente | PARCIAL | Existen contratos y plan determinista sin interfaz de red, con `vsock`; el estado declarado es planificado y no aplicado (`planned_not_applied`). |
| Asignación de CID con arrendamiento y cerca | PARCIAL | Existen puerto y adaptador SQL con recuperación y protección contra reutilización antigua; falta conectarlo a la migración y composición canónicas. |
| Prueba de credencial de un solo uso | PARCIAL | Existe contrato que vincula almacén de credenciales, atestación y desafío; falta el servicio físico completo. |
| Descriptor de lote y plan de lanzamiento | PARCIAL | Están modelados; el propio plan declara pendiente la aplicación física y la verificación completa desde CAS. |
| Adaptador Firecracker de `AgentLauncher`, observador y controlador | PENDIENTE | No existe composición productiva para agentes. |
| Huésped específico de agente | PENDIENTE | No existe imagen medida con Codex, Git, corredor y protocolo de agente. |
| Corredor `vsock`, representante de red y canal de resultados | PENDIENTE | No existe la integración física completa. |
| Sellado e inventario de entorno | PENDIENTE | No existe el flujo durable de conservación pendiente de revisión (`preserved_pending_review`). |
| Demanda lógica del entorno de agentes | PENDIENTE | El contrato planificado V38 exige exactamente 1/16/70/500 ejecuciones durables. Esta serie no afirma simultaneidad física y no tiene recibos. |
| Pasos físicos del entorno de agentes | PENDIENTE | El mismo contrato exige exactamente 1/5/10/16/20 agentes simultáneos sobre recursos medidos. No incluye 70 ni 500 y no tiene recibos. |

### 2.1 Estado canónico de V38

La revisión actual de `product/roadmap.json` contiene 257 capacidades, 38
verticales y 38 contratos. V1–V22 están acreditadas y V23–V38 continúan
planificadas. V38 `agent_runtime_elastic` es canónica y prioritaria, pero no
está implementada, conectada, ejercitada ni acreditada por ello.

V38 posee solo `ORC-28`. `ORC-15`, `OPS-16` y `OPS-17` aparecen como
requisitos no propios que V38 no reasigna ni acredita: siguen respectivamente
en V27, V32 y V32.

V23 y V38 son independientes en ambos sentidos. V23 no depende de Firecracker,
KVM ni microVM para acreditar el Wizard; V38 y Firecracker tampoco dependen de
cerrar V23. Firecracker continúa siendo un adaptador opcional y su presencia,
sus binarios o sus pruebas parciales no acreditan V38.

### 2.2 Compuertas A, B y C

V38 separa el núcleo neutral de la infraestructura física:

| Compuerta | Alcance exacto | Efecto de estado |
|---|---|---|
| A | Núcleo elástico neutral; no requiere KVM ni Firecracker. | No puede acreditar V38. |
| B | Adaptador Firecracker para agentes, activado expresamente, sin sustitución automática y con una microVM por agente. | No puede acreditar V38 por sí sola. |
| C | Ola física real 1/5/10/16/20 sobre el mismo candidato que superó A y B. | Solo permite acreditar V38 después de A y B. |

Solo A+B+C superadas por el mismo candidato permiten acreditar V38. La
caracterización neutral de una microVM conserva
`agent_microvm_network=planned_not_applied`; no completa B, no ejecuta C y no
crea recibo ni evidencia.

### 2.3 Cuota, capacidad externa y contrato físico

`AGT-12` y `ORC-28` son obligaciones distintas:

- `AGT-12` pertenece a V25 y descubre disponibilidad, cuota y límites del
  proveedor;
- `ORC-28` representa agrupadores, licencias y plazas como capacidad externa y
  pertenece canónicamente a V38.

Si V38 necesita `AGT-12`, la dependencia debe añadirse y adoptarse primero en
el catálogo. Este capítulo no puede crearla por inferencia.

`ORC-15` sigue en V27 y `OPS-16`/`OPS-17` en V32. V38 exige continuidad de
mensajes, parada exacta y conservación como conductas no propias; no mueve ni
vuelve a acreditar esas tres capacidades.

Los puertos neutrales permiten otros adaptadores de ejecución. Eso no rebaja
las compuertas B y C de V38, que exigen una microVM Firecracker por agente. Una
composición local de procesos o Bubblewrap puede satisfacer A o su contrato
propio, pero no sustituye la prueba física Firecracker.

Es esencial no promover piezas parciales por parecido:

- una microVM que ejecuta pruebas no es una microVM que aloja un agente;
- un plan determinista no demuestra que se aplicó;
- un zócalo accesible no demuestra autenticación;
- un proceso lanzado no demuestra trabajo útil ni aislamiento;
- una prueba con un agente no demuestra gestión elástica.

## 3. Tres mecanismos distintos

### 3.1 Bubblewrap para atestación de pruebas

Bubblewrap crea aislamiento por espacios de nombres en el mismo núcleo Linux.
V17 acredita su composición exacta de `TestAttestor` sobre el candidato
sellado de aquella revisión; no acredita Bubblewrap en general ni un entorno de
agentes. Su entrada rechaza rutas absolutas, componentes `..`, enlaces y tipos
especiales no permitidos antes de materializar el sujeto.

El paso físico de 16 agentes pertenece a la compuerta C de V38. Ejecutar
dieciséis atestaciones Bubblewrap no acredita ese paso de dieciséis agentes
Firecracker: cambian el sujeto, la imagen, el proceso y el recibo.

### 3.2 Firecracker para atestación de pruebas

El código existente crea una microVM efímera sin red, con discos de entrada y
salida, activos fijados, grupo de control y un lanzador privilegiado externo.
El huésped ejecuta el protocolo de atestación, no un agente interactivo. Sirve
como fuente de patrones de seguridad, pero no debe compartirse ni ampliarse
hasta convertirlo accidentalmente en entorno de ejecución de agentes.

### 3.3 Firecracker para agentes

Necesita otra imagen, otro protocolo y otro ciclo operativo:

- sesión de larga duración;
- órdenes y progreso bidireccionales;
- acceso controlado a proveedor y herramientas;
- espacio de trabajo mutable dentro de la microVM;
- devolución de cambios y artefactos;
- salud, aviso, relevo y parada;
- conservación inspeccionable.

Los tres mecanismos pueden reutilizar contratos criptográficos o utilidades
pequeñas si la frontera es realmente común. No comparten ciclo de vida, estado
privado ni autoridad de cierre.

## 4. Frontera de confianza

| Código | Invariante | Consecuencia |
|---|---|---|
| `sin_atestacion_remota_de_hardware` | El diseño actual no incorpora una atestación remota de hardware como SEV-SNP, TDX o una raíz remota equivalente. | El saludo autenticado y las huellas locales no demuestran frente a un tercero que el anfitrión o su administrador sean confiables. |

La base de cómputo de confianza, conocida como TCB, incluye como mínimo:

- hardware y firmware del anfitrión;
- núcleo Linux y KVM;
- administrador del anfitrión;
- lanzador externo y programa de confinamiento;
- constructor de núcleos e imágenes;
- activos medidos, configuración y políticas cargadas.

Comprometer cualquiera de esas piezas puede falsear una medición o leer la
memoria del huésped. La acreditación debe declarar este límite; no debe llamar
«atestación remota» a una comprobación local de huellas.

El proceso principal de Orquesta se ejecuta sin privilegios de administración.
No abre `/dev/kvm`, no crea dispositivos de red, no configura grupos de control
privilegiados y no invoca directamente el programa de confinamiento
(`jailer`).

Un **lanzador externo** mínimo recibe por un zócalo local privado —`socket` en
la interfaz del sistema— una orden cerrada y validable. El zócalo vive en una
ruta administrativa no atravesable, con propietario, grupo y modo exactos;
valida las credenciales del proceso par y no escucha por TCP. Es el único
componente autorizado para las operaciones de anfitrión necesarias. Debe:

- autenticar al cliente local;
- aceptar solo un esquema versionado y una lista cerrada de operaciones;
- verificar huellas y propietarios de activos;
- imponer rutas, permisos y límites;
- crear una raíz confinada (`chroot`), un espacio de nombres de montajes
  (`mount namespace`) y montajes cerrados;
- aplicar `no_new_privs`, filtro seccomp y lista permitida de descriptores
  heredados (`FD allowlist`);
- aplicar grupo de control v2 (`cgroup v2`) con CPU, memoria, procesos y
  entrada/salida verificables;
- aceptar solo argumentos de arranque y dispositivos enumerados por política;
- crear exactamente una microVM por orden idempotente;
- devolver identidad externa y recibo;
- rechazar parámetros libres de consola o rutas arbitrarias;
- limpiar solo recursos que él mismo creó e identificó.

La aplicación conserva la decisión y el gobierno del efecto. El lanzador no
elige trabajo, prioridad, presupuesto, credenciales, red permitida ni cuándo
el `Goal` termina.

Pruebas negativas mínimas:

- usuario o grupo no autorizado no abre el zócalo;
- ruta absoluta, `..`, enlace o cambio de propietario se rechazan;
- descriptor adicional no permitido no llega a Firecracker;
- argumento de arranque, dispositivo o montaje no enumerado se rechaza;
- el proceso no puede recuperar privilegios tras `no_new_privs`;
- una llamada prohibida por seccomp termina el entorno sin ampliar permisos;
- superar un límite del grupo de control v2 no afecta a otra microVM;
- una raíz fuera del confinamiento no puede leerse ni montarse.

## 5. Activos medidos

Una ejecución reproducible referencia por huella:

- binario Firecracker;
- binario de confinamiento (`jailer`);
- núcleo Linux;
- sistema de archivos raíz específico del agente;
- manifiesto de paquetes del huésped;
- corredor interno y versión del protocolo;
- binarios necesarios de Codex y Git;
- identidad del corredor `vsock` del anfitrión;
- identidad del representante de salida de red;
- configuración efectiva redactada;
- política de recursos;
- lote de entrada;
- política de red y credenciales.

Los activos base son inmutables, de propietario administrativo y no escribibles
por Orquesta ni por el agente. Cada ejecución crea capas o discos propios. La
acreditación se refiere a sus huellas, no a etiquetas flotantes como
`latest` —«último»—.

El manifiesto del huésped declara:

- versiones y procedencia;
- arquitectura;
- bibliotecas y certificados;
- usuarios y permisos;
- servicios habilitados;
- puntos de montaje;
- protocolo de arranque y apagado;
- fecha y herramienta de construcción;
- licencia y comprobación de vulnerabilidades;
- procedimiento de retirada.

La imagen del atestador de pruebas no se reutiliza como imagen de agente.

## 6. Preparación y admisión del anfitrión

Antes de reservar una microVM se miden:

- presencia y permisos de KVM;
- CPU y virtualización compatibles;
- memoria libre y comprometida;
- espacio e inodos para capas, lotes y conservación;
- procesos, hilos y descriptores;
- capacidad del grupo de control;
- intervalo disponible de CID;
- estado del lanzador y sus zócalos;
- huellas de activos esperadas;
- capacidad del corredor y del representante de red.

La admisión descuenta reservas vivas, margen del anfitrión y coste del propio
orquestador. El límite es explícito y observable; no existe un máximo oculto.

La política de recursos por microVM incluye:

- memoria fija y, si se admite, mecanismo explícito de sobrecompromiso;
- número de CPU virtuales;
- peso o cuota de CPU;
- límite de procesos e hilos;
- entrada/salida de disco;
- tamaño máximo de lote de entrada y salida;
- tiempo de arranque, inactividad y ejecución;
- máximo de mensajes y transferencia;
- presupuesto de red y proveedor.

Los valores se justifican con medición. No se copia la fórmula del huésped de
atestación: el agente tiene otra carga y requiere su propio perfil.

## 7. Red: sin interfaz, solo `vsock`

La microVM de agente no recibe interfaz Ethernet, TAP, puente, NAT ni dirección
IP. El único transporte con el anfitrión es `virtio-vsock`.

La política predeterminada prohíbe:

- Internet directo;
- red local del anfitrión;
- bucle local del anfitrión;
- rangos privados IPv4;
- rangos locales únicos IPv6;
- direcciones de enlace local;
- servicios de metadatos de nube;
- tráfico lateral entre microVM;
- conexiones entrantes no solicitadas.

Si una herramienta necesita red, el huésped solicita una operación semántica a
un representante del anfitrión. Este aplica lista de destinos y operaciones,
autorización, presupuesto, tiempo máximo, tamaño y recibo. No entrega un
conector TCP general.

### 7.1 CID con arrendamiento y cerca

El CID de `vsock` permite encaminar; no autentica. Se asigna mediante un
registro durable con:

- intervalo configurado;
- ejecución y microVM propietarias;
- arrendamiento, caducidad y cerca;
- revisión esperada;
- marca de máximo histórico para evitar reutilización ABA;
- recuperación y liberación idempotentes.

Una cerca antigua no puede registrar servicios ni consumir mensajes. Tras un
reinicio se cotejan registro, microVM reales y sesiones antes de reasignar.

### 7.2 Servicios separados

Conviene separar al menos:

- canal de control y buzón;
- entrega de lote de entrada;
- devolución de cambios y artefactos;
- representante de red;
- servicio de credenciales;
- salud y apagado.

Cada servicio tiene identidad, versión, límites y autorización propios. Que una
microVM alcance el corredor no le concede todos los servicios.

## 8. Credenciales mínimas y efímeras

El agente nunca recibe el almacén de credenciales, variables globales del
anfitrión ni un archivo con todos los secretos.

El flujo recomendado es:

1. la aplicación aprueba una necesidad exacta para una ejecución;
2. el `CredentialStore` crea una concesión de alcance mínimo;
3. el servicio verifica identidad de microVM, atestación, CID, cerca y un
   desafío aleatorio de un solo uso;
4. entrega un valor de un solo uso o una credencial efímera;
5. el huésped acusa recepción sin reproducir el secreto;
6. la concesión caduca, se consume o se revoca;
7. el recibo conserva referencias y huellas, nunca el valor.

El CID no sustituye la autenticación. La identidad debe estar vinculada a la
ejecución, activos medidos, desafío aleatorio de arranque y canal. Los secretos
no aparecen
en políticas, órdenes, eventos, diarios, artefactos, inventarios ni
configuración efectiva.

## 9. Espacio de trabajo y artefactos

La microVM no monta directamente el árbol de trabajo del anfitrión ni comparte
el zócalo de Git.

### 9.1 Entrada

La aplicación construye un lote inmutable desde el almacén de contenido:

- instantánea o archivos permitidos;
- referencias de artefactos;
- intención y contrato de aceptación;
- contexto mínimo;
- configuración pública;
- manifiesto con tamaños, tipos y huellas.

Antes de exponerlo al huésped se valida:

- huella de cada contenido;
- tamaño individual y expansión total;
- rechazo de rutas absolutas o con `..`;
- rechazo de enlaces simbólicos y duros;
- dispositivos, tuberías y archivos especiales;
- colisiones por mayúsculas, normalización o separadores;
- permisos ejecutables;
- versión del esquema.

El plan actual modela parte de esta frontera, pero la lectura física desde CAS
y todas estas validaciones siguen pendientes.

### 9.2 Trabajo dentro del huésped

El agente opera sobre un disco o capa privada. Git, si se usa, vive dentro del
huésped y no recibe credenciales del anfitrión salvo concesión exacta. El
agente no envía cambios al repositorio remoto, fusiona, despliega ni publica
por sí mismo: devuelve una propuesta y Orquesta gobierna cualquier efecto
posterior.

### 9.3 Salida

El huésped devuelve:

- conjunto de cambios o lote de archivos;
- artefactos por contenido;
- pruebas ejecutadas y resultados;
- mensajes finales;
- manifiesto de tamaños, tipos, permisos y huellas;
- recibo firmado o autenticado por el canal.

El anfitrión valida la salida con las mismas defensas de rutas y expansión. La
integración en un espacio de trabajo ocurre fuera de la microVM, con revisión
esperada, identidad, autorización y recibo.

## 10. Secuencia física de una ejecución

### 10.1 Lanzamiento

1. derivar ejecución lista y admitirla por capacidad;
2. reservar CPU, memoria, disco, proveedor y CID;
3. resolver activos por huella;
4. generar desafío aleatorio, política y lote sellado;
5. persistir intención, aprobación e intento durable con clave idempotente,
   revisión y cerca;
6. enviar al lanzador la orden idempotente y cerrada del intento ya persistido;
7. crear grupo de control y directorio privado;
8. materializar capa de escritura y dispositivos permitidos;
9. arrancar Firecracker mediante el programa de confinamiento (`jailer`);
10. verificar proceso, grupo de control y huellas;
11. realizar saludo autenticado por `vsock`;
12. en una sola transacción con revisión esperada y cerca vigente, vincular
    microVM, identidad externa, CID, sesión y ejecución; persistir el recibo;
    consumir la reserva, cerrar el intento y publicar la acción de observación;
13. entregar la orden inicial mediante la salida transaccional;
14. comenzar supervisión y renovación.

Un acuse del lanzador solo prueba admisión. El recibo de microVM arrancada, el
saludo del huésped y el trabajo útil son evidencias distintas.

| Código | Invariante físico |
|---|---|
| `intento_durable_antes_del_efecto` | No se invoca el lanzamiento (`Launch`) hasta que intención, aprobación, intento, idempotencia y cerca están persistidos. |

Pseudocódigo del lanzamiento:

```text
lanzar_agente(ejecucion, reserva):
    exigir_cerca_vigente(ejecucion, reserva)
    activos = resolver_y_verificar_huellas(ejecucion.perfil)
    cid = asignar_cid_con_arrendamiento(ejecucion)
    lote = construir_y_sellar_entrada(ejecucion)
    orden = autorizar_efecto(ejecucion, reserva, activos, cid, lote)
    intento = persistir_intencion_aprobacion_e_intento(
        orden,
        clave_idempotente,
        cerca,
        revision_esperada
    )

    respuesta = lanzador.aplicar_idempotente(intento)
    si respuesta.definitivamente_no_aplicada:
        cerrar_intento_y_liberar(cid, reserva)
        devolver_a_espera(la_misma_ejecucion)
    si respuesta.aplicacion_desconocida:
        poner_en_cuarentena_y_consultar_lanzador(
            intento.clave_idempotente,
            intento.identidad_externa_esperada
        )
    si respuesta.aplicada:
        saludo = autenticar_huesped(respuesta.microvm, cid, orden.desafio)
        transaccion_con_revision_y_cerca_vigentes:
            vincular_ejecucion_microvm_y_sesion(ejecucion, respuesta, saludo)
            persistir_recibo_de_lanzamiento(respuesta.identidad_externa)
            consumir_reserva_y_cerrar_intento(reserva, intento)
            publicar_accion_de_observacion()
```

Si Orquesta cae después del lanzamiento y antes de esa transacción, la
recuperación consulta al lanzador por clave idempotente e identidad externa. Si
encuentra la microVM, repite el saludo y completa la misma transacción bajo una
cerca vigente. Solo reintenta cuando el lanzador demuestra que el efecto no se
aplicó; una respuesta incierta permanece en cuarentena.

### 10.2 Ejecución

La aplicación mantiene:

- buzón causal recuperable;
- latidos y progreso estructurados;
- límites de recursos y presupuesto;
- renovación de arrendamientos;
- revocación de credenciales;
- puntos de control;
- recibos de herramientas y artefactos.

El agente no escribe el ciclo de vida. Propone resultado; el motor valida
contratos, pruebas y evidencia.

### 10.3 Parada, sellado e inventario

El orden exacto evita corrupción y pérdida:

1. bloquear nuevas órdenes;
2. solicitar punto de control;
3. pedir apagado cooperativo al agente;
4. esperar hasta el plazo y verificar;
5. si procede, apagar forzosamente la microVM exacta con autorización;
6. cerrar y revocar credenciales y representantes;
7. asegurar que no quedan procesos del huésped;
8. sincronizar capas y salidas;
9. calcular huellas e inventario;
10. sellar discos, lotes, cambios, diarios y recibos;
11. persistir el estado conservado pendiente de revisión
    (`preserved_pending_review`);
12. desmontar dispositivos;
13. liberar CID, grupo de control, zócalos y recursos activos;
14. verificar cero procesos propios no reclamados.

Desmontar no es borrar. La capa y el inventario se conservan hasta una decisión
posterior, autorizada e idempotente de retirada.

Pseudocódigo de cierre:

```text
cerrar_y_conservar(vinculo):
    bloquear_nuevas_ordenes(vinculo)
    pedir_punto_de_control(vinculo)
    resultado = solicitar_apagado_cooperativo(vinculo, plazo)
    si no resultado.termino:
        exigir_autorizacion_forzada_y_cerca_vigente()
        lanzador.detener_exactamente(vinculo.microvm)

    revocar_credenciales_y_servicios(vinculo)
    sincronizar_y_verificar_salida(vinculo)
    inventario = calcular_inventario_y_huellas(vinculo)
    sellar(inventario)
    persistir_estado("conservado_pendiente_revision", inventario)
    desmontar_y_liberar_recursos_activos_sin_borrar(inventario)
```

## 11. Conservación y revisión

El inventario de un entorno conservado contiene:

- ejecución, generación, intento, agente y microVM;
- activos medidos;
- tiempos de creación, parada y sellado;
- motivo y modo de parada;
- tamaños, permisos, propietarios y huellas;
- referencias de entrada, salida, capa y diarios;
- credenciales revocadas;
- recibos de cierre;
- incidencias o datos pendientes de estudio;
- política y fecha mínima de conservación.

El estado conservado pendiente de revisión (`preserved_pending_review`) no
significa que el entorno siga ejecutable. Los
procesos, canales de escritura y credenciales están cerrados; solo persisten
los datos sellados.

La retirada posterior exige una orden diferente con principal, alcance,
objetos exactos, motivo, aprobación, idempotencia y recibo. Un fallo a mitad de
retirada se reconcilia sin ampliar el conjunto de objetos.

## 12. Amenazas y controles

| Amenaza | Control obligatorio |
|---|---|
| Escape del agente al anfitrión | MicroVM, programa de confinamiento (`jailer`), usuario sin privilegios, activos inmutables, superficie mínima y actualización medida. |
| Uso arbitrario de KVM o del lanzador | Proceso principal sin privilegios, zócalo protegido, cliente autenticado y esquema cerrado. |
| Acceso a red interna | Sin interfaz de red; representante semántico con lista permitida y recibos. |
| Suplantación por CID | Autenticación ligada a ejecución, desafío aleatorio, medición local de activos, arrendamiento y cerca; nunca se presenta como atestación remota de hardware. |
| Robo o repetición de secreto | Concesión mínima, efímera, de un solo uso y revocable. |
| Lectura del espacio de trabajo anfitrión | Lotes sellados; ningún montaje directo. |
| Escritura fuera del lote | Validación de rutas, enlaces, dispositivos, expansión y permisos. |
| Agotamiento de recursos | Grupo de control, cuotas de disco, límites de mensajes, presupuesto y admisión. |
| Reutilización de identidad antigua | Cerca, máximo histórico de CID y saludo causal. |
| Resultado falso del agente | Revisión independiente y atestación sobre los mismos cambios y pruebas. |
| Borrado para ocultar una incidencia | Conservación obligatoria y retirada como efecto separado. |
| Lanzamiento duplicado tras caída | Idempotencia, identidad externa recuperable y cuarentena de resultado desconocido. |
| Argumento, dispositivo o descriptor adicional | Esquema cerrado, lista permitida, `no_new_privs`, seccomp y rechazo antes de ejecutar. |
| Administrador o constructor comprometido | Declarar la base de confianza; rotar y volver a medir activos. El diseño actual no puede demostrar lo contrario remotamente. |

El aislamiento no convierte al agente en confiable. Limita el daño; la
autorización, la revisión y los recibos siguen siendo necesarios.

## 13. Fallos y recuperación

| Fallo | Respuesta |
|---|---|
| KVM ausente o sin permisos | Marcar capacidad no disponible; mantener la ejecución esperando; no degradar silenciosamente. |
| Activo con huella o propietario incorrectos | Rechazar antes de lanzar y emitir recibo de precondición. |
| CID agotados | Contrapresión; no reutilizar uno vivo ni ampliar el intervalo sin configuración. |
| Caída tras crear la microVM y antes del recibo | Consultar al lanzador por clave idempotente e identidad; no relanzar a ciegas. |
| Saludo `vsock` no autenticado | No entregar lote ni credenciales; detener y conservar. |
| Representante de red caído | Mantener trabajo local si es seguro o esperar; no abrir red directa. |
| Cuota de proveedor agotada | Punto de control y relevo según evidencia estructurada. |
| Disco o memoria bajo presión | Detener nuevas admisiones, conservar las ejecuciones afectadas y aplicar política explícita. |
| Agente silencioso | Comprobar salud, avisar y pedir punto de control; no matar solo por silencio. |
| Apagado cooperativo vencido | Autorizar parada forzada exacta, inventariar y conservar. |
| Inventario incompleto | No marcar como conservado pendiente de revisión (`preserved_pending_review`); mantener cuarentena y recursos necesarios para recuperar. |
| Desmontaje fallido | Registrar incidencia y reintentar de forma idempotente; nunca borrar para desbloquear. |
| Reinicio del orquestador | Cotejar estado durable, lanzador, procesos, CID, grupos y sesiones antes de adoptar o detener. |

El lanzador debe ofrecer inventario de recursos propios para recuperar
microVM que existen aunque el último recibo se perdiera.

## 14. Oleadas físicas y lógicas

El contrato planificado V38 fija dos series exactas y separadas.

Demanda lógica 1/16/70/500:

1. una ejecución durable: recorrido causal mínimo completo;
2. dieciséis: generación íntegra y admisión sin recorte oculto;
3. setenta: espera y progreso por oleadas sin pérdida;
4. quinientas: convergencia completa con capacidad física inferior.

Pasos físicos 1/5/10/16/20:

1. una microVM simultánea: flujo completo, fallo inducido y conservación;
2. cinco: carreras de CID, zócalos, grupos y credenciales;
3. diez: presión y reinicio del orquestador;
4. dieciséis: presión combinada sobre aislamiento, reservas y recuperación;
5. veinte: fallo parcial del lanzador y parada individual.

Que 1 y 16 aparezcan en ambas series no fusiona sus sujetos. La demanda lógica
prueba que todas las ejecuciones existen y esperan o progresan sin pérdida; el
paso físico prueba simultaneidad en un anfitrión con perfil, presupuesto y
  presión registrados. Ni 70 ni 500 son pasos físicos. Las cifras son
  canónicas, pero continúan sin acreditarse hasta superar A+B+C sobre el mismo
  candidato.

Para cada tamaño lógico o paso físico se mide:

- latencia de creación y saludo;
- memoria residente y comprometida;
- CPU de huésped, lanzador y orquestador;
- disco, inodos y velocidad de sellado;
- uso y reciclaje seguro de CID;
- descriptores, procesos e hilos;
- capacidad del corredor y del representante;
- tiempo de parada y conservación;
- recursos huérfanos;
- efecto de reiniciar Orquesta y el lanzador.

Un fallo en la oleada detiene el aumento, no borra los entornos. Se estudia la
causa y se repite la misma escala antes de avanzar.

## 15. Diagnóstico seguro

Antes de reiniciar o detener nada:

1. identificar ejecución, microVM, CID, cerca y clave de idempotencia;
2. leer intención, aprobación, intento y recibo;
3. consultar el inventario del lanzador sin pedir una nueva creación;
4. cotejar proceso Firecracker, programa de confinamiento (`jailer`), grupo de
   control y directorio;
5. revisar el arrendamiento de CID y el saludo autenticado;
6. comprobar huellas y propietarios de los activos;
7. inspeccionar métricas de CPU, memoria, disco, procesos y canal;
8. clasificar el efecto como no aplicado, aplicado o desconocido;
9. usar la operación de recuperación prevista para esa clase.

| Síntoma | Comprobación inicial | Acción segura |
|---|---|---|
| La microVM no arranca | KVM, huellas, propietario, memoria, disco y respuesta del lanzador | Corregir la precondición y reintentar la misma orden idempotente |
| Existe proceso pero no recibo | Inventario del lanzador por clave | Adoptar si toda la identidad coincide; si no, poner en cuarentena |
| No hay saludo por `vsock` | CID, desafío, versión de corredor y estado del huésped | No entregar datos ni credenciales; detener y conservar |
| El agente no alcanza una herramienta | Política del representante y recibo de denegación | Corregir autorización explícita; nunca añadir red directa |
| Crece la memoria del anfitrión | Grupo de control, suma de reservas y presión real | Detener admisiones y cerrar individualmente según política |
| El CID parece ocupado dos veces | Arrendamientos, cerca y máximo histórico | Rechazar al propietario antiguo; no reasignar hasta reconciliar |
| No se puede desmontar | Proceso o descriptor que retiene el recurso | Conservar la incidencia y reintentar; no forzar borrado |
| Falta un entorno conservado | Inventario, recibo de sellado y orden de retirada | Tratarlo como incidente de integridad y preservar toda evidencia restante |

No se ejecutan barridos por nombre de proceso, borrados recursivos ni cambios
manuales de CID. El diagnóstico debe conservar la prueba del fallo.

## 16. Plan de construcción en tareas acotadas

La compuerta A se implementa y prueba primero en el núcleo elástico neutral,
sin KVM ni Firecracker. Las tareas siguientes construyen B detrás de sus
puertos y culminan en C; no autorizan a omitir A ni a mezclar las tres
compuertas en una sola tarea.

1. fijar el contrato de perfil de anfitrión y adaptador falso;
2. conectar el registro durable de CID a migración y composición;
3. cerrar el contrato versionado del lanzador externo;
4. implementar cliente no privilegiado y servidor mínimo;
5. construir y medir el sistema raíz específico del agente;
6. implementar verificación física de activos y directorios;
7. implementar capa privada y lote de entrada desde CAS;
8. completar negativos de rutas, enlaces, dispositivos y expansión;
9. implementar saludo autenticado y canal de control por `vsock`;
10. implementar servicio de credenciales de un solo uso;
11. implementar representante de red semántico;
12. implementar devolución validada de cambios y artefactos;
13. conectar `AgentLauncher`, observador y controlador;
14. integrar capacidad, reservas y grupos de control;
15. implementar parada cooperativa y forzada exacta;
16. implementar sellado, inventario y el estado conservado pendiente de
    revisión (`preserved_pending_review`);
17. implementar recuperación conjunta de lanzador, CID y sesiones;
18. ejecutar negativos de seguridad y fallos;
19. ejecutar la demanda lógica 1, 16, 70 y 500;
20. ejecutar los pasos físicos 1, 5, 10, 16 y 20 como compuerta C sobre el
    mismo candidato de A y B;
21. acreditar V38 solo si A+B+C pasan sobre ese candidato y, después, estudiar
    la retirada de mecanismos sustituidos.

Cada tarea mantiene un conjunto de escritura pequeño y un contrato ejecutable.
No debe añadirse un nuevo servicio interno por cada fase: lanzador, corredor y
representante existen fuera de proceso por privilegio o aislamiento; la
política sigue en el monolito modular.

## 17. Criterio de acreditación

La capacidad de agentes Firecracker solo puede acreditarse cuando una misma
revisión demuestre:

- composición productiva seleccionable;
- una microVM por agente;
- base de cómputo de confianza declarada y ausencia explícita de atestación
  remota de hardware;
- activos medidos y configuración efectiva;
- lanzador externo con privilegios mínimos;
- zócalo privado, credenciales de proceso par, confinamiento de montajes,
  `no_new_privs`, seccomp, descriptores cerrados y grupo de control v2;
- red solo por `vsock` y negativos de acceso;
- credencial mínima de un solo uso;
- límites efectivos de CPU, memoria, procesos y disco;
- intención, aprobación e intento durables antes del lanzamiento;
- vínculo, recibo, consumo y acción de observación atómicos después del saludo;
- recuperación por identidad externa e idempotencia en cada frontera de caída;
- entrada y salida selladas;
- mensajería y trabajo útil;
- parada individual cooperativa y forzada;
- inventario y conservación sin borrado;
- recuperación sin duplicados;
- demanda lógica 1/16/70/500 y pasos físicos 1/5/10/16/20, acreditados como
  series distintas dentro de A+B+C;
- revisión independiente sobre las mismas huellas;
- cero procesos, zócalos, grupos o montajes propios no reclamados al cierre.

Un recibo de atestación Firecracker, si existe, no se reutiliza como recibo de
entorno de ejecución de agentes. Son sujetos y contratos distintos.

## 18. Lecciones y fuentes

Las consultas automáticas de lecciones para `AGT-01`, `AGT-03`, `AGT-12`,
`ORC-28` y `EVD-13` no encontraron coincidencias para la ruta de este capítulo.
El hueco se registra; no se rellena copiando el entorno de ejecución antiguo ni
se interpreta como acreditación.

Fuentes vigentes revisadas:

- `AGENTS.md`
- `product/roadmap.json`
- `product/knowledge/tooling_adoption_v1.json`
- `docs/reconstruccion/ruta_total_100.md`
- `docs/reconstruccion/handoff_continuacion_agente_2026-07-30.md`
- `docs/reconstruccion/inventario_herramientas_transversales_2026-07-29.md`
- `docs/reconstruccion/corte_alcance_v23_firecracker_diferido_2026-07-26.md`
- `docs/decision_atestacion_bubblewrap_microvm_2026-07-25.md`
- `docs/arquitectura_comunicacion_agentes_firecracker_2026-07-25.md`
- `docs/corte_agente_firecracker_uno_2026-07-29.md`
- `docs/perfil_firecracker_host_128g_2026-07-25.md`
- `docs/runbooks/instalacion_launcher_firecracker_2026-07-25.md`
- `docs/runbooks/formato_receipt_activacion_firecracker_v2_2026-07-25.md`
- `internal/ports/agent_microvm_network.go`
- `127c0a45`, retirada del antiguo puerto y asignador CID de Orquesta
- `internal/ports/agent_microvm_launch_auth.go`
- `internal/ports/agent_microvm_bundle.go`
- `internal/adapters/agent/firecracker/`
- `internal/adapters/attestor/firecrackerclient/`
- `internal/adapters/attestor/firecrackerlauncher/`
- `internal/e2e/firecrackerattestor/`

Contrato planificado canónico, sin evidencia acreditadora:

- `acceptance/fixtures/v38_agent_runtime_elastic_plan.json`

Antes de implementar se vuelve a consultar el estado canónico y el
`AGENTS.md` del módulo. Este capítulo enseña a construir y verificar el
adaptador; no permite afirmar que Firecracker está activo ni que el producto
está terminado.
