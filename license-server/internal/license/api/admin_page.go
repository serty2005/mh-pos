package api

import "net/http"

func (h *Handler) adminPage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(adminPageHTML))
}

const adminPageHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>License Server</title>
  <style>
    :root { color-scheme: light; font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
    * { box-sizing: border-box; }
    body { margin: 0; background: #f8fafc; color: #0f172a; }
    main { min-height: 100vh; padding: 24px; }
    .shell { max-width: 1180px; margin: 0 auto; display: grid; gap: 16px; }
    header, section { border: 1px solid #e2e8f0; border-radius: 8px; background: #fff; padding: 18px; }
    h1, h2 { margin: 0; letter-spacing: 0; }
    h1 { font-size: 22px; }
    h2 { font-size: 16px; }
    p { margin: 6px 0 0; color: #475569; line-height: 1.5; font-size: 14px; }
    label { display: grid; gap: 6px; color: #334155; font-size: 12px; font-weight: 700; }
    input, select, textarea { width: 100%; border: 1px solid #cbd5e1; border-radius: 6px; padding: 10px 11px; font: inherit; color: #0f172a; background: #fff; }
    textarea { min-height: 132px; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 12px; }
    button { border: 1px solid #0f172a; border-radius: 6px; background: #0f172a; color: #fff; padding: 10px 13px; font-weight: 700; cursor: pointer; }
    button.secondary { background: #fff; color: #0f172a; border-color: #cbd5e1; }
    button:disabled { opacity: .55; cursor: not-allowed; }
    table { width: 100%; border-collapse: collapse; font-size: 13px; }
    th, td { border-bottom: 1px solid #e2e8f0; padding: 10px; text-align: left; vertical-align: top; }
    th { color: #64748b; font-size: 11px; text-transform: uppercase; letter-spacing: .04em; }
    code { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 12px; }
    .grid { display: grid; gap: 12px; grid-template-columns: repeat(2, minmax(0, 1fr)); }
    .toolbar { display: flex; flex-wrap: wrap; align-items: end; gap: 12px; }
    .grow { flex: 1 1 220px; }
    .status { min-height: 22px; color: #475569; font-size: 13px; }
    .pill { display: inline-flex; align-items: center; border-radius: 999px; border: 1px solid #cbd5e1; padding: 3px 8px; font-size: 12px; font-weight: 700; }
    .active { border-color: #bbf7d0; background: #f0fdf4; color: #15803d; }
    .revoked { border-color: #fecaca; background: #fef2f2; color: #b91c1c; }
    .modules { display: flex; flex-wrap: wrap; gap: 6px; }
    .module { border: 1px solid #e2e8f0; border-radius: 999px; padding: 2px 7px; color: #334155; background: #f8fafc; }
    @media (max-width: 760px) { main { padding: 12px; } .grid { grid-template-columns: 1fr; } table { display: block; overflow-x: auto; } }
  </style>
</head>
<body>
  <main>
    <div class="shell">
      <header>
        <h1>License Server</h1>
        <p>Minimal operator page for connected server entitlement snapshots.</p>
      </header>

      <section>
        <h2>Connection</h2>
        <div class="toolbar" style="margin-top:12px">
          <label class="grow">Admin token
            <input id="token" type="password" autocomplete="off" placeholder="Bearer token">
          </label>
          <button id="refresh" type="button">Refresh</button>
        </div>
        <p id="status" class="status"></p>
      </section>

      <section>
        <h2>Snapshots</h2>
        <div style="margin-top:12px; overflow-x:auto">
          <table>
            <thead>
              <tr><th>Tenant</th><th>Server</th><th>Status</th><th>Version</th><th>Expires</th><th>Modules</th><th></th></tr>
            </thead>
            <tbody id="rows"></tbody>
          </table>
        </div>
      </section>

      <section>
        <h2>Edit snapshot</h2>
        <div class="grid" style="margin-top:12px">
          <label>Tenant ID<input id="tenant" placeholder="local-tenant"></label>
          <label>Server ID<input id="server" placeholder="cloud-local"></label>
          <label>Version<input id="version" type="number" min="1" value="1"></label>
          <label>Status<select id="snapshotStatus"><option value="active">active</option><option value="revoked">revoked</option></select></label>
          <label>Issued at<input id="issued" type="datetime-local"></label>
          <label>Expires at<input id="expires" type="datetime-local"></label>
        </div>
        <label style="margin-top:12px">Entitlements JSON
          <textarea id="entitlements">{
  "table-mode": true,
  "telegram-worker": false,
  "kitchen-space": true,
  "waiter-space": false,
  "checker-flow": false,
  "warehouse-mode": false
}</textarea>
        </label>
        <div class="toolbar" style="margin-top:12px">
          <button id="save" type="button">Save snapshot</button>
          <button id="activePreset" type="button" class="secondary">Active baseline</button>
          <button id="minimalPreset" type="button" class="secondary">Minimal POS</button>
          <button id="revokedPreset" type="button" class="secondary">Revoked</button>
        </div>
      </section>
    </div>
  </main>
  <script>
    const moduleIds = ['table-mode','telegram-worker','kitchen-space','waiter-space','checker-flow','warehouse-mode'];
    const token = document.querySelector('#token');
    const rows = document.querySelector('#rows');
    const statusLine = document.querySelector('#status');
    const fields = {
      tenant: document.querySelector('#tenant'),
      server: document.querySelector('#server'),
      version: document.querySelector('#version'),
      status: document.querySelector('#snapshotStatus'),
      issued: document.querySelector('#issued'),
      expires: document.querySelector('#expires'),
      entitlements: document.querySelector('#entitlements')
    };

    function setStatus(message) { statusLine.textContent = message || ''; }
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
    function authHeaders() {
      return { 'Authorization': 'Bearer ' + token.value.trim(), 'Content-Type': 'application/json' };
    }
    function fill(snapshot) {
      fields.tenant.value = snapshot.tenant_id || '';
      fields.server.value = snapshot.server_id || '';
      fields.version.value = String((snapshot.version || 0) + 1);
      fields.status.value = snapshot.status || 'active';
      fields.issued.value = toLocalInput(snapshot.issued_at || new Date().toISOString());
      fields.expires.value = toLocalInput(snapshot.expires_at || new Date(Date.now() + 86400000 * 30).toISOString());
      fields.entitlements.value = JSON.stringify(snapshot.entitlements || {}, null, 2);
    }
    function render(items) {
      rows.innerHTML = '';
      if (!items.length) {
        rows.innerHTML = '<tr><td colspan="7">No snapshots yet.</td></tr>';
        return;
      }
      for (const item of items) {
        const tr = document.createElement('tr');
        const enabled = Object.entries(item.entitlements || {}).filter((entry) => entry[1]).map((entry) => entry[0]);
        tr.innerHTML = '<td><code></code></td><td><code></code></td><td></td><td></td><td><code></code></td><td><div class="modules"></div></td><td><button type="button" class="secondary">Edit</button></td>';
        tr.children[0].querySelector('code').textContent = item.tenant_id;
        tr.children[1].querySelector('code').textContent = item.server_id;
        tr.children[2].innerHTML = '<span class="pill ' + item.status + '">' + item.status + '</span>';
        tr.children[3].textContent = item.version;
        tr.children[4].querySelector('code').textContent = new Date(item.expires_at).toLocaleString();
        const modules = tr.children[5].querySelector('.modules');
        for (const id of enabled) {
          const span = document.createElement('span');
          span.className = 'module';
          span.textContent = id;
          modules.appendChild(span);
        }
        tr.children[6].querySelector('button').addEventListener('click', () => fill(item));
        rows.appendChild(tr);
      }
    }
    async function refresh() {
      if (!token.value.trim()) { setStatus('Admin token is required.'); return; }
      setStatus('Loading...');
      const response = await fetch('/api/v1/entitlements', { headers: authHeaders() });
      if (!response.ok) { setStatus('Load failed: HTTP ' + response.status); return; }
      render(await response.json());
      setStatus('Loaded.');
    }
    async function save() {
      if (!token.value.trim()) { setStatus('Admin token is required.'); return; }
      const tenant = fields.tenant.value.trim();
      const server = fields.server.value.trim();
      if (!tenant || !server) { setStatus('Tenant and server are required.'); return; }
      let entitlements;
      try { entitlements = JSON.parse(fields.entitlements.value); } catch { setStatus('Entitlements JSON is invalid.'); return; }
      const payload = {
        version: Number(fields.version.value),
        status: fields.status.value,
        entitlements,
        issued_at: fromLocalInput(fields.issued.value),
        expires_at: fromLocalInput(fields.expires.value)
      };
      setStatus('Saving...');
      const response = await fetch('/api/v1/entitlements/' + encodeURIComponent(tenant) + '/' + encodeURIComponent(server), { method: 'PUT', headers: authHeaders(), body: JSON.stringify(payload) });
      if (!response.ok) { setStatus('Save failed: HTTP ' + response.status); return; }
      fill(await response.json());
      await refresh();
      setStatus('Saved.');
    }
    function preset(kind) {
      const now = new Date();
      const expires = new Date(Date.now() + 86400000 * 30);
      fields.issued.value = toLocalInput(now.toISOString());
      fields.expires.value = toLocalInput(expires.toISOString());
      fields.status.value = kind === 'revoked' ? 'revoked' : 'active';
      const enabled = Object.fromEntries(moduleIds.map((id) => [id, kind === 'active' || (kind === 'minimal' && id === 'table-mode')]));
      if (kind === 'revoked') for (const id of moduleIds) enabled[id] = false;
      fields.entitlements.value = JSON.stringify(enabled, null, 2);
    }
    document.querySelector('#refresh').addEventListener('click', () => refresh().catch((error) => setStatus(String(error))));
    document.querySelector('#save').addEventListener('click', () => save().catch((error) => setStatus(String(error))));
    document.querySelector('#activePreset').addEventListener('click', () => preset('active'));
    document.querySelector('#minimalPreset').addEventListener('click', () => preset('minimal'));
    document.querySelector('#revokedPreset').addEventListener('click', () => preset('revoked'));
    const now = new Date();
    fields.issued.value = toLocalInput(now.toISOString());
    fields.expires.value = toLocalInput(new Date(Date.now() + 86400000 * 30).toISOString());
  </script>
</body>
</html>`
