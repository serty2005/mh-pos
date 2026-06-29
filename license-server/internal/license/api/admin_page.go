package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"mh-pos-platform/licensegate"
)

func (h *Handler) adminPage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(adminPageHTML))
}

var adminPageHTML = strings.NewReplacer(
	"__MODULE_CATALOG_JSON__", moduleCatalogJSON(),
).Replace(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>License Server</title>
  <style>
    :root { color-scheme: light; font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
    * { box-sizing: border-box; }
    body { margin: 0; background: #f7f8fa; color: #111827; }
    main { min-height: 100vh; padding: 20px; }
    .shell { max-width: 1240px; margin: 0 auto; display: grid; gap: 14px; }
    header, section { border: 1px solid #d8dee8; border-radius: 8px; background: #fff; padding: 16px; }
    header { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
    h1, h2 { margin: 0; letter-spacing: 0; }
    h1 { font-size: 22px; }
    h2 { font-size: 16px; }
    p { margin: 6px 0 0; color: #4b5563; line-height: 1.45; font-size: 14px; }
    label { display: grid; gap: 6px; color: #374151; font-size: 12px; font-weight: 700; }
    input, select, textarea { width: 100%; border: 1px solid #cbd5e1; border-radius: 6px; padding: 10px 11px; font: inherit; color: #111827; background: #fff; }
    textarea { min-height: 128px; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 12px; }
    button { min-height: 38px; border: 1px solid #111827; border-radius: 6px; background: #111827; color: #fff; padding: 9px 13px; font-weight: 700; cursor: pointer; }
    button.secondary { background: #fff; color: #111827; border-color: #cbd5e1; }
    button.danger { background: #b91c1c; border-color: #b91c1c; }
    button:disabled { opacity: .55; cursor: not-allowed; }
    table { width: 100%; border-collapse: collapse; font-size: 13px; table-layout: fixed; }
    th, td { border-bottom: 1px solid #e5e7eb; padding: 10px; text-align: left; vertical-align: top; overflow-wrap: anywhere; }
    th { color: #64748b; font-size: 11px; text-transform: uppercase; letter-spacing: .04em; }
    tr.selected { background: #f8fafc; }
    code { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 12px; }
    .grid { display: grid; gap: 12px; grid-template-columns: repeat(2, minmax(0, 1fr)); }
    .toolbar { display: flex; flex-wrap: wrap; align-items: end; gap: 10px; }
    .grow { flex: 1 1 240px; }
    .status { min-height: 22px; color: #4b5563; font-size: 13px; }
    .pill { display: inline-flex; align-items: center; border-radius: 999px; border: 1px solid #cbd5e1; padding: 3px 8px; font-size: 12px; font-weight: 700; }
    .active { border-color: #86efac; background: #f0fdf4; color: #15803d; }
    .revoked { border-color: #fecaca; background: #fef2f2; color: #b91c1c; }
    .missing { border-color: #fde68a; background: #fffbeb; color: #92400e; }
    .modules { display: flex; flex-wrap: wrap; gap: 6px; }
    .module { border: 1px solid #d8dee8; border-radius: 999px; padding: 2px 7px; color: #374151; background: #f8fafc; }
    .module-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; margin-top: 12px; }
    .toggle { display: grid; grid-template-columns: auto 1fr; gap: 10px; align-items: start; border: 1px solid #d8dee8; border-radius: 8px; padding: 12px; background: #f8fafc; }
    .toggle input { width: 18px; height: 18px; margin-top: 2px; }
    .toggle strong { display: block; font-size: 13px; color: #111827; }
    .toggle span { display: block; margin-top: 3px; color: #64748b; font-size: 12px; line-height: 1.35; }
    details { margin-top: 12px; border: 1px dashed #cbd5e1; border-radius: 8px; padding: 10px 12px; }
    summary { cursor: pointer; color: #374151; font-size: 12px; font-weight: 800; }
    .hidden { display: none !important; }
    @media (max-width: 820px) { main { padding: 12px; } .grid { grid-template-columns: 1fr; } table { table-layout: auto; } .table-wrap { overflow-x: auto; } }
    @media (max-width: 640px) { .module-grid { grid-template-columns: 1fr; } .toolbar button { flex: 1 1 150px; } }
  </style>
</head>
<body>
  <main>
    <div class="shell">
      <header>
        <div>
          <h1>License Server</h1>
          <p>Operator console for counterparty servers and entitlement snapshots.</p>
        </div>
        <button id="logout" type="button" class="secondary hidden">Sign out</button>
      </header>

      <section id="loginPanel">
        <h2>Sign in</h2>
        <div class="toolbar" style="margin-top:12px">
          <label class="grow">Login
            <input id="username" autocomplete="username" placeholder="admin">
          </label>
          <label class="grow">Password
            <input id="password" type="password" autocomplete="current-password">
          </label>
          <button id="login" type="button">Sign in</button>
        </div>
        <p id="loginStatus" class="status"></p>
      </section>

      <section id="consolePanel" class="hidden">
        <h2>Connected servers</h2>
        <div class="toolbar" style="margin-top:12px">
          <label class="grow">Search by tenant ID
            <input id="tenantSearch" placeholder="tenant id">
          </label>
          <button id="refresh" type="button" class="secondary">Refresh</button>
        </div>
        <div class="table-wrap" style="margin-top:12px">
          <table>
            <thead>
              <tr><th style="width:22%">Counterparty tenant</th><th style="width:20%">Server</th><th style="width:18%">Last seen</th><th style="width:13%">Status</th><th>Modules</th><th style="width:86px"></th></tr>
            </thead>
            <tbody id="rows"></tbody>
          </table>
        </div>
        <p id="status" class="status"></p>
      </section>

      <section id="editorPanel" class="hidden">
        <h2>License snapshot</h2>
        <p id="selectedScope"></p>
        <div class="grid" style="margin-top:12px">
          <label>Version<input id="version" type="number" min="1" value="1"></label>
          <label>Status<select id="snapshotStatus"><option value="active">active</option><option value="revoked">revoked</option></select></label>
          <label>Issued at<input id="issued" type="datetime-local"></label>
          <label>Expires at<input id="expires" type="datetime-local"></label>
        </div>
        <div id="moduleToggles" class="module-grid" aria-label="Product modules"></div>
        <details>
          <summary>Advanced support JSON</summary>
          <textarea id="entitlements" style="margin-top:8px"></textarea>
          <button id="applyJson" type="button" class="secondary" style="margin-top:8px">Apply JSON to toggles</button>
        </details>
        <div class="toolbar" style="margin-top:12px">
          <button id="save" type="button">Save snapshot</button>
          <button id="activePreset" type="button" class="secondary">Full pilot</button>
          <button id="cloudPreset" type="button" class="secondary">Tenant Cloud</button>
          <button id="minimalPreset" type="button" class="secondary">Basic POS</button>
          <button id="revokedPreset" type="button" class="danger">Revoked</button>
        </div>
      </section>
    </div>
  </main>
  <script>
    const moduleCatalog = __MODULE_CATALOG_JSON__;
    const moduleIds = moduleCatalog.map((module) => module.id);
    const rows = document.querySelector('#rows');
    const loginPanel = document.querySelector('#loginPanel');
    const consolePanel = document.querySelector('#consolePanel');
    const editorPanel = document.querySelector('#editorPanel');
    const statusLine = document.querySelector('#status');
    const loginStatus = document.querySelector('#loginStatus');
    const moduleToggles = document.querySelector('#moduleToggles');
    const selectedScope = document.querySelector('#selectedScope');
    const state = { servers: [], selectedKey: '' };
    const fields = {
      username: document.querySelector('#username'),
      password: document.querySelector('#password'),
      search: document.querySelector('#tenantSearch'),
      version: document.querySelector('#version'),
      status: document.querySelector('#snapshotStatus'),
      issued: document.querySelector('#issued'),
      expires: document.querySelector('#expires'),
      entitlements: document.querySelector('#entitlements')
    };

    function keyOf(item) { return item.tenant_id + '\n' + item.server_id; }
    function selectedServer() { return state.servers.find((item) => keyOf(item) === state.selectedKey); }
    function setStatus(message) { statusLine.textContent = message || ''; }
    function showConsole(show) {
      loginPanel.classList.toggle('hidden', show);
      consolePanel.classList.toggle('hidden', !show);
      editorPanel.classList.toggle('hidden', !show || !selectedServer());
      document.querySelector('#logout').classList.toggle('hidden', !show);
    }
    function renderModuleToggles() {
      moduleToggles.innerHTML = '';
      for (const module of moduleCatalog) {
        const label = document.createElement('label');
        label.className = 'toggle';
        label.innerHTML = '<input type="checkbox"><span><strong></strong><span></span></span>';
        label.querySelector('input').dataset.moduleId = module.id;
        label.querySelector('strong').textContent = module.id + ' - ' + module.label;
        label.querySelector('span span').textContent = module.description;
        moduleToggles.appendChild(label);
      }
    }
    function readEntitlements() {
      const out = {};
      for (const id of moduleIds) out[id] = false;
      for (const input of moduleToggles.querySelectorAll('input[type="checkbox"]')) {
        out[input.dataset.moduleId] = input.checked;
      }
      fields.entitlements.value = JSON.stringify(out, null, 2);
      return out;
    }
    function setEntitlements(next) {
      const source = next || {};
      for (const input of moduleToggles.querySelectorAll('input[type="checkbox"]')) {
        input.checked = source[input.dataset.moduleId] === true;
      }
      readEntitlements();
    }
    function toLocalInput(date) {
      const d = new Date(date);
      if (Number.isNaN(d.getTime())) return '';
      const pad = (v) => String(v).padStart(2, '0');
      return d.getFullYear() + '-' + pad(d.getMonth() + 1) + '-' + pad(d.getDate()) + 'T' + pad(d.getHours()) + ':' + pad(d.getMinutes());
    }
    function fromLocalInput(value) {
      const d = new Date(value);
      return Number.isNaN(d.getTime()) ? new Date().toISOString() : d.toISOString();
    }
    function fill(item) {
      const snapshot = item.snapshot;
      selectedScope.textContent = 'Counterparty tenant: ' + item.tenant_id + ' / server: ' + item.server_id;
      fields.version.value = String(snapshot ? snapshot.version + 1 : 1);
      fields.status.value = snapshot ? snapshot.status : 'active';
      fields.issued.value = toLocalInput(snapshot ? snapshot.issued_at : new Date().toISOString());
      fields.expires.value = toLocalInput(snapshot ? snapshot.expires_at : new Date(Date.now() + 86400000 * 30).toISOString());
      setEntitlements(snapshot ? snapshot.entitlements : {});
      editorPanel.classList.remove('hidden');
    }
    function statusBadge(item) {
      if (!item.snapshot) return '<span class="pill missing">missing</span>';
      return '<span class="pill ' + item.snapshot.status + '">' + item.snapshot.status + '</span>';
    }
    function enabledModules(item) {
      return Object.entries((item.snapshot && item.snapshot.entitlements) || {}).filter((entry) => entry[1]).map((entry) => entry[0]);
    }
    function render() {
      rows.innerHTML = '';
      const query = fields.search.value.trim().toLowerCase();
      const items = state.servers.filter((item) => !query || item.tenant_id.toLowerCase().includes(query));
      if (!items.length) {
        rows.innerHTML = '<tr><td colspan="6">No connected servers found.</td></tr>';
        return;
      }
      for (const item of items) {
        const tr = document.createElement('tr');
        if (keyOf(item) === state.selectedKey) tr.className = 'selected';
        tr.innerHTML = '<td><code></code></td><td><code></code></td><td><code></code></td><td></td><td><div class="modules"></div></td><td><button type="button" class="secondary">Select</button></td>';
        tr.children[0].querySelector('code').textContent = item.tenant_id;
        tr.children[1].querySelector('code').textContent = item.server_id;
        tr.children[2].querySelector('code').textContent = new Date(item.last_seen_at).toLocaleString();
        tr.children[3].innerHTML = statusBadge(item);
        const modules = tr.children[4].querySelector('.modules');
        for (const id of enabledModules(item)) {
          const span = document.createElement('span');
          span.className = 'module';
          span.textContent = id;
          modules.appendChild(span);
        }
        tr.children[5].querySelector('button').addEventListener('click', () => {
          state.selectedKey = keyOf(item);
          fill(item);
          render();
        });
        rows.appendChild(tr);
      }
    }
    async function refresh() {
      setStatus('Loading...');
      const response = await fetch('/api/v1/servers');
      if (response.status === 401) { showConsole(false); loginStatus.textContent = 'Sign in required.'; return; }
      if (!response.ok) { setStatus('Load failed: HTTP ' + response.status); return; }
      state.servers = await response.json();
      if (!state.servers.some((item) => keyOf(item) === state.selectedKey)) {
        state.selectedKey = state.servers[0] ? keyOf(state.servers[0]) : '';
      }
      render();
      const selected = selectedServer();
      if (selected) fill(selected);
      showConsole(true);
      setStatus('Loaded.');
    }
    async function login() {
      loginStatus.textContent = 'Signing in...';
      const response = await fetch('/api/v1/admin/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: fields.username.value.trim(), password: fields.password.value })
      });
      fields.password.value = '';
      if (!response.ok) { loginStatus.textContent = 'Sign in failed.'; return; }
      loginStatus.textContent = '';
      await refresh();
    }
    async function logout() {
      await fetch('/api/v1/admin/logout', { method: 'POST' });
      state.servers = [];
      state.selectedKey = '';
      showConsole(false);
    }
    async function save() {
      const item = selectedServer();
      if (!item) { setStatus('Select a connected server first.'); return; }
      const payload = {
        version: Number(fields.version.value),
        status: fields.status.value,
        entitlements: readEntitlements(),
        issued_at: fromLocalInput(fields.issued.value),
        expires_at: fromLocalInput(fields.expires.value)
      };
      setStatus('Saving...');
      const path = '/api/v1/entitlements/' + encodeURIComponent(item.tenant_id) + '/' + encodeURIComponent(item.server_id);
      const response = await fetch(path, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) });
      if (response.status === 401) { showConsole(false); loginStatus.textContent = 'Sign in required.'; return; }
      if (!response.ok) { setStatus('Save failed: HTTP ' + response.status); return; }
      await refresh();
      setStatus('Saved.');
    }
    function preset(kind) {
      fields.issued.value = toLocalInput(new Date().toISOString());
      fields.expires.value = toLocalInput(new Date(Date.now() + 86400000 * 30).toISOString());
      fields.status.value = kind === 'revoked' ? 'revoked' : 'active';
      const enabled = Object.fromEntries(moduleIds.map((id) => [id, kind === 'active' || (kind === 'cloud' && id === 'cloud-subscription')]));
      if (kind === 'revoked' || kind === 'minimal') for (const id of moduleIds) enabled[id] = false;
      setEntitlements(enabled);
    }
    function applyJsonToToggles() {
      try {
        setEntitlements(JSON.parse(fields.entitlements.value));
        setStatus('Advanced JSON applied to toggles.');
      } catch {
        setStatus('Entitlements JSON is invalid.');
      }
    }
    renderModuleToggles();
    document.querySelector('#login').addEventListener('click', () => login().catch((error) => { loginStatus.textContent = String(error); }));
    document.querySelector('#logout').addEventListener('click', () => logout().catch((error) => setStatus(String(error))));
    document.querySelector('#refresh').addEventListener('click', () => refresh().catch((error) => setStatus(String(error))));
    document.querySelector('#save').addEventListener('click', () => save().catch((error) => setStatus(String(error))));
    document.querySelector('#activePreset').addEventListener('click', () => preset('active'));
    document.querySelector('#cloudPreset').addEventListener('click', () => preset('cloud'));
    document.querySelector('#minimalPreset').addEventListener('click', () => preset('minimal'));
    document.querySelector('#revokedPreset').addEventListener('click', () => preset('revoked'));
    document.querySelector('#applyJson').addEventListener('click', applyJsonToToggles);
    fields.search.addEventListener('input', render);
    refresh().catch(() => showConsole(false));
  </script>
</body>
</html>`)

func moduleCatalogJSON() string {
	type moduleDTO struct {
		ID          string `json:"id"`
		Label       string `json:"label"`
		Description string `json:"description"`
	}
	modules := licensegate.CanonicalModules()
	out := make([]moduleDTO, 0, len(modules))
	for _, module := range modules {
		out = append(out, moduleDTO{ID: module.ID, Label: module.Label, Description: module.Description})
	}
	body, err := json.Marshal(out)
	if err != nil {
		return "[]"
	}
	return string(body)
}
