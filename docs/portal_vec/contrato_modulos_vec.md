# Contrato modular VEC para apps administrativas

## Objetivo

Definir una forma unica de enganchar modulos al portal VEC principal. La meta es
que Bolsa, concursos, nominas, permisos, accion social, formacion, CAP,
procesos selectivos o cualquier bloque futuro entren igual: mismo login, mismo
shell, mismo menu, mismas reglas de permisos, misma auditoria, mismo sistema de
notificaciones, mismo i18n y mismas capacidades transversales.

Este documento es una propuesta de discusion para otros agentes. No es una
implementacion cerrada. Sirve para que cualquier agente que programe un modulo
sepa como acoplarse al VEC sin crear otra aplicacion aislada.

## Fuentes revisadas

- Portal del Empleo Publico de la Junta de Andalucia:
  `https://portalempleopublico.juntadeandalucia.es/`
- Sede Electronica del Empleo Publico:
  `https://portalempleopublico.juntadeandalucia.es/sede`
- Catalogo de procedimientos y servicios para personal empleado:
  `https://portalempleopublico.juntadeandalucia.es/sede/acceso-tramites/rps-personal-empleado`
- Web del Empleado Publico Andaluz, mapa del sitio:
  `https://ws045.juntadeandalucia.es/empleadopublico/emp-mapasitio-.html?p=%2FCategorias_Principales%2F`
- Preguntas frecuentes IAAP sobre Web del Empleado Publico y procesos
  selectivos:
  `https://www.juntadeandalucia.es/organismos/iaap/areas/empleo-publico/preguntas-frecuentes.html`

## Decision base

VEC no debe ser una aplicacion monolitica de Bolsa. Debe ser un portal modular
de empleado publico con una zona unica de identificacion para personal interno,
preferentemente Kerberos/AD en red corporativa y adaptadores compatibles para
otros mecanismos permitidos por producto. Una vez autenticado, el usuario puede
saltar de un modulo a otro sin repetir login.

Reglas:

- El shell VEC es propietario de identidad, sesion, layout, menu, permisos
  globales, busqueda global, notificaciones, auditoria transversal e i18n base.
- Cada modulo aporta dominio, casos de uso, rutas, pantallas, permisos propios,
  widgets, eventos, tipos documentales y acciones.
- Ningun modulo implementa su propio login.
- Ningun modulo decide por strings de UI si el usuario puede operar; usa claims
  y permisos del contexto VEC.
- Ningun modulo comparte DB interna ni filesystem interno con el shell.
- Bolsa entra como `vec.module.bolsa`, igual que entrarian Nominas, CAP,
  Permisos o Accion Social.

## Arquitectura objetivo

```text
vec-shell
  identidad Kerberos/AD
  sesion y permisos
  menu modular
  tablero
  busqueda global
  notificaciones
  expediente/documentos comunes
  auditoria comun
  i18n comun
  registro de modulos

vec-core
  contratos de modulo
  contratos de identidad
  contratos de navegacion
  contratos de documentos
  contratos de notificacion
  contratos de auditoria
  contratos de eventos

vec-module-bolsa
  dominio bolsa
  solicitudes
  meritos/RUM
  autobaremacion
  listados
  alegaciones

vec-module-*
  cualquier otro modulo con el mismo contrato
```

La frontera debe seguir arquitectura hexagonal. `vec-core` define contratos y
puertos; `vec-shell` adapta autenticacion, HTTP, UI y composicion; cada modulo
expone casos de uso propios detras de sus puertos. Orquesta puede generar,
probar y revisar estos modulos, pero no debe convertirse en dependencia de
dominio de VEC ni de los modulos.

## Contrato unico de modulo

Todo modulo debe publicar un manifiesto estable. El shell no debe conocer
detalles internos del modulo; solo consume este contrato.

