package orquestaweb

const opsDashboardHTMLChunk3V0 = `          currently_visible: true,
          last_seen_ms: nowMs,
          last_seen_at: now
        });
      });
      Object.keys(agentDisplayCache || {}).forEach(function(key) {
        if (!freshKeys.has(key)) agentDisplayCache[key].currently_visible = false;
      });
      const allowedRuns = new Set((runs || []).map(function(run) { return run.run_ref; }).filter(Boolean));
      const merged = Object.values(agentDisplayCache || {}).filter(function(agent) {
        if (!allowedRuns.has(agent.run_ref)) return false;
        if (agent.currently_visible !== false) return true;
        return Number(agent.last_seen_ms || 0) >= nowMs - graceMs;
      });
      return merged.slice(-500);
    }
    function agentPercent(agent, progress) {
      const status = String(agent.status || '').toLowerCase();
      const pstatus = String(progress.status || '').toLowerCase();
      if (status.includes('completed') || status.includes('delivered')) return 100;
      if (agentNeedsAttention(agent)) return 35;
      if (agent.in_flight && pstatus) return 65;
      if (agent.in_flight) return 45;
      return 0;
    }
    function filterValue(id) {
      const node = byId(id);
      return node ? String(node.value || '').toLowerCase().trim() : '';
    }
    function rowText(value) {
      return Object.keys(value || {}).map(function(key) {
        const item = value[key];
        return Array.isArray(item) ? item.join(' ') : String(item == null ? '' : item);
      }).join(' ').toLowerCase();
    }
    function matchesText(value, query) {
      if (!query) return true;
      return rowText(value).indexOf(query) >= 0;
    }
    function matchesStatus(status, filter) {
      const value = String(status || '').toLowerCase();
      if (!filter) return true;
      if (filter === 'running') return value.includes('running') || value.includes('ready') || value.includes('progress') || value.includes('wait');
      if (filter === 'completed') return value.includes('completed') || value.includes('closed') || value.includes('delivered');
      if (filter === 'blocked') return value.includes('block') || value.includes('fail') || value.includes('error');
      if (filter === 'open') return !matchesStatus(status, 'completed');
      if (filter === 'closed') return matchesStatus(status, 'completed');
      return value.includes(filter);
    }
    function renderCurrentSnapshot() {
      renderProjects(lastSnapshot.runs || []);
      renderPhaseMatrix(lastSnapshot.runs || []);
      renderUsageMatrix(lastSnapshot.runs || [], lastSnapshot.agents || []);
      renderTasks(lastSnapshot.tasks || []);
      renderAgents(lastSnapshot.agents || []);
      renderQueue(lastSnapshot.ranked || []);
      renderCompletedHistory();
    }
    function renderProjects(runs) {
      const query = filterValue('filter-runs');
      const status = filterValue('filter-status');
      const validation = filterValue('filter-validation');
      runs = (runs || []).filter(function(run) {
        return matchesText(run, query) &&
          matchesStatus(run.status || run.closure_status || '', status) &&
          (!validation || String(run.validation || '').toLowerCase().includes(validation));
      });
      if (!runs.length) {
        byId('projects-body').innerHTML = '<tr><td colspan="6" class="empty">Sin proyectos activos</td></tr>';
        return;
      }
      byId('projects-body').innerHTML = runs.map(function(run) {
        const percent = Number(run.percent_complete || 0);
        const bad = runNeedsAttention(run);
        const selected = run.run_ref === selectedRunRef ? ' class="selected"' : '';
        return '<tr data-run-ref="' + esc(run.run_ref || '') + '"' + selected + ' onclick="selectRun(\'' + jsArg(run.run_ref || '') + '\')">' +
          tableCell('Tarea', taskTitleCell(runTitle(run), run.app_ref || '-', run.task_id || taskIDFromRef(run.run_ref)), 'title="' + esc(runTitle(run)) + '"') +
          tableCell('Run', esc(shortRef(run.run_ref || '-')), 'class="mono" title="' + esc(run.run_ref || '-') + '"') +
          tableCell('Estado', statusPill(run.status || 'unknown')) +
          tableCell('Progreso', bar(percent, bad) + '<span class="sub">' + percent + '% · ' + esc(run.current_phase || '-') + '</span>') +
          tableCell('Agentes', esc(String(run.agents_in_flight || 0)) + ' / ' + esc(String(run.agents_started || 0))) +
          tableCell('Validación', statusPill(run.validation || 'pendiente')) +
        '</tr>';
      }).join('');
    }
    function renderTasks(tasks) {
      const query = filterValue('filter-tasks');
      const status = filterValue('filter-task-status');
      const agentFilter = filterValue('filter-task-agent');
      tasks = (tasks || []).filter(function(task) {
        const hasAgent = !!task.agent_ref;
        return matchesText(task, query) &&
          matchesStatus(task.status || task.progress_status || '', status) &&
          (agentFilter !== 'with_agent' || hasAgent) &&
          (agentFilter !== 'without_agent' || !hasAgent);
      });
      if (!tasks.length) {
        byId('tasks-body').innerHTML = '<tr><td colspan="5" class="empty">Sin tareas que coincidan con los filtros</td></tr>';
        return;
      }
      byId('tasks-body').innerHTML = tasks.map(function(task) {
        const selected = task.run_ref === selectedRunRef ? ' class="selected"' : '';
        return '<tr data-run-ref="' + esc(task.run_ref || '') + '"' + selected + ' onclick="selectRun(\'' + jsArg(task.run_ref || '') + '\', \'' + jsArg(task.agent_ref || '') + '\')">' +
          tableCell('Tarea', taskTitleCell(task.title || titleFromRef(task.task_ref), task.app_ref || shortRef(task.task_ref || ''), task.task_id || taskIDFromRef(task.task_ref)), 'title="' + esc(task.task_ref || '-') + '"') +
          tableCell('Run', esc(shortRef(task.run_ref || '-')), 'class="mono" title="' + esc(task.run_ref || '-') + '"') +
          tableCell('Estado', statusPill(task.status || 'unknown')) +
          tableCell('Agente', esc(shortRef(task.agent_ref || '-')), 'class="mono" title="' + esc(task.agent_ref || '-') + '"') +
          tableCell('Señal', esc(task.progress_status || '-') + ' · ticks ' + esc(String(task.no_progress_ticks || 0))) +
        '</tr>';
      }).join('');
    }
    function renderAgents(agents) {
      const query = filterValue('filter-agents');
      const status = filterValue('filter-agent-status');
      const signal = filterValue('filter-agent-signal');
      agents = (agents || []).filter(function(agent) {
        return matchesText(agent, query) &&
          (status !== 'attention' ? matchesStatus(agent.status || agent.progress_status || '', status) : agentNeedsAttention(agent)) &&
          (signal !== 'progressing' || String(agent.progress_status || '').toLowerCase().includes('progress')) &&
          (signal !== 'stalled' || Number(agent.no_progress_ticks || 0) > 0 || String(agent.progress_status || '').toLowerCase().includes('stall'));
      });
      if (!agents.length) {
        byId('agents-body').innerHTML = '<tr><td colspan="7" class="empty">Sin agentes observados</td></tr>';
        return;
      }
      byId('agents-body').innerHTML = agents.map(function(agent) {
        const selected = agent.run_ref === selectedRunRef ? ' class="selected"' : '';
        return '<tr data-run-ref="' + esc(agent.run_ref || '') + '" data-agent-ref="' + esc(agent.agent_ref || '') + '"' + selected + ' onclick="selectRun(\'' + jsArg(agent.run_ref || '') + '\', \'' + jsArg(agent.agent_ref || '') + '\')">' +
          tableCell('Agente', esc(shortRef(agent.agent_ref || '-')), 'class="mono" title="' + esc(agent.agent_ref || '-') + '"') +
          tableCell('Run', esc(shortRef(agent.run_ref || '-')), 'class="mono" title="' + esc(agent.run_ref || '-') + '"') +
	  tableCell('Estado', statusPill(agent.status || 'unknown')) +
	  tableCell('Tarea', taskTitleCell(agent.task_title || titleFromRef(agent.task_ref), shortRef(agent.task_ref || ''), agent.task_id || taskIDFromRef(agent.task_ref)), 'title="' + esc(agent.task_ref || '-') + '"') +
	  tableCell('Progreso', bar(agent.percent, agentNeedsAttention(agent)) + '<span class="sub">' + agent.percent + '%</span>') +
	  tableCell('Capacidad / uso', esc(agentUsageLabel(agent)), 'title="' + esc(agentUsageLabel(agent)) + '"') +
	  tableCell('Señal', esc(agent.progress_status || '-') + ' · ticks ' + esc(String(agent.no_progress_ticks || 0)) + (agent.currently_visible === false ? ' · visto ' + esc(agent.last_seen_at || '-') : '')) +
        '</tr>';
      }).join('');
    }
    function renderQueue(ranked) {
      if (!ranked.length) {
        byId('queue-body').innerHTML = '<tr><td colspan="7" class="empty">Sin tareas en cola</td></tr>';
        return;
      }
      byId('queue-body').innerHTML = ranked.map(function(item) {
        const title = queueTitle(item);
        const detail = queueDetail(item);
        const stableID = item.stable_id || queueStableID(item);
        const selected = item.run_ref === selectedRunRef ? ' class="selected"' : '';
        return '<tr data-run-ref="' + esc(item.run_ref || '') + '" data-stable-id="' + esc(stableID) + '"' + selected + ' onclick="selectRun(\'' + jsArg(item.run_ref || '') + '\')">' +
          tableCell('#', esc(item.rank || '-')) +
          tableCell('Tarea', taskTitleCell(title, detail, taskIDFromRef(item.run_ref)), 'title="' + esc(detail) + '"') +
          tableCell('Proyecto', esc(shortRef(item.app_ref || '-')), 'title="' + esc(item.app_ref || '-') + '"') +
          tableCell('Estado', statusPill(item.status || 'unknown')) +
          tableCell('Prioridad', esc(item.priority_score || 0)) +
          tableCell('Validación', statusPill(validationState(item))) +
          tableCell('Control', '<div class="row-actions">' +
            '<button class="small" type="button" title="Subir prioridad" onclick="event.stopPropagation(); setQueueRowPriority(\'' + jsArg(item.run_ref || '') + '\', 25)">↑</button>' +
            '<button class="small" type="button" title="Bajar prioridad" onclick="event.stopPropagation(); setQueueRowPriority(\'' + jsArg(item.run_ref || '') + '\', -25)">↓</button>' +
            '<button class="small" type="button" title="Pausar run" onclick="event.stopPropagation(); controlQueueRow(\'' + jsArg(item.run_ref || '') + '\', \'pause\')">Ⅱ</button>' +
            '<button class="small" type="button" title="Reanudar run" onclick="event.stopPropagation(); controlQueueRow(\'' + jsArg(item.run_ref || '') + '\', \'resume\')">▶</button>' +
            '<button class="small" type="button" title="Avanzar run desde fila" onclick="event.stopPropagation(); superviseQueueRow(\'' + jsArg(item.run_ref || '') + '\')">↪</button>' +
          '</div>') +
        '</tr>';
      }).join('');
    }
    function selectRun(runRef, agentRef) {
      selectedRunRef = runRef || '';
      selectedAgentRef = agentRef || '';
      controlMessage = '';
      renderProjects(lastSnapshot.runs || []);
      renderUsageMatrix(lastSnapshot.runs || [], lastSnapshot.agents || []);
      renderTasks(lastSnapshot.tasks || []);
      renderAgents(lastSnapshot.agents || []);
      renderQueue(lastSnapshot.ranked || []);
      renderCompletedHistory();
      renderSelectedDetail();
      ensureRuntimeDetail(selectedRunRef);
    }
    function updateCompletedHistory(runs, ranked) {
      const activeRefs = new Set((ranked || []).map(function(item) { return item.run_ref; }).filter(Boolean));
      const knownRefs = new Set(completedHistory.map(function(item) { return item.run_ref; }));
      (runs || []).forEach(function(run) {
        if (!run.run_ref || knownRefs.has(run.run_ref)) return;
        const terminal = isTerminalRun(run);
        const disappeared = !activeRefs.has(run.run_ref) && Number(run.percent_complete || 0) >= 100;
        if (!terminal && !disappeared) return;
        completedHistory.unshift({
          run_ref: run.run_ref,
          app_ref: run.app_ref,
          task_id: run.task_id || taskIDFromRef(run.run_ref),
          title: runTitle(run),
          status: terminal || 'salio_de_cola',
          percent_complete: run.percent_complete || 0,
          observed_at: new Date().toLocaleString()
        });
        knownRefs.add(run.run_ref);
      });
      completedHistory = completedHistory.slice(0, 200);
      saveCompletedHistory();
    }
    function isTerminalRun(run) {
      const status = String(run.status || '').toLowerCase();
      const closure = String(run.closure_status || '').toLowerCase();
      if (closure === 'closed') return 'closed';
      if (status.includes('completed') || status.includes('closed')) return status;
      if (Number(run.tasks_total || 0) > 0 && Number(run.tasks_closed || 0) >= Number(run.tasks_total || 0)) return 'completed';
      return '';
    }
    function renderCompletedHistory() {
      if (!completedHistory.length) {
        byId('completed-body').innerHTML = '<tr><td colspan="6" class="empty">Aún no hay tareas completadas observadas por este panel</td></tr>';
        return;
      }
      byId('completed-body').innerHTML = completedHistory.slice(0, 80).map(function(item) {
        const selected = item.run_ref === selectedRunRef ? ' class="selected"' : '';
        return '<tr data-run-ref="' + esc(item.run_ref || '') + '"' + selected + ' onclick="selectRun(\'' + jsArg(item.run_ref || '') + '\')">' +
          tableCell('Tarea', taskTitleCell(item.title || titleFromRef(item.run_ref), item.app_ref || '-', item.task_id || taskIDFromRef(item.run_ref))) +
          tableCell('Run', esc(shortRef(item.run_ref || '-')), 'class="mono" title="' + esc(item.run_ref || '-') + '"') +
          tableCell('Proyecto', esc(shortRef(item.app_ref || '-')), 'title="' + esc(item.app_ref || '-') + '"') +
          tableCell('Estado final', statusPill(item.status || 'completed')) +
          tableCell('Progreso', bar(item.percent_complete || 100, false) + '<span class="sub">' + esc(String(item.percent_complete || 100)) + '%</span>') +
          tableCell('Hora', esc(item.observed_at || '-')) +
        '</tr>';
      }).join('');
    }
    function listInline(values) {
      const items = (values || []).filter(Boolean);
      if (!items.length) return '<span class="sub">-</span>';
      return items.map(function(value) { return '<div class="mono">' + esc(value) + '</div>'; }).join('');
    }
    function runFlowHTML(run, runAgents, tasks) {
      const attention = runNeedsAttention(run);
      return '<div class="run-flow" aria-label="esquema de run">' +
        '<div class="run-flow-step"><div class="label">Run</div><div class="value">' + esc(shortRef(run.run_ref || '-')) + '</div><div class="hint">' + esc(run.app_ref || '-') + '</div></div>' +
        '<div class="run-flow-step"><div class="label">Fase</div><div class="value">' + esc(run.current_phase || '-') + '</div><div class="hint">' + esc(run.status || 'unknown') + '</div></div>' +
        '<div class="run-flow-step"><div class="label">Agentes</div><div class="value">' + esc(String(run.agents_in_flight || 0)) + ' / ' + esc(String(run.agents_started || runAgents.length || 0)) + '</div><div class="hint">' + esc(String(run.progressing_agents || 0)) + ' progreso · ' + esc(String(run.stalled_agents || 0)) + ' sin progreso</div></div>' +
        '<div class="run-flow-step"><div class="label">Cierre</div><div class="value">' + esc(run.closure_status || run.validation || 'pendiente') + '</div><div class="hint">' + esc(tasks.length ? ((run.tasks_closed || 0) + '/' + (run.tasks_total || tasks.length) + ' tareas') : (attention ? 'requiere atención' : 'sin tareas publicadas')) + '</div></div>' +
      '</div>';
    }
    function renderSelectedDetail() {
      captureDetailOpenState();
      const run = runByRef(selectedRunRef);
      if (!run) {
        byId('selected-detail').innerHTML = '<div class="empty">Selecciona una tarea o agente.</div>';
        return;
      }
      const runAgents = (lastSnapshot.agents || []).filter(function(agent) { return agent.run_ref === run.run_ref; });
      const tasks = run.tasks || [];
      const priority = run.priority_score == null ? 10 : run.priority_score;
      const isCompletedSnapshot = !!run.observed_at;
      const usageAgents = runAgents.filter(function(agent) {
        return agent.capacity_level || agent.quota_status || Number(agent.total_tokens || 0) > 0;
      });
      const usageTokens = usageAgents.reduce(function(total, agent) {
        return total + Number(agent.total_tokens || 0);
      }, 0);
      const taskRows = tasks.length ? tasks.map(function(task) {
        return '<div class="kv"><div class="k">' + esc(task.status || '-') + '</div><div>' +
          '<div class="task-title">' + esc(task.summary || titleFromRef(task.task_ref)) + '</div>' +
          '<div class="task-subtitle mono">' + esc(task.task_ref || '-') + '</div>' +
          '</div></div>';
      }).join('') : '<div class="sub">Sin subtareas detalladas publicadas por stats.</div>';
      const agentRows = runAgents.length ? runAgents.map(function(agent) {
        return '<div class="kv"><div class="k">' + esc(agent.status || '-') + '</div><div>' +
          '<div class="task-title">' + esc(agent.task_title || titleFromRef(agent.agent_ref)) + '</div>' +
          '<div class="task-subtitle mono">' + esc(shortRef(agent.agent_ref || '-')) + '</div>' +`
