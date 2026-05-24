package orquestaweb

import (
	"net/http"
	"strings"
)

const WebOpsDashboardPageEndpointV0 = "/ops"

type OpsDashboardWebEndpointV0 struct{}

func NewOpsDashboardWebEndpointV0() OpsDashboardWebEndpointV0 {
	return OpsDashboardWebEndpointV0{}
}

func (endpoint OpsDashboardWebEndpointV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "metodo no permitido", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(opsDashboardHTMLV0()))
}

func opsDashboardHTMLV0() string {
	return strings.TrimSpace(`<!doctype html>
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
    .grid { display: grid; gap: 12px; }
    .kpis { grid-template-columns: repeat(6, minmax(150px, 1fr)); margin-bottom: 12px; }
    .layout { grid-template-columns: minmax(0, 1.4fr) minmax(360px, .7fr); align-items: start; }
    .panel { border: 1px solid var(--line); background: var(--panel); border-radius: 8px; overflow: hidden; }
    .panel h2 { margin: 0; padding: 12px 14px; font-size: 14px; border-bottom: 1px solid var(--line); background: rgba(255,255,255,.02); }
    .pad { padding: 12px 14px; }
    .kpi { min-height: 96px; padding: 12px; border: 1px solid var(--line); background: var(--panel); border-radius: 8px; }
    .kpi .label { color: var(--muted); font-size: 12px; text-transform: uppercase; }
    .kpi .value { font-size: 28px; font-weight: 750; margin-top: 8px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
    .kpi .hint { color: var(--muted); margin-top: 4px; font-size: 12px; min-height: 18px; }
    table { width: 100%; border-collapse: collapse; table-layout: fixed; }
    th, td { padding: 9px 10px; border-bottom: 1px solid var(--line); text-align: left; vertical-align: middle; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    th { color: var(--muted); font-weight: 600; font-size: 12px; background: rgba(255,255,255,.02); }
    tr:hover td { background: rgba(124,212,253,.05); }
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
    .kv { display: grid; grid-template-columns: 150px minmax(0, 1fr); gap: 8px; padding: 7px 0; border-bottom: 1px solid rgba(255,255,255,.06); }
    .kv:last-child { border-bottom: 0; }
    .kv .k { color: var(--muted); }
    .mono { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 12px; }
    .empty { color: var(--muted); padding: 18px 14px; }
    .error { color: var(--bad); white-space: pre-wrap; }
    @media (max-width: 1180px) {
      .kpis { grid-template-columns: repeat(3, minmax(150px, 1fr)); }
      .layout { grid-template-columns: 1fr; }
    }
    @media (max-width: 720px) {
      body { padding: 12px; }
      header { align-items: flex-start; flex-direction: column; }
      .kpis { grid-template-columns: 1fr 1fr; }
      th:nth-child(2), td:nth-child(2), th:nth-child(5), td:nth-child(5) { display: none; }
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

    <main class="grid layout">
      <section class="stack">
        <div class="panel">
          <h2>Proyectos / runs</h2>
          <div class="table-wrap">
            <table>
              <thead><tr><th>Proyecto</th><th>Run</th><th>Estado</th><th>Progreso</th><th>Agentes</th><th>Validación</th></tr></thead>
              <tbody id="projects-body"><tr><td colspan="6" class="empty">Sin datos todavía</td></tr></tbody>
            </table>
          </div>
        </div>
        <div class="panel">
          <h2>Agentes</h2>
          <table>
            <thead><tr><th>Agente</th><th>Run</th><th>Estado</th><th>Tarea</th><th>Progreso</th><th>Señal</th></tr></thead>
            <tbody id="agents-body"><tr><td colspan="6" class="empty">Sin agentes observados</td></tr></tbody>
          </table>
        </div>
        <div class="panel">
          <h2>Cola</h2>
          <table>
            <thead><tr><th>#</th><th>Run</th><th>Proyecto</th><th>Estado</th><th>Prioridad</th><th>Validación</th></tr></thead>
            <tbody id="queue-body"><tr><td colspan="6" class="empty">Sin cola</td></tr></tbody>
          </table>
        </div>
      </section>

      <aside class="stack">
        <div class="panel">
          <h2>Servidor</h2>
          <div id="server-detail" class="pad"></div>
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
    const maxRuns = 12;
    let timer = null;
    document.getElementById('refresh-ms').textContent = refreshMs + ' ms';
    document.getElementById('refresh-now').addEventListener('click', refreshAll);

    function byId(id) { return document.getElementById(id); }
    function text(id, value) { byId(id).textContent = value == null || value === '' ? '-' : String(value); }
    function esc(value) {
      return String(value == null ? '' : value).replace(/[&<>"']/g, function(ch) {
        return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch];
      });
    }
    function shortRef(value) {
      const raw = String(value || '');
      if (raw.length <= 42) return raw || '-';
      return raw.slice(0, 22) + '…' + raw.slice(-16);
    }
    function fmtBytes(value) {
      const n = Number(value || 0);
      if (!n) return '-';
      const units = ['B','KB','MB','GB','TB'];
      let v = n; let i = 0;
      while (v >= 1024 && i < units.length - 1) { v = v / 1024; i++; }
      return (i === 0 ? v.toFixed(0) : v.toFixed(1)) + ' ' + units[i];
    }
    function statusClass(value) {
      const v = String(value || '').toLowerCase();
      if (['ok','running','ready','completed','accepted','validada','validated'].includes(v)) return 'ok';
      if (v.includes('block') || v.includes('fail') || v.includes('error') || v.includes('stalled')) return 'bad';
      if (v.includes('pending') || v.includes('wait') || v.includes('unknown')) return 'warn';
      return 'info';
    }
    function statusPill(value) {
      const label = value || '-';
      return '<span class="status ' + statusClass(label) + '">' + esc(label) + '</span>';
    }
    function bar(percent, bad) {
      const p = Math.max(0, Math.min(100, Number(percent || 0)));
      const cls = bad ? 'bad' : (p >= 90 ? 'ok' : (p >= 40 ? 'warn' : ''));
      return '<div class="bar" title="' + p + '%"><span class="' + cls + '" style="width:' + p + '%"></span></div>';
    }
    function validationState(run) {
      const status = String(run.status || '').toLowerCase();
      const refs = (run.evidence_refs || []).join(' ').toLowerCase();
      if (status.includes('blocked') || status.includes('failed') || status.includes('error')) return 'bloqueada';
      if (refs.includes('prepare-run-enqueued') || refs.includes('run-coordinator-executed') || refs.includes('validated')) return 'validada';
      return 'pendiente';
    }
    async function fetchJSON(url, opts) {
      const res = await fetch(url, opts || {});
      if (!res.ok) throw new Error(url + ' -> HTTP ' + res.status);
      return await res.json();
    }
    async function fetchRunStats(runRef) {
      const body = {
        request_id: 'ops-' + Date.now() + '-' + Math.random().toString(16).slice(2),
        run_ref: runRef,
        include_process_refs: false,
        include_agent_progress: true,
        include_agent_usage: true
      };
      return await fetchJSON('/api/v0/director/stats', {
        method: 'POST',
        headers: {'content-type': 'application/json'},
        body: JSON.stringify(body)
      });
    }
    async function refreshAll() {
      const diagnostics = [];
      let server = {};
      let resources = {};
      let auto = {};
      try {
        const result = await Promise.allSettled([
          fetchJSON('/api/v0/server/status'),
          fetchJSON('/api/v0/server/resources'),
          fetchJSON('/api/v0/autoprogramming/status', {
            method: 'POST',
            headers: {'content-type': 'application/json'},
            body: JSON.stringify({
              request_id: 'ops-status-' + Date.now(),
              queue_ref: 'global',
              queue_limit: 25,
              include_process_refs: false,
              include_agent_progress: true,
              include_agent_usage: true
            })
          })
        ]);
        if (result[0].status === 'fulfilled') server = result[0].value; else diagnostics.push(result[0].reason.message);
        if (result[1].status === 'fulfilled') resources = result[1].value; else diagnostics.push(result[1].reason.message);
        if (result[2].status === 'fulfilled') auto = result[2].value; else diagnostics.push(result[2].reason.message);
        const queue = auto.queue || {};
        const ranked = queue.ranked || [];
        const runRefs = ranked.slice(0, maxRuns).map(function(item) { return item.run_ref; }).filter(Boolean);
        const statResults = await Promise.allSettled(runRefs.map(fetchRunStats));
        const stats = statResults.map(function(item, index) {
          if (item.status === 'fulfilled') return item.value;
          diagnostics.push(runRefs[index] + ': ' + item.reason.message);
          return null;
        }).filter(Boolean);
        render({server: server, resources: resources, auto: auto, ranked: ranked, stats: stats, diagnostics: diagnostics});
        markLive(true);
      } catch (err) {
        diagnostics.push(err.message || String(err));
        render({server: server, resources: resources, auto: auto, ranked: [], stats: [], diagnostics: diagnostics});
        markLive(false);
      }
    }
    function markLive(ok) {
      byId('live-dot').className = 'dot ' + (ok ? 'ok' : 'bad');
      text('live-text', ok ? 'live' : 'error');
      text('last-refresh', new Date().toLocaleTimeString());
    }
    function render(data) {
      const queue = data.auto.queue || {};
      const ranked = data.ranked || [];
      const activeRuns = data.auto.operator && data.auto.operator.active_runs ? data.auto.operator.active_runs : [];
      const runs = buildRuns(ranked, data.stats || []);
      const agents = buildAgents(data.stats || []);
      const projectCount = new Set(runs.map(function(run) { return run.app_ref || run.project_ref; }).filter(Boolean)).size;
      const worstDisk = (data.resources.disks || []).reduce(function(acc, disk) {
        return !acc || Number(disk.used_percent || 0) > Number(acc.used_percent || 0) ? disk : acc;
      }, null);
      text('kpi-server', data.server.status || 'unknown');
      text('kpi-server-hint', 'ticks ' + (data.server.supervisor_ticks || 0) + ' · ejecuciones ' + (data.server.supervisor_executions || 0));
      text('kpi-projects', projectCount);
      text('kpi-projects-hint', activeRuns.length + ' runs activos declarados');
      text('kpi-queue', queue.count == null ? ranked.length : queue.count);
      text('kpi-queue-hint', 'supervisor cola ' + (data.server.last_supervisor_queue_size == null ? '-' : data.server.last_supervisor_queue_size));
      text('kpi-agents', agents.filter(function(a) { return a.in_flight; }).length);
      text('kpi-agents-hint', agents.length + ' observados');
      text('kpi-memory', fmtBytes((data.resources.process || {}).rss_bytes || (data.resources.process || {}).go_alloc_bytes));
      text('kpi-memory-hint', 'goroutines ' + ((data.resources.process || {}).num_goroutine || '-'));
      text('kpi-disk', worstDisk ? ((worstDisk.used_percent || 0) + '%') : '-');
      text('kpi-disk-hint', worstDisk ? shortRef(worstDisk.path) : '-');
      renderProjects(runs);
      renderAgents(agents);
      renderQueue(ranked);
      renderServer(data.server);
      renderResources(data.resources);
      renderDiagnostics((data.auto.diagnostics || []).concat((data.resources.issues || [])).concat((data.diagnostics || []).map(function(message) { return {code: 'fetch_error', message: message}; })));
    }
    function buildRuns(ranked, statsResults) {
      const byRun = {};
      ranked.forEach(function(item) {
        byRun[item.run_ref] = Object.assign({}, item, {validation: validationState(item)});
      });
      statsResults.forEach(function(result) {
        const stats = result.stats || result.director_stats || {};
        const runRef = stats.run_ref || result.run_ref;
        if (!runRef) return;
        const counts = stats.counts || {};
        const progress = stats.progress || {};
        const closure = stats.closure || {};
        byRun[runRef] = Object.assign(byRun[runRef] || {}, {
          run_ref: runRef,
          app_ref: stats.project_ref || (byRun[runRef] || {}).app_ref,
          status: stats.status || result.estado || (byRun[runRef] || {}).status,
          current_phase: stats.current_phase,
          percent_complete: progress.percent_complete || 0,
          tasks_total: progress.tasks_total || counts.tasks_total || 0,
          tasks_closed: progress.tasks_closed || counts.tasks_closed || 0,
          agents_in_flight: counts.agents_in_flight || 0,
          agents_started: counts.agents_started || 0,
          agents_failed: counts.agents_failed || 0,
          agents_need_attention: counts.agents_need_attention || 0,
          progressing_agents: progress.progressing_agents || 0,
          stalled_agents: progress.stalled_agents || 0,
          closure_status: closure.status,
          blocked: closure.blocked,
          validation: (byRun[runRef] || {}).validation || 'pendiente'
        });
      });
      return Object.values(byRun);
    }
    function buildAgents(statsResults) {
      const out = [];
      statsResults.forEach(function(result) {
        const stats = result.stats || result.director_stats || {};
        (stats.agents || []).forEach(function(agent) {
          const progress = agent.last_progress || {};
          out.push({
            run_ref: stats.run_ref || result.run_ref,
            agent_ref: agent.agent_request_id,
            status: agent.status,
            in_flight: !!agent.in_flight,
            needs_attention: !!agent.needs_attention,
            task_ref: progress.task_ref || '',
            progress_status: progress.status || '',
            no_progress_ticks: progress.no_progress_ticks || 0,
            repeated_action_count: progress.repeated_action_count || 0,
            percent: agentPercent(agent, progress)
          });
        });
      });
      return out;
    }
    function agentPercent(agent, progress) {
      const status = String(agent.status || '').toLowerCase();
      const pstatus = String(progress.status || '').toLowerCase();
      if (status.includes('completed') || status.includes('delivered')) return 100;
      if (agent.needs_attention || pstatus.includes('stalled')) return 35;
      if (agent.in_flight && pstatus) return 65;
      if (agent.in_flight) return 45;
      return 0;
    }
    function renderProjects(runs) {
      if (!runs.length) {
        byId('projects-body').innerHTML = '<tr><td colspan="6" class="empty">Sin proyectos activos</td></tr>';
        return;
      }
      byId('projects-body').innerHTML = runs.map(function(run) {
        const percent = Number(run.percent_complete || 0);
        const bad = run.blocked || Number(run.stalled_agents || 0) > 0 || Number(run.agents_failed || 0) > 0;
        return '<tr>' +
          '<td title="' + esc(run.app_ref || '-') + '">' + esc(shortRef(run.app_ref || '-')) + '</td>' +
          '<td class="mono" title="' + esc(run.run_ref || '-') + '">' + esc(shortRef(run.run_ref || '-')) + '</td>' +
          '<td>' + statusPill(run.status || 'unknown') + '</td>' +
          '<td>' + bar(percent, bad) + '<span class="sub">' + percent + '% · ' + esc(run.current_phase || '-') + '</span></td>' +
          '<td>' + esc(String(run.agents_in_flight || 0)) + ' / ' + esc(String(run.agents_started || 0)) + '</td>' +
          '<td>' + statusPill(run.validation || 'pendiente') + '</td>' +
        '</tr>';
      }).join('');
    }
    function renderAgents(agents) {
      if (!agents.length) {
        byId('agents-body').innerHTML = '<tr><td colspan="6" class="empty">Sin agentes observados</td></tr>';
        return;
      }
      byId('agents-body').innerHTML = agents.map(function(agent) {
        return '<tr>' +
          '<td class="mono" title="' + esc(agent.agent_ref || '-') + '">' + esc(shortRef(agent.agent_ref || '-')) + '</td>' +
          '<td class="mono" title="' + esc(agent.run_ref || '-') + '">' + esc(shortRef(agent.run_ref || '-')) + '</td>' +
          '<td>' + statusPill(agent.status || 'unknown') + '</td>' +
          '<td class="mono" title="' + esc(agent.task_ref || '-') + '">' + esc(shortRef(agent.task_ref || '-')) + '</td>' +
          '<td>' + bar(agent.percent, agent.needs_attention) + '<span class="sub">' + agent.percent + '%</span></td>' +
          '<td>' + esc(agent.progress_status || '-') + ' · ticks ' + esc(String(agent.no_progress_ticks || 0)) + '</td>' +
        '</tr>';
      }).join('');
    }
    function renderQueue(ranked) {
      if (!ranked.length) {
        byId('queue-body').innerHTML = '<tr><td colspan="6" class="empty">Sin tareas en cola</td></tr>';
        return;
      }
      byId('queue-body').innerHTML = ranked.map(function(item) {
        return '<tr>' +
          '<td>' + esc(item.rank || '-') + '</td>' +
          '<td class="mono" title="' + esc(item.run_ref || '-') + '">' + esc(shortRef(item.run_ref || '-')) + '</td>' +
          '<td title="' + esc(item.app_ref || '-') + '">' + esc(shortRef(item.app_ref || '-')) + '</td>' +
          '<td>' + statusPill(item.status || 'unknown') + '</td>' +
          '<td>' + esc(item.priority_score || 0) + '</td>' +
          '<td>' + statusPill(validationState(item)) + '</td>' +
        '</tr>';
      }).join('');
    }
    function renderServer(server) {
      byId('server-detail').innerHTML = [
        kv('PID', server.pid),
        kv('Dirección', server.addr),
        kv('Heartbeat', server.last_heartbeat_at),
        kv('Supervisor', (server.last_supervisor_status || '-') + ' · tick ' + (server.supervisor_ticks || 0)),
        kv('Automejora', (server.idle_self_improvement_reason || '-') + ' · ok ' + (server.idle_self_improvement_ok || 0) + '/' + (server.idle_self_improvement_runs || 0)),
        kv('Proyecto', server.project_work_dir),
        kv('Runtime', server.runtime_work_dir)
      ].join('');
    }
    function renderResources(resources) {
      const process = resources.process || {};
      const diskRows = (resources.disks || []).map(function(disk) {
        return kv('Disco ' + (disk.used_percent || 0) + '%', shortRef(disk.path) + ' · libre ' + fmtBytes(disk.available_bytes));
      }).join('');
      byId('resources-detail').innerHTML = [
        kv('RSS', fmtBytes(process.rss_bytes)),
        kv('Go alloc', fmtBytes(process.go_alloc_bytes)),
        kv('Go heap', fmtBytes(process.go_heap_in_use_bytes)),
        kv('Goroutines', process.num_goroutine),
        diskRows || kv('Disco', 'sin datos')
      ].join('');
    }
    function renderDiagnostics(items) {
      if (!items.length) {
        byId('diagnostics').innerHTML = '<div class="sub">Sin incidencias publicadas.</div>';
        return;
      }
      byId('diagnostics').innerHTML = items.slice(0, 12).map(function(item) {
        if (typeof item === 'string') return '<div class="error">' + esc(item) + '</div>';
        return '<div class="kv"><div class="k">' + esc(item.code || '-') + '</div><div>' + esc(item.message || item.scope || item.field || '-') + '</div></div>';
      }).join('');
    }
    function kv(k, v) {
      return '<div class="kv"><div class="k">' + esc(k) + '</div><div class="mono" title="' + esc(v || '-') + '">' + esc(v == null || v === '' ? '-' : v) + '</div></div>';
    }
    refreshAll();
    timer = setInterval(refreshAll, refreshMs);
  </script>
</body>
</html>`) + "\n"
}
