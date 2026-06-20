// Faillefox — logique du panneau de contrôle.
// Vanilla JS, aucune dépendance. Communique avec l'API loopback du pare-feu.

const API = ""; // même origine (servi par le démon Go)

// ---- utilitaires ----------------------------------------------------------
async function getJSON(path) {
  const r = await fetch(API + path);
  if (!r.ok) throw new Error(`${path}: ${r.status}`);
  return r.json();
}
async function postJSON(path, body) {
  const r = await fetch(API + path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body || {}),
  });
  if (!r.ok && r.status !== 204) throw new Error(`${path}: ${r.status}`);
  return r.status === 204 ? null : r.json();
}
function esc(s) {
  return String(s ?? "").replace(/[&<>"']/g, (c) => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
  }[c]));
}
function fmtTime(iso) {
  const d = new Date(iso);
  return d.toLocaleTimeString("fr-FR");
}

// ---- navigation par onglets ----------------------------------------------
document.querySelectorAll(".tab").forEach((tab) => {
  tab.addEventListener("click", () => {
    document.querySelectorAll(".tab").forEach((t) => t.classList.remove("active"));
    document.querySelectorAll(".panel").forEach((p) => p.classList.remove("active"));
    tab.classList.add("active");
    document.getElementById("tab-" + tab.dataset.tab).classList.add("active");
  });
});

// ---- état global ----------------------------------------------------------
let apps = [];
let rules = [];

// ---- applications (mode simple) ------------------------------------------
async function loadApps() {
  apps = await getJSON("/api/apps");
  const ul = document.getElementById("apps-list");
  ul.innerHTML = apps
    .map(
      (a) => `
      <li>
        <div>
          <div class="app-name">${esc(a.name)}</div>
          <div class="app-path">${esc(a.path)}</div>
        </div>
        <label class="switch">
          <input type="checkbox" data-app="${esc(a.id)}" class="app-toggle" />
          <span class="slider"></span>
        </label>
      </li>`
    )
    .join("");

  // Remplit aussi le sélecteur du mode avancé.
  const sel = document.getElementById("rule-app");
  sel.innerHTML =
    `<option value="">(toutes les applications)</option>` +
    apps.map((a) => `<option value="${esc(a.id)}">${esc(a.name)}</option>`).join("");

  // Synchronise l'état des interrupteurs avec les règles existantes.
  syncAppToggles();
}

function syncAppToggles() {
  document.querySelectorAll(".app-toggle").forEach((cb) => {
    const denyRule = rules.find(
      (r) => r.app_id === cb.dataset.app && r.action === "deny" && !r.port && !r.ip
    );
    cb.checked = !denyRule; // coché = Internet autorisé
  });
}

// Gestion du clic sur un interrupteur : crée/supprime une règle deny.
document.getElementById("apps-list").addEventListener("change", async (e) => {
  if (!e.target.classList.contains("app-toggle")) return;
  const appId = e.target.dataset.app;
  const allow = e.target.checked;
  try {
    if (allow) {
      // Retire une éventuelle règle deny existante.
      const deny = rules.find(
        (r) => r.app_id === appId && r.action === "deny" && !r.port && !r.ip
      );
      if (deny) await fetch(`${API}/api/rules?id=${encodeURIComponent(deny.id)}`, { method: "DELETE" });
    } else {
      await postJSON("/api/rules", {
        app_id: appId,
        action: "deny",
        protocol: "",
        port: 0,
        ip: "",
        note: "bloqué depuis le mode simple",
      });
    }
    await refreshRules();
  } catch (err) {
    alert("Erreur: " + err.message);
    e.target.checked = !allow; // rollback visuel
  }
});

// ---- politique par défaut ------------------------------------------------
function renderDefault(d) {
  document.querySelectorAll("#default-choice button").forEach((b) => {
    b.classList.toggle("active", b.dataset.decision === d);
  });
}
document.getElementById("default-choice").addEventListener("click", async (e) => {
  const btn = e.target.closest("button");
  if (!btn) return;
  try {
    await postJSON("/api/default", { decision: btn.dataset.decision });
    renderDefault(btn.dataset.decision);
    refreshStatus();
  } catch (err) {
    alert("Erreur: " + err.message);
  }
});

// ---- règles avancées ------------------------------------------------------
async function refreshRules() {
  rules = await getJSON("/api/rules");
  const body = document.getElementById("rules-body");
  body.innerHTML = rules
    .map(
      (r) => `
      <tr>
        <td>${esc(appName(r.app_id))}</td>
        <td><span class="badge ${r.action}">${r.action}</span></td>
        <td>${esc(r.protocol || "tous")}</td>
        <td>${r.port || "tous"}</td>
        <td>${esc(r.ip || "toutes")}</td>
        <td><button class="del-btn" data-id="${esc(r.id)}" title="Supprimer">✕</button></td>
      </tr>`
    )
    .join("");
  syncAppToggles();
}
function appName(id) {
  if (!id) return "(toutes)";
  const a = apps.find((x) => x.id === id);
  return a ? a.name : id;
}

document.getElementById("rules-body").addEventListener("click", async (e) => {
  if (!e.target.classList.contains("del-btn")) return;
  if (!confirm("Supprimer cette règle ?")) return;
  await fetch(`${API}/api/rules?id=${encodeURIComponent(e.target.dataset.id)}`, { method: "DELETE" });
  refreshRules();
  refreshStatus();
});

document.getElementById("rule-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  const fd = new FormData(e.target);
  const port = parseInt(fd.get("port"), 10) || 0;
  await postJSON("/api/rules", {
    app_id: fd.get("app_id"),
    action: fd.get("action"),
    protocol: fd.get("protocol"),
    port,
    ip: fd.get("ip"),
  });
  e.target.reset();
  refreshRules();
  refreshStatus();
});

// ---- statut ---------------------------------------------------------------
async function refreshStatus() {
  const s = await getJSON("/api/status");
  const dot = document.getElementById("status-dot");
  const txt = document.getElementById("status-text");
  txt.textContent = `Pilote: ${s.driver} · ${s.rules_count} règle(s) · défaut: ${s.default_decision}`;
  dot.className = "dot ok";
  renderDefault(s.default_decision);
}

// ---- journal temps réel (SSE) --------------------------------------------
function startEventStream() {
  const es = new EventSource(API + "/api/events");
  const logs = document.getElementById("logs");
  es.onmessage = (msg) => {
    const ev = JSON.parse(msg.data);
    const line = document.createElement("div");
    line.className = "log-line";
    line.innerHTML = `
      <span class="log-time">${esc(fmtTime(ev.At))}</span>
      <span class="log-app">${esc(ev.Connection.app_name)}</span>
      <span class="log-target">${esc(ev.Connection.protocol)} → ${esc(ev.Connection.remote_addr)}:${ev.Connection.remote_port}</span>
      <span class="log-verdict ${ev.Decision}">${ev.Decision}</span>`;
    logs.prepend(line);
    // Limite à 300 lignes pour éviter l'explosion mémoire.
    while (logs.children.length > 300) logs.removeChild(logs.lastChild);
  };
  es.onerror = () => {
    document.getElementById("status-dot").className = "dot bad";
  };
}

// ---- bootstrap ------------------------------------------------------------
(async function init() {
  try {
    await loadApps();
    await refreshRules();
    await refreshStatus();
    startEventStream();
  } catch (err) {
    document.getElementById("status-text").textContent = "Erreur: " + err.message;
    document.getElementById("status-dot").className = "dot bad";
  }
})();