```json
{
  "module_ref": "vec.module.bolsa",
  "version": "v1",
  "title_i18n_key": "module.bolsa.title",
  "description_i18n_key": "module.bolsa.description",
  "category_ref": "seleccion_y_bolsas",
  "base_route": "/bolsa",
  "api_prefix": "/api/modules/bolsa",
  "required_roles": ["employee", "candidate_manager"],
  "menu_entries": [
    {
      "entry_ref": "bolsa.solicitudes",
      "label_i18n_key": "module.bolsa.menu.solicitudes",
      "route": "/bolsa/solicitudes",
      "required_permissions": ["bolsa.solicitud.read"]
    }
  ],
  "widgets": [
    {
      "widget_ref": "bolsa.pending_actions",
      "slot": "dashboard.summary",
      "required_permissions": ["bolsa.dashboard.read"]
    }
  ],
  "capabilities": [
    "expediente",
    "documentos",
    "notificaciones",
    "auditoria",
    "tramites"
  ],
  "events_published": [
    "bolsa.solicitud_registrada",
    "bolsa.autobaremo_calculado",
    "bolsa.alegacion_presentada"
  ],
  "events_subscribed": [
    "documento.validado",
    "notificacion.leida"
  ],
  "health_route": "/api/modules/bolsa/healthz"
}
```

Campos obligatorios:

| Campo | Uso |
| --- | --- |
| `module_ref` | Identidad opaca, estable y unica del modulo. |
| `version` | Version del contrato del modulo. |
| `category_ref` | Agrupacion del menu principal. |
| `base_route` | Ruta frontend propiedad del modulo. |
| `api_prefix` | Prefijo API del modulo. |
| `required_roles` | Roles minimos para ver el modulo. |
| `menu_entries` | Entradas de menu que el shell dibuja. |
| `widgets` | Resumenes que el shell puede colocar en tablero. |
| `capabilities` | Capacidades transversales que consume. |
| `events_published` | Eventos que emite al bus VEC. |
| `events_subscribed` | Eventos que escucha. |
| `health_route` | Comprobacion de salud del modulo. |

## Backend: interfaz comun

El shell debe depender de una interfaz comun, no de paquetes concretos como
Bolsa.

```go
type VECModuleProvider interface {
    Manifest(ctx context.Context) (VECModuleManifest, error)
    Menu(ctx context.Context, principal VECPrincipal) ([]VECMenuEntry, error)
    DashboardWidgets(ctx context.Context, principal VECPrincipal) ([]VECWidget, error)
    Routes() []VECHTTPRoute
    Health(ctx context.Context) VECModuleHealth
}
```

`VECPrincipal` lo crea el shell desde Kerberos/AD:

```go
type VECPrincipal struct {
    SubjectRef     string
    EmployeeRef    string
    DisplayName    string
    Units          []string
    Roles          []string
    Permissions    []string
    AuthMethod     string // kerberos_ad
    SessionRef     string
    CorrelationRef string
}
```

Reglas:

- `VECPrincipal` llega ya autenticado al modulo.
- El modulo puede autorizar por permisos, unidad, rol o regla de negocio, pero
  no puede pedir credenciales.
- Las reglas de dominio viven en casos de uso del modulo.
- Los handlers HTTP del modulo solo adaptan transporte, DTOs e i18n.
- Los permisos se expresan como claves: `bolsa.solicitud.read`,
  `nomina.recibo.download`, `permisos.solicitud.submit`, etc.

## Frontend: interfaz comun

El shell renderiza estructura comun. El modulo registra vistas y acciones.

```ts
type VECFrontendModule = {
  moduleRef: string;
  mount(base: VECMountContext): void;
  unmount(): void;
  routes: VECRoute[];
  widgets: VECWidgetDescriptor[];
};

type VECMountContext = {
  principal: VECPrincipalView;
  api: VECAPIClient;
  i18n: VECI18n;
  notify: VECNotificationClient;
  audit: VECAuditClient;
  documents: VECDocumentClient;
  navigate: (route: string) => void;
};
```

Reglas:

- El modulo no dibuja sidebar global ni topbar global.
- El modulo no guarda tokens ni implementa login.
- El modulo no calcula baremos, nominas, permisos ni resoluciones legales en
  JavaScript; pide estados y resultados al backend.
- El modulo puede tener componentes propios, pero usa controles base del shell
  para tabla, filtros, detalle, timeline, documentos, notificaciones y recibos.
