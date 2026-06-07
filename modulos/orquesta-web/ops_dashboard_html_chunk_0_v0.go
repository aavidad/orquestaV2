package orquestaweb

const opsDashboardHTMLChunk0V0 = `<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Orquesta Ops</title>
  <style>
    :root {
      color-scheme: dark;
      --bg: #111316;
      --panel: #181c20;
      --panel-2: #20262b;
      --line: #343d46;
      --text: #edf2f7;
      --muted: #a8b3bf;
      --good: #32d583;
      --warn: #fdb022;
      --bad: #f97066;
      --info: #7cd4fd;
      --ink: #0b0d10;
    }
    * { box-sizing: border-box; }
    html, body { margin: 0; min-height: 100%; background: var(--bg); color: var(--text); font: 14px/1.45 system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; letter-spacing: 0; }
    body { padding: 20px; }
    .shell { max-width: 1640px; margin: 0 auto; }
    header { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 16px; }
    h1 { margin: 0; font-size: 22px; font-weight: 700; }
    .sub { color: var(--muted); font-size: 13px; }
    .toolbar { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
    .pill { display: inline-flex; align-items: center; gap: 6px; min-height: 28px; padding: 4px 10px; border: 1px solid var(--line); background: var(--panel); border-radius: 6px; color: var(--muted); }
    .dot { width: 8px; height: 8px; border-radius: 50%; background: var(--muted); display: inline-block; }
    .dot.ok { background: var(--good); }
    .dot.warn { background: var(--warn); }
    .dot.bad { background: var(--bad); }
    button { border: 1px solid var(--line); background: var(--panel-2); color: var(--text); border-radius: 6px; padding: 6px 10px; cursor: pointer; }
    button:hover { border-color: var(--info); }
    button.small { min-width: 32px; min-height: 28px; padding: 4px 7px; }
    .grid { display: grid; gap: 12px; }
    .kpis { grid-template-columns: repeat(6, minmax(150px, 1fr)); margin-bottom: 12px; }
    .layout { grid-template-columns: minmax(0, 1.4fr) minmax(360px, .7fr); align-items: start; }
    .ops-flow { display: grid; grid-template-columns: repeat(4, minmax(170px, 1fr)); gap: 12px; margin-bottom: 12px; }
    .flow-card { min-height: 82px; border: 1px solid var(--line); background: var(--panel); border-radius: 8px; padding: 11px 12px; }
    .flow-card .label { color: var(--muted); font-size: 12px; text-transform: uppercase; }
    .flow-card .metric { margin-top: 6px; font-size: 24px; font-weight: 750; }
    .flow-card.attention { border-color: rgba(253,176,34,.7); }
    .director-callout { border: 1px solid var(--line); background: rgba(124,212,253,.06); border-radius: 8px; margin-bottom: 12px; padding: 12px 14px; display: grid; gap: 8px; }
    .director-callout.attention { border-color: rgba(253,176,34,.7); background: rgba(253,176,34,.08); }
    .director-callout .label { color: var(--muted); font-size: 12px; text-transform: uppercase; }
    .director-callout .decision { font-size: 18px; font-weight: 750; overflow-wrap: anywhere; }
    .director-callout .reason { color: var(--muted); overflow-wrap: anywhere; }
    .panel { border: 1px solid var(--line); background: var(--panel); border-radius: 8px; overflow: hidden; }
    .panel h2 { margin: 0; padding: 12px 14px; font-size: 14px; border-bottom: 1px solid var(--line); background: rgba(255,255,255,.02); }
    .pad { padding: 12px 14px; }
    .kpi { min-height: 96px; padding: 12px; border: 1px solid var(--line); background: var(--panel); border-radius: 8px; }
    .kpi .label { color: var(--muted); font-size: 12px; text-transform: uppercase; }
    .kpi .value { font-size: 28px; font-weight: 750; margin-top: 8px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
    .kpi .hint { color: var(--muted); margin-top: 4px; font-size: 12px; min-height: 18px; }
    .table-wrap { overflow: auto; max-height: 520px; }
    table { width: 100%; min-width: 980px; border-collapse: collapse; table-layout: fixed; }
    table.compact-table { min-width: 620px; }
    th, td { padding: 9px 10px; border-bottom: 1px solid var(--line); text-align: left; vertical-align: middle; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    th { color: var(--muted); font-weight: 600; font-size: 12px; background: rgba(255,255,255,.02); }
    tr:hover td { background: rgba(124,212,253,.05); }
    tr.selected td { background: rgba(124,212,253,.12); }
    tr[data-run-ref], tr[data-agent-ref] { cursor: pointer; }
    .status { display: inline-flex; align-items: center; gap: 6px; padding: 2px 8px; border-radius: 999px; font-size: 12px; color: var(--ink); background: var(--muted); }
    .status.ok { background: var(--good); }
    .status.warn { background: var(--warn); }
    .status.bad { background: var(--bad); }
    .status.info { background: var(--info); }
    .bar { position: relative; height: 8px; border-radius: 999px; overflow: hidden; background: #2a3138; }
    .bar span { display: block; height: 100%; width: 0%; background: var(--info); }
    .bar span.ok { background: var(--good); }
    .bar span.warn { background: var(--warn); }
    .bar span.bad { background: var(--bad); }
    .stack { display: grid; gap: 12px; }
    .run-flow { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 8px; margin-bottom: 10px; }
    .run-flow-step { min-height: 74px; border: 1px solid var(--line); border-radius: 6px; padding: 8px; background: rgba(255,255,255,.025); }
    .run-flow-step .label { color: var(--muted); font-size: 11px; text-transform: uppercase; }
    .run-flow-step .value { margin-top: 5px; font-weight: 700; overflow-wrap: anywhere; }
    .run-flow-step .hint { margin-top: 4px; color: var(--muted); font-size: 12px; overflow-wrap: anywhere; }
    .kv { display: grid; grid-template-columns: 150px minmax(0, 1fr); gap: 8px; padding: 7px 0; border-bottom: 1px solid rgba(255,255,255,.06); }
    .kv:last-child { border-bottom: 0; }
    .kv .k { color: var(--muted); }
    .mono { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 12px; }
    .task-title { font-weight: 650; }
    .task-subtitle { display: block; color: var(--muted); font-size: 12px; margin-top: 2px; }
    .task-id { display: inline-flex; align-items: center; justify-content: center; min-width: 42px; min-height: 20px; margin-right: 8px; padding: 1px 6px; border-radius: 5px; background: var(--info); color: var(--ink); font-size: 12px; font-weight: 750; }
    .phase-map { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 8px; padding: 10px 14px; border-bottom: 1px solid var(--line); background: rgba(255,255,255,.015); }
    .phase-node { display: grid; gap: 7px; min-height: 94px; padding: 10px; border: 1px solid var(--line); border-radius: 8px; background: var(--panel-2); }
    .phase-node.attention { border-color: rgba(249,112,102,.7); }
    .phase-node .phase-title { font-weight: 700; overflow-wrap: anywhere; }
    .phase-node .phase-meta { display: flex; align-items: center; justify-content: space-between; gap: 8px; color: var(--muted); font-size: 12px; }
    .controls { display: grid; grid-template-columns: 1fr auto; gap: 8px; margin-top: 10px; }
    .button-row { display: flex; gap: 8px; flex-wrap: wrap; margin-top: 10px; }
    .row-actions { display: flex; gap: 6px; align-items: center; flex-wrap: nowrap; }
    .config-row { display: grid; grid-template-columns: minmax(150px, 1fr) 120px; gap: 8px; align-items: center; padding: 7px 0; border-bottom: 1px solid rgba(255,255,255,.06); }
    .config-row:last-child { border-bottom: 0; }
    .config-row.changed input { border-color: var(--warn); }
    .warning-box { border: 1px solid rgba(253,176,34,.55); background: rgba(253,176,34,.08); border-radius: 6px; padding: 9px 10px; color: var(--text); margin-bottom: 10px; }
    .detail-section { margin-top: 12px; border-top: 1px solid rgba(255,255,255,.08); padding-top: 10px; }
    details { border: 1px solid rgba(255,255,255,.08); border-radius: 6px; margin-top: 8px; background: rgba(255,255,255,.02); }
    summary { cursor: pointer; padding: 8px 10px; color: var(--info); }
    pre { margin: 0; max-height: 360px; overflow: auto; padding: 10px; white-space: pre-wrap; word-break: break-word; background: #0b0d10; border-top: 1px solid rgba(255,255,255,.08); font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 12px; }
    input[type="number"], input[type="search"], select { width: 100%; min-height: 32px; border: 1px solid var(--line); border-radius: 6px; background: #0f1215; color: var(--text); padding: 6px 8px; }
    .filters { display: grid; grid-template-columns: minmax(180px, 1fr) 150px 150px; gap: 8px; padding: 10px 14px; border-bottom: 1px solid var(--line); background: rgba(255,255,255,.015); }
    .compact-table { min-width: 620px; }
    .empty { color: var(--muted); padding: 18px 14px; }
    .error { color: var(--bad); white-space: pre-wrap; }
    @media (max-width: 1180px) {
      .kpis { grid-template-columns: repeat(3, minmax(150px, 1fr)); }
      .ops-flow { grid-template-columns: repeat(2, minmax(170px, 1fr)); }
      .layout { grid-template-columns: 1fr; }
    }
    @media (max-width: 720px) {
      body { padding: 12px; }
      header { align-items: flex-start; flex-direction: column; }
      .kpis { grid-template-columns: 1fr 1fr; }
      .ops-flow { grid-template-columns: 1fr; }
      .run-flow { grid-template-columns: 1fr 1fr; }
      .filters { grid-template-columns: 1fr; }
      .table-wrap { max-height: none; overflow: visible; padding: 0 10px 10px; }
      .table-wrap table { min-width: 0; }
      .table-wrap thead { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0 0 0 0); }
      .table-wrap tbody, .table-wrap tr, .table-wrap td { display: block; width: 100%; }
      .table-wrap tr { margin-top: 10px; padding: 8px; border: 1px solid var(--line); border-radius: 8px; background: rgba(255,255,255,.02); }
      .table-wrap td { display: grid; grid-template-columns: 96px minmax(0, 1fr); gap: 8px; padding: 7px 6px; border-bottom: 1px solid rgba(255,255,255,.06); white-space: normal; }
      .table-wrap td::before { content: attr(data-label); color: var(--muted); font-size: 12px; font-weight: 650; }
      .table-wrap td:last-child { border-bottom: 0; }
      .table-wrap td.empty { display: block; }
      .table-wrap td.empty::before { content: none; }
      .row-actions { flex-wrap: wrap; }
    }
    @media (max-width: 520px) {
      .kpis { grid-template-columns: 1fr; }
      .table-wrap td { grid-template-columns: 82px minmax(0, 1fr); }
      .kv { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <div class="shell">
    <header>
      <div>
        <h1>Orquesta Ops</h1>
        <div class="sub">Panel operativo live de servidor, automejora, cola, proyectos y agentes.</div>
      </div>
      <div class="toolbar">
        <span class="pill"><span id="live-dot" class="dot warn"></span><span id="live-text">conectando</span></span>
        <span class="pill">refresco <strong id="refresh-ms">1000 ms</strong></span>
        <span class="pill">última lectura <strong id="last-refresh">-</strong></span>
        <button id="refresh-now" type="button">Actualizar</button>
      </div>
    </header>

    <section class="grid kpis" aria-label="indicadores">
      <div class="kpi"><div class="label">Servidor</div><div id="kpi-server" class="value">-</div><div id="kpi-server-hint" class="hint">-</div></div>
      <div class="kpi"><div class="label">Proyectos activos</div><div id="kpi-projects" class="value">-</div><div id="kpi-projects-hint" class="hint">-</div></div>
      <div class="kpi"><div class="label">Tareas en cola</div><div id="kpi-queue" class="value">-</div><div id="kpi-queue-hint" class="hint">-</div></div>
      <div class="kpi"><div class="label">Agentes activos</div><div id="kpi-agents" class="value">-</div><div id="kpi-agents-hint" class="hint">-</div></div>
      <div class="kpi"><div class="label">Memoria proceso</div><div id="kpi-memory" class="value">-</div><div id="kpi-memory-hint" class="hint">-</div></div>
      <div class="kpi"><div class="label">Disco peor uso</div><div id="kpi-disk" class="value">-</div><div id="kpi-disk-hint" class="hint">-</div></div>
    </section>

    <section class="ops-flow" aria-label="flujo operativo">
      <div class="flow-card"><div class="label">Cola</div><div id="flow-queue" class="metric">-</div><div id="flow-queue-hint" class="sub">-</div></div>
      <div class="flow-card"><div class="label">Ejecución</div><div id="flow-execution" class="metric">-</div><div id="flow-execution-hint" class="sub">-</div></div>
      <div id="flow-attention-card" class="flow-card"><div class="label">Atención</div><div id="flow-attention" class="metric">-</div><div id="flow-attention-hint" class="sub">-</div></div>
      <div class="flow-card"><div class="label">Cierre</div><div id="flow-closure" class="metric">-</div><div id="flow-closure-hint" class="sub">-</div></div>
    </section>

    <main class="grid layout">
      <section class="stack">
        <div id="director-callout" class="director-callout" aria-label="director autónomo">
          <div class="label">Director</div>
          <div id="director-decision" class="decision">Esperando datos</div>
          <div id="director-reason" class="reason">Aún no hay snapshot operativo.</div>
        </div>
        <div class="panel">
          <h2>Proyectos / runs</h2>
          <div class="filters" aria-label="filtros de proyectos">
            <input id="filter-runs" class="ops-filter" type="search" placeholder="Filtrar por tarea, run o proyecto">
            <select id="filter-status" class="ops-filter" aria-label="Estado"><option value="">Todos</option><option value="running">Activas</option><option value="blocked">Bloqueadas</option><option value="completed">Completadas</option></select>
            <select id="filter-validation" class="ops-filter" aria-label="Validación"><option value="">Validación</option><option value="validada">Validada</option><option value="pendiente">Pendiente</option><option value="bloqueada">Bloqueada</option></select>
          </div>
          <div class="table-wrap">
            <table>
              <thead><tr><th>Tarea</th><th>Run</th><th>Estado</th><th>Progreso</th><th>Agentes</th><th>Validación</th></tr></thead>
              <tbody id="projects-body"><tr><td colspan="6" class="empty">Sin datos todavía</td></tr></tbody>
            </table>
          </div>
        </div>
        <div class="panel">
          <h2>Fases comparadas</h2>
          <div id="phase-map" class="phase-map" aria-label="esquema de fases"><div class="empty">Sin fases observadas</div></div>
          <div class="table-wrap compact-wrap">
            <table class="compact-table">
              <thead><tr><th>Fase</th><th>Runs</th><th>Progreso medio</th><th>Atención</th><th>Tareas</th></tr></thead>
              <tbody id="phase-body"><tr><td colspan="5" class="empty">Sin fases observadas</td></tr></tbody>
            </table>
          </div>
        </div>
        <div class="panel">
          <h2>Uso comparado</h2>
          <div class="table-wrap compact-wrap">
            <table class="compact-table">
              <thead><tr><th>Tarea</th><th>Cuota</th><th>Tokens</th><th>Agentes</th><th>Atención</th></tr></thead>
              <tbody id="usage-body"><tr><td colspan="5" class="empty">Sin uso publicado por stats</td></tr></tbody>
            </table>
          </div>
        </div>
        <div class="panel">
          <h2>Tareas completas</h2>
          <div class="filters" aria-label="filtros de tareas">
            <input id="filter-tasks" class="ops-filter" type="search" placeholder="Filtrar tareas por texto o ref">
            <select id="filter-task-status" class="ops-filter" aria-label="Estado de tarea"><option value="">Todas</option><option value="open">Abiertas</option><option value="closed">Cerradas</option><option value="blocked">Bloqueadas</option></select>
            <select id="filter-task-agent" class="ops-filter" aria-label="Agente"><option value="">Con o sin agente</option><option value="with_agent">Con agente</option><option value="without_agent">Sin agente</option></select>
          </div>
          <div class="table-wrap">
            <table>
              <thead><tr><th>Tarea</th><th>Run</th><th>Estado</th><th>Agente</th><th>Señal</th></tr></thead>
              <tbody id="tasks-body"><tr><td colspan="5" class="empty">Sin tareas publicadas por stats</td></tr></tbody>
            </table>
          </div>
        </div>
        <div class="panel">
          <h2>Agentes</h2>
          <div class="filters" aria-label="filtros de agentes">
            <input id="filter-agents" class="ops-filter" type="search" placeholder="Filtrar por agente, run o tarea">
            <select id="filter-agent-status" class="ops-filter" aria-label="Estado de agente"><option value="">Todos</option><option value="running">Activos</option><option value="attention">Atención</option><option value="completed">Completados</option></select>
            <select id="filter-agent-signal" class="ops-filter" aria-label="Señal"><option value="">Señal</option><option value="progressing">Con progreso</option><option value="stalled">Sin progreso</option></select>
          </div>
          <div class="table-wrap">
            <table>
              <thead><tr><th>Agente</th><th>Run</th><th>Estado</th><th>Tarea</th><th>Progreso</th><th>Capacidad / uso</th><th>Señal</th></tr></thead>
              <tbody id="agents-body"><tr><td colspan="7" class="empty">Sin agentes observados</td></tr></tbody>
            </table>
          </div>
        </div>
        <div class="panel">
          <h2>Cola</h2>
          <div class="table-wrap">
            <table>
              <thead><tr><th>#</th><th>Tarea</th><th>Proyecto</th><th>Estado</th><th>Prioridad</th><th>Validación</th><th>Control</th></tr></thead>
              <tbody id="queue-body"><tr><td colspan="7" class="empty">Sin cola</td></tr></tbody>
            </table>
          </div>
        </div>
        <div class="panel">
          <h2>Completadas observadas</h2>
          <div class="table-wrap">
            <table>
              <thead><tr><th>Tarea</th><th>Run</th><th>Proyecto</th><th>Estado final</th><th>Progreso</th><th>Hora</th></tr></thead>
              <tbody id="completed-body"><tr><td colspan="6" class="empty">Aún no hay tareas completadas observadas por este panel</td></tr></tbody>
            </table>
          </div>
        </div>
      </section>

      <aside class="stack">
        <div class="panel">
          <h2>Detalle y control</h2>
          <div id="selected-detail" class="pad"><div class="empty">Selecciona una tarea o agente.</div></div>
        </div>
        <div class="panel">
          <h2>Servidor</h2>
          <div id="server-detail" class="pad"></div>
        </div>
        <div class="panel">
          <h2>Admin</h2>
          <div id="admin-config" class="pad"></div>
        </div>
        <div class="panel">
          <h2>Recursos</h2>
          <div id="resources-detail" class="pad"></div>
        </div>
        <div class="panel">
          <h2>Diagnóstico</h2>
          <div id="diagnostics" class="pad"></div>
        </div>
      </aside>
    </main>
  </div>

  <script>
    const refreshMs = 1000;
    const queueLimit = 200;
    const maxRuns = 80;
    let timer = null;
    let selectedRunRef = '';
    let selectedAgentRef = '';
    let controlMessage = '';
    let lastSnapshot = {runs: [], agents: [], ranked: [], tasks: []};
    let completedHistory = loadCompletedHistory();
    let runtimeDetails = {};
    let runtimeDetailLoading = {};
    let openDetailKeys = new Set();
    let agentDisplayCache = {};
    let pendingEnvEdits = loadPendingEnvEdits();
    let adminMessage = '';
    let lastServer = {};
    let stableOrders = loadStableOrders();
    let stableCounters = {
      runs: nextStableIndex(stableOrders.runs),
      agents: nextStableIndex(stableOrders.agents),
      queue: nextStableIndex(stableOrders.queue)
    };
    let stableOrdersDirty = false;
    document.getElementById('refresh-ms').textContent = refreshMs + ' ms';
    document.getElementById('refresh-now').addEventListener('click', refreshAll);
    document.querySelectorAll('.ops-filter').forEach(function(input) {
      input.addEventListener('input', renderCurrentSnapshot);
      input.addEventListener('change', renderCurrentSnapshot);
    });

    function byId(id) { return document.getElementById(id); }
    function text(id, value) { byId(id).textContent = value == null || value === '' ? '-' : String(value); }
    function esc(value) {
      return String(value == null ? '' : value).replace(/[&<>"']/g, function(ch) {
        return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch];
      });
    }
    function jsArg(value) {
      return String(value == null ? '' : value).replace(/\\/g, '\\\\').replace(/'/g, "\\'").replace(/\n/g, ' ');
    }
    function shortRef(value) {
      const raw = String(value || '');
      if (raw.length <= 42) return raw || '-';
      return raw.slice(0, 22) + '…' + raw.slice(-16);
    }
    function fmtBytes(value) {
      const n = Number(value || 0);`
