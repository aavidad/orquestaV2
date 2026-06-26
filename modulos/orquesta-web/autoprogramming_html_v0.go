package orquestaweb

func autoprogrammingHTMLV0() string {
	return `<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Orquesta Autoprogramacion</title>
  <style>
    :root { color-scheme: dark; --bg:#101316; --panel:#181d22; --line:#34404a; --text:#eef4f8; --muted:#aab7c3; --accent:#7cd4fd; --good:#32d583; --warn:#fdb022; --bad:#f97066; }
    * { box-sizing: border-box; }
    body { margin:0; min-height:100%; padding:22px; background: radial-gradient(circle at top left, rgba(124,212,253,.17), transparent 32rem), var(--bg); color:var(--text); font:14px/1.5 ui-sans-serif, system-ui, sans-serif; }
    .shell { max-width:1260px; margin:0 auto; display:grid; gap:16px; }
    header { display:flex; justify-content:space-between; gap:14px; align-items:flex-start; }
    h1 { margin:0; font-size:clamp(28px,4vw,52px); letter-spacing:-.04em; }
    h2 { margin:0 0 12px; font-size:18px; }
    a { color:var(--accent); }
    .nav { display:flex; gap:8px; flex-wrap:wrap; }
    .nav a, button { border:1px solid var(--line); background:#202832; color:var(--text); border-radius:8px; padding:8px 10px; text-decoration:none; cursor:pointer; }
    button.primary { background:var(--accent); color:#071015; border-color:var(--accent); font-weight:800; }
    .grid { display:grid; grid-template-columns:minmax(0,1fr) minmax(340px,.75fr); gap:16px; align-items:start; }
    .panel { border:1px solid var(--line); background:rgba(24,29,34,.92); border-radius:14px; padding:16px; }
    form { display:grid; gap:12px; }
    label { display:grid; gap:5px; color:var(--muted); }
    input, textarea, select { width:100%; border:1px solid var(--line); background:#0e1216; color:var(--text); border-radius:8px; padding:9px; font:inherit; }
    textarea { min-height:78px; resize:vertical; }
    .row { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:10px; }
    .hint { color:var(--muted); }
    .result { display:grid; gap:10px; }
    .toggle { display:flex; align-items:center; gap:8px; width:max-content; max-width:100%; color:var(--text); }
    .toggle input { width:auto; }
    .pill { display:inline-flex; width:max-content; border-radius:999px; padding:3px 9px; background:#2a3540; color:var(--muted); }
    .pill.ok { background:rgba(50,213,131,.14); color:var(--good); }
    .pill.warn { background:rgba(253,176,34,.14); color:var(--warn); }
    .pill.bad { background:rgba(249,112,102,.14); color:var(--bad); }
    .status-grid { display:grid; grid-template-columns:max-content minmax(0,1fr); gap:6px 10px; margin:0; }
    .status-grid dt { color:var(--muted); }
    .status-grid dd { margin:0; min-width:0; word-break:break-word; }
    .action-list { display:grid; gap:8px; }
    .action-item { display:grid; gap:4px; padding:9px; border:1px solid #3d4b56; border-left:4px solid var(--warn); border-radius:8px; background:#111820; }
    .action-item.blocked { border-left-color:var(--bad); }
    .action-item.info { border-left-color:var(--accent); }
    .action-head { display:flex; justify-content:space-between; gap:10px; align-items:center; }
    .action-meta { display:flex; gap:6px; flex-wrap:wrap; color:var(--muted); font-size:12px; }
    pre { margin:0; max-height:420px; overflow:auto; padding:12px; border-radius:10px; background:#090d11; border:1px solid #24303a; white-space:pre-wrap; word-break:break-word; }
    .actions { display:flex; gap:8px; flex-wrap:wrap; }
    button:disabled { opacity:.55; cursor:not-allowed; }
    @media (max-width: 860px) { body{padding:14px} header,.grid,.row{grid-template-columns:1fr; display:grid} .nav{justify-content:flex-start} }
  </style>
</head>
<body>
  <main class="shell">
    <header>
      <div>
        <h1>Autoprogramacion</h1>
        <p class="hint">Prepara trabajo programable, consulta estado y avanza una run sin salir de la web.</p>
      </div>
      <nav class="nav" aria-label="navegacion">
        <a href="/">Inicio</a><a href="/ops">Ops</a><a href="/run-queue">Cola</a><a href="/run-control">Control</a>
      </nav>
    </header>
    <section class="grid">
      <div class="panel">
        <h2>Preparar run</h2>
        <form id="prepare-form">
          <div class="row">
            <label>Proyecto <input name="project_ref" value="project-ref-orquesta-web" required></label>
            <label>Worktree ref <input name="worktree_ref" value="worktree-ref-orquesta-web" required></label>
          </div>
          <div class="row">
            <label>Branch ref <input name="branch_ref" value="branch-ref-autoprogramming-web" required></label>
            <label>Prioridad <input name="priority_score" type="number" value="80" min="0" max="100"></label>
          </div>
          <div class="row">
            <label>Task ref <input name="task_ref" value="task-ref-web-autoprogramming-001" required></label>
            <label>Area <input name="area" value="web" required></label>
          </div>
          <label>Titulo <input name="title" value="Mejorar superficie web de operador"></label>
          <label>Objetivo <textarea name="objective">Implementar un corte pequeno, probado y usable desde la web de Orquesta.</textarea></label>
          <label>Write-set <textarea name="write_set">modulos/orquesta-web</textarea></label>
          <label>Tests requeridos <textarea name="required_tests">go test -count=1 ./modulos/orquesta-web</textarea></label>
          <label>Criterios de aceptacion <textarea name="acceptance">La web conserva refs opacas
No accede a stores ni runtime
El cambio queda cubierto por tests</textarea></label>
          <label class="toggle" title="Usa Codex Goal como loop interno y deja Orquesta como preparador, observador y validador.">
            <input name="goal_first" type="checkbox" checked>
            Codex Goal automatico
          </label>
          <div class="actions">
            <button class="primary" type="submit">Preparar run</button>
            <button type="button" id="refresh-status">Consultar estado</button>
            <button type="button" id="supervise-run">Supervisar legacy</button>
          </div>
        </form>
      </div>
      <aside class="panel result" aria-live="polite">
        <h2>Resultado</h2>
        <span id="state-pill" class="pill warn">sin accion</span>
        <div class="hint">Run actual: <code id="run-ref">-</code></div>
        <div class="actions">
          <a id="stats-link" href="/director-stats">Abrir stats</a>
          <a href="/ops">Ver en cockpit</a>
        </div>
        <section id="goal-panel" class="action-list" hidden>
          <h2>Goal</h2>
          <dl class="status-grid">
            <dt>Goal ref</dt><dd><code id="goal-ref">-</code></dd>
            <dt>Externo</dt><dd><code id="external-goal-ref">-</code></dd>
            <dt>Goal</dt><dd><span id="goal-status" class="pill warn">pendiente</span></dd>
            <dt>Cierre</dt><dd><span id="closure-status" class="pill warn">sin validar</span></dd>
          </dl>
          <div class="actions">
            <button type="button" id="observe-goal">Observar goal</button>
            <button type="button" id="stop-goal-poll">Parar seguimiento</button>
          </div>
          <div id="goal-note" class="hint">Sin goal lanzado.</div>
        </section>
        <section id="stale-running-panel" class="action-list" hidden>
          <h2>Acciones requeridas</h2>
          <div id="stale-running-list" class="action-list"></div>
        </section>
        <pre id="output">{}</pre>
      </aside>
    </section>
  </main>
  <script>
    const output = document.getElementById('output');
    const runRefNode = document.getElementById('run-ref');
    const pill = document.getElementById('state-pill');
    const statsLink = document.getElementById('stats-link');
    const stalePanel = document.getElementById('stale-running-panel');
    const staleList = document.getElementById('stale-running-list');
    const superviseRunButton = document.getElementById('supervise-run');
    const goalPanel = document.getElementById('goal-panel');
    const goalRefNode = document.getElementById('goal-ref');
    const externalGoalRefNode = document.getElementById('external-goal-ref');
    const goalStatusNode = document.getElementById('goal-status');
    const closureStatusNode = document.getElementById('closure-status');
    const goalNote = document.getElementById('goal-note');
    let currentRunRef = '';
    let currentGoalRef = '';
    let currentExternalGoalRef = '';
    let goalPollTimer = 0;
    let goalPollCount = 0;

    function lines(value) {
      return String(value || '').split(/\n|,/).map(v => v.trim()).filter(Boolean);
    }
    function requestId(prefix) {
      return prefix + '-' + Date.now().toString(36);
    }
    function setResult(kind, value) {
      output.textContent = JSON.stringify(value || {}, null, 2);
      const estado = String((value || {}).estado || kind || 'unknown');
      pill.textContent = estado;
      pill.className = 'pill ' + (estado === 'ok' ? 'ok' : (estado === 'error' ? 'bad' : 'warn'));
      const runRef = resultRunRef(value);
      if (runRef) {
        currentRunRef = runRef;
        runRefNode.textContent = currentRunRef;
        statsLink.href = '/director-stats?include_agent_progress=true&run_ref=' + encodeURIComponent(currentRunRef);
      }
      renderGoal(value || {});
      renderStaleRunning((value || {}).stale_running || []);
      updateSuperviseState();
    }
    function resultRunRef(value) {
      const goal = (value || {}).goal || {};
      const spec = (((value || {}).goal_specs || [])[0]) || {};
      return String(goal.run_ref || (value || {}).run_ref || spec.run_ref || '').trim();
    }
    function goalInfo(value) {
      const goal = (value || {}).goal || {};
      const spec = (((value || {}).goal_specs || [])[0]) || {};
      return {
        run_ref: resultRunRef(value),
        goal_ref: String(goal.goal_ref || (value || {}).goal_ref || spec.goal_ref || '').trim(),
        external_goal_ref: String(goal.external_goal_ref || (value || {}).external_goal_ref || '').trim(),
        goal_status: String(goal.goal_status || (value || {}).goal_status || spec.status || '').trim(),
        closure_status: String((value || {}).closure_status || '').trim(),
        closure_accepted: Boolean((value || {}).closure_accepted),
        closure_needs_rework: Boolean((value || {}).closure_needs_rework)
      };
    }
    function renderGoal(value) {
      const info = goalInfo(value);
      const hasGoal = Boolean(info.goal_ref || ((value || {}).goal_specs || []).length);
      goalPanel.hidden = !hasGoal;
      if (!hasGoal) return;
      currentGoalRef = info.goal_ref || currentGoalRef;
      currentExternalGoalRef = info.external_goal_ref || currentExternalGoalRef;
      goalRefNode.textContent = currentGoalRef || '-';
      externalGoalRefNode.textContent = currentExternalGoalRef || '-';
      setStatusPill(goalStatusNode, info.goal_status || 'preparado');
      setStatusPill(closureStatusNode, info.closure_status || (info.closure_accepted ? 'aceptado' : 'pendiente'));
      if (info.closure_accepted) {
        goalNote.textContent = 'Cierre aceptado por evidencias.';
      } else if (info.closure_needs_rework) {
        goalNote.textContent = 'Cierre bloqueado: requiere rework.';
      } else if (!info.run_ref) {
        goalNote.textContent = 'Goal spec preparado; falta backend Goal opt-in para lanzarlo.';
      } else {
        goalNote.textContent = 'Seguimiento goal-first activo por run_ref.';
      }
    }
    function setStatusPill(node, status) {
      const normalized = String(status || 'pendiente').trim();
      const low = normalized.toLowerCase();
      node.textContent = normalized;
      node.className = 'pill ' + (
        low === 'complete' || low === 'completed' || low === 'accepted' || low === 'aceptado' || low === 'cerrada' || low === 'closed'
          ? 'ok'
          : (low === 'blocked' || low === 'failed' || low === 'invalid' || low === 'rejected' ? 'bad' : 'warn')
      );
    }
    function updateSuperviseState() {
      const goalActive = Boolean(currentGoalRef);
      superviseRunButton.disabled = goalActive;
      superviseRunButton.title = goalActive
        ? 'Esta run usa Goal; usa Observar goal.'
        : 'Solo para runs legacy de autoprogramacion.';
    }
    function renderStaleRunning(items) {
      staleList.textContent = '';
      stalePanel.hidden = !items || !items.length;
      (items || []).slice(0, 8).forEach(item => {
        const row = document.createElement('article');
        const severity = String(item.severity || 'warning');
        row.className = 'action-item ' + severity;
        const head = document.createElement('div');
        head.className = 'action-head';
        const code = document.createElement('strong');
        code.textContent = item.code || 'estado_accionable';
        const status = document.createElement('span');
        status.className = 'pill ' + (severity === 'blocked' ? 'bad' : (severity === 'info' ? 'ok' : 'warn'));
        status.textContent = item.status || severity;
        head.append(code, status);
        const meta = document.createElement('div');
        meta.className = 'action-meta';
        [item.run_ref, item.app_ref, item.reason].filter(Boolean).forEach(value => {
          const span = document.createElement('span');
          span.textContent = value;
          meta.appendChild(span);
        });
        const action = document.createElement('div');
        action.className = 'hint';
        action.textContent = item.recommended_action || 'inspect_run_ref';
        row.append(head, meta, action);
        staleList.appendChild(row);
      });
    }
    async function postJSON(url, payload) {
      const response = await fetch(url, {
        method: 'POST',
        headers: {'Content-Type': 'application/json', 'Accept': 'application/json'},
        body: JSON.stringify(payload)
      });
      const data = await response.json().catch(() => ({estado:'error', errores_publicos:[{code:'respuesta_no_json'}]}));
      if (!response.ok && !data.estado) data.estado = 'error';
      return data;
    }
    function goalContextRefs(enabled) {
      if (!enabled) return [];
      return [
        'goal_migration:goal-first',
        'goal_capability:starter',
        'goal_capability:observer',
        'goal_capability:closure-validator'
      ];
    }
    function preparePayload(form) {
      const id = requestId('web-autoprogramming');
      const taskContextRefs = goalContextRefs(form.goal_first && form.goal_first.checked);
      return {
        request_id: id,
        correlation_id: id,
        requested_by: 'orquesta-web',
        priority_score: Number(form.priority_score.value || 0),
        max_bursts: 4,
        max_steps_per_burst: 8,
        max_dispatches_per_wait: 8,
        autoprogramming_request: {
          request_ref: id,
          project_ref: form.project_ref.value.trim(),
          worktree_ref: form.worktree_ref.value.trim(),
          worktree_isolated: true,
          branch_ref: form.branch_ref.value.trim(),
          write_set: lines(form.write_set.value),
          required_tests: lines(form.required_tests.value),
          tasks: [{
            task_ref: form.task_ref.value.trim(),
            area: form.area.value.trim(),
            title: form.title.value.trim(),
            objective: form.objective.value.trim(),
            acceptance_criteria: lines(form.acceptance.value),
            required_tests: lines(form.required_tests.value),
            context_refs: taskContextRefs
          }]
        }
      };
    }
    async function refreshStatus() {
      const id = requestId('web-autoprogramming-status');
      const data = await postJSON('/api/v0/autoprogramming/status', {
        request_id: id,
        correlation_id: id,
        run_ref: currentRunRef,
        queue_ref: 'global',
        queue_limit: 20,
        include_agent_progress: true,
        include_agent_usage: true
      });
      setResult('status', data);
    }
    async function observeGoal(startPolling) {
      if (!currentRunRef) {
        goalPanel.hidden = false;
        goalNote.textContent = 'No hay run_ref de Goal para observar.';
        return;
      }
      const id = requestId('web-autoprogramming-goal');
      const data = await postJSON('/api/v0/autoprogramming/goal/observe', {
        request_id: id,
        correlation_id: id,
        run_ref: currentRunRef,
        requested_by: 'orquesta-web'
      });
      setResult('goal', data);
      if (goalTerminal(data)) {
        stopGoalPolling();
        return;
      }
      if (startPolling) startGoalPolling();
    }
    function startGoalPolling() {
      stopGoalPolling();
      goalPollCount = 0;
      goalPollTimer = window.setInterval(async () => {
        goalPollCount += 1;
        if (goalPollCount > 60 || !currentRunRef || !currentGoalRef) {
          stopGoalPolling();
          return;
        }
        await observeGoal(false);
      }, 5000);
    }
    function stopGoalPolling() {
      if (goalPollTimer) window.clearInterval(goalPollTimer);
      goalPollTimer = 0;
    }
    function goalTerminal(value) {
      const info = goalInfo(value || {});
      const goal = String(info.goal_status || '').toLowerCase();
      const closure = String(info.closure_status || '').toLowerCase();
      const run = String((value || {}).run_status || '').toLowerCase();
      return info.closure_accepted ||
        ['complete','completed','blocked','failed','invalid','canceled','cancelled'].includes(goal) ||
        ['accepted','blocked','failed','rejected'].includes(closure) ||
        ['closed','cerrada','blocked','failed'].includes(run);
    }
    document.getElementById('prepare-form').addEventListener('submit', async function(event) {
      event.preventDefault();
      const data = await postJSON('/api/v0/autoprogramming/prepare-run', preparePayload(event.target));
      setResult('prepare', data);
      if (currentRunRef && currentGoalRef) await observeGoal(true);
    });
    document.getElementById('refresh-status').addEventListener('click', refreshStatus);
    document.getElementById('observe-goal').addEventListener('click', function() {
      observeGoal(true);
    });
    document.getElementById('stop-goal-poll').addEventListener('click', stopGoalPolling);
    document.getElementById('supervise-run').addEventListener('click', async function() {
      if (currentGoalRef) return;
      const id = requestId('web-supervise');
      const data = await postJSON('/api/v0/autoprogramming/supervise', {
        request_id: id,
        correlation_id: id,
        run_ref: currentRunRef,
        queue_ref: currentRunRef ? '' : 'global',
        max_ticks: 20,
        max_executions: 20,
        allow_repeated_runs: true
      });
      setResult('supervise', data);
    });
  </script>
</body>
</html>
`
}