- Todo texto visible usa i18n por claves. No hay literales de producto en
  componentes reutilizables.

## Eventos estandar

Todos los modulos publican eventos con envelope comun.

```json
{
  "event_ref": "evt-...",
  "event_type": "bolsa.autobaremo_calculado",
  "module_ref": "vec.module.bolsa",
  "subject_ref": "employee-or-candidate-ref",
  "actor_ref": "employee-ref",
  "occurred_at": "2026-06-19T12:00:00Z",
  "correlation_ref": "corr-...",
  "payload_ref": "opaque-payload-ref",
  "audit_level": "administrative"
}
```

Eventos transversales recomendados:

| Evento | Productor | Consumidores |
| --- | --- | --- |
| `documento.aportado` | Cualquier modulo | Expediente, auditoria, notificaciones |
| `documento.validado` | Documentos | Modulos con evidencias |
| `notificacion.emitida` | Cualquier modulo | Buzon, auditoria |
| `tramite.borrador_creado` | Cualquier modulo | Tablero, auditoria |
| `tramite.presentado` | Cualquier modulo | Expediente, registro, notificaciones |
| `alegacion.presentada` | Cualquier modulo | Unidad revisora, auditoria |
| `resolucion.publicada` | Cualquier modulo | Notificaciones, expediente |

## Capacidades transversales del shell

Estas capacidades no deben duplicarse por modulo:

| Capacidad | Propietario | Uso por modulos |
| --- | --- | --- |
| Identidad Kerberos/AD | `vec-shell` | Reciben `VECPrincipal`. |
| Sesion | `vec-shell` | No hay login por modulo. |
| Menu | `vec-shell` | Consume `menu_entries`. |
| Permisos | `vec-core`/shell | Evalua visibilidad; modulo valida accion. |
| i18n | `vec-core` | Catalogo por modulo, claves sin texto hardcodeado. |
| Notificaciones | `vec-core` | Buzon unico y eventos por modulo. |
| Documentos/expediente | `vec-core` | Adjuntos, CSV, ENI, firma, versionado. |
| Auditoria | `vec-core` | Timeline unico por sujeto/tramite/modulo. |
| Busqueda global | `vec-shell` | Indices aportados por modulos. |
| Salud | `vec-shell` | Consulta `health_route` de cada modulo. |

## Mapa de modulos inicial

Basado en el mapa del sitio y las paginas oficiales revisadas, estas opciones
de menu pueden modelarse como modulos o familias de modulos.

| Categoria VEC | Modulo candidato | Opciones fuente |
| --- | --- | --- |
| Seleccion y bolsas | `vec.module.procesos_selectivos` | Acceso funcionarios, acceso laborales, promocion interna, modelos, consulta plazas. |
| Seleccion y bolsas | `vec.module.bolsa` | Bolsa de trabajo, bolsa unica comun, interinos, laborales, vista expediente, solicitud, alegaciones. |
| Provision | `vec.module.cap` | Concurso abierto y permanente funcionario/laboral, participacion, consulta puestos, alegaciones, desistimiento, opcion, vista expediente. |
| Provision | `vec.module.concursos` | Concursos de meritos, concurso funcionarios, concurso laborales, PLD, SNL, permutas. |
| Tramites laborales | `vec.module.accion_social` | Ayudas, anticipos reintegrables, alegaciones, ayudas discapacidad. |
| Tramites laborales | `vec.module.derechos_deberes` | Compatibilidad, horarios, salud laboral, vacaciones, permisos y licencias. |
| Puesto y carrera | `vec.module.puesto_trabajo` | RPT, caracteristicas de puesto, justicia, plantillas. |
| Puesto y carrera | `vec.module.carrera_horizontal` | Evaluacion del desempeno, carrera horizontal, consolidacion de grado. |
| Retribuciones | `vec.module.nominas` | Nomina, certificados de retenciones IRPF, certificados personales. |
| Formacion | `vec.module.formacion` | Catalogo formacion, SAFO, acciones formativas. |
| Tramitacion electronica | `vec.module.tramites` | Consulta de tramites, presentaciones, desistimientos, peticion destino. |
| Datos personales | `vec.module.mis_datos` | Historial administrativo, hoja acreditacion, vida administrativa, curriculum, domicilio, IRPF, haberes. |
| Servicios | `vec.module.servicios_empleado` | Calendarios oficiales, direcciones, junta de personal, informacion DGRRHHFP. |
| Seguridad documental | `vec.module.verificacion` | Verificador de firma, formatos admisibles, codigos de organos, registros. |

