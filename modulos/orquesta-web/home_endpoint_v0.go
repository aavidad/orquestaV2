package orquestaweb

import "net/http"

const WebHomePageEndpointV0 = "/"

type HomeWebEndpointV0 struct{}

func NewHomeWebEndpointV0() HomeWebEndpointV0 {
	return HomeWebEndpointV0{}
}

func (endpoint HomeWebEndpointV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != WebHomePageEndpointV0 {
		http.NotFound(w, r)
		return
	}
	if handleWebPublicHTTPOptionsV0(w, r, http.MethodGet) {
		return
	}
	if r.Method != http.MethodGet {
		setWebPublicHTTPAllowV0(w, http.MethodGet)
		http.Error(w, "metodo no permitido", http.StatusMethodNotAllowed)
		return
	}
	writeWebHTMLStringResponseV0(w, http.StatusOK, homeHTMLV0(), "es")
}

func homeHTMLV0() string {
	return `<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Orquesta</title>
  <style>
    :root {
      color-scheme: dark;
      --bg: #101512;
      --panel: #17211d;
      --panel-2: #20342c;
      --line: #355247;
      --text: #f2f7f3;
      --muted: #abc1b5;
      --accent: #9fe870;
      --accent-2: #5eead4;
      --warn: #fdb022;
      --ink: #0a0f0c;
    }
    * { box-sizing: border-box; }
    html, body { margin: 0; min-height: 100%; background:
      radial-gradient(circle at 15% 10%, rgba(159,232,112,.18), transparent 32rem),
      radial-gradient(circle at 85% 4%, rgba(94,234,212,.16), transparent 28rem),
      linear-gradient(140deg, #101512 0%, #121a17 50%, #0c1110 100%);
      color: var(--text); font: 15px/1.55 ui-sans-serif, system-ui, sans-serif; }
    body { padding: 28px; }
    .shell { max-width: 1280px; margin: 0 auto; }
    header { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: 18px; align-items: end; margin-bottom: 22px; }
    h1 { margin: 0; font-size: clamp(34px, 5vw, 68px); line-height: .95; letter-spacing: -.05em; }
    .lead { max-width: 760px; margin: 14px 0 0; color: var(--muted); font-size: 18px; }
    .badge { border: 1px solid rgba(159,232,112,.45); background: rgba(159,232,112,.12); color: var(--accent); border-radius: 999px; padding: 7px 11px; white-space: nowrap; }
    .grid { display: grid; gap: 14px; }
    .primary { grid-template-columns: minmax(0, 1.2fr) minmax(320px, .8fr); margin-bottom: 14px; }
    .cards { grid-template-columns: repeat(3, minmax(0, 1fr)); }
    .panel, .card { border: 1px solid var(--line); background: rgba(23,33,29,.86); border-radius: 18px; box-shadow: 0 20px 60px rgba(0,0,0,.22); }
    .panel { padding: 22px; }
    .card { display: grid; gap: 10px; padding: 18px; min-height: 190px; text-decoration: none; color: inherit; }
    .card:hover { border-color: var(--accent); transform: translateY(-1px); }
    h2, h3 { margin: 0; letter-spacing: -.02em; }
    h2 { font-size: 22px; }
    h3 { font-size: 18px; }
    p { margin: 0; }
    .muted { color: var(--muted); }
    .actions { display: flex; gap: 10px; flex-wrap: wrap; margin-top: 18px; }
    .button { display: inline-flex; align-items: center; justify-content: center; min-height: 42px; padding: 9px 13px; border-radius: 11px; border: 1px solid var(--line); background: var(--panel-2); color: var(--text); text-decoration: none; font-weight: 760; }
    .button.primary { background: var(--accent); color: var(--ink); border-color: var(--accent); }
    .button:hover { border-color: var(--accent-2); }
    .tag { width: max-content; border-radius: 999px; background: rgba(94,234,212,.12); color: var(--accent-2); padding: 3px 9px; font-size: 12px; font-weight: 760; }
    .tag.warn { background: rgba(253,176,34,.12); color: var(--warn); }
    ul { margin: 12px 0 0; padding-left: 18px; color: var(--muted); }
    li { margin: 5px 0; }
    .route { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; color: var(--accent-2); }
    @media (max-width: 900px) {
      body { padding: 16px; }
      header, .primary, .cards { grid-template-columns: 1fr; }
      .badge { justify-self: start; }
    }
  </style>
</head>
<body>
  <div class="shell">
    <header>
      <div>
        <h1>Orquesta</h1>
        <p class="lead">Consola operativa para pedir apps, observar runs, lanzar ticks supervisados y controlar cola sin entrar por API ni CLI.</p>
      </div>
      <span class="badge">web de operador</span>
    </header>

    <section class="grid primary" aria-label="acciones principales">
      <div class="panel">
        <h2>Trabajo normal</h2>
        <p class="muted">Empieza aqui: pide una app o cambio, deja que el Director prepare el run y sigue el progreso desde el cockpit.</p>
        <div class="actions">
          <a class="button primary" href="/nueva-app">Pedir app</a>
          <a class="button" href="/ops">Abrir cockpit</a>
          <a class="button" href="/autoprogramming">Autoprogramar</a>
          <a class="button" href="/app-change">Pedir cambio</a>
        </div>
      </div>
      <div class="panel">
        <h2>Estado del sistema</h2>
        <ul>
          <li><span class="route">/ops</span>: vision live, cola, agentes, fases y acciones de supervision.</li>
          <li><span class="route">/run-queue</span>: prioridad y cola multiapp.</li>
          <li><span class="route">/run-control</span>: pausar, reanudar, parar o cancelar runs.</li>
        </ul>
      </div>
    </section>

    <section class="grid cards" aria-label="secciones web">
      <a class="card" href="/ops">
        <span class="tag">operacion</span>
        <h3>Cockpit autonomo</h3>
        <p class="muted">Panel unico para ver actividad, detectar atencion real y pedir una pasada acotada del supervisor.</p>
      </a>
      <a class="card" href="/nueva-app">
        <span class="tag">creacion</span>
        <h3>Nueva app</h3>
        <p class="muted">Formulario web hacia contratos publicos de spec y arranque de Director.</p>
      </a>
      <a class="card" href="/app-change">
        <span class="tag">externas</span>
        <h3>Cambios y dominio</h3>
        <p class="muted">Solicita cambios o trabajos externos mediante refs opacas, sin acoplar Orquesta a la app destino.</p>
      </a>
      <a class="card" href="/autoprogramming">
        <span class="tag">autonomia</span>
        <h3>Autoprogramacion</h3>
        <p class="muted">Prepara runs programables, consulta estado y pide supervision desde la web.</p>
      </a>
      <a class="card" href="/director-stats">
        <span class="tag">diagnostico</span>
        <h3>Stats de run</h3>
        <p class="muted">Detalle de progreso, agentes, tareas, validacion y evidencias saneadas.</p>
      </a>
      <a class="card" href="/run-queue">
        <span class="tag">cola</span>
        <h3>Prioridad</h3>
        <p class="muted">Reordena trabajo sin leer stores ni tocar runtime desde la web.</p>
      </a>
      <a class="card" href="/run-control">
        <span class="tag warn">seguro</span>
        <h3>Control</h3>
        <p class="muted">Pausa, reanuda, para o cancela runs por el contrato publico de control.</p>
      </a>
    </section>
  </div>
</body>
</html>
`
}