## Bolsa como primer modulo

Bolsa debe engancharse al VEC asi:

```text
module_ref: vec.module.bolsa
category_ref: seleccion_y_bolsas
base_route: /bolsa
api_prefix: /api/modules/bolsa
uses:
  - identidad Kerberos/AD del shell
  - expediente/documentos comun
  - notificaciones comun
  - auditoria comun
  - i18n comun + catalogo propio
owns:
  - convocatoria bolsa
  - solicitud bolsa
  - meritos/RUM aplicados a bolsa
  - autobaremacion
  - listados provisional/definitivo
  - alegaciones de bolsa
```

Bolsa no debe ser especial. Si un segundo modulo, por ejemplo `nominas`, no
puede enganchar usando el mismo manifiesto y las mismas interfaces, el contrato
esta mal disenado.

## Ciclo de vida de instalacion

1. El modulo registra su `VECModuleManifest`.
2. El shell valida version, rutas, permisos y colisiones.
3. El shell monta entradas de menu segun permisos del usuario.
4. El shell llama `DashboardWidgets` para el tablero.
5. Al navegar, el shell monta la vista del modulo en la zona de contenido.
6. Las acciones del modulo llaman su API bajo `api_prefix`.
7. El modulo publica eventos administrativos.
8. Shell/core actualizan auditoria, expediente y notificaciones.
9. `health_route` permite saber si el modulo esta operativo.

## Gates para aceptar un modulo

Un modulo VEC solo se acepta si cumple:

- `gate-manifest`: manifiesto valido, versionado y sin colision de rutas.
- `gate-sso`: no tiene login propio; usa `VECPrincipal`.
- `gate-hexagonal`: dominio y casos de uso no importan shell, HTTP, DB,
  Orquesta, rutas locales ni proveedor externo.
- `gate-i18n`: textos visibles con claves del catalogo.
- `gate-permissions`: permisos declarados en manifiesto y validados en backend.
- `gate-documents`: documentos por puerto comun, no filesystem interno.
- `gate-notifications`: avisos por bus/notificacion comun, no canal propio.
- `gate-audit`: acciones administrativas emiten evento auditable.
- `gate-health`: expone salud del modulo.
- `gate-tests`: pruebas focales del modulo y prueba de integracion con shell.

## Preguntas para discutir con otros agentes

1. El manifiesto debe ser JSON estatico, Go struct registrado en bootstrap o
   ambos?
2. El shell debe montar frontend por imports locales, web components o bundle
   por modulo?
3. Conviene un bus de eventos en memoria primero y adaptador persistente
   despues?
4. `Documentos`, `Notificaciones` y `Auditoria` son modulos visibles propios o
   capacidades internas del shell con pantallas globales?
5. Bolsa debe exponer RUM/meritos como parte propia o consumir un modulo comun
   `vec.module.rum`?
6. Como versionar permisos para que un modulo nuevo no rompa perfiles
   existentes?
7. Que prueba minima debe demostrar que un modulo se puede instalar, ver en
   menu, ejecutar una accion, emitir evento y auditarse?

## Propuesta de primera entrega

Antes de programar mas funcionalidad VEC, crear una vertical minima:

```text
vec-core/module_manifest.go
vec-core/principal.go
vec-core/menu.go
vec-core/events.go
vec-shell/module_registry.go
vec-module-bolsa/manifest.go
tests:
  - registra Bolsa
  - menu aparece con permisos correctos
  - usuario Kerberos salta de Dashboard a Bolsa sin nuevo login
  - accion Bolsa emite evento auditable
  - modulo sin permiso no aparece
```

Cuando esto este verde, cada opcion del menu real puede convertirse en modulo
sin volver a debatir como se engancha al VEC principal.
